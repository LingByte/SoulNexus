import { computeMonoRms } from '@/lib/voice/pcmStreamPlayer'
import { applyPetLipSync, resetPetLipSync } from '@/lib/voice/petLipSyncBridge'

/**
 * Derive lip-sync volume from a playing MediaStream (WebRTC remote audio).
 */
export class MediaStreamLipSyncAnalyzer {
  private ctx: AudioContext | null = null
  private analyser: AnalyserNode | null = null
  private rafId = 0
  private buf: Float32Array | null = null

  start(stream: MediaStream): void {
    this.stop()
    if (!stream.getAudioTracks().length) return

    const ctx = new AudioContext()
    this.ctx = ctx
    const src = ctx.createMediaStreamSource(stream)
    const analyser = ctx.createAnalyser()
    analyser.fftSize = 512
    analyser.smoothingTimeConstant = 0.35
    src.connect(analyser)
    this.analyser = analyser
    this.buf = new Float32Array(analyser.fftSize)

    const tick = () => {
      if (!this.analyser || !this.buf) return
      this.analyser.getFloatTimeDomainData(this.buf)
      applyPetLipSync(computeMonoRms(this.buf))
      this.rafId = requestAnimationFrame(tick)
    }
    this.rafId = requestAnimationFrame(tick)

    void ctx.resume().catch(() => {
      /* autoplay policy */
    })
  }

  stop(): void {
    if (this.rafId) {
      cancelAnimationFrame(this.rafId)
      this.rafId = 0
    }
    if (this.ctx) {
      void this.ctx.close().catch(() => {})
      this.ctx = null
    }
    this.analyser = null
    this.buf = null
    resetPetLipSync()
  }
}

/**
 * Derive lip-sync volume from an HTMLAudioElement (legacy tts_audio URL playback).
 */
export class AudioElementLipSyncAnalyzer {
  private ctx: AudioContext | null = null
  private analyser: AnalyserNode | null = null
  private rafId = 0
  private buf: Float32Array | null = null
  private boundEl: HTMLAudioElement | null = null

  start(el: HTMLAudioElement): void {
    this.stop()
    this.boundEl = el

    const ctx = new AudioContext()
    this.ctx = ctx
    const src = ctx.createMediaElementSource(el)
    const analyser = ctx.createAnalyser()
    analyser.fftSize = 512
    analyser.smoothingTimeConstant = 0.35
    src.connect(analyser)
    analyser.connect(ctx.destination)
    this.analyser = analyser
    this.buf = new Float32Array(analyser.fftSize)

    const tick = () => {
      if (!this.analyser || !this.buf || !this.boundEl) return
      if (this.boundEl.paused || this.boundEl.ended) {
        resetPetLipSync()
        this.rafId = requestAnimationFrame(tick)
        return
      }
      this.analyser.getFloatTimeDomainData(this.buf)
      applyPetLipSync(computeMonoRms(this.buf))
      this.rafId = requestAnimationFrame(tick)
    }
    this.rafId = requestAnimationFrame(tick)

    void ctx.resume().catch(() => {})
  }

  stop(): void {
    if (this.rafId) {
      cancelAnimationFrame(this.rafId)
      this.rafId = 0
    }
    if (this.ctx) {
      void this.ctx.close().catch(() => {})
      this.ctx = null
    }
    this.analyser = null
    this.buf = null
    this.boundEl = null
    resetPetLipSync()
  }
}
