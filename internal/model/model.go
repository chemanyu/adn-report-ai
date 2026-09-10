// Package model owns all business SQL and database transaction boundaries.
package model

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

var ErrNotFound = sql.ErrNoRows

func IsDuplicate(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}

type User struct {
	ID       int64
	Identity string
	Name     string
}
type Account struct {
	ID   int64
	Name string
	Code string
}
type Upload struct {
	ID          int64
	UserID      int64
	Uploader    string
	Operator    string
	AccountID   int64
	Account     string
	AccountCode string
	Filename    string
	Sheet       string
	Columns     json.RawMessage
	RowCount    int
	Total       string
	DateFrom    string
	DateTo      string
	CreatedAt   time.Time
	CSVPath     string
}
type SettlementRow struct {
	SourceRow int
	Date      string
	Count     string
	Price     string
	Amount    string
	Extra     map[string]string
	Source    []string
}
type NewUpload struct {
	Upload Upload
	Hash   string
	Rows   []SettlementRow
}

// OwnerID is nil only when the caller has explicitly granted administrator scope.
type UploadFilter struct {
	OwnerID   *int64
	AccountID int64
	Query     string
	Page      int
}
type UploadList struct {
	Items        []Upload
	Total        int
	RowCount     int
	TotalAmount  string
	AccountCount int
}
