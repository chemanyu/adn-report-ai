package report

import (
	"mime"
	"net/http"
	"strconv"
	"time"

	reportlogic "github.com/chemanyu/adn-report-ai/internal/logic/report"
	"github.com/chemanyu/adn-report-ai/internal/middleware"
	"github.com/chemanyu/adn-report-ai/internal/response"
	"github.com/chemanyu/adn-report-ai/internal/svc"
	"github.com/chemanyu/adn-report-ai/internal/types"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func PreviewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		in, err := readMultipart(w, r)
		if err != nil {
			response.Error(w, err)
			return
		}
		result, err := reportlogic.NewReportLogic(r.Context(), svcCtx).Preview(in)
		if err != nil {
			response.Error(w, err)
			return
		}
		response.JSON(w, 200, result)
	}
}
func UploadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		in, err := readMultipart(w, r)
		if err != nil {
			response.Error(w, err)
			return
		}
		result, err := reportlogic.NewReportLogic(r.Context(), svcCtx).Upload(in, middleware.CurrentUser(r.Context()))
		if err != nil {
			response.Error(w, err)
			return
		}
		response.JSON(w, 201, result)
	}
}
func ListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		page, _ := strconv.Atoi(q.Get("page"))
		in := types.ListUploadsRequest{Page: page, Query: q.Get("q")}
		result, err := reportlogic.NewReportLogic(r.Context(), svcCtx).List(in, middleware.CurrentUser(r.Context()))
		if err != nil {
			response.Error(w, err)
			return
		}
		response.JSON(w, 200, result)
	}
}
func DetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		in := types.UploadDetailRequest{ID: pathvar.Vars(r)["id"], Page: page}
		result, err := reportlogic.NewReportLogic(r.Context(), svcCtx).Detail(in, middleware.CurrentUser(r.Context()))
		if err != nil {
			response.Error(w, err)
			return
		}
		response.JSON(w, 200, result)
	}
}
func DownloadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, err := reportlogic.NewReportLogic(r.Context(), svcCtx).OpenDownload(pathvar.Vars(r)["id"], middleware.CurrentUser(r.Context()))
		if err != nil {
			response.Error(w, err)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": file.Name}))
		http.ServeContent(w, r, file.Name, time.Time{}, file.File)
	}
}
func TemplateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := reportlogic.NewReportLogic(r.Context(), svcCtx).Template()
		if err != nil {
			response.Error(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", `attachment; filename="adn-settlement-template.xlsx"`)
		w.Write(data)
	}
}
