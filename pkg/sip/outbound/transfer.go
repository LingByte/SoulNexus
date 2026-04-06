package outbound

import (
	"context"

	sipSession "github.com/LingByte/SoulNexus/pkg/sip/session"
)

// TransferCoordinator bridges an inbound caller to an agent reached via a second SIP leg.
// Transfer: outbound Dial with MediaProfileTransferBridge + pkg/sip/bridge between two RTP legs.
type TransferCoordinator interface {
	// TransferToAgent places an outbound call to the agent and connects audio to the inbound leg.
	// inbound is the existing AI (or PSTN) CallSession; agent describes the agent SIP destination.
	TransferToAgent(ctx context.Context, inbound *sipSession.CallSession, agent DialTarget) (TransferHandle, error)
}

// TransferHandle closes the bridge and optional legs.
type TransferHandle interface {
	Done() <-chan struct{}
	Close(ctx context.Context) error
}

// NoopTransferCoordinator returns ErrNotImplemented for all operations (placeholder).
type NoopTransferCoordinator struct{}

func (NoopTransferCoordinator) TransferToAgent(ctx context.Context, inbound *sipSession.CallSession, agent DialTarget) (TransferHandle, error) {
	return nil, ErrNotImplemented
}
