import { floatToPCM16, resampleLinear } from '@/pages/assistants/assistantDebugAudio'

function encodeWavPcm16(pcm: Int16Array, sampleRate: number): Blob {
  const dataSize = pcm.length * 2
  const buffer = new ArrayBuffer(44 + dataSize)
  const view = new DataView(buffer)

  const writeStr = (offset: number, str: string) => {
    for (let i = 0; i < str.length; i++) view.setUint8(offset + i, str.charCodeAt(i))
  }

  writeStr(0, 'RIFF')
  view.setUint32(4, 36 + dataSize, true)
  writeStr(8, 'WAVE')
  writeStr(12, 'fmt ')
  view.setUint32(16, 16, true)
  view.setUint16(20, 1, true)
  view.setUint16(22, 1, true)
  view.setUint32(24, sampleRate, true)
  view.setUint32(28, sampleRate * 2, true)
  view.setUint16(32, 2, true)
  view.setUint16(34, 16, true)
  writeStr(36, 'data')
  view.setUint32(40, dataSize, true)

  let offset = 44
  for (let i = 0; i < pcm.length; i++, offset += 2) {
    view.setInt16(offset, pcm[i], true)
  }
  return new Blob([buffer], { type: 'audio/wav' })
}

/** Browser microphone recorder that outputs 16 kHz mono WAV. */
export class WavRecorder {
  private ctx: AudioContext | null = null
  private stream: MediaStream | null = null
  private processor: ScriptProcessorNode | null = null
  private source: MediaStreamAudioSourceNode | null = null
  private sink: GainNode | null = null
  private chunks: Float32Array[] = []
  private recording = false
  private readonly targetRate = 16000

  get isRecording() {
    return this.recording
  }

  async start(): Promise<void> {
    this.disposeTracks()
    this.stream = await navigator.mediaDevices.getUserMedia({
      audio: { echoCancellation: true, noiseSuppression: true, autoGainControl: true },
      video: false,
    })
    this.ctx = new AudioContext()
    if (this.ctx.state === 'suspended') await this.ctx.resume()
    this.source = this.ctx.createMediaStreamSource(this.stream)
    this.processor = this.ctx.createScriptProcessor(4096, 1, 1)
    this.sink = this.ctx.createGain()
    this.sink.gain.value = 0
    this.chunks = []
    this.processor.onaudioprocess = (ev) => {
      if (!this.recording) return
      const input = ev.inputBuffer.getChannelData(0)
      this.chunks.push(new Float32Array(input))
    }
    this.source.connect(this.processor)
    this.processor.connect(this.sink)
    this.sink.connect(this.ctx.destination)
    this.recording = true
  }

  stop(): File | null {
    this.recording = false
    if (!this.ctx || this.chunks.length === 0) {
      this.dispose()
      return null
    }
    let total = 0
    for (const c of this.chunks) total += c.length
    const merged = new Float32Array(total)
    let pos = 0
    for (const c of this.chunks) {
      merged.set(c, pos)
      pos += c.length
    }
    const resampled = resampleLinear(merged, this.ctx.sampleRate, this.targetRate)
    const pcm = floatToPCM16(resampled)
    const blob = encodeWavPcm16(pcm, this.targetRate)
    this.dispose()
    return new File([blob], `voiceprint-${Date.now()}.wav`, { type: 'audio/wav' })
  }

  cancel(): void {
    this.recording = false
    this.dispose()
  }

  private disposeTracks() {
    this.processor?.disconnect()
    this.source?.disconnect()
    this.sink?.disconnect()
    this.stream?.getTracks().forEach((t) => t.stop())
    this.processor = null
    this.source = null
    this.sink = null
    this.stream = null
  }

  private dispose() {
    this.disposeTracks()
    void this.ctx?.close()
    this.ctx = null
    this.chunks = []
  }
}
