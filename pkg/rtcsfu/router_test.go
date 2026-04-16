// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package rtcsfu

import "testing"

func TestRoomRouter_Pick_RegionAffinity(t *testing.T) {
	nodes := []SFUNode{
		{ID: "a", Region: "us", Healthy: true},
		{ID: "b", Region: "eu", Healthy: true},
		{ID: "c", Region: "eu", Healthy: true},
	}
	rt := NewRoomRouter(nodes)
	got, err := rt.Pick(RoomRouteRequest{RoomID: "room-1", ClientRegion: "eu"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Region != "eu" {
		t.Fatalf("expected eu node, got %q region %q", got.ID, got.Region)
	}
}

func TestRoomRouter_Pick_FallbackWhenRegionEmpty(t *testing.T) {
	nodes := []SFUNode{
		{ID: "a", Region: "us", Healthy: true},
	}
	rt := NewRoomRouter(nodes)
	got, err := rt.Pick(RoomRouteRequest{RoomID: "room-x", ClientRegion: "eu"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "a" {
		t.Fatalf("expected fallback to us node, got %q", got.ID)
	}
}

func TestRoomRouter_Pick_Deterministic(t *testing.T) {
	nodes := []SFUNode{
		{ID: "n0", Region: "r", Healthy: true},
		{ID: "n1", Region: "r", Healthy: true},
		{ID: "n2", Region: "r", Healthy: true},
	}
	rt := NewRoomRouter(nodes)
	a, _ := rt.Pick(RoomRouteRequest{RoomID: "fixed-room", ClientRegion: "r"})
	b, _ := rt.Pick(RoomRouteRequest{RoomID: "fixed-room", ClientRegion: "r"})
	if a.ID != b.ID {
		t.Fatalf("expected deterministic pick, %q vs %q", a.ID, b.ID)
	}
}

func TestRoomRouter_Pick_SkipsIneligible(t *testing.T) {
	nodes := []SFUNode{
		{ID: "bad", Region: "r", Healthy: false},
		{ID: "ok", Region: "r", Healthy: true},
	}
	rt := NewRoomRouter(nodes)
	got, err := rt.Pick(RoomRouteRequest{RoomID: "any", ClientRegion: "r"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "ok" {
		t.Fatalf("expected ok, got %q", got.ID)
	}
}
