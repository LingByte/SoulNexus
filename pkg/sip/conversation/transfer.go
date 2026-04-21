package conversation

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/media"
	sipdtmf "github.com/LingByte/SoulNexus/pkg/sip/dtmf"
	"github.com/LingByte/SoulNexus/pkg/sip/outbound"
	"github.com/LingByte/SoulNexus/pkg/sip/scriptlisten"
	sipSession "github.com/LingByte/SoulNexus/pkg/sip/session"
	"github.com/LingByte/SoulNexus/pkg/sip/webseat"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"go.uber.org/zap"
)

// TransferDialer is implemented by outbound.Manager (Dial).
type TransferDialer interface {
	Dial(ctx context.Context, req outbound.DialRequest) (callID string, err error)
}

var (
	transferMu     sync.Mutex
	transferDialer TransferDialer
	// Optional: DB-backed dial target (e.g. sip_users) tried before TransferDialTargetFromEnv.
	// inboundCallID is the PSTN inbound Call-ID (used to bind Web ACD rows to this call).
	transferDialTarget func(context.Context, string) (outbound.DialTarget, bool)
	// WebSeatTransfer starts inbound ↔ browser WebRTC bridging when DialTarget.WebSeat (SIP_TRANSFER_NUMBER=web).
	// If nil and WebSeat is requested, transfer logs a warning and releases the dedupe slot.
	webSeatTransfer  func(inboundCallID string, lg *zap.Logger)
	transferStarted  sync.Map // inbound Call-ID -> bool (dedupe)
	transferRingMu   sync.Mutex
	transferRingStop map[string]context.CancelFunc
)

// SetTransferDialer wires the outbound module (call from cmd/sip after creating outbound.Manager).
func SetTransferDialer(d TransferDialer) {
	transferMu.Lock()
	defer transferMu.Unlock()
	transferDialer = d
}

// SetTransferDialTargetResolver sets an optional resolver (e.g. DB lookup by SIP_TRANSFER_NUMBER).
// When it returns ok=false, outbound.TransferDialTargetFromEnv is used.
func SetTransferDialTargetResolver(fn func(context.Context, string) (outbound.DialTarget, bool)) {
	transferMu.Lock()
	defer transferMu.Unlock()
	transferDialTarget = fn
}

// SetWebSeatTransfer registers the handler for SIP_TRANSFER_NUMBER=web (browser agent). Optional until WebRTC gateway ships.
func SetWebSeatTransfer(fn func(inboundCallID string, lg *zap.Logger)) {
	transferMu.Lock()
	defer transferMu.Unlock()
	webSeatTransfer = fn
}

// HandleSIPINFODTMF parses SIP INFO (application/dtmf-relay). In script mode, digits wake listen waiters.
func HandleSIPINFODTMF(inboundCallID string, contentType, body string, lg *zap.Logger) {
	if lg == nil && logger.Lg != nil {
		lg = logger.Lg
	}
	if lg == nil {
		lg = zap.NewNop()
	}
	if d, ok := sipdtmf.DigitFromSIPINFO(contentType, body); ok && isSIPScriptMode(inboundCallID) {
		scriptlisten.PublishDTMF(inboundCallID, d)
		lg.Info("sip info dtmf (script)",
			zap.String("call_id", inboundCallID),
			zap.String("digit", d),
		)
		return
	}
	lg.Info("sip info received (dtmf transfer disabled)",
		zap.String("call_id", inboundCallID),
		zap.String("content_type", strings.TrimSpace(contentType)),
		zap.Int("body_len", len(strings.TrimSpace(body))),
	)
}

// TriggerTransferToAgent starts transfer for an inbound call (AI/tool/fallback text).
func TriggerTransferToAgent(ctx context.Context, inboundCallID string, lg *zap.Logger) {
	transferMu.Lock()
	d := transferDialer
	resolveTgt := transferDialTarget
	webFn := webSeatTransfer
	transferMu.Unlock()

	var tgt outbound.DialTarget
	var ok bool
	if resolveTgt != nil {
		tgt, ok = resolveTgt(ctx, inboundCallID)
	}
	// When cmd/sip wires a DB resolver, targets come only from acd_pool_targets — do not fall back to SIP_TRANSFER_* env.
	if !ok && resolveTgt == nil {
		tgt, ok = outbound.TransferDialTargetFromEnv()
	}
	if !ok {
		if resolveTgt != nil {
			lg.Warn("sip transfer: no eligible acd_pool_targets row (need weight>0, work_state=available, route sip|web; trunk must have dial target + gateway env; web seat needs fresh heartbeat)")
		} else {
			lg.Warn("sip transfer: configure database for cmd/sip (ACD pool), or set SIP_TRANSFER_REQUEST_URI + SIP_TRANSFER_SIGNALING_ADDR, or SIP_TRANSFER_NUMBER + SIP_TRANSFER_HOST (web for browser agent)")
		}
		go playNoSeatGoodbyeAndHangup(ctx, inboundCallID, lg)
		return
	}

	if _, loaded := transferStarted.LoadOrStore(inboundCallID, true); loaded {
		lg.Info("sip transfer: already started for this call", zap.String("call_id", inboundCallID))
		return
	}

	if tgt.WebSeat {
		if webFn == nil {
			lg.Warn("sip transfer: WebSeat (SIP_TRANSFER_NUMBER=web) but SetWebSeatTransfer not configured")
			webseat.ReleaseInboundWebACDOffer(inboundCallID)
			transferStarted.Delete(inboundCallID)
			return
		}
		lg.Info("sip transfer: web seat — handing off to WebRTC bridge", zap.String("inbound_call_id", inboundCallID))
		startTransferRinging(ctx, inboundCallID, lg)
		go func() { webFn(inboundCallID, lg) }()
		return
	}

	if d == nil {
		lg.Warn("sip transfer: no TransferDialer (SetTransferDialer not called)")
		transferStarted.Delete(inboundCallID)
		return
	}

	lg.Info("sip transfer: dialing agent leg", zap.String("inbound_call_id", inboundCallID), zap.String("agent_uri", tgt.RequestURI))
	startTransferRinging(ctx, inboundCallID, lg)

	go func() {
		cid, err := d.Dial(ctx, outbound.DialRequest{
			Scenario:      outbound.ScenarioTransferAgent,
			Target:        tgt,
			CorrelationID: inboundCallID,
			MediaProfile:  outbound.MediaProfileTransferBridge,
		})
		if err != nil {
			stopTransferRinging(inboundCallID)
			transferStarted.Delete(inboundCallID)
			lg.Warn("sip transfer: outbound dial failed", zap.String("inbound_call_id", inboundCallID), zap.Error(err))
			return
		}
		lg.Info("sip transfer: agent leg INVITE sent", zap.String("inbound_call_id", inboundCallID), zap.String("outbound_call_id", cid))
	}()
}

func startTransferRinging(ctx context.Context, inboundCallID string, lg *zap.Logger) {
	inbound := lookupInboundSession(inboundCallID)
	if inbound == nil {
		return
	}
	stopTransferRinging(inboundCallID)

	runCtx, cancel := context.WithCancel(ctx)
	transferRingMu.Lock()
	if transferRingStop == nil {
		transferRingStop = make(map[string]context.CancelFunc)
	}
	transferRingStop[inboundCallID] = cancel
	transferRingMu.Unlock()

	go func() {
		defer stopTransferRinging(inboundCallID)
		if err := playTransferRingingLoop(runCtx, inbound, lg); err != nil && !errorsIsCtxDone(err) {
			lg.Warn("sip transfer ring playback failed", zap.String("inbound_call_id", inboundCallID), zap.Error(err))
		}
	}()
}

func stopTransferRinging(inboundCallID string) {
	transferRingMu.Lock()
	cancel := transferRingStop[inboundCallID]
	delete(transferRingStop, inboundCallID)
	transferRingMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func lookupInboundSession(callID string) *sipSession.CallSession {
	if lookupInbound == nil {
		return nil
	}
	return lookupInbound(callID)
}

func playTransferRingingLoop(ctx context.Context, inbound *sipSession.CallSession, lg *zap.Logger) error {
	if inbound == nil {
		return fmt.Errorf("nil inbound session")
	}
	ms := inbound.MediaSession()
	if ms == nil {
		return fmt.Errorf("nil inbound media session")
	}
	path := strings.TrimSpace(utils.GetEnv("SIP_TRANSFER_RINGING_WAV_PATH"))
	if path == "" {
		path = "scripts/ringing.wav"
	}
	if !filepath.IsAbs(path) {
		path = filepath.Clean(path)
	}
	pcm, err := loadWAVAsPCM16Mono(path, 16000)
	if err != nil {
		return fmt.Errorf("load transfer ringing wav: %w", err)
	}
	bytesPerFrame := 16000 * 2 * 20 / 1000
	if bytesPerFrame <= 0 {
		bytesPerFrame = 640
	}
	const maxRingDuration = 35 * time.Second
	deadline := time.Now().Add(maxRingDuration)
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	if lg != nil {
		lg.Info("sip transfer ring playback started", zap.Int("bytes", len(pcm)))
	}
	offset := 0
	for {
		if time.Now().After(deadline) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ms.GetContext().Done():
			return ms.GetContext().Err()
		case <-ticker.C:
		}
		if ActiveTransferBridgeForCallID(inbound.CallID) || ActiveWebSeatBridge(inbound.CallID) {
			return nil
		}
		end := offset + bytesPerFrame
		if end > len(pcm) {
			end = len(pcm)
		}
		frame := pcm[offset:end]
		if len(frame) > 0 {
			ms.SendToOutput("sip-transfer-ringing", &media.AudioPacket{
				Payload:       frame,
				IsSynthesized: true,
			})
		}
		offset = end
		if offset >= len(pcm) {
			offset = 0
		}
	}
}

func errorsIsCtxDone(err error) bool {
	return err == context.Canceled || err == context.DeadlineExceeded
}

func playNoSeatGoodbyeAndHangup(ctx context.Context, inboundCallID string, lg *zap.Logger) {
	inbound := lookupInboundSession(inboundCallID)
	if inbound == nil {
		RequestSIPHangup(inboundCallID)
		return
	}
	ms := inbound.MediaSession()
	if ms == nil {
		RequestSIPHangup(inboundCallID)
		return
	}
	path := "scripts/goodbye.wav"
	if !filepath.IsAbs(path) {
		path = filepath.Clean(path)
	}
	pcm, err := loadWAVAsPCM16Mono(path, 16000)
	if err != nil {
		if lg != nil {
			lg.Warn("sip transfer: load goodbye wav failed, hangup directly",
				zap.String("inbound_call_id", inboundCallID),
				zap.Error(err))
		}
		RequestSIPHangup(inboundCallID)
		return
	}
	runCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	bytesPerFrame := 16000 * 2 * 20 / 1000
	if bytesPerFrame <= 0 {
		bytesPerFrame = 640
	}
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for off := 0; off < len(pcm); off += bytesPerFrame {
		select {
		case <-runCtx.Done():
			RequestSIPHangup(inboundCallID)
			return
		case <-ms.GetContext().Done():
			RequestSIPHangup(inboundCallID)
			return
		case <-ticker.C:
		}
		end := off + bytesPerFrame
		if end > len(pcm) {
			end = len(pcm)
		}
		frame := pcm[off:end]
		if len(frame) == 0 {
			continue
		}
		ms.SendToOutput("sip-transfer-no-seat-goodbye", &media.AudioPacket{
			Payload:       frame,
			IsSynthesized: true,
		})
	}
	RequestSIPHangup(inboundCallID)
}
