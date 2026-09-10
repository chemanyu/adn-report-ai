package report

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/chemanyu/adn-report-ai/internal/config"
	"github.com/chemanyu/adn-report-ai/internal/errcode"
	"github.com/chemanyu/adn-report-ai/internal/model"
	"github.com/chemanyu/adn-report-ai/internal/security"
	"github.com/chemanyu/adn-report-ai/internal/svc"
	"github.com/chemanyu/adn-report-ai/internal/types"
	"github.com/jackc/pgx/v5/pgconn"
)

type accountStub struct{ model.AccountModel }

func (accountStub) Exists(context.Context, int64) (bool, error) { return true, nil }

type uploadStub struct {
	model.UploadModel
	insert func(model.NewUpload) (int64, error)
}

func (m uploadStub) InsertWithRows(_ context.Context, in model.NewUpload) (int64, error) {
	return m.insert(in)
}

// Exercise the CSV/database boundary without HTTP or a live database. Model errors
// must remove the newly archived file; successful inserts must retain it.
func TestUploadArchiveCompensation(t *testing.T) {
	dbError := errors.New("database unavailable")
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{
		{name: "committed"},
		{name: "duplicate", err: &pgconn.PgError{Code: "23505"}, status: 409},
		{name: "database failure", err: dbError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			if err := os.Mkdir(filepath.Join(directory, "csv"), 0700); err != nil {
				t.Fatal(err)
			}
			svcCtx := &svc.ServiceContext{Config: config.Config{DataDir: directory}, AccountModel: accountStub{}}
			l := NewReportLogic(context.Background(), svcCtx)
			data, err := l.Template()
			if err != nil {
				t.Fatal(err)
			}
			called := false
			svcCtx.UploadModel = uploadStub{insert: func(in model.NewUpload) (int64, error) {
				called = true
				if in.Upload.UserID != 7 || in.Upload.Uploader != "实际上传者" || in.Upload.Operator != "业务负责人" {
					t.Fatal("upload identity and operator were mixed")
				}
				if in.Hash != security.Hash(string(data)) || len(in.Rows) != 1 || in.Rows[0].Source[5] != "001234" {
					t.Fatal("model input lost original data")
				}
				if _, err := os.Stat(filepath.Join(directory, "csv", in.Upload.CSVPath)); err != nil {
					t.Fatal("CSV must exist before database commit")
				}
				return 42, tc.err
			}}
			result, err := l.Upload(types.FileRequest{Data: data, Filename: "report.xlsx", Sheet: "Sheet1", Operator: "业务负责人", AccountID: "1"}, types.User{ID: 7, Name: "实际上传者"})
			if !called {
				t.Fatal("model insert was not called")
			}
			if tc.err == nil {
				if err != nil || result.ID != 42 {
					t.Fatalf("result=%v error=%v", result, err)
				}
			} else if tc.status != 0 {
				var known *errcode.Error
				if !errors.As(err, &known) || known.Code != tc.status {
					t.Fatalf("unexpected error: %v", err)
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
				want = 1
			}
			if len(files) != want {
				t.Fatalf("CSV count=%d want=%d", len(files), want)
			}
		})
	}
}
