package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/chemanyu/adn-report-ai/internal/config"
	"github.com/chemanyu/adn-report-ai/internal/svc"
	"github.com/zeromicro/go-zero/core/conf"
)

func TestRESTRoutingAndStaticAssets(t *testing.T) {
	var c config.Config
	if err := conf.FillDefault(&c.RestConf); err != nil {
		t.Fatal(err)
	}
	c.Name = "adn-routing-test"
	c.BaseURL = "http://127.0.0.1:18080"
	c.Port = 18080
	srv := New(svc.NewServiceContext(c, nil), fstest.MapFS{
		"index.html":    &fstest.MapFile{Data: []byte("<div id=app></div>")},
		"assets/app.js": &fstest.MapFile{Data: []byte("/* Vue bundle */")},
	})
	for _, tc := range []struct {
		method, path string
		status       int
		text         string
	}{
		{"GET", "/", 200, "<div id=app>"},
		{"HEAD", "/", 200, ""},
		{"GET", "/assets/app.js", 200, "Vue bundle"},
		{"GET", "/src/App.vue", 404, ""},
		{"GET", "/api/missing", 404, "接口不存在"},
		{"GET", "/api/me", 401, ""},
		{"GET", "/api/uploads/123", 401, ""},
		{"GET", "/api/uploads/123/csv", 401, ""},
		{"POST", "/api/auth/local", 403, "请求来源无效"},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.text) {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if w.Header().Get("X-Frame-Options") != "DENY" {
				t.Fatal("missing security headers")
			}
		})
	}
	// The framework's default 1 MB limit must not reject supported Excel requests
	// before they reach application authentication/import validation.
	r := httptest.NewRequest(http.MethodPost, "/api/preview", bytes.NewReader(make([]byte, 2<<20)))
	r.Header.Set("Origin", c.BaseURL)
	r.Header.Set("X-Requested-With", "ADN")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("2 MB request rejected before authentication: %d", w.Code)
	}
}
