package timeutil

import "testing"

func TestInitAndName(t *testing.T) {
	Init("Asia/Shanghai")
	if Name() != "Asia/Shanghai" {
		t.Fatalf("name=%q", Name())
	}
	now := Now()
	if now.Location().String() != "Asia/Shanghai" {
		t.Fatalf("loc=%v", now.Location())
	}
}

func TestInitInvalidFallback(t *testing.T) {
	Init("Not/A/Timezone")
	if Name() != DefaultTimezone {
		t.Fatalf("name=%q", Name())
	}
}
