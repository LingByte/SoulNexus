package voiceattach

import (
	"context"
	"fmt"

	dialogrealtime "github.com/LingByte/SoulNexus/pkg/dialog/realtime"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/lingllm/realtime"
	"go.uber.org/zap"
)

// nativeRealtimeBuilderConfig closes over tenant credentials for one call.
type nativeRealtimeBuilderConfig struct {
	CredentialCfg      map[string]any
	Options            realtime.Options
	Session            *nativeRealtimeSession
	MediaCtx           context.Context
	ObserveVendorEvent func(realtime.Event)
}

// wireNativeRealtimeToolsAndTransfer formerly registered human transfer
// tools. With telephony removed it only returns the session vendor observer.
func wireNativeRealtimeToolsAndTransfer(
	callID string,
	_ tenantcfg.VoiceEnv,
	_ *realtime.Options,
	_ *zap.Logger,
) func(realtime.Event) {
	sess := lookupNativeRealtimeSession(callID)
	return func(ev realtime.Event) {
		if sess != nil {
			sess.observeVendorEvent(ev)
		}
	}
}

func newNativeRealtimeBuilder(cfg nativeRealtimeBuilderConfig) dialogrealtime.AgentBuilder {
	return dialogrealtime.AgentBuilderFunc(func(sink dialogrealtime.EventSink) (dialogrealtime.Agent, error) {
		if sink == nil {
			return nil, fmt.Errorf("voiceattach: nil EventSink")
		}
		opts := cfg.Options
		outSR := opts.OutputSampleRate
		if outSR <= 0 {
			outSR = 24000
		}
		observe := cfg.ObserveVendorEvent
		opts.OnEvent = func(ev realtime.Event) {
			if observe != nil {
				observe(ev)
			}
			_ = sink.Emit(mapLingllmRealtimeEvent(ev, outSR))
		}
		cred := cfg.CredentialCfg
		if cred == nil {
			cred = map[string]any{}
		}
		inner, err := realtime.NewAgentFromCredential(cred, opts)
		if err != nil {
			return nil, err
		}
		return &nativeRealtimeAgentAdapter{
			inner:   inner,
			session: cfg.Session,
			mediaCtx: cfg.MediaCtx,
		}, nil
	})
}

type nativeRealtimeAgentAdapter struct {
	inner    realtime.Agent
	session  *nativeRealtimeSession
	mediaCtx context.Context
}

func (a *nativeRealtimeAgentAdapter) Start(ctx context.Context) error {
	if a == nil || a.inner == nil {
		return fmt.Errorf("voiceattach: nil realtime agent")
	}
	if err := a.inner.Start(ctx); err != nil {
		return err
	}
	if a.session != nil {
		mc := a.mediaCtx
		if mc == nil {
			mc = ctx
		}
		a.session.onAgentStarted(a.inner, mc)
	}
	return nil
}

func (a *nativeRealtimeAgentAdapter) PushAudio(pcm []byte) error {
	if a == nil || a.inner == nil {
		return fmt.Errorf("voiceattach: nil realtime agent")
	}
	return a.inner.PushAudio(pcm)
}

func (a *nativeRealtimeAgentAdapter) Cancel() error {
	if a == nil || a.inner == nil {
		return nil
	}
	return a.inner.Cancel()
}

func (a *nativeRealtimeAgentAdapter) Close() error {
	if a == nil || a.inner == nil {
		return nil
	}
	return a.inner.Close()
}

func (a *nativeRealtimeAgentAdapter) CreateResponse(instructions string) error {
	if a == nil || a.inner == nil {
		return fmt.Errorf("voiceattach: nil realtime agent")
	}
	if starter, ok := a.inner.(dialogrealtime.ResponseStarter); ok {
		return starter.CreateResponse(instructions)
	}
	return fmt.Errorf("voiceattach: agent does not support CreateResponse")
}

func (a *nativeRealtimeAgentAdapter) CommitInputAudio() error {
	if a == nil || a.inner == nil {
		return nil
	}
	return a.inner.CommitInputAudio()
}

func (a *nativeRealtimeAgentAdapter) ClearInputAudio() error {
	if a == nil || a.inner == nil {
		return nil
	}
	if clearer, ok := a.inner.(dialogrealtime.InputBufferClearer); ok {
		return clearer.ClearInputAudio()
	}
	return nil
}

func (a *nativeRealtimeAgentAdapter) EnableServerVAD() error {
	if a == nil || a.inner == nil {
		return nil
	}
	if enabler, ok := a.inner.(dialogrealtime.ServerVADEnabler); ok {
		return enabler.EnableServerVAD()
	}
	return nil
}

func (a *nativeRealtimeAgentAdapter) UpdateInstructions(instructions string) error {
	if a == nil || a.inner == nil {
		return nil
	}
	return a.inner.UpdateInstructions(instructions)
}

func mapLingllmRealtimeEvent(ev realtime.Event, outSR int) dialogrealtime.Event {
	switch ev.Type {
	case realtime.EventUserTranscript:
		return dialogrealtime.Event{Kind: dialogrealtime.EventUserTranscript, Text: ev.Text, Final: ev.Final, Vendor: ev.Vendor}
	case realtime.EventUserSpeechStarted:
		return dialogrealtime.Event{Kind: dialogrealtime.EventUserSpeechStarted, Vendor: ev.Vendor}
	case realtime.EventUserSpeechEnded:
		return dialogrealtime.Event{Kind: dialogrealtime.EventUserSpeechEnded, Vendor: ev.Vendor}
	case realtime.EventAssistantText:
		return dialogrealtime.Event{Kind: dialogrealtime.EventAssistantText, Text: ev.Text, Final: ev.Final, Vendor: ev.Vendor}
	case realtime.EventAssistantAudio:
		return dialogrealtime.Event{Kind: dialogrealtime.EventAssistantAudio, Audio: ev.AudioPC, SampleRate: outSR, Vendor: ev.Vendor}
	case realtime.EventAssistantTurnEnd:
		return dialogrealtime.Event{Kind: dialogrealtime.EventAssistantTurnEnd, Vendor: ev.Vendor}
	case realtime.EventSessionClose:
		return dialogrealtime.Event{Kind: dialogrealtime.EventSessionClose, Vendor: ev.Vendor}
	case realtime.EventError:
		return dialogrealtime.Event{Kind: dialogrealtime.EventError, Err: ev.Err, Fatal: ev.Fatal, Vendor: ev.Vendor}
	default:
		return dialogrealtime.Event{}
	}
}
