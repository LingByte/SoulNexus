package media

import (
	"testing"
)

func TestPacketRequest(t *testing.T) {
	req := PacketRequest[string]{
		H:         nil,
		Interrupt: false,
		Req:       "test",
	}

	if req.Req != "test" {
		t.Errorf("PacketRequest.Req = %v, want test", req.Req)
	}
	if req.Interrupt {
		t.Error("PacketRequest.Interrupt should be false")
	}
}

func TestAsyncTaskRunner_Creation(t *testing.T) {
	runner := NewAsyncTaskRunner[string](2)

	if runner.WorkerPoolSize != 2 {
		t.Errorf("WorkerPoolSize = %v, want 2", runner.WorkerPoolSize)
	}
}
