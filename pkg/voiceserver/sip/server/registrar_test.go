package server

import "testing"

func TestParseURIUserHost_nameAddr(t *testing.T) {
	cases := []struct {
		in       string
		wantUser string
		wantHost string
		ok       bool
	}{
		{`"bob" <sip:bob@192.168.72.149>`, "bob", "192.168.72.149", true},
		{`<sip:bob@192.168.72.149>`, "bob", "192.168.72.149", true},
		{`sip:bob@192.168.72.149:6050`, "bob", "192.168.72.149", true},
		{`sip:bob@192.168.72.149;user=phone`, "bob", "192.168.72.149", true},
	}
	for _, tc := range cases {
		u, h, ok := parseURIUserHost(tc.in)
		if ok != tc.ok || u != tc.wantUser || h != tc.wantHost {
			t.Errorf("parseURIUserHost(%q) = (%q,%q,%v) want (%q,%q,%v)",
				tc.in, u, h, ok, tc.wantUser, tc.wantHost, tc.ok)
		}
	}
}
