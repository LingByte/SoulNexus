package cascaded

import "context"

type speculativeLLMKey struct{}
type preferToolsKey struct{}
type textDialogKey struct{}
type maxToolRoundsKey struct{}
type forcedKBBlockKey struct{}

// DefaultTextToolRounds aligns text/IM tool-chain budget with hard cap.
const DefaultTextToolRounds = 12

// WithSpeculativeLLM marks a StreamReply as ASR-partial speculative so
// adapters can skip irreversible side effects (e.g. NLU canned replies).
func WithSpeculativeLLM(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, speculativeLLMKey{}, true)
}

// IsSpeculativeLLM reports whether ctx was created via WithSpeculativeLLM.
func IsSpeculativeLLM(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(speculativeLLMKey{}).(bool)
	return v
}

// WithPreferTools asks StreamReply to use the non-stream tool path when any
// function tools are registered (text / IM dialogs).
func WithPreferTools(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, preferToolsKey{}, true)
}

// PreferTools reports whether ctx was created via WithPreferTools.
func PreferTools(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(preferToolsKey{}).(bool)
	return v
}

// WithTextDialog marks an accuracy-first text/IM turn (not voice realtime).
func WithTextDialog(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, textDialogKey{}, true)
}

// IsTextDialog reports whether ctx was created via WithTextDialog.
func IsTextDialog(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(textDialogKey{}).(bool)
	return v
}

// WithMaxToolRounds sets the tool-chain round budget for QueryWithOptions.
func WithMaxToolRounds(ctx context.Context, n int) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if n <= 0 {
		n = DefaultTextToolRounds
	}
	return context.WithValue(ctx, maxToolRoundsKey{}, n)
}

// MaxToolRounds returns the tool-chain budget from ctx, or 0 if unset.
func MaxToolRounds(ctx context.Context) int {
	if ctx == nil {
		return 0
	}
	v, _ := ctx.Value(maxToolRoundsKey{}).(int)
	return v
}

// WithForcedKnowledgeBlock injects a precomputed KB prompt block and skips
// the parallel SearchBlockForQuery in StreamReply (text remediation path).
func WithForcedKnowledgeBlock(ctx context.Context, block string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, forcedKBBlockKey{}, block)
}

// ForcedKnowledgeBlock returns a precomputed KB block from ctx, if any.
func ForcedKnowledgeBlock(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(forcedKBBlockKey{}).(string)
	return v
}
