package response

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/chemanyu/adn-report-ai/internal/errcode"
)

func JSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}
func Error(w http.ResponseWriter, err error) {
	var known *errcode.Error
	if errors.As(err, &known) {
		body := map[string]any{"error": known.Message}
		if len(known.Details) > 0 {
			body["errors"] = known.Details
		}
		JSON(w, known.Code, body)
		return
	}
	log.Printf("request failed: %v", err)
	JSON(w, 500, map[string]string{"error": "服务处理失败，请稍后重试或查看服务日志"})
}
func Decode(w http.ResponseWriter, r *http.Request, value any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		Error(w, errcode.New(400, "请求参数格式错误"))
		return false
	}
	if decoder.Decode(new(any)) != io.EOF {
		Error(w, errcode.New(400, "请求参数格式错误"))
		return false
	}
	return true
}
func CookieValue(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}
