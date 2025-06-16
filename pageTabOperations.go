package gobookmarks

func moveTab(tabs []*BookmarkTab, index, delta int) []*BookmarkTab {
	if index+delta < 0 || index+delta >= len(tabs) {
		return tabs
	}
	tabs[index], tabs[index+delta] = tabs[index+delta], tabs[index]
	return tabs
}

func movePage(t *BookmarkTab, index, delta int) {
	if index+delta < 0 || index+delta >= len(t.Pages) {
		return
	}
	t.Pages[index], t.Pages[index+delta] = t.Pages[index+delta], t.Pages[index]
}

func insertPage(t *BookmarkTab, index int) {
	if index < 0 || index > len(t.Pages) {
		return
	}
	p := &BookmarkPage{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}
	t.Pages = append(t.Pages[:index], append([]*BookmarkPage{p}, t.Pages[index:]...)...)
}

func deletePage(t *BookmarkTab, index int) {
	if index < 0 || index >= len(t.Pages) {
		return
	}
	t.Pages = append(t.Pages[:index], t.Pages[index+1:]...)
	if len(t.Pages) == 0 {
		t.Pages = []*BookmarkPage{{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}}
	}
}

func insertTab(tabs []*BookmarkTab, index int) []*BookmarkTab {
	if index < 0 || index > len(tabs) {
		return tabs
	}
	tab := &BookmarkTab{}
	tabs = append(tabs[:index], append([]*BookmarkTab{tab}, tabs[index:]...)...)
	return tabs
}

func deleteTab(tabs []*BookmarkTab, index int) []*BookmarkTab {
	if index < 0 || index >= len(tabs) {
		return tabs
	}
	tabs = append(tabs[:index], tabs[index+1:]...)
	if len(tabs) == 0 {
		tabs = []*BookmarkTab{{Pages: []*BookmarkPage{{Blocks: []*BookmarkBlock{{Columns: []*BookmarkColumn{{}}}}}}}}
	}
	return tabs
}
