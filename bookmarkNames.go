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

// IndexName returns a name suitable for the navigation index.
func (p *BookmarkPage) IndexName() string {
	return p.DisplayName()
}

// DisplayName returns a useful name for the tab.
func (t *BookmarkTab) DisplayName() string {
	if strings.TrimSpace(t.Name) != "" {
		return t.Name
	}
	var pages []*BookmarkPage
	for _, p := range t.Pages {
		if !p.IsEmpty() {
			pages = append(pages, p)
		}
	}
	if len(pages) == 1 {
		if n := pages[0].DisplayName(); n != "" {
			return n
		}
	}
	if len(pages) == 2 {
		n1 := pages[0].DisplayName()
		n2 := pages[1].DisplayName()
		if n1 != "" && n2 != "" && len(n1) <= 15 && len(n2) <= 15 {
			return n1 + ", " + n2
		}
	}
	return ""
}

// IndexName returns a name suitable for the navigation index.
func (t *BookmarkTab) IndexName() string {
	return t.DisplayName()
}
