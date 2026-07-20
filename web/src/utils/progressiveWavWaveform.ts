/**
 * Build a waveform by fetching a PCM WAV via HTTP Range (206).
 * Requires the origin/CDN to allow CORS and expose Content-Range / Content-Length.
 */

export type WaveformProgress = {
  bars: number[]
  /** 0–1: bytes of PCM data received (not file header). */
  progress: number
}

const DEFAULT_BAR_COUNT = 100
const HEADER_PROBE_BYTES = 8192
const PCM_CHUNK_BYTES = 256 * 1024

function readFourCC(view: DataView, offset: number): string {
  let s = ''
  for (let i = 0; i < 4; i++) s += String.fromCharCode(view.getUint8(offset + i))
  return s
}

export type WavPCMInfo = {
  dataOffset: number
  dataSize: number
  channels: number
  sampleRate: number
  bitsPerSample: number
}

export function parseWavPCMHeader(buf: ArrayBuffer): WavPCMInfo | null {
  const view = new DataView(buf)
  if (buf.byteLength < 12 || readFourCC(view, 0) !== 'RIFF' || readFourCC(view, 8) !== 'WAVE') {
    return null
  }
  let pos = 12
  let channels = 0
  let sampleRate = 0
  let bitsPerSample = 0
  let dataOffset = 0
  let dataSize = 0
  while (pos + 8 <= buf.byteLength) {
    const id = readFourCC(view, pos)
    const size = view.getUint32(pos + 4, true)
    const chunkStart = pos + 8
    if (id === 'fmt ' && chunkStart + 16 <= buf.byteLength) {
      channels = view.getUint16(chunkStart + 2, true)
      sampleRate = view.getUint32(chunkStart + 4, true)
      bitsPerSample = view.getUint16(chunkStart + 14, true)
    } else if (id === 'data') {
      dataOffset = chunkStart
      dataSize = size
      break
    }
    pos = chunkStart + size + (size & 1)
  }
  if (!dataSize || !channels || bitsPerSample !== 16) return null
  return { dataOffset, dataSize, channels, sampleRate, bitsPerSample }
}

class WaveformPeakAccumulator {
  private readonly peaks: number[]
  private globalMax = 0
  private samplesProcessed = 0

  constructor(
    private readonly barCount: number,
    private readonly totalSamples: number
  ) {
    this.peaks = new Array(barCount).fill(0)
  }

  appendPCM16LE(bytes: ArrayBuffer, channels: number): void {
    const view = new DataView(bytes)
    const frameBytes = channels * 2
    const frames = Math.floor(bytes.byteLength / frameBytes)
    for (let f = 0; f < frames; f++) {
      const sampleIdx = this.samplesProcessed + f
      if (sampleIdx >= this.totalSamples) break
      let peak = 0
      for (let c = 0; c < channels; c++) {
        const off = f * frameBytes + c * 2
        if (off + 2 > bytes.byteLength) break
        peak = Math.max(peak, Math.abs(view.getInt16(off, true)))
      }
      const barIdx = Math.min(this.barCount - 1, Math.floor((sampleIdx / this.totalSamples) * this.barCount))
      if (peak > this.peaks[barIdx]) this.peaks[barIdx] = peak
      if (peak > this.globalMax) this.globalMax = peak
    }
    this.samplesProcessed = Math.min(this.totalSamples, this.samplesProcessed + frames)
  }

  pcmProgress(): number {
    return this.totalSamples > 0 ? this.samplesProcessed / this.totalSamples : 0
  }

  toBars(): number[] {
    const max = this.globalMax || 1
    return this.peaks.map((p) => (p / max) * 100)
  }
}

async function fetchByteRange(url: string, start: number, end: number, signal?: AbortSignal): Promise<Response> {
  return fetch(url, {
    headers: { Range: `bytes=${start}-${end}` },
    signal,
  })
}

function contentLengthFromResponse(res: Response): number | null {
  const cr = res.headers.get('Content-Range')
  if (cr) {
    const m = /\/(\d+)\s*$/.exec(cr)
    if (m) return parseInt(m[1], 10)
  }
  const cl = res.headers.get('Content-Length')
  if (cl) return parseInt(cl, 10)
  return null
}

async function probeContentLength(url: string, signal?: AbortSignal): Promise<number> {
  try {
    const head = await fetch(url, { method: 'HEAD', signal })
    if (head.ok) {
      const cl = head.headers.get('Content-Length')
      if (cl) return parseInt(cl, 10)
    }
  } catch {
    /* HEAD often blocked on CDN; fall through */
  }
  const res = await fetchByteRange(url, 0, 0, signal)
  const len = contentLengthFromResponse(res)
  if (len != null && len > 0) return len
  if (res.ok && res.status === 200) {
    const buf = await res.arrayBuffer()
    return buf.byteLength
  }
  throw new Error('cannot determine content length')
}

function processFullBuffer(
  acc: WaveformPeakAccumulator,
  info: WavPCMInfo,
  full: ArrayBuffer,
  onUpdate: (p: WaveformProgress) => void,
  emitThrottled: (p: WaveformProgress) => void
): void {
  const pcmStart = info.dataOffset
  const pcmEnd = info.dataOffset + info.dataSize
  const step = PCM_CHUNK_BYTES
  for (let off = pcmStart; off < pcmEnd; off += step) {
    const end = Math.min(off + step, pcmEnd)
    acc.appendPCM16LE(full.slice(off, end), info.channels)
    emitThrottled({ bars: acc.toBars(), progress: acc.pcmProgress() })
  }
  onUpdate({ bars: acc.toBars(), progress: 1 })
}

/**
 * Fetches WAV in Range chunks and calls onUpdate as peaks accumulate.
 */
export async function loadProgressiveWavWaveform(
  url: string,
  onUpdate: (p: WaveformProgress) => void,
  opts?: { barCount?: number; signal?: AbortSignal }
): Promise<void> {
  const barCount = opts?.barCount ?? DEFAULT_BAR_COUNT
  const signal = opts?.signal
  let lastEmit = 0
  const emitThrottled = (p: WaveformProgress) => {
    const now = performance.now()
    if (now - lastEmit < 64 && p.progress < 1) return
    lastEmit = now
    onUpdate(p)
  }

  const fileSize = await probeContentLength(url, signal)
  const headerEnd = Math.min(fileSize - 1, HEADER_PROBE_BYTES - 1)
  const headerRes = await fetchByteRange(url, 0, headerEnd, signal)
  if (!headerRes.ok) throw new Error(`header fetch ${headerRes.status}`)

  let headerBuf = await headerRes.arrayBuffer()
  let info = parseWavPCMHeader(headerBuf)

  if (!info && headerRes.status === 206) {
    const biggerEnd = Math.min(fileSize - 1, 65535)
    const res2 = await fetchByteRange(url, 0, biggerEnd, signal)
    headerBuf = await res2.arrayBuffer()
    info = parseWavPCMHeader(headerBuf)
  }

  if (!info) throw new Error('unsupported or invalid WAV')

  const bytesPerSample = info.channels * 2
  const totalSamples = Math.floor(info.dataSize / bytesPerSample)
  const acc = new WaveformPeakAccumulator(barCount, totalSamples)

  if (headerRes.status === 200) {
    processFullBuffer(acc, info, headerBuf, onUpdate, emitThrottled)
    return
  }

  const dataEnd = info.dataOffset + info.dataSize
  let nextByte = info.dataOffset

  const headerPcmLen = Math.max(0, headerBuf.byteLength - info.dataOffset)
  if (headerPcmLen > 0) {
    acc.appendPCM16LE(headerBuf.slice(info.dataOffset, info.dataOffset + Math.min(headerPcmLen, info.dataSize)), info.channels)
    nextByte += Math.min(headerPcmLen, info.dataSize)
    emitThrottled({ bars: acc.toBars(), progress: acc.pcmProgress() })
  }

  while (nextByte < dataEnd) {
    if (signal?.aborted) return
    const end = Math.min(nextByte + PCM_CHUNK_BYTES - 1, dataEnd - 1)
    const res = await fetchByteRange(url, nextByte, end, signal)
    if (!res.ok) throw new Error(`range fetch ${res.status}`)
    if (res.status === 200) {
      const full = await res.arrayBuffer()
      processFullBuffer(acc, info, full, onUpdate, emitThrottled)
      return
    }
    const chunk = await res.arrayBuffer()
    acc.appendPCM16LE(chunk, info.channels)
    nextByte = end + 1
    emitThrottled({ bars: acc.toBars(), progress: acc.pcmProgress() })
  }

  onUpdate({ bars: acc.toBars(), progress: 1 })
}
