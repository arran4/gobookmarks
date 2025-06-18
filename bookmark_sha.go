package gobookmarks

import (
	"crypto/sha256"
	"encoding/hex"
)

func shaOf(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func (e *BookmarkEntry) Sha() string { return shaOf(e.String()) }

func (c *BookmarkCategory) Sha() string { return shaOf(c.String()) }

func (c *BookmarkColumn) Sha() string { return shaOf(c.String()) }

func (b *BookmarkBlock) Sha() string { return shaOf(b.String()) }

func (p *BookmarkPage) Sha() string { return shaOf(p.Name + "\n" + p.String()) }

func (t *BookmarkTab) Sha() string { return shaOf(t.String()) }

func (b BookmarkList) Sha() string { return shaOf(b.String()) }
