package report

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/chemanyu/adn-report-ai/internal/errcode"
	"github.com/chemanyu/adn-report-ai/internal/importer"
	"github.com/chemanyu/adn-report-ai/internal/model"
	"github.com/chemanyu/adn-report-ai/internal/types"
)

func (l *ReportLogic) findUpload(idText string, u types.User) (model.Upload, error) {
	id, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || id < 1 {
		return model.Upload{}, errcode.New(404, "记录不存在")
	}
	upload, err := l.svcCtx.UploadModel.FindScoped(l.ctx, id, ownerScope(u))
	if errors.Is(err, model.ErrNotFound) {
		return upload, errcode.New(404, "记录不存在")
	}
	return upload, err
}
func (l *ReportLogic) Detail(in types.UploadDetailRequest, u types.User) (*types.UploadDetailResponse, error) {
	upload, err := l.findUpload(in.ID, u)
	if err != nil {
		return nil, err
	}
	page := pageNumber(in.Page)
	rows, err := l.svcCtx.UploadModel.Rows(l.ctx, upload.ID, page)
	if err != nil {
		return nil, err
	}
	var columns []string
	if err = json.Unmarshal(upload.Columns, &columns); err != nil {
		return nil, err
	}
	result := &types.UploadDetailResponse{Upload: types.Upload(upload), Rows: make([]importer.Row, 0, len(rows)), Page: page, PageSize: 100}
	for _, row := range rows {
		out := importer.Row{SourceRow: row.SourceRow, Date: row.Date, Count: row.Count, Price: row.Price, Amount: row.Amount, Extra: row.Extra, Source: row.Source, Values: make([]string, len(columns))}
		for i, col := range columns {
			switch strings.TrimSpace(col) {
			case "日期":
				out.Values[i] = row.Date
			case "结算数":
				out.Values[i] = row.Count
			case "结算单价":
				out.Values[i] = row.Price
			case "结算金额":
				out.Values[i] = row.Amount
			default:
				out.Values[i] = row.Extra[col]
			}
		}
		result.Rows = append(result.Rows, out)
	}
	return result, nil
}
