package captcha

import "testing"

func TestRandomType_InRange(t *testing.T) {
	seen := map[Type]bool{}
	for i := 0; i < 200; i++ {
		got := RandomType()
		switch got {
		case TypeSlider, TypeImage, TypeClick:
			seen[got] = true
		default:
			t.Fatalf("unexpected type %q", got)
		}
	}
	if len(seen) != 3 {
		t.Fatalf("expected all three types over 200 draws, got %d: %v", len(seen), seen)
	}
}

func TestGenerateRandom(t *testing.T) {
	m := NewManager(DefaultConfig())
	for i := 0; i < 20; i++ {
		result, err := m.GenerateRandom()
		if err != nil {
			t.Fatalf("GenerateRandom failed: %v", err)
		}
		if result.Type != TypeSlider && result.Type != TypeImage && result.Type != TypeClick {
			t.Fatalf("unexpected type %q", result.Type)
		}
	}
}
