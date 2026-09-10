package report

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/chemanyu/adn-report-ai/internal/importer"
	"github.com/chemanyu/adn-report-ai/internal/types"
)

type Download struct {
	File *bytes.Reader
	Name string
}

// Download the current effective rows, not the immutable import archive.
func (l *ReportLogic) OpenDownload(id string, u types.User) (*Download, error) {
	upload, rows, err := l.snapshot(id, u, 0)
	if err != nil {
		return nil, err
	}
	values, err := detailRows(upload, rows)
	if err != nil {
		return nil, err
	}
	var columns []string
	if err = json.Unmarshal(upload.Columns, &columns); err != nil {
		return nil, err
	}
	result := importer.Result{Columns: columns, Rows: values}
	data, err := result.CSV()
	if err != nil {
		return nil, err
	}
	name := strings.TrimSuffix(upload.Filename, filepath.Ext(upload.Filename)) + "-" + upload.Sheet + ".csv"
	return &Download{File: bytes.NewReader(data), Name: name}, nil
}
