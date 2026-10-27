package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"app.ddd/internal/adapter/config"
)

func TestMain(m *testing.M) {
	log.Logger = zerolog.Nop()
	os.Exit(m.Run())
}

type runningServer struct {
	baseURL string
	stop    func() error
}

func start(t *testing.T, databasePath string, seed ...bool) runningServer {
	t.Helper()

	cfg := config.Config{
		HTTP:     config.HTTPConfig{Addr: "127.0.0.1:0"},
		Database: config.DatabaseConfig{Path: databasePath},
	}
	if len(seed) > 0 {
		cfg.Seed.Enabled = seed[0]
	}
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan net.Addr, 1)
	done := make(chan error, 1)
	go func() { done <- run(ctx, cfg, func(addr net.Addr) { ready <- addr }) }()

	var addr net.Addr
	select {
	case addr = <-ready:
	case err := <-done:
		cancel()
		t.Fatalf("server stopped before listening: %v", err)
	case <-time.After(30 * time.Second):
		cancel()
		t.Fatal("server did not start")
	}

	stopped := false
	stop := func() error {
		if stopped {
			return nil
		}
		stopped = true
		cancel()
		select {
		case err := <-done:
			return err
		case <-time.After(30 * time.Second):
			t.Fatal("server did not stop")
			return nil
		}
	}
	t.Cleanup(func() { _ = stop() })

	return runningServer{baseURL: "http://" + addr.String(), stop: stop}
}

func TestStartupMigratesBeforeOptionalSeeding(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "disabled", true: "enabled"}[enabled], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "app.db")
			first := start(t, path, enabled)
			status, body := request(t, http.MethodGet, first.baseURL+"/users", nil)
			require.Equal(t, http.StatusOK, status)
			assert.Equal(t, enabled, strings.Contains(body, "Ada Lovelace"))
			assert.Equal(t, enabled, strings.Contains(body, "Grace Hopper"))
			require.NoError(t, first.stop())
			second := start(t, path, enabled)
			status, body = request(t, http.MethodGet, second.baseURL+"/users", nil)
			require.Equal(t, http.StatusOK, status)
			assert.Equal(t, map[bool]int{false: 0, true: 1}[enabled], strings.Count(body, "<td>Ada Lovelace</td>"))
			require.NoError(t, second.stop())
		})
	}
}

func request(t *testing.T, method, target string, form url.Values) (int, string) {
	t.Helper()

	req, err := http.NewRequest(method, target, strings.NewReader(form.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	return res.StatusCode, string(body)
}

func TestTheUserMaintainerWorksOverRealHTTPAndSQLite(t *testing.T) {
	server := start(t, filepath.Join(t.TempDir(), "data", "app.db"))

	status, body := request(t, http.MethodGet, server.baseURL+"/users", nil)
	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, "No users yet")

	status, body = request(t, http.MethodPost, server.baseURL+"/users", url.Values{"name": {"Ada Lovelace"}, "email": {"ada@example.com"}})
	assert.Equal(t, http.StatusCreated, status)
	assert.Contains(t, body, "Ada Lovelace")

	status, body = request(t, http.MethodPost, server.baseURL+"/users", url.Values{"name": {"Other"}, "email": {"ada@example.com"}})
	assert.Equal(t, http.StatusConflict, status)
	assert.Contains(t, body, "That email is already in use.")

	status, body = request(t, http.MethodPost, server.baseURL+"/users", url.Values{"name": {"Other"}, "email": {"nope"}})
	assert.Equal(t, http.StatusUnprocessableEntity, status)
	assert.Contains(t, body, "Please enter a valid email.")

	match := regexp.MustCompile(`hx-get="/users/([0-9a-f]{32})/edit"`).FindStringSubmatch(body)
	require.Len(t, match, 2, "the listed user has a generated id")
	id := match[1]

	status, body = request(t, http.MethodGet, server.baseURL+"/users/"+id+"/edit", nil)
	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, `hx-put="/users/`+id+`"`)

	status, body = request(t, http.MethodPut, server.baseURL+"/users/"+id, url.Values{"name": {"Ada King"}, "email": {"king@example.com"}})
	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, "Ada King")
	assert.NotContains(t, body, "Ada Lovelace")

	status, body = request(t, http.MethodDelete, server.baseURL+"/users/"+id, nil)
	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, "No users yet")

	status, _ = request(t, http.MethodDelete, server.baseURL+"/users/"+id, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status, body = request(t, http.MethodGet, server.baseURL+"/static/htmx.min.js", nil)
	assert.Equal(t, http.StatusOK, status)
	assert.NotEmpty(t, body)
}

func TestUsersSurviveARestartAndShutdownIsGraceful(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")

	first := start(t, path)
	status, _ := request(t, http.MethodPost, first.baseURL+"/users", url.Values{"name": {"Grace Hopper"}, "email": {"grace@example.com"}})
	require.Equal(t, http.StatusCreated, status)
	require.NoError(t, first.stop())

	second := start(t, path)
	status, body := request(t, http.MethodGet, second.baseURL+"/users", nil)

	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, "Grace Hopper")
	assert.NoError(t, second.stop())
}

func TestRunFailsWhenTheDatabaseCannotBePrepared(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocker, nil, 0o600))
	cfg := config.Config{
		HTTP:     config.HTTPConfig{Addr: "127.0.0.1:0"},
		Database: config.DatabaseConfig{Path: filepath.Join(blocker, "app.db")},
	}

	err := run(context.Background(), cfg, nil)

	assert.Error(t, err)
}

func TestRunFailsWhenTheAddressIsInvalid(t *testing.T) {
	cfg := config.Config{
		HTTP:     config.HTTPConfig{Addr: "not-an-address"},
		Database: config.DatabaseConfig{Path: filepath.Join(t.TempDir(), "app.db")},
	}

	err := run(context.Background(), cfg, nil)

	assert.Error(t, err)
}
