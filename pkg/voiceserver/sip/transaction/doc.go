// Package transaction provides SIP UDP transaction helpers layered under pkg/sip/stack.Endpoint.
//
// UAC (client):
//   - INVITE client: RunInviteClient, HandleResponse (wire from stack.Endpoint.OnSIPResponse), ACK helpers.
//
// UAS INVITE (server):
//   - RegisterPendingInviteServer when an INVITE arrives before the final; ClearPendingInviteServer or let
//     BeginInviteServer clear it after a final.
//   - HandleCancelRequest for CANCEL matching pending Call-ID + CSeq (sends 200 to CANCEL); TU still sends 487 (or similar) on INVITE.
//   - After sending a final on the wire: BeginInviteServer; duplicate INVITE: HandleInviteRequest; 2xx: HandleAck.
//
// UAS non-INVITE (OPTIONS, REGISTER, …):
//   - After sending a final: BeginNonInviteServer; duplicate request: HandleNonInviteRequest (Timer J window).
//
// Keys: InviteTransactionKey(branch, Call-ID); NonInviteServerKey(req) for non-INVITE map.
//
// TCP/TLS signaling is handled at pkg/sip/server (per-connection framing via stack.ReadMessage);
// this package still lacks RFC 3261 connection reuse / outbound TX mirroring for TCP.
//
// Not implemented: forked responses, full PRACK/re-INVITE state machines beyond invite_rfc3262 hooks,
// every non-INVITE server-transaction edge case.
package transaction
