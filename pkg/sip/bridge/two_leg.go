package bridge

import (
	"context"
	"fmt"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/media/encoder"
	"github.com/LingByte/SoulNexus/pkg/sip/rtp"
)

// TwoLegPCMBridge forwards audio between two SIP legs at PCM (16 kHz mono) in the middle.
// Uses four half-duplex SIPRTPTransport instances (caller rx/tx + agent rx/tx).
type TwoLegPCMBridge struct {
	ctx    context.Context
	cancel context.CancelFunc

	callerToAgent *media.MediaSession
	agentToCaller *media.MediaSession

	startOnce sync.Once
	stopOnce  sync.Once
}

// NewTwoLegPCMBridge builds a bidirectional bridge. Transports must use the same codec config
// as their respective RTP sessions (callerRx/callerTx share one session; agentRx/agentTx share another).
func NewTwoLegPCMBridge(
	callerRx, callerTx, agentRx, agentTx *rtp.SIPRTPTransport,
) (*TwoLegPCMBridge, error) {
	if callerRx == nil || callerTx == nil || agentRx == nil || agentTx == nil {
		return nil, fmt.Errorf("bridge: nil transport")
	}

	codecCaller := callerRx.Codec()
	codecAgent := agentRx.Codec()

	pcm := media.CodecConfig{
		Codec:      "pcm",
		SampleRate: 16000,
		Channels:   1,
		BitDepth:   16,
	}

	decCaller, err := encoder.CreateDecode(codecCaller, pcm)
	if err != nil {
		return nil, fmt.Errorf("bridge: decode caller: %w", err)
	}
	encAgent, err := encoder.CreateEncode(codecAgent, pcm)
	if err != nil {
		return nil, fmt.Errorf("bridge: encode agent: %w", err)
	}

	decAgent, err := encoder.CreateDecode(codecAgent, pcm)
	if err != nil {
		return nil, fmt.Errorf("bridge: decode agent: %w", err)
	}
	encCaller, err := encoder.CreateEncode(codecCaller, pcm)
	if err != nil {
		return nil, fmt.Errorf("bridge: encode caller: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	cToA := media.NewDefaultSession().
		Context(ctx).
		SetSessionID("sip-bridge-caller-to-agent").
		Decode(decCaller).
		Encode(encAgent).
		Input(callerRx).
		Output(agentTx)

	aToC := media.NewDefaultSession().
		Context(ctx).
		SetSessionID("sip-bridge-agent-to-caller").
		Decode(decAgent).
		Encode(encCaller).
		Input(agentRx).
		Output(callerTx)

	return &TwoLegPCMBridge{
		ctx:             ctx,
		cancel:          cancel,
		callerToAgent:   cToA,
		agentToCaller:   aToC,
	}, nil
}

// Start runs both bridge directions (non-blocking).
func (b *TwoLegPCMBridge) Start() {
	if b == nil {
		return
	}
	b.startOnce.Do(func() {
		go func() { _ = b.callerToAgent.Serve() }()
		go func() { _ = b.agentToCaller.Serve() }()
	})
}

// Stop tears down both MediaSessions and closes RTP via transport Close (unless preserved).
func (b *TwoLegPCMBridge) Stop() {
	if b == nil {
		return
	}
	b.stopOnce.Do(func() {
		if b.cancel != nil {
			b.cancel()
		}
		if b.callerToAgent != nil {
			_ = b.callerToAgent.Close()
		}
		if b.agentToCaller != nil {
			_ = b.agentToCaller.Close()
		}
	})
}
