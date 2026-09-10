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

func (l *ReportLogic) snapshot(idText string, u types.User, page int) (model.Upload, []model.SettlementRow, error) {
	id, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || id < 1 {
		return model.Upload{}, nil, errcode.New(404, "记录不存在")
	}
	upload, rows, err := l.svcCtx.UploadModel.Snapshot(l.ctx, id, ownerScope(u), page)
	if errors.Is(err, model.ErrNotFound) {
		return upload, nil, errcode.New(404, "记录不存在")
	}
	return upload, rows, err
}
func (l *ReportLogic) Detail(in types.UploadDetailRequest, u types.User) (*types.UploadDetailResponse, error) {
	page := pageNumber(in.Page)
	upload, rows, err := l.snapshot(in.ID, u, page)
	if err != nil {
		return nil, err
	}
	result, err := detailRows(upload, rows)
	if err != nil {
		return nil, err
	}
	return &types.UploadDetailResponse{Upload: types.Upload(upload), Rows: result, Page: page, PageSize: 100}, nil
}
func detailRows(upload model.Upload, rows []model.SettlementRow) ([]importer.Row, error) {
	var columns []string
	if err := json.Unmarshal(upload.Columns, &columns); err != nil {
		return nil, err
	}
	result := make([]importer.Row, 0, len(rows))
	for _, row := range rows {
		out := importer.Row{SourceRow: row.SourceRow, Agency: row.Agency, Advertiser: row.Advertiser, TaskName: row.TaskName, Date: row.Date, Count: row.Count, Price: row.Price, Amount: row.Amount, Extra: row.Extra, Values: make([]string, len(columns))}
		for i, col := range columns {
			switch strings.TrimSpace(col) {
			case "代理商":
				out.Values[i] = row.Agency
			case "广告主":
				out.Values[i] = row.Advertiser
			case "任务名称":
				out.Values[i] = row.TaskName
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
		result = append(result, out)
	}
	return result, nil
}
