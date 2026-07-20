package tenantcfg

// PCMBridgeRate normalizes the PCM bridge sample rate (RTP decode ↔ encode).
func PCMBridgeRate(sr int) int {
	if sr <= 0 {
		return 16000
	}
	return sr
}

// TTSCloudSampleRate chooses the cloud-side TTS output sample rate.
//
//   - When tenant TTS JSON pins `sampleRate`, honor it.
//   - Else trust the resolved synthesizer's native rate.
//   - Else fall back to the PCM bridge rate.
func TTSCloudSampleRate(env VoiceEnv, synthRate, pcmBridgeSR int) int {
	if env.TTSSampleRate > 0 {
		return env.TTSSampleRate
	}
	if synthRate > 0 {
		return synthRate
	}
	if pcmBridgeSR > 0 {
		return pcmBridgeSR
	}
	return 16000
}
