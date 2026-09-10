package config

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	BaseURL    string
	DataDir    string
	PostgreSQL struct {
		DSN    string `json:",optional"`
		Schema string `json:",default=adn_report"`
	}
	Auth struct {
		ClientID      string   `json:",optional"`
		ClientSecret  string   `json:",optional"`
		AdminUnionIDs []string `json:",optional"`
		AdminUsername string   `json:",default=admin"`
		AdminPassword string   `json:",optional"`
	}
}

func (c *Config) Validate() error {
	for key, target := range map[string]*string{"DINGTALK_CLIENT_ID": &c.Auth.ClientID, "DINGTALK_CLIENT_SECRET": &c.Auth.ClientSecret, "ADN_ADMIN_PASSWORD": &c.Auth.AdminPassword, "DATABASE_URL": &c.PostgreSQL.DSN} {
		if v, ok := os.LookupEnv(key); ok {
			*target = v
		}
	}
	if v := os.Getenv("ADN_ADMIN_UNION_IDS"); v != "" {
		c.Auth.AdminUnionIDs = strings.Split(v, ",")
	}
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.RawQuery != "" || u.Fragment != "" || u.User != nil || u.RawPath != "" || !regexp.MustCompile(`^(?:/[A-Za-z0-9_-]+)*$`).MatchString(u.Path) {
		return fmt.Errorf("BaseURL 必须是完整站点地址，可包含独立路径前缀，不含查询参数或片段")
	}
	pg, pgErr := url.Parse(c.PostgreSQL.DSN)
	if pgErr != nil || pg.Hostname() == "" || (pg.Scheme != "postgres" && pg.Scheme != "postgresql") || pg.Path == "" || pg.Path == "/" || pg.Fragment != "" {
		return fmt.Errorf("PostgreSQL.DSN 或 DATABASE_URL 必须为有效的 PostgreSQL 连接地址")
	}
	if !regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`).MatchString(c.PostgreSQL.Schema) || strings.HasPrefix(c.PostgreSQL.Schema, "pg_") || c.PostgreSQL.Schema == "information_schema" || c.PostgreSQL.Schema == "public" {
		return fmt.Errorf("PostgreSQL.Schema 必须为独立业务 schema 名称（小写字母、数字、下划线，最多 63 字符）")
	}
	if c.Auth.AdminPassword != "" && len(c.Auth.AdminPassword) < 12 {
		return fmt.Errorf("管理员密码至少 12 位")
	}
	if c.Port < 1 || c.Port > 65535 || c.DataDir == "" {
		return fmt.Errorf("Port 或 DataDir 无效")
	}
	return nil
}

// Origin is the browser origin; URL path prefixes are not part of an Origin header.
func (c Config) Origin() string {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
func (c Config) BasePath() string {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return ""
	}
	return strings.TrimRight(u.Path, "/")
}
