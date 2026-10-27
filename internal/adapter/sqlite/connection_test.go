package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenConfiguresTheConnection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "app.db")

	db, err := Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	tests := []struct {
		pragma string
		want   any
	}{
		{pragma: "foreign_keys", want: int64(1)},
		{pragma: "busy_timeout", want: int64(5000)},
		{pragma: "journal_mode", want: "wal"},
	}

	for _, tt := range tests {
		t.Run(tt.pragma, func(t *testing.T) {
			var got any
			err := db.QueryRowContext(context.Background(), "PRAGMA "+tt.pragma).Scan(&got)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
	assert.FileExists(t, path)
}

func TestOpenFailsWhenTheDirectoryCannotBeCreated(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocker, nil, 0o600))

	_, err := Open(filepath.Join(blocker, "app.db"))

	assert.Error(t, err)
}
