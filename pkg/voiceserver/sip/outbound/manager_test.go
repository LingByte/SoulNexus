// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package outbound

import (
	"context"
	"errors"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/transaction"
)

func TestNew_RequiresFields(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
	}{
		{"nil endpoint", Config{TxManager: transaction.NewManager(), LocalIP: "1.1.1.1", LocalSigPort: 5060}},
		{"nil tx", Config{Endpoint: stack.NewEndpoint(stack.EndpointConfig{}), LocalIP: "1.1.1.1", LocalSigPort: 5060}},
		{"empty ip", Config{Endpoint: stack.NewEndpoint(stack.EndpointConfig{}), TxManager: transaction.NewManager(), LocalSigPort: 5060}},
		{"bad port", Config{Endpoint: stack.NewEndpoint(stack.EndpointConfig{}), TxManager: transaction.NewManager(), LocalIP: "1.1.1.1"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := New(c.cfg); err == nil {
				t.Fatalf("expected error for %s", c.name)
			}
		})
	}
}

func TestParseSIPTarget(t *testing.T) {
	cases := []struct {
		raw     string
		wantURI string
		wantErr bool
	}{
		{"sip:bob@127.0.0.1:5060", "sip:bob@127.0.0.1:5060", false},
		{"sip:bob@127.0.0.1", "sip:bob@127.0.0.1:5060", false}, // default port
		{"<sip:bob@127.0.0.1:5060>", "sip:bob@127.0.0.1:5060", false},
		{"sip:bob@127.0.0.1:5060;transport=udp", "sip:bob@127.0.0.1:5060", false},
		{"", "", true},
		{"bob@127.0.0.1", "", true},     // missing sip:
		{"sip:127.0.0.1", "", true},     // missing user
		{"sip:bob@host:0", "", true},    // bad port
		{"sip:bob@host:99999", "", true}, // bad port
	}
	for _, c := range cases {
		t.Run(c.raw, func(t *testing.T) {
			got, err := parseSIPTarget(c.raw)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.uri != c.wantURI {
				t.Errorf("uri=%q, want %q", got.uri, c.wantURI)
			}
		})
	}
}

func TestManager_DialOnClosedReturnsErr(t *testing.T) {
	mgr, err := New(Config{
		Endpoint:     stack.NewEndpoint(stack.EndpointConfig{Host: "127.0.0.1", Port: 0}),
		TxManager:    transaction.NewManager(),
		LocalIP:      "127.0.0.1",
		LocalSigPort: 5060,
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	mgr.Close(context.Background())
	if _, err := mgr.Dial(context.Background(), DialRequest{TargetURI: "sip:bob@127.0.0.1:5060"}); !errors.Is(err, ErrManagerClosed) {
		t.Errorf("expected ErrManagerClosed, got %v", err)
	}
	if err := mgr.Hangup(context.Background(), "anything", "x"); !errors.Is(err, ErrManagerClosed) {
		t.Errorf("expected ErrManagerClosed on hangup, got %v", err)
	}
}

func TestManager_HangupUnknownReturnsErrCallNotFound(t *testing.T) {
	mgr, err := New(Config{
		Endpoint:     stack.NewEndpoint(stack.EndpointConfig{Host: "127.0.0.1", Port: 0}),
		TxManager:    transaction.NewManager(),
		LocalIP:      "127.0.0.1",
		LocalSigPort: 5060,
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if err := mgr.Hangup(context.Background(), "nonexistent", "x"); !errors.Is(err, ErrCallNotFound) {
		t.Errorf("expected ErrCallNotFound, got %v", err)
	}
}

func TestManager_ListEmpty(t *testing.T) {
	mgr, err := New(Config{
		Endpoint:     stack.NewEndpoint(stack.EndpointConfig{Host: "127.0.0.1", Port: 0}),
		TxManager:    transaction.NewManager(),
		LocalIP:      "127.0.0.1",
		LocalSigPort: 5060,
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if got := mgr.List(); len(got) != 0 {
		t.Errorf("List on empty manager = %v, want []", got)
	}
}
