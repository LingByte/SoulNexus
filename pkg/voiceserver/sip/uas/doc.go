// Package uas registers inbound (UAS-side) SIP method handlers on stack.Endpoint using typed callbacks.
//
// Media path: after SDP negotiation yields a payload type and codec name (e.g. pcmu, pcma, opus),
// build RTP↔PCM paths with pkg/media and pkg/media/encoder (CreateEncode / CreateDecode, registry).
// Do not duplicate G.711/Opus/G.722 logic inside pkg/sip.
//
// Inbound wiring (manual): before your INVITE handler, transaction.Manager.HandleInviteRequest;
// after sending a final, BeginInviteServer; on ACK, HandleAck.
//
// Composable helpers (see server_tx.go): ChainInviteServerTx, AfterResponseSentBeginInviteServer,
// WithOnResponseSentAppended, ChainAckServerTx — wire mgr + stack.EndpointConfig.OnResponseSent + Handlers.
//
// See also: pkg/sip/dialog, pkg/sip/session, pkg/sip/transaction (Register stack.MethodCancel → HandleCancelRequest).
package uas
