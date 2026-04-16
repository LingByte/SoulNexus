// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package rtcsfu

import (
	"fmt"
	"hash/fnv"
	"slices"
)

// RoomRouter picks an SFU node for a room using deterministic hashing.
type RoomRouter struct {
	nodes []SFUNode
}

// NewRoomRouter returns a router over the given snapshot of nodes.
// Nodes should be sorted or stable-ordered by ID for reproducible tie-breaking.
func NewRoomRouter(nodes []SFUNode) *RoomRouter {
	cp := make([]SFUNode, len(nodes))
	copy(cp, nodes)
	return &RoomRouter{nodes: cp}
}

// Pick returns the best eligible node for the request.
// Preference order: same-region eligible nodes, then any eligible node.
func (r *RoomRouter) Pick(req RoomRouteRequest) (SFUNode, error) {
	eligible := filterEligible(r.nodes)
	if len(eligible) == 0 {
		return SFUNode{}, fmt.Errorf("rtcsfu: no eligible SFU nodes")
	}
	if req.ClientRegion != "" {
		regional := filterRegion(eligible, req.ClientRegion)
		if len(regional) > 0 {
			eligible = regional
		}
	}
	slices.SortFunc(eligible, func(a, b SFUNode) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	idx := int(roomHash(string(req.RoomID))) % len(eligible)
	if idx < 0 {
		idx += len(eligible)
	}
	return eligible[idx], nil
}

func filterEligible(nodes []SFUNode) []SFUNode {
	out := make([]SFUNode, 0, len(nodes))
	for _, n := range nodes {
		if n.Eligible() {
			out = append(out, n)
		}
	}
	return out
}

func filterRegion(nodes []SFUNode, region RegionID) []SFUNode {
	out := make([]SFUNode, 0, len(nodes))
	for _, n := range nodes {
		if n.Region == region {
			out = append(out, n)
		}
	}
	return out
}

func roomHash(room string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(room))
	return h.Sum32()
}
