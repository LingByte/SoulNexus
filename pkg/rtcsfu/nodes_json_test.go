// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package rtcsfu

import "testing"

func TestParseNodesJSONFlexible_Object(t *testing.T) {
	raw := []byte(`{"id":"r1","region":"cn","signal_url":"https://s.example/ws"}`)
	nodes, err := ParseNodesJSONFlexible(raw)
	if err != nil || len(nodes) != 1 || nodes[0].ID != "r1" {
		t.Fatalf("got %+v err=%v", nodes, err)
	}
}
