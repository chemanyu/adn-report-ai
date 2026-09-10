// deployconfig renders a private deployment config without printing credentials.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/chemanyu/adn-report-ai/internal/config"
	"github.com/zeromicro/go-zero/core/conf"
)

func main() {
	input := flag.String("f", "etc/config.yaml", "source private config")
	output := flag.String("out", ".cache/deploy/config.yaml", "rendered private config")
	stage := flag.Bool("stage", false, "allow a password placeholder when staging only")
	flag.Parse()
	var c config.Config
	if err := conf.Load(*input, &c); err != nil {
		log.Fatal("无法读取部署配置，请检查源文件结构")
	}
	c.Host = "127.0.0.1"
	c.Port = 18080
	c.BaseURL = "https://172.16.3.34:18080"
	c.DataDir = "/data/adn-report-ai/data"
	if err := c.Validate(); err != nil {
		log.Fatal(err)
	}
	u, err := url.Parse(c.PostgreSQL.DSN)
	if err != nil {
		log.Fatal("PostgreSQL.DSN 格式无效")
	}
	password := ""
	if u.User != nil {
		password, _ = u.User.Password()
	}
	if !*stage && (password == "" || strings.Contains(password, "PG_PASSWORD_REQUIRED")) {
		log.Fatal("请先在 etc/config.yaml 中填写真实 PG 密码；如仅上传文件，请使用 --stage")
	}
	data, err := json.MarshalIndent(map[string]any{"Name": c.Name, "Host": c.Host, "Port": c.Port, "BaseURL": c.BaseURL, "DataDir": c.DataDir, "PostgreSQL": c.PostgreSQL, "Auth": c.Auth}, "", "  ")
	if err != nil {
		log.Fatal("生成配置失败")
	}
	file, err := os.OpenFile(*output, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal("无法写入部署配置")
	}
	if err = file.Chmod(0600); err != nil {
		file.Close()
		log.Fatal("无法设置私有配置权限")
	}
	_, err = file.Write(data)
	closeErr := file.Close()
	if err != nil || closeErr != nil {
		log.Fatal("写入部署配置失败")
	}
	fmt.Println("部署配置已生成（权限 0600，未输出凭据）。")
}
