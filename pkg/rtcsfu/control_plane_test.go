// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package rtcsfu

import (
	"testing"
	"time"
)

func TestControlPlane_Join_ReassignWhenNodeGone(t *testing.T) {
	p := NewControlPlane([]SFUNode{
		{ID: "n0", Region: "r", Healthy: true, SignalURL: "https://n0"},
	}, 0)
	a1, err := p.Join("room-x", "r")
	if err != nil || a1.Node.ID != "n0" {
		t.Fatalf("first join: %+v err=%v", a1, err)
	}
	p.Reload([]SFUNode{
		{ID: "n1", Region: "r", Healthy: true, SignalURL: "https://n1"},
	})
	a2, err := p.Join("room-x", "r")
	if err != nil {
		t.Fatal(err)
	}
	if a2.Node.ID != "n1" {
		t.Fatalf("expected reassignment to n1, got %q", a2.Node.ID)
	}
}

func TestControlPlane_TouchReplicaUpdatesLastSeen(t *testing.T) {
	p := NewControlPlane(nil, 0)
	p.UpsertReplica(SFUNode{ID: "r1", Region: "x", Healthy: true, SignalURL: "https://a"})
	rows := p.ReplicaRows()
	if len(rows) != 1 || rows[0].LastSeenUnix == 0 {
		t.Fatalf("unexpected rows: %+v", rows)
	}
	first := rows[0].LastSeenUnix
	time.Sleep(1100 * time.Millisecond)
	if !p.TouchReplica("r1") {
		t.Fatal("touch failed")
	}
	rows2 := p.ReplicaRows()
	if len(rows2) != 1 || rows2[0].LastSeenUnix < first {
		t.Fatalf("last_seen not advanced: first=%d second=%d", first, rows2[0].LastSeenUnix)
	}
}

func TestControlPlane_StaleReplicaUnhealthyInMerge(t *testing.T) {
	p := NewControlPlane(nil, 1)
	p.UpsertReplica(SFUNode{ID: "r1", Region: "x", Healthy: true, SignalURL: "https://a"})
	time.Sleep(1200 * time.Millisecond)
	merged := p.MergedSnapshot()
	if len(merged) != 1 || merged[0].Healthy {
		t.Fatalf("want unhealthy stale replica, got %+v", merged)
	}
	rows := p.ReplicaRows()
	if len(rows) != 1 || !rows[0].Stale || rows[0].Node.Healthy {
		t.Fatalf("want stale row, got %+v", rows[0])
	}
}

func TestControlPlane_ReplicasMergedIntoRouting(t *testing.T) {
	p := NewControlPlane([]SFUNode{
		{ID: "static-1", Region: "r", Healthy: true, SignalURL: "https://static"},
	}, 0)
	p.UpsertReplica(SFUNode{ID: "rep-1", Region: "r", Healthy: true, SignalURL: "https://rep"})
	if p.NodeCount() != 2 {
		t.Fatalf("node count want 2 got %d", p.NodeCount())
	}
	a, err := p.Join("room-a", "r")
	if err != nil {
		t.Fatal(err)
	}
	if a.Node.SignalURL == "" {
		t.Fatalf("empty assignment: %+v", a)
	}
}

func TestParseNodesJSON_DefaultHealthy(t *testing.T) {
	raw := `[{"id":"a","region":"cn","signal_url":"https://s"}]`
	nodes, err := ParseNodesJSON([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 || !nodes[0].Healthy || nodes[0].Draining {
		t.Fatalf("%+v", nodes[0])
	}
}
