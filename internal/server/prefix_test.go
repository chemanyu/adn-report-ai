package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/chemanyu/adn-report-ai/internal/config"
	authhandler "github.com/chemanyu/adn-report-ai/internal/handler/auth"
	"github.com/chemanyu/adn-report-ai/internal/model"
	"github.com/chemanyu/adn-report-ai/internal/svc"
	"github.com/zeromicro/go-zero/core/conf"
)

type consumedState struct{ model.AuthModel }

func (consumedState) ConsumeOAuthState(context.Context, string) (bool, error) { return false, nil }

func TestReverseProxyPrefix(t *testing.T) {
	var c config.Config
	if err := conf.FillDefault(&c.RestConf); err != nil {
		t.Fatal(err)
	}
	c.Name = "adn-prefix-test"
	c.Port = 18080
	c.BaseURL = "http://172.16.3.34/adn-report"
	svcCtx := svc.NewServiceContext(c, nil)
	server := New(svcCtx, fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ADN")}})
	// nginx strips /adn-report/ before proxying, but Origin never contains a path.
	for _, tc := range []struct {
		origin string
		status int
	}{{"http://172.16.3.34", 401}, {"http://172.16.3.34/adn-report", 403}, {"http://foreign.example", 403}} {
		r := httptest.NewRequest("POST", "/api/auth/local", strings.NewReader(`{"username":"admin","password":"wrong"}`))
		r.Header.Set("Origin", tc.origin)
		r.Header.Set("X-Requested-With", "ADN")
		w := httptest.NewRecorder()
		server.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("origin=%s status=%d", tc.origin, w.Code)
		}
	}
	svcCtx.AuthModel = consumedState{}
	r := httptest.NewRequest("GET", "/auth/callback?state=used&code=fake", nil)
	r.AddCookie(&http.Cookie{Name: "adn_oauth", Value: "used"})
	w := httptest.NewRecorder()
	authhandler.CallbackHandler(svcCtx)(w, r)
	if w.Header().Get("Location") != "/adn-report/?error=oauth_state" {
		t.Fatal("callback escaped application prefix")
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Path != "/adn-report/" || !cookies[0].HttpOnly {
		t.Fatal("cookie path/protection is incorrect")
	}
}
