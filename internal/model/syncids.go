package model

import (
	"context"
	"database/sql"
	"fmt"
)

// BackfillIDs resolves all account/task combinations, including existing account IDs.
// A failed or ambiguous lookup writes nothing.
func BackfillIDs(ctx context.Context, db *sql.DB, resolver IDResolver) (int64, error) {
	result, err := db.QueryContext(ctx, `SELECT DISTINCT COALESCE(agency,''),COALESCE(advertiser,''),COALESCE(task_name,'') FROM settlement_rows ORDER BY 1,2,3`)
	if err != nil {
		return 0, err
	}
	var rows []SettlementRow
	for result.Next() {
		var r SettlementRow
		if err = result.Scan(&r.Agency, &r.Advertiser, &r.TaskName); err != nil {
			result.Close()
			return 0, err
		}
		rows = append(rows, r)
	}
	err = result.Err()
	result.Close()
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	projects, err := resolver.Resolve(ctx, rows)
	if err != nil {
		return 0, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	// Replace the old account-wide project cache only after every lookup succeeds.
	if _, err = tx.ExecContext(ctx, `DELETE FROM account_projects`); err != nil {
		return 0, err
	}
	// Use the same order as imports: projects before settlement rows.
	if err = saveProjects(ctx, tx, projects); err != nil {
		return 0, err
	}
	var count int64
	for _, r := range rows {
		if r.AgencyID < 0 || r.AdvertiserID < 0 {
			return 0, fmt.Errorf("账户解析未返回有效 ID")
		}
		res, err := tx.ExecContext(ctx, `UPDATE settlement_rows SET agency_id=$3,advertiser_id=$4 WHERE COALESCE(agency,'') COLLATE "C"=$1 AND COALESCE(advertiser,'') COLLATE "C"=$2 AND COALESCE(task_name,'') COLLATE "C"=$5`, r.Agency, r.Advertiser, r.AgencyID, r.AdvertiserID, r.TaskName)
		if err != nil {
			return 0, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return 0, err
		}
		count += n
	}
	return count, tx.Commit()
}
