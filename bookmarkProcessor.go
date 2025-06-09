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
}

func PreprocessBookmarks(bookmarks string) []*BookmarkPage {
	lines := strings.Split(bookmarks, "\n")
	var result = []*BookmarkPage{{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}}
	var currentCategory *BookmarkCategory
	currentTab := ""
	idx := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "tab:") {
			if currentCategory != nil {
				currentCategory.Index = idx
				idx++
				lastBlock := result[len(result)-1].Blocks[len(result[len(result)-1].Blocks)-1]
				lastColumn := lastBlock.Columns[len(lastBlock.Columns)-1]
				lastColumn.Categories = append(lastColumn.Categories, currentCategory)
				currentCategory = nil
			}
			currentTab = strings.TrimSpace(line[4:])
			result = append(result, &BookmarkPage{Tab: currentTab, Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}})
			continue
		}
		if strings.EqualFold(line, "Page") {
			if currentCategory != nil {
				currentCategory.Index = idx
				idx++
				lastBlock := result[len(result)-1].Blocks[len(result[len(result)-1].Blocks)-1]
				lastColumn := lastBlock.Columns[len(lastBlock.Columns)-1]
				lastColumn.Categories = append(lastColumn.Categories, currentCategory)
				currentCategory = nil
			}
			result = append(result, &BookmarkPage{Tab: currentTab, Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}})
			continue
		}
		if line == "--" {
			if currentCategory != nil {
				currentCategory.Index = idx
				idx++
				lastBlock := result[len(result)-1].Blocks[len(result[len(result)-1].Blocks)-1]
				lastColumn := lastBlock.Columns[len(lastBlock.Columns)-1]
				lastColumn.Categories = append(lastColumn.Categories, currentCategory)
				currentCategory = nil
			}
			// add hr block then start a new column block
			result[len(result)-1].Blocks = append(result[len(result)-1].Blocks, &BookmarkBlock{HR: true})
			result[len(result)-1].Blocks = append(result[len(result)-1].Blocks, &BookmarkBlock{Columns: []*BookmarkColumn{{}}})
			continue
		}
		if strings.EqualFold(line, "column") {
			if currentCategory != nil {
				currentCategory.Index = idx
				idx++
				lastBlock := result[len(result)-1].Blocks[len(result[len(result)-1].Blocks)-1]
				lastColumn := lastBlock.Columns[len(lastBlock.Columns)-1]
				lastColumn.Categories = append(lastColumn.Categories, currentCategory)
				currentCategory = nil
			}
			lastBlock := result[len(result)-1].Blocks[len(result[len(result)-1].Blocks)-1]
			lastBlock.Columns = append(lastBlock.Columns, &BookmarkColumn{})
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		if len(parts) > 0 && strings.EqualFold(parts[0], "Category:") {
			categoryName := strings.Join(parts[1:], " ")
			if currentCategory == nil {
				currentCategory = &BookmarkCategory{Name: categoryName}
			} else if currentCategory.Name != "" {
				currentCategory.Index = idx
				idx++
				lastBlock := result[len(result)-1].Blocks[len(result[len(result)-1].Blocks)-1]
				lastColumn := lastBlock.Columns[len(lastBlock.Columns)-1]
				lastColumn.Categories = append(lastColumn.Categories, currentCategory)
				currentCategory = &BookmarkCategory{Name: categoryName}
			} else {
				currentCategory.Name = categoryName
			}
		} else if len(parts) > 0 && currentCategory != nil {
			var entry BookmarkEntry
			entry.Url = parts[0]
			entry.Name = parts[0]
			if len(parts) > 1 {
				entry.Name = strings.Join(parts[1:], " ")
			}
			currentCategory.Entries = append(currentCategory.Entries, &entry)
		}
	}

	if currentCategory != nil && currentCategory.Name != "" {
		currentCategory.Index = idx
		idx++
		lastBlock := result[len(result)-1].Blocks[len(result[len(result)-1].Blocks)-1]
		lastColumn := lastBlock.Columns[len(lastBlock.Columns)-1]
		lastColumn.Categories = append(lastColumn.Categories, currentCategory)
	}

	return result
}
