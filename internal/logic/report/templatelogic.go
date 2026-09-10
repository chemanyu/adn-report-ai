package report

import (
	"github.com/xuri/excelize/v2"
)

func (l *ReportLogic) Template() ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	headers := []any{"代理商", "广告主", "日期", "任务名称", "结算数", "结算单价", "结算金额"}
	sample := []any{"示例代理商", "示例广告主", "2026-08-23", "示例任务", 100, "0.30", "30.00"}
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
