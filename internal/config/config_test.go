package config

import (
	"strings"
	"testing"
)

func TestPostgreSQLConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test:secret@127.0.0.1:5432/test?sslmode=disable")
	t.Setenv("ADN_ADMIN_PASSWORD", "")
	base := Config{BaseURL: "http://127.0.0.1:18080", DataDir: "data"}
	base.Port = 18080
	base.PostgreSQL.Schema = "adn_report"
	if e := base.Validate(); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(base.PostgreSQL.DSN, "127.0.0.1:5432") {
		t.Fatal("DATABASE_URL override not applied")
	}
	for _, name := range []string{"public", "pg_catalog", "information_schema", "BadSchema", "a;drop schema public", strings.Repeat("a", 64)} {
		c := base
		c.PostgreSQL.Schema = name
		if e := c.Validate(); e == nil {
			t.Fatalf("accepted invalid schema %q", name)
		}
	}
	for _, dsn := range []string{"", "mysql://test:secret@localhost/test", "postgres://test:secret@localhost", "postgres://test:secret@%zz/test"} {
		t.Setenv("DATABASE_URL", dsn)
		c := base
		if e := c.Validate(); e == nil {
			t.Fatal("accepted invalid DSN")
		} else if strings.Contains(e.Error(), "secret") {
			t.Fatal("password in error")
		}
	}
}

func TestBaseURLPrefix(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test:secret@127.0.0.1:5432/test")
	t.Setenv("ADN_ADMIN_PASSWORD", "")
	for _, baseURL := range []string{"http://172.16.3.34", "http://172.16.3.34/adn-report/", "https://example.com:8443/apps/adn"} {
		c := Config{BaseURL: baseURL, DataDir: "data"}
		c.Port = 18080
		c.PostgreSQL.Schema = "adn_report"
		if err := c.Validate(); err != nil {
			t.Fatal(err)
		}
		if c.Origin()+c.BasePath() != strings.TrimRight(baseURL, "/") {
			t.Fatal("origin/path split is incorrect")
		}
	}
	for _, baseURL := range []string{"http://example.com/a/../b", "http://example.com/a//b", "http://example.com/a?b=c", "http://user:pass@example.com/a", "http://example.com/a%2Fb"} {
		c := Config{BaseURL: baseURL, DataDir: "data"}
		c.Port = 18080
		c.PostgreSQL.Schema = "adn_report"
		if err := c.Validate(); err == nil {
			t.Fatalf("accepted unsafe BaseURL %q", baseURL)
		}
	}
}
