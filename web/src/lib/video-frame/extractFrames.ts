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

export type ExtractOptions = {
  mode: ExtractMode
  /** 目标 FPS，对应 ffmpeg vf=fps={fps} */
  fps: number
  /** 按源帧间隔抽帧时的步长（帧） */
  frameInterval: number
  maxFrames: number
  /** 假设的源视频帧率，用于 frame-interval 模式 */
  assumedSourceFps?: number
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
  opts: Pick<ExtractOptions, 'mode' | 'fps' | 'frameInterval' | 'maxFrames' | 'assumedSourceFps'>,
): number[] {
  const times: number[] = []
  const max = Math.max(1, opts.maxFrames)
  const end = Math.max(0, duration - 0.001)

  if (opts.mode === 'fps') {
    const fps = Math.max(0.1, opts.fps)
    const step = 1 / fps
    for (let t = 0; t <= end && times.length < max; t += step) {
      times.push(t)
    }
    return times
  }

  const sourceFps = Math.max(1, opts.assumedSourceFps ?? 30)
  const step = Math.max(1, opts.frameInterval) / sourceFps
  for (let t = 0; t <= end && times.length < max; t += step) {
    times.push(t)
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

function captureFrame(video: HTMLVideoElement): Promise<Blob> {
  const canvas = document.createElement('canvas')
  canvas.width = video.videoWidth
  canvas.height = video.videoHeight
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('无法创建画布')
  ctx.drawImage(video, 0, 0, canvas.width, canvas.height)
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
    await seekVideo(video, times[i])
    const blob = await captureFrame(video)
    const dataUrl = URL.createObjectURL(blob)
    const index = i + 1
    frames.push({
      index,
      name: `frame_${String(index).padStart(6, '0')}.png`,
      blob,
      dataUrl,
      timeSec: times[i],
    })
    opts.onProgress?.(i + 1, times.length, '拆帧')
  }

  opts.onProgress?.(times.length, times.length, '完成')
  return frames
}

export function revokeFrameUrls(frames: ExtractedFrame[]) {
  for (const f of frames) URL.revokeObjectURL(f.dataUrl)
}
