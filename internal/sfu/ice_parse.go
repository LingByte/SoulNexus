// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pion/webrtc/v3"
)

// DefaultICEServersJSON is used when RTCSFU_ICE_SERVERS_JSON is empty.
const DefaultICEServersJSON = `[{"urls":["stun:stun.l.google.com:19302"]}]`

type iceServerSpec struct {
	URLs       json.RawMessage `json:"urls"`
	Username   string          `json:"username,omitempty"`
	Credential string          `json:"credential,omitempty"`
}

// ParseICEServersJSON parses WebRTC ICE server definitions (browser-compatible JSON)
// into pion ICE servers and returns normalized JSON for API clients.
func ParseICEServersJSON(raw string) ([]webrtc.ICEServer, []byte, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		s = DefaultICEServersJSON
	}
	var specs []iceServerSpec
	if err := json.Unmarshal([]byte(s), &specs); err != nil {
		return nil, nil, fmt.Errorf("sfu ice json: %w", err)
	}
	out := make([]webrtc.ICEServer, 0, len(specs))
	for i, sp := range specs {
		urls, err := parseURLsField(sp.URLs)
		if err != nil {
			return nil, nil, fmt.Errorf("sfu ice json entry %d: %w", i, err)
		}
		if len(urls) == 0 {
			return nil, nil, fmt.Errorf("sfu ice json entry %d: empty urls", i)
		}
		srv := webrtc.ICEServer{URLs: urls}
		if sp.Username != "" {
			srv.Username = sp.Username
			srv.Credential = sp.Credential
			srv.CredentialType = webrtc.ICECredentialTypePassword
		}
		out = append(out, srv)
	}
	norm, err := json.Marshal(specs)
	if err != nil {
		return nil, nil, err
	}
	return out, norm, nil
}

func parseURLsField(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	if raw[0] == '"' {
		var one string
		if err := json.Unmarshal(raw, &one); err != nil {
			return nil, err
		}
		return []string{one}, nil
	}
	var many []string
	if err := json.Unmarshal(raw, &many); err != nil {
		return nil, err
	}
	return many, nil
}
