package report

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chemanyu/adn-report-ai/internal/types"
)

type Download struct {
	File    *os.File
	Name    string
	ModTime time.Time
}

// OpenDownload checks ownership before opening the CSV. The caller closes File.
func (l *ReportLogic) OpenDownload(id string, u types.User) (*Download, error) {
	upload, err := l.findUpload(id, u)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(filepath.Join(l.svcCtx.Config.DataDir, "csv", filepath.Base(upload.CSVPath)))
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	name := strings.TrimSuffix(upload.Filename, filepath.Ext(upload.Filename)) + "-" + upload.Sheet + ".csv"
	return &Download{File: file, Name: name, ModTime: info.ModTime()}, nil
}
