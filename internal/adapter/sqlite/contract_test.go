package sqlite_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"app.ddd/internal/adapter/sqlite"
	"app.ddd/internal/domain/user"
)

func person(id, name, email string) user.User {
	return user.New(user.WithID(id), user.WithName(name), user.WithEmail(email))
}

func TestHostileDataRoundTripsUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hostile.db")
	require.NoError(t, sqlite.Migrate(path))
	db, err := sqlite.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := sqlite.NewUserRepository(db)
	ctx := context.Background()

	names := []string{"'; DROP TABLE users;--", `"quoted" 'quoted'`, "日本語 😀", strings.Repeat("x", 100*1024)}
	for i, name := range names {
		id := "id" + string(rune('a'+i))
		require.NoError(t, repo.Create(ctx, person(id, name, id+"@example.com")))

		got, err := repo.FindByID(ctx, id)

		require.NoError(t, err)
		assert.Equal(t, name, got.Name())
	}

	var exists user.AlreadyExistsError
	err = repo.Update(ctx, person("ida", "Changed", "idb@example.com"))
	require.True(t, errors.As(err, &exists))
	assert.Equal(t, "email", exists.Field)
}

func TestRollbackThenMigrateRestoresTheSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollback.db")
	require.NoError(t, sqlite.Migrate(path))
	require.NoError(t, sqlite.Migrate(path))
	require.NoError(t, sqlite.Rollback(path))
	require.NoError(t, sqlite.Migrate(path))
	db, err := sqlite.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = sqlite.NewUserRepository(db).List(context.Background())

	assert.NoError(t, err)
}

func TestMigrateAndOpenUseTheSameFileForPathsWithSpacesAndUnicode(t *testing.T) {
	for _, name := range []string{"with space.db", "ü.db", "日本語.db"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), name)
			require.NoError(t, sqlite.Migrate(path))
			db, err := sqlite.Open(path)
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })
			repo := sqlite.NewUserRepository(db)
			require.NoError(t, repo.Create(context.Background(), person("u1", "Ada", "ada@example.com")))

			_, err = repo.List(context.Background())

			assert.NoError(t, err)
			assert.FileExists(t, path)
		})
	}
}

func TestPathsThatTheDriversReadDifferentlyAreRejected(t *testing.T) {
	for _, name := range []string{"h#x.db", "q?x.db", "p%20x.db"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, name)

			assert.ErrorContains(t, sqlite.Migrate(path), "must not contain")
			_, err := sqlite.Open(path)
			assert.ErrorContains(t, err, "must not contain")
			entries, readErr := os.ReadDir(dir)
			require.NoError(t, readErr)
			assert.Empty(t, entries, "nothing may be created for a rejected path")
		})
	}
}
