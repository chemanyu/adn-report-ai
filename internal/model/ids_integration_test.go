package model_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/chemanyu/adn-report-ai/internal/config"
	"github.com/chemanyu/adn-report-ai/internal/model"
	"github.com/chemanyu/adn-report-ai/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/zeromicro/go-zero/core/conf"
)

type resolver struct{ fail bool }

func (r resolver) Resolve(_ context.Context, rows []model.SettlementRow) ([]model.Project, error) {
	if r.fail {
		return nil, fmt.Errorf("ambiguous account")
	}
	for i := range rows {
		rows[i].AgencyID = 10
		rows[i].AdvertiserID = 20
	}
	return []model.Project{{ID: 30, Name: "任务", AdvertiserID: 20}}, nil
}
func TestIDsTransaction(t *testing.T) {
	if os.Getenv("ADN_TEST_POSTGRES") != "1" {
		t.Skip("requires isolated PostgreSQL schema")
	}
	var c config.Config
	if dsn := os.Getenv("ADN_TEST_DATABASE_URL"); dsn != "" {
		c.PostgreSQL.DSN = dsn
	} else if err := conf.Load("../../etc/config.yaml", &c); err != nil {
		t.Fatal("cannot load test database config")
	}
	c.PostgreSQL.Schema = fmt.Sprintf("adn_report_test_ids_%d", time.Now().UnixNano())
	db, err := store.Open(c)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	defer func() {
		if _, err := db.Exec("DROP SCHEMA " + pgx.Identifier{c.PostgreSQL.Schema}.Sanitize() + " CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	again, err := store.Open(c)
	if err != nil {
		t.Fatal(err)
	}
	again.Close()
	var uid int64
	if err = db.QueryRow(`INSERT INTO users(identity_key,name) VALUES('test','test') RETURNING id`).Scan(&uid); err != nil {
		t.Fatal(err)
	}
	in := model.NewUpload{Upload: model.Upload{UserID: uid, Uploader: "test", Operator: "test", Filename: "test.xlsx", Sheet: "Sheet1", Columns: []byte(`["代理商","广告主","日期","任务名称","结算数","结算单价","结算金额"]`), RowCount: 1, Total: "1", DateFrom: "2026-09-16", DateTo: "2026-09-16", CSVPath: "test.csv"}, Hash: "hash", Rows: []model.SettlementRow{{SourceRow: 2, Agency: "代理", Advertiser: "账户", TaskName: "任务", Date: "2026-09-16", Amount: "1", Extra: map[string]string{}}}}
	ctx := context.Background()
	old := model.NewUploadModel(db)
	first, err := old.SaveWithRows(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := model.BackfillIDs(ctx, db, resolver{}); err != nil || n != 1 {
		t.Fatalf("backfill %d %v", n, err)
	}
	m := model.NewUploadModelWithResolver(db, resolver{})
	in.Rows[0].Amount = "2"
	saved, err := m.SaveWithRows(ctx, in)
	if err != nil || saved.ID != first.ID || saved.Updated != 1 {
		t.Fatalf("save %+v %v", saved, err)
	}
	_, rows, err := m.Snapshot(ctx, saved.ID, &uid, 0)
	if err != nil || len(rows) != 1 || rows[0].AgencyID != 10 || rows[0].AdvertiserID != 20 || rows[0].Amount != "2.000000" || len(rows[0].ProjectIDs) != 1 || rows[0].ProjectIDs[0] != "30" {
		t.Fatalf("snapshot %+v %v", rows, err)
	}
	var name string
	if err = db.QueryRow(`SELECT project_name FROM account_projects WHERE project_id=30 AND advertiser_id=20`).Scan(&name); err != nil || name != "任务" {
		t.Fatalf("project %s %v", name, err)
	}
	if _, err = model.NewUploadModelWithResolver(db, resolver{fail: true}).SaveWithRows(ctx, in); err == nil {
		t.Fatal("accepted ambiguity")
	}
	// Zero must overwrite previously resolved IDs and distinct accounts can each have project 0.
	if _, err = model.NewUploadModelWithResolver(db, zeroResolver{}).SaveWithRows(ctx, in); err != nil {
		t.Fatal(err)
	}
	var a, b int64
	if err = db.QueryRow(`SELECT agency_id,advertiser_id FROM settlement_rows WHERE upload_id=$1`, saved.ID).Scan(&a, &b); err != nil || a != 0 || b != 0 {
		t.Fatalf("zero IDs %d %d %v", a, b, err)
	}
	var placeholders int
	if err = db.QueryRow(`SELECT COUNT(*) FROM account_projects WHERE project_id=0`).Scan(&placeholders); err != nil || placeholders != 2 {
		t.Fatalf("placeholders %d %v", placeholders, err)
	}
	if n, err := model.BackfillIDs(ctx, db, resolver{}); err != nil || n != 1 {
		t.Fatalf("retry zeros %d %v", n, err)
	}
	// A database error after project upsert must roll back mappings and history.
	if _, err = db.Exec(`DELETE FROM account_projects`); err != nil {
		t.Fatal(err)
	}
	in.Rows[0].Amount = "invalid"
	if _, err = m.SaveWithRows(ctx, in); err == nil {
		t.Fatal("accepted bad amount")
	}
	var n int
	if err = db.QueryRow(`SELECT COUNT(*) FROM account_projects`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("mapping survived rollback %d %v", n, err)
	}
	if err = db.QueryRow(`SELECT COUNT(*) FROM upload_history`).Scan(&n); err != nil || n != 3 {
		t.Fatalf("history changed %d %v", n, err)
	}
}

type zeroResolver struct{}

func (zeroResolver) Resolve(_ context.Context, rows []model.SettlementRow) ([]model.Project, error) {
	for i := range rows {
		rows[i].AgencyID = 0
		rows[i].AdvertiserID = 0
	}
	return []model.Project{{ID: 0, AdvertiserID: 0}, {ID: 0, AdvertiserID: 21}}, nil
}
