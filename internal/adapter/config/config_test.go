package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func unsetEnv(t *testing.T, keys ...string) {
	t.Helper()

	for _, key := range keys {
		t.Setenv(key, "")
		require.NoError(t, os.Unsetenv(key))
	}
}

func TestParseUsesDefaults(t *testing.T) {
	unsetEnv(t, "HTTP_ADDR", "DATABASE_PATH", "LOG_LEVEL", "LOG_PRETTY", "DB_SEED_ENABLED")

	cfg, err := Parse()

	require.NoError(t, err)
	assert.Equal(t, ":8080", cfg.HTTP.Addr)
	assert.Equal(t, "./data/app.db", cfg.Database.Path)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.False(t, cfg.Log.Pretty)
	assert.False(t, cfg.Seed.Enabled)
}

func TestParseReadsTheEnvironment(t *testing.T) {
	t.Setenv("HTTP_ADDR", "127.0.0.1:9000")
	t.Setenv("DATABASE_PATH", "/tmp/users.db")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_PRETTY", "true")
	t.Setenv("DB_SEED_ENABLED", "true")

	cfg, err := Parse()

	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:9000", cfg.HTTP.Addr)
	assert.Equal(t, "/tmp/users.db", cfg.Database.Path)
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.True(t, cfg.Log.Pretty)
	assert.True(t, cfg.Seed.Enabled)
}

func TestParseRejectsInvalidBooleanSettings(t *testing.T) {
	for _, key := range []string{"LOG_PRETTY", "DB_SEED_ENABLED"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, "invalid")
			_, err := Parse()
			assert.ErrorContains(t, err, "invalid")
		})
	}
}

func TestGetReturnsTheParsedConfiguration(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":7000")
	unsetEnv(t, "DATABASE_PATH")

	cfg := Get()

	assert.Equal(t, ":7000", cfg.HTTP.Addr)
	assert.Equal(t, "./data/app.db", cfg.Database.Path)
}
