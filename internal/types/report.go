package types

import (
	"encoding/json"
	"time"

	"github.com/chemanyu/adn-report-ai/internal/importer"
)

type Upload struct {
	ID          int64           `json:"id"`
	UserID      int64           `json:"user_id"`
	Uploader    string          `json:"uploader"`
	Operator    string          `json:"operator"`
	AccountID   int64           `json:"account_id"`
	Account     string          `json:"account"`
	AccountCode string          `json:"account_code"`
	Filename    string          `json:"filename"`
	Sheet       string          `json:"sheet"`
	Columns     json.RawMessage `json:"columns"`
	RowCount    int             `json:"row_count"`
	Total       string          `json:"total_amount"`
	DateFrom    string          `json:"date_from"`
	DateTo      string          `json:"date_to"`
	CreatedAt   time.Time       `json:"created_at"`
	CSVPath     string          `json:"-"`
}

// FileRequest contains the decoded multipart file and form fields.
type FileRequest struct {
	Data      []byte
	Filename  string
	Sheet     string
	Operator  string
	AccountID string
}
type ListUploadsRequest struct {
	Page      int
	Query     string
	AccountID string
}
type UploadDetailRequest struct {
	ID   string
	Page int
}
type PreviewResponse struct {
	Result   *importer.Result `json:"result"`
	RowCount int              `json:"row_count"`
	Valid    bool             `json:"valid"`
}
type UploadResponse struct {
	ID       int64  `json:"id"`
	RowCount int    `json:"row_count"`
	Total    string `json:"total_amount"`
}
type UploadListResponse struct {
	Items        []Upload `json:"items"`
	Total        int      `json:"total"`
	Page         int      `json:"page"`
	PageSize     int      `json:"page_size"`
	RowCount     int      `json:"row_count"`
	TotalAmount  string   `json:"total_amount"`
	AccountCount int      `json:"account_count"`
}
type UploadDetailResponse struct {
	Upload   Upload         `json:"upload"`
	Rows     []importer.Row `json:"rows"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}
