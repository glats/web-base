package sqlite

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"github.com/stephenafamo/bob"
)

//go:embed seeds/*.sql
var seedSQL embed.FS

func Seed(ctx context.Context, db bob.DB) error {
	files, err := fs.Sub(seedSQL, "seeds")
	if err != nil {
		return fmt.Errorf("open seeds: %w", err)
	}
	return seedFiles(ctx, db, files)
}

func seedFiles(ctx context.Context, db bob.DB, files fs.FS) error {
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return fmt.Errorf("list seeds: %w", err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin seeds: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		path := entry.Name()
		data, err := fs.ReadFile(files, path)
		if err != nil {
			return fmt.Errorf("read seed %s: %w", path, err)
		}
		if _, err := tx.ExecContext(ctx, string(data)); err != nil {
			return fmt.Errorf("apply seed %s: %w", path, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit seeds: %w", err)
	}
	return nil
}
