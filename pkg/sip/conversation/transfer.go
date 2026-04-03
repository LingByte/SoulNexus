package conversation

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/sip/dtmf"
	"github.com/LingByte/SoulNexus/pkg/sip/outbound"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"go.uber.org/zap"
)

// TransferDialer is implemented by outbound.Manager (Dial).
type TransferDialer interface {
	Dial(ctx context.Context, req outbound.DialRequest) (callID string, err error)
}

var (
	transferMu       sync.Mutex
	transferDialer   TransferDialer
	// Optional: DB-backed dial target (e.g. sip_users) tried before TransferDialTargetFromEnv.
	transferDialTarget func(context.Context) (outbound.DialTarget, bool)
	transferStarted    sync.Map // inbound Call-ID -> bool (dedupe)
	// RFC2833 often emits several events per keypress; ignore repeats within this window.
	transferDTMFLast sync.Map // inbound Call-ID -> time.Time
)

const transferDTMFDebounce = 450 * time.Millisecond

func transferDTMFAllow(inboundCallID string) bool {
	now := time.Now()
	if t, ok := transferDTMFLast.Load(inboundCallID); ok {
		if now.Sub(t.(time.Time)) < transferDTMFDebounce {
			return false
		}
	}
	transferDTMFLast.Store(inboundCallID, now)
	return true
}

// SetTransferDialer wires the outbound module (call from cmd/sip after creating outbound.Manager).
func SetTransferDialer(d TransferDialer) {
	transferMu.Lock()
	defer transferMu.Unlock()
	transferDialer = d
}

// SetTransferDialTargetResolver sets an optional resolver (e.g. DB lookup by SIP_TRANSFER_NUMBER).
// When it returns ok=false, TransferDialTargetFromEnv is used.
func SetTransferDialTargetResolver(fn func(context.Context) (outbound.DialTarget, bool)) {
	transferMu.Lock()
	defer transferMu.Unlock()
	transferDialTarget = fn
}

// HandleSIPINFODTMF parses SIP INFO (application/dtmf-relay) and triggers transfer when digit matches.
func HandleSIPINFODTMF(ctx context.Context, inboundCallID string, contentType, body string, lg *zap.Logger) {
	if lg == nil && logger.Lg != nil {
		lg = logger.Lg
	}
	if lg == nil {
		lg = zap.NewNop()
	}
	digit, ok := dtmf.DigitFromSIPINFO(contentType, body)
	if !ok {
		return
	}
	lg.Info("sip info dtmf", zap.String("call_id", inboundCallID), zap.String("digit", digit))
	tryTransferToAgent(ctx, inboundCallID, digit, lg)
}

func tryTransferToAgent(ctx context.Context, inboundCallID, digit string, lg *zap.Logger) {
	want := strings.TrimSpace(utils.GetEnv("SIP_TRANSFER_TO_AGENT_DIGIT"))
	if want == "" {
		want = "0"
	}
	if digit != want {
		return
	}
	if !transferDTMFAllow(inboundCallID) {
		return
	}

	transferMu.Lock()
	d := transferDialer
	resolveTgt := transferDialTarget
	transferMu.Unlock()
	if d == nil {
		lg.Warn("sip transfer: no TransferDialer (SetTransferDialer not called)")
		return
	}

	var tgt outbound.DialTarget
	var ok bool
	if resolveTgt != nil {
		tgt, ok = resolveTgt(ctx)
	}
	if !ok {
		tgt, ok = outbound.TransferDialTargetFromEnv()
	}
	if !ok {
		lg.Warn("sip transfer: set SIP_TRANSFER_REQUEST_URI + SIP_TRANSFER_SIGNALING_ADDR, or SIP_TRANSFER_NUMBER with DB (SIP_DEFAULT_DOMAIN) / SIP_TRANSFER_HOST")
		return
	}

	if _, loaded := transferStarted.LoadOrStore(inboundCallID, true); loaded {
		lg.Info("sip transfer: already started for this call", zap.String("call_id", inboundCallID))
		return
	}

	lg.Info("sip transfer: dialing agent leg", zap.String("inbound_call_id", inboundCallID), zap.String("agent_uri", tgt.RequestURI))

	go func() {
		cid, err := d.Dial(ctx, outbound.DialRequest{
			Scenario:      outbound.ScenarioTransferAgent,
			Target:        tgt,
			CorrelationID: inboundCallID,
			MediaProfile:  outbound.MediaProfileBridgePCM,
		})
		if err != nil {
			transferStarted.Delete(inboundCallID)
			lg.Warn("sip transfer: outbound dial failed", zap.String("inbound_call_id", inboundCallID), zap.Error(err))
			return
		}
		lg.Info("sip transfer: agent leg INVITE sent", zap.String("inbound_call_id", inboundCallID), zap.String("outbound_call_id", cid))
	}()
}
