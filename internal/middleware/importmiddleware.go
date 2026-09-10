package middleware

import (
	"net/http"

	"github.com/chemanyu/adn-report-ai/internal/errcode"
	"github.com/chemanyu/adn-report-ai/internal/response"
	"github.com/chemanyu/adn-report-ai/internal/svc"
)

// ImportLimit reserves capacity before multipart parsing and Excel decompression.
func ImportLimit(svcCtx *svc.ServiceContext) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			select {
			case svcCtx.ImportSlots <- struct{}{}:
				defer func() { <-svcCtx.ImportSlots }()
				next(w, r)
			default:
				response.Error(w, errcode.New(429, "当前有文件正在处理，请稍后重试"))
			}
		}
	}
}
