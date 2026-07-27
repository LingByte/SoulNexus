import { get, post, type ApiResponse } from '@/utils/request'

const prefix = '/media'

export interface MediaGenerateConfig {
  configured: boolean
  videoPollIntervalMs: number
  videoPollMaxAttempts: number
  seedreamModelId: string
  seedanceModelId: string
}

export interface ImageGenerateResult {
  jobId?: string
  url: string
  cached?: boolean
  storageKey?: string
  status?: string
}

export interface VideoTaskCreateResult {
  taskId: string
  jobId?: string
  status: string
}

export interface MediaJobStep {
  name: string
  status: string
  startedAt?: string
  finishedAt?: string
  durationMs?: number
  message?: string
  meta?: Record<string, unknown>
}

export interface MediaJobMetrics {
  queueWaitMs?: number
  providerCreateMs?: number
  providerPollMs?: number
  downloadMs?: number
  totalMs?: number
  pollAttempts?: number
}

export interface VideoTaskResult {
  taskId: string
  jobId?: string
  kind?: string
  status: string
  prompt?: string
  style?: string
  url?: string
  remoteUrl?: string
  storageKey?: string
  resolution?: string
  ratio?: string
  duration?: number
  width?: number
  height?: number
  errorMessage?: string
  progress?: number
  steps?: MediaJobStep[]
  metrics?: MediaJobMetrics
  createdAt?: number
}

let configCache: MediaGenerateConfig | null = null

export async function getMediaGenerateConfig(): Promise<ApiResponse<MediaGenerateConfig>> {
  if (configCache) {
    return { code: 200, msg: 'ok', data: configCache }
  }
  const res = await get<MediaGenerateConfig>(`${prefix}/config`)
  if (res.code === 200 && res.data) {
    configCache = res.data
  }
  return res
}

export async function generateTextToImage(body: {
  prompt: string
  negative?: string
  width: number
  height: number
  style?: string
  category?: string
}): Promise<ApiResponse<VideoTaskCreateResult>> {
  return post(`${prefix}/image/generate`, body, { timeout: 60000 })
}

export async function generateImageToImage(body: {
  image: File
  prompt: string
  negative?: string
  width: number
  height: number
  style?: string
  category?: string
}): Promise<ApiResponse<VideoTaskCreateResult>> {
  const form = new FormData()
  form.append('image', body.image)
  form.append('prompt', body.prompt)
  form.append('width', String(body.width))
  form.append('height', String(body.height))
  if (body.style) form.append('style', body.style)
  if (body.negative) form.append('negative', body.negative)
  if (body.category) form.append('category', body.category)
  return post(`${prefix}/image/image-to-image`, form, { timeout: 60000 })
}

export async function createTextToVideo(body: {
  prompt: string
  ratio?: string
  duration: number
  resolution?: string
  generateAudio?: boolean
  watermark?: boolean
  category?: string
  motion?: string
  fps?: string
}): Promise<ApiResponse<VideoTaskCreateResult>> {
  return post(`${prefix}/video/generate`, body, { timeout: 60000 })
}

export async function createImageToVideo(body: {
  image: File
  lastImage?: File
  prompt: string
  ratio?: string
  duration: number
  resolution?: string
  generateAudio?: boolean
  watermark?: boolean
  category?: string
  motion?: string
  fps?: string
}): Promise<ApiResponse<VideoTaskCreateResult>> {
  const form = new FormData()
  form.append('image', body.image)
  if (body.lastImage) form.append('lastImage', body.lastImage)
  form.append('prompt', body.prompt)
  form.append('ratio', body.ratio ?? '16:9')
  form.append('duration', String(body.duration))
  form.append('resolution', body.resolution ?? '1080p')
  form.append('generateAudio', String(body.generateAudio ?? true))
  form.append('watermark', String(body.watermark ?? false))
  if (body.category) form.append('category', body.category)
  if (body.motion) form.append('motion', body.motion)
  if (body.fps) form.append('fps', body.fps)
  return post(`${prefix}/video/image-to-video`, form, { timeout: 60000 })
}

export async function getVideoTask(taskId: string): Promise<ApiResponse<VideoTaskResult>> {
  return get(`${prefix}/video/tasks/${encodeURIComponent(taskId)}`, { timeout: 60000 })
}

export async function listMediaJobs(params?: {
  kind?: string
  status?: string
  page?: number
  pageSize?: number
}): Promise<ApiResponse<{ list: VideoTaskResult[]; total: number; page: number; pageSize: number }>> {
  return get(`${prefix}/jobs`, {
    params: {
      kind: params?.kind,
      status: params?.status,
      page: params?.page ?? 1,
      pageSize: params?.pageSize ?? 20,
    },
  })
}

export async function getMediaJob(jobId: string): Promise<ApiResponse<VideoTaskResult>> {
  return get(`${prefix}/jobs/${encodeURIComponent(jobId)}`)
}

const TERMINAL = new Set(['succeeded', 'failed', 'cancelled', 'expired'])

export function parseDurationSeconds(label: string): number {
  const n = parseInt(String(label).replace(/\D/g, ''), 10)
  if (Number.isNaN(n)) return 5
  return Math.min(10, Math.max(5, n))
}

export function parseImageSize(size: string): { width: number; height: number } {
  const [w, h] = size.split('x').map((v) => parseInt(v, 10))
  return {
    width: Number.isFinite(w) && w > 0 ? w : 1024,
    height: Number.isFinite(h) && h > 0 ? h : 1024,
  }
}

export function resolveMediaUrl(url?: string): string {
  if (!url) return ''
  if (url.startsWith('http://') || url.startsWith('https://')) return url
  if (url.startsWith('/')) return url
  return `/uploads/${url}`
}

/** Hide provider / network internals from end users. */
export function friendlyMediaError(raw?: string | null, fallback = '生成失败请联系管理员'): string {
  const msg = String(raw ?? '').trim()
  if (!msg) return fallback
  const lower = msg.toLowerCase()
  if (
    /seedream|seedance|volces\.com|ark\.cn-beijing|context canceled|context deadline|api\s*失败|post\s+"https?:\/\//i.test(
      lower,
    ) ||
    /https?:\/\//i.test(msg) ||
    msg.length > 120
  ) {
    return fallback
  }
  return msg
}

async function pollMediaTask(
  taskId: string,
  fetchTask: (id: string) => Promise<ApiResponse<VideoTaskResult>>,
  opts?: {
    onProgress?: (task: VideoTaskResult, attempt: number) => void
    signal?: AbortSignal
    label?: string
  },
): Promise<VideoTaskResult> {
  const cfgRes = await getMediaGenerateConfig()
  const intervalMs = cfgRes.data?.videoPollIntervalMs ?? 3000
  const maxAttempts = cfgRes.data?.videoPollMaxAttempts ?? 120
  const label = opts?.label ?? '任务'

  let attempts = 0
  return new Promise((resolve, reject) => {
    const tick = async () => {
      if (opts?.signal?.aborted) {
        reject(new DOMException('已取消', 'AbortError'))
        return
      }
      attempts += 1
      try {
        const res = await fetchTask(taskId)
        if (res.code !== 200 || !res.data) {
          reject(new Error(res.msg || `查询${label}失败`))
          return
        }
        const task = res.data
        opts?.onProgress?.(task, attempts)
        const status = (task.status ?? '').toLowerCase()
        if (TERMINAL.has(status)) {
          if (status === 'succeeded') resolve(task)
          else reject(new Error(friendlyMediaError(task.errorMessage, `${label}生成失败请联系管理员`)))
          return
        }
        if (attempts >= maxAttempts) {
          reject(new Error(`${label}生成超时，请稍后在历史记录中查看`))
          return
        }
        setTimeout(tick, intervalMs)
      } catch (err) {
        reject(err)
      }
    }
    tick()
  })
}

export async function pollVideoTask(
  taskId: string,
  opts?: {
    onProgress?: (task: VideoTaskResult, attempt: number) => void
    signal?: AbortSignal
  },
): Promise<VideoTaskResult> {
  return pollMediaTask(taskId, getVideoTask, { ...opts, label: '视频' })
}

export async function pollMediaJob(
  jobId: string,
  opts?: {
    onProgress?: (task: VideoTaskResult, attempt: number) => void
    signal?: AbortSignal
  },
): Promise<VideoTaskResult> {
  // Images finish faster than video; poll more frequently than the video default (10s).
  const cfgRes = await getMediaGenerateConfig()
  const videoInterval = cfgRes.data?.videoPollIntervalMs ?? 10000
  const intervalMs = Math.min(3000, videoInterval)
  const maxAttempts = cfgRes.data?.videoPollMaxAttempts ?? 120

  let attempts = 0
  return new Promise((resolve, reject) => {
    const tick = async () => {
      if (opts?.signal?.aborted) {
        reject(new DOMException('已取消', 'AbortError'))
        return
      }
      attempts += 1
      try {
        const res = await getMediaJob(jobId)
        if (res.code !== 200 || !res.data) {
          reject(new Error(res.msg || '查询图片任务失败'))
          return
        }
        const task = res.data
        opts?.onProgress?.(task, attempts)
        const status = (task.status ?? '').toLowerCase()
        if (TERMINAL.has(status)) {
          if (status === 'succeeded') resolve(task)
          else reject(new Error(friendlyMediaError(task.errorMessage, '生成失败请联系管理员')))
          return
        }
        if (attempts >= maxAttempts) {
          reject(new Error('图片生成超时，请稍后在历史记录中查看'))
          return
        }
        setTimeout(tick, intervalMs)
      } catch (err) {
        reject(err)
      }
    }
    tick()
  })
}
