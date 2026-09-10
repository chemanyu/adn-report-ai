package middleware

import (
	"net/http"

	"github.com/chemanyu/adn-report-ai/internal/errcode"
	"github.com/chemanyu/adn-report-ai/internal/response"
)

func Security(baseURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "same-origin")
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
			if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" {
				if r.Header.Get("Origin") != baseURL || r.Header.Get("X-Requested-With") != "ADN" {
					response.Error(w, errcode.New(403, "请求来源无效，请使用配置的站点地址访问"))
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
