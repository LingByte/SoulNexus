import { pcm16ToFloat } from '@/pages/assistants/assistantDebugAudio'

let previewCtx: AudioContext | null = null
let previewSource: AudioBufferSourceNode | null = null
let previewAudioEl: HTMLAudioElement | null = null

function getPreviewContext(): AudioContext {
  if (!previewCtx || previewCtx.state === 'closed') {
    previewCtx = new AudioContext()
  }
  return previewCtx
}

export function stopVoicePreview() {
  try {
    previewSource?.stop()
  } catch {
    /* already stopped */
  }
  previewSource = null
  if (previewAudioEl) {
    previewAudioEl.pause()
    previewAudioEl.src = ''
    previewAudioEl = null
  }
}

/** Play cached WAV/MP3 from object storage URL. */
export function playVoicePreviewUrl(audioUrl: string): Promise<void> {
  stopVoicePreview()
  return new Promise((resolve, reject) => {
    const audio = new Audio(audioUrl)
    previewAudioEl = audio
    audio.onended = () => {
      if (previewAudioEl === audio) previewAudioEl = null
      resolve()
    }
    audio.onerror = () => {
      if (previewAudioEl === audio) previewAudioEl = null
      reject(new Error('audio playback failed'))
    }
    void audio.play().catch(reject)
  })
}

/** Play mono PCM16LE from base64 returned by /voices/preview */
export async function playVoicePreviewPcm(pcmBase64: string, sampleRate: number) {
  stopVoicePreview()
  const ctx = getPreviewContext()
  if (ctx.state === 'suspended') await ctx.resume()

  const binary = atob(pcmBase64)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
  const pcm = new Int16Array(bytes.buffer, bytes.byteOffset, bytes.byteLength / 2)
  if (pcm.length === 0) throw new Error('empty audio')

  const floats = pcm16ToFloat(pcm)
  const sr = sampleRate > 0 ? sampleRate : 16000
  const buf = ctx.createBuffer(1, floats.length, sr)
  buf.copyToChannel(floats as Float32Array<ArrayBuffer>, 0)

  const src = ctx.createBufferSource()
  src.buffer = buf
  src.connect(ctx.destination)
  previewSource = src
  src.onended = () => {
    if (previewSource === src) previewSource = null
  }
  src.start()
}
