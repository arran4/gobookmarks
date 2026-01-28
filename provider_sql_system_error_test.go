package gobookmarks

import "testing"

func TestOpenDB_SystemError(t *testing.T) {
	if _, err := OpenDB("", ""); err == nil {
		t.Fatalf("expected error when DB not configured")
	} else if serr, ok := err.(SystemError); !ok || serr.Msg != "Database error" {
		t.Fatalf("expected SystemError 'Database error', got %T %v", err, err)
	}
}
