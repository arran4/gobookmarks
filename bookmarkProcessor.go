package gobookmarks

import (
	"strings"
)

type BookmarkEntry struct {
	Url  string
	Name string
}

type BookmarkCategory struct {
	Name    string
	Entries []*BookmarkEntry
	Index   int
}

type BookmarkColumn struct {
	Categories []*BookmarkCategory
}

type BookmarkBlock struct {
	Columns []*BookmarkColumn
	HR      bool
}

type BookmarkPage struct {
	Blocks []*BookmarkBlock
	Tab    string
	Name   string
}

func PreprocessBookmarks(bookmarks string) []*BookmarkPage {
	lines := strings.Split(bookmarks, "\n")
	var result []*BookmarkPage
	var currentCategory *BookmarkCategory
	currentTab := ""
	idx := 0

	ensurePage := func() *BookmarkPage {
		if len(result) == 0 {
			p := &BookmarkPage{Tab: currentTab, Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}
			result = append(result, p)
			return p
		}
		return result[len(result)-1]
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		if lower == "tab" || strings.HasPrefix(lower, "tab ") || strings.HasPrefix(lower, "tab:") {
			rest := strings.TrimSpace(line[len("tab"):])
			if strings.HasPrefix(rest, ":") {
				rest = strings.TrimSpace(rest[1:])
			}
			if currentCategory != nil {
				currentCategory.Index = idx
				idx++
				lastPage := ensurePage()
				lastBlock := lastPage.Blocks[len(lastPage.Blocks)-1]
				lastColumn := lastBlock.Columns[len(lastBlock.Columns)-1]
				lastColumn.Categories = append(lastColumn.Categories, currentCategory)
				currentCategory = nil
			}
			currentTab = rest
			result = append(result, &BookmarkPage{Tab: currentTab, Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}})
			continue
		}
		if lower == "page" || strings.HasPrefix(lower, "page ") || strings.HasPrefix(lower, "page:") {
			rest := strings.TrimSpace(line[len("page"):])
			if strings.HasPrefix(rest, ":") {
				rest = strings.TrimSpace(rest[1:])
			}
			if currentCategory != nil {
				currentCategory.Index = idx
				idx++
				lastPage := ensurePage()
				lastBlock := lastPage.Blocks[len(lastPage.Blocks)-1]
				lastColumn := lastBlock.Columns[len(lastBlock.Columns)-1]
				lastColumn.Categories = append(lastColumn.Categories, currentCategory)
				currentCategory = nil
			}
			result = append(result, &BookmarkPage{Tab: currentTab, Name: rest, Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}})
			continue
		}
		if line == "--" {
			if currentCategory != nil {
				currentCategory.Index = idx
				idx++
				lastPage := ensurePage()
				lastBlock := lastPage.Blocks[len(lastPage.Blocks)-1]
				lastColumn := lastBlock.Columns[len(lastBlock.Columns)-1]
				lastColumn.Categories = append(lastColumn.Categories, currentCategory)
				currentCategory = nil
			}
			lastPage := ensurePage()
			lastPage.Blocks = append(lastPage.Blocks, &BookmarkBlock{HR: true})
			lastPage.Blocks = append(lastPage.Blocks, &BookmarkBlock{Columns: []*BookmarkColumn{{}}})
			continue
		}
		if strings.EqualFold(line, "column") {
			if currentCategory != nil {
				currentCategory.Index = idx
				idx++
				lastPage := ensurePage()
				lastBlock := lastPage.Blocks[len(lastPage.Blocks)-1]
				lastColumn := lastBlock.Columns[len(lastBlock.Columns)-1]
				lastColumn.Categories = append(lastColumn.Categories, currentCategory)
				currentCategory = nil
			}
			lastPage := ensurePage()
			lastBlock := lastPage.Blocks[len(lastPage.Blocks)-1]
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
			if currentCategory != nil {
				currentCategory.Index = idx
				idx++
				lastPage := ensurePage()
				lastBlock := lastPage.Blocks[len(lastPage.Blocks)-1]
				lastColumn := lastBlock.Columns[len(lastBlock.Columns)-1]
				lastColumn.Categories = append(lastColumn.Categories, currentCategory)
			}
			ensurePage()
			currentCategory = &BookmarkCategory{Name: rest}
		} else if currentCategory != nil {
			var entry BookmarkEntry
			entry.Url = parts[0]
			entry.Name = parts[0]
			if len(parts) > 1 {
				entry.Name = strings.Join(parts[1:], " ")
			}
			currentCategory.Entries = append(currentCategory.Entries, &entry)
		}
	}

	if currentCategory != nil {
		currentCategory.Index = idx
		idx++
		lastPage := ensurePage()
		lastBlock := lastPage.Blocks[len(lastPage.Blocks)-1]
		lastColumn := lastBlock.Columns[len(lastBlock.Columns)-1]
		lastColumn.Categories = append(lastColumn.Categories, currentCategory)
	}

	if len(result) == 0 {
		// create an empty page if no directives produced one
		result = append(result, &BookmarkPage{Tab: currentTab, Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}})
	}

	return result
}
