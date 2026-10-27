package sqlite

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/sqlite"

	"app.ddd/internal/adapter/sqlite/migrations"
)

func newDBMate(path string) (*dbmate.DB, error) {
	if err := validatePath(path); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	u, err := url.Parse("sqlite:" + path)
	if err != nil {
		return nil, fmt.Errorf("parse database path: %w", err)
	}

	db := dbmate.New(u)
	db.FS = migrations.Files
	db.MigrationsDir = []string{"."}
	db.AutoDumpSchema = false
	db.Log = io.Discard

	return db, nil
}

func Migrate(path string) error {
	db, err := newDBMate(path)
	if err != nil {
		return err
	}

	if err := db.Migrate(); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

func Rollback(path string) error {
	db, err := newDBMate(path)
	if err != nil {
		return err
	}

	if err := db.Rollback(); err != nil {
		return fmt.Errorf("roll back migration: %w", err)
	}

	return nil
}
