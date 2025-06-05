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
}

func PreprocessBookmarks(bookmarks string) []*BookmarkPage {
	lines := strings.Split(bookmarks, "\n")
	var result = []*BookmarkPage{{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}}
	var currentCategory *BookmarkCategory

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.EqualFold(line, "Page") {
			if currentCategory != nil {
				lastBlock := result[len(result)-1].Blocks[len(result[len(result)-1].Blocks)-1]
				lastColumn := lastBlock.Columns[len(lastBlock.Columns)-1]
				lastColumn.Categories = append(lastColumn.Categories, currentCategory)
				currentCategory = nil
			}
			result = append(result, &BookmarkPage{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}})
			continue
		}
		if line == "--" {
			if currentCategory != nil {
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
		lastBlock := result[len(result)-1].Blocks[len(result[len(result)-1].Blocks)-1]
		lastColumn := lastBlock.Columns[len(lastBlock.Columns)-1]
		lastColumn.Categories = append(lastColumn.Categories, currentCategory)
	}

	return result
}
