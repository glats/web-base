package web

import (
	"bytes"
	"embed"
	"errors"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"

	"app.ddd/internal/app"
	"app.ddd/internal/app/query"
	"app.ddd/internal/domain/user"
)

//go:embed templates/*.html
var templateFiles embed.FS

//go:embed static
var staticFiles embed.FS

const genericErrorMessage = "Something went wrong. Please try again."

type Handler struct {
	app        *app.Application
	logger     *slog.Logger
	newID      func() string
	pages      *template.Template
	errorPages *template.Template
}

type Option func(*Handler)

func WithLogger(logger *slog.Logger) Option {
	return func(h *Handler) { h.logger = logger }
}

func WithIDGenerator(newID func() string) Option {
	return func(h *Handler) { h.newID = newID }
}

func NewHandler(application *app.Application, opts ...Option) http.Handler {
	funcs := template.FuncMap{"pathEscape": url.PathEscape}

	h := &Handler{
		app:    application,
		logger: slog.Default(),
		newID:  randomID,
		pages: template.Must(template.New("pages").Funcs(funcs).ParseFS(templateFiles,
			"templates/layout.html", "templates/users_page.html", "templates/users_section.html", "templates/flash.html")),
		errorPages: template.Must(template.New("errors").Funcs(funcs).ParseFS(templateFiles,
			"templates/layout.html", "templates/error_main.html")),
	}
	for _, opt := range opts {
		opt(h)
	}

	static, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", h.root)
	mux.HandleFunc("GET /users", h.list)
	mux.HandleFunc("GET /users/{id}/edit", h.edit)
	mux.HandleFunc("POST /users", h.create)
	mux.HandleFunc("PUT /users/{id}", h.update)
	mux.HandleFunc("DELETE /users/{id}", h.remove)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))

	return http.NewCrossOriginProtection().Handler(mux)
}

func (h *Handler) root(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/users", http.StatusFound)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	view, err := h.sectionView(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	h.render(w, r, http.StatusOK, view)
}

func (h *Handler) edit(w http.ResponseWriter, r *http.Request) {
	u, err := h.app.Query.ObtainUser.Handle(r.Context(), query.ObtainUser{ID: r.PathValue("id")})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	view, err := h.sectionView(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	view.Editing = &editView{ID: u.ID(), Name: u.Name(), Email: u.Email()}

	h.render(w, r, http.StatusOK, view)
}

func (h *Handler) sectionView(r *http.Request) (sectionView, error) {
	users, err := h.app.Query.ListUsers.Handle(r.Context(), query.ListUsers{})
	if err != nil {
		return sectionView{}, err
	}

	view := sectionView{Users: make([]userView, 0, len(users))}
	for _, u := range users {
		view.Users = append(view.Users, userView{ID: u.ID(), Name: u.Name(), Email: u.Email()})
	}

	return view, nil
}

func (h *Handler) render(w http.ResponseWriter, r *http.Request, status int, view sectionView) {
	name := "layout"
	if isHTMX(r) {
		name = "users_section"
	}

	h.execute(w, h.pages, name, status, view)
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	status, message := http.StatusInternalServerError, genericErrorMessage

	var notFound user.NotFoundError
	if errors.As(err, &notFound) {
		status, message = http.StatusNotFound, "User not found."
	} else {
		h.logger.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	}

	h.flash(w, r, status, message)
}

func (h *Handler) flash(w http.ResponseWriter, r *http.Request, status int, message string) {
	if isHTMX(r) {
		w.Header().Set("HX-Retarget", "#flash")
		w.Header().Set("HX-Reswap", "innerHTML")
		h.execute(w, h.pages, "flash", status, message)
		return
	}

	h.execute(w, h.errorPages, "layout", status, errorView{Flash: message})
}

func (h *Handler) execute(w http.ResponseWriter, set *template.Template, name string, status int, data any) {
	var body bytes.Buffer
	if err := set.ExecuteTemplate(&body, name, data); err != nil {
		h.logger.Error("render template", "template", name, "error", err)
		http.Error(w, genericErrorMessage, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Add("Vary", "HX-Request")
	w.WriteHeader(status)
	_, _ = body.WriteTo(w)
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}
