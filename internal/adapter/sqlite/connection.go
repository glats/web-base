package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stephenafamo/bob"
)

func Open(path string) (bob.DB, error) {
	if err := validatePath(path); err != nil {
		return bob.DB{}, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return bob.DB{}, fmt.Errorf("create database directory: %w", err)
	}

	sqlDB, err := sql.Open("sqlite3", "file:"+path+"?_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		return bob.DB{}, fmt.Errorf("open database: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return bob.DB{}, fmt.Errorf("connect to database: %w", err)
	}

	return bob.NewDB(sqlDB), nil
}

func validatePath(path string) error {
	if strings.ContainsAny(path, "?#%") {
		return fmt.Errorf("database path %q must not contain '?', '#' or '%%': the migration and connection drivers read them differently", path)
	}

	return nil
}
