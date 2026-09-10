package report

import (
	"io"
	"net/http"

	"github.com/chemanyu/adn-report-ai/internal/errcode"
	"github.com/chemanyu/adn-report-ai/internal/types"
)

// readMultipart only decodes transport data; workbook validation is in logic.
func readMultipart(w http.ResponseWriter, r *http.Request) (types.FileRequest, error) {
	var in types.FileRequest
	r.Body = http.MaxBytesReader(w, r.Body, 21<<20)
	err := r.ParseMultipartForm(22 << 20)
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if err != nil {
		return in, errcode.New(400, "文件过大或上传格式错误，最大 20 MB")
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return in, errcode.New(400, "请选择 Excel 文件")
	}
	defer file.Close()
	if header.Size > 20<<20 {
		return in, errcode.New(400, "仅支持 20 MB 以内的 .xlsx 文件")
	}
	in.Data, err = io.ReadAll(io.LimitReader(file, (20<<20)+1))
	if err != nil || len(in.Data) > 20<<20 {
		return in, errcode.New(400, "读取文件失败或文件超过 20 MB")
	}
	in.Filename = header.Filename
	in.Sheet = r.FormValue("sheet")
	in.Operator = r.FormValue("operator")
	return in, nil
}
