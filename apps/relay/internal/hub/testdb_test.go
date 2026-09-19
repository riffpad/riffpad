package hub

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testDatabaseURL decides where a test's store lives.
//
// Unset RIFFPAD_TEST_DATABASE_URL means SQLite in the test's temp dir, which
// is what every local run and the default CI job use. Setting it to a base
// Postgres DSN runs the exact same tests against Postgres — production's
// dialect — each in its own schema so tests stay isolated and can be dropped
// together afterwards (#333).
//
// The schema is derived from the test name, so a test that reopens the store
// (the "restart" cases) gets the same database it started with.
func testDatabaseURL(t *testing.T) string {
	t.Helper()
	base := os.Getenv("RIFFPAD_TEST_DATABASE_URL")
	if base == "" {
		return ""
	}
	schema := testSchema(t.Name())

	db, err := gorm.Open(postgres.Open(base), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("connect to test postgres: %v", err)
	}
	if err := db.Exec(`CREATE SCHEMA IF NOT EXISTS "` + schema + `"`).Error; err != nil {
		t.Fatalf("create schema %s: %v", schema, err)
	}
	t.Cleanup(func() {
		if err := db.Exec(`DROP SCHEMA IF EXISTS "` + schema + `" CASCADE`).Error; err != nil {
			t.Logf("drop schema %s: %v", schema, err)
		}
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return withSearchPath(base, schema)
}

// testSchema turns a test name into a safe, unique, lower-case schema name.
func testSchema(name string) string {
	s := strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(name, "_"))
	s = strings.Trim(s, "_")
	if len(s) > 40 {
		s = s[:40]
	}
	return "t_" + s
}

// withSearchPath appends the schema to a DSN, handling both the URL form and
// the key=value form.
func withSearchPath(dsn, schema string) string {
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	return dsn + sep + "search_path=" + schema
}
