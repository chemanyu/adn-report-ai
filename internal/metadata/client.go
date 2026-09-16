// Package metadata resolves settlement account names through the execute_sql API.
package metadata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chemanyu/adn-report-ai/internal/model"
)

type Client struct {
	Endpoint, OpenID string
	HTTP             *http.Client
}

func New(endpoint, openID string) *Client {
	if endpoint == "" {
		endpoint = "http://172.16.3.25:8081/api/v1/execute_sql"
	}
	return &Client{Endpoint: endpoint, OpenID: openID, HTTP: &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (c *Client) query(ctx context.Context, database, sql string, dest any) error {
	body, _ := json.Marshal(map[string]string{"database_id": database, "sql": sql})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("ID 查询地址无效")
	}
	req.Header.Set("Authorization", "Bearer openid."+c.OpenID)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("ID 查询服务连接失败")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("ID 查询服务返回 HTTP %d", resp.StatusCode)
	}
	var result struct {
		Rows      json.RawMessage `json:"rows"`
		Truncated bool            `json:"truncated"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&result); err != nil {
		return fmt.Errorf("ID 查询响应无效")
	}
	if result.Truncated {
		return fmt.Errorf("ID 查询结果被截断，请缩小查询范围")
	}
	if len(result.Rows) == 0 || string(result.Rows) == "null" {
		return fmt.Errorf("ID 查询响应缺少 rows")
	}
	if err := json.Unmarshal(result.Rows, dest); err != nil {
		return fmt.Errorf("ID 查询数据格式无效")
	}
	return nil
}

// ID accepts both JSON numbers and decimal strings without float precision loss.
type ID int64

func (i *ID) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	n, e := strconv.ParseInt(s, 10, 64)
	if e != nil || n <= 0 {
		return fmt.Errorf("无效 ID")
	}
	*i = ID(n)
	return nil
}
func quote(s string) string {
	return "'" + strings.ReplaceAll(strings.ReplaceAll(s, "\\", "\\\\"), "'", "''") + "'"
}
func alias(agency, advertiser string) (string, string) {
	switch {
	case agency == "支付宝" && advertiser == "短剧-短剧":
		return "支付宝-短剧", "短剧"
	case agency == "程投" && advertiser == "程投-微博":
		return agency, "微博-程投"
	case agency == "微博CPA" && advertiser == "知乎":
		return "微博-CPA", advertiser
	}
	return agency, advertiser
}
func (c *Client) Resolve(ctx context.Context, rows []model.SettlementRow) ([]model.Project, error) {
	if strings.TrimSpace(c.OpenID) == "" {
		return nil, fmt.Errorf("请配置 Ding.OpenID 后解析账户 ID")
	}
	type account struct {
		Agency     ID `json:"agency_id"`
		Advertiser ID `json:"advertiser_id"`
	}
	accounts := map[[2]string]account{}
	projects := []model.Project{}
	seen := map[struct {
		account int64
		task    string
	}]bool{}
	for i := range rows {
		r := &rows[i]
		a, b := alias(r.Agency, r.Advertiser)
		key := [2]string{a, b}
		acc, ok := accounts[key]
		if !ok {
			var matches []account
			sql := "SELECT c.account_id AS advertiser_id,p.account_id AS agency_id FROM account c JOIN account p ON p.account_id=c.pid WHERE c.account_type=2 AND p.account_type=1 AND c.account_name=" + quote(b) + " AND p.account_name=" + quote(a) + " LIMIT 2"
			if err := c.query(ctx, "adt", sql, &matches); err != nil {
				return nil, err
			}
			if len(matches) > 1 {
				return nil, fmt.Errorf("第 %d 行：代理商 %q / 账户 %q 匹配到 %d 个账户，请核对名称或补充别名映射", r.SourceRow, r.Agency, r.Advertiser, len(matches))
			}
			if len(matches) == 1 {
				acc = matches[0]
			} else {
				// An unknown advertiser must not discard an independently known agency.
				var agencies []struct {
					ID ID `json:"agency_id"`
				}
				if err := c.query(ctx, "adt", "SELECT p.account_id AS agency_id FROM account p WHERE p.account_type=1 AND p.account_name="+quote(a)+" LIMIT 2", &agencies); err != nil {
					return nil, err
				}
				if len(agencies) > 1 {
					return nil, fmt.Errorf("第 %d 行：代理商 %q 匹配到多个 ID", r.SourceRow, r.Agency)
				}
				if len(agencies) == 1 {
					acc.Agency = agencies[0].ID
				}
			}
			accounts[key] = acc
		}
		r.AgencyID = int64(acc.Agency)
		r.AdvertiserID = int64(acc.Advertiser)
		projectKey := struct {
			account int64
			task    string
		}{r.AdvertiserID, r.TaskName}
		if seen[projectKey] {
			continue
		}
		seen[projectKey] = true
		matched := model.Project{ID: 0, Name: r.TaskName, AdvertiserID: r.AdvertiserID}
		if r.AdvertiserID != 0 {
			var page []struct {
				ID      ID     `json:"project_id"`
				Name    string `json:"project_name"`
				Account ID     `json:"advertiser_id"`
			}
			sql := fmt.Sprintf("SELECT p.id AS project_id,p.name AS project_name,p.account_id AS advertiser_id FROM adn_project p WHERE p.account_id=%d AND HEX(p.name)=HEX(%s) ORDER BY p.id LIMIT 2", r.AdvertiserID, quote(r.TaskName))
			if err := c.query(ctx, "adn", sql, &page); err != nil {
				return nil, err
			}
			if len(page) > 1 {
				return nil, fmt.Errorf("第 %d 行：任务名称 %q 在账户下匹配到多个项目", r.SourceRow, r.TaskName)
			}
			if len(page) == 1 {
				p := page[0]
				if int64(p.Account) != r.AdvertiserID || p.Name != r.TaskName {
					return nil, fmt.Errorf("项目查询返回了不一致的账户或任务名称")
				}
				matched.ID = int64(p.ID)
			}
		}
		projects = append(projects, matched)
	}
	return projects, nil
}
