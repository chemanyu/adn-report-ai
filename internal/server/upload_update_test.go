package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/chemanyu/adn-report-ai/internal/config"
	"github.com/chemanyu/adn-report-ai/internal/store"
	"github.com/chemanyu/adn-report-ai/internal/svc"
	"github.com/chemanyu/adn-report-ai/internal/types"
	"github.com/xuri/excelize/v2"
)

// Uses only the parent test's disposable schema and temporary CSV directory.
func checkBusinessKeyUpdates(t *testing.T, c config.Config, db *sql.DB, handler http.Handler, alice, bob, boss *http.Cookie) {
	workbook := func(sheet string, rows ...[]any) []byte {
		t.Helper()
		f := excelize.NewFile()
		defer f.Close()
		if err := f.SetSheetName("Sheet1", sheet); err != nil {
			t.Fatal(err)
		}
		headers := []any{"代理商", "广告主", "日期", "任务名称", "结算数", "结算单价", "结算金额", "备注"}
		if err := f.SetSheetRow(sheet, "A1", &headers); err != nil {
			t.Fatal(err)
		}
		for i, row := range rows {
			if err := f.SetSheetRow(sheet, fmt.Sprintf("A%d", i+2), &row); err != nil {
				t.Fatal(err)
			}
		}
		b, err := f.WriteToBuffer()
		if err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	row := func(day, task, amount string) []any {
		return []any{"代理商甲", "广告主甲", day, task, "1", amount, amount, "新备注"}
	}
	upload := func(name, sheet string, data []byte, cookie *http.Cookie, targets ...http.Handler) *httptest.ResponseRecorder {
		var b bytes.Buffer
		mw := multipart.NewWriter(&b)
		part, _ := mw.CreateFormFile("file", name)
		part.Write(data)
		mw.WriteField("sheet", sheet)
		mw.WriteField("operator", "新负责人")
		mw.Close()
		req := httptest.NewRequest("POST", "/api/uploads", &b)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		req.Header.Set("Origin", c.Origin())
		req.Header.Set("X-Requested-With", "ADN")
		req.AddCookie(cookie)
		target := handler
		if len(targets) > 0 {
			target = targets[0]
		}
		w := httptest.NewRecorder()
		target.ServeHTTP(w, req)
		return w
	}
	saved := func(w *httptest.ResponseRecorder, inserted, updated int) types.UploadResponse {
		t.Helper()
		if w.Code != 201 {
			t.Fatalf("upload: %d %s", w.Code, w.Body.String())
		}
		var v types.UploadResponse
		if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
			t.Fatal(err)
		}
		if v.Inserted != inserted || v.Updated != updated {
			t.Fatalf("counts=%+v want %d/%d", v, inserted, updated)
		}
		return v
	}
	get := func(id int64, csv bool, cookie *http.Cookie) *httptest.ResponseRecorder {
		path := fmt.Sprintf("/api/uploads/%d", id)
		if csv {
			path += "/csv"
		}
		req := httptest.NewRequest("GET", path, nil)
		req.AddCookie(cookie)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}
	assertBatch := func(id int64, n int, amount string) {
		t.Helper()
		var count, actual int
		var total, sum string
		err := db.QueryRow(`SELECT u.row_count,u.total_amount,COUNT(r.id),COALESCE(SUM(r.amount),0)::numeric(30,6) FROM uploads u LEFT JOIN settlement_rows r ON r.upload_id=u.id WHERE u.id=$1 GROUP BY u.id`, id).Scan(&count, &total, &actual, &sum)
		if err != nil || count != n || actual != n || total != amount || sum != amount {
			t.Fatalf("batch %d: %d %d %s %s %v", id, count, actual, total, sum, err)
		}
	}
	day := "2026-09-01"
	task := "更新任务"
	first := saved(upload("第一份.xlsx", "Sheet1", workbook("Sheet1", row(day, task, "2"), row("2026-09-02", task, "3")), alice), 2, 0)
	var rowID int64
	if err := db.QueryRow(`SELECT id FROM settlement_rows WHERE upload_id=$1 AND settlement_date=$2`, first.ID, day).Scan(&rowID); err != nil {
		t.Fatal(err)
	}
	second := saved(upload("改名.xlsx", "另一个工作表", workbook("另一个工作表", row(day, task, "9")), alice), 0, 1)
	assertBatch(first.ID, 1, "3.000000")
	assertBatch(second.ID, 1, "9.000000")
	var currentID int64
	if err := db.QueryRow(`SELECT id FROM settlement_rows WHERE upload_id=$1`, second.ID).Scan(&currentID); err != nil || currentID != rowID {
		t.Fatalf("row ID changed: %v", err)
	}
	csv := get(first.ID, true, alice)
	if csv.Code != 200 || bytes.Contains(csv.Body.Bytes(), []byte(day)) || !bytes.Contains(csv.Body.Bytes(), []byte("2026-09-02")) {
		t.Fatalf("old batch CSV is stale: %s", csv.Body.String())
	}
	// Empty and zero are different. An explicit empty value clears an earlier amount.
	nullable := row(day, task, "")
	nullable[4] = "0"
	third := saved(upload("改名.xlsx", "Sheet1", workbook("Sheet1", nullable), alice), 0, 1)
	assertBatch(second.ID, 0, "0.000000")
	assertBatch(third.ID, 1, "0.000000")
	var count string
	var price, amount sql.NullString
	if err := db.QueryRow(`SELECT settlement_count,unit_price,amount FROM settlement_rows WHERE upload_id=$1`, third.ID).Scan(&count, &price, &amount); err != nil || count != "0.000000" || price.Valid || amount.Valid {
		t.Fatalf("nulls changed: %s %+v %+v %v", count, price, amount, err)
	}
	detail := get(third.ID, false, alice)
	if detail.Code != 200 || !bytes.Contains(detail.Body.Bytes(), []byte(`"amount":""`)) || bytes.Contains(detail.Body.Bytes(), []byte(`"source":`)) {
		t.Fatal(detail.Body.String())
	}
	// Advertiser is the account; different advertisers and login users stay independent.
	otherAdvertiser := row(day, task, "7")
	otherAdvertiser[1] = "独立广告主"
	data := workbook("Sheet1", row(day, task, "7"))
	isolated := []types.UploadResponse{
		saved(upload("相同.xlsx", "Sheet1", workbook("Sheet1", otherAdvertiser), alice), 1, 0),
		saved(upload("相同.xlsx", "Sheet1", data, bob), 1, 0),
		saved(upload("相同.xlsx", "Sheet1", data, boss), 1, 0),
	}
	for _, v := range isolated {
		assertBatch(v.ID, 1, "7.000000")
	}
	if get(third.ID, false, bob).Code != 404 || get(third.ID, true, bob).Code != 404 {
		t.Fatal("cross-user read allowed")
	}
	// A mixed upload updates one key and inserts another; omitted dates remain untouched.
	mixed := saved(upload("混合.xlsx", "Sheet1", workbook("Sheet1", row(day, task, "11"), row(day, "新任务", "5")), alice), 1, 1)
	assertBatch(first.ID, 1, "3.000000")
	assertBatch(mixed.ID, 2, "16.000000")
	for _, v := range isolated {
		assertBatch(v.ID, 1, "7.000000")
	}
	multi := saved(upload("多广告主.xlsx", "Sheet1", workbook("Sheet1", row("2026-09-05", "多账户任务", "2"), otherAdvertiser), alice), 1, 1)
	multiDetail := get(multi.ID, false, alice)
	var multiResult types.UploadDetailResponse
	if err := json.Unmarshal(multiDetail.Body.Bytes(), &multiResult); err != nil {
		t.Fatal(err)
	}
	if len(multiResult.Upload.Advertisers) != 2 {
		t.Fatalf("advertisers: %v", multiResult.Upload.Advertisers)
	}
	req := httptest.NewRequest("GET", "/api/uploads?q="+url.QueryEscape("多广告主"), nil)
	req.AddCookie(alice)
	summary := httptest.NewRecorder()
	handler.ServeHTTP(summary, req)
	var listed types.UploadListResponse
	if err := json.Unmarshal(summary.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if summary.Code != 200 || listed.AdvertiserCount != 2 || listed.Total != 1 || listed.RowCount != 2 || listed.TotalAmount != "9.000000" {
		t.Fatalf("advertiser summary: %+v %d", listed, summary.Code)
	}
	// Validation and mid-transaction database failures keep every prior row and archive.
	before := get(mixed.ID, true, alice).Body.String()
	archives, _ := filepath.Glob(filepath.Join(c.DataDir, "csv", "*"))
	invalid := upload("失败.xlsx", "Sheet1", workbook("Sheet1", row(day, task, "1"), row(day, task, "2")), alice)
	if invalid.Code != 422 {
		t.Fatalf("duplicate keys accepted: %d", invalid.Code)
	}
	if _, err := db.Exec(`ALTER TABLE settlement_rows ADD CONSTRAINT test_row_failure CHECK(amount<>999)`); err != nil {
		t.Fatal(err)
	}
	failed := upload("失败.xlsx", "Sheet1", workbook("Sheet1", row(day, task, "12"), row(day, "失败任务", "999")), alice)
	if _, err := db.Exec(`ALTER TABLE settlement_rows DROP CONSTRAINT test_row_failure`); err != nil {
		t.Fatal(err)
	}
	if failed.Code != 500 {
		t.Fatalf("expected transaction failure: %d", failed.Code)
	}
	if get(mixed.ID, true, alice).Body.String() != before {
		t.Fatal("failed import changed old data")
	}
	after, _ := filepath.Glob(filepath.Join(c.DataDir, "csv", "*"))
	if len(after) != len(archives) {
		t.Fatal("failed import left CSV")
	}
	// Migrate legacy extra fields, remove source_values, retain incomplete legacy records.
	var legacyID, unknownID int64
	if _, err := db.Exec(`ALTER TABLE settlement_rows ADD COLUMN source_values JSONB`); err != nil {
		t.Fatal(err)
	}
	err := db.QueryRow(`INSERT INTO settlement_rows(upload_id,source_row,settlement_date,settlement_count,unit_price,amount,extra_fields,source_values) VALUES($1,100,'2026-09-03',1,4,4,'{"代理商":"历史代理","广告主":"历史广告","任务名称":"历史任务","保留":"00123"}','[]') RETURNING id`, first.ID).Scan(&legacyID)
	if err != nil {
		t.Fatal(err)
	}
	err = db.QueryRow(`INSERT INTO settlement_rows(upload_id,source_row,settlement_date,settlement_count,unit_price,amount,extra_fields) VALUES($1,101,'2026-09-03',1,8,8,'{}') RETURNING id`, first.ID).Scan(&unknownID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TABLE adn_accounts(id BIGINT PRIMARY KEY,name TEXT); INSERT INTO adn_accounts VALUES(1,'旧账户'); ALTER TABLE uploads ADD COLUMN account_id BIGINT DEFAULT 1 REFERENCES adn_accounts(id)`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		migrated, err := store.Open(c)
		if err != nil {
			t.Fatal(err)
		}
		migrated.Close()
	}
	var n int
	if err = db.QueryRow(`SELECT count(*) FROM information_schema.tables WHERE table_schema=current_schema() AND table_name='adn_accounts'`).Scan(&n); err != nil || n != 0 {
		t.Fatal("legacy account catalog retained")
	}
	if err = db.QueryRow(`SELECT count(*) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='uploads' AND column_name='account_id'`).Scan(&n); err != nil || n != 0 {
		t.Fatal("legacy account_id retained")
	}
	if err = db.QueryRow(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=current_schema() AND column_name='source_values'`).Scan(&n); err != nil || n != 0 {
		t.Fatal("source_values retained")
	}
	historical := []any{"历史代理", "历史广告", "2026-09-03", "历史任务", "0", "", "", "更新"}
	migrated := saved(upload("历史更新.xlsx", "Sheet1", workbook("Sheet1", historical), alice), 0, 1)
	if err = db.QueryRow(`SELECT id FROM settlement_rows WHERE upload_id=$1`, migrated.ID).Scan(&currentID); err != nil || currentID != legacyID {
		t.Fatal("legacy key not matched")
	}
	if err = db.QueryRow(`SELECT COUNT(*) FROM settlement_rows WHERE id=$1 AND agency IS NULL`, unknownID).Scan(&n); err != nil || n != 1 {
		t.Fatal("incomplete legacy row removed")
	}
	// Several instances sharing the same login user must not duplicate a new key.
	variants := [][]byte{workbook("Sheet1", row(day, "并发任务", "21")), workbook("Sheet1", row(day, "并发任务", "22"))}
	var wg sync.WaitGroup
	results := make(chan *httptest.ResponseRecorder, 8)
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		instance := New(svc.NewServiceContext(c, db), fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ADN")}})
		wg.Add(1)
		go func(data []byte, h http.Handler) {
			defer wg.Done()
			<-start
			results <- upload("并发.xlsx", "Sheet1", data, alice, h)
		}(variants[i%2], instance)
	}
	close(start)
	wg.Wait()
	close(results)
	inserted, updated := 0, 0
	for w := range results {
		if w.Code != 201 {
			t.Fatalf("concurrency: %d %s", w.Code, w.Body.String())
		}
		var v types.UploadResponse
		if err = json.Unmarshal(w.Body.Bytes(), &v); err != nil {
			t.Fatal(err)
		}
		inserted += v.Inserted
		updated += v.Updated
	}
	if inserted != 1 || updated != 7 {
		t.Fatalf("concurrent result %d/%d", inserted, updated)
	}
	if err = db.QueryRow(`SELECT COUNT(*) FROM settlement_rows WHERE task_name='并发任务'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("concurrent duplicate: %d %v", n, err)
	}
	// All immutable archives remain referenced, while dynamic downloads use current rows.
	rows, err := db.Query(`SELECT csv_path FROM uploads`)
	if err != nil {
		t.Fatal(err)
	}
	refs := map[string]bool{}
	for rows.Next() {
		var path string
		if err = rows.Scan(&path); err != nil {
			t.Fatal(err)
		}
		refs[path] = true
		if _, err = os.Stat(filepath.Join(c.DataDir, "csv", path)); err != nil {
			t.Fatal(err)
		}
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	files, _ := filepath.Glob(filepath.Join(c.DataDir, "csv", "*"))
	for _, file := range files {
		if !refs[filepath.Base(file)] {
			t.Fatalf("orphan CSV %s", file)
		}
	}
}
