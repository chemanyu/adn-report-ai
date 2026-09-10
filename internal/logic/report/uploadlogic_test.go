package report

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/chemanyu/adn-report-ai/internal/config"
	"github.com/chemanyu/adn-report-ai/internal/model"
	"github.com/chemanyu/adn-report-ai/internal/security"
	"github.com/chemanyu/adn-report-ai/internal/svc"
	"github.com/chemanyu/adn-report-ai/internal/types"
)

type uploadStub struct {
	model.UploadModel
	save func(model.NewUpload) (model.UploadSaveResult, error)
}

func (m uploadStub) SaveWithRows(_ context.Context, in model.NewUpload) (model.UploadSaveResult, error) {
	return m.save(in)
}

// Exercise the CSV/database boundary without HTTP or a live database. Model errors
// must remove the newly archived file; successful inserts must retain it.
func TestUploadArchiveCompensation(t *testing.T) {
	dbError := errors.New("database unavailable")
	for _, tc := range []struct {
		name     string
		err      error
		replaced bool
	}{
		{name: "committed"},
		{name: "replacement", replaced: true},
		{name: "failed replacement", replaced: true, err: dbError},
		{name: "database failure", err: dbError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			if err := os.Mkdir(filepath.Join(directory, "csv"), 0700); err != nil {
				t.Fatal(err)
			}
			svcCtx := &svc.ServiceContext{Config: config.Config{DataDir: directory}}
			l := NewReportLogic(context.Background(), svcCtx)
			data, err := l.Template()
			if err != nil {
				t.Fatal(err)
			}
			oldPaths := []string{}
			if tc.replaced {
				oldPaths = []string{"old.csv", "legacy.csv"}
				for _, name := range oldPaths {
					if err := os.WriteFile(filepath.Join(directory, "csv", name), []byte("old data"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			called := false
			svcCtx.UploadModel = uploadStub{save: func(in model.NewUpload) (model.UploadSaveResult, error) {
				called = true
				if in.Upload.UserID != 7 || in.Upload.Uploader != "实际上传者" || in.Upload.Operator != "业务负责人" {
					t.Fatal("upload identity and operator were mixed")
				}
				if in.Hash != security.Hash(string(data)) || len(in.Rows) != 1 || in.Rows[0].Agency != "示例代理商" || in.Rows[0].Advertiser != "示例广告主" || in.Rows[0].TaskName != "示例任务" {
					t.Fatal("model input lost original data")
				}
				if _, err := os.Stat(filepath.Join(directory, "csv", in.Upload.CSVPath)); err != nil {
					t.Fatal("CSV must exist before database commit")
				}
				for _, name := range oldPaths {
					if _, err := os.Stat(filepath.Join(directory, "csv", name)); err != nil {
						t.Fatal("old CSV removed before commit")
					}
				}
				return model.UploadSaveResult{ID: 42, Inserted: 1}, tc.err
			}}
			result, err := l.Upload(types.FileRequest{Data: data, Filename: "report.xlsx", Sheet: "Sheet1", Operator: "业务负责人"}, types.User{ID: 7, Name: "实际上传者"})
			if !called {
				t.Fatal("model insert was not called")
			}
			if tc.err == nil {
				if err != nil || result.ID != 42 || result.Inserted != 1 {
					t.Fatalf("result=%v error=%v", result, err)
				}
			} else if !errors.Is(err, tc.err) {
				t.Fatalf("unexpected error: %v", err)
			}
			files, err := filepath.Glob(filepath.Join(directory, "csv", "*"))
			if err != nil {
				t.Fatal(err)
			}
			want := 0
			if tc.err == nil {
				want = 1 + len(oldPaths)
			} else {
				want = len(oldPaths)
			}
			for _, name := range oldPaths {
				body, err := os.ReadFile(filepath.Join(directory, "csv", name))
				if err != nil || string(body) != "old data" {
					t.Fatal("failed replacement changed old CSV")
				}
			}
			if len(files) != want {
				t.Fatalf("CSV count=%d want=%d", len(files), want)
			}
		})
	}
}
