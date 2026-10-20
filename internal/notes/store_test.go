package notes

import (
	"reflect"
	"testing"
)

func TestCreate(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		body      string
		wantTitle string
		wantBody  string
		wantID    int64
	}{
		{name: "no body defaults to empty string", title: "first", wantTitle: "first", wantBody: "", wantID: 1},
		{name: "with title and body", title: "second", body: "hello world", wantTitle: "second", wantBody: "hello world", wantID: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := NewStore()
			got := s.Create(tc.title, tc.body)

			if got.ID != tc.wantID {
				t.Errorf("Create() ID = %d, want %d", got.ID, tc.wantID)
			}
			if got.Title != tc.wantTitle {
				t.Errorf("Create() Title = %q, want %q", got.Title, tc.wantTitle)
			}
			if got.Body != tc.wantBody {
				t.Errorf("Create() Body = %q, want %q", got.Body, tc.wantBody)
			}

			// Side effect: note stored and retrievable via Get with matching title.
			stored, ok := s.Get(got.ID)
			if !ok {
				t.Fatalf("created note id %d not found in store", got.ID)
			}
			if stored.Title != tc.title {
				t.Errorf("stored Title = %q, want %q", stored.Title, tc.title)
			}

			// Invariant: ids unique and monotonically increasing starting at 1.
			second := s.Create("another", "x")
			if second.ID != got.ID+1 {
				t.Errorf("second Create() ID = %d, want %d", second.ID, got.ID+1)
			}
		})
	}
}

func TestCreateIDsUniqueAndIncrementing(t *testing.T) {
	s := NewStore()
	seen := map[int64]bool{}
	for i := int64(1); i <= 5; i++ {
		n := s.Create("t", "b")
		if n.ID != i {
			t.Errorf("Create() ID = %d, want %d", n.ID, i)
		}
		if seen[n.ID] {
			t.Errorf("duplicate id %d", n.ID)
		}
		seen[n.ID] = true
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		name     string
		id       int64
		wantOK   bool
		wantNote Note
	}{
		{
			name:     "existing note",
			id:       1,
			wantOK:   true,
			wantNote: Note{ID: 1, Title: "first", Body: ""},
		},
		{
			name:   "missing note",
			id:     9999,
			wantOK: false,
			wantNote: Note{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := NewStore()
			s.Create("first", "")

			got, ok := s.Get(tc.id)
			if ok != tc.wantOK {
				t.Fatalf("Get(%d) ok = %v, want %v", tc.id, ok, tc.wantOK)
			}
			if got != tc.wantNote {
				t.Errorf("Get(%d) = %+v, want %+v", tc.id, got, tc.wantNote)
			}
		})
	}
}

func TestList(t *testing.T) {
	tests := []struct {
		name      string
		seed      []Note
		wantCount int
	}{
		{name: "empty store returns empty slice", seed: nil, wantCount: 0},
		{name: "returns all stored notes", seed: []Note{{Title: "a", Body: "1"}, {Title: "b", Body: "2"}}, wantCount: 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := NewStore()
			for _, n := range tc.seed {
				s.Create(n.Title, n.Body)
			}

			got := s.List()
			if len(got) != tc.wantCount {
				t.Fatalf("List() length = %d, want %d", len(got), tc.wantCount)
			}

			// Invariant: mutating the returned slice does not affect the store.
			for i := range got {
				got[i].Title = "mutated"
			}
			for _, n := range s.List() {
				if n.Title == "mutated" {
					t.Errorf("List() returned a reference; mutating it affected the store")
				}
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	title := "updated"
	body := "new body"
	empty := ""

	tests := []struct {
		name    string
		setup   func(*Store) int64
		fields  UpdateFields
		want    Note
		wantOK  bool
	}{
		{
			name: "update title only",
			setup: func(s *Store) int64 {
				return s.Create("original", "body").ID
			},
			fields: UpdateFields{Title: &title},
			want:   Note{ID: 1, Title: "updated", Body: "body"},
			wantOK: true,
		},
		{
			name: "update body only",
			setup: func(s *Store) int64 {
				return s.Create("original", "body").ID
			},
			fields: UpdateFields{Body: &body},
			want:   Note{ID: 1, Title: "original", Body: "new body"},
			wantOK: true,
		},
		{
			name: "set title to empty string explicitly",
			setup: func(s *Store) int64 {
				return s.Create("original", "body").ID
			},
			fields: UpdateFields{Title: &empty},
			want:   Note{ID: 1, Title: "", Body: "body"},
			wantOK: true,
		},
		{
			name: "update both fields",
			setup: func(s *Store) int64 {
				return s.Create("original", "body").ID
			},
			fields: UpdateFields{Title: &title, Body: &body},
			want:   Note{ID: 1, Title: "updated", Body: "new body"},
			wantOK: true,
		},
		{
			name: "missing id returns false and zero note",
			setup: func(s *Store) int64 {
				s.Create("original", "body")
				return 9999
			},
			fields: UpdateFields{Title: &title},
			want:   Note{},
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := NewStore()
			id := tc.setup(s)

			got, ok := s.Update(id, tc.fields)
			if ok != tc.wantOK {
				t.Fatalf("Update(%d) ok = %v, want %v", id, ok, tc.wantOK)
			}
			if got != tc.want {
				t.Errorf("Update(%d) = %+v, want %+v", id, got, tc.want)
			}

			// Invariant: nonexistent ids are never created by update.
			if !tc.wantOK {
				if len(s.List()) != 1 {
					t.Errorf("Update on missing id created a note; store has %d notes", len(s.List()))
				}
			} else {
				stored, ok := s.Get(id)
				if !ok || stored != tc.want {
					t.Errorf("stored note = %+v (ok=%v), want %+v", stored, ok, tc.want)
				}
			}
		})
	}
}

func TestRemove(t *testing.T) {
	tests := []struct {
		name       string
		id         int64
		wantResult bool
	}{
		{name: "existing note", id: 1, wantResult: true},
		{name: "missing note id 9999", id: 9999, wantResult: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := NewStore()
			s.Create("first", "")

			got := s.Remove(tc.id)
			if got != tc.wantResult {
				t.Errorf("Remove(%d) = %v, want %v", tc.id, got, tc.wantResult)
			}

			if tc.wantResult {
				if _, ok := s.Get(tc.id); ok {
					t.Errorf("note %d still present after Remove", tc.id)
				}
			} else {
				if _, ok := s.Get(1); !ok {
					t.Errorf("Remove on missing id deleted an existing note")
				}
			}
		})
	}
}

func TestStoreIsolation(t *testing.T) {
	// Invariant: in-memory; state does not persist across instances.
	a := NewStore()
	a.Create("a", "a-body")
	b := NewStore()
	if len(b.List()) != 0 {
		t.Errorf("new store should be empty, got %+v", b.List())
	}
	if !reflect.DeepEqual(len(a.List()), 1) {
		t.Errorf("first store should have 1 note")
	}
}

func TestZeroValueStoreUsable(t *testing.T) {
	var s Store
	s.notes = make(map[int64]Note)
	s.nextID = 1
	n := s.Create("zero", "")
	if n.ID != 1 {
		t.Errorf("Create() on zero-value store ID = %d, want 1", n.ID)
	}
}