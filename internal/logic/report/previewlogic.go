package report

import (
	"path/filepath"
	"strings"

	"github.com/chemanyu/adn-report-ai/internal/errcode"
	"github.com/chemanyu/adn-report-ai/internal/importer"
	"github.com/chemanyu/adn-report-ai/internal/types"
)

func parseFile(in types.FileRequest) (string, *importer.Result, error) {
	if !strings.EqualFold(filepath.Ext(in.Filename), ".xlsx") || len(in.Data) > 20<<20 {
		return "", nil, errcode.New(400, "仅支持 20 MB 以内的 .xlsx 文件")
	}
	parsed, err := importer.Read(in.Data, in.Sheet)
	if err != nil {
		return "", nil, errcode.New(422, err.Error())
	}
	name := filepath.Base(strings.ReplaceAll(in.Filename, "\\", "/"))
	if len([]rune(name)) > 255 {
		return "", nil, errcode.New(400, "文件名过长")
	}
	return name, parsed, nil
}
func (l *ReportLogic) Preview(in types.FileRequest) (*types.PreviewResponse, error) {
	_, p, err := parseFile(in)
	if err != nil {
		return nil, err
	}
	count := len(p.Rows)
	if count > 10 {
		p.Rows = p.Rows[:10]
	}
	return &types.PreviewResponse{Result: p, RowCount: count, Valid: len(p.Errors) == 0}, nil
}
