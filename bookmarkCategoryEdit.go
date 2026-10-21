package gobookmarks

import (
	"fmt"
	"strings"
)

// getCategoryBounds locates the lines belonging to a given category index.
// It returns startIdx (the first line of the category, including its leading comments),
// endIdx (the index of the first line following the category), and an error.
func getCategoryBounds(bookmarks string, index int) (start int, end int, err error) {
	lines := strings.Split(bookmarks, "\n")
	currentIndex := -1
	start = -1
	end = -1

	// pendingCommentsStart tracks the index where the current block of standalone comments begins.
	pendingCommentsStart := -1

	for i, rawLine := range lines {
		if len(strings.TrimSpace(rawLine)) == 0 {
			continue
		}
		isStandalone, line, _, _ := parseLineForComments(rawLine)
		if isStandalone {
			if pendingCommentsStart == -1 {
				pendingCommentsStart = i
			}
			continue
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)

		// Is it a directive?
		isDirective := false
		isCategory := false

		if lower == "tab" || strings.HasPrefix(lower, "tab ") || strings.HasPrefix(lower, "tab:") {
			isDirective = true
		} else if lower == "page" || strings.HasPrefix(lower, "page ") || strings.HasPrefix(lower, "page:") {
			isDirective = true
		} else if trimmed == "--" {
			isDirective = true
		} else if strings.EqualFold(trimmed, "column") || strings.HasPrefix(lower, "column ") || strings.HasPrefix(lower, "column:") {
			isDirective = true
		} else {
			parts := strings.Fields(trimmed)
			if len(parts) > 0 {
				lowerFirst := strings.ToLower(parts[0])
				if strings.HasPrefix(lowerFirst, "category") {
					isDirective = true
					isCategory = true
				}
			}
		}

		if isDirective {
			if isCategory {
				currentIndex++
				if currentIndex == index {
					// We found our category!
					if pendingCommentsStart != -1 {
						start = pendingCommentsStart
					} else {
						start = i
					}
				} else if currentIndex == index+1 {
					// We found the NEXT category, so the current category ends before its pending comments.
					if pendingCommentsStart != -1 {
						end = pendingCommentsStart
					} else {
						end = i
					}
					break
				}
			} else {
				// We found a non-category directive (Tab, Page, Block, Column)
				if currentIndex == index {
					// It ends before the pending comments of this directive
					if pendingCommentsStart != -1 {
						end = pendingCommentsStart
					} else {
						end = i
					}
					break
				}
			}
			pendingCommentsStart = -1
		} else {
			// It's a regular entry inside a category. It resets pending comments because
			// they belong to this entry.
			pendingCommentsStart = -1
		}
	}

	if start != -1 && end == -1 {
		// It was the last category in the file.
		end = len(lines)
		if pendingCommentsStart != -1 {
			end = pendingCommentsStart
		}
	}

	if start == -1 || end == -1 {
		return -1, -1, fmt.Errorf("category index %d not found", index)
	}

	return start, end, nil
}

// ExtractCategoryByIndex returns the category text for the nth category (0 based)
func ExtractCategoryByIndex(bookmarks string, index int) (string, error) {
	lines := strings.Split(bookmarks, "\n")
	start, end, err := getCategoryBounds(bookmarks, index)
	if err != nil {
		return "", err
	}
	return strings.Join(lines[start:end], "\n"), nil
}

// ReplaceCategoryByIndex replaces the nth category with newText
func ReplaceCategoryByIndex(bookmarks string, index int, newText string) (string, error) {
	lines := strings.Split(bookmarks, "\n")
	start, end, err := getCategoryBounds(bookmarks, index)
	if err != nil {
		return "", err
	}

	var result []string
	result = append(result, lines[:start]...)

	if newText != "" {
		newLines := strings.Split(newText, "\n")
		result = append(result, newLines...)
	}

	result = append(result, lines[end:]...)
	return strings.Join(result, "\n"), nil
}
