package web_test

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	appmock "app.ddd/internal/app/mock"
	"app.ddd/internal/domain/user"
	"app.ddd/internal/port/web"
)

var (
	errBoom       = errors.New("boom")
	idNotFound    = func(id string) error { return user.NotFoundError{Field: "id", Value: id} }
	emailNotFound = func(email string) error { return user.NotFoundError{Field: "email", Value: email} }
	newUser       = user.New(user.WithID("new-id"), user.WithName("Alan Turing"), user.WithEmail("alan@example.com"))
)

func withFixedID(id string) web.Option {
	return web.WithIDGenerator(func() string { return id })
}

func send(t *testing.T, h http.Handler, method, target string, form url.Values, mods ...func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, mod := range mods {
		mod(req)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func form(name, email string) url.Values {
	return url.Values{"name": {name}, "email": {email}}
}

func expectList(repo *appmock.UserRepository, users ...user.User) {
	repo.EXPECT().List(mock.Anything).Return(users, nil).Once()
}

func expectFreeUser(repo *appmock.UserRepository, id, email string) {
	repo.EXPECT().FindByID(mock.Anything, id).Return(user.User{}, idNotFound(id)).Once()
	repo.EXPECT().FindByEmail(mock.Anything, email).Return(user.User{}, emailNotFound(email)).Once()
}

func TestCreateAddsTheUserAndAnswersTheSection(t *testing.T) {
	repo := appmock.NewUserRepository(t)
	expectFreeUser(repo, "new-id", "alan@example.com")
	repo.EXPECT().Create(mock.Anything, newUser).Return(nil).Once()
	expectList(repo, newUser)

	rec := send(t, newHandler(t, repo, withFixedID("new-id")), http.MethodPost, "/users", form("Alan Turing", "alan@example.com"), fromHTMX)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), `id="users-section"`)
	assert.Contains(t, rec.Body.String(), "Alan Turing")
	assert.NotContains(t, rec.Body.String(), `aria-invalid`)
	assert.NotContains(t, rec.Body.String(), "<html")
}

func TestCreateTrimsTheSubmittedValues(t *testing.T) {
	repo := appmock.NewUserRepository(t)
	expectFreeUser(repo, "new-id", "alan@example.com")
	repo.EXPECT().Create(mock.Anything, newUser).Return(nil).Once()
	expectList(repo, newUser)

	rec := send(t, newHandler(t, repo, withFixedID("new-id")), http.MethodPost, "/users", form("  Alan Turing ", " alan@example.com\t"), fromHTMX)

	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestCreateGeneratesARandomIDByDefault(t *testing.T) {
	var ids []string
	repo := appmock.NewUserRepository(t)
	repo.EXPECT().FindByID(mock.Anything, mock.Anything).Return(user.User{}, idNotFound("x")).Times(2)
	repo.EXPECT().FindByEmail(mock.Anything, mock.Anything).Return(user.User{}, emailNotFound("x")).Times(2)
	repo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(u user.User) bool {
		ids = append(ids, u.ID())
		return true
	})).Return(nil).Times(2)
	repo.EXPECT().List(mock.Anything).Return([]user.User{}, nil).Times(2)
	h := newHandler(t, repo)

	send(t, h, http.MethodPost, "/users", form("A", "a@example.com"), fromHTMX)
	send(t, h, http.MethodPost, "/users", form("B", "b@example.com"), fromHTMX)

	if assert.Len(t, ids, 2) {
		assert.Regexp(t, `^[0-9a-f]{32}$`, ids[0])
		assert.NotEqual(t, ids[0], ids[1])
	}
}

func TestCreateRejectsInvalidAndDuplicateData(t *testing.T) {
	tests := []struct {
		name        string
		form        url.Values
		setup       func(repo *appmock.UserRepository)
		wantStatus  int
		wantMessage string
	}{
		{
			name:        "malformed email",
			form:        form("Alan", "alan"),
			setup:       func(repo *appmock.UserRepository) {},
			wantStatus:  http.StatusUnprocessableEntity,
			wantMessage: "Please enter a valid email.",
		},
		{
			name:        "blank name",
			form:        form("   ", "alan@example.com"),
			setup:       func(repo *appmock.UserRepository) {},
			wantStatus:  http.StatusUnprocessableEntity,
			wantMessage: "Please enter a valid name.",
		},
		{
			name: "email already in use",
			form: form("Alan", "ada@example.com"),
			setup: func(repo *appmock.UserRepository) {
				repo.EXPECT().FindByID(mock.Anything, "new-id").Return(user.User{}, idNotFound("new-id")).Once()
				repo.EXPECT().FindByEmail(mock.Anything, "ada@example.com").Return(ada, nil).Once()
			},
			wantStatus:  http.StatusConflict,
			wantMessage: "That email is already in use.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := appmock.NewUserRepository(t)
			tt.setup(repo)
			expectList(repo, ada)

			rec := send(t, newHandler(t, repo, withFixedID("new-id")), http.MethodPost, "/users", tt.form, fromHTMX)

			body := rec.Body.String()
			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Contains(t, body, tt.wantMessage)
			assert.Contains(t, body, `aria-invalid="true"`)
			assert.Contains(t, body, `value="`+strings.TrimSpace(tt.form.Get("email"))+`"`)
			assert.Contains(t, body, "Ada Lovelace")
			assert.Empty(t, rec.Header().Get("HX-Retarget"))
		})
	}
}

func TestCreateKeepsTheSubmittedValuesInTheFormOnError(t *testing.T) {
	repo := appmock.NewUserRepository(t)
	expectList(repo)

	rec := send(t, newHandler(t, repo), http.MethodPost, "/users", form(`Al "the" <b>`, "bad"), fromHTMX)

	assert.Contains(t, rec.Body.String(), `value="Al &#34;the&#34; &lt;b&gt;"`)
}

func TestCreateFailuresAreFlashedWithoutLeakingTheCause(t *testing.T) {
	tests := []struct {
		name         string
		mods         []func(*http.Request)
		wantRetarget bool
	}{
		{name: "htmx", mods: []func(*http.Request){fromHTMX}, wantRetarget: true},
		{name: "page"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := appmock.NewUserRepository(t)
			repo.EXPECT().FindByID(mock.Anything, "new-id").Return(user.User{}, errBoom).Once()

			rec := send(t, newHandler(t, repo, withFixedID("new-id")), http.MethodPost, "/users", form("Alan", "alan@example.com"), tt.mods...)

			assert.Equal(t, http.StatusInternalServerError, rec.Code)
			assert.Contains(t, rec.Body.String(), "Something went wrong")
			assert.NotContains(t, rec.Body.String(), "boom")
			if tt.wantRetarget {
				assert.Equal(t, "#flash", rec.Header().Get("HX-Retarget"))
			} else {
				assert.Empty(t, rec.Header().Get("HX-Retarget"))
			}
		})
	}
}

func TestCreateAcceptsMultipartForms(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("name", "Alan Turing")
	_ = writer.WriteField("email", "alan@example.com")
	_ = writer.Close()
	repo := appmock.NewUserRepository(t)
	expectFreeUser(repo, "new-id", "alan@example.com")
	repo.EXPECT().Create(mock.Anything, newUser).Return(nil).Once()
	expectList(repo, newUser)
	req := httptest.NewRequest(http.MethodPost, "/users", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	newHandler(t, repo, withFixedID("new-id")).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestRequestBodiesAreLimitedToOneMiB(t *testing.T) {
	repo := appmock.NewUserRepository(t)
	h := newHandler(t, repo)
	huge := url.Values{"name": {strings.Repeat("a", 2<<20)}, "email": {"a@example.com"}}

	tests := []struct {
		method string
		target string
	}{
		{method: http.MethodPost, target: "/users"},
		{method: http.MethodPut, target: "/users/u1"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			rec := send(t, h, tt.method, tt.target, huge, fromHTMX)

			assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
			assert.Contains(t, rec.Body.String(), "too large")
			assert.Equal(t, "#flash", rec.Header().Get("HX-Retarget"))
		})
	}
}

func TestUpdateChangesTheUserAndAnswersTheSection(t *testing.T) {
	changed := user.New(user.WithID("u1"), user.WithName("Ada King"), user.WithEmail("king@example.com"))
	repo := appmock.NewUserRepository(t)
	repo.EXPECT().FindByID(mock.Anything, "u1").Return(ada, nil).Once()
	repo.EXPECT().FindByEmail(mock.Anything, "king@example.com").Return(user.User{}, emailNotFound("king@example.com")).Once()
	repo.EXPECT().Update(mock.Anything, changed).Return(nil).Once()
	expectList(repo, changed)

	rec := send(t, newHandler(t, repo), http.MethodPut, "/users/u1", form("Ada King", "king@example.com"), fromHTMX)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Ada King")
	assert.NotContains(t, rec.Body.String(), `hx-put=`)
}

func TestUpdateRejectsInvalidAndDuplicateDataKeepingTheRowInEditMode(t *testing.T) {
	tests := []struct {
		name        string
		form        url.Values
		setup       func(repo *appmock.UserRepository)
		wantStatus  int
		wantMessage string
	}{
		{
			name: "malformed email",
			form: form("Ada", "ada"),
			setup: func(repo *appmock.UserRepository) {
			},
			wantStatus:  http.StatusUnprocessableEntity,
			wantMessage: "Please enter a valid email.",
		},
		{
			name: "email of another user",
			form: form("Ada", "grace@example.com"),
			setup: func(repo *appmock.UserRepository) {
				repo.EXPECT().FindByID(mock.Anything, "u1").Return(ada, nil).Once()
				repo.EXPECT().FindByEmail(mock.Anything, "grace@example.com").Return(grace, nil).Once()
			},
			wantStatus:  http.StatusConflict,
			wantMessage: "That email is already in use.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := appmock.NewUserRepository(t)
			tt.setup(repo)
			expectList(repo, ada, grace)

			rec := send(t, newHandler(t, repo), http.MethodPut, "/users/u1", tt.form, fromHTMX)

			body := rec.Body.String()
			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Contains(t, body, tt.wantMessage)
			assert.Contains(t, body, `hx-put="/users/u1"`)
			assert.Contains(t, body, `value="`+tt.form.Get("email")+`"`)
			assert.Contains(t, body, `aria-invalid="true"`)
			assert.Contains(t, body, `hx-get="/users/u2/edit"`)
			assert.Empty(t, rec.Header().Get("HX-Retarget"))
		})
	}
}

func TestUpdateAndDeleteOfAnUnknownUserAreFlashedAsNotFound(t *testing.T) {
	tests := []struct {
		name   string
		method string
		form   url.Values
	}{
		{name: "update", method: http.MethodPut, form: form("Ada", "ada@example.com")},
		{name: "delete", method: http.MethodDelete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := appmock.NewUserRepository(t)
			repo.EXPECT().FindByID(mock.Anything, "u1").Return(user.User{}, idNotFound("u1")).Once()

			rec := send(t, newHandler(t, repo), tt.method, "/users/u1", tt.form, fromHTMX)

			assert.Equal(t, http.StatusNotFound, rec.Code)
			assert.Equal(t, "#flash", rec.Header().Get("HX-Retarget"))
			assert.Equal(t, "innerHTML", rec.Header().Get("HX-Reswap"))
			assert.Contains(t, rec.Body.String(), "User not found")
		})
	}
}

func TestUpdateAndDeleteRepositoryFailuresAreFlashed(t *testing.T) {
	tests := []struct {
		name   string
		method string
		form   url.Values
	}{
		{name: "update", method: http.MethodPut, form: form("Ada", "ada@example.com")},
		{name: "delete", method: http.MethodDelete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := appmock.NewUserRepository(t)
			repo.EXPECT().FindByID(mock.Anything, "u1").Return(user.User{}, errBoom).Once()

			rec := send(t, newHandler(t, repo), tt.method, "/users/u1", tt.form, fromHTMX)

			assert.Equal(t, http.StatusInternalServerError, rec.Code)
			assert.Equal(t, "#flash", rec.Header().Get("HX-Retarget"))
			assert.NotContains(t, rec.Body.String(), "boom")
		})
	}
}

func TestDeleteRemovesTheUserAndAnswersTheSection(t *testing.T) {
	repo := appmock.NewUserRepository(t)
	repo.EXPECT().FindByID(mock.Anything, "u1").Return(ada, nil).Once()
	repo.EXPECT().Delete(mock.Anything, "u1").Return(nil).Once()
	expectList(repo, grace)

	rec := send(t, newHandler(t, repo), http.MethodDelete, "/users/u1", nil, fromHTMX)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Grace Hopper")
	assert.NotContains(t, rec.Body.String(), "Ada Lovelace")
}

func TestListAfterAMutationFailsGracefully(t *testing.T) {
	repo := appmock.NewUserRepository(t)
	repo.EXPECT().FindByID(mock.Anything, "u1").Return(ada, nil).Once()
	repo.EXPECT().Delete(mock.Anything, "u1").Return(nil).Once()
	repo.EXPECT().List(mock.Anything).Return(nil, errBoom).Once()

	rec := send(t, newHandler(t, repo), http.MethodDelete, "/users/u1", nil, fromHTMX)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "#flash", rec.Header().Get("HX-Retarget"))
}

func TestCrossOriginWritesAreRejected(t *testing.T) {
	tests := []struct {
		name string
		mod  func(*http.Request)
	}{
		{name: "cross-site fetch metadata", mod: func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") }},
		{name: "foreign origin", mod: func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") }},
	}

	for _, tt := range tests {
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
			t.Run(tt.name+" "+method, func(t *testing.T) {
				repo := appmock.NewUserRepository(t)
				target := "/users"
				if method != http.MethodPost {
					target = "/users/u1"
				}

				rec := send(t, newHandler(t, repo), method, target, form("Ada", "ada@example.com"), tt.mod)

				assert.Equal(t, http.StatusForbidden, rec.Code)
			})
		}
	}
}

func TestSameOriginAndSafeRequestsAreAllowed(t *testing.T) {
	repo := appmock.NewUserRepository(t)
	repo.EXPECT().FindByID(mock.Anything, "u1").Return(ada, nil).Once()
	repo.EXPECT().Delete(mock.Anything, "u1").Return(nil).Once()
	expectList(repo)
	expectList(repo, ada)
	h := newHandler(t, repo)

	deleted := send(t, h, http.MethodDelete, "/users/u1", nil, func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "same-origin") })
	listed := do(t, h, http.MethodGet, "/users", func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") })

	assert.Equal(t, http.StatusOK, deleted.Code)
	assert.Equal(t, http.StatusOK, listed.Code)
}

func TestFormErrorsFailWhenTheListCannotBeLoaded(t *testing.T) {
	tests := []struct {
		name   string
		method string
		target string
		form   url.Values
		setup  func(repo *appmock.UserRepository)
	}{
		{
			name: "create", method: http.MethodPost, target: "/users", form: form("Ada", "ada"),
			setup: func(repo *appmock.UserRepository) {},
		},
		{
			name: "update", method: http.MethodPut, target: "/users/u1", form: form("Ada", "ada"),
			setup: func(repo *appmock.UserRepository) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := appmock.NewUserRepository(t)
			tt.setup(repo)
			repo.EXPECT().List(mock.Anything).Return(nil, errBoom).Once()

			rec := send(t, newHandler(t, repo), tt.method, tt.target, tt.form, fromHTMX)

			assert.Equal(t, http.StatusInternalServerError, rec.Code)
			assert.Equal(t, "#flash", rec.Header().Get("HX-Retarget"))
		})
	}
}

func TestEditFailsWhenTheListCannotBeLoaded(t *testing.T) {
	repo := appmock.NewUserRepository(t)
	repo.EXPECT().FindByID(mock.Anything, "u1").Return(ada, nil).Once()
	repo.EXPECT().List(mock.Anything).Return(nil, errBoom).Once()

	rec := do(t, newHandler(t, repo), http.MethodGet, "/users/u1/edit", fromHTMX)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "#flash", rec.Header().Get("HX-Retarget"))
}

func TestAnUnreadableFormIsABadRequest(t *testing.T) {
	repo := appmock.NewUserRepository(t)
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader("--x"))
	req.Header.Set("Content-Type", "multipart/form-data")
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	newHandler(t, repo).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "could not be read")
	assert.Equal(t, "#flash", rec.Header().Get("HX-Retarget"))
}
