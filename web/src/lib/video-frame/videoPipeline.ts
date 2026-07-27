import type { ExtractOptions, VideoMeta } from '@/lib/video-frame/extractFrames'
import { extractVideoFrames } from '@/lib/video-frame/extractFrames'
import { composeTransparentPng, revokeComposedUrl } from '@/lib/video-frame/composeFrame'
import { segmentFrameBlob, type SegmentModel } from '@/lib/video-frame/segmentFrame'
import { smoothAlphaSequence } from '@/lib/video-frame/temporalSmooth'
import { blobToImage } from '@/lib/pixel/canvasUtils'

export type MattingFrame = {
  index: number
  name: string
  blob: Blob
  dataUrl: string
  timeSec: number
}

export type VideoMattingOptions = ExtractOptions & {
  segmentModel: SegmentModel
  emaBeta: number
  onProgress?: (done: number, total: number, stage: string) => void
}

export async function runVideoMattingPipeline(
  video: HTMLVideoElement,
  meta: VideoMeta,
  opts: VideoMattingOptions,
): Promise<MattingFrame[]> {
  opts.onProgress?.(0, 0, '拆帧')
  const rawFrames = await extractVideoFrames(video, meta, {
    ...opts,
    onProgress: (d, t) => opts.onProgress?.(d, t, '拆帧'),
  })
  return mattingFromExtracted(rawFrames, opts)
}

export async function mattingFromExtracted(
  rawFrames: Awaited<ReturnType<typeof extractVideoFrames>>,
  opts: Pick<VideoMattingOptions, 'segmentModel' | 'emaBeta' | 'onProgress'>,
): Promise<MattingFrame[]> {
  if (!rawFrames.length) throw new Error('未拆出任何帧')

  const alphas: Float32Array[] = []
  const sizes: Array<{ w: number; h: number }> = []

  for (let i = 0; i < rawFrames.length; i++) {
    const seg = await segmentFrameBlob(rawFrames[i].blob, opts.segmentModel)
    alphas.push(seg.alpha)
    sizes.push({ w: seg.width, h: seg.height })
    opts.onProgress?.(i + 1, rawFrames.length, 'AI 分割')
  }

  opts.onProgress?.(0, rawFrames.length, '时序平滑')
  const smoothed = smoothAlphaSequence(alphas, opts.emaBeta)

  const results: MattingFrame[] = []
  for (let i = 0; i < rawFrames.length; i++) {
    const f = rawFrames[i]
    const { w, h } = sizes[i]
    const { blob, dataUrl } = await composeTransparentPng(f.blob, smoothed[i], w, h)
    results.push({
      index: f.index,
      name: f.name.replace('.png', '_alpha.png'),
      blob,
      dataUrl,
      timeSec: f.timeSec,
    })
    opts.onProgress?.(i + 1, rawFrames.length, '合成透明 PNG')
  }

  opts.onProgress?.(rawFrames.length, rawFrames.length, '完成')
  return results
}

export function revokeMattingFrames(frames: MattingFrame[]) {
  for (const f of frames) revokeComposedUrl(f.dataUrl)
}
