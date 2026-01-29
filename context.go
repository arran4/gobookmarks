package gobookmarks

import (
	"context"

	"github.com/arran4/gobookmarks/core"
)

// GetCore retrieves the Core interface from the context.
// It returns nil if not found or if the value does not implement Core.
func GetCore(ctx context.Context) core.Core {
	if v := ctx.Value(core.ContextValues("coreData")); v != nil {
		if c, ok := v.(core.Core); ok {
			return c
		}
	}
	return nil
}
