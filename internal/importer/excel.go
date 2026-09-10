package importer

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

var Required = []string{"代理商", "广告主", "日期", "任务名称", "结算数", "结算单价", "结算金额"}

const MaxRows = 20000

type Row struct {
	Agency     string            `json:"agency"`
	Advertiser string            `json:"advertiser"`
	TaskName   string            `json:"task_name"`
	SourceRow  int               `json:"source_row"`
	Date       string            `json:"date"`
	Count      string            `json:"count"`
	Price      string            `json:"price"`
	Amount     string            `json:"amount"`
	Extra      map[string]string `json:"extra"`
	Values     []string          `json:"values"`
}
type Result struct {
	Sheets   []string `json:"sheets"`
	Sheet    string   `json:"sheet"`
	Columns  []string `json:"columns"`
	Rows     []Row    `json:"rows"`
	Errors   []string `json:"errors"`
	Total    string   `json:"total_amount"`
	DateFrom string   `json:"date_from"`
	DateTo   string   `json:"date_to"`
}

// Read validates the selected worksheet. No aliases, inferred prices or implicit aggregation.
func Read(data []byte, sheet string) (*Result, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{RawCellValue: true, UnzipSizeLimit: 128 << 20, UnzipXMLSizeLimit: 16 << 20})
	if err != nil {
		return nil, fmt.Errorf("无法读取 Excel，请上传有效的 .xlsx 文件")
	}
	defer f.Close()
	out := &Result{Sheets: f.GetSheetList(), Rows: []Row{}, Errors: []string{}, Columns: []string{}, Total: "0.000000"}
	if len(out.Sheets) == 0 {
		return nil, fmt.Errorf("文件没有工作表")
	}
	if sheet == "" {
		sheet = out.Sheets[0]
	}
	out.Sheet = sheet
	exists := false
	for _, s := range out.Sheets {
		if s == sheet {
			exists = true
		}
	}
	if !exists {
		return nil, fmt.Errorf("工作表不存在")
	}
	props, err := f.GetWorkbookProps()
	if err != nil {
		return nil, err
	}
	use1904 := props.Date1904 != nil && *props.Date1904
	it, err := f.Rows(sheet)
	if err != nil {
		return nil, err
	}
	defer it.Close()
	if !it.Next() {
		return nil, fmt.Errorf("工作表为空")
	}
	headers, err := it.Columns(excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, err
	}
	if len(headers) == 0 || len(headers) > 100 {
		return nil, fmt.Errorf("表头为空或超过 100 列")
	}
	out.Columns = headers
	positions := map[string]int{}
	for i, h := range headers {
		key := strings.TrimSpace(h)
		if key == "" {
			out.Errors = append(out.Errors, fmt.Sprintf("第 %d 列缺少列名", i+1))
			continue
		}
		if _, ok := positions[key]; ok {
			out.Errors = append(out.Errors, "重复列名："+key)
		}
		positions[key] = i
	}
	for _, key := range Required {
		if _, ok := positions[key]; !ok {
			out.Errors = append(out.Errors, "缺少必填列："+key)
		}
	}
	if len(out.Errors) > 0 {
		return out, nil
	}
	total := new(big.Rat)
	seen := map[[4]string]int{}
	line := 1
	for it.Next() {
		line++
		if line > MaxRows+1 {
			return nil, fmt.Errorf("单次最多支持 %d 行，请拆分文件", MaxRows)
		}
		cells, e := it.Columns(excelize.Options{RawCellValue: true})
		if e != nil {
			return nil, e
		}
		empty := true
		for _, v := range cells {
			if strings.TrimSpace(v) != "" {
				empty = false
				break
			}
		}
		if empty {
			continue
		}
		if len(cells) > len(headers) {
			out.Errors = append(out.Errors, fmt.Sprintf("第 %d 行存在无列名的数据", line))
			break
		}
		source := make([]string, len(headers))
		copy(source, cells)
		row := Row{SourceRow: line, Extra: map[string]string{}, Values: append([]string(nil), source...)}
		for i, v := range source {
			if len(v) > 32767 {
				return nil, fmt.Errorf("第 %d 行单元格过长", line)
			}
			key := strings.TrimSpace(headers[i])
			fixed := false
			for _, k := range Required {
				if key == k {
					fixed = true
				}
			}
			if !fixed {
				row.Extra[headers[i]] = v
			}
		}
		for _, field := range []struct {
			key    string
			target *string
		}{
			{"代理商", &row.Agency}, {"广告主", &row.Advertiser}, {"任务名称", &row.TaskName},
		} {
			value := strings.TrimSpace(source[positions[field.key]])
			if value == "" || len([]rune(value)) > 255 {
				out.Errors = append(out.Errors, fmt.Sprintf("第 %d 行「%s」：不能为空，最多 255 字", line, field.key))
			}
			*field.target = value
			row.Values[positions[field.key]] = value
		}
		row.Date, e = parseDate(source[positions["日期"]], use1904)
		if e != nil {
			out.Errors = append(out.Errors, fmt.Sprintf("第 %d 行「日期」：%s", line, e))
		}
		for _, p := range []struct {
			key    string
			target *string
		}{{"结算数", &row.Count}, {"结算单价", &row.Price}, {"结算金额", &row.Amount}} {
			v, e := decimal(source[positions[p.key]])
			if e != nil {
				out.Errors = append(out.Errors, fmt.Sprintf("第 %d 行「%s」：%s", line, p.key, e))
			} else {
				*p.target = v
				row.Values[positions[p.key]] = v
			}
		}
		row.Values[positions["日期"]] = row.Date
		if row.Agency != "" && row.Advertiser != "" && row.Date != "" && row.TaskName != "" {
			key := [4]string{row.Agency, row.Advertiser, row.Date, row.TaskName}
			if first, exists := seen[key]; exists {
				out.Errors = append(out.Errors, fmt.Sprintf("第 %d 行与第 %d 行的代理商、广告主、日期、任务名称重复，请合并或修正后上传", line, first))
			} else {
				seen[key] = line
			}
		}
		if row.Amount != "" {
			n, _ := new(big.Rat).SetString(row.Amount)
			total.Add(total, n)
		}
		if row.Date != "" {
			if out.DateFrom == "" || row.Date < out.DateFrom {
				out.DateFrom = row.Date
			}
			if row.Date > out.DateTo {
				out.DateTo = row.Date
			}
		}
		out.Rows = append(out.Rows, row)
		if len(out.Errors) >= 50 {
			out.Errors = append(out.Errors, "错误过多，仅显示前 50 项；请修正后重新校验")
			break
		}
	}
	if err := it.Error(); err != nil {
		return nil, err
	}
	if len(out.Rows) == 0 && len(out.Errors) == 0 {
		out.Errors = append(out.Errors, "工作表没有数据行")
	}
	out.Total = total.FloatString(6)
	return out, nil
}

var number = regexp.MustCompile(`^[+-]?(?:[0-9]+)(?:\.[0-9]{1,6})?$`)

func decimal(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if !number.MatchString(s) {
		return "", fmt.Errorf("必须是数字，最多 6 位小数，不支持千分位或公式错误")
	}
	integer := strings.TrimLeft(strings.Split(strings.TrimLeft(s, "+-"), ".")[0], "0")
	if len(integer) > 18 {
		return "", fmt.Errorf("整数部分最多 18 位")
	}
	n, ok := new(big.Rat).SetString(s)
	if !ok {
		return "", fmt.Errorf("数字格式错误")
	}
	return n.FloatString(6), nil
}
func parseDate(s string, use1904 bool) (string, error) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{"2006-01-02", "2006/1/2", "2006-1-2", "2006年1月2日", "2006-01-02 15:04:05", time.RFC3339} {
		if t, e := time.Parse(layout, s); e == nil && t.Year() >= 1900 && t.Year() <= 9999 {
			return t.Format("2006-01-02"), nil
		}
	}
	if n, e := strconv.ParseFloat(s, 64); e == nil && n >= 1 && n <= 2958465 {
		if t, e := excelize.ExcelDateToTime(n, use1904); e == nil && t.Year() <= 9999 {
			return t.Format("2006-01-02"), nil
		}
	}
	return "", fmt.Errorf("需要有效日期（如 2026-08-23）或 Excel 日期单元格")
}
func (r *Result) CSV() ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("\xef\xbb\xbf")
	w := csv.NewWriter(&b)
	if err := w.Write(r.Columns); err != nil {
		return nil, err
	}
	for _, row := range r.Rows {
		if err := w.Write(row.Values); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return b.Bytes(), w.Error()
}
