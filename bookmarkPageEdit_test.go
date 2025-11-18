package gobookmarks

import "testing"

const pageBookmarkText = `Tab: One
Page: First
Category: A
--
Page: Second
Category: B
Tab: Two
Page: Third
Category: C
`

func TestExtractPage(t *testing.T) {
	text, name, err := ExtractPage(pageBookmarkText, 0, 1)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if text != "Category: B\n" {
		t.Fatalf("expected Category B text, got %q", text)
	}
	if name != "Second" {
		t.Fatalf("expected name Second, got %q", name)
	}
}

func TestExtractPageErrors(t *testing.T) {
	if _, _, err := ExtractPage(pageBookmarkText, 2, 0); err == nil {
		t.Fatalf("expected tab error")
	}
	if _, _, err := ExtractPage(pageBookmarkText, 0, 3); err == nil {
		t.Fatalf("expected page error")
	}
}
