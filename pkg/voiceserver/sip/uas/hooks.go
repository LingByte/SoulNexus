package uas

import (
	"fmt"
	"net"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// InviteHandler handles an inbound INVITE. Return nil, nil to send nothing (rare); return an error to answer 500.
type InviteHandler func(req *stack.Message, addr *net.UDPAddr) (*stack.Message, error)

// SimpleHandler handles a request that typically answers with a small final response (BYE, CANCEL, …).
type SimpleHandler func(req *stack.Message, addr *net.UDPAddr) (*stack.Message, error)

// AckHandler handles inbound ACK (no response is sent on the same socket for ACK).
type AckHandler func(req *stack.Message, addr *net.UDPAddr) error

// Handlers lists optional UAS callbacks. Nil fields are not registered.
// Register with (*stack.Endpoint).RegisterHandler via Handlers.Attach.
type Handlers struct {
	Invite    InviteHandler
	Ack       AckHandler
	Bye       SimpleHandler
	Cancel    SimpleHandler
	Options   SimpleHandler
	Register  SimpleHandler
	Info      SimpleHandler
	Prack     SimpleHandler
	Subscribe SimpleHandler
	Notify    SimpleHandler
	Publish   SimpleHandler
	Refer     SimpleHandler
	Message   SimpleHandler
	Update    SimpleHandler
}

func wrapInvite(h InviteHandler) stack.HandlerFunc {
	if h == nil {
		return nil
	}
	return func(req *stack.Message, addr *net.UDPAddr) *stack.Message {
		resp, err := h(req, addr)
		if err != nil {
			r, e2 := ErrorResponse(req, 500, "Server Internal Error")
			if e2 != nil || r == nil {
				return nil
			}
			return r
		}
		return resp
	}
}

func wrapSimple(h SimpleHandler) stack.HandlerFunc {
	if h == nil {
		return nil
	}
	return func(req *stack.Message, addr *net.UDPAddr) *stack.Message {
		resp, err := h(req, addr)
		if err != nil {
			r, _ := ErrorResponse(req, 500, "Server Internal Error")
			return r
		}
		return resp
	}
}

func wrapAck(h AckHandler) stack.HandlerFunc {
	if h == nil {
		return nil
	}
	return func(req *stack.Message, addr *net.UDPAddr) *stack.Message {
		if err := h(req, addr); err != nil {
			// ACK is not answered with SIP response on same transaction; log at application layer.
			return nil
		}
		return nil
	}
}

// defaultOptions answers OPTIONS with a static Allow list (override with Handlers.Options).
func defaultOptions(req *stack.Message, addr *net.UDPAddr) (*stack.Message, error) {
	allow := strings.Join([]string{
		stack.MethodInvite,
		stack.MethodAck,
		stack.MethodBye,
		stack.MethodCancel,
		stack.MethodOptions,
		stack.MethodRegister,
		stack.MethodPrack,
		stack.MethodSubscribe,
		stack.MethodNotify,
		stack.MethodPublish,
		stack.MethodInfo,
		stack.MethodRefer,
		stack.MethodMessage,
		stack.MethodUpdate,
	}, ", ")
	resp, err := NewResponse(req, 200, "OK", "", "")
	if err != nil {
		return nil, err
	}
	resp.SetHeader("Allow", allow)
	resp.SetHeader("Accept", "application/sdp")
	return resp, nil
}

// Attach registers all non-nil handlers on ep. If Options is nil, a default OPTIONS 200 handler is installed.
func (h Handlers) Attach(ep *stack.Endpoint) error {
	if ep == nil {
		return fmt.Errorf("sip/uas: nil endpoint")
	}
	if h.Invite != nil {
		ep.RegisterHandler(stack.MethodInvite, wrapInvite(h.Invite))
	}
	if h.Ack != nil {
		ep.RegisterHandler(stack.MethodAck, wrapAck(h.Ack))
	}
	if h.Bye != nil {
		ep.RegisterHandler(stack.MethodBye, wrapSimple(h.Bye))
	}
	if h.Cancel != nil {
		ep.RegisterHandler(stack.MethodCancel, wrapSimple(h.Cancel))
	}
	if h.Options != nil {
		ep.RegisterHandler(stack.MethodOptions, wrapSimple(h.Options))
	} else {
		ep.RegisterHandler(stack.MethodOptions, wrapSimple(defaultOptions))
	}
	if h.Register != nil {
		ep.RegisterHandler(stack.MethodRegister, wrapSimple(h.Register))
	}
	if h.Info != nil {
		ep.RegisterHandler(stack.MethodInfo, wrapSimple(h.Info))
	}
	if h.Prack != nil {
		ep.RegisterHandler(stack.MethodPrack, wrapSimple(h.Prack))
	}
	if h.Subscribe != nil {
		ep.RegisterHandler(stack.MethodSubscribe, wrapSimple(h.Subscribe))
	}
	if h.Notify != nil {
		ep.RegisterHandler(stack.MethodNotify, wrapSimple(h.Notify))
	}
	if h.Publish != nil {
		ep.RegisterHandler(stack.MethodPublish, wrapSimple(h.Publish))
	}
	if h.Refer != nil {
		ep.RegisterHandler(stack.MethodRefer, wrapSimple(h.Refer))
	}
	if h.Message != nil {
		ep.RegisterHandler(stack.MethodMessage, wrapSimple(h.Message))
	}
	if h.Update != nil {
		ep.RegisterHandler(stack.MethodUpdate, wrapSimple(h.Update))
	}
	return nil
}
