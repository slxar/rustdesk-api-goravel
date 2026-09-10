package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/slxar/rustdesk-api-goravel/v3/global"
)

func TestBackupRejectsMissingSQLiteBeforeInitialization(t *testing.T) {
	configPath, err := filepath.Abs("../conf/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	previousConfigPath := global.ConfigPath
	global.ConfigPath = configPath
	t.Cleanup(func() { global.ConfigPath = previousConfigPath })
	t.Chdir(t.TempDir())

	if rootCmd.PersistentPreRunE == nil {
		t.Fatal("backup preflight is not registered")
	}
	if err := rootCmd.PersistentPreRunE(backupCmd, nil); err == nil {
		t.Fatal("backup accepted a missing SQLite source")
	}
	if _, err := os.Stat(filepath.Join("data", "rustdeskapi.db")); !os.IsNotExist(err) {
		t.Fatalf("backup initialized its missing source: %v", err)
	}
}
