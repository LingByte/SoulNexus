package devices

import "testing"

func TestListPlaybackDevices(t *testing.T) {
	streamCtx, err := NewStreamContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	devices, err := ListPlaybackDevices(streamCtx.ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, device := range devices {
		t.Logf("Device: %s (%s)", device.Name, device.ID)
	}
}

func TestListCaptureDevices(t *testing.T) {
	streamCtx, err := NewStreamContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	devices, err := ListCaptureDevices(streamCtx.ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, device := range devices {
		t.Logf("Device: %s (%s)", device.Name, device.ID)
	}
}

func TestPrintPlaybackDevices(t *testing.T) {
	streamCtx, err := NewStreamContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	err = PrintPlaybackDevices(streamCtx.ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPrintCaptureDevices(t *testing.T) {
	streamCtx, err := NewStreamContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	err = PrintCaptureDevices(streamCtx.ctx)
	if err != nil {
		t.Fatal(err)
	}
}
