package uas

import (
	"context"
	"net"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/transaction"
)

func TestWrapHandlersWithTransaction_NoMgr(t *testing.T) {
	var h Handlers
	h.Invite = func(req *stack.Message, addr *net.UDPAddr) (*stack.Message, error) { return nil, nil }
	w := WrapHandlersWithTransaction(h, TransactionBinding{})
	if w.Invite == nil {
		t.Fatal("expected invite preserved")
	}
}

func TestAttachWithTransaction_AppendsOnResponseSent(t *testing.T) {
	mgr := transaction.NewManager()
	ep := stack.NewEndpoint(stack.EndpointConfig{Host: "127.0.0.1", Port: 0})
	if err := ep.Open(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ep.Close() }()

	var h Handlers
	send := func(msg *stack.Message, addr *net.UDPAddr) error { return ep.Send(msg, addr) }
	b := TransactionBinding{Mgr: mgr, Send: send, Ctx: context.Background()}
	if err := h.AttachWithTransaction(ep, b); err != nil {
		t.Fatal(err)
	}
	// Smoke: handler registration did not panic
	_ = ep
}
