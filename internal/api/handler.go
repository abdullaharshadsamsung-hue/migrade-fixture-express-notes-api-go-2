// Package api contains the HTTP handlers migrated from the source
// Express application (index.js). All routes are registered on a
// chi.Router via RegisterRoutes, which is called from cmd/server.
package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"migrated-app/internal/notes"
)

// API holds the dependencies needed by the note handlers.
type API struct {
	store *notes.Store
}

// New returns an API backed by the given store.
func New(store *notes.Store) *API {
	return &API{store: store}
}

// RegisterRoutes registers every route this package owns on r:
//
//	GET    /health       -> health check, {"status":"ok"}
//	GET    /notes        -> list all notes
//	POST   /notes        -> create a note
//	GET    /notes/{id}   -> fetch one note by id
//	PUT    /notes/{id}   -> partially update a note
//	DELETE /notes/{id}   -> remove a note
//
// The source's Express global express.json() body-parsing middleware is
// reproduced per-handler by decodeJSON, since Go's net/http has no
// implicit body parsing.
func RegisterRoutes(r chi.Router) {
	a := New(notes.NewStore())
	r.Get("/health", a.handleHealth)
	r.Get("/notes", a.handleList)
	r.Post("/notes", a.handleCreate)
	r.Get("/notes/{id}", a.handleGet)
	r.Put("/notes/{id}", a.handleUpdate)
	r.Delete("/notes/{id}", a.handleRemove)
}

// writeJSON writes v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a JSON error body of the shape {"error": msg}.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// decodeJSON decodes the request body into dst. A completely empty body
// is tolerated (treated like the source's `req.body || {}` fallback) and
// returns nil. Any malformed JSON yields an error, which callers map to
// 400.
func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}

// parseID converts a path id to an int64. Non-numeric ids (or any value
// outside int64 range) return 0, which matches no note — mirroring the
// source's Number() coercion where NaN never equals a stored id.
func parseID(raw string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

// handleHealth handles GET /health.
func (a *API) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleList handles GET /notes.
func (a *API) handleList(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.store.List())
}

// handleGet handles GET /notes/{id}.
func (a *API) handleGet(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(chi.URLParam(r, "id"))
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	note, found := a.store.Get(id)
	if !found {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, note)
}

// createRequest is the accepted POST /notes payload.
type createRequest struct {
	Title *string `json:"title"`
	Body  *string `json:"body"`
}

// handleCreate handles POST /notes.
func (a *API) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Title == nil || *req.Title == "" {
		// Source: !title || typeof title !== "string" -> 400. Numbers,
		// null and booleans decode into non-string fields here and land
		// in the same 400 branch.
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	body := ""
	if req.Body != nil {
		body = *req.Body
	}
	writeJSON(w, http.StatusCreated, a.store.Create(*req.Title, body))
}

// updateRequest is the accepted PUT /notes/{id} payload. Pointers
// distinguish "field present" from "field absent", matching the source's
// conditional spread of the partial update payload.
type updateRequest struct {
	Title *string `json:"title"`
	Body  *string `json:"body"`
}

// handleUpdate handles PUT /notes/{id}.
func (a *API) handleUpdate(w http.ResponseWriter, r *http.Request) {
	var req updateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Title != nil && *req.Title == "" {
		writeError(w, http.StatusBadRequest, "title must be a non-empty string")
		return
	}
	id, ok := parseID(chi.URLParam(r, "id"))
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	note, found := a.store.Update(id, notes.UpdateFields{Title: req.Title, Body: req.Body})
	if !found {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, note)
}

// handleRemove handles DELETE /notes/{id}.
func (a *API) handleRemove(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(chi.URLParam(r, "id"))
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if !a.store.Remove(id) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ensure strings is used (keeps imports tidy if future handlers trim input).
var _ = strings.TrimSpace
