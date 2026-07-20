/** PCM helpers for voice-session WebSocket debug (16-bit LE mono). */

export function floatToPCM16(input: Float32Array): Int16Array {
  const out = new Int16Array(input.length)
  for (let i = 0; i < input.length; i++) {
    const s = Math.max(-1, Math.min(1, input[i]))
    out[i] = s < 0 ? s * 0x8000 : s * 0x7fff
  }
  return out
}

export function pcm16ToFloat(input: Int16Array): Float32Array {
  const out = new Float32Array(input.length)
  for (let i = 0; i < input.length; i++) out[i] = input[i] / 0x8000
  return out
}

export function resampleLinear(input: Float32Array, fromRate: number, toRate: number): Float32Array {
  if (fromRate === toRate || input.length === 0) return input
  const ratio = toRate / fromRate
  const outLen = Math.max(1, Math.round(input.length * ratio))
  const out = new Float32Array(outLen)
  for (let i = 0; i < outLen; i++) {
    const src = i / ratio
    const idx = Math.floor(src)
    const frac = src - idx
    const a = input[Math.min(idx, input.length - 1)]
    const b = input[Math.min(idx + 1, input.length - 1)]
    out[i] = a + (b - a) * frac
  }
  return out
}

/** RMS of PCM16 mono; used to detect user speech over TTS bleed. */
export function pcm16RMS(pcm: Int16Array): number {
  if (pcm.length === 0) return 0
  let sum = 0
  for (let i = 0; i < pcm.length; i++) {
    const s = pcm[i]
    sum += s * s
  }
  return Math.sqrt(sum / pcm.length)
}

/** Shared capture + playback for WebSocket debug (one AudioContext, half-duplex uplink gate). */
export class VoiceDebugAudio {
  private ctx: AudioContext
  private nextTime = 0
  private playbackUntilMs = 0
  private stream: MediaStream | null = null
  private processor: ScriptProcessorNode | null = null
  private source: MediaStreamAudioSourceNode | null = null
  private silentSink: GainNode | null = null
  private targetRate: number
  private fullDuplex: boolean

  constructor(sampleRate: number, opts?: { fullDuplex?: boolean }) {
    this.targetRate = sampleRate > 0 ? sampleRate : 16000
    this.fullDuplex = opts?.fullDuplex === true
    this.ctx = new AudioContext({ sampleRate: this.targetRate })
  }

  async resume() {
    if (this.ctx.state === 'suspended') await this.ctx.resume()
  }

  /** True while downlink audio is still playing (includes tail buffer). */
  isPlaying(): boolean {
    return performance.now() < this.playbackUntilMs
  }

  /** Actual AudioContext sample rate (may differ from target on some browsers). */
  captureSampleRate(): number {
    return this.ctx.sampleRate
  }

  async startCapture(onFrame: (pcm: Int16Array) => void) {
    this.stream = await navigator.mediaDevices.getUserMedia({
      audio: {
        echoCancellation: true,
        noiseSuppression: true,
        autoGainControl: true,
      },
      video: false,
    })
    await this.resume()
    this.source = this.ctx.createMediaStreamSource(this.stream)
    const bufferSize = 4096
    this.processor = this.ctx.createScriptProcessor(bufferSize, 1, 1)
    this.processor.onaudioprocess = (ev) => {
      // Realtime multimodal APIs need continuous uplink; pipeline mode keeps half-duplex.
      if (!this.fullDuplex && this.isPlaying()) return
      const input = ev.inputBuffer.getChannelData(0)
      const resampled = resampleLinear(input, this.ctx.sampleRate, this.targetRate)
      const pcm = floatToPCM16(resampled)
      onFrame(pcm)
    }
    this.silentSink = this.ctx.createGain()
    this.silentSink.gain.value = 0
    this.source.connect(this.processor)
    this.processor.connect(this.silentSink)
    this.silentSink.connect(this.ctx.destination)
  }

  enqueue(pcm: Int16Array, sampleRate: number) {
    let floats = pcm16ToFloat(pcm)
    const ctxRate = this.ctx.sampleRate
    if (sampleRate > 0 && sampleRate !== ctxRate) {
      floats = resampleLinear(floats, sampleRate, ctxRate)
    }
    const buf = this.ctx.createBuffer(1, floats.length, ctxRate)
    buf.copyToChannel(floats as Float32Array<ArrayBuffer>, 0)
    const src = this.ctx.createBufferSource()
    src.buffer = buf
    src.connect(this.ctx.destination)
    const now = this.ctx.currentTime
    if (this.nextTime < now) this.nextTime = now + 0.02
    src.start(this.nextTime)
    this.nextTime += buf.duration
    const aheadSec = Math.max(0, this.nextTime - this.ctx.currentTime)
    this.playbackUntilMs = performance.now() + aheadSec * 1000 + 400
  }

  close() {
    this.processor?.disconnect()
    this.source?.disconnect()
    this.silentSink?.disconnect()
    this.stream?.getTracks().forEach((t) => t.stop())
    void this.ctx.close()
    this.processor = null
    this.source = null
    this.silentSink = null
    this.stream = null
    this.playbackUntilMs = 0
  }
}

/** @deprecated use VoiceDebugAudio */
export class PCMPlayer {
  private inner: VoiceDebugAudio

  constructor(sampleRate: number) {
    this.inner = new VoiceDebugAudio(sampleRate)
  }

  async resume() {
    await this.inner.resume()
  }

  isPlaying(): boolean {
    return this.inner.isPlaying()
  }

  enqueue(pcm: Int16Array, sampleRate: number) {
    this.inner.enqueue(pcm, sampleRate)
  }

  close() {
    this.inner.close()
  }
}

/** @deprecated use VoiceDebugAudio */
export class PCMCapture {
  private inner: VoiceDebugAudio | null = null

  async start(targetRate: number, onFrame: (pcm: Int16Array) => void) {
    this.inner = new VoiceDebugAudio(targetRate)
    await this.inner.startCapture(onFrame)
  }

  stop() {
    this.inner?.close()
    this.inner = null
  }
}
