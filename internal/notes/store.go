package notes

import (
	"sync"
)

// Note is a single note in the in-memory store.
type Note struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

// UpdateFields carries optional fields for Update. A nil pointer means
// "leave this field unchanged"; a non-nil pointer (even to "") means
// "set this field to this value".
type UpdateFields struct {
	Title *string
	Body  *string
}

// Store is a concurrency-safe in-memory store for notes. The zero value
// is ready to use; NewStore is provided for clarity.
type Store struct {
	mu     sync.RWMutex
	notes  map[int64]Note
	nextID int64
}

// NewStore returns an empty, ready-to-use Store.
func NewStore() *Store {
	// The source Express app starts with an empty notes list; ids are
	// auto-incremented from 1 by POST /notes.
	return &Store{notes: make(map[int64]Note), nextID: 1}
}

// List returns a snapshot of all notes, in unspecified order.
func (s *Store) List() []Note {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Note, 0, len(s.notes))
	for _, n := range s.notes {
		out = append(out, n)
	}
	return out
}

// Get returns the note with the given id, and whether it exists.
func (s *Store) Get(id int64) (Note, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.notes[id]
	return n, ok
}

// Create adds a new note with an auto-incrementing id and returns it.
// The source's default-parameter behavior (body defaults to "") is
// preserved by the handler layer; Store itself always stores exactly
// what it is given.
func (s *Store) Create(title, body string) Note {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := Note{ID: s.nextID, Title: title, Body: body}
	s.notes[n.ID] = n
	s.nextID++
	return n
}

// Update applies the non-nil fields of f to the note with the given id.
// It returns the updated note and true if the id exists, or the zero
// Note and false otherwise. The stored note is not mutated in place;
// callers receive a copy, matching value semantics.
func (s *Store) Update(id int64, f UpdateFields) (Note, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.notes[id]
	if !ok {
		return Note{}, false
	}
	if f.Title != nil {
		n.Title = *f.Title
	}
	if f.Body != nil {
		n.Body = *f.Body
	}
	s.notes[id] = n
	return n, true
}

// Remove deletes the note with the given id and reports whether it existed.
func (s *Store) Remove(id int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.notes[id]
	if !ok {
		return false
	}
	delete(s.notes, id)
	return true
}
