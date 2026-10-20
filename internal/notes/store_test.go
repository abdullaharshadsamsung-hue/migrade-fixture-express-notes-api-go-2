package notes

import "testing"

func TestCreateThenGet(t *testing.T) {
	s := NewStore()
	n := s.Create("t1", "b1")
	if n.ID != 1 {
		t.Fatalf("first id = %d, want 1", n.ID)
	}
	got, ok := s.Get(n.ID)
	if !ok {
		t.Fatalf("Get(%d) not found", n.ID)
	}
	if got.Title != "t1" || got.Body != "b1" {
		t.Fatalf("got %+v, want title/body t1/b1", got)
	}
	if _, ok := s.Get(99); ok {
		t.Fatal("Get(99) should not exist")
	}
}

func TestRemoveMissing(t *testing.T) {
	s := NewStore()
	if s.Remove(42) {
		t.Fatal("Remove on missing id should return false")
	}
}

func TestUpdateMissing(t *testing.T) {
	s := NewStore()
	title := "x"
	n, ok := s.Update(7, UpdateFields{Title: &title})
	if ok {
		t.Fatalf("Update on missing id should return false, got %+v", n)
	}
	if n != (Note{}) {
		t.Fatalf("expected zero Note, got %+v", n)
	}
}

func TestUpdateNilPointersLeaveFieldsUnchanged(t *testing.T) {
	s := NewStore()
	created := s.Create("keep", "orig")
	empty := ""
	n, ok := s.Update(created.ID, UpdateFields{Title: &empty})
	if !ok {
		t.Fatal("update should succeed")
	}
	if n.Body != "orig" {
		t.Fatalf("nil Body changed body to %q", n.Body)
	}
	if n.Title != "" {
		t.Fatalf("empty-string title should be set, got %q", n.Title)
	}
}

func TestDefaultBodyEmpty(t *testing.T) {
	s := NewStore()
	n := s.Create("only title", "")
	if n.Body != "" {
		t.Fatalf("default body = %q, want empty", n.Body)
	}
}

func TestListReturnsCopies(t *testing.T) {
	s := NewStore()
	s.Create("a", "b")
	list := s.List()
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}
	list[0].Title = "mutated"
	fresh := s.List()
	if fresh[0].Title != "a" {
		t.Fatalf("List did not return copies: stored title = %q", fresh[0].Title)
	}
}
