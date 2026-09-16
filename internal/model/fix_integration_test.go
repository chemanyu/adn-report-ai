package model_test

import (
	"context"
	"fmt"
	"github.com/chemanyu/adn-report-ai/internal/config"
	"github.com/chemanyu/adn-report-ai/internal/model"
	"github.com/chemanyu/adn-report-ai/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/zeromicro/go-zero/core/conf"
	"os"
	"testing"
	"time"
)

func TestFixBusinessKey(t *testing.T) {
	if os.Getenv("ADN_TEST_POSTGRES") != "1" {
		t.Skip("requires isolated PostgreSQL schema")
	}
	var c config.Config
	if dsn := os.Getenv("ADN_TEST_DATABASE_URL"); dsn != "" {
		c.PostgreSQL.DSN = dsn
	} else if err := conf.Load("../../etc/config.yaml", &c); err != nil {
		t.Fatal("cannot load test database config")
	}
	c.PostgreSQL.Schema = fmt.Sprintf("adn_report_test_fix_%d", time.Now().UnixNano())
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

	m := model.NewUploadModel(db)
	ctx := context.Background()
	save := func(fix *string, count string, inserted, updated int) {
		t.Helper()
		in.Rows[0].Extra = map[string]string{}
		if fix != nil {
			in.Rows[0].Extra["fix"] = *fix
		}
		in.Rows[0].Count = count
		saved, err := m.SaveWithRows(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if saved.Inserted != inserted || saved.Updated != updated {
			t.Fatalf("unexpected save: %+v", saved)
		}
	}
	a, b, empty := "A", "B", ""
	save(nil, "1", 1, 0)
	save(&a, "10", 1, 0)
	save(&b, "20", 1, 0)
	save(&a, "30", 0, 1)
	save(&empty, "2", 0, 1)
	save(nil, "3", 0, 1)
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM settlement_rows`).Scan(&n); err != nil || n != 3 {
		t.Fatalf("row count %d: %v", n, err)
	}
	var count string
	if err := db.QueryRow(`SELECT settlement_count::text FROM settlement_rows WHERE extra_fields->>'fix'='A'`).Scan(&count); err != nil || count != "30.000000" {
		t.Fatalf("count %s: %v", count, err)
	}
	// Different task names in one account each retain their own zero placeholder.
	m = model.NewUploadModelWithResolver(db, taskProjectResolver{})
	in.Rows[0].TaskName = "未匹配一"
	save(nil, "1", 1, 0)
	in.Rows[0].TaskName = "未匹配二"
	save(nil, "1", 1, 0)
	in.Rows[0].TaskName = "匹配任务"
	save(nil, "1", 1, 0)
	if err := db.QueryRow(`SELECT count(*) FROM account_projects WHERE advertiser_id=20 AND project_id=0`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("task placeholders %d: %v", n, err)
	}
	var uploadID int64
	if err := db.QueryRow(`SELECT id FROM uploads WHERE user_id=$1`, uid).Scan(&uploadID); err != nil {
		t.Fatal(err)
	}
	_, detail, err := m.Snapshot(ctx, uploadID, &uid, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range detail {
		want := "0"
		if row.TaskName == "匹配任务" {
			want = "99"
		}
		if len(row.ProjectIDs) != 1 || row.ProjectIDs[0] != want {
			t.Fatalf("wrong task project: %+v", row)
		}
	}

}

type taskProjectResolver struct{}

func (taskProjectResolver) Resolve(_ context.Context, rows []model.SettlementRow) ([]model.Project, error) {
	projects := []model.Project{}
	for i := range rows {
		rows[i].AgencyID, rows[i].AdvertiserID = 10, 20
		id := int64(0)
		if rows[i].TaskName == "匹配任务" {
			id = 99
		}
		projects = append(projects, model.Project{ID: id, Name: rows[i].TaskName, AdvertiserID: 20})
	}
	return projects, nil
}
