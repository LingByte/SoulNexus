package callflags_test

import (
	"testing"

	"github.com/LingByte/SoulNexus/pkg/dialog/callflags"
)

func TestScriptModeRoundTrip(t *testing.T) {
	callID := "script-mode-test"
	if callflags.IsScriptMode(callID) {
		t.Fatal("expected unset")
	}
	callflags.MarkScriptMode(callID)
	if !callflags.IsScriptMode(callID) {
		t.Fatal("expected set")
	}
	callflags.ClearScriptMode(callID)
	if callflags.IsScriptMode(callID) {
		t.Fatal("expected cleared")
	}
	callflags.MarkScriptMode("")
	if callflags.IsScriptMode("") {
		t.Fatal("empty id must stay unset")
	}
}
