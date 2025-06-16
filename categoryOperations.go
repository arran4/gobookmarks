package gobookmarks

func moveCategory(t *BookmarkTab, pageIdx, blockIdx, colIdx, index, delta int) {
	if pageIdx < 0 || pageIdx >= len(t.Pages) {
		return
	}
	p := t.Pages[pageIdx]
	if blockIdx < 0 || blockIdx >= len(p.Blocks) {
		return
	}
	b := p.Blocks[blockIdx]
	if colIdx < 0 || colIdx >= len(b.Columns) {
		return
	}
	c := b.Columns[colIdx]
	if index+delta < 0 || index+delta >= len(c.Categories) {
		return
	}
	c.Categories[index], c.Categories[index+delta] = c.Categories[index+delta], c.Categories[index]
}

func insertCategory(t *BookmarkTab, pageIdx, blockIdx, colIdx, index int) {
	if pageIdx < 0 || pageIdx > len(t.Pages) {
		return
	}
	p := t.Pages[pageIdx]
	if blockIdx < 0 || blockIdx >= len(p.Blocks) {
		return
	}
	b := p.Blocks[blockIdx]
	if colIdx < 0 || colIdx >= len(b.Columns) {
		return
	}
	c := b.Columns[colIdx]
	if index < 0 || index > len(c.Categories) {
		return
	}
	cat := &BookmarkCategory{}
	c.Categories = append(c.Categories[:index], append([]*BookmarkCategory{cat}, c.Categories[index:]...)...)
}

func deleteCategory(t *BookmarkTab, pageIdx, blockIdx, colIdx, index int) {
	if pageIdx < 0 || pageIdx >= len(t.Pages) {
		return
	}
	p := t.Pages[pageIdx]
	if blockIdx < 0 || blockIdx >= len(p.Blocks) {
		return
	}
	b := p.Blocks[blockIdx]
	if colIdx < 0 || colIdx >= len(b.Columns) {
		return
	}
	c := b.Columns[colIdx]
	if index < 0 || index >= len(c.Categories) {
		return
	}
	c.Categories = append(c.Categories[:index], c.Categories[index+1:]...)
}
