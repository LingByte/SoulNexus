// Package pipeline implements the Chain-of-Responsibility pattern
// for streaming dialog processing. It is the runtime spine of the
// cascaded engine and the structural backbone of the realtime
// engine (which has only one stage).
//
// Concepts:
//
//   - Frame   — the union value flowing between stages (PCM, text,
//               tool calls, control signals, errors).
//   - Stage   — one processing step. Implementations: VAD, ASR, LLM,
//               TTS, Recorder, BargeInGuard, ...
//   - Pipeline — wires Stages into a directed chain. Each stage runs
//               in its own goroutine; channels carry frames.
//
// Lifecycle:
//
//	p := pipeline.New("cascaded", []Stage{
//	    stages.NewVAD(...),
//	    stages.NewASR(asrProvider),
//	    stages.NewLLM(llmProvider, tools),
//	    stages.NewTTS(ttsProvider),
//	})
//	out, errs := p.Run(ctx, source)
//	for f := range out { ... }
//	if err := <-errs; err != nil { ... }
//
// The Pipeline owns goroutine lifecycles; cancelling ctx or closing
// the source channel cleanly tears down every stage in order.
//
// Design notes:
//
//   - We chose a single Frame union type rather than separate audio /
//     text / control channels because real stages frequently need to
//     react to control + data interleaved (e.g. ASR final flushes any
//     buffered interim, then emits the final). One queue + Kind tag
//     keeps the API small.
//   - Stages are NOT reused across calls. Each call constructs a
//     fresh Pipeline; engines may pool the underlying providers but
//     the Stage wrappers are cheap.
//   - Errors are surfaced via a dedicated errs channel rather than
//     embedded in Frame so callers can use the standard "select on
//     errs to fail fast" pattern.
//
// See docs/refactor-architecture.md §3.3 for the bigger picture.
package pipeline
