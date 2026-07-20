package tts

import "github.com/LingByte/lingllm/utils"

// SanitizeSpeech prepares text for cloud TTS synthesis.
// Returns empty when the segment has no speakable content (punctuation-only, SSML, emoji, etc.).
func SanitizeSpeech(text string) string {
	return utils.SanitizeForSpeech(text)
}
