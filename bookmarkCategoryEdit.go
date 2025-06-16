package gobookmarks

import (
	"fmt"
	"strings"
)

// ExtractCategoryByIndex returns the category text for the nth category (0 based)
func ExtractCategoryByIndex(bookmarks string, index int) (string, error) {
	lines := strings.Split(bookmarks, "\n")
	currentIndex := -1
	start := -1
	end := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), "category:") {
			currentIndex++
			if currentIndex == index {
				start = i
				for j := i + 1; j <= len(lines); j++ {
					if j == len(lines) {
						end = j
						break
					}
					t := strings.TrimSpace(lines[j])
					lower := strings.ToLower(t)
					if strings.HasPrefix(lower, "category:") || strings.EqualFold(lower, "column") || strings.HasPrefix(lower, "page") || t == "--" {
						end = j
						break
					}
				}
				break
			}
		}
	}
	if start == -1 || end == -1 {
		return "", fmt.Errorf("category index %d not found", index)
	}
	return strings.Join(lines[start:end], "\n"), nil
}

// ReplaceCategoryByIndex replaces the nth category with newText
func ReplaceCategoryByIndex(bookmarks string, index int, newText string) (string, error) {
	lines := strings.Split(bookmarks, "\n")
	currentIndex := -1
	start := -1
	end := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), "category:") {
			currentIndex++
			if currentIndex == index {
				start = i
				for j := i + 1; j <= len(lines); j++ {
					if j == len(lines) {
						end = j
						break
					}
					t := strings.TrimSpace(lines[j])
					lower := strings.ToLower(t)
					if strings.HasPrefix(lower, "category:") || strings.EqualFold(lower, "column") || strings.HasPrefix(lower, "page") || t == "--" {
						end = j
						break
					}
				}
				break
			}
		}
	}
	if start == -1 || end == -1 {
		return "", fmt.Errorf("category index %d not found", index)
	}
	var result []string
	result = append(result, lines[:start]...)
	newLines := strings.Split(newText, "\n")
	result = append(result, newLines...)
	result = append(result, lines[end:]...)
	return strings.Join(result, "\n"), nil
}
