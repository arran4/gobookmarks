package gobookmarks

import "strings"

// DisplayName returns a useful name for the entry.
func (e *BookmarkEntry) DisplayName() string {
	if strings.TrimSpace(e.Name) != "" {
		return e.Name
	}
	return e.Url
}

// DisplayName returns a useful name for the category.
func (c *BookmarkCategory) DisplayName() string {
	if strings.TrimSpace(c.Name) != "" {
		return c.Name
	}
	if len(c.Entries) == 1 {
		e := c.Entries[0]
		if strings.TrimSpace(e.Name) != "" {
			return e.Name
		}
		return e.Url
	}
	return ""
}

// DisplayName returns a useful name for the page.
func (p *BookmarkPage) DisplayName() string {
	if strings.TrimSpace(p.Name) != "" {
		return p.Name
	}
	// gather category names
	var cats []*BookmarkCategory
	for _, b := range p.Blocks {
		for _, col := range b.Columns {
			cats = append(cats, col.Categories...)
		}
	}
	if len(cats) == 1 {
		return cats[0].DisplayName()
	}
	if len(cats) == 2 {
		n1 := cats[0].DisplayName()
		n2 := cats[1].DisplayName()
		if n1 != "" && n2 != "" && len(n1) <= 15 && len(n2) <= 15 {
			return n1 + ", " + n2
		}
	}
	return ""
}

// DisplayName returns a useful name for the tab.
func (t *BookmarkTab) DisplayName() string {
	if strings.TrimSpace(t.Name) != "" {
		return t.Name
	}
	if len(t.Pages) == 1 {
		if n := t.Pages[0].DisplayName(); n != "" {
			return n
		}
	}
	if len(t.Pages) == 2 {
		n1 := t.Pages[0].DisplayName()
		n2 := t.Pages[1].DisplayName()
		if n1 != "" && n2 != "" && len(n1) <= 15 && len(n2) <= 15 {
			return n1 + ", " + n2
		}
	}
	return ""
}
