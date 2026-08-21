package dsn

import (
	"testing"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
)

func sqliteCfg(name string) *config.Config {
	cfg := &config.Config{}
	cfg.DB.Name = name

	return cfg
}

func TestCreateSQLite_ReturnsBarePath(t *testing.T) {
	got := CreateSQLite(sqliteCfg("/var/lib/go-pdns/go-pdns.db"))

	want := "/var/lib/go-pdns/go-pdns.db"
	if got != want {
		t.Fatalf("CreateSQLite = %q, want %q", got, want)
	}
}

func TestCreateSQLiteGorm_EnablesForeignKeys(t *testing.T) {
	got := CreateSQLiteGorm(sqliteCfg("/var/lib/go-pdns/go-pdns.db"))

	want := "/var/lib/go-pdns/go-pdns.db?_pragma=foreign_keys(1)"
	if got != want {
		t.Fatalf("CreateSQLiteGorm = %q, want %q", got, want)
	}
}

// The session DB path is derived from the bare path, so CreateSQLite must not carry
// DSN query parameters that would corrupt the sibling file name.
func TestCreateSQLite_SafeForSessionPathDerivation(t *testing.T) {
	sessionPath := CreateSQLite(sqliteCfg("go-pdns.db")) + "-sessions.db"

	want := "go-pdns.db-sessions.db"
	if sessionPath != want {
		t.Fatalf("session path = %q, want %q", sessionPath, want)
	}
}
