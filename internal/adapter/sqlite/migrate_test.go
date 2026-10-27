package sqlite

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tableNames(t *testing.T, path string) []string {
	t.Helper()

	db, err := sql.Open("sqlite3", path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	var names []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		names = append(names, name)
	}
	require.NoError(t, rows.Err())
	return names
}

func appliedMigrations(t *testing.T, path string) int {
	t.Helper()

	db, err := sql.Open("sqlite3", path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count))
	return count
}

func TestMigrateCreatesTheSchemaAndIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "app.db")

	require.NoError(t, Migrate(path))
	require.NoError(t, Migrate(path))

	assert.Equal(t, []string{"schema_migrations", "users"}, tableNames(t, path))
	assert.Equal(t, 1, appliedMigrations(t, path))
}

func TestRollbackRemovesTheLastMigrationAndMigrateRestoresIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	require.NoError(t, Migrate(path))

	require.NoError(t, Rollback(path))
	assert.Equal(t, []string{"schema_migrations"}, tableNames(t, path))

	require.NoError(t, Migrate(path))
	assert.Equal(t, []string{"schema_migrations", "users"}, tableNames(t, path))
}

func TestRollbackFailsWhenNothingIsApplied(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	require.NoError(t, Migrate(path))
	require.NoError(t, Rollback(path))

	assert.Error(t, Rollback(path))
}

func TestMigrateFailsWhenTheDirectoryCannotBeCreated(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocker, nil, 0o600))

	assert.Error(t, Migrate(filepath.Join(blocker, "app.db")))
}
