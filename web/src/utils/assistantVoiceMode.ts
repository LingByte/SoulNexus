import type { VoiceSessionTransport } from '@/api/voiceSession'

/** Clone timbres only work on the pipeline TTS path. */
export function assistantUsesCloneVoice(
  ttsVoice: string | undefined,
  cloneVoiceIds: ReadonlySet<string>,
): boolean {
  const id = ttsVoice?.trim() ?? ''
  return id !== '' && cloneVoiceIds.has(id)
}

/**
 * Effective attach mode for assistant debug.
 * WebSocket/WebRTC always use pipeline on the server; text follows tenant mode.
 */
export function assistantDebugVoiceMode(
  tenantVoiceMode: 'pipeline' | 'realtime' | undefined,
  ttsVoice: string | undefined,
  cloneVoiceIds: ReadonlySet<string>,
  transport: VoiceSessionTransport | 'text',
): 'pipeline' | 'realtime' {
  if (transport === 'websocket' || transport === 'webrtc') {
    return 'pipeline'
  }
  if (assistantUsesCloneVoice(ttsVoice, cloneVoiceIds)) {
    return 'pipeline'
  }
  return tenantVoiceMode === 'realtime' ? 'realtime' : 'pipeline'
}
