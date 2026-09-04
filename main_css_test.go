package gobookmarks

import (
	"strings"
	"testing"
)

func TestBookmarkColumnsLayoutDoesNotDependOnCSSColumnsClass(t *testing.T) {
	css := string(GetMainCSSData())

	if strings.Contains(css, ".cssColumns .bookmarkColumns") {
		t.Fatal("bookmark column layout must not depend on the optional cssColumns class")
	}
	if !strings.Contains(css, ".bookmarkColumns {\n        display: flex;") {
		t.Fatal("bookmarkColumns must use flex layout by default")
	}
	if !strings.Contains(css, ".bookmarkColumn {\n        padding-right: 1em;\n        display: flex;") {
		t.Fatal("bookmarkColumn must retain its column flex layout by default")
	}
}
