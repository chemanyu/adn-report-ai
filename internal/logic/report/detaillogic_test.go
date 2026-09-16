package report

import (
	"github.com/chemanyu/adn-report-ai/internal/model"
	"testing"
)

func TestDetailRowsIDsAndCSVValues(t *testing.T) {
	rows, err := detailRows(model.Upload{Columns: []byte(`["代理商","广告主","fix"]`)}, []model.SettlementRow{{Agency: "代理", Advertiser: "广告", AgencyID: 9007199254740993, AdvertiserID: 42, ProjectIDs: []string{"9007199254740995", "9007199254740996"}, Extra: map[string]string{"fix": "row-A"}}})
	if err != nil {
		t.Fatal(err)
	}
	row := rows[0]
	if row.AgencyID != "9007199254740993" || row.AdvertiserID != "42" || len(row.ProjectIDs) != 2 {
		t.Fatalf("IDs lost: %+v", row)
	}
	if row.Values[2] != "row-A" || row.Extra["fix"] != "row-A" {
		t.Fatal("CSV extra values lost")
	}
}
