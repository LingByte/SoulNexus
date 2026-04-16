package llm

import "strings"

// emotionalToneInstruction is appended to the system / instruction side when QueryOptions.EmotionalTone is true.
const emotionalToneInstruction = "【输出风格】请在准确传达信息的前提下，使用自然、有温度、略带情感色彩的表达（可适当使用语气词、共情与比喻），避免冰冷罗列；不要编造事实，不要过度戏剧化或滥用感叹号。"

func emotionalToneEnabled(opts *QueryOptions) bool {
	return opts != nil && opts.EmotionalTone
}

// appendEmotionalStyle merges the emotional-tone instruction into the base system or instruction text.
func appendEmotionalStyle(base string, opts *QueryOptions) string {
	if !emotionalToneEnabled(opts) {
		return base
	}
	base = strings.TrimSpace(base)
	if base == "" {
		return emotionalToneInstruction
	}
	return base + "\n\n" + emotionalToneInstruction
}
