//go:build ledenoise

package denoise

import "testing"

func TestLedenoiseVersion(t *testing.T) {
	if !LedenoiseEnabled() {
		t.Fatal("expected enabled")
	}
	if LedenoiseVersion() == "" {
		t.Fatal("empty version")
	}
}

func TestLedenoiseProcess16k(t *testing.T) {
	d, err := NewLedenoise(16000)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	// 20ms @ 16k = 320 samples
	pcm := make([]byte, 320*2)
	out := d.Process(pcm)
	if len(out) < 2 {
		t.Fatalf("empty out len=%d", len(out))
	}
}
