package gobookmarks

func insertEntry(cat *BookmarkCategory, index int) {
	if index < 0 || index > len(cat.Entries) {
		return
	}
	e := &BookmarkEntry{Url: "http://", Name: ""}
	cat.Entries = append(cat.Entries[:index], append([]*BookmarkEntry{e}, cat.Entries[index:]...)...)
}

func deleteEntry(cat *BookmarkCategory, index int) {
	if index < 0 || index >= len(cat.Entries) {
		return
	}
	cat.Entries = append(cat.Entries[:index], cat.Entries[index+1:]...)
}

func moveEntry(cat *BookmarkCategory, index, delta int) {
	if index+delta < 0 || index+delta >= len(cat.Entries) {
		return
	}
	cat.Entries[index], cat.Entries[index+delta] = cat.Entries[index+delta], cat.Entries[index]
}
