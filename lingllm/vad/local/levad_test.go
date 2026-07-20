//go:build levad

package local

import "testing"

func TestLevadVersion(t *testing.T) {
	if !LevadEnabled() {
		t.Fatal("expected LevadEnabled")
	}
	if LevadVersion() == "" {
		t.Fatal("empty version")
	}
}

func TestLevadTinySileroSilence(t *testing.T) {
	v, err := NewTinySileroVAD(16000)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	// 512 samples silence @ 16k
	pcm := make([]byte, 512*2)
	ok, err := v.IsSpeech(pcm)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("silence should not be speech")
	}
}

func TestLevadTinyTenNeedMoreThenSilence(t *testing.T) {
	v, err := NewTinyTenVAD(16000)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	pcm := make([]byte, 256*2)
	ok, err := v.IsSpeech(pcm)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("silence should not be speech")
	}
}
