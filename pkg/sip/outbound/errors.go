package outbound

import "errors"

var (
	// ErrNoSignalingSender is returned when Dial is called before BindSender.
	ErrNoSignalingSender = errors.New("sip/outbound: signaling sender not bound")
	// ErrNotImplemented marks transfer/bridge paths not yet wired.
	ErrNotImplemented = errors.New("sip/outbound: not implemented")
)
