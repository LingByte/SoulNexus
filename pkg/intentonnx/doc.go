// Package intentonnx runs a Hugging Face–style BERT ONNX classifier (logits) plus
// optional keyword logit bias and canned replies from JSON configuration.
// Optional per-intent "reply_variants" lists extra canned lines; [Engine.Route] picks one at random with "reply".
//
// Call [InitRuntime] once per process with the path to libonnxruntime, then [NewEngine].
//
// For telephony / ASR text, enable [RouteOptions.VoiceASRHints] so phrases like「人工帮我查」
// do not get pulled toward 查询 when the user clearly asked for a human.
//
// # Integration rule (no double answers)
//
// Call [Engine.Route], inspect [RouteOutput.Channel]:
//
//   - [AnswerChannelIntent] — show [RouteOutput.Reply] as the only user-facing
//     answer. Do not call an LLM for the same user turn.
//
//   - [AnswerChannelLLM] — [RouteOutput.Reply] is empty; obtain the answer from
//     your LLM path once for that turn. Do not also append or merge intent replies.
//
// Never concatenate intent canned text with LLM output for the same user message.

package intentonnx
