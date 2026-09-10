package handler

import (
	"net/http"

	"github.com/chemanyu/adn-report-ai/internal/handler/auth"
	"github.com/chemanyu/adn-report-ai/internal/handler/report"
	"github.com/chemanyu/adn-report-ai/internal/middleware"
	"github.com/chemanyu/adn-report-ai/internal/svc"
	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, svcCtx *svc.ServiceContext) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/auth/config", Handler: auth.ConfigHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/auth/login", Handler: auth.LoginHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/auth/callback", Handler: auth.CallbackHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/api/auth/local", Handler: auth.LocalLoginHandler(svcCtx)},
	})
	protected := middleware.Auth(svcCtx)
	server.AddRoutes(rest.WithMiddleware(protected, []rest.Route{
		{Method: http.MethodPost, Path: "/api/auth/logout", Handler: auth.LogoutHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/me", Handler: auth.MeHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/uploads", Handler: report.ListHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/uploads/:id", Handler: report.DetailHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/uploads/:id/csv", Handler: report.DownloadHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/template", Handler: report.TemplateHandler(svcCtx)},
	}...))
	limit := middleware.ImportLimit(svcCtx)
	server.AddRoutes(rest.WithMiddleware(protected, []rest.Route{
		{Method: http.MethodPost, Path: "/api/preview", Handler: limit(report.PreviewHandler(svcCtx))},
		{Method: http.MethodPost, Path: "/api/uploads", Handler: limit(report.UploadHandler(svcCtx))},
	}...))
}
