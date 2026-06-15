package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"moneytracker/backend/internal/store"
)

// validator is optionally implemented by entity payloads to self-validate.
type validator interface {
	Validate() error
}

// crud wires a store's per-entity methods into REST handlers using generics,
// so every resource (income, expense, investment, category) shares one path.
type crud[T any] struct {
	list   func() []T
	create func(T) (T, error)
	update func(string, T) (T, error)
	delete func(string) error
}

func (c crud[T]) mount(r chi.Router, base string) {
	r.Route(base, func(r chi.Router) {
		r.Get("/", c.handleList)
		r.Post("/", c.handleCreate)
		r.Put("/{id}", c.handleUpdate)
		r.Delete("/{id}", c.handleDelete)
	})
}

func (c crud[T]) handleList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, c.list())
}

func (c crud[T]) handleCreate(w http.ResponseWriter, r *http.Request) {
	var in T
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if v, ok := any(&in).(validator); ok {
		if err := v.Validate(); err != nil {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
	}
	out, err := c.create(in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (c crud[T]) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in T
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if v, ok := any(&in).(validator); ok {
		if err := v.Validate(); err != nil {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
	}
	out, err := c.update(id, in)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (c crud[T]) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := c.delete(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
