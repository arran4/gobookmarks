package gobookmarks

import (
	"encoding/json"
	"fmt"
	"strings"
)

func escapeForComment(val string) string {
	var sb strings.Builder
	for i := 0; i < len(val); i++ {
		if i+1 < len(val) && val[i] == '#' && val[i+1] == '#' {
			if i == 0 || val[i-1] == ' ' || val[i-1] == '\t' {
				sb.WriteString("\\##")
				i++
				continue
			}
		}
		sb.WriteByte(val[i])
	}
	return sb.String()
}

func parseLineForComments(line string) (bool, string, string, bool) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "##") {
		comment := strings.TrimPrefix(trimmed, "##")
		return true, "", comment, true
	}

	var unescaped strings.Builder
	var bsCount int
	var runStart int

	for i := 0; i < len(line); i++ {
		if line[i] == '\\' {
			if bsCount == 0 {
				runStart = i
			}
			bsCount++
			unescaped.WriteByte(line[i])
			continue
		}

		if i+1 < len(line) && line[i] == '#' && line[i+1] == '#' {
			isBoundary := runStart == 0 || (runStart > 0 && (line[runStart-1] == ' ' || line[runStart-1] == '\t'))
			if bsCount == 0 && (i == 0 || line[i-1] == ' ' || line[i-1] == '\t') {
				comment := line[i+2:]
				return false, unescaped.String(), comment, true
			} else if bsCount > 0 && isBoundary {
				str := unescaped.String()
				unescaped.Reset()
				unescaped.WriteString(str[:len(str)-1])
				unescaped.WriteString("##")
				i++
				bsCount = 0
				continue
			}
		}

		bsCount = 0
		unescaped.WriteByte(line[i])
	}

	return false, unescaped.String(), "", false
}

// BookmarkEntry represents a single link.
type BookmarkEntry struct {
	Url              string
	Name             string
	LeadingComments  []string
	InlineComment    string
	HasInlineComment bool
}

// String serializes the entry.
func (e *BookmarkEntry) String() string {
	if e == nil {
		return ""
	}
	var sb strings.Builder
	for _, c := range e.LeadingComments {
		sb.WriteString("##" + c + "\n")
	}
	sb.WriteString(escapeForComment(e.Url))
	if e.Name != "" && e.Name != e.Url {
		sb.WriteString(" " + escapeForComment(e.Name))
	}
	if e.HasInlineComment {
		sb.WriteString(" ##" + e.InlineComment)
	}
	sb.WriteString("\n")
	return sb.String()
}

// BookmarkCategory groups entries together.
type BookmarkCategory struct {
	Name             string
	Entries          []*BookmarkEntry
	Index            int
	LeadingComments  []string
	InlineComment    string
	HasInlineComment bool
}

// String serializes the category.
func (c *BookmarkCategory) String() string {
	var sb strings.Builder
	for _, comment := range c.LeadingComments {
		sb.WriteString("##" + comment + "\n")
	}
	sb.WriteString("Category: ")
	sb.WriteString(escapeForComment(c.Name))
	if c.HasInlineComment {
		sb.WriteString(" ##" + c.InlineComment)
	}
	sb.WriteString("\n")
	for _, e := range c.Entries {
		sb.WriteString(e.String())
	}
	return sb.String()
}

// BookmarkColumn contains a list of categories.
type BookmarkColumn struct {
	Name             string
	LeadingComments  []string
	InlineComment    string
	HasInlineComment bool
	Categories       []*BookmarkCategory
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
	Columns          []*BookmarkColumn
	HR               bool
	LeadingComments  []string
	InlineComment    string
	HasInlineComment bool
}

// String serializes the block.
func (b *BookmarkBlock) String() string {
	var sb strings.Builder
	for _, comment := range b.LeadingComments {
		sb.WriteString("##" + comment + "\n")
	}
	if b.HR {
		sb.WriteString("--")
		if b.HasInlineComment {
			sb.WriteString(" ##" + b.InlineComment)
		}
		sb.WriteString("\n")
		return sb.String()
	}
	for i, col := range b.Columns {
		for _, comment := range col.LeadingComments {
			sb.WriteString("##" + comment + "\n")
		}
		if i > 0 || strings.TrimSpace(col.Name) != "" || len(col.LeadingComments) > 0 || col.HasInlineComment {
			if strings.TrimSpace(col.Name) != "" {
				sb.WriteString("Column: ")
				sb.WriteString(escapeForComment(col.Name))
			} else {
				sb.WriteString("Column")
			}
			if col.HasInlineComment {
				sb.WriteString(" ##" + col.InlineComment)
			}
			sb.WriteString("\n")
		}
		sb.WriteString(col.String())
	}
	return sb.String()
}

// Interchange formats for conversion
type JSONEntry struct {
	URL              string   `json:"url"`
	Name             string   `json:"name"`
	LeadingComments  []string `json:"leading_comments,omitempty"`
	InlineComment    string   `json:"inline_comment,omitempty"`
	HasInlineComment bool     `json:"has_inline_comment,omitempty"`
}

type JSONCategory struct {
	Name             string       `json:"name"`
	LeadingComments  []string     `json:"leading_comments,omitempty"`
	InlineComment    string       `json:"inline_comment,omitempty"`
	HasInlineComment bool         `json:"has_inline_comment,omitempty"`
	Entries          []*JSONEntry `json:"entries,omitempty"`
}

type JSONColumn struct {
	Name             string          `json:"name,omitempty"`
	LeadingComments  []string        `json:"leading_comments,omitempty"`
	InlineComment    string          `json:"inline_comment,omitempty"`
	HasInlineComment bool            `json:"has_inline_comment,omitempty"`
	Categories       []*JSONCategory `json:"categories,omitempty"`
}

type JSONBlock struct {
	HR               bool          `json:"hr"`
	LeadingComments  []string      `json:"leading_comments,omitempty"`
	InlineComment    string        `json:"inline_comment,omitempty"`
	HasInlineComment bool          `json:"has_inline_comment,omitempty"`
	Columns          []*JSONColumn `json:"columns,omitempty"`
}

type JSONPage struct {
	Name             string       `json:"name,omitempty"`
	LeadingComments  []string     `json:"leading_comments,omitempty"`
	InlineComment    string       `json:"inline_comment,omitempty"`
	HasInlineComment bool         `json:"has_inline_comment,omitempty"`
	Blocks           []*JSONBlock `json:"blocks,omitempty"`
}

type JSONTab struct {
	Name             string      `json:"name,omitempty"`
	ExplicitTab      bool        `json:"explicit,omitempty"`
	LeadingComments  []string    `json:"leading_comments,omitempty"`
	InlineComment    string      `json:"inline_comment,omitempty"`
	HasInlineComment bool        `json:"has_inline_comment,omitempty"`
	TrailingComments []string    `json:"trailing_comments,omitempty"`
	Pages            []*JSONPage `json:"pages,omitempty"`
}

// ToJSON transforms a BookmarkList into a stable JSON struct format.
func (b BookmarkList) ToJSON() []*JSONTab {
	var tabs []*JSONTab
	for _, t := range b {
		jTab := &JSONTab{
			Name:             t.Name,
			ExplicitTab:      t.ExplicitTab,
			LeadingComments:  t.LeadingComments,
			InlineComment:    t.InlineComment,
			HasInlineComment: t.HasInlineComment,
			TrailingComments: t.TrailingComments,
		}
		for _, p := range t.Pages {
			jPage := &JSONPage{
				Name:             p.Name,
				LeadingComments:  p.LeadingComments,
				InlineComment:    p.InlineComment,
				HasInlineComment: p.HasInlineComment,
			}
			for _, blk := range p.Blocks {
				jBlk := &JSONBlock{
					HR:               blk.HR,
					LeadingComments:  blk.LeadingComments,
					InlineComment:    blk.InlineComment,
					HasInlineComment: blk.HasInlineComment,
				}
				for _, col := range blk.Columns {
					jCol := &JSONColumn{
						Name:             col.Name,
						LeadingComments:  col.LeadingComments,
						InlineComment:    col.InlineComment,
						HasInlineComment: col.HasInlineComment,
					}
					for _, cat := range col.Categories {
						jCat := &JSONCategory{
							Name:             cat.Name,
							LeadingComments:  cat.LeadingComments,
							InlineComment:    cat.InlineComment,
							HasInlineComment: cat.HasInlineComment,
						}
						for _, ent := range cat.Entries {
							jCat.Entries = append(jCat.Entries, &JSONEntry{
								URL:              ent.Url,
								Name:             ent.Name,
								LeadingComments:  ent.LeadingComments,
								InlineComment:    ent.InlineComment,
								HasInlineComment: ent.HasInlineComment,
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
			Name:             t.Name,
			ExplicitTab:      t.ExplicitTab,
			LeadingComments:  t.LeadingComments,
			InlineComment:    t.InlineComment,
			HasInlineComment: t.HasInlineComment,
			TrailingComments: t.TrailingComments,
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
				Name:             p.Name,
				LeadingComments:  p.LeadingComments,
				InlineComment:    p.InlineComment,
				HasInlineComment: p.HasInlineComment,
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
					hasAnyComments := len(t.LeadingComments) > 0 || len(t.TrailingComments) > 0 || t.HasInlineComment ||
						len(p.LeadingComments) > 0 || p.HasInlineComment ||
						len(p.Blocks[0].LeadingComments) > 0 || p.Blocks[0].HasInlineComment
					if !hasAnyComments {
						for _, col := range p.Blocks[0].Columns {
							if col != nil && (len(col.LeadingComments) > 0 || col.HasInlineComment) {
								hasAnyComments = true
								break
							}
						}
					}
					if !hasAnyComments {
						return nil, fmt.Errorf("invalid json: degenerate completely empty implicit model cannot be serialized natively")
					}
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
					HR:               blk.HR,
					LeadingComments:  blk.LeadingComments,
					InlineComment:    blk.InlineComment,
					HasInlineComment: blk.HasInlineComment,
				}
				if !blk.HR && len(blk.Columns) == 0 {
					return nil, fmt.Errorf("invalid json: block without HR must have at least one column")
				}
				for _, col := range blk.Columns {
					if col == nil {
						return nil, fmt.Errorf("invalid json: null column object")
					}
					bc := &BookmarkColumn{
						Name:             col.Name,
						LeadingComments:  col.LeadingComments,
						InlineComment:    col.InlineComment,
						HasInlineComment: col.HasInlineComment,
					}
					if strings.TrimSpace(col.Name) != "" && strings.TrimSpace(col.Name) != col.Name {
						return nil, fmt.Errorf("invalid json: explicitly white space name cannot be represented losslessly (lossy shape)")
					}
					for _, cat := range col.Categories {
						if cat == nil {
							return nil, fmt.Errorf("invalid json: null category object")
						}
						// Do not normalize by trimming; use actual values and let the general representability backstop catch lossy whitespace modifications.
						if cat.Name == "" || strings.TrimSpace(cat.Name) == "" {
							return nil, fmt.Errorf("invalid json: category name cannot be explicitly empty or only whitespace (lossy shape)")
						}
						bcat := &BookmarkCategory{
							Name:             cat.Name,
							LeadingComments:  cat.LeadingComments,
							InlineComment:    cat.InlineComment,
							HasInlineComment: cat.HasInlineComment,
						}
						for _, ent := range cat.Entries {
							if ent == nil {
								return nil, fmt.Errorf("invalid json: null entry object in category %q", cat.Name)
							}
							if ent.URL == "" || strings.TrimSpace(ent.URL) == "" {
								return nil, fmt.Errorf("invalid json: entry url cannot be explicitly empty or only whitespace (lossy shape)")
							}
							bcat.Entries = append(bcat.Entries, &BookmarkEntry{
								Url:              ent.URL,
								Name:             ent.Name,
								LeadingComments:  ent.LeadingComments,
								InlineComment:    ent.InlineComment,
								HasInlineComment: ent.HasInlineComment,
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
	Blocks           []*BookmarkBlock
	Name             string
	LeadingComments  []string
	InlineComment    string
	HasInlineComment bool
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

	Name             string
	LeadingComments  []string
	InlineComment    string
	HasInlineComment bool
	TrailingComments []string
	Pages            []*BookmarkPage
}

func (t *BookmarkTab) stringWithContext(first bool) string {
	var sb strings.Builder
	for _, comment := range t.LeadingComments {
		sb.WriteString("##" + comment + "\n")
	}
	if !first || t.Name != "" || t.ExplicitTab {
		if t.Name != "" {
			sb.WriteString("Tab: ")
			sb.WriteString(escapeForComment(t.Name))
		} else {
			sb.WriteString("Tab")
		}
		if t.HasInlineComment {
			sb.WriteString(" ##" + t.InlineComment)
		}
		sb.WriteString("\n")
	}
	for i, p := range t.Pages {
		for _, comment := range p.LeadingComments {
			sb.WriteString("##" + comment + "\n")
		}
		if i == 0 {
			if p.Name != "" || (len(t.Pages) > 1 && p.Name != "") {
				if p.Name != "" {
					sb.WriteString("Page: ")
					sb.WriteString(escapeForComment(p.Name))
				} else {
					sb.WriteString("Page")
				}
				if p.HasInlineComment {
					sb.WriteString(" ##" + p.InlineComment)
				}
				sb.WriteString("\n")
			}
		} else {
			if p.Name != "" {
				sb.WriteString("Page: ")
				sb.WriteString(escapeForComment(p.Name))
			} else {
				sb.WriteString("Page")
			}
			if p.HasInlineComment {
				sb.WriteString(" ##" + p.InlineComment)
			}
			sb.WriteString("\n")
		}
		sb.WriteString(p.String())
	}
	for _, comment := range t.TrailingComments {
		sb.WriteString("##" + comment + "\n")
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
	var firstColClaimed bool
	idx := 0

	var pendingComments []string
	var comment string
	_ = comment
	var hasComment bool
	_ = hasComment
	ensureTab := func() *BookmarkTab {
		if currentTab == nil {
			t := &BookmarkTab{ExplicitTab: false}
			if len(pendingComments) > 0 {
				t.LeadingComments = pendingComments
				pendingComments = nil
			}
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
			firstColClaimed = false
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

	for _, rawLine := range lines {
		if len(strings.TrimSpace(rawLine)) == 0 {
			continue
		}
		var isStandalone bool
		var line string
		var comment string
		var hasComment bool
		isStandalone, line, comment, hasComment = parseLineForComments(rawLine)
		if isStandalone {
			pendingComments = append(pendingComments, comment)
			continue
		}

		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		lower := strings.ToLower(line)
		if lower == "tab" || strings.HasPrefix(lower, "tab ") || strings.HasPrefix(lower, "tab:") {
			rest := strings.TrimSpace(line[len("tab"):])
			if strings.HasPrefix(rest, ":") {
				rest = strings.TrimSpace(rest[1:])
			}
			flushCategory()
			currentTab = &BookmarkTab{Name: rest, ExplicitTab: true, LeadingComments: pendingComments, InlineComment: comment, HasInlineComment: hasComment}
			pendingComments = nil
			currentPage = &BookmarkPage{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}
			if len(pendingComments) > 0 {
				currentPage.LeadingComments = pendingComments
				pendingComments = nil
			}
			currentTab.AddPage(currentPage)
			result.AddTab(currentTab)
			firstColClaimed = false
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
				if len(pendingComments) > 0 {
					currentPage.LeadingComments = append(currentPage.LeadingComments, pendingComments...)
					pendingComments = nil
				}
				if hasComment {
					currentPage.InlineComment = comment
					currentPage.HasInlineComment = true
				}
			} else {
				currentPage = &BookmarkPage{Name: rest, Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}, LeadingComments: pendingComments, InlineComment: comment, HasInlineComment: hasComment}
				pendingComments = nil
				currentTab.AddPage(currentPage)
			}
			firstColClaimed = false
			continue
		}
		if line == "--" {
			flushCategory()
			page := ensurePage()
			page.Blocks = append(page.Blocks, &BookmarkBlock{HR: true, LeadingComments: pendingComments, InlineComment: comment, HasInlineComment: hasComment})
			pendingComments = nil
			page.Blocks = append(page.Blocks, &BookmarkBlock{Columns: []*BookmarkColumn{{}}})
			firstColClaimed = false
			continue
		}

		isColonForm := false
		columnName := ""
		if strings.HasPrefix(lower, "column:") {
			isColonForm = true
			columnName = strings.TrimSpace(line[7:])
		} else if strings.HasPrefix(lower, "column ") {
			isColonForm = true
			columnName = strings.TrimSpace(line[7:])
			if strings.HasPrefix(columnName, ":") {
				columnName = strings.TrimSpace(columnName[1:])
			} else {
				isColonForm = false
				columnName = ""
			}
		}

		if isColonForm || strings.EqualFold(line, "column") {
			flushCategory()
			page := ensurePage()
			lastBlock := page.Blocks[len(page.Blocks)-1]

			if isColonForm {
				if len(lastBlock.Columns) == 1 && len(lastBlock.Columns[0].Categories) == 0 && !firstColClaimed {
					// Claim the implicit first column
					lastBlock.Columns[0].Name = columnName
					if len(pendingComments) > 0 {
						lastBlock.Columns[0].LeadingComments = append(lastBlock.Columns[0].LeadingComments, pendingComments...)
						pendingComments = nil
					}
					if hasComment {
						lastBlock.Columns[0].InlineComment = comment
						lastBlock.Columns[0].HasInlineComment = true
					}
					firstColClaimed = true
				} else {
					lastBlock.Columns = append(lastBlock.Columns, &BookmarkColumn{Name: columnName, LeadingComments: pendingComments, InlineComment: comment, HasInlineComment: hasComment})
					pendingComments = nil
				}
			} else {
				// Bare "Column" retains existing separator semantics and does not claim the first column
				lastBlock.Columns = append(lastBlock.Columns, &BookmarkColumn{LeadingComments: pendingComments, InlineComment: comment, HasInlineComment: hasComment})
				pendingComments = nil
			}
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
			currentCategory = &BookmarkCategory{Name: rest, LeadingComments: pendingComments, InlineComment: comment, HasInlineComment: hasComment}
			pendingComments = nil
		} else if currentCategory != nil {
			entry := BookmarkEntry{Url: parts[0], Name: parts[0], LeadingComments: pendingComments, InlineComment: comment, HasInlineComment: hasComment}
			pendingComments = nil
			if len(parts) > 1 {
				entry.Name = strings.Join(parts[1:], " ")
			}
			currentCategory.Entries = append(currentCategory.Entries, &entry)
		}
	}

	flushCategory()

	if len(result) == 0 {
		t := &BookmarkTab{ExplicitTab: false}
		if len(pendingComments) > 0 {
			t.LeadingComments = pendingComments
			pendingComments = nil
		}
		p := &BookmarkPage{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}
		t.AddPage(p)
		result.AddTab(t)
	} else if len(pendingComments) > 0 {
		lastTab := result[len(result)-1]
		lastTab.TrailingComments = pendingComments
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
	var firstColClaimed bool
	idx := 0

	var pendingComments []string
	var comment string
	_ = comment
	var hasComment bool
	_ = hasComment
	ensureTab := func() *BookmarkTab {
		if currentTab == nil {
			t := &BookmarkTab{ExplicitTab: false}
			if len(pendingComments) > 0 {
				t.LeadingComments = pendingComments
				pendingComments = nil
			}
			if len(pendingComments) > 0 {
				t.LeadingComments = pendingComments
				pendingComments = nil
			}
			result.AddTab(t)
			currentTab = t
		}
		return currentTab
	}

	ensurePage := func() *BookmarkPage {
		ensureTab()
		if currentPage == nil {
			p := &BookmarkPage{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}
			if len(pendingComments) > 0 {
				p.LeadingComments = pendingComments
				pendingComments = nil
			}
			if len(pendingComments) > 0 {
				p.LeadingComments = pendingComments
				pendingComments = nil
			}
			currentTab.AddPage(p)
			currentPage = p
			firstColClaimed = false
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

	for i, rawLine := range lines {
		if len(strings.TrimSpace(rawLine)) == 0 {
			continue
		}
		var isStandalone bool
		var line string
		var hasComment bool
		isStandalone, line, comment, hasComment = parseLineForComments(rawLine)
		if isStandalone {
			pendingComments = append(pendingComments, comment)
			continue
		}

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
			currentTab = &BookmarkTab{Name: rest, ExplicitTab: true, LeadingComments: pendingComments, InlineComment: comment, HasInlineComment: hasComment}
			pendingComments = nil
			currentPage = &BookmarkPage{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}
			if len(pendingComments) > 0 {
				currentPage.LeadingComments = pendingComments
				pendingComments = nil
			}
			if len(pendingComments) > 0 {
				currentPage.LeadingComments = pendingComments
				pendingComments = nil
			}
			currentTab.AddPage(currentPage)
			result.AddTab(currentTab)
			firstColClaimed = false
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
				if len(pendingComments) > 0 {
					currentPage.LeadingComments = append(currentPage.LeadingComments, pendingComments...)
					pendingComments = nil
				}
				if hasComment {
					currentPage.InlineComment = comment
					currentPage.HasInlineComment = true
				}
				if len(pendingComments) > 0 {
					currentPage.LeadingComments = append(currentPage.LeadingComments, pendingComments...)
					pendingComments = nil
				}
				if hasComment {
					currentPage.InlineComment = comment
					currentPage.HasInlineComment = true
				}
			} else {
				currentPage = &BookmarkPage{Name: rest, Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}, LeadingComments: pendingComments, InlineComment: comment, HasInlineComment: hasComment}
				pendingComments = nil
				currentTab.AddPage(currentPage)
			}
			firstColClaimed = false
			continue
		}
		if trimmed == "--" {
			flushCategory()
			page := ensurePage()
			page.Blocks = append(page.Blocks, &BookmarkBlock{HR: true, LeadingComments: pendingComments, InlineComment: comment, HasInlineComment: hasComment})
			pendingComments = nil
			page.Blocks = append(page.Blocks, &BookmarkBlock{Columns: []*BookmarkColumn{{}}})
			firstColClaimed = false
			continue
		}

		isColonForm := false
		columnName := ""
		trimmedColName := strings.TrimSpace(lower)
		if strings.HasPrefix(lower, "column:") {
			isColonForm = true
			columnName = strings.TrimSpace(trimmed[7:])
		} else if strings.HasPrefix(trimmedColName, "column") && len(trimmedColName) > 6 && strings.HasPrefix(strings.TrimSpace(trimmedColName[6:]), ":") {
			return nil, fmt.Errorf("line %d: expected category, tab, page, or separator, got %s", i+1, trimmed)
		} else if strings.HasPrefix(lower, "column ") {
			isColonForm = true
			columnName = strings.TrimSpace(trimmed[7:])
			if strings.HasPrefix(columnName, ":") {
				columnName = strings.TrimSpace(columnName[1:])
			} else {
				isColonForm = false
				columnName = ""
			}
		}

		if isColonForm || strings.EqualFold(trimmed, "column") {
			flushCategory()
			page := ensurePage()
			lastBlock := page.Blocks[len(page.Blocks)-1]

			if isColonForm {
				if len(lastBlock.Columns) == 1 && len(lastBlock.Columns[0].Categories) == 0 && !firstColClaimed {
					// Claim the implicit first column
					lastBlock.Columns[0].Name = columnName
					if len(pendingComments) > 0 {
						lastBlock.Columns[0].LeadingComments = append(lastBlock.Columns[0].LeadingComments, pendingComments...)
						pendingComments = nil
					}
					if hasComment {
						lastBlock.Columns[0].InlineComment = comment
						lastBlock.Columns[0].HasInlineComment = true
					}
					if len(pendingComments) > 0 {
						lastBlock.Columns[0].LeadingComments = append(lastBlock.Columns[0].LeadingComments, pendingComments...)
						pendingComments = nil
					}
					if hasComment {
						lastBlock.Columns[0].InlineComment = comment
						lastBlock.Columns[0].HasInlineComment = true
					}
					firstColClaimed = true
				} else {
					lastBlock.Columns = append(lastBlock.Columns, &BookmarkColumn{Name: columnName, LeadingComments: pendingComments, InlineComment: comment, HasInlineComment: hasComment})
					pendingComments = nil
				}
			} else {
				// Bare "Column" retains existing separator semantics and does not claim the first column
				lastBlock.Columns = append(lastBlock.Columns, &BookmarkColumn{LeadingComments: pendingComments, InlineComment: comment, HasInlineComment: hasComment})
				pendingComments = nil
			}
			continue
		}

		parts := strings.Fields(trimmed)
		lowerFirst := strings.ToLower(parts[0])

		if strings.HasPrefix(lowerFirst, "category") {
			restIdx := len("category")
			rest := strings.TrimSpace(trimmed[restIdx:])
			if strings.HasPrefix(rest, ":") {
				rest = strings.TrimSpace(rest[1:])
			}
			if rest == "" {
				rest = "Category"
			}
			flushCategory()
			ensurePage()
			currentCategory = &BookmarkCategory{Name: rest, LeadingComments: pendingComments, InlineComment: comment, HasInlineComment: hasComment}
			pendingComments = nil
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
		// E.g. `categor: foo`, `pagge: foo`, `colum:`. We look for words that start with directive
		// prefixes and contain a colon (or literally "colum:"). If there's no colon and it's just a word like "page.example.com"
		// or "tabby", it shouldn't be rejected, as the native parser treats it as an entry URL.
		if strings.HasSuffix(lowerFirst, ":") {
			if strings.HasPrefix(lowerFirst, "categor") || strings.HasPrefix(lowerFirst, "tab") || strings.HasPrefix(lowerFirst, "page") || strings.HasPrefix(lowerFirst, "pagge") || strings.HasPrefix(lowerFirst, "column") || strings.HasPrefix(lowerFirst, "colum") {
				// We already processed exact valid directives. If it reached here, it's a typo.
				if lowerFirst != "tab:" && lowerFirst != "page:" && lowerFirst != "column:" && lowerFirst != "category:" {
					return nil, fmt.Errorf("line %d: misspelled or malformed directive inside category: %q", i+1, trimmed)
				}
			}
		}

		// Since the permissive parser accepts any string inside a category as a valid URL,
		// we must not invent a narrower URL grammar here that rejects single-token entries or unlisted schemes.
		// Strict lint only catches strings outside of categories (handled above) or malformed directives.
		entry := BookmarkEntry{Url: parts[0], Name: parts[0], LeadingComments: pendingComments, InlineComment: comment, HasInlineComment: hasComment}
		pendingComments = nil
		if len(parts) > 1 {
			entry.Name = strings.Join(parts[1:], " ")
		}
		currentCategory.Entries = append(currentCategory.Entries, &entry)
	}

	flushCategory()

	if len(result) == 0 && len(pendingComments) > 0 {
		t := &BookmarkTab{ExplicitTab: false, LeadingComments: pendingComments}
		p := &BookmarkPage{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}
		t.AddPage(p)
		result.AddTab(t)
		return result, nil
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no bookmarks found")
	}

	if len(pendingComments) > 0 {
		lastTab := result[len(result)-1]
		lastTab.TrailingComments = pendingComments
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

	if len(src.column.Categories) == 0 && src.column != destColumn && strings.TrimSpace(src.column.Name) == "" {
		if len(src.block.Columns) > 1 {
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
