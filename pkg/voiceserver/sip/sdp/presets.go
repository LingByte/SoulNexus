package sdp

// DefaultOutboundOfferCodecs is the standard outbound INVITE audio preference list:
// PCMA first for carrier interoperability, then PCMU, G.722, Opus, telephone-event.
func DefaultOutboundOfferCodecs() []Codec {
	return []Codec{
		{PayloadType: 8, Name: "pcma", ClockRate: 8000, Channels: 1},
		{PayloadType: 0, Name: "pcmu", ClockRate: 8000, Channels: 1},
		{PayloadType: 9, Name: "g722", ClockRate: 8000, Channels: 1},
		{PayloadType: 111, Name: "opus", ClockRate: 48000, Channels: 1},
		{PayloadType: 101, Name: "telephone-event", ClockRate: 8000, Channels: 1},
	}
}

// TransferAgentBridgeOfferCodecs is the INVITE offer for the human/agent leg after transfer
// (narrowband-first to simplify PCM bridging).
func TransferAgentBridgeOfferCodecs() []Codec {
	return []Codec{
		{PayloadType: 8, Name: "pcma", ClockRate: 8000, Channels: 1},
		{PayloadType: 0, Name: "pcmu", ClockRate: 8000, Channels: 1},
		{PayloadType: 101, Name: "telephone-event", ClockRate: 8000, Channels: 1},
	}
}
