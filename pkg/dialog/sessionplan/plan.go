// Package sessionplan holds a dialplan-lite config bag for one voice attach.
// One object owns mode / welcome / audio policy / flags.
package sessionplan

import (
	"strings"

	"github.com/LingByte/SoulNexus/pkg/dialog/callflags"
	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/session"
	stageknow "github.com/LingByte/SoulNexus/pkg/dialog/stages/knowledge"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/dialog/voiceruntime"
)

// WelcomePolicy selects preamble behavior for attach.
type WelcomePolicy string

const (
	// WelcomeAuto lets callflags decide WAV vs TTS vs skip.
	WelcomeAuto WelcomePolicy = "auto"
	// WelcomeSkip suppresses assistant welcome (script-mode, etc.).
	WelcomeSkip WelcomePolicy = "skip"
)

// Flags are sticky/readable session marks at Build time.
type Flags struct {
	ScriptMode bool
}

// KnowledgeHint is a non-authoritative snapshot of KB binding for observability.
type KnowledgeHint struct {
	Bound       bool
	Collection  string
	NamespaceID uint
}

// Plan is the per-call attach policy document.
type Plan struct {
	CallID       string
	Mode         engine.Mode
	Welcome      WelcomePolicy
	Interruption voiceruntime.InterruptionPolicy
	Audio        voiceruntime.ProcessConfig
	Flags        Flags
	Knowledge    KnowledgeHint
}

// SnapshotFlags reads non-destructive callflags for callID.
func SnapshotFlags(callID string) Flags {
	return Flags{
		ScriptMode: callflags.IsScriptMode(callID),
	}
}

// Build constructs a Plan from tenant VoiceEnv and callflags.
// Mode uses session.ResolveMode; callers may overwrite Mode with a protocol-resolved value.
func Build(callID string, env tenantcfg.VoiceEnv) Plan {
	callID = strings.TrimSpace(callID)
	flags := SnapshotFlags(callID)
	plan := Plan{
		CallID:       callID,
		Mode:         session.ResolveMode(env),
		Welcome:      WelcomeAuto,
		Interruption: voiceruntime.InterruptionFromEnv(env),
		Audio:        voiceruntime.ProcessConfigFromEnv(env),
		Flags:        flags,
	}
	if flags.ScriptMode {
		plan.Welcome = WelcomeSkip
	}
	if b := stageknow.ResolveBinding(callID); b.Enabled {
		plan.Knowledge = KnowledgeHint{
			Bound:       true,
			Collection:  b.Collection,
			NamespaceID: b.NamespaceID,
		}
	}
	return plan
}

// SkipWelcome reports whether preamble audio should be suppressed.
func (p Plan) SkipWelcome() bool {
	return p.Welcome == WelcomeSkip || p.Flags.ScriptMode
}
