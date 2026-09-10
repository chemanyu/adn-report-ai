// Package model owns all business SQL and database transaction boundaries.
package model

import (
	"database/sql"
	"encoding/json"
	"time"
)

var ErrNotFound = sql.ErrNoRows

type User struct {
	ID       int64
	Identity string
	Name     string
}
type Upload struct {
	ID          int64
	UserID      int64
	Uploader    string
	Operator    string
	Advertisers []string
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
	Agency     string
	Advertiser string
	TaskName   string
	SourceRow  int
	Date       string
	Count      string
	Price      string
	Amount     string
	Extra      map[string]string
}
type NewUpload struct {
	Upload Upload
	Hash   string
	Rows   []SettlementRow
}

type UploadSaveResult struct {
	ID       int64
	Inserted int
	Updated  int
}

// OwnerID is nil only when the caller has explicitly granted administrator scope.
type UploadFilter struct {
	OwnerID *int64
	Query   string
	Page    int
}
type UploadList struct {
	Items           []Upload
	Total           int
	RowCount        int
	TotalAmount     string
	AdvertiserCount int
}
