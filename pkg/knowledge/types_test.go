package knowledge

import "testing"

func TestTypes_ErrorsNonNil(t *testing.T) {
	if ErrEmptyText == nil || ErrNoChunks == nil || ErrChunkerNotFound == nil {
		t.Fatalf("expected sentinel errors")
	}
}

