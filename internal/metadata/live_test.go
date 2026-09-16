package metadata

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/chemanyu/adn-report-ai/internal/config"
	"github.com/chemanyu/adn-report-ai/internal/model"
	"github.com/zeromicro/go-zero/core/conf"
)

func TestLiveReadOnlyLookup(t *testing.T) {
	if os.Getenv("ADN_TEST_METADATA") != "1" {
		t.Skip("opt-in read-only metadata API verification")
	}
	var c config.Config
	if err := conf.Load("../../etc/config.yaml", &c); err != nil {
		t.Fatal("无法加载接口测试配置")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rows := []model.SettlementRow{{Agency: "微博汽水", Advertiser: "微博-汽水-1户", SourceRow: 2}}
	projects, err := New(c.Metadata.Endpoint, c.Ding.OpenID).Resolve(ctx, rows)
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].AgencyID <= 0 || rows[0].AdvertiserID <= 0 || len(projects) == 0 {
		t.Fatal("账户或项目结果为空")
	}
}
