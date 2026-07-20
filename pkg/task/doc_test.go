package task

import "testing"

func TestPackageDocSmoke(t *testing.T) {
	if TaskStatusPending.String() != "pending" {
		t.Fatalf("TaskStatusPending: %q", TaskStatusPending.String())
	}
}
