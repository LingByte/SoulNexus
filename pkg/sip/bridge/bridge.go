package bridge

import (
	"context"
	"fmt"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/media/encoder"
)

// Bridge wires two MediaTransports together bidirectionally using two MediaSessions:
// - SIP (or any RTP transport) -> WebRTC (or any other transport)
// - WebRTC -> SIP
//
// This avoids output broadcasting loops and keeps routing simple:
// each MediaSession has exactly ONE input and ONE output transport.
//
// The Bridge operates at PCM level:
// input codec -> PCM(16k/16bit/mono) -> output codec.
type Bridge struct {
	ctx    context.Context
	cancel context.CancelFunc

	aToB *media.MediaSession
	bToA *media.MediaSession

	startOnce sync.Once
	stopOnce  sync.Once
}

type Config struct {
	// A is typically SIP RTP transport, B is typically WebRTC RTP transport,
	// but both just need to implement media.MediaTransport.
	A media.MediaTransport
	B media.MediaTransport

	CodecA media.CodecConfig
	CodecB media.CodecConfig

	// Optional: override internal PCM format (defaults to 16k/16bit/mono)
	PCM media.CodecConfig

	// Optional IDs for debugging/metrics
	ID string
}

func New(cfg Config) (*Bridge, error) {
	if cfg.A == nil || cfg.B == nil {
		return nil, fmt.Errorf("bridge: A/B transport is nil")
	}

	pcm := cfg.PCM
	if pcm.Codec == "" {
		pcm = media.CodecConfig{
			Codec:      "pcm",
			SampleRate: 16000,
			Channels:   1,
			BitDepth:   16,
		}
	}

	decA, err := encoder.CreateDecode(cfg.CodecA, pcm)
	if err != nil {
		return nil, fmt.Errorf("bridge: create decode A->PCM: %w", err)
	}
	encB, err := encoder.CreateEncode(cfg.CodecB, pcm)
	if err != nil {
		return nil, fmt.Errorf("bridge: create encode PCM->B: %w", err)
	}

	decB, err := encoder.CreateDecode(cfg.CodecB, pcm)
	if err != nil {
		return nil, fmt.Errorf("bridge: create decode B->PCM: %w", err)
	}
	encA, err := encoder.CreateEncode(cfg.CodecA, pcm)
	if err != nil {
		return nil, fmt.Errorf("bridge: create encode PCM->A: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	id := cfg.ID
	if id == "" {
		id = "sip-webrtc-bridge"
	}

	aToB := media.NewDefaultSession().
		Context(ctx).
		SetSessionID(id + ":A->B").
		Decode(decA).
		Encode(encB).
		Input(cfg.A).
		Output(cfg.B)

	bToA := media.NewDefaultSession().
		Context(ctx).
		SetSessionID(id + ":B->A").
		Decode(decB).
		Encode(encA).
		Input(cfg.B).
		Output(cfg.A)

	return &Bridge{
		ctx:    ctx,
		cancel: cancel,
		aToB:   aToB,
		bToA:   bToA,
	}, nil
}

func (b *Bridge) Start() {
	if b == nil {
		return
	}
	b.startOnce.Do(func() {
		go func() { _ = b.aToB.Serve() }()
		go func() { _ = b.bToA.Serve() }()
	})
}

func (b *Bridge) Stop() {
	if b == nil {
		return
	}
	b.stopOnce.Do(func() {
		if b.cancel != nil {
			b.cancel()
		}
		if b.aToB != nil {
			_ = b.aToB.Close()
		}
		if b.bToA != nil {
			_ = b.bToA.Close()
		}
	})
}

