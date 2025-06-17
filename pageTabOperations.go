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

func insertColumn(p *BookmarkPage, blockIdx, index int) {
	if blockIdx < 0 || blockIdx >= len(p.Blocks) {
		return
	}
	b := p.Blocks[blockIdx]
	if index < 0 || index > len(b.Columns) {
		return
	}
	col := &BookmarkColumn{}
	b.Columns = append(b.Columns[:index], append([]*BookmarkColumn{col}, b.Columns[index:]...)...)
}

func moveCategoryTo(t *BookmarkTab, sp, sb, sc, si, dp, db, dc, di int) {
	if sp < 0 || sp >= len(t.Pages) || dp < 0 || dp >= len(t.Pages) {
		return
	}
	spg, dpg := t.Pages[sp], t.Pages[dp]
	if sb < 0 || sb >= len(spg.Blocks) || db < 0 || db >= len(dpg.Blocks) {
		return
	}
	sbk, dbk := spg.Blocks[sb], dpg.Blocks[db]
	if sc < 0 || sc >= len(sbk.Columns) || dc < 0 || dc >= len(dbk.Columns) {
		return
	}
	sCol, dCol := sbk.Columns[sc], dbk.Columns[dc]
	if si < 0 || si >= len(sCol.Categories) || di < 0 || di > len(dCol.Categories) {
		return
	}
	cat := sCol.Categories[si]
	sCol.Categories = append(sCol.Categories[:si], sCol.Categories[si+1:]...)
	dCol.Categories = append(dCol.Categories[:di], append([]*BookmarkCategory{cat}, dCol.Categories[di:]...)...)
}

func moveEntryBetween(src *BookmarkCategory, si int, dst *BookmarkCategory, di int) {
	if si < 0 || si >= len(src.Entries) || di < 0 || di > len(dst.Entries) {
		return
	}
	e := src.Entries[si]
	src.Entries = append(src.Entries[:si], src.Entries[si+1:]...)
	dst.Entries = append(dst.Entries[:di], append([]*BookmarkEntry{e}, dst.Entries[di:]...)...)
}
