/**
 * Schedules incoming PCM16LE chunks for gapless playback (边收边播).
 */

export type PcmFormat = {
    sampleRate: number
    channels: number
    bitDepth: number
}

type PcmStreamPlayerOptions = {
    onIdle?: () => void
    /** 0~1 RMS volume per scheduled frame (for sprite lip sync) */
    onVolume?: (level: number) => void
}

/** RMS loudness of mono PCM samples, normalized 0~1 */
export function computeMonoRms(mono: Float32Array): number {
    if (mono.length === 0) return 0
    let sum = 0
    for (let i = 0; i < mono.length; i++) {
        const s = mono[i]
        sum += s * s
    }
    return Math.min(1, Math.sqrt(sum / mono.length) * 2.8)
}

function pcm16ToFloat32(bytes: Uint8Array, bitDepth: number): Float32Array | null {
    const bytesPerSample = bitDepth / 8
    if (bytesPerSample !== 2) {
        return null
    }
    const samples = Math.floor(bytes.length / bytesPerSample)
    if (samples === 0) {
        return null
    }
    const out = new Float32Array(samples)
    const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength)
    for (let i = 0; i < samples; i++) {
        out[i] = Math.max(-1, Math.min(1, view.getInt16(i * 2, true) / 32768))
    }
    return out
}

export class PcmStreamPlayer {
    private readonly ctx: AudioContext
    private readonly format: PcmFormat
    private readonly onIdle?: () => void
    private readonly onVolume?: (level: number) => void
    private readonly sources = new Set<AudioBufferSourceNode>()

    private receiveEnded = false
    private playbackStarted = false
    private idleNotified = false
    private nextPlayTime = 0
    private pending = new Uint8Array(0)

    constructor(ctx: AudioContext, format: PcmFormat, opts?: PcmStreamPlayerOptions) {
        this.ctx = ctx
        this.format = format
        this.onIdle = opts?.onIdle
        this.onVolume = opts?.onVolume
    }

    /** Start scheduling audio to the destination (call once when this segment is at the head of the queue). */
    beginPlayback(): void {
        if (this.playbackStarted) {
            return
        }
        this.playbackStarted = true
        void this.ensureRunning()
        this.schedulePending()
    }

    feed(chunk: Uint8Array): void {
        if (chunk.length === 0) {
            return
        }
        const merged = new Uint8Array(this.pending.length + chunk.length)
        merged.set(this.pending, 0)
        merged.set(chunk, this.pending.length)
        this.pending = merged
        if (this.playbackStarted) {
            this.schedulePending()
        }
    }

    /** No more PCM will arrive for this utterance. */
    markReceiveEnd(): void {
        this.receiveEnded = true
        if (this.playbackStarted) {
            this.schedulePending()
        }
    }

    stop(): void {
        this.receiveEnded = true
        for (const src of this.sources) {
            try {
                src.stop()
                src.disconnect()
            } catch {
                // already stopped
            }
        }
        this.sources.clear()
        this.pending = new Uint8Array(0)
        this.onVolume?.(0)
        this.notifyIdle()
    }

    isBusy(): boolean {
        return this.sources.size > 0 || (this.playbackStarted && !this.receiveEnded) || this.pending.length > 0
    }

    private async ensureRunning(): Promise<void> {
        if (this.ctx.state === 'suspended') {
            try {
                await this.ctx.resume()
            } catch {
                // ignore
            }
        }
    }

    private schedulePending(): void {
        const { sampleRate, channels, bitDepth } = this.format
        const bytesPerFrame = (bitDepth / 8) * channels
        if (bytesPerFrame <= 0) {
            return
        }

        // ~40ms frames: smooth enough without excessive scheduling overhead.
        const frameBytes = Math.max(bytesPerFrame, Math.floor((sampleRate * bytesPerFrame * 40) / 1000))

        while (this.pending.length >= frameBytes || (this.receiveEnded && this.pending.length > 0)) {
            const take = this.receiveEnded ? this.pending.length : Math.min(this.pending.length, frameBytes)
            if (take < bytesPerFrame) {
                break
            }
            const slice = this.pending.subarray(0, take)
            this.pending = this.pending.subarray(take)

            const mono = pcm16ToFloat32(slice, bitDepth)
            if (!mono) {
                continue
            }

            if (this.onVolume) {
                this.onVolume(computeMonoRms(mono))
            }

            const samplesPerChannel = Math.floor(mono.length / channels)
            if (samplesPerChannel === 0) {
                continue
            }

            const buffer = this.ctx.createBuffer(channels, samplesPerChannel, sampleRate)
            if (channels === 1) {
                buffer.getChannelData(0).set(mono.subarray(0, samplesPerChannel))
            } else {
                for (let ch = 0; ch < channels; ch++) {
                    const channelData = buffer.getChannelData(ch)
                    for (let i = 0; i < samplesPerChannel; i++) {
                        channelData[i] = mono[i * channels + ch]
                    }
                }
            }

            const source = this.ctx.createBufferSource()
            source.buffer = buffer
            source.connect(this.ctx.destination)
            this.sources.add(source)

            const startAt = Math.max(this.ctx.currentTime + 0.02, this.nextPlayTime)
            source.start(startAt)
            this.nextPlayTime = startAt + buffer.duration

            source.onended = () => {
                this.sources.delete(source)
                this.maybeNotifyIdle()
            }
        }

        if (this.receiveEnded && this.pending.length > 0) {
            const tail = this.pending
            this.pending = new Uint8Array(0)
            this.feed(tail)
            this.markReceiveEnd()
        }

        this.maybeNotifyIdle()
    }

    private maybeNotifyIdle(): void {
        if (!this.receiveEnded || !this.playbackStarted) {
            return
        }
        if (this.sources.size === 0 && this.pending.length === 0) {
            this.notifyIdle()
        }
    }

    private notifyIdle(): void {
        if (this.idleNotified) {
            return
        }
        this.idleNotified = true
        this.onIdle?.()
    }
}

/** Queues multiple xiaozhi TTS utterances; each streams while PCM arrives. */
export type PcmUtteranceQueueOptions = {
    onVolume?: (level: number) => void
    onIdle?: () => void
}

export class PcmUtteranceQueue {
    private readonly ctx: AudioContext
    private readonly opts: PcmUtteranceQueueOptions
    private readonly segments: PcmStreamPlayer[] = []
    private playIndex = 0

    constructor(ctx: AudioContext, opts?: PcmUtteranceQueueOptions) {
        this.ctx = ctx
        this.opts = opts ?? {}
    }

    beginUtterance(format: PcmFormat): void {
        const player = new PcmStreamPlayer(this.ctx, format, {
            onVolume: this.opts.onVolume,
            onIdle: () => {
                this.opts.onVolume?.(0)
                this.playIndex++
                this.startNext()
                if (this.playIndex >= this.segments.length) {
                    this.opts.onIdle?.()
                }
            },
        })
        this.segments.push(player)
        if (this.segments.length === 1) {
            player.beginPlayback()
        }
    }

    feed(chunk: Uint8Array): void {
        const last = this.segments[this.segments.length - 1]
        last?.feed(chunk)
    }

    endUtterance(): void {
        const last = this.segments[this.segments.length - 1]
        last?.markReceiveEnd()
    }

    stopAll(): void {
        for (const p of this.segments) {
            p.stop()
        }
        this.segments.length = 0
        this.playIndex = 0
        this.opts.onVolume?.(0)
        this.opts.onIdle?.()
    }

    private startNext(): void {
        if (this.playIndex < this.segments.length) {
            this.segments[this.playIndex].beginPlayback()
        }
    }
}
