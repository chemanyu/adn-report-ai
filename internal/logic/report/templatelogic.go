package report

import (
	"github.com/xuri/excelize/v2"
)

func (l *ReportLogic) Template() ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	headers := []any{"日期", "结算数", "结算单价", "结算金额", "渠道名称", "广告位ID"}
	sample := []any{"2026-08-23", 100, "0.30", "30.00", "示例渠道", "001234"}
	if err := f.SetSheetRow("Sheet1", "A1", &headers); err != nil {
		return nil, err
	}
	if err := f.SetSheetRow("Sheet1", "A2", &sample); err != nil {
		return nil, err
	}
	b, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
