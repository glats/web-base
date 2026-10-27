package web_test

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	appmock "app.ddd/internal/app/mock"
	"app.ddd/internal/domain/user"
	"app.ddd/internal/port/web"
	"app.ddd/internal/service"
)

var (
	ada   = user.New(user.WithID("u1"), user.WithName("Ada Lovelace"), user.WithEmail("ada@example.com"))
	grace = user.New(user.WithID("u2"), user.WithName("Grace Hopper"), user.WithEmail("grace@example.com"))
)

func newHandler(t *testing.T, repo *appmock.UserRepository, opts ...web.Option) http.Handler {
	t.Helper()

	quiet := web.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))

	return web.NewHandler(service.NewApp(repo), append([]web.Option{quiet}, opts...)...)
}

func fromHTMX(r *http.Request) {
	r.Header.Set("HX-Request", "true")
}

func do(t *testing.T, h http.Handler, method, target string, mods ...func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, nil)
	for _, mod := range mods {
		mod(req)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRootRedirectsToUsers(t *testing.T) {
	rec := do(t, newHandler(t, appmock.NewUserRepository(t)), http.MethodGet, "/")

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/users", rec.Header().Get("Location"))
}

func TestListRendersTheFullPage(t *testing.T) {
	repo := appmock.NewUserRepository(t)
	repo.EXPECT().List(mock.Anything).Return([]user.User{ada, grace}, nil).Once()

	rec := do(t, newHandler(t, repo), http.MethodGet, "/users")

	body := rec.Body.String()
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	for _, want := range []string{
		"<!doctype html>",
		`<link rel="icon" href="data:,">`,
		`href="/static/pico.min.css"`,
		`src="/static/htmx.min.js"`,
		`id="flash"`,
		`id="users-section"`,
		`hx-post="/users"`,
		"Ada Lovelace", "ada@example.com", "Grace Hopper",
		`hx-get="/users/u1/edit"`,
		`hx-delete="/users/u1"`,
		`hx-confirm="Delete Ada Lovelace?"`,
	} {
		assert.Contains(t, body, want)
	}
	assert.Less(t, strings.Index(body, "Ada Lovelace"), strings.Index(body, "Grace Hopper"))
}

func TestListFromHTMXRendersOnlyTheSection(t *testing.T) {
	repo := appmock.NewUserRepository(t)
	repo.EXPECT().List(mock.Anything).Return([]user.User{ada}, nil).Once()

	rec := do(t, newHandler(t, repo), http.MethodGet, "/users", fromHTMX)

	body := rec.Body.String()
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, body, `id="users-section"`)
	assert.Contains(t, body, "Ada Lovelace")
	assert.NotContains(t, body, "<html")
	assert.NotContains(t, body, "<!doctype")
}

func TestListShowsAMessageWhenThereAreNoUsers(t *testing.T) {
	repo := appmock.NewUserRepository(t)
	repo.EXPECT().List(mock.Anything).Return([]user.User{}, nil).Once()

	rec := do(t, newHandler(t, repo), http.MethodGet, "/users")

	assert.Contains(t, rec.Body.String(), "No users yet")
}

func TestListEscapesUserData(t *testing.T) {
	hostile := user.New(user.WithID("u9"), user.WithName("<script>alert(1)</script>"), user.WithEmail("x@example.com"))
	repo := appmock.NewUserRepository(t)
	repo.EXPECT().List(mock.Anything).Return([]user.User{hostile}, nil).Once()

	rec := do(t, newHandler(t, repo), http.MethodGet, "/users")

	assert.Contains(t, rec.Body.String(), "&lt;script&gt;alert(1)&lt;/script&gt;")
	assert.NotContains(t, rec.Body.String(), "<script>alert(1)")
}

func TestEditRendersTheRowInEditMode(t *testing.T) {
	repo := appmock.NewUserRepository(t)
	repo.EXPECT().FindByID(mock.Anything, "u1").Return(ada, nil).Once()
	repo.EXPECT().List(mock.Anything).Return([]user.User{ada, grace}, nil).Once()

	rec := do(t, newHandler(t, repo), http.MethodGet, "/users/u1/edit", fromHTMX)

	body := rec.Body.String()
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, body, `hx-put="/users/u1"`)
	assert.Contains(t, body, `value="Ada Lovelace"`)
	assert.Contains(t, body, `value="ada@example.com"`)
	assert.Contains(t, body, `hx-get="/users/u2/edit"`)
	assert.NotContains(t, body, `hx-get="/users/u1/edit"`)
	assert.NotContains(t, body, "<html")
}

func TestEditRendersTheFullPageWithoutHTMX(t *testing.T) {
	repo := appmock.NewUserRepository(t)
	repo.EXPECT().FindByID(mock.Anything, "u1").Return(ada, nil).Once()
	repo.EXPECT().List(mock.Anything).Return([]user.User{ada}, nil).Once()

	rec := do(t, newHandler(t, repo), http.MethodGet, "/users/u1/edit")

	assert.Contains(t, rec.Body.String(), "<!doctype html>")
	assert.Contains(t, rec.Body.String(), `hx-put="/users/u1"`)
}

func TestNotFoundIsRetargetedToTheFlashAreaForHTMX(t *testing.T) {
	repo := appmock.NewUserRepository(t)
	repo.EXPECT().FindByID(mock.Anything, "nope").Return(user.User{}, user.NotFoundError{Field: "id", Value: "nope"}).Once()

	rec := do(t, newHandler(t, repo), http.MethodGet, "/users/nope/edit", fromHTMX)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "#flash", rec.Header().Get("HX-Retarget"))
	assert.Equal(t, "innerHTML", rec.Header().Get("HX-Reswap"))
	assert.Contains(t, rec.Body.String(), "User not found")
	assert.NotContains(t, rec.Body.String(), "<html")
}

func TestNotFoundRendersAPageWithoutHTMX(t *testing.T) {
	repo := appmock.NewUserRepository(t)
	repo.EXPECT().FindByID(mock.Anything, "nope").Return(user.User{}, user.NotFoundError{Field: "id", Value: "nope"}).Once()

	rec := do(t, newHandler(t, repo), http.MethodGet, "/users/nope/edit")

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Empty(t, rec.Header().Get("HX-Retarget"))
	assert.Contains(t, rec.Body.String(), "<!doctype html>")
	assert.Contains(t, rec.Body.String(), "User not found")
	assert.Contains(t, rec.Body.String(), `href="/users"`)
}

func TestUnexpectedErrorsAreHiddenFromTheUserAndLogged(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))

	tests := []struct {
		name string
		mods []func(*http.Request)
	}{
		{name: "page"},
		{name: "htmx", mods: []func(*http.Request){fromHTMX}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs.Reset()
			repo := appmock.NewUserRepository(t)
			repo.EXPECT().List(mock.Anything).Return(nil, errors.New("disk exploded")).Once()

			rec := do(t, newHandler(t, repo, web.WithLogger(logger)), http.MethodGet, "/users", tt.mods...)

			assert.Equal(t, http.StatusInternalServerError, rec.Code)
			assert.Contains(t, rec.Body.String(), "Something went wrong")
			assert.NotContains(t, rec.Body.String(), "disk exploded")
			assert.Contains(t, logs.String(), "disk exploded")
		})
	}
}

func TestStaticFilesAreServedFromTheBinary(t *testing.T) {
	h := newHandler(t, appmock.NewUserRepository(t))

	tests := []struct {
		path     string
		wantCode int
		wantType string
	}{
		{path: "/static/htmx.min.js", wantCode: http.StatusOK, wantType: "javascript"},
		{path: "/static/pico.min.css", wantCode: http.StatusOK, wantType: "text/css"},
		{path: "/static/THIRD_PARTY.md", wantCode: http.StatusOK},
		{path: "/static/missing.js", wantCode: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := do(t, h, http.MethodGet, tt.path)

			assert.Equal(t, tt.wantCode, rec.Code)
			if tt.wantCode == http.StatusOK {
				assert.NotEmpty(t, rec.Body.Bytes())
				assert.Contains(t, rec.Header().Get("Content-Type"), tt.wantType)
			}
		})
	}
}

func TestUnknownRoutesAreNotFound(t *testing.T) {
	rec := do(t, newHandler(t, appmock.NewUserRepository(t)), http.MethodGet, "/nope")

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHTMLResponsesVaryOnTheHTMXHeader(t *testing.T) {
	tests := []struct {
		name   string
		target string
		setup  func(repo *appmock.UserRepository)
	}{
		{
			name:   "list",
			target: "/users",
			setup: func(repo *appmock.UserRepository) {
				repo.EXPECT().List(mock.Anything).Return([]user.User{ada}, nil).Once()
			},
		},
		{
			name:   "error page",
			target: "/users/nope/edit",
			setup: func(repo *appmock.UserRepository) {
				repo.EXPECT().FindByID(mock.Anything, "nope").Return(user.User{}, user.NotFoundError{Field: "id", Value: "nope"}).Once()
			},
		},
	}

	for _, tt := range tests {
		for _, mods := range [][]func(*http.Request){nil, {fromHTMX}} {
			t.Run(tt.name, func(t *testing.T) {
				repo := appmock.NewUserRepository(t)
				tt.setup(repo)

				rec := do(t, newHandler(t, repo), http.MethodGet, tt.target, mods...)

				assert.Contains(t, rec.Header().Values("Vary"), "HX-Request")
			})
		}
	}
}
