package gobookmarks

import (
	"fmt"
	"strings"
)

func tabBoundaries(lines []string) []int {
	starts := []int{0}
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if lower == "tab" || strings.HasPrefix(lower, "tab ") || strings.HasPrefix(lower, "tab:") {
			if i != 0 {
				starts = append(starts, i)
			}
		}
	}
	starts = append(starts, len(lines))
	return starts
}

// ExtractTab returns the text for a tab by name including the 'Tab:' line.
func ExtractTab(bookmarks, name string) (string, error) {
	lines := strings.Split(bookmarks, "\n")
	start := -1
	end := len(lines)
	lower := strings.ToLower(name)
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), "tab:") {
			tabName := strings.TrimSpace(line[4:])
			if start == -1 && strings.EqualFold(tabName, lower) {
				start = i
			} else if start != -1 {
				end = i
				break
			}
		}
	}
	if start == -1 {
		return "", fmt.Errorf("tab %s not found", name)
	}
	return strings.Join(lines[start:end], "\n"), nil
}

// ExtractTabByIndex returns the text for a tab by position including the tab header if present.
func ExtractTabByIndex(bookmarks string, idx int) (string, error) {
	lines := strings.Split(bookmarks, "\n")
	starts := tabBoundaries(lines)
	if idx < 0 || idx >= len(starts)-1 {
		return "", fmt.Errorf("tab %d not found", idx)
	}
	start := starts[idx]
	end := starts[idx+1]
	return strings.Join(lines[start:end], "\n"), nil
}

// ReplaceTab replaces the tab with name with newName and newText.
// newText should not include the leading 'Tab:' line.
func ReplaceTab(bookmarks, name, newName, newText string) (string, error) {
	lines := strings.Split(bookmarks, "\n")
	start := -1
	end := len(lines)
	lower := strings.ToLower(name)
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), "tab:") {
			tabName := strings.TrimSpace(line[4:])
			if start == -1 && strings.EqualFold(tabName, lower) {
				start = i
			} else if start != -1 {
				end = i
				break
			}
		}
	}
	if start == -1 {
		return "", fmt.Errorf("tab %s not found", name)
	}
	var result []string
	result = append(result, lines[:start]...)
	result = append(result, "Tab: "+newName)
	if newText != "" {
		newLines := strings.Split(strings.TrimSuffix(newText, "\n"), "\n")
		result = append(result, newLines...)
	}
	result = append(result, lines[end:]...)
	return strings.Join(result, "\n"), nil
}

// ReplaceTabByIndex replaces the tab at index idx with a new name and text.
// newText should not include the leading 'Tab:' line.
func ReplaceTabByIndex(bookmarks string, idx int, newName, newText string) (string, error) {
	lines := strings.Split(bookmarks, "\n")
	starts := tabBoundaries(lines)
	if idx < 0 || idx >= len(starts)-1 {
		return "", fmt.Errorf("tab %d not found", idx)
	}
	start := starts[idx]
	end := starts[idx+1]

	var replacement []string
	includeHeader := !(idx == 0 && newName == "")
	if includeHeader {
		if newName != "" {
			replacement = append(replacement, "Tab: "+newName)
		} else {
			replacement = append(replacement, "Tab")
		}
	}
	newText = strings.TrimSuffix(newText, "\n")
	if newText != "" {
		replacement = append(replacement, strings.Split(newText, "\n")...)
	}

	var result []string
	result = append(result, lines[:start]...)
	result = append(result, replacement...)
	result = append(result, lines[end:]...)
	return strings.Join(result, "\n"), nil
}

// AppendTab appends a new tab with name and text to bookmarks.
func AppendTab(bookmarks, name, text string) string {
	if !strings.HasSuffix(bookmarks, "\n") {
		bookmarks += "\n"
	}
	bookmarks += "Tab: " + name
	if text != "" {
		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		bookmarks += "\n" + strings.TrimSuffix(text, "\n")
	}
	if !strings.HasSuffix(bookmarks, "\n") {
		bookmarks += "\n"
	} else {
		if !strings.HasSuffix(bookmarks, "\n\n") {
			bookmarks += ""
		}
	}
	return bookmarks
}
