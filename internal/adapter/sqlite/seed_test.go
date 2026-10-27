package sqlite

import (
	"context"
	"io/fs"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"app.ddd/internal/domain/user"
)

func TestSeedIsRepeatableAndDoesNotOverwriteUsers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	require.NoError(t, Migrate(path))
	db, err := Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	repo := NewUserRepository(db)
	existing := user.New(user.WithID("existing"), user.WithName("My Ada"), user.WithEmail("ada@example.com"))
	require.NoError(t, repo.Create(ctx, existing))
	require.NoError(t, Seed(ctx, db))
	require.NoError(t, Seed(ctx, db))
	users, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, users, 2)
	ada, err := repo.FindByID(ctx, "existing")
	require.NoError(t, err)
	assert.Equal(t, existing, ada)
	grace, err := repo.FindByEmail(ctx, "grace@example.com")
	require.NoError(t, err)
	assert.Equal(t, "Grace Hopper", grace.Name())
}

func TestSeedFilesAreOrderedAndAtomic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	require.NoError(t, Migrate(path))
	db, err := Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	files := fstest.MapFS{
		"02_update.sql": {Data: []byte("UPDATE users SET name='Changed' WHERE id='seed';")},
		"01_create.sql": {Data: []byte("INSERT INTO users VALUES ('seed','Original','seed@example.com');")},
	}
	require.NoError(t, seedFiles(context.Background(), db, files))
	got, err := NewUserRepository(db).FindByID(context.Background(), "seed")
	require.NoError(t, err)
	assert.Equal(t, "Changed", got.Name())
	files["03_broken.sql"] = &fstest.MapFile{Data: []byte("INVALID SQL")}
	files["01_create.sql"] = &fstest.MapFile{Data: []byte("INSERT INTO users VALUES ('other','Other','other@example.com');")}
	files["02_update.sql"] = &fstest.MapFile{Data: []byte("UPDATE users SET name='Corrupted' WHERE id='seed';")}
	assert.Error(t, seedFiles(context.Background(), db, files))
	users, err := NewUserRepository(db).List(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []user.User{got}, users)
}

func TestSeedPreservesAnExistingDemoID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	require.NoError(t, Migrate(path))
	db, err := Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewUserRepository(db)
	ctx := context.Background()
	existing := user.New(user.WithID("demo-ada"), user.WithName("Edited Ada"), user.WithEmail("edited@example.com"))
	require.NoError(t, repo.Create(ctx, existing))
	require.NoError(t, Seed(ctx, db))
	require.NoError(t, Seed(ctx, db))
	got, err := repo.FindByID(ctx, "demo-ada")
	require.NoError(t, err)
	assert.Equal(t, existing, got)
	users, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestSeedCancellationAndUnreadableFilesPreserveData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	require.NoError(t, Migrate(path))
	db, err := Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.ErrorIs(t, Seed(ctx, db), context.Canceled)
	assert.Error(t, seedFiles(context.Background(), db, unreadableSeeds{}))
	users, err := NewUserRepository(db).List(context.Background())
	require.NoError(t, err)
	assert.Empty(t, users)
}

type unreadableSeeds struct{}

func (unreadableSeeds) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
}
