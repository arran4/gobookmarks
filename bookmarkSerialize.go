package gobookmarks

import "strings"

// SerializeBookmarks converts a tree of BookmarkTab back into the
// textual bookmark representation understood by PreprocessBookmarks.
func SerializeBookmarks(tabs []*BookmarkTab) string {
	var b strings.Builder
	for ti, t := range tabs {
		// omit explicit Tab directive for the first unnamed tab
		if !(ti == 0 && t.Name == "") {
			if t.Name != "" {
				b.WriteString("Tab: ")
				b.WriteString(t.Name)
				b.WriteString("\n")
			} else {
				b.WriteString("Tab\n")
			}
		}
		for pj, p := range t.Pages {
			if pj == 0 {
				if p.Name != "" {
					b.WriteString("Page: ")
					b.WriteString(p.Name)
					b.WriteString("\n")
				}
			} else {
				if p.Name != "" {
					b.WriteString("Page: ")
					b.WriteString(p.Name)
					b.WriteString("\n")
				} else {
					b.WriteString("Page\n")
				}
			}
			for _, blk := range p.Blocks {
				if blk.HR {
					b.WriteString("--\n")
					continue
				}
				for ci, col := range blk.Columns {
					if ci > 0 {
						b.WriteString("Column\n")
					}
					for _, cat := range col.Categories {
						b.WriteString("Category: ")
						b.WriteString(cat.Name)
						b.WriteString("\n")
						for _, ent := range cat.Entries {
							b.WriteString(ent.Url)
							if ent.Name != ent.Url {
								b.WriteString(" ")
								b.WriteString(ent.Name)
							}
							b.WriteString("\n")
						}
					}
				}
			}
		}
	}
	return b.String()
}
