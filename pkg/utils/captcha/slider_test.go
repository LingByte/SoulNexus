package captcha

import (
	"testing"
	"time"
)

func TestSliderCaptcha_Verify(t *testing.T) {
	sc := NewSliderCaptcha(300, 0.9, 5*time.Minute, nil)
	result, err := sc.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	ok, err := sc.Verify(result.ID, 280)
	if err != nil || !ok {
		t.Fatalf("expected pass at 280, ok=%v err=%v", ok, err)
	}

	result2, err := sc.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	ok, err = sc.Verify(result2.ID, 50)
	if err != nil || ok {
		t.Fatalf("expected fail at 50, ok=%v err=%v", ok, err)
	}
}
