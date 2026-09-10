package orm

import (
	"os"
	"path/filepath"

	"gorm.io/gorm"
)

// BackupSQLite writes a consistent SQLite snapshot before atomically replacing destination.
func BackupSQLite(db *gorm.DB, destination string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(destination), "."+filepath.Base(destination)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Remove(tempPath); err != nil {
		return err
	}
	defer os.Remove(tempPath)

	if err := db.Exec("VACUUM INTO ?", tempPath).Error; err != nil {
		return err
	}
	return os.Rename(tempPath, destination)
}
