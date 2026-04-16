// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package rtcsfu

import "testing"

func TestRoomRegistry_GetOrAssign_Sticky(t *testing.T) {
	nodes := []SFUNode{
		{ID: "n0", Region: "r", Healthy: true, SignalURL: "https://n0/s"},
		{ID: "n1", Region: "r", Healthy: true, SignalURL: "https://n1/s"},
	}
	rt := NewRoomRouter(nodes)
	reg := NewRoomRegistry()
	a1, err := reg.GetOrAssign("room-a", "r", rt)
	if err != nil {
		t.Fatal(err)
	}
	a2, err := reg.GetOrAssign("room-a", "r", rt)
	if err != nil {
		t.Fatal(err)
	}
	if a1.Node.ID != a2.Node.ID {
		t.Fatalf("sticky assignment broken: %q vs %q", a1.Node.ID, a2.Node.ID)
	}
}

func TestRoomRegistry_Migrate(t *testing.T) {
	nodes := []SFUNode{
		{ID: "n0", Region: "r", Healthy: true},
		{ID: "n1", Region: "r", Healthy: true},
	}
	rt := NewRoomRouter(nodes)
	reg := NewRoomRegistry()
	_, _ = reg.GetOrAssign("room-m", "r", rt)
	if err := reg.Migrate("room-m", SFUNode{ID: "n1", Region: "r", Healthy: true, SignalURL: "x"}); err != nil {
		t.Fatal(err)
	}
	a, ok := reg.Get("room-m")
	if !ok || a.Node.ID != "n1" {
		t.Fatalf("migrate failed: %+v ok=%v", a, ok)
	}
	rooms0 := reg.RoomsOnNode("n0")
	rooms1 := reg.RoomsOnNode("n1")
	if len(rooms0) != 0 || len(rooms1) != 1 {
		t.Fatalf("index broken: n0=%v n1=%v", rooms0, rooms1)
	}
}
