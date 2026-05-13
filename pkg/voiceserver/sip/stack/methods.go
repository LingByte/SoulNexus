package stack

// SIP method names (RFC 3261 and common extensions), upper-case as on the wire.
//
// IANA registry: https://www.iana.org/assignments/sip-methods/sip-methods.xhtml
// Not every deployment uses all of these; stack only needs constants for registration and builders.
const (
	// RFC 3261 core
	MethodInvite   = "INVITE"
	MethodAck      = "ACK"
	MethodBye      = "BYE"
	MethodCancel   = "CANCEL"
	MethodOptions  = "OPTIONS"
	MethodRegister = "REGISTER"
	// RFC 3262 reliable provisional responses
	MethodPrack = "PRACK"
	// RFC 3265 / 6665 event state; RFC 3856 presence
	MethodSubscribe = "SUBSCRIBE"
	MethodNotify    = "NOTIFY"
	MethodPublish   = "PUBLISH"
	// RFC 6086 session-info; RFC 3515 transfer; RFC 3428 instant messages
	MethodInfo    = "INFO"
	MethodRefer   = "REFER"
	MethodMessage = "MESSAGE"
	// RFC 3311 mid-dialog parameter refresh
	MethodUpdate = "UPDATE"
)
