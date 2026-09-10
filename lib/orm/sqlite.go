package orm

import (
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
)

type SqliteConfig struct {
	MaxIdleConns int
	MaxOpenConns int
	Path         string
}

const DefaultSQLitePath = "./data/rustdeskapi.db"

func NewSqlite(sqliteConf *SqliteConfig, logwriter logger.Writer) *gorm.DB {
	path := sqliteConf.Path
	if path == "" {
		path = DefaultSQLitePath
	}
	db, err := gorm.Open(sqlite.Open(path+"?_busy_timeout=5000&_foreign_keys=on&_journal_mode=WAL&_synchronous=FULL"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger: logger.New(
			logwriter, // io writer
			logger.Config{
				SlowThreshold:             time.Second, // Slow SQL threshold
				LogLevel:                  logger.Warn, // Log level
				IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
				ParameterizedQueries:      true,        // Don't include params in the SQL log
				Colorful:                  true,
			},
		),
	})
	if err != nil {
		fmt.Println(err)
	}
	sqlDB, err2 := db.DB()
	if err2 != nil {
		fmt.Println(err2)
	}
	// SetMaxIdleConns 设置空闲连接池中连接的最大数量
	sqlDB.SetMaxIdleConns(sqliteConf.MaxIdleConns)

	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	sqlDB.SetMaxOpenConns(sqliteConf.MaxOpenConns)

	if err := db.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		panic(err)
	}
	if err := db.Exec("PRAGMA busy_timeout=5000").Error; err != nil {
		panic(err)
	}
	if err := db.Exec("PRAGMA foreign_keys=ON").Error; err != nil {
		panic(err)
	}
	if err := db.Exec("PRAGMA synchronous=FULL").Error; err != nil {
		panic(err)
	}

	return db
}
