package protocol

import (
	"net"
)

// EventType describes a SIP protocol event emitted by the server.
type EventType string

const (
	EventDatagramReceived   EventType = "sip.datagram.received"
	EventParseError         EventType = "sip.parse.error"
	EventRequestReceived    EventType = "sip.request.received"
	EventResponseReceived   EventType = "sip.response.received"
	EventResponseSent       EventType = "sip.response.sent"
)

type Event struct {
	Type EventType

	Addr *net.UDPAddr

	Raw []byte

	Request  *Message
	Response *Message

	Err error
}

