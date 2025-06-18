package gobookmarks

import "strings"

// BookmarkEntry represents a single link.
type BookmarkEntry struct {
	Url  string
	Name string
}

// String serializes the entry.
func (e *BookmarkEntry) String() string {
	if e == nil {
		return ""
	}
	if e.Name != "" && e.Name != e.Url {
		return e.Url + " " + e.Name + "\n"
	}
	return e.Url + "\n"
}

// BookmarkCategory groups entries together.
type BookmarkCategory struct {
	Name    string
	Entries []*BookmarkEntry
	Index   int
}

// String serializes the category.
func (c *BookmarkCategory) String() string {
	var b strings.Builder
	b.WriteString("Category: ")
	b.WriteString(c.Name)
	b.WriteString("\n")
	for _, e := range c.Entries {
		b.WriteString(e.String())
	}
	return b.String()
}

// BookmarkColumn contains a list of categories.
type BookmarkColumn struct {
	Categories []*BookmarkCategory
}

// String serializes the column.
func (c *BookmarkColumn) String() string {
	var b strings.Builder
	for _, cat := range c.Categories {
		b.WriteString(cat.String())
	}
	return b.String()
}

// AddCategory appends a category to the column.
func (c *BookmarkColumn) AddCategory(cat *BookmarkCategory) {
	c.Categories = append(c.Categories, cat)
}

// InsertCategory inserts a category at the given index.
func (c *BookmarkColumn) InsertCategory(idx int, cat *BookmarkCategory) {
	if idx < 0 || idx > len(c.Categories) {
		return
	}
	c.Categories = append(c.Categories, nil)
	copy(c.Categories[idx+1:], c.Categories[idx:])
	c.Categories[idx] = cat
}

// SwitchCategories swaps two categories in the column.
func (c *BookmarkColumn) SwitchCategories(i, j int) {
	if i < 0 || j < 0 || i >= len(c.Categories) || j >= len(c.Categories) {
		return
	}
	c.Categories[i], c.Categories[j] = c.Categories[j], c.Categories[i]
}

// BookmarkBlock groups columns and optional horizontal rule.
type BookmarkBlock struct {
	Columns []*BookmarkColumn
	HR      bool
}

// String serializes the block.
func (b *BookmarkBlock) String() string {
	if b.HR {
		return "--\n"
	}
	var sb strings.Builder
	for i, col := range b.Columns {
		if i > 0 {
			sb.WriteString("Column\n")
		}
		sb.WriteString(col.String())
	}
	return sb.String()
}

// BookmarkPage contains a number of blocks.
type BookmarkPage struct {
	Blocks []*BookmarkBlock
	Name   string
}

// String serializes the page (excluding the Page line).
func (p *BookmarkPage) String() string {
	var sb strings.Builder
	for _, blk := range p.Blocks {
		sb.WriteString(blk.String())
	}
	return sb.String()
}

// AddPage appends a page to the tab.
func (t *BookmarkTab) AddPage(p *BookmarkPage) {
	t.Pages = append(t.Pages, p)
}

// InsertPage inserts a page at the given index.
func (t *BookmarkTab) InsertPage(idx int, p *BookmarkPage) {
	if idx < 0 || idx > len(t.Pages) {
		return
	}
	t.Pages = append(t.Pages, nil)
	copy(t.Pages[idx+1:], t.Pages[idx:])
	t.Pages[idx] = p
}

// SwitchPages swaps two pages within the tab.
func (t *BookmarkTab) SwitchPages(i, j int) {
	if i < 0 || j < 0 || i >= len(t.Pages) || j >= len(t.Pages) {
		return
	}
	t.Pages[i], t.Pages[j] = t.Pages[j], t.Pages[i]
}

// BookmarkTab represents a tab of pages.
type BookmarkTab struct {
	Name  string
	Pages []*BookmarkPage
}

func (t *BookmarkTab) stringWithContext(first bool) string {
	var sb strings.Builder
	if !(first && t.Name == "") {
		if t.Name != "" {
			sb.WriteString("Tab: ")
			sb.WriteString(t.Name)
			sb.WriteString("\n")
		} else {
			sb.WriteString("Tab\n")
		}
	}
	for i, p := range t.Pages {
		if i == 0 {
			if p.Name != "" {
				sb.WriteString("Page: ")
				sb.WriteString(p.Name)
				sb.WriteString("\n")
			}
		} else {
			if p.Name != "" {
				sb.WriteString("Page: ")
				sb.WriteString(p.Name)
				sb.WriteString("\n")
			} else {
				sb.WriteString("Page\n")
			}
		}
		sb.WriteString(p.String())
	}
	return sb.String()
}

// String serializes the tab including Tab/Page directives.
func (t *BookmarkTab) String() string {
	return t.stringWithContext(false)
}

// Bookmarks is a collection of tabs.
type BookmarkList []*BookmarkTab

// AddTab appends a tab to the list.
func (b *BookmarkList) AddTab(t *BookmarkTab) {
	*b = append(*b, t)
}

// String serializes the bookmark list back into textual form.
func (b BookmarkList) String() string {
	var sb strings.Builder
	for i, t := range b {
		sb.WriteString(t.stringWithContext(i == 0))
	}
	return sb.String()
}

// InsertTab inserts a tab at the given index.
func (b *BookmarkList) InsertTab(idx int, t *BookmarkTab) {
	if idx < 0 || idx > len(*b) {
		return
	}
	*b = append(*b, nil)
	copy((*b)[idx+1:], (*b)[idx:])
	(*b)[idx] = t
}

// SwitchTabs swaps two tabs in the list.
func (b BookmarkList) SwitchTabs(i, j int) {
	if i < 0 || j < 0 || i >= len(b) || j >= len(b) {
		return
	}
	b[i], b[j] = b[j], b[i]
}

// ParseBookmarks converts the textual bookmark representation into a
// BookmarkList structure.
func ParseBookmarks(bookmarks string) BookmarkList {
	lines := strings.Split(bookmarks, "\n")
	var result BookmarkList
	var currentTab *BookmarkTab
	var currentPage *BookmarkPage
	var currentCategory *BookmarkCategory
	idx := 0

	ensureTab := func() *BookmarkTab {
		if currentTab == nil {
			t := &BookmarkTab{}
			result.AddTab(t)
			currentTab = t
		}
		return currentTab
	}

	ensurePage := func() *BookmarkPage {
		ensureTab()
		if currentPage == nil {
			p := &BookmarkPage{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}
			currentTab.AddPage(p)
			currentPage = p
		}
		return currentPage
	}

	flushCategory := func() {
		if currentCategory != nil {
			currentCategory.Index = idx
			idx++
			page := ensurePage()
			lastBlock := page.Blocks[len(page.Blocks)-1]
			lastColumn := lastBlock.Columns[len(lastBlock.Columns)-1]
			lastColumn.AddCategory(currentCategory)
			currentCategory = nil
		}
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		if lower == "tab" || strings.HasPrefix(lower, "tab ") || strings.HasPrefix(lower, "tab:") {
			rest := strings.TrimSpace(line[len("tab"):])
			if strings.HasPrefix(rest, ":") {
				rest = strings.TrimSpace(rest[1:])
			}
			flushCategory()
			currentTab = &BookmarkTab{Name: rest}
			currentPage = &BookmarkPage{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}
			currentTab.AddPage(currentPage)
			result.AddTab(currentTab)
			continue
		}
		if lower == "page" || strings.HasPrefix(lower, "page ") || strings.HasPrefix(lower, "page:") {
			rest := strings.TrimSpace(line[len("page"):])
			if strings.HasPrefix(rest, ":") {
				rest = strings.TrimSpace(rest[1:])
			}
			flushCategory()
			ensureTab()
			currentPage = &BookmarkPage{Name: rest, Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}
			currentTab.AddPage(currentPage)
			continue
		}
		if line == "--" {
			flushCategory()
			page := ensurePage()
			page.Blocks = append(page.Blocks, &BookmarkBlock{HR: true})
			page.Blocks = append(page.Blocks, &BookmarkBlock{Columns: []*BookmarkColumn{{}}})
			continue
		}
		if strings.EqualFold(line, "column") {
			flushCategory()
			page := ensurePage()
			lastBlock := page.Blocks[len(page.Blocks)-1]
			lastBlock.Columns = append(lastBlock.Columns, &BookmarkColumn{})
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		lowerFirst := strings.ToLower(parts[0])
		if strings.HasPrefix(lowerFirst, "category") {
			rest := strings.TrimSpace(line[len("category"):])
			if strings.HasPrefix(rest, ":") {
				rest = strings.TrimSpace(rest[1:])
			}
			if rest == "" {
				rest = "Category"
			}
			flushCategory()
			ensurePage()
			currentCategory = &BookmarkCategory{Name: rest}
		} else if currentCategory != nil {
			entry := BookmarkEntry{Url: parts[0], Name: parts[0]}
			if len(parts) > 1 {
				entry.Name = strings.Join(parts[1:], " ")
			}
			currentCategory.Entries = append(currentCategory.Entries, &entry)
		}
	}

	flushCategory()

	if len(result) == 0 {
		t := &BookmarkTab{}
		p := &BookmarkPage{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}
		t.AddPage(p)
		result.AddTab(t)
	}

	return result
}
