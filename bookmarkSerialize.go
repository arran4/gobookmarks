package gobookmarks

import "strings"

func SerializeBookmarks(tabs []*BookmarkTab) string {
	var lines []string
	for ti, t := range tabs {
		if ti > 0 || t.Name != "" {
			if t.Name != "" {
				lines = append(lines, "Tab: "+t.Name)
			} else {
				lines = append(lines, "Tab:")
			}
		}
		for pi, p := range t.Pages {
			if pi > 0 || p.Name != "" {
				if p.Name != "" {
					lines = append(lines, "Page: "+p.Name)
				} else {
					lines = append(lines, "Page")
				}
			} else if p.Name != "" {
				lines = append(lines, "Page: "+p.Name)
			}
			for _, b := range p.Blocks {
				if b.HR {
					lines = append(lines, "--")
				}
				for ci, c := range b.Columns {
					if ci > 0 {
						lines = append(lines, "Column")
					}
					for _, cat := range c.Categories {
						if cat.Name != "" {
							lines = append(lines, "Category: "+cat.Name)
						} else {
							lines = append(lines, "Category:")
						}
						for _, e := range cat.Entries {
							if e.Name == e.Url {
								lines = append(lines, e.Url)
							} else {
								lines = append(lines, e.Url+" "+e.Name)
							}
						}
					}
				}
			}
		}
	}
	return strings.Join(lines, "\n")
}
