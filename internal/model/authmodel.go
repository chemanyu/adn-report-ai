package model

import (
	"context"
	"database/sql"
	"time"
)

type AuthModel interface {
	FindSession(context.Context, string) (User, error)
	CreateSession(context.Context, string, string, string, string, time.Time) error
	DeleteSession(context.Context, string) error
	CreateOAuthState(context.Context, string, time.Time) error
	ConsumeOAuthState(context.Context, string) (bool, error)
	DeleteExpired(context.Context) error
}

type authModel struct{ db *sql.DB }

func NewAuthModel(db *sql.DB) AuthModel { return &authModel{db: db} }
func (m *authModel) FindSession(ctx context.Context, tokenHash string) (User, error) {
	var user User
	err := m.db.QueryRowContext(ctx, `SELECT u.id,u.identity_key,u.name FROM sessions se JOIN users u ON u.id=se.user_id WHERE se.token_hash=$1 AND se.expires_at>CURRENT_TIMESTAMP`, tokenHash).Scan(&user.ID, &user.Identity, &user.Name)
	return user, err
}

// CreateSession atomically updates the user and replaces the previous session.
func (m *authModel) CreateSession(ctx context.Context, identity, name, oldHash, newHash string, expires time.Time) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO users(identity_key,name) VALUES($1,$2) ON CONFLICT (identity_key) DO UPDATE SET name=EXCLUDED.name,last_login_at=CURRENT_TIMESTAMP`, identity, name)
	if err != nil {
		return err
	}
	var id int64
	if err = tx.QueryRowContext(ctx, "SELECT id FROM users WHERE identity_key=$1", identity).Scan(&id); err != nil {
		return err
	}
	if oldHash != "" {
		if _, err = tx.ExecContext(ctx, "DELETE FROM sessions WHERE token_hash=$1", oldHash); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO sessions(token_hash,user_id,expires_at) VALUES($1,$2,$3)", newHash, id, expires); err != nil {
		return err
	}
	return tx.Commit()
}
func (m *authModel) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := m.db.ExecContext(ctx, "DELETE FROM sessions WHERE token_hash=$1", tokenHash)
	return err
}
func (m *authModel) CreateOAuthState(ctx context.Context, hash string, expires time.Time) error {
	_, err := m.db.ExecContext(ctx, "INSERT INTO oauth_states(token_hash,expires_at) VALUES($1,$2)", hash, expires)
	return err
}
func (m *authModel) ConsumeOAuthState(ctx context.Context, hash string) (bool, error) {
	result, err := m.db.ExecContext(ctx, "DELETE FROM oauth_states WHERE token_hash=$1 AND expires_at>CURRENT_TIMESTAMP", hash)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}
func (m *authModel) DeleteExpired(ctx context.Context) error {
	if _, err := m.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at<CURRENT_TIMESTAMP"); err != nil {
		return err
	}
	_, err := m.db.ExecContext(ctx, "DELETE FROM oauth_states WHERE expires_at<CURRENT_TIMESTAMP")
	return err
}
