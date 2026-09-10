package importer

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func workbook(t *testing.T, rows [][]any) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	for i, row := range rows {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		if e := f.SetSheetRow("Sheet1", cell, &row); e != nil {
			t.Fatal(e)
		}
	}
	b, e := f.WriteToBuffer()
	if e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}
func TestImportPreservesExtraAndExactMoney(t *testing.T) {
	data := workbook(t, [][]any{{"日期", "结算数", "结算单价", "结算金额", "广告位ID", "备注"}, {46257, "55", "0.3", "16.5", "00123", "含,逗号\n换行"}, {"2026/8/24", "1", "0.1", "0.1", "9007199254740993", ""}})
	p, e := Read(data, "")
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Errors) != 0 {
		t.Fatal(p.Errors)
	}
	if p.Total != "16.600000" || p.Rows[0].Date != "2026-08-23" || p.Rows[0].Extra["广告位ID"] != "00123" || p.Rows[1].Extra["广告位ID"] != "9007199254740993" {
		t.Fatalf("unexpected result: %+v", p)
	}
	raw, e := p.CSV()
	if e != nil {
		t.Fatal(e)
	}
	rows, e := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(raw), "\ufeff"))).ReadAll()
	if e != nil {
		t.Fatal(e)
	}
	if rows[1][5] != "含,逗号\n换行" || rows[1][0] != "2026-08-23" || rows[1][4] != "00123" {
		t.Fatal(rows)
	}
}
func TestValidation(t *testing.T) {
	cases := []struct {
		name string
		rows [][]any
		want string
	}{
		{"missing", [][]any{{"日期", "结算金额"}, {"2026-08-23", 1}}, "缺少必填列：结算数"},
		{"duplicate", [][]any{{"日期", "结算数", "结算单价", "结算金额", " 日期"}}, "重复列名"},
		{"invalid date", [][]any{{"日期", "结算数", "结算单价", "结算金额"}, {"2026-02-30", 1, 1, 1}}, "第 2 行「日期」"},
		{"missing number", [][]any{{"日期", "结算数", "结算单价", "结算金额"}, {"2026-08-23", "", 1, 1}}, "第 2 行「结算数」"},
		{"precision", [][]any{{"日期", "结算数", "结算单价", "结算金额"}, {"2026-08-23", 1, "0.1234567", 1}}, "最多 6 位小数"},
		{"extra data", [][]any{{"日期", "结算数", "结算单价", "结算金额"}, {"2026-08-23", 1, 1, 1, "orphan"}}, "无列名"},
		{"empty", [][]any{{"日期", "结算数", "结算单价", "结算金额"}}, "没有数据行"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, e := Read(workbook(t, tc.rows), "")
			if e != nil {
				t.Fatal(e)
			}
			if !strings.Contains(strings.Join(p.Errors, "\n"), tc.want) {
				t.Fatal(p.Errors)
			}
		})
	}
}
func TestDecimalBoundaries(t *testing.T) {
	for _, value := range []string{"NaN", "Inf", "1e10", "1,000", "1000000000000000000", ".5", ""} {
		if _, e := decimal(value); e == nil {
			t.Errorf("accepted %q", value)
		}
	}
	for _, value := range []string{"999999999999999999.999999", "-0.100000", "+1.25", "0001"} {
		if _, e := decimal(value); e != nil {
			t.Errorf("rejected %q: %v", value, e)
		}
	}
}
func TestWorkbook1904AndSheetSelection(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	v := true
	f.SetWorkbookProps(&excelize.WorkbookPropsOptions{Date1904: &v})
	f.NewSheet("结算明细")
	headers := []any{"日期", "结算数", "结算单价", "结算金额"}
	row := []any{1, 1, 1, 1}
	f.SetSheetRow("结算明细", "A1", &headers)
	f.SetSheetRow("结算明细", "A2", &row)
	b, _ := f.WriteToBuffer()
	p, e := Read(b.Bytes(), "结算明细")
	if e != nil || len(p.Errors) > 0 || p.Rows[0].Date != "1904-01-02" {
		t.Fatalf("%+v %v", p, e)
	}
	if _, e = Read(b.Bytes(), "不存在"); e == nil {
		t.Fatal("missing sheet accepted")
	}
}
func TestProvidedExamplesRequireStandardColumns(t *testing.T) {
	paths, e := filepath.Glob("../../*.xlsx")
	if e != nil {
		t.Fatal(e)
	}
	if len(paths) == 0 {
		t.Skip("example workbooks not available")
	}
	for _, p := range paths {
		t.Run(filepath.Base(p), func(t *testing.T) {
			data, e := os.ReadFile(p)
			if e != nil {
				t.Fatal(e)
			}
			parsed, e := Read(data, "")
			if e != nil {
				t.Fatal(e)
			}
			if len(parsed.Errors) == 0 {
				t.Fatal("expected incomplete standard columns")
			}
			t.Log(strings.Join(parsed.Errors, "；"))
		})
	}
}
