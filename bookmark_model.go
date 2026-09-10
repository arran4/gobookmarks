package gobookmarks

import (
	"encoding/json"
	"fmt"
	"strings"
)

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

// MoveEntry moves an entry within the category from index i to j.
func (c *BookmarkCategory) MoveEntry(i, j int) {
	if i < 0 || j < 0 || i >= len(c.Entries) || j >= len(c.Entries) || i == j {
		return
	}
	entry := c.Entries[i]
	if i < j {
		copy(c.Entries[i:j], c.Entries[i+1:j+1])
	} else {
		copy(c.Entries[j+1:i+1], c.Entries[j:i])
	}
	c.Entries[j] = entry
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

// Interchange formats for conversion
type JSONEntry struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}

type JSONCategory struct {
	Name    string       `json:"name"`
	Entries []*JSONEntry `json:"entries,omitempty"`
}

type JSONColumn struct {
	Categories []*JSONCategory `json:"categories,omitempty"`
}

type JSONBlock struct {
	HR      bool          `json:"hr"`
	Columns []*JSONColumn `json:"columns,omitempty"`
}

type JSONPage struct {
	Name   string       `json:"name,omitempty"`
	Blocks []*JSONBlock `json:"blocks,omitempty"`
}

type JSONTab struct {
	Name        string      `json:"name,omitempty"`
	ExplicitTab bool        `json:"explicit,omitempty"`
	Pages       []*JSONPage `json:"pages,omitempty"`
}

// ToJSON transforms a BookmarkList into a stable JSON struct format.
func (b BookmarkList) ToJSON() []*JSONTab {
	var tabs []*JSONTab
	for _, t := range b {
		jTab := &JSONTab{
			Name:        t.Name,
			ExplicitTab: t.ExplicitTab,
		}
		for _, p := range t.Pages {
			jPage := &JSONPage{
				Name: p.Name,
			}
			for _, blk := range p.Blocks {
				jBlk := &JSONBlock{
					HR: blk.HR,
				}
				for _, col := range blk.Columns {
					jCol := &JSONColumn{}
					for _, cat := range col.Categories {
						jCat := &JSONCategory{
							Name: cat.Name,
						}
						for _, ent := range cat.Entries {
							jCat.Entries = append(jCat.Entries, &JSONEntry{
								URL:  ent.Url,
								Name: ent.Name,
							})
						}
						jCol.Categories = append(jCol.Categories, jCat)
					}
					jBlk.Columns = append(jBlk.Columns, jCol)
				}
				jPage.Blocks = append(jPage.Blocks, jBlk)
			}
			jTab.Pages = append(jTab.Pages, jPage)
		}
		tabs = append(tabs, jTab)
	}
	return tabs
}

// BookmarkListFromJSON creates a BookmarkList from the interchange JSON format.
func BookmarkListFromJSON(tabs []*JSONTab) (BookmarkList, error) {
	if len(tabs) == 0 {
		return nil, fmt.Errorf("no tabs found in json data")
	}

	var list BookmarkList
	for i, t := range tabs {
		if t == nil {
			return nil, fmt.Errorf("invalid json: null tab object")
		}
		if t.Name != "" && !t.ExplicitTab {
			return nil, fmt.Errorf("invalid json: named tab must be explicit (lossy shape)")
		}
		if i > 0 && !t.ExplicitTab {
			return nil, fmt.Errorf("invalid json: non-first tab must be explicit (lossy shape)")
		}
		bt := &BookmarkTab{
			Name:        t.Name,
			ExplicitTab: t.ExplicitTab,
		}
		if t.Pages == nil {
			// A nil page array normalizes safely to the default shape during parsing
			t.Pages = []*JSONPage{{}}
		} else if len(t.Pages) == 0 {
			return nil, fmt.Errorf("invalid json: explicitly empty pages array cannot be represented (lossy shape)")
		}

		for _, p := range t.Pages {
			if p == nil {
				return nil, fmt.Errorf("invalid json: null page object in tab %q", t.Name)
			}
			bp := &BookmarkPage{
				Name: p.Name,
			}
			if p.Blocks == nil {
				// A nil block array normalizes safely to the default shape
				p.Blocks = []*JSONBlock{{Columns: []*JSONColumn{{}}}}
			} else if len(p.Blocks) == 0 {
				return nil, fmt.Errorf("invalid json: explicitly empty blocks array cannot be represented (lossy shape)")
			}

			// Validate degenerate empty model (implicit unnamed tab, with one implicit unnamed page, with one empty block).
			// If it has NO columns, it parses back as an empty text document which strict parser will reject.
			// Native parsing always guarantees at least one column structure, even if empty.
			if len(tabs) == 1 && !t.ExplicitTab && t.Name == "" && len(t.Pages) == 1 && p.Name == "" && len(p.Blocks) == 1 && p.Blocks[0] != nil && !p.Blocks[0].HR {
				if len(p.Blocks[0].Columns) == 0 {
					return nil, fmt.Errorf("invalid json: degenerate empty block in implicit model (lossy shape)")
				}
				isEmpty := true
				for _, col := range p.Blocks[0].Columns {
					if col != nil && len(col.Categories) > 0 {
						isEmpty = false
					}
				}
				if isEmpty {
					return nil, fmt.Errorf("invalid json: degenerate completely empty implicit model cannot be serialized natively")
				}
			}

			for j, blk := range p.Blocks {
				if blk == nil {
					return nil, fmt.Errorf("invalid json: null block object in page %q", p.Name)
				}
				// Verify HR block sequence constraints to avoid lossy block shapes
				if j%2 == 0 {
					if blk.HR {
						return nil, fmt.Errorf("invalid json: even-indexed block must not be HR (lossy shape)")
					}
				} else {
					if !blk.HR {
						return nil, fmt.Errorf("invalid json: odd-indexed block must be HR (lossy shape)")
					}
				}
				if j == len(p.Blocks)-1 && blk.HR {
					return nil, fmt.Errorf("invalid json: final block must not be HR (lossy shape)")
				}

				if blk.HR && len(blk.Columns) > 0 {
					return nil, fmt.Errorf("invalid json: hr block cannot contain columns (lossy semantic shape)")
				}
				bb := &BookmarkBlock{
					HR: blk.HR,
				}
				if !blk.HR && len(blk.Columns) == 0 {
					return nil, fmt.Errorf("invalid json: block without HR must have at least one column")
				}
				for _, col := range blk.Columns {
					if col == nil {
						return nil, fmt.Errorf("invalid json: null column object")
					}
					bc := &BookmarkColumn{}
					for _, cat := range col.Categories {
						if cat == nil {
							return nil, fmt.Errorf("invalid json: null category object")
						}
						catName := strings.TrimSpace(cat.Name)
						if catName == "" {
							return nil, fmt.Errorf("invalid json: category name cannot be empty (lossy shape)")
						}
						bcat := &BookmarkCategory{
							Name: catName,
						}
						for _, ent := range cat.Entries {
							if ent == nil {
								return nil, fmt.Errorf("invalid json: null entry object in category %q", cat.Name)
							}
							entUrl := strings.TrimSpace(ent.URL)
							if entUrl == "" {
								return nil, fmt.Errorf("invalid json: entry url cannot be empty (lossy shape)")
							}
							bcat.Entries = append(bcat.Entries, &BookmarkEntry{
								Url:  entUrl,
								Name: ent.Name,
							})
						}
						bc.Categories = append(bc.Categories, bcat)
					}
					bb.Columns = append(bb.Columns, bc)
				}
				bp.Blocks = append(bp.Blocks, bb)
			}
			bt.AddPage(bp)
		}
		list.AddTab(bt)
	}

	// General representability invariant check:
	// If the constructed list serialized to string and parsed back strictly produces
	// a list with a different ToJSON() structure, it means the input JSON was lossy
	// and semantically altered by the text representation rules.
	// E.g. empty names mutating to "Category" or whitespace normalization.
	serializedList := list.String()
	reparsedList, err := StrictParseBookmarks(serializedList)
	if err != nil {
		return nil, fmt.Errorf("invalid json: semantically unrepresentable list (serialization failed to parse: %w)", err)
	}

	// Compare JSON round-trip forms since ToJSON drops internal/transient fields
	// and accurately reflects the stable semantic structure.
	originalJson, err := json.Marshal(list.ToJSON())
	if err != nil {
		return nil, fmt.Errorf("internal error marshaling model: %w", err)
	}
	reparsedJson, err := json.Marshal(reparsedList.ToJSON())
	if err != nil {
		return nil, fmt.Errorf("internal error marshaling reparsed model: %w", err)
	}

	if string(originalJson) != string(reparsedJson) {
		return nil, fmt.Errorf("invalid json: semantically unrepresentable/lossy shape (reparsing serialized text changed semantic structure)")
	}

	// Set category indices
	idx := 0
	for _, t := range list {
		for _, p := range t.Pages {
			for _, blk := range p.Blocks {
				for _, col := range blk.Columns {
					for _, cat := range col.Categories {
						cat.Index = idx
						idx++
					}
				}
			}
		}
	}
	return list, nil
}

// BookmarkPage contains a number of blocks.
type BookmarkPage struct {
	Blocks []*BookmarkBlock
	Name   string
}

// IsEmpty returns true if the page contains no categories.
func (p *BookmarkPage) IsEmpty() bool {
	for _, blk := range p.Blocks {
		for _, col := range blk.Columns {
			if len(col.Categories) > 0 {
				return false
			}
		}
	}
	return true
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

// MovePage moves a page from index i to j within the tab.
func (t *BookmarkTab) MovePage(i, j int) {
	if i < 0 || j < 0 || i >= len(t.Pages) || j >= len(t.Pages) || i == j {
		return
	}
	page := t.Pages[i]
	if i < j {
		copy(t.Pages[i:j], t.Pages[i+1:j+1])
	} else {
		copy(t.Pages[j+1:i+1], t.Pages[j:i])
	}
	t.Pages[j] = page
}

// BookmarkTab represents a tab of pages.
type BookmarkTab struct {
	ExplicitTab bool

	Name  string
	Pages []*BookmarkPage
}

func (t *BookmarkTab) stringWithContext(first bool) string {
	var sb strings.Builder
	if !first || t.Name != "" || t.ExplicitTab {
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
			if p.Name != "" || (len(t.Pages) > 1 && p.Name != "") {
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

// MoveTab moves a tab from index i to j in the list.
func (b BookmarkList) MoveTab(i, j int) {
	if i < 0 || j < 0 || i >= len(b) || j >= len(b) || i == j {
		return
	}
	tab := b[i]
	if i < j {
		copy(b[i:j], b[i+1:j+1])
	} else {
		copy(b[j+1:i+1], b[j:i])
	}
	b[j] = tab
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
			t := &BookmarkTab{ExplicitTab: false}
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
			currentTab = &BookmarkTab{Name: rest, ExplicitTab: true}
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
			if currentPage != nil && currentPage.IsEmpty() && len(currentTab.Pages) == 1 && currentPage.Name == "" && rest != "" {
				currentPage.Name = rest
			} else {
				currentPage = &BookmarkPage{Name: rest, Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}
				currentTab.AddPage(currentPage)
			}
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
		t := &BookmarkTab{ExplicitTab: false}
		p := &BookmarkPage{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}
		t.AddPage(p)
		result.AddTab(t)
	}

	return result
}

// ValidateBookmarks parses the provided text and ensures it contains at least
// one tab of bookmarks.
func ValidateBookmarks(bookmarks string) (BookmarkList, error) {
	parsed := ParseBookmarks(bookmarks)
	if len(parsed) == 0 {
		return parsed, fmt.Errorf("no bookmarks found")
	}
	return parsed, nil
}

// StrictParseBookmarks parses the provided text strictly, rejecting malformed directives and unhandled text.
func StrictParseBookmarks(bookmarks string) (BookmarkList, error) {
	if strings.TrimSpace(bookmarks) == "" {
		return nil, fmt.Errorf("no bookmarks found")
	}
	lines := strings.Split(bookmarks, "\n")
	var result BookmarkList
	var currentTab *BookmarkTab
	var currentPage *BookmarkPage
	var currentCategory *BookmarkCategory
	idx := 0

	ensureTab := func() *BookmarkTab {
		if currentTab == nil {
			t := &BookmarkTab{ExplicitTab: false}
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

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue // empty lines are ignored by format definition
		}
		lower := strings.ToLower(trimmed)

		// Directives
		if lower == "tab" || strings.HasPrefix(lower, "tab ") || strings.HasPrefix(lower, "tab:") {
			rest := strings.TrimSpace(trimmed[len("tab"):])
			if strings.HasPrefix(rest, ":") {
				rest = strings.TrimSpace(rest[1:])
			}
			flushCategory()
			currentTab = &BookmarkTab{Name: rest, ExplicitTab: true}
			currentPage = &BookmarkPage{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}
			currentTab.AddPage(currentPage)
			result.AddTab(currentTab)
			continue
		}
		if lower == "page" || strings.HasPrefix(lower, "page ") || strings.HasPrefix(lower, "page:") {
			rest := strings.TrimSpace(trimmed[len("page"):])
			if strings.HasPrefix(rest, ":") {
				rest = strings.TrimSpace(rest[1:])
			}
			flushCategory()
			ensureTab()
			if currentPage != nil && currentPage.IsEmpty() && len(currentTab.Pages) == 1 && currentPage.Name == "" && rest != "" {
				currentPage.Name = rest
			} else {
				currentPage = &BookmarkPage{Name: rest, Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}
				currentTab.AddPage(currentPage)
			}
			continue
		}
		if trimmed == "--" {
			flushCategory()
			page := ensurePage()
			page.Blocks = append(page.Blocks, &BookmarkBlock{HR: true})
			page.Blocks = append(page.Blocks, &BookmarkBlock{Columns: []*BookmarkColumn{{}}})
			continue
		}
		if strings.EqualFold(trimmed, "column") {
			flushCategory()
			page := ensurePage()
			lastBlock := page.Blocks[len(page.Blocks)-1]
			lastBlock.Columns = append(lastBlock.Columns, &BookmarkColumn{})
			continue
		}

		parts := strings.Fields(trimmed)
		lowerFirst := strings.ToLower(parts[0])

		if strings.HasPrefix(lowerFirst, "category") {
			if lowerFirst != "category" && lowerFirst != "category:" {
				return nil, fmt.Errorf("line %d: malformed category directive: %q", i+1, trimmed)
			}
			// Safe trim avoiding panic if it's literally just "category"
			restIdx := len("category")
			if len(trimmed) > restIdx {
				if trimmed[restIdx] != ':' && trimmed[restIdx] != ' ' && trimmed[restIdx] != '\t' {
					return nil, fmt.Errorf("line %d: malformed category directive: %q", i+1, trimmed)
				}
			}
			rest := strings.TrimSpace(trimmed[restIdx:])
			if strings.HasPrefix(rest, ":") {
				rest = strings.TrimSpace(rest[1:])
			}
			if rest == "" {
				rest = "Category"
			}
			flushCategory()
			ensurePage()
			currentCategory = &BookmarkCategory{Name: rest}
			continue
		}

		// Not a directive. It must be a valid link.
		// A valid link must have a URL as parts[0] (starts with http, https, search:, etc.)
		// But let's check if it's a completely unhandled string. In normal ParseBookmarks,
		// if currentCategory == nil, it is silently ignored! In Strict mode, it's rejected.

		// Wait, what's a valid link? The app accepts any string as a URL essentially, but requires
		// a category to put it in. If a category hasn't been started, it's invalid unless it creates a default one?
		// Normal parser drops it silently. Let's reject it.
		if currentCategory == nil {
			return nil, fmt.Errorf("line %d: link outside of category or unrecognized directive: %q", i+1, trimmed)
		}

		// Ensure it's not a misspelled directive while inside a category.
		// E.g. `categor: foo`, `pagge: foo`, `colum:`
		if strings.HasPrefix(lowerFirst, "categor") || strings.HasPrefix(lowerFirst, "tab") || strings.HasPrefix(lowerFirst, "page") || strings.HasPrefix(lowerFirst, "pagge") || strings.HasPrefix(lowerFirst, "column") || strings.HasPrefix(lowerFirst, "colum") {
			// If it matches exactly one of the valid directives, it would have been caught above.
			// The only exception is if it has a typo.
			// Let's check common prefix typos. If it looks like a directive but isn't one, we reject it.
			if lowerFirst != "tab" && lowerFirst != "tab:" && lowerFirst != "page" && lowerFirst != "page:" && lowerFirst != "column" && lowerFirst != "column:" && lowerFirst != "category" && lowerFirst != "category:" {
				return nil, fmt.Errorf("line %d: misspelled or malformed directive inside category: %q", i+1, trimmed)
			}
		}

		// Since the permissive parser accepts any string inside a category as a valid URL,
		// we must not invent a narrower URL grammar here that rejects single-token entries or unlisted schemes.
		// Strict lint only catches strings outside of categories (handled above) or malformed directives.
		entry := BookmarkEntry{Url: parts[0], Name: parts[0]}
		if len(parts) > 1 {
			entry.Name = strings.Join(parts[1:], " ")
		}
		currentCategory.Entries = append(currentCategory.Entries, &entry)
	}

	flushCategory()

	if len(result) == 0 {
		return nil, fmt.Errorf("no bookmarks found")
	}

	return result, nil
}

// MoveCategory moves the category at fromIndex so it appears before toIndex.
// If toIndex equals the total number of categories, the item is moved to the end.
// When newColumn is true a new column directive is inserted before the moved category.
func (b BookmarkList) MoveCategory(fromIndex, toIndex int, newColumn bool, destPage *BookmarkPage, destCol int) error {
	type loc struct {
		block  *BookmarkBlock
		column *BookmarkColumn
		cat    *BookmarkCategory
		colIdx int
		catIdx int
	}
	var cats []loc
	idx := 0
	for _, t := range b {
		for _, p := range t.Pages {
			for _, b := range p.Blocks {
				for ci, col := range b.Columns {
					for cj, c := range col.Categories {
						cats = append(cats, loc{b, col, c, ci, cj})
						c.Index = idx
						idx++
					}
				}
			}
		}
	}

	if fromIndex < 0 || fromIndex >= len(cats) {
		return fmt.Errorf("category index %d not found", fromIndex)
	}
	var beforeLoc *loc
	if toIndex >= 0 && toIndex < len(cats) {
		beforeLoc = &cats[toIndex]
	}

	src := cats[fromIndex]
	// remove from source column
	src.column.Categories = append(src.column.Categories[:src.catIdx], src.column.Categories[src.catIdx+1:]...)
	if beforeLoc != nil && toIndex > fromIndex && src.column == beforeLoc.column {
		beforeLoc.catIdx--
	}

	var destColumn *BookmarkColumn
	if beforeLoc == nil { // append to end or specified column
		destBlock := cats[len(cats)-1].block
		destColObj := destBlock.Columns[len(destBlock.Columns)-1]
		destColIdx := len(destBlock.Columns) - 1
		if destPage != nil {
			destBlock = destPage.Blocks[len(destPage.Blocks)-1]
			if destCol >= len(destBlock.Columns) {
				destCol = len(destBlock.Columns) - 1
			}
			destColObj = destBlock.Columns[destCol]
			destColIdx = destCol
		}
		if newColumn {
			destColIdx++
			newCol := &BookmarkColumn{}
			destBlock.Columns = append(destBlock.Columns, nil)
			copy(destBlock.Columns[destColIdx+1:], destBlock.Columns[destColIdx:])
			destBlock.Columns[destColIdx] = newCol
			destColObj = newCol
		}
		destColObj.Categories = append(destColObj.Categories, src.cat)
		destColumn = destColObj
	} else {
		dest := *beforeLoc
		destBlock := dest.block
		destColObj := dest.column
		destColIdx := dest.colIdx
		insertIdx := dest.catIdx
		if newColumn {
			destColIdx++
			newCol := &BookmarkColumn{}
			destBlock.Columns = append(destBlock.Columns, nil)
			copy(destBlock.Columns[destColIdx+1:], destBlock.Columns[destColIdx:])
			destBlock.Columns[destColIdx] = newCol
			destColObj = newCol
			insertIdx = 0
		}
		destColObj.InsertCategory(insertIdx, src.cat)
		destColumn = destColObj
	}

	if len(src.column.Categories) == 0 && src.column != destColumn {
		// Find the current index of the source column, as it may have shifted
		colIdx := -1
		for i, col := range src.block.Columns {
			if col == src.column {
				colIdx = i
				break
			}
		}
		if colIdx != -1 {
			src.block.Columns = append(src.block.Columns[:colIdx], src.block.Columns[colIdx+1:]...)
		}
	}

	// reindex
	idx = 0
	for _, t := range b {
		for _, p := range t.Pages {
			for _, b := range p.Blocks {
				for _, col := range b.Columns {
					for _, c := range col.Categories {
						c.Index = idx
						idx++
					}
				}
			}
		}
	}
	return nil
}

// MoveCategoryBefore moves the category at fromIndex so it appears before beforeIndex.
func (b BookmarkList) MoveCategoryBefore(fromIndex, beforeIndex int) error {
	return b.MoveCategory(fromIndex, beforeIndex, false, nil, 0)
}

// MoveCategoryToEnd moves the category to the end of the specified column.
func (b BookmarkList) MoveCategoryToEnd(fromIndex int, page *BookmarkPage, colIdx int) error {
	return b.MoveCategory(fromIndex, -1, false, page, colIdx)
}

// MoveCategoryNewColumn moves the category into a new column inserted after
// the specified column index on the given page. When page is nil the category
// is moved to a new column on the last page. If destCol is negative the column
// is appended to the end of the page.
func (b BookmarkList) MoveCategoryNewColumn(fromIndex int, page *BookmarkPage, destCol int) error {
	if page == nil {
		return b.MoveCategory(fromIndex, -1, true, nil, destCol)
	}
	if destCol < 0 {
		last := page.Blocks[len(page.Blocks)-1]
		destCol = len(last.Columns) - 1
	}
	return b.MoveCategory(fromIndex, -1, true, page, destCol)
}

// PageForCategory returns the page containing the category with the given index.
func PageForCategory(tabs BookmarkList, index int) *BookmarkPage {
	idx := 0
	for _, t := range tabs {
		for _, p := range t.Pages {
			for _, b := range p.Blocks {
				for _, col := range b.Columns {
					for range col.Categories {
						if idx == index {
							return p
						}
						idx++
					}
				}
			}
		}
	}
	return nil
}

// FindPageBySha returns the page matching the sha.
func FindPageBySha(tabs BookmarkList, sha string) *BookmarkPage {
	for _, t := range tabs {
		for _, p := range t.Pages {
			if p.Sha() == sha {
				return p
			}
		}
	}
	return nil
}
