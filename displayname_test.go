package gobookmarks

import "testing"

func TestBookmarkEntryDisplayName(t *testing.T) {
	e := &BookmarkEntry{Url: "http://a.com", Name: "A"}
	if e.DisplayName() != "A" {
		t.Fatalf("expected A got %q", e.DisplayName())
	}
	e2 := &BookmarkEntry{Url: "http://b.com"}
	if e2.DisplayName() != "http://b.com" {
		t.Fatalf("expected url fallback got %q", e2.DisplayName())
	}
}

func TestBookmarkCategoryDisplayName(t *testing.T) {
	c := &BookmarkCategory{Name: "C"}
	if c.DisplayName() != "C" {
		t.Fatalf("expected C got %q", c.DisplayName())
	}
	c2 := &BookmarkCategory{Entries: []*BookmarkEntry{{Url: "u", Name: "N"}}}
	if c2.DisplayName() != "N" {
		t.Fatalf("expected N got %q", c2.DisplayName())
	}
	c3 := &BookmarkCategory{Entries: []*BookmarkEntry{{Url: "u1"}, {Url: "u2"}}}
	if c3.DisplayName() != "" {
		t.Fatalf("expected empty got %q", c3.DisplayName())
	}
}

func TestBookmarkPageDisplayName(t *testing.T) {
	p := &BookmarkPage{Name: "P"}
	if p.DisplayName() != "P" {
		t.Fatalf("expected P got %q", p.DisplayName())
	}
	p2 := &BookmarkPage{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{Categories: []*BookmarkCategory{{Entries: []*BookmarkEntry{{Url: "u", Name: "N"}}}}}}}}}
	if p2.DisplayName() != "N" {
		t.Fatalf("expected N got %q", p2.DisplayName())
	}
	p3 := &BookmarkPage{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{Categories: []*BookmarkCategory{{Name: "A"}, {Name: "B"}}}}}}}
	if p3.DisplayName() != "A, B" {
		t.Fatalf("expected \"A, B\" got %q", p3.DisplayName())
	}
}

func TestBookmarkTabDisplayName(t *testing.T) {
	t1 := &BookmarkTab{Name: "T"}
	if t1.DisplayName() != "T" {
		t.Fatalf("expected T got %q", t1.DisplayName())
	}
	page := &BookmarkPage{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{Categories: []*BookmarkCategory{{Entries: []*BookmarkEntry{{Name: "E"}}}}}}}}}
	t2 := &BookmarkTab{Pages: []*BookmarkPage{page}}
	if t2.DisplayName() != "E" {
		t.Fatalf("expected E got %q", t2.DisplayName())
	}

	empty := &BookmarkTab{}
	if empty.DisplayName() != "Main" {
		t.Fatalf("expected Main got %q", empty.DisplayName())
	}
}
