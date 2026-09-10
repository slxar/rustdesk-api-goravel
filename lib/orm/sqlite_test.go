package orm

import (
	"path/filepath"
	"testing"
)

func TestNewSqliteEnablesDurablePragmas(t *testing.T) {
	db := NewSqlite(&SqliteConfig{Path: filepath.Join(t.TempDir(), "api.db")}, nil)
	for pragma, want := range map[string]string{
		"journal_mode": "wal", "foreign_keys": "1", "synchronous": "2", "busy_timeout": "5000",
	} {
		var got string
		if err := db.Raw("PRAGMA " + pragma).Scan(&got).Error; err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("%s = %q, want %q", pragma, got, want)
		}
	}
}

func TestBackupSQLiteCopiesDatabaseAtomically(t *testing.T) {
	dir := t.TempDir()
	db := NewSqlite(&SqliteConfig{Path: filepath.Join(dir, "api.db")}, nil)
	if err := db.Exec("CREATE TABLE records (value TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO records VALUES ('kept')").Error; err != nil {
		t.Fatal(err)
	}

	backup := filepath.Join(dir, "backup.db")
	if err := BackupSQLite(db, backup); err != nil {
		t.Fatal(err)
	}

	restored := NewSqlite(&SqliteConfig{Path: backup}, nil)
	var got string
	if err := restored.Raw("SELECT value FROM records").Scan(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got != "kept" {
		t.Fatalf("backup value = %q", got)
	}
}
