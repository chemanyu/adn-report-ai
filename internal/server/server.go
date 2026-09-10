// Package server assembles the REST server and embedded static file handler.
package server

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/chemanyu/adn-report-ai/internal/errcode"
	"github.com/chemanyu/adn-report-ai/internal/handler"
	"github.com/chemanyu/adn-report-ai/internal/middleware"
	"github.com/chemanyu/adn-report-ai/internal/response"
	"github.com/chemanyu/adn-report-ai/internal/svc"
	"github.com/zeromicro/go-zero/rest"
)

func New(svcCtx *svc.ServiceContext, web fs.FS) *rest.Server {
	c := svcCtx.Config
	security := middleware.Security(c.Origin())
	// Existing configuration files remain usable. Imports need more than the
	// framework's default 1 MB / 3 seconds. The importer enforces the 20 MB file limit.
	c.RestConf.MaxBytes = 21 << 20
	c.RestConf.Timeout = 180000
	// go-zero's request logger can dump credentials on failed/slow requests.
	c.RestConf.Middlewares.Log = false
	// Serve only the configured application port, without a separate pprof server.
	c.RestConf.DevServer.Enabled = false
	fallback := security(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/auth/") {
			response.Error(w, errcode.New(404, "接口不存在"))
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		http.FileServer(http.FS(web)).ServeHTTP(w, r)
	}))
	srv := rest.MustNewServer(c.RestConf, rest.WithNotFoundHandler(fallback), rest.WithNotAllowedHandler(security(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.Error(w, errcode.New(405, "请求方法不支持"))
	}))))
	srv.Use(rest.ToMiddleware(security))
	handler.RegisterHandlers(srv, svcCtx)
	return srv
}
