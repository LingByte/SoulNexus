// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package app

import (
	"sync"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/sfu"
)

// sfuManagers tracks every Manager constructed by the SFU mount so the
// process shutdown path can Close() them and reclaim webhook goroutines
// + active participants. Slice + mutex over a sync.Map because we only
// iterate it once at shutdown and the call count is tiny (typically 1).
var (
	sfuManagersMu sync.Mutex
	sfuManagers   []*sfu.Manager
)

// RegisterSFUManagerForShutdown adds m to the shutdown registry.
// Idempotent on Manager.Close — safe to call once per Manager from the
// SFU handler mount.
func RegisterSFUManagerForShutdown(m *sfu.Manager) {
	if m == nil {
		return
	}
	sfuManagersMu.Lock()
	sfuManagers = append(sfuManagers, m)
	sfuManagersMu.Unlock()
}

// ShutdownAllSFUManagers is called by the HTTP listener teardown to
// stop every registered Manager. Idempotent on Manager.Close.
func ShutdownAllSFUManagers() {
	sfuManagersMu.Lock()
	mgrs := append([]*sfu.Manager(nil), sfuManagers...)
	sfuManagers = nil
	sfuManagersMu.Unlock()
	for _, m := range mgrs {
		m.Close()
	}
}
