package bridge

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/media/encoder"
)

// pcmBridgeLeg is the minimal read/write + codec surface for PCM bridging (SIP RTP or WebRTC).
type pcmBridgeLeg interface {
	Next(ctx context.Context) (media.MediaPacket, error)
	Send(ctx context.Context, p media.MediaPacket) (int, error)
	Codec() media.CodecConfig
	WakeupRead()
}

// TwoLegPCMBridge transcodes between two SIP legs. Transfer agent leg is PCMU/8k; inbound may be Opus, G.722, etc.
// Mid PCM is 8 kHz mono for dual G.711, otherwise 16 kHz mono (typical Opus/PCMU bridge).
type TwoLegPCMBridge struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	callerRx, callerTx, agentRx, agentTx pcmBridgeLeg
	c2aDec, c2aEnc, a2cDec, a2cEnc       media.EncoderFunc

	startOnce sync.Once
	stopOnce  sync.Once
}

// NewTwoLegPCMBridge builds a bidirectional bridge. Transports must use the same codec config
// as their respective RTP sessions (callerRx/callerTx share one session; agentRx/agentTx share another).
func NewTwoLegPCMBridge(
	callerRx, callerTx, agentRx, agentTx pcmBridgeLeg,
) (*TwoLegPCMBridge, error) {
	if callerRx == nil || callerTx == nil || agentRx == nil || agentTx == nil {
		return nil, fmt.Errorf("bridge: nil transport")
	}

	codecCaller := callerRx.Codec()
	codecAgent := agentRx.Codec()
	codecCaller = opusBridgeDecodeConfig(codecCaller)
	codecAgent = opusBridgeDecodeConfig(codecAgent)

	pcm := bridgeMidPCM(codecCaller, codecAgent)

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

	return &TwoLegPCMBridge{
		ctx:      ctx,
		cancel:   cancel,
		callerRx: callerRx,
		callerTx: callerTx,
		agentRx:  agentRx,
		agentTx:  agentTx,
		c2aDec:   decCaller,
		c2aEnc:   encAgent,
		a2cDec:   decAgent,
		a2cEnc:   encCaller,
	}, nil
}

func runPCMBridgeHalf(ctx context.Context, rx, tx pcmBridgeLeg, dec, enc media.EncoderFunc) {
	if rx == nil || tx == nil || dec == nil || enc == nil {
		return
	}
	for ctx.Err() == nil {
		pkt, err := rx.Next(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		if pkt == nil {
			continue
		}
		dps, err := dec(pkt)
		if err != nil {
			continue
		}
		for _, dp := range dps {
			if dp == nil {
				continue
			}
			eps, err := enc(dp)
			if err != nil {
				continue
			}
			for _, ep := range eps {
				if ep == nil {
					continue
				}
				_, _ = tx.Send(ctx, ep)
			}
		}
	}
}

// opusBridgeDecodeConfig: inbound Opus/48000/2 needs stereo decode before downmix to mono bridge PCM.
func opusBridgeDecodeConfig(c media.CodecConfig) media.CodecConfig {
	if strings.EqualFold(strings.TrimSpace(c.Codec), "opus") && c.OpusDecodeChannels == 2 {
		c.OpusPCMBridgeDecodeStereo = true
	}
	return c
}

func narrowbandG711(c media.CodecConfig) bool {
	n := strings.ToLower(strings.TrimSpace(c.Codec))
	return (n == "pcmu" || n == "pcma") && c.SampleRate == 8000
}

// bridgeMidPCM: dual narrowband G.711 uses 8 kHz PCM; all other pairs (e.g. Opus + PCMU) use 16 kHz mono.
func bridgeMidPCM(caller, agent media.CodecConfig) media.CodecConfig {
	if narrowbandG711(caller) && narrowbandG711(agent) {
		return media.CodecConfig{Codec: "pcm", SampleRate: 8000, Channels: 1, BitDepth: 16}
	}
	return media.CodecConfig{Codec: "pcm", SampleRate: 16000, Channels: 1, BitDepth: 16}
}

// Start runs both bridge directions (non-blocking).
func (b *TwoLegPCMBridge) Start() {
	if b == nil {
		return
	}
	b.startOnce.Do(func() {
		b.wg.Add(2)
		go func() {
			defer b.wg.Done()
			runPCMBridgeHalf(b.ctx, b.callerRx, b.agentTx, b.c2aDec, b.c2aEnc)
		}()
		go func() {
			defer b.wg.Done()
			runPCMBridgeHalf(b.ctx, b.agentRx, b.callerTx, b.a2cDec, b.a2cEnc)
		}()
	})
}

// Stop cancels the bridge and unblocks RTP reads; RTP sockets are closed by the transfer teardown path.
func (b *TwoLegPCMBridge) Stop() {
	if b == nil {
		return
	}
	b.stopOnce.Do(func() {
		if b.cancel != nil {
			b.cancel()
		}
		if b.callerRx != nil {
			b.callerRx.WakeupRead()
		}
		if b.agentRx != nil {
			b.agentRx.WakeupRead()
		}
		b.wg.Wait()
	})
}
