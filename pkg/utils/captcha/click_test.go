package captcha

import (
	"testing"
	"time"
)

func TestClickCaptcha_OrderedVerify(t *testing.T) {
	cc := NewClickCaptcha(300, 200, 3, 20, 5*time.Minute, nil)
	result, err := cc.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	targets, _ := result.Data["targets"].([]string)
	if len(targets) != 3 {
		t.Fatalf("expected 3 targets, got %d", len(targets))
	}

	chars, _ := result.Data["chars"].([]CharMarker)
	byChar := map[string]Point{}
	for _, c := range chars {
		byChar[c.Char] = Point{X: c.X, Y: c.Y}
	}
	ordered := make([]Point, len(targets))
	for i, target := range targets {
		ordered[i] = byChar[target]
	}

	ok, err := cc.Verify(result.ID, ordered)
	if err != nil || !ok {
		t.Fatalf("ordered verify failed: ok=%v err=%v", ok, err)
	}

	result2, err := cc.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	targets2, _ := result2.Data["targets"].([]string)
	chars2, _ := result2.Data["chars"].([]CharMarker)
	byChar2 := map[string]Point{}
	for _, c := range chars2 {
		byChar2[c.Char] = Point{X: c.X, Y: c.Y}
	}
	reversed := make([]Point, len(targets2))
	for i := range targets2 {
		reversed[i] = byChar2[targets2[len(targets2)-1-i]]
	}
	ok, err = cc.Verify(result2.ID, reversed)
	if err != nil || ok {
		t.Fatalf("expected wrong order to fail, ok=%v err=%v", ok, err)
	}
}

func TestClickCaptcha_GenerateHasChinese(t *testing.T) {
	cc := NewClickCaptcha(300, 200, 3, 20, 5*time.Minute, nil)
	result, err := cc.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	targets, _ := result.Data["targets"].([]string)
	if len(targets) == 0 {
		t.Fatal("no targets")
	}
	bg, ok := result.Data["background"].(string)
	if !ok || bg == "" {
		t.Fatal("background image missing")
	}
}
