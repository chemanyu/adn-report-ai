package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/chemanyu/adn-report-ai/internal/config"
	authhandler "github.com/chemanyu/adn-report-ai/internal/handler/auth"
	authlogic "github.com/chemanyu/adn-report-ai/internal/logic/auth"
	"github.com/chemanyu/adn-report-ai/internal/security"
	"github.com/chemanyu/adn-report-ai/internal/store"
	"github.com/chemanyu/adn-report-ai/internal/svc"
	"github.com/jackc/pgx/v5"
	"github.com/xuri/excelize/v2"
	"github.com/zeromicro/go-zero/core/conf"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// This test creates and deletes only its own uniquely named PostgreSQL schema.
func TestPostgreSQLWorkflowAndPermissions(t *testing.T) {
	if os.Getenv("ADN_TEST_POSTGRES") != "1" {
		t.Skip("set ADN_TEST_POSTGRES=1 for PostgreSQL integration test")
	}
	var c config.Config
	if dsn := os.Getenv("ADN_TEST_DATABASE_URL"); dsn != "" {
		c.PostgreSQL.DSN = dsn
	} else if e := conf.Load("../../etc/config.yaml", &c); e != nil {
		t.Fatal("无法读取本地测试连接配置")
	}
	c.Host = "127.0.0.1"
	c.Port = 18080
	c.BaseURL = "http://127.0.0.1:18080"
	c.DataDir = t.TempDir()
	c.PostgreSQL.Schema = fmt.Sprintf("adn_report_test_%d", time.Now().UnixNano())
	c.Auth.AdminUsername = "admin"
	c.Auth.AdminPassword = "test-password-12345"
	c.Auth.AdminUnionIDs = []string{"boss"}
	db, e := store.Open(c)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	defer func() {
		if _, e := db.Exec("DROP SCHEMA " + pgx.Identifier{c.PostgreSQL.Schema}.Sanitize() + " CASCADE"); e != nil {
			t.Error(e)
		}
	}()
	// Startup remains repeatable and table/column comments are present in PostgreSQL catalogs.
	again, e := store.Open(c)
	if e != nil {
		t.Fatal(e)
	}
	again.Close()
	var tables, columns int
	e = db.QueryRow(`SELECT COUNT(*),COUNT(*) FILTER (WHERE obj_description(oid,'pg_class') IS NOT NULL) FROM pg_class WHERE relnamespace=$1::regnamespace AND relkind='r'`, c.PostgreSQL.Schema).Scan(&tables, &columns)
	if e != nil || tables != 6 || columns != 6 {
		t.Fatalf("table comments: %d/%d %v", columns, tables, e)
	}
	e = db.QueryRow(`SELECT COUNT(*) FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid WHERE c.relnamespace=$1::regnamespace AND c.relkind='r' AND a.attnum>0 AND NOT a.attisdropped AND col_description(c.oid,a.attnum) IS NOT NULL`, c.PostgreSQL.Schema).Scan(&columns)
	if e != nil || columns != 44 {
		t.Fatalf("column comments: %d %v", columns, e)
	}
	os.MkdirAll(filepath.Join(c.DataDir, "csv"), 0700)
	svcCtx := svc.NewServiceContext(c, db)
	handler := New(svcCtx, fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ADN")}})
	auth := authlogic.NewAuthLogic(context.Background(), svcCtx)
	login := func(identity, name string) *http.Cookie {
		token, err := auth.SignIn(identity, name, "")
		if err != nil {
			t.Fatal(err)
		}
		return &http.Cookie{Name: "adn_session", Value: token}
	}

	alice := login("ding:alice", "运营甲")
	bob := login("ding:bob", "运营乙")
	boss := login("ding:boss", "负责人")
	call := func(method, path string, body io.Reader, contentType string, cookie *http.Cookie, origin bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, body)
		if origin {
			r.Header.Set("Origin", c.BaseURL)
			r.Header.Set("X-Requested-With", "ADN")
		}
		if contentType != "" {
			r.Header.Set("Content-Type", contentType)
		}
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	check := func(w *httptest.ResponseRecorder, status int) {
		t.Helper()
		if w.Code != status {
			t.Fatalf("status %d want %d: %s", w.Code, status, w.Body.String())
		}
	}
	check(call("GET", "/api/uploads", nil, "", nil, false), 401)
	check(call("GET", "/api/accounts", nil, "", alice, false), 404)
	check(call("POST", "/api/accounts", strings.NewReader(`{}`), "application/json", alice, true), 404)
	var w *httptest.ResponseRecorder
	f := excelize.NewFile()
	headers := []any{"代理商", "广告主", "任务名称", "日期", "结算数", "结算单价", "结算金额", "渠道", "ID"}
	row := []any{"代理甲", "广告甲", "任务甲", "2026-08-23", "10", "0.1", "1.0", "微博,汽水", "000123"}
	f.SetSheetRow("Sheet1", "A1", &headers)
	f.SetSheetRow("Sheet1", "A2", &row)
	buf, _ := f.WriteToBuffer()
	f.Close()
	upload := func(path string, data []byte, cookie *http.Cookie) *httptest.ResponseRecorder {
		var b bytes.Buffer
		mw := multipart.NewWriter(&b)
		part, _ := mw.CreateFormFile("file", "测试.xlsx")
		part.Write(data)
		mw.WriteField("sheet", "Sheet1")
		mw.WriteField("operator", "运营乙")
		mw.Close()
		return call("POST", path, &b, mw.FormDataContentType(), cookie, true)
	}
	check(upload("/api/preview", buf.Bytes(), alice), 200)
	check(upload("/api/uploads", buf.Bytes(), alice), 201)
	for _, path := range []string{"/api/uploads/1", "/api/uploads/1/csv"} {
		check(call("GET", path, nil, "", bob, false), 404)
		check(call("GET", path, nil, "", alice, false), 200)
		check(call("GET", path, nil, "", boss, false), 200)
	}
	for _, item := range []struct {
		cookie *http.Cookie
		total  int
	}{{alice, 1}, {bob, 0}, {boss, 1}} {
		w = call("GET", "/api/uploads", nil, "", item.cookie, false)
		check(w, 200)
		var list struct {
			Total int `json:"total"`
		}
		json.Unmarshal(w.Body.Bytes(), &list)
		if list.Total != item.total {
			t.Fatalf("scope total %d want %d", list.Total, item.total)
		}
	}
	w = call("GET", "/api/uploads/1/csv", nil, "", alice, false)
	if !strings.Contains(w.Body.String(), `"微博,汽水",000123`) {
		t.Fatal(w.Body.String())
	}
	w = call("GET", "/api/uploads/1", nil, "", alice, false)
	if !strings.Contains(w.Body.String(), `"operator":"运营乙"`) || !strings.Contains(w.Body.String(), `"uploader":"运营甲"`) {
		t.Fatal(w.Body.String())
	}
	// PostgreSQL placeholder numbering changes with owner scope and optional filters.
	for _, item := range []struct {
		cookie *http.Cookie
		total  int
	}{{alice, 1}, {bob, 0}, {boss, 1}} {
		for _, suffix := range []string{"?q=" + url.QueryEscape("测试"), "?q=" + url.QueryEscape("广告甲"), "?q=" + url.QueryEscape("运营乙")} {
			w = call("GET", "/api/uploads"+suffix, nil, "", item.cookie, false)
			check(w, 200)
			var list struct {
				Total int `json:"total"`
			}
			if e = json.Unmarshal(w.Body.Bytes(), &list); e != nil {
				t.Fatal(e)
			}
			if list.Total != item.total {
				t.Fatalf("filtered scope: got %d want %d", list.Total, item.total)
			}
		}
	}
	w = call("GET", "/api/uploads?page=2", nil, "", alice, false)
	check(w, 200)
	if !strings.Contains(w.Body.String(), `"items":[]`) {
		t.Fatal(w.Body.String())
	}
	var fixed, extra, agency string
	e = db.QueryRow("SELECT amount,extra_fields->>'ID',agency FROM settlement_rows WHERE upload_id=1").Scan(&fixed, &extra, &agency)
	if e != nil || fixed != "1.000000" || extra != "000123" || agency != "代理甲" {
		t.Fatalf("stored values %s %s %s: %v", fixed, extra, agency, e)
	}
	// Failed imports leave neither records nor additional CSV files.
	f = excelize.NewFile()
	badHeaders := []any{"日期", "结算金额"}
	f.SetSheetRow("Sheet1", "A1", &badHeaders)
	bad, _ := f.WriteToBuffer()
	f.Close()
	check(upload("/api/uploads", bad.Bytes(), alice), 422)
	files, _ := filepath.Glob(filepath.Join(c.DataDir, "csv", "*"))
	if len(files) != 1 {
		t.Fatalf("orphan files: %v", files)
	}
	t.Run("business key updates", func(t *testing.T) {
		checkBusinessKeyUpdates(t, c, db, handler, alice, bob, boss)
	})
	check(call("POST", "/api/auth/local", strings.NewReader(`{"username":"admin","password":"wrong"}`), "application/json", nil, true), 401)
	w = call("POST", "/api/auth/local", strings.NewReader(`{"username":"admin","password":"test-password-12345"}`), "application/json", nil, true)
	check(w, 200)
	local := w.Result().Cookies()[0]
	w = call("GET", "/api/me", nil, "", local, false)
	if !strings.Contains(w.Body.String(), `"admin":true`) {
		t.Fatal(w.Body.String())
	}
	check(call("POST", "/api/auth/logout", nil, "", alice, true), 200)
	check(call("GET", "/api/me", nil, "", alice, false), 401)
	_, e = db.ExecContext(context.Background(), "UPDATE sessions SET expires_at=$1 WHERE token_hash=$2", time.Now().Add(-time.Hour), security.Hash(bob.Value))
	if e != nil {
		t.Fatal(e)
	}
	check(call("GET", "/api/me", nil, "", bob, false), 401)
	check(call("GET", "/auth/callback?state=invalid&code=fake", nil, "", nil, false), 303)
	// Exercise the OAuth exchange with mocked DingTalk responses and real PostgreSQL storage.
	svcCtx.Config.Auth.ClientID = "test-client"
	svcCtx.Config.Auth.ClientSecret = "test-secret"
	exchanges := 0
	svcCtx.DingTalk.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		exchanges++
		body := `{"accessToken":"mock-token"}`
		if r.URL.Path == "/v1.0/contact/users/me" {
			if r.Header.Get("x-acs-dingtalk-access-token") != "mock-token" {
				t.Fatal("missing user access token")
			}
			body = `{"unionId":"oauth-member","nick":"钉钉测试成员"}`
		} else if r.URL.Path != "/v1.0/oauth2/userAccessToken" || r.Method != "POST" {
			t.Fatal("unexpected OAuth request")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
	})}
	start := httptest.NewRecorder()
	authhandler.LoginHandler(svcCtx)(start, httptest.NewRequest("GET", "/auth/login", nil))
	check(start, 302)
	redirect, _ := url.Parse(start.Header().Get("Location"))
	state := redirect.Query().Get("state")
	if redirect.Query().Get("redirect_uri") != c.BaseURL+"/auth/callback" {
		t.Fatal("wrong callback URL")
	}
	cb := httptest.NewRequest("GET", "/auth/callback?state="+state+"&authCode=test-code", nil)
	cb.AddCookie(start.Result().Cookies()[0])
	callback := httptest.NewRecorder()
	authhandler.CallbackHandler(svcCtx)(callback, cb)
	check(callback, 303)
	if exchanges != 2 || callback.Header().Get("Location") != "/" {
		t.Fatalf("OAuth exchange failed: %d", exchanges)
	}
	var oauthCookie *http.Cookie
	for _, cookie := range callback.Result().Cookies() {
		if cookie.Name == "adn_session" {
			oauthCookie = cookie
		}
	}
	if oauthCookie == nil || !oauthCookie.HttpOnly || oauthCookie.SameSite != http.SameSiteLaxMode {
		t.Fatal("session cookie missing protections")
	}
	w = call("GET", "/api/me", nil, "", oauthCookie, false)
	check(w, 200)
	if !strings.Contains(w.Body.String(), `"name":"钉钉测试成员"`) {
		t.Fatal(w.Body.String())
	}
	replay := httptest.NewRecorder()
	authhandler.CallbackHandler(svcCtx)(replay, cb)
	if exchanges != 2 || replay.Header().Get("Location") != "/?error=oauth_state" {
		t.Fatal("OAuth state replay accepted")
	}
}
