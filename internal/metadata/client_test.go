package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chemanyu/adn-report-ai/internal/model"
)

func TestResolve(t *testing.T) {
	for _, tc := range []struct {
		name, accounts  string
		status          int
		truncated, fail bool
	}{
		{name: "unique", accounts: `[{"agency_id":"1073772041","advertiser_id":1073772042}]`},
		{name: "missing", accounts: `[]`},
		{name: "missing advertiser", accounts: `[]`},
		{name: "missing project", accounts: `[{"agency_id":1073772041,"advertiser_id":1073772042}]`},
		{name: "ambiguous project", accounts: `[{"agency_id":1073772041,"advertiser_id":1073772042}]`, fail: true},
		{name: "ambiguous", accounts: `[{"agency_id":1,"advertiser_id":2},{"agency_id":3,"advertiser_id":4}]`, fail: true},
		{name: "truncated", accounts: `[]`, truncated: true, fail: true},
		{name: "forbidden", status: 403, fail: true},
		{name: "invalid ID", accounts: `[{"agency_id":0,"advertiser_id":2}]`, fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Header.Get("Authorization") != "Bearer openid.test-secret" || r.Method != "POST" {
					t.Error("invalid request")
				}
				var q map[string]string
				if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
					t.Error(err)
				}
				if tc.status != 0 {
					w.WriteHeader(tc.status)
					fmt.Fprint(w, "test-secret")
					return
				}
				if q["database_id"] == "adt" {
					if !strings.Contains(q["sql"], "JOIN") {
						if tc.name == "missing advertiser" {
							fmt.Fprint(w, `{"rows":[{"agency_id":1073772041}]}`)
						} else {
							fmt.Fprint(w, `{"rows":[]}`)
						}
						return
					}
					if !strings.Contains(q["sql"], "p.account_name='微博-CPA'") || !strings.Contains(q["sql"], "c.account_type=2") {
						t.Error(q["sql"])
					}
					fmt.Fprintf(w, `{"rows":%s,"truncated":%t}`, tc.accounts, tc.truncated)
				} else {
					if !strings.Contains(q["sql"], "HEX(p.name)=HEX('项目')") {
						t.Error(q["sql"])
					}
					if tc.name == "ambiguous project" {
						fmt.Fprint(w, `{"rows":[{"project_id":1,"project_name":"项目","advertiser_id":1073772042},{"project_id":2,"project_name":"项目","advertiser_id":1073772042}]}`)
						return
					}
					if tc.name == "missing project" {
						fmt.Fprint(w, `{"rows":[]}`)
						return
					}
					fmt.Fprint(w, `{"rows":[{"project_id":"9007199254740993","project_name":"项目","advertiser_id":1073772042}],"truncated":false}`)
				}
			}))
			defer srv.Close()
			rows := []model.SettlementRow{{Agency: "微博CPA", Advertiser: "知乎", SourceRow: 2, TaskName: "项目"}, {Agency: "微博CPA", Advertiser: "知乎", SourceRow: 3, TaskName: "项目"}}
			p, err := New(srv.URL, "test-secret").Resolve(context.Background(), rows)
			if tc.fail {
				if err == nil || strings.Contains(err.Error(), "test-secret") {
					t.Fatalf("unsafe or missing error: %v", err)
				}
				return
			}
			if strings.HasPrefix(tc.name, "missing") {
				wantAgency, wantAccount := int64(0), int64(0)
				if tc.name == "missing advertiser" || tc.name == "missing project" {
					wantAgency = 1073772041
				}
				if tc.name == "missing project" {
					wantAccount = 1073772042
				}
				if err != nil || len(p) != 1 || p[0].ID != 0 || p[0].AdvertiserID != wantAccount || rows[1].AgencyID != wantAgency || rows[1].AdvertiserID != wantAccount || calls != 2 {
					t.Fatalf("missing result rows=%+v projects=%+v calls=%d err=%v", rows, p, calls, err)
				}
				return
			}
			if err != nil || len(p) != 1 || p[0].ID != 9007199254740993 || rows[1].AgencyID != 1073772041 || rows[0].Agency != "微博CPA" || calls != 2 {
				t.Fatalf("rows=%+v projects=%+v calls=%d err=%v", rows, p, calls, err)
			}
		})
	}
}
func TestQuote(t *testing.T) {
	if got := quote("a'b\\c"); got != "'a''b\\\\c'" {
		t.Fatal(got)
	}
}
func TestRedirectDoesNotForwardCredential(t *testing.T) {
	reached := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) }))
	defer source.Close()
	_, err := New(source.URL, "secret").Resolve(context.Background(), []model.SettlementRow{{Agency: "a", Advertiser: "b"}})
	if err == nil || reached {
		t.Fatal("redirect followed")
	}
}
