// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package rtcsfu_replica

import "testing"

func TestJoinPrimaryAPI(t *testing.T) {
	u := JoinPrimaryAPI("https://p.example:7075/", "/api/rtcsfu/v1/admin/nodes/register")
	want := "https://p.example:7075/api/rtcsfu/v1/admin/nodes/register"
	if u != want {
		t.Fatalf("got %q want %q", u, want)
	}
}
