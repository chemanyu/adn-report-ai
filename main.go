package main

import (
	"context"
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/chemanyu/adn-report-ai/internal/config"
	"github.com/chemanyu/adn-report-ai/internal/metadata"
	"github.com/chemanyu/adn-report-ai/internal/model"
	"github.com/chemanyu/adn-report-ai/internal/server"
	"github.com/chemanyu/adn-report-ai/internal/store"
	"github.com/chemanyu/adn-report-ai/internal/svc"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/proc"
)

//go:embed web/dist
var assets embed.FS

func main() {
	path := flag.String("f", "etc/config.yaml", "YAML config file")
	check := flag.Bool("check", false, "check configuration and PostgreSQL connectivity without starting the service")
	syncIDs := flag.Bool("sync-ids", false, "migrate schema and backfill missing account IDs and project mappings, without starting HTTP")
	flag.Parse()
	var c config.Config
	conf.MustLoad(*path, &c)
	if e := c.Validate(); e != nil {
		log.Fatal(e)
	}
	if *syncIDs && *check {
		log.Fatal("-check 与 -sync-ids 不能同时使用")
	}
	if *syncIDs && c.Ding.OpenID == "" {
		log.Fatal("请配置 Ding.OpenID")
	}
	if *check {
		db, err := store.Connect(c)
		if err != nil {
			log.Fatal(err)
		}
		db.Close()
		log.Print("配置及 PostgreSQL 连接检查通过")
		return
	}
	if e := os.MkdirAll(filepath.Join(c.DataDir, "csv"), 0700); e != nil {
		log.Fatal(e)
	}
	db, e := store.Open(c)
	if e != nil {
		log.Fatal(e)
	}
	defer db.Close()
	if *syncIDs {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		n, err := model.BackfillIDs(ctx, db, metadata.New(c.Metadata.Endpoint, c.Ding.OpenID))
		if err != nil {
			log.Fatal(store.RedactError(c, err))
		}
		log.Printf("账户 ID 回填完成：%d 条明细", n)
		return
	}
	web, e := fs.Sub(assets, "web/dist")
	if e != nil {
		log.Fatal(e)
	}
	svcCtx := svc.NewServiceContext(c, db)
	srv := server.New(svcCtx, web)
	defer srv.Stop()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	proc.SetTimeToForceQuit(15 * time.Second)
	proc.AddShutdownListener(cancel)
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanup, cancel := context.WithTimeout(ctx, 10*time.Second)
				if e := svcCtx.AuthModel.DeleteExpired(cleanup); e != nil {
					log.Printf("auth cleanup: %v", e)
				}
				cancel()
			}
		}
	}()
	log.Printf("启动 ADN HTTP 服务，监听 %s:%d", c.Host, c.Port)
	srv.StartWithOpts(func(h *http.Server) {
		h.ReadHeaderTimeout = 10 * time.Second
		h.ReadTimeout = 120 * time.Second
		h.WriteTimeout = 180 * time.Second
		h.IdleTimeout = 60 * time.Second
		h.MaxHeaderBytes = 1 << 20
	})
}
