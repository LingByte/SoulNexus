// Package dialog tracks minimal SIP dialog state for pkg/sip1 (Call-ID, tags, early/confirmed).
//
// It aligns with transaction keys via InviteTransactionKey (branch + Call-ID from the INVITE).
// It does not parse SDP or touch RTP; use pkg/sip1/session for media.
package dialog
