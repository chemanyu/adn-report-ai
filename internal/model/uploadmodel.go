package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type UploadModel interface {
	List(context.Context, UploadFilter) (UploadList, error)
	FindScoped(context.Context, int64, *int64) (Upload, error)
	Rows(context.Context, int64, int) ([]SettlementRow, error)
	InsertWithRows(context.Context, NewUpload) (int64, error)
}

type uploadModel struct{ db *sql.DB }

func NewUploadModel(db *sql.DB) UploadModel { return &uploadModel{db: db} }

const uploadSelect = `SELECT u.id,u.user_id,u.uploader_name,u.operator_name,u.account_id,a.name,a.account_code,u.original_name,u.sheet_name,u.columns_json,u.row_count,u.total_amount,u.date_from::text,u.date_to::text,u.created_at,u.csv_path FROM uploads u JOIN adn_accounts a ON a.id=u.account_id`

type scanner interface{ Scan(...any) error }

func scanUpload(sc scanner) (Upload, error) {
	var v Upload
	err := sc.Scan(&v.ID, &v.UserID, &v.Uploader, &v.Operator, &v.AccountID, &v.Account, &v.AccountCode, &v.Filename, &v.Sheet, &v.Columns, &v.RowCount, &v.Total, &v.DateFrom, &v.DateTo, &v.CreatedAt, &v.CSVPath)
	return v, err
}
func ownerScope(ownerID *int64) (string, []any) {
	if ownerID == nil {
		return "1=1", []any{}
	}
	return "u.user_id=$1", []any{*ownerID}
}
func (m *uploadModel) List(ctx context.Context, filter UploadFilter) (UploadList, error) {
	where, args := ownerScope(filter.OwnerID)
	if filter.Query != "" {
		args = append(args, "%"+filter.Query+"%")
		where += fmt.Sprintf(" AND (u.original_name ILIKE $%d OR u.operator_name ILIKE $%d OR u.uploader_name ILIKE $%d)", len(args), len(args), len(args))
	}
	if filter.AccountID > 0 {
		args = append(args, filter.AccountID)
		where += fmt.Sprintf(" AND u.account_id=$%d", len(args))
	}
	result := UploadList{Items: []Upload{}}
	err := m.db.QueryRowContext(ctx, "SELECT COUNT(*),COALESCE(SUM(u.row_count),0),COALESCE(SUM(u.total_amount),0),COUNT(DISTINCT u.account_id) FROM uploads u WHERE "+where, args...).Scan(&result.Total, &result.RowCount, &result.TotalAmount, &result.AccountCount)
	if err != nil {
		return result, err
	}
	queryArgs := append(append([]any{}, args...), 20, (filter.Page-1)*20)
	rows, err := m.db.QueryContext(ctx, uploadSelect+" WHERE "+where+fmt.Sprintf(" ORDER BY u.id DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2), queryArgs...)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		v, err := scanUpload(rows)
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, v)
	}
	return result, rows.Err()
}
func (m *uploadModel) FindScoped(ctx context.Context, id int64, ownerID *int64) (Upload, error) {
	where, args := ownerScope(ownerID)
	args = append(args, id)
	return scanUpload(m.db.QueryRowContext(ctx, uploadSelect+" WHERE "+where+fmt.Sprintf(" AND u.id=$%d", len(args)), args...))
}
func (m *uploadModel) Rows(ctx context.Context, id int64, page int) ([]SettlementRow, error) {
	rows, err := m.db.QueryContext(ctx, `SELECT source_row,settlement_date::text,settlement_count,unit_price,amount,extra_fields,source_values FROM settlement_rows WHERE upload_id=$1 ORDER BY source_row LIMIT 100 OFFSET $2`, id, (page-1)*100)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []SettlementRow{}
	for rows.Next() {
		var row SettlementRow
		var extra, source []byte
		if err = rows.Scan(&row.SourceRow, &row.Date, &row.Count, &row.Price, &row.Amount, &extra, &source); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(extra, &row.Extra); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(source, &row.Source); err != nil {
			return nil, err
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

// InsertWithRows commits the upload and every settlement row in one transaction.
func (m *uploadModel) InsertWithRows(ctx context.Context, in NewUpload) (int64, error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	u := in.Upload
	var id int64
	err = tx.QueryRowContext(ctx, `INSERT INTO uploads(user_id,uploader_name,operator_name,account_id,original_name,sheet_name,sha256,csv_path,columns_json,row_count,total_amount,date_from,date_to) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING id`, u.UserID, u.Uploader, u.Operator, u.AccountID, u.Filename, u.Sheet, in.Hash, u.CSVPath, string(u.Columns), u.RowCount, u.Total, u.DateFrom, u.DateTo).Scan(&id)
	if err != nil {
		return 0, err
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO settlement_rows(upload_id,source_row,settlement_date,settlement_count,unit_price,amount,extra_fields,source_values) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	for _, row := range in.Rows {
		extra, err := json.Marshal(row.Extra)
		if err != nil {
			return 0, err
		}
		source, err := json.Marshal(row.Source)
		if err != nil {
			return 0, err
		}
		if _, err = stmt.ExecContext(ctx, id, row.SourceRow, row.Date, row.Count, row.Price, row.Amount, string(extra), string(source)); err != nil {
			return 0, err
		}
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}
