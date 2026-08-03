/** 浏览器内视频抽帧 — 对应 example/SoulMyimage framing.py 的 ffmpeg fps 拆帧逻辑 */

export type ExtractMode = 'fps' | 'frame-interval'

export type VideoMeta = {
  duration: number
  width: number
  height: number
  fileName: string
}

export type ExtractedFrame = {
  index: number
  name: string
  blob: Blob
  dataUrl: string
  timeSec: number
}

/** 四边裁切像素（相对源视频分辨率） */
export type CropRegion = {
  left: number
  top: number
  right: number
  bottom: number
}

export type ExtractOptions = {
  mode: ExtractMode
  /** 目标 FPS，对应 ffmpeg vf=fps={fps} */
  fps: number
  /** 按源帧间隔抽帧时的步长（帧） */
  frameInterval: number
  maxFrames: number
  /** 假设的源视频帧率，用于 frame-interval 模式 */
  assumedSourceFps?: number
  /** 采样起始秒（含） */
  startSec?: number
  /** 采样结束秒（含）；缺省为视频末尾 */
  endSec?: number
  /** 可选矩形裁切 */
  crop?: CropRegion
  onProgress?: (done: number, total: number, stage: string) => void
}

export async function loadVideoFile(file: File): Promise<{ video: HTMLVideoElement; objectUrl: string; meta: VideoMeta }> {
  const objectUrl = URL.createObjectURL(file)
  const video = document.createElement('video')
  video.preload = 'auto'
  video.muted = true
  video.playsInline = true
  video.src = objectUrl

  await new Promise<void>((resolve, reject) => {
    video.onloadedmetadata = () => resolve()
    video.onerror = () => reject(new Error('无法读取视频元数据'))
  })

  if (!Number.isFinite(video.duration) || video.duration <= 0) {
    URL.revokeObjectURL(objectUrl)
    throw new Error('视频时长无效')
  }

  return {
    video,
    objectUrl,
    meta: {
      duration: video.duration,
      width: video.videoWidth,
      height: video.videoHeight,
      fileName: file.name,
    },
  }
}

export function disposeVideo(objectUrl: string) {
  URL.revokeObjectURL(objectUrl)
}

function computeSampleTimes(
  duration: number,
  opts: Pick<ExtractOptions, 'mode' | 'fps' | 'frameInterval' | 'maxFrames' | 'assumedSourceFps' | 'startSec' | 'endSec'>,
): number[] {
  const times: number[] = []
  const max = Math.max(1, opts.maxFrames)
  const start = Math.max(0, opts.startSec ?? 0)
  const endRaw = opts.endSec != null && Number.isFinite(opts.endSec) ? opts.endSec : duration
  const end = Math.min(Math.max(start, endRaw), Math.max(0, duration - 0.001))

  if (end <= start) {
    times.push(start)
    return times.slice(0, max)
  }

  if (opts.mode === 'fps') {
    const fps = Math.max(0.1, opts.fps)
    const step = 1 / fps
    for (let t = start; t <= end && times.length < max; t += step) {
      times.push(Math.min(t, end))
    }
    return times
  }

  const sourceFps = Math.max(1, opts.assumedSourceFps ?? 30)
  const step = Math.max(1, opts.frameInterval) / sourceFps
  for (let t = start; t <= end && times.length < max; t += step) {
    times.push(Math.min(t, end))
  }
  return times
}

function seekVideo(video: HTMLVideoElement, time: number): Promise<void> {
  return new Promise((resolve, reject) => {
    const onSeeked = () => {
      cleanup()
      resolve()
    }
    const onError = () => {
      cleanup()
      reject(new Error('视频定位失败'))
    }
    const cleanup = () => {
      video.removeEventListener('seeked', onSeeked)
      video.removeEventListener('error', onError)
    }
    video.addEventListener('seeked', onSeeked)
    video.addEventListener('error', onError)
    video.currentTime = Math.max(0, time)
  })
}

function normalizeCrop(video: HTMLVideoElement, crop?: CropRegion): { sx: number; sy: number; sw: number; sh: number } {
  const vw = video.videoWidth
  const vh = video.videoHeight
  const left = Math.max(0, Math.floor(crop?.left ?? 0))
  const top = Math.max(0, Math.floor(crop?.top ?? 0))
  const right = Math.max(0, Math.floor(crop?.right ?? 0))
  const bottom = Math.max(0, Math.floor(crop?.bottom ?? 0))
  const sw = Math.max(1, vw - left - right)
  const sh = Math.max(1, vh - top - bottom)
  return { sx: left, sy: top, sw: Math.min(sw, vw - left), sh: Math.min(sh, vh - top) }
}

function captureFrame(video: HTMLVideoElement, crop?: CropRegion): Promise<Blob> {
  const { sx, sy, sw, sh } = normalizeCrop(video, crop)
  const canvas = document.createElement('canvas')
  canvas.width = sw
  canvas.height = sh
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('无法创建画布')
  ctx.drawImage(video, sx, sy, sw, sh, 0, 0, sw, sh)
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => (blob ? resolve(blob) : reject(new Error('帧导出失败'))),
      'image/png',
    )
  })
}

export async function extractVideoFrames(
  video: HTMLVideoElement,
  meta: VideoMeta,
  opts: ExtractOptions,
): Promise<ExtractedFrame[]> {
  const times = computeSampleTimes(meta.duration, opts)
  if (times.length === 0) {
    throw new Error('未生成任何采样时间点，请检查参数')
  }

  opts.onProgress?.(0, times.length, '拆帧')

  const frames: ExtractedFrame[] = []
  for (let i = 0; i < times.length; i++) {
    await seekVideo(video, times[i]!)
    const blob = await captureFrame(video, opts.crop)
    const dataUrl = URL.createObjectURL(blob)
    const index = i + 1
    frames.push({
      index,
      name: `frame_${String(index).padStart(6, '0')}.png`,
      blob,
      dataUrl,
      timeSec: times[i]!,
    })
    opts.onProgress?.(i + 1, times.length, '拆帧')
  }

  opts.onProgress?.(times.length, times.length, '完成')
  return frames
}

export function revokeFrameUrls(frames: ExtractedFrame[]) {
  for (const f of frames) URL.revokeObjectURL(f.dataUrl)
}
