package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type UploadModel interface {
	List(context.Context, UploadFilter) (UploadList, error)
	Snapshot(context.Context, int64, *int64, int) (Upload, []SettlementRow, error)
	SaveWithRows(context.Context, NewUpload) (UploadSaveResult, error)
}

type uploadModel struct{ db *sql.DB }

func NewUploadModel(db *sql.DB) UploadModel { return &uploadModel{db: db} }

const uploadSelect = `SELECT u.id,u.user_id,u.uploader_name,u.operator_name,COALESCE((SELECT jsonb_agg(DISTINCT r.advertiser ORDER BY r.advertiser) FROM settlement_rows r WHERE r.upload_id=u.id AND r.advertiser IS NOT NULL),'[]'::jsonb),u.original_name,u.sheet_name,u.columns_json,u.row_count,u.total_amount,COALESCE(u.date_from::text,''),COALESCE(u.date_to::text,''),u.created_at,u.csv_path FROM uploads u`

type scanner interface{ Scan(...any) error }

func scanUpload(sc scanner) (Upload, error) {
	var v Upload
	var advertisers []byte
	err := sc.Scan(&v.ID, &v.UserID, &v.Uploader, &v.Operator, &advertisers, &v.Filename, &v.Sheet, &v.Columns, &v.RowCount, &v.Total, &v.DateFrom, &v.DateTo, &v.CreatedAt, &v.CSVPath)
	if err != nil {
		return v, err
	}
	err = json.Unmarshal(advertisers, &v.Advertisers)
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
		where += fmt.Sprintf(" AND (u.original_name ILIKE $%d OR u.operator_name ILIKE $%d OR u.uploader_name ILIKE $%d OR EXISTS (SELECT 1 FROM settlement_rows r WHERE r.upload_id=u.id AND (r.advertiser ILIKE $%d OR r.agency ILIKE $%d)))", len(args), len(args), len(args), len(args), len(args))
	}
	result := UploadList{Items: []Upload{}}
	err := m.db.QueryRowContext(ctx, "SELECT COUNT(*),COALESCE(SUM(u.row_count),0),COALESCE(SUM(u.total_amount),0),(SELECT COUNT(DISTINCT r.advertiser) FROM settlement_rows r JOIN uploads u ON u.id=r.upload_id WHERE "+where+") FROM uploads u WHERE "+where, args...).Scan(&result.Total, &result.RowCount, &result.TotalAmount, &result.AdvertiserCount)
	if err != nil {
		return result, err
	}
	queryArgs := append(append([]any{}, args...), 20, (filter.Page-1)*20)
	rows, err := m.db.QueryContext(ctx, uploadSelect+" WHERE "+where+fmt.Sprintf(" ORDER BY u.created_at DESC,u.id DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2), queryArgs...)
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

// Snapshot keeps metadata and detail/CSV rows consistent during concurrent imports.
// page=0 reads all current rows for a CSV download.
func (m *uploadModel) Snapshot(ctx context.Context, id int64, ownerID *int64, page int) (Upload, []SettlementRow, error) {
	tx, err := m.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return Upload{}, nil, err
	}
	defer tx.Rollback()
	where, args := ownerScope(ownerID)
	args = append(args, id)
	upload, err := scanUpload(tx.QueryRowContext(ctx, uploadSelect+" WHERE "+where+fmt.Sprintf(" AND u.id=$%d", len(args)), args...))
	if err != nil {
		return Upload{}, nil, err
	}
	query := `SELECT source_row,COALESCE(agency,''),COALESCE(advertiser,''),COALESCE(task_name,''),settlement_date::text,COALESCE(settlement_count::text,''),COALESCE(unit_price::text,''),COALESCE(amount::text,''),extra_fields FROM settlement_rows WHERE upload_id=$1 ORDER BY source_row`
	params := []any{id}
	if page > 0 {
		query += " LIMIT 100 OFFSET $2"
		params = append(params, (page-1)*100)
	}
	rows, err := tx.QueryContext(ctx, query, params...)
	if err != nil {
		return Upload{}, nil, err
	}
	defer rows.Close()
	list := []SettlementRow{}
	for rows.Next() {
		var row SettlementRow
		var extra []byte
		if err = rows.Scan(&row.SourceRow, &row.Agency, &row.Advertiser, &row.TaskName, &row.Date, &row.Count, &row.Price, &row.Amount, &extra); err != nil {
			return Upload{}, nil, err
		}
		if err = json.Unmarshal(extra, &row.Extra); err != nil {
			return Upload{}, nil, err
		}
		list = append(list, row)
	}
	if err = rows.Err(); err != nil {
		return Upload{}, nil, err
	}
	rows.Close()
	if err = tx.Commit(); err != nil {
		return Upload{}, nil, err
	}
	return upload, list, nil
}

// SaveWithRows updates only matching business keys in the authenticated owner
// scope. A user-level transaction lock prevents races and overlapping-import deadlocks.
func (m *uploadModel) SaveWithRows(ctx context.Context, in NewUpload) (UploadSaveResult, error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return UploadSaveResult{}, err
	}
	defer tx.Rollback()
	u := in.Upload
	lock := fmt.Sprintf("settlement:user:%d", u.UserID)
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(current_schema() || ':' || $1,0))`, lock); err != nil {
		return UploadSaveResult{}, err
	}
	result := UploadSaveResult{}
	err = tx.QueryRowContext(ctx, `INSERT INTO uploads(user_id,uploader_name,operator_name,original_name,sheet_name,sha256,csv_path,columns_json,row_count,total_amount,date_from,date_to) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id`, u.UserID, u.Uploader, u.Operator, u.Filename, u.Sheet, in.Hash, u.CSVPath, string(u.Columns), u.RowCount, u.Total, u.DateFrom, u.DateTo).Scan(&result.ID)
	if err != nil {
		return UploadSaveResult{}, err
	}
	affected := map[int64]bool{result.ID: true}
	for _, row := range in.Rows {
		old, err := tx.QueryContext(ctx, `SELECT r.id,r.upload_id FROM settlement_rows r JOIN uploads u ON u.id=r.upload_id WHERE u.user_id=$1 AND r.agency COLLATE "C"=$2 AND r.advertiser COLLATE "C"=$3 AND r.settlement_date=$4 AND r.task_name COLLATE "C"=$5 ORDER BY r.id DESC FOR UPDATE OF r`, u.UserID, row.Agency, row.Advertiser, row.Date, row.TaskName)
		if err != nil {
			return UploadSaveResult{}, err
		}
		ids := []int64{}
		for old.Next() {
			var id, batch int64
			if err = old.Scan(&id, &batch); err != nil {
				old.Close()
				return UploadSaveResult{}, err
			}
			ids = append(ids, id)
			affected[batch] = true
		}
		err = old.Err()
		old.Close()
		if err != nil {
			return UploadSaveResult{}, err
		}
		extra, err := json.Marshal(row.Extra)
		if err != nil {
			return UploadSaveResult{}, err
		}
		args := []any{result.ID, row.SourceRow, row.Agency, row.Advertiser, row.TaskName, row.Date, row.Count, row.Price, row.Amount, string(extra)}
		if len(ids) == 0 {
			_, err = tx.ExecContext(ctx, `INSERT INTO settlement_rows(upload_id,source_row,agency,advertiser,task_name,settlement_date,settlement_count,unit_price,amount,extra_fields) VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,'')::numeric,NULLIF($8,'')::numeric,NULLIF($9,'')::numeric,$10)`, args...)
			result.Inserted++
		} else {
			// Consolidate historical duplicates only when this exact key is explicitly uploaded.
			if len(ids) > 1 {
				if _, err = tx.ExecContext(ctx, `DELETE FROM settlement_rows WHERE id=ANY($1::bigint[])`, ids[1:]); err != nil {
					return UploadSaveResult{}, err
				}
			}
			args = append(args, ids[0])
			_, err = tx.ExecContext(ctx, `UPDATE settlement_rows SET upload_id=$1,source_row=$2,agency=$3,advertiser=$4,task_name=$5,settlement_date=$6,settlement_count=NULLIF($7,'')::numeric,unit_price=NULLIF($8,'')::numeric,amount=NULLIF($9,'')::numeric,extra_fields=$10 WHERE id=$11`, args...)
			result.Updated++
		}
		if err != nil {
			return UploadSaveResult{}, err
		}
	}
	ids := make([]int64, 0, len(affected))
	for id := range affected {
		ids = append(ids, id)
	}
	_, err = tx.ExecContext(ctx, `UPDATE uploads u SET row_count=s.n,total_amount=s.total,date_from=s.first_date,date_to=s.last_date FROM (SELECT b.id,COUNT(r.id)::int AS n,COALESCE(SUM(r.amount),0) AS total,MIN(r.settlement_date) AS first_date,MAX(r.settlement_date) AS last_date FROM uploads b LEFT JOIN settlement_rows r ON r.upload_id=b.id WHERE b.id=ANY($1::bigint[]) GROUP BY b.id) s WHERE u.id=s.id`, ids)
	if err != nil {
		return UploadSaveResult{}, err
	}
	if err = tx.Commit(); err != nil {
		return UploadSaveResult{}, err
	}
	return result, nil
}
