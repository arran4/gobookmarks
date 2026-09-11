package harness_tests

import (
	"github.com/arran4/gobookmarks"
	"testing"
)

// Dummy file to not break CI.
// We moved TestSessionLifecycleRegression into cmd/gobookmarks where setupRouter is accessible.
func TestDummy(t *testing.T) {
	_ = gobookmarks.Config
}
