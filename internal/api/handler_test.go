package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"migrated-app/internal/notes"
)

func newTestRouter(t *testing.T) (*http.ServeMux, *notes.Store) {
	t.Helper()
	store := notes.NewStore()
	a := New(store)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.handleHealth)
	mux.HandleFunc("GET /notes", a.handleList)
	mux.HandleFunc("POST /notes", a.handleCreate)
	mux.HandleFunc("GET /notes/{id}", a.handleGet)
	mux.HandleFunc("PUT /notes/{id}", a.handleUpdate)
	mux.HandleFunc("DELETE /notes/{id}", a.handleRemove)
	return mux, store
}

func doRequest(t *testing.T, r *http.ServeMux, method, path, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var parsed map[string]any
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &parsed)
	}
	return rec, parsed
}

func TestHealth(t *testing.T) {
	r, _ := newTestRouter(t)
	rec, parsed := doRequest(t, r, http.MethodGet, "/health", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if v, ok := parsed["status"]; !ok || v != "ok" {
		t.Fatalf("body = %v, want {\"status\":\"ok\"}", parsed)
	}
}

func TestListNotes(t *testing.T) {
	tests := []struct {
		name    string
		seed    func(*notes.Store)
		wantLen int
	}{
		{name: "empty store returns empty array", seed: func(s *notes.Store) {}, wantLen: 0},
		{
			name: "store with notes returns all notes",
			seed: func(s *notes.Store) {
				s.Create("one", "body one")
				s.Create("two", "")
			},
			wantLen: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, store := newTestRouter(t)
			tt.seed(store)
			req := httptest.NewRequest(http.MethodGet, "/notes", nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			var got []notes.Note
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("body is not a JSON array: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("len(notes) = %d, want %d", len(got), tt.wantLen)
			}
			if rec.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("content-type = %q, want application/json", rec.Header().Get("Content-Type"))
			}
		})
	}
}

func TestGetNote(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		seed       bool
		wantStatus int
		wantNote   *notes.Note
	}{
		{name: "existing note", id: "1", seed: true, wantStatus: http.StatusOK, wantNote: &notes.Note{ID: 1, Title: "t", Body: "b"}},
		{name: "non-existent id", id: "99", seed: true, wantStatus: http.StatusNotFound},
		{name: "non-numeric id", id: "abc", seed: true, wantStatus: http.StatusNotFound},
		{name: "empty store", id: "1", seed: false, wantStatus: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, store := newTestRouter(t)
			if tt.seed {
				store.Create("t", "b")
			}
			rec, parsed := doRequest(t, r, http.MethodGet, "/notes/"+tt.id, "")
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantNote != nil {
				var got notes.Note
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if got != *tt.wantNote {
					t.Fatalf("note = %+v, want %+v", got, *tt.wantNote)
				}
			} else if parsed["error"] != "not found" {
				t.Fatalf("error = %v, want 'not found'", parsed["error"])
			}
		})
	}
}

func TestCreateNote(t *testing.T) {
	title := "hello"
	tests := []struct {
		name        string
		body        string
		omitBody    bool
		wantStatus  int
		wantErr     string
		wantTitle   string
		wantBodyStr string
		wantCount   int
	}{
		{name: "valid with body", body: `{"title":"hello","body":"world"}`, wantStatus: http.StatusCreated, wantTitle: "hello", wantBodyStr: "world", wantCount: 1},
		{name: "valid without body", body: `{"title":"hello"}`, wantStatus: http.StatusCreated, wantTitle: "hello", wantBodyStr: "", wantCount: 1},
		{name: "missing request body", omitBody: true, wantStatus: http.StatusBadRequest, wantErr: "title is required", wantCount: 0},
		{name: "title missing", body: `{"body":"x"}`, wantStatus: http.StatusBadRequest, wantErr: "title is required", wantCount: 0},
		{name: "title empty string", body: `{"title":""}`, wantStatus: http.StatusBadRequest, wantErr: "title is required", wantCount: 0},
		{name: "title not a string", body: `{"title":42}`, wantStatus: http.StatusBadRequest, wantErr: "title is required", wantCount: 0},
		{name: "malformed JSON", body: `{"title":`, wantStatus: http.StatusBadRequest, wantErr: "invalid JSON", wantCount: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, store := newTestRouter(t)
			var rec *httptest.ResponseRecorder
			var parsed map[string]any
			if tt.omitBody {
				rec, parsed = doRequest(t, r, http.MethodPost, "/notes", "")
			} else {
				rec, parsed = doRequest(t, r, http.MethodPost, "/notes", tt.body)
			}
			_ = title
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantErr != "" {
				if parsed["error"] != tt.wantErr {
					t.Fatalf("error = %v, want %q", parsed["error"], tt.wantErr)
				}
			} else {
				var got notes.Note
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if got.ID != 1 {
					t.Fatalf("id = %d, want 1", got.ID)
				}
				if got.Title != tt.wantTitle || got.Body != tt.wantBodyStr {
					t.Fatalf("note = %+v, want title %q body %q", got, tt.wantTitle, tt.wantBodyStr)
				}
			}
			if len(store.List()) != tt.wantCount {
				t.Fatalf("store count = %d, want %d", len(store.List()), tt.wantCount)
			}
		})
	}
}

func TestUpdateNote(t *testing.T) {
	newTitle := "updated"
	newBody := "updated body"
	tests := []struct {
		name       string
		id         string
		body       string
		wantStatus int
		wantErr    string
		wantNote   *notes.Note
		wantCount  int
	}{
		{
			name: "update title", id: "1", body: `{"title":"updated"}`,
			wantStatus: http.StatusOK, wantNote: &notes.Note{ID: 1, Title: "updated", Body: "orig"},
		},
		{
			name: "update body only", id: "1", body: `{"body":"updated body"}`,
			wantStatus: http.StatusOK, wantNote: &notes.Note{ID: 1, Title: "orig", Body: "updated body"},
		},
		{
			name: "update both", id: "1", body: `{"title":"updated","body":"updated body"}`,
			wantStatus: http.StatusOK, wantNote: &notes.Note{ID: 1, Title: "updated", Body: "updated body"},
		},
		{
			name: "empty object leaves note unchanged", id: "1", body: `{}`,
			wantStatus: http.StatusOK, wantNote: &notes.Note{ID: 1, Title: "orig", Body: "orig"},
		},
		{
			name: "empty body field sets empty string", id: "1", body: `{"body":""}`,
			wantStatus: http.StatusOK, wantNote: &notes.Note{ID: 1, Title: "orig", Body: ""},
		},
		{
			name: "empty string title rejected", id: "1", body: `{"title":""}`,
			wantStatus: http.StatusBadRequest, wantErr: "title must be a non-empty string", wantNote: &notes.Note{ID: 1, Title: "orig", Body: "orig"},
		},
		{
			name: "non-string title rejected", id: "1", body: `{"title":42}`,
			wantStatus: http.StatusBadRequest, wantErr: "title must be a non-empty string", wantNote: &notes.Note{ID: 1, Title: "orig", Body: "orig"},
		},
		{
			name: "malformed JSON", id: "1", body: `{"title":`,
			wantStatus: http.StatusBadRequest, wantErr: "invalid JSON", wantNote: &notes.Note{ID: 1, Title: "orig", Body: "orig"},
		},
		{
			name: "id not found", id: "99", body: `{"title":"updated"}`,
			wantStatus: http.StatusNotFound, wantErr: "not found", wantCount: 1,
		},
		{
			name: "non-numeric id", id: "abc", body: `{"title":"updated"}`,
			wantStatus: http.StatusNotFound, wantErr: "not found", wantCount: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, store := newTestRouter(t)
			store.Create("orig", "orig")
			rec, parsed := doRequest(t, r, http.MethodPut, "/notes/"+tt.id, tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantErr != "" {
				if parsed["error"] != tt.wantErr {
					t.Fatalf("error = %v, want %q", parsed["error"], tt.wantErr)
				}
			} else {
				var got notes.Note
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if got != *tt.wantNote {
					t.Fatalf("note = %+v, want %+v", got, *tt.wantNote)
				}
			}
			if tt.wantCount > 0 && len(store.List()) != tt.wantCount {
				t.Fatalf("store count = %d, want %d", len(store.List()), tt.wantCount)
			}
			// Verify persisted state matches expectation.
			if tt.wantNote != nil {
				stored, ok := store.Get(tt.wantNote.ID)
				if !ok || stored != *tt.wantNote {
					t.Fatalf("stored note = %+v (ok=%v), want %+v", stored, ok, *tt.wantNote)
				}
			}
			_ = newTitle
			_ = newBody
		})
	}
}

func TestDeleteNote(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		seed       bool
		wantStatus int
		wantErr    string
		wantCount  int
	}{
		{name: "existing note returns 204", id: "1", seed: true, wantStatus: http.StatusNoContent, wantCount: 0},
		{name: "id not found", id: "99", seed: true, wantStatus: http.StatusNotFound, wantErr: "not found", wantCount: 1},
		{name: "non-numeric id", id: "abc", seed: true, wantStatus: http.StatusNotFound, wantErr: "not found", wantCount: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, store := newTestRouter(t)
			if tt.seed {
				store.Create("t", "b")
			}
			rec, parsed := doRequest(t, r, http.MethodDelete, "/notes/"+tt.id, "")
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantErr != "" {
				if parsed["error"] != tt.wantErr {
					t.Fatalf("error = %v, want %q", parsed["error"], tt.wantErr)
				}
			} else if rec.Body.Len() != 0 {
				t.Fatalf("204 body = %q, want empty", rec.Body.String())
			}
			if len(store.List()) != tt.wantCount {
				t.Fatalf("store count = %d, want %d", len(store.List()), tt.wantCount)
			}
		})
	}
}