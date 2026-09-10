package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/chemanyu/adn-report-ai/internal/errcode"
	"github.com/chemanyu/adn-report-ai/internal/model"
	"github.com/chemanyu/adn-report-ai/internal/security"
	"github.com/chemanyu/adn-report-ai/internal/types"
)

func (l *ReportLogic) Upload(in types.FileRequest, u types.User) (*types.UploadResponse, error) {
	name, p, err := parseFile(in)
	if err != nil {
		return nil, err
	}
	if len(p.Errors) > 0 {
		return nil, &errcode.Error{Code: 422, Message: "文件校验失败", Details: p.Errors}
	}
	operator := strings.TrimSpace(in.Operator)
	if operator == "" || len([]rune(operator)) > 191 {
		return nil, errcode.New(400, "请填写运营人员，最多 191 字")
	}
	account, err := strconv.ParseInt(in.AccountID, 10, 64)
	if err != nil || account < 1 {
		return nil, errcode.New(400, "请选择 ADN 账户")
	}
	exists, err := l.svcCtx.AccountModel.Exists(l.ctx, account)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errcode.New(400, "ADN 账户不存在")
	}
	csvData, err := p.CSV()
	if err != nil {
		return nil, err
	}
	csvName := security.RandomToken() + ".csv"
	csvPath := filepath.Join(l.svcCtx.Config.DataDir, "csv", csvName)
	file, err := os.OpenFile(csvPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			os.Remove(csvPath)
		}
	}()
	_, err = file.Write(csvData)
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, err
	}
	columns, err := json.Marshal(p.Columns)
	if err != nil {
		return nil, err
	}
	rows := make([]model.SettlementRow, 0, len(p.Rows))
	for _, row := range p.Rows {
		rows = append(rows, model.SettlementRow{SourceRow: row.SourceRow, Date: row.Date, Count: row.Count, Price: row.Price, Amount: row.Amount, Extra: row.Extra, Source: row.Source})
	}
	id, err := l.svcCtx.UploadModel.InsertWithRows(l.ctx, model.NewUpload{
		Upload: model.Upload{UserID: u.ID, Uploader: u.Name, Operator: operator, AccountID: account, Filename: name, Sheet: p.Sheet, Columns: columns, RowCount: len(p.Rows), Total: p.Total, DateFrom: p.DateFrom, DateTo: p.DateTo, CSVPath: csvName},
		Hash:   security.Hash(string(in.Data)), Rows: rows,
	})
	if model.IsDuplicate(err) {
		return nil, errcode.New(409, "你已向该 ADN 账户上传过相同文件的此工作表，请查看已有记录")
	}
	if err != nil {
		return nil, err
	}
	committed = true
	return &types.UploadResponse{ID: id, RowCount: len(p.Rows), Total: p.Total}, nil
}
