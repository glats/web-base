package web

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"app.ddd/internal/app/command"
	"app.ddd/internal/domain/user"
)

const maxBodyBytes = 1 << 20

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	name, email, ok := h.readForm(w, r)
	if !ok {
		return
	}

	err := h.app.Command.CreateUser.Handle(r.Context(), command.CreateUser{ID: h.newID(), Name: name, Email: email})
	if status, message, isFormError := formError(err); isFormError {
		view, listErr := h.sectionView(r)
		if listErr != nil {
			h.fail(w, r, listErr)
			return
		}
		view.Form = formView{Name: name, Email: email, Error: message}
		h.render(w, r, status, view)
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}

	h.renderFresh(w, r, http.StatusCreated)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	name, email, ok := h.readForm(w, r)
	if !ok {
		return
	}

	err := h.app.Command.UpdateUser.Handle(r.Context(), command.UpdateUser{ID: id, Name: name, Email: email})
	if status, message, isFormError := formError(err); isFormError {
		view, listErr := h.sectionView(r)
		if listErr != nil {
			h.fail(w, r, listErr)
			return
		}
		view.Editing = &editView{ID: id, Name: name, Email: email, Error: message}
		h.render(w, r, status, view)
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}

	h.renderFresh(w, r, http.StatusOK)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	if err := h.app.Command.DeleteUser.Handle(r.Context(), command.DeleteUser{ID: r.PathValue("id")}); err != nil {
		h.fail(w, r, err)
		return
	}

	h.renderFresh(w, r, http.StatusOK)
}

func (h *Handler) renderFresh(w http.ResponseWriter, r *http.Request, status int) {
	view, err := h.sectionView(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	h.render(w, r, status, view)
}

func (h *Handler) readForm(w http.ResponseWriter, r *http.Request) (name, email string, ok bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	err := r.ParseForm()
	if err == nil {
		err = r.ParseMultipartForm(maxBodyBytes)
	}
	if err != nil && !errors.Is(err, http.ErrNotMultipart) {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			h.flash(w, r, http.StatusRequestEntityTooLarge, "The request is too large.")
			return "", "", false
		}
		h.flash(w, r, http.StatusBadRequest, "The request could not be read.")
		return "", "", false
	}

	return strings.TrimSpace(r.PostFormValue("name")), strings.TrimSpace(r.PostFormValue("email")), true
}

func formError(err error) (status int, message string, isFormError bool) {
	var invalid user.InvalidError
	var exists user.AlreadyExistsError

	switch {
	case errors.As(err, &invalid) && (invalid.Field == "name" || invalid.Field == "email"):
		return http.StatusUnprocessableEntity, "Please enter a valid " + invalid.Field + ".", true
	case errors.As(err, &exists) && exists.Field == "email":
		return http.StatusConflict, "That email is already in use.", true
	default:
		return 0, "", false
	}
}

func randomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return hex.EncodeToString(b)
}
