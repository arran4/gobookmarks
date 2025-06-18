package gobookmarks

import "fmt"

// MoveCategory moves the category at fromIndex so it appears before beforeIndex.
// If beforeIndex is -1 the category is moved to the end. When newColumn is true
// a new column directive is inserted before the moved category.
func (tabs BookmarkList) MoveCategory(fromIndex, beforeIndex int, newColumn bool) error {
	type loc struct {
		block  *BookmarkBlock
		column *BookmarkColumn
		cat    *BookmarkCategory
		colIdx int
		catIdx int
	}
	var cats []loc
	idx := 0
	for _, t := range tabs {
		for _, p := range t.Pages {
			for _, b := range p.Blocks {
				if b.HR {
					continue
				}
				for ci, col := range b.Columns {
					for cj, c := range col.Categories {
						cats = append(cats, loc{b, col, c, ci, cj})
						c.Index = idx
						idx++
					}
				}
			}
		}
	}
	if fromIndex < 0 || fromIndex >= len(cats) {
		return fmt.Errorf("category index %d not found", fromIndex)
	}
	var beforeLoc *loc
	if beforeIndex >= 0 {
		if beforeIndex >= len(cats) {
			return fmt.Errorf("category index %d not found", beforeIndex)
		}
		beforeLoc = &cats[beforeIndex]
	}

	src := cats[fromIndex]
	// remove from source column
	src.column.Categories = append(src.column.Categories[:src.catIdx], src.column.Categories[src.catIdx+1:]...)
	if beforeLoc != nil && beforeIndex > fromIndex && src.column == beforeLoc.column {
		beforeLoc.catIdx--
	}

	if beforeLoc == nil { // append to end
		destBlock := cats[len(cats)-1].block
		destCol := destBlock.Columns[len(destBlock.Columns)-1]
		if newColumn {
			destCol = &BookmarkColumn{}
			destBlock.Columns = append(destBlock.Columns, destCol)
		}
		destCol.Categories = append(destCol.Categories, src.cat)
	} else {
		dest := *beforeLoc
		destCol := dest.column
		insertIdx := dest.catIdx
		if newColumn {
			destCol = &BookmarkColumn{}
			dest.block.Columns = append(dest.block.Columns, destCol)
			insertIdx = 0
		}
		destCol.InsertCategory(insertIdx, src.cat)
	}

	// reindex
	idx = 0
	for _, t := range tabs {
		for _, p := range t.Pages {
			for _, b := range p.Blocks {
				if b.HR {
					continue
				}
				for _, col := range b.Columns {
					for _, c := range col.Categories {
						c.Index = idx
						idx++
					}
				}
			}
		}
	}
	return nil
}

// PageForCategory returns the page containing the category with the given index.
func PageForCategory(tabs BookmarkList, index int) *BookmarkPage {
	idx := 0
	for _, t := range tabs {
		for _, p := range t.Pages {
			for _, b := range p.Blocks {
				if b.HR {
					continue
				}
				for _, col := range b.Columns {
					for range col.Categories {
						if idx == index {
							return p
						}
						idx++
					}
				}
			}
		}
	}
	return nil
}
