package store

import (
	"errors"
	"strings"
	"testing"

	"github.com/chemanyu/adn-report-ai/internal/config"
)

func TestRedactDatabaseCredentials(t *testing.T) {
	var c config.Config
	c.PostgreSQL.DSN = "postgres://user:private%21value@localhost/test"
	err := RedactError(c, errors.New("failed "+c.PostgreSQL.DSN+" password private!value"))
	if strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), c.PostgreSQL.DSN) {
		t.Fatal("credentials leaked")
	}
}
