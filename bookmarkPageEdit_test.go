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

func TestExtractPageErrors(t *testing.T) {
	if _, _, err := ExtractPage(pageBookmarkText, 2, 0); err == nil {
		t.Fatalf("expected tab error")
	}
	if _, _, err := ExtractPage(pageBookmarkText, 0, 3); err == nil {
		t.Fatalf("expected page error")
	}
}
