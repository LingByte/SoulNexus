// Package stack is the SIP/2.0 signaling layer for pkg/sip.
//
// It provides:
//   - Message parsing and serialization (RFC 3261–style text format)
//   - UDP datagram transport abstraction
//   - A minimal signaling Endpoint (listen, parse, dispatch requests; pass responses to a hook;
//     optional OnResponseSent after a successful response send for UAS server-tx binding)
//
// This package intentionally does not implement full SIP transaction state machines,
// digest authentication, or TCP/TLS transports yet; those belong in
// higher layers built on top of Message + Endpoint.
//
// Design goals:
//   - No process-wide mutable defaults; configure via structs and context.
//   - Read loop returns on non-timeout transport errors (after optional OnReadError logging).
//   - SIP wire parsing (Parse/String/BodyBytesLen) and UDP signaling live in this package.
//
// Audio codecs for RTP live in pkg/media/encoder (CreateEncode/CreateDecode); stack does not implement codecs.
package stack
