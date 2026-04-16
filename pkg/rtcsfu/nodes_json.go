// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package rtcsfu

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type sfuNodeSpec struct {
	ID        string `json:"id"`
	Region    string `json:"region"`
	SignalURL string `json:"signal_url"`
	MediaURL  string `json:"media_url,omitempty"`
	Healthy   *bool  `json:"healthy,omitempty"`
	Draining  bool   `json:"draining,omitempty"`
}

// ParseNodesJSON parses a JSON array of SFU node objects into []SFUNode.
func ParseNodesJSON(data []byte) ([]SFUNode, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var specs []sfuNodeSpec
	if err := json.Unmarshal(data, &specs); err != nil {
		return nil, fmt.Errorf("rtcsfu nodes json: %w", err)
	}
	out := make([]SFUNode, 0, len(specs))
	for i, s := range specs {
		if s.ID == "" {
			return nil, fmt.Errorf("rtcsfu nodes json: index %d missing id", i)
		}
		h := true
		if s.Healthy != nil {
			h = *s.Healthy
		}
		out = append(out, SFUNode{
			ID:        NodeID(s.ID),
			Region:    RegionID(s.Region),
			SignalURL: s.SignalURL,
			MediaURL:  s.MediaURL,
			Healthy:   h,
			Draining:  s.Draining,
		})
	}
	return out, nil
}

// ParseNodesJSONFlexible parses the same objects as ParseNodesJSON, but accepts either a JSON array
// or a single JSON object (common for one replica descriptor).
func ParseNodesJSONFlexible(raw []byte) ([]SFUNode, error) {
	b := bytes.TrimSpace(raw)
	if len(b) == 0 {
		return nil, fmt.Errorf("rtcsfu nodes json: empty input")
	}
	if b[0] == '[' {
		return ParseNodesJSON(raw)
	}
	wrapped := make([]byte, 0, len(b)+2)
	wrapped = append(wrapped, '[')
	wrapped = append(wrapped, b...)
	wrapped = append(wrapped, ']')
	return ParseNodesJSON(wrapped)
}
