package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/chemanyu/adn-report-ai/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

//go:embed schema.sql
var schema string

// Connect opens only the configured database; it does not create or modify tables.
func Connect(c config.Config) (*sql.DB, error) {
	cfg, err := pgx.ParseConfig(c.PostgreSQL.DSN)
	if err != nil {
		return nil, fmt.Errorf("PostgreSQL 连接配置无效")
	}
	cfg.ConnectTimeout = 5 * time.Second
	cfg.RuntimeParams["search_path"] = pgx.Identifier{c.PostgreSQL.Schema}.Sanitize()
	cfg.RuntimeParams["timezone"] = "UTC"
	cfg.RuntimeParams["DateStyle"] = "ISO, YMD"
	cfg.RuntimeParams["application_name"] = "adn-report-ai"
	db := stdlib.OpenDB(*cfg)
	db.SetMaxOpenConns(12)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		db.Close()
		return nil, RedactError(c, err)
	}
	return db, nil
}

// Open initializes the application's own schema atomically. The database must already exist.
func Open(c config.Config) (*sql.DB, error) {
	db, err := Connect(c)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		db.Close()
		return nil, RedactError(c, err)
	}
	defer tx.Rollback()
	// Serialize concurrent startup of this schema; PostgreSQL DDL is transactional.
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtext($1))", "adn-report-ai:"+c.PostgreSQL.Schema); err == nil {
		_, err = tx.ExecContext(ctx, "CREATE SCHEMA IF NOT EXISTS "+pgx.Identifier{c.PostgreSQL.Schema}.Sanitize())
	}
	if err == nil {
		for _, q := range strings.Split(schema, ";") {
			if strings.TrimSpace(q) == "" {
				continue
			}
			if _, err = tx.ExecContext(ctx, q); err != nil {
				break
			}
		}
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		tx.Rollback()
		db.Close()
		return nil, RedactError(c, err)
	}
	return db, nil
}

// Do not expose a configured DSN or password in database startup errors.
func RedactError(c config.Config, err error) error {
	message := err.Error()
	if c.PostgreSQL.DSN != "" {
		message = strings.ReplaceAll(message, c.PostgreSQL.DSN, "[PostgreSQL DSN]")
	}
	if u, e := url.Parse(c.PostgreSQL.DSN); e == nil && u.User != nil {
		if password, ok := u.User.Password(); ok && password != "" {
			message = strings.ReplaceAll(message, password, "***")
		}
	}
	return fmt.Errorf("PostgreSQL: %s", message)
}
