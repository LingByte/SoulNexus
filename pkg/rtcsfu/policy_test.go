// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package rtcsfu

import (
	"testing"
	"time"
)

func TestLastN(t *testing.T) {
	c := []UserID{"a", "b", "c", "d"}
	got := LastN(c, 2)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("unexpected %v", got)
	}
}

func TestLayerSelector_Hysteresis(t *testing.T) {
	sel := NewLayerSelector(1*time.Hour, 250_000, 900_000)
	t0 := time.Unix(1000, 0)
	if got := sel.Update(t0, 2_000_000); got != LayerHigh {
		t.Fatalf("expected high, got %v", got)
	}
	// would want low immediately, but hold blocks
	if got := sel.Update(t0.Add(time.Second), 100_000); got != LayerHigh {
		t.Fatalf("expected hold keeps high, got %v", got)
	}
	t1 := t0.Add(2 * time.Hour)
	if got := sel.Update(t1, 100_000); got != LayerLow {
		t.Fatalf("expected low after hold, got %v", got)
	}
}
