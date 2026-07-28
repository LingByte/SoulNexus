import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Drawer } from '@arco-design/web-react'
import { Button, Empty, Input } from '@/components/ui'
import BaseLayout from '@/components/Layout/BaseLayout'
import {
  ChevronRight,
  Copy,
  Download,
  Eye,
  Film,
  ImagePlus,
  Loader2,
  RotateCcw,
  Sparkles,
  Trash2,
} from 'lucide-react'
import {
  createImageToVideo,
  createTextToVideo,
  friendlyMediaError,
  getMediaGenerateConfig,
  listMediaJobs,
  parseDurationSeconds,
  pollVideoTask,
  resolveMediaUrl,
  type VideoTaskResult,
} from '@/api/mediaGenerate'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'
import { ChipGroup } from './seaartLayout'

type HistoryItem = {
  id: string
  url: string
  prompt: string
  duration: string
  ratio: string
  resolution: string
  status: string
  progress: number
  at: number
  errorMessage?: string
}

const DURATIONS = ['5s', '6s', '8s', '10s'] as const
const RATIOS = ['16:9', '1:1', '9:16', '4:3', '3:4'] as const
const MOTIONS = [
  { value: 'low', label: '轻微' },
  { value: 'medium', label: '中等' },
  { value: 'high', label: '强烈' },
] as const
const FPS = ['12', '24', '30'] as const

function jobToHistoryItem(job: VideoTaskResult): HistoryItem {
  return {
    id: job.jobId || job.taskId,
    url: resolveMediaUrl(job.url),
    prompt: job.prompt || '',
    duration: job.duration ? `${job.duration}s` : '',
    ratio: job.ratio || '',
    resolution: job.resolution || '',
    status: (job.status || '').toLowerCase(),
    progress: job.progress ?? 0,
    at: job.createdAt || Date.now(),
    errorMessage: job.errorMessage ? friendlyMediaError(job.errorMessage) : undefined,
  }
}

function statusMeta(status: string): { label: string; className: string } {
  switch (status) {
    case 'succeeded':
      return { label: '已完成', className: 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400' }
    case 'failed':
    case 'cancelled':
      return { label: status === 'cancelled' ? '已取消' : '失败', className: 'bg-rose-500/15 text-rose-600 dark:text-rose-400' }
    case 'running':
      return { label: '渲染中', className: 'bg-sky-500/15 text-sky-600 dark:text-sky-400' }
    default:
      return { label: '排队中', className: 'bg-amber-500/15 text-amber-700 dark:text-amber-400' }
  }
}

export default function VideoGeneratePage() {
  const [prompt, setPrompt] = useState('')
  const [duration, setDuration] = useState<(typeof DURATIONS)[number]>('5s')
  const [ratio, setRatio] = useState<(typeof RATIOS)[number]>('16:9')
  const [motion, setMotion] = useState<'low' | 'medium' | 'high'>('medium')
  const [fps, setFps] = useState<(typeof FPS)[number]>('24')
  const [generating, setGenerating] = useState(false)
  const [taskStatus, setTaskStatus] = useState('')
  const [progressHint, setProgressHint] = useState('')
  const [history, setHistory] = useState<HistoryItem[]>([])
  const [historyLoading, setHistoryLoading] = useState(true)
  const [queued, setQueued] = useState(0)
  const [rendering, setRendering] = useState(0)
  const [done, setDone] = useState(0)
  const [referenceFile, setReferenceFile] = useState<File | null>(null)
  const [referencePreview, setReferencePreview] = useState('')
  const [lastFile, setLastFile] = useState<File | null>(null)
  const [lastPreview, setLastPreview] = useState('')
  const [detail, setDetail] = useState<HistoryItem | null>(null)
  const [timeFilter, setTimeFilter] = useState('全部时间')
  const abortRef = useRef<AbortController | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const lastInputRef = useRef<HTMLInputElement>(null)

  const refreshHistory = useCallback(async () => {
    try {
      const res = await listMediaJobs({ kind: 'video', page: 1, pageSize: 40 })
      if (res.code === 200 && res.data?.list) {
        setHistory(res.data.list.map(jobToHistoryItem))
      }
    } catch {
      // ignore
    } finally {
      setHistoryLoading(false)
    }
  }, [])

  useEffect(() => {
    void refreshHistory()
  }, [refreshHistory])

  const onReferenceChange = (file?: File | null) => {
    if (referencePreview) URL.revokeObjectURL(referencePreview)
    if (!file) {
      setReferenceFile(null)
      setReferencePreview('')
      return
    }
    setReferenceFile(file)
    setReferencePreview(URL.createObjectURL(file))
  }

  const onLastChange = (file?: File | null) => {
    if (lastPreview) URL.revokeObjectURL(lastPreview)
    if (!file) {
      setLastFile(null)
      setLastPreview('')
      return
    }
    setLastFile(file)
    setLastPreview(URL.createObjectURL(file))
  }

  const resetForm = () => {
    setPrompt('')
    setDuration('5s')
    setRatio('16:9')
    setMotion('medium')
    setFps('24')
    onReferenceChange(null)
    onLastChange(null)
  }

  const cancelGenerate = () => {
    abortRef.current?.abort()
    abortRef.current = null
    setGenerating(false)
    setQueued(0)
    setRendering(0)
    setProgressHint('已取消')
    setTaskStatus('cancelled')
    void refreshHistory()
  }

  const reusePrompt = (item: HistoryItem) => {
    setPrompt(item.prompt)
    if (item.duration) {
      const d = item.duration.endsWith('s') ? item.duration : `${item.duration}s`
      if ((DURATIONS as readonly string[]).includes(d)) setDuration(d as (typeof DURATIONS)[number])
    }
    if (item.ratio && (RATIOS as readonly string[]).includes(item.ratio)) {
      setRatio(item.ratio as (typeof RATIOS)[number])
    }
    setDetail(null)
    showAlert('已填入提示词与参数', 'success')
  }

  const copyPrompt = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text)
      showAlert('提示词已复制', 'success')
    } catch {
      showAlert('复制失败', 'error')
    }
  }

  const handleGenerate = async () => {
    const trimmed = prompt.trim()
    if (!trimmed) {
      showAlert('请输入提示词', 'warning')
      return
    }
    const cfg = await getMediaGenerateConfig()
    if (!cfg.data?.configured) {
      showAlert('未配置 SEEDREAM_API_KEY，请联系管理员在后端环境变量中设置', 'error')
      return
    }

    abortRef.current?.abort()
    const controller = new AbortController()
    abortRef.current = controller

    const durationSec = parseDurationSeconds(duration)
    setGenerating(true)
    setTaskStatus('submitting')
    setProgressHint('正在提交任务…')
    setQueued(1)
    setRendering(0)
    setDone(0)

    try {
      const createRes = referenceFile
        ? await createImageToVideo({
            image: referenceFile,
            lastImage: lastFile || undefined,
            prompt: trimmed,
            ratio,
            duration: durationSec,
            resolution: '1080p',
            generateAudio: true,
            category: 'SCENE',
            motion,
            fps,
          })
        : await createTextToVideo({
            prompt: trimmed,
            ratio,
            duration: durationSec,
            resolution: '1080p',
            generateAudio: true,
            category: 'SCENE',
            motion,
            fps,
          })

      if (createRes.code !== 200 || !createRes.data?.taskId) {
        throw new Error(createRes.msg || '创建视频任务失败')
      }

      setQueued(0)
      setRendering(1)
      setTaskStatus(createRes.data.status || 'queued')
      setProgressHint('任务已提交，正在渲染…')
      void refreshHistory()

      await pollVideoTask(createRes.data.taskId, {
        signal: controller.signal,
        onProgress: (t: VideoTaskResult, attempt) => {
          setTaskStatus(t.status || 'running')
          setProgressHint(`渲染中（第 ${attempt} 次查询）…`)
        },
      })

      setTaskStatus('succeeded')
      setProgressHint('生成完成')
      setRendering(0)
      setDone(1)
      await refreshHistory()
      showAlert('视频生成完成', 'success')
    } catch (e) {
      if (e instanceof DOMException && e.name === 'AbortError') return
      setTaskStatus('failed')
      setProgressHint('')
      showAlert(friendlyMediaError(extractApiErrorMessage(e, '生成失败请联系管理员'), '生成失败请联系管理员'), 'error')
      void refreshHistory()
    } finally {
      setGenerating(false)
      setQueued(0)
      setRendering(0)
      abortRef.current = null
    }
  }

  const filteredHistory = useMemo(() => {
    const now = Date.now()
    return history.filter((item) => {
      if (timeFilter === '今天' && now - item.at > 24 * 3600 * 1000) return false
      if (timeFilter === '近 7 天' && now - item.at > 7 * 24 * 3600 * 1000) return false
      return true
    })
  }, [history, timeFilter])

  const detailStatus = detail ? statusMeta(detail.status) : null
  const modeLabel = referenceFile ? '图生视频' : '文生视频'

  return (
    <BaseLayout title="视频生成" description="文生视频 / 图生视频" contentPadding="0" contentClassName="overflow-hidden">
      <div className="flex h-[calc(100vh-4rem)] min-h-[560px] overflow-hidden bg-background">
        <aside className="flex w-[340px] shrink-0 flex-col border-r border-border bg-card">
          <div className="flex items-center justify-between border-b border-border px-4 py-3">
            <h2 className="text-sm font-semibold text-foreground">视频生成</h2>
            <button
              type="button"
              onClick={resetForm}
              disabled={generating}
              className="rounded-md p-1.5 text-muted-foreground transition hover:bg-muted hover:text-foreground disabled:opacity-50"
              title="重置"
              aria-label="重置"
            >
              <RotateCcw size={15} />
            </button>
          </div>

          <div className="flex-1 space-y-5 overflow-y-auto px-4 py-4">
            <div className="flex w-full items-center gap-3 rounded-xl border border-border bg-muted/30 p-2.5">
              <div className="flex size-11 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-muted">
                {referencePreview ? (
                  <img src={referencePreview} alt="" className="size-full object-cover" />
                ) : (
                  <Film size={18} className="text-muted-foreground" />
                )}
              </div>
              <div className="min-w-0 flex-1">
                <div className="truncate text-sm font-medium text-foreground">Seedance</div>
                <div className="text-xs text-muted-foreground">{modeLabel} · {duration} · {ratio}</div>
              </div>
              <ChevronRight size={16} className="shrink-0 text-muted-foreground" />
            </div>

            <div>
              <div className="mb-2 flex items-center justify-between">
                <span className="text-xs font-medium text-muted-foreground">{modeLabel}</span>
                <span className="text-xs text-muted-foreground">首帧 / 尾帧可选</span>
              </div>

              <div className="mb-3 grid grid-cols-2 gap-2">
                <button
                  type="button"
                  disabled={generating}
                  onClick={() => fileInputRef.current?.click()}
                  className="relative aspect-video overflow-hidden rounded-lg border border-dashed border-border bg-muted/20 transition hover:border-foreground/30 disabled:opacity-50"
                >
                  {referencePreview ? (
                    <>
                      <img src={referencePreview} alt="首帧" className="size-full object-cover" />
                      <span
                        className="absolute right-1 top-1 rounded bg-black/60 px-1.5 py-0.5 text-[10px] text-white"
                        onClick={(e) => {
                          e.stopPropagation()
                          onReferenceChange(null)
                        }}
                      >
                        移除
                      </span>
                    </>
                  ) : (
                    <span className="grid size-full place-items-center gap-0.5 text-center">
                      <ImagePlus size={16} className="mx-auto text-muted-foreground" />
                      <span className="text-[10px] text-muted-foreground">上传首帧</span>
                    </span>
                  )}
                </button>
                <button
                  type="button"
                  disabled={generating || !referenceFile}
                  onClick={() => lastInputRef.current?.click()}
                  className="relative aspect-video overflow-hidden rounded-lg border border-dashed border-border bg-muted/20 transition hover:border-foreground/30 disabled:opacity-50"
                >
                  {lastPreview ? (
                    <>
                      <img src={lastPreview} alt="尾帧" className="size-full object-cover" />
                      <span
                        className="absolute right-1 top-1 rounded bg-black/60 px-1.5 py-0.5 text-[10px] text-white"
                        onClick={(e) => {
                          e.stopPropagation()
                          onLastChange(null)
                        }}
                      >
                        移除
                      </span>
                    </>
                  ) : (
                    <span className="grid size-full place-items-center text-[10px] text-muted-foreground">
                      {referenceFile ? '上传尾帧' : '需先传首帧'}
                    </span>
                  )}
                </button>
              </div>
              <input ref={fileInputRef} type="file" accept="image/*" className="hidden" onChange={(e) => onReferenceChange(e.target.files?.[0] ?? null)} />
              <input ref={lastInputRef} type="file" accept="image/*" className="hidden" onChange={(e) => onLastChange(e.target.files?.[0] ?? null)} />

              <div className="overflow-hidden rounded-xl border border-border bg-muted/20">
                <Input.TextArea
                  value={prompt}
                  onChange={setPrompt}
                  disabled={generating}
                  placeholder="请详细描述镜头运动、人物动作、光影与节奏…"
                  autoSize={{ minRows: 5, maxRows: 10 }}
                  className="!border-0 !bg-transparent !shadow-none"
                />
                <div className="flex items-center justify-end gap-0.5 border-t border-border/60 px-2 py-1.5">
                  <button
                    type="button"
                    className="rounded-md p-1.5 text-muted-foreground transition hover:bg-muted hover:text-foreground"
                    title="复制"
                    onClick={() => void copyPrompt(prompt)}
                  >
                    <Copy size={15} />
                  </button>
                  <button
                    type="button"
                    className="rounded-md p-1.5 text-muted-foreground transition hover:bg-muted hover:text-foreground disabled:opacity-50"
                    title="清空"
                    disabled={generating}
                    onClick={() => setPrompt('')}
                  >
                    <Trash2 size={15} />
                  </button>
                </div>
              </div>
            </div>

            <ChipGroup label="时长" options={DURATIONS} value={duration} onChange={setDuration} disabled={generating} />
            <ChipGroup label="比例" options={RATIOS} value={ratio} onChange={setRatio} disabled={generating} />
            <ChipGroup
              label="运动幅度"
              options={MOTIONS.map((m) => m.value)}
              value={motion}
              onChange={(v) => setMotion(v as typeof motion)}
              renderLabel={(v) => MOTIONS.find((m) => m.value === v)?.label ?? v}
              disabled={generating}
            />
            <ChipGroup label="帧率" options={FPS} value={fps} onChange={setFps} renderLabel={(v) => `${v} FPS`} disabled={generating} />

            {generating ? (
              <div className="rounded-lg border border-border bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
                排队 {queued} · 渲染 {rendering} · 完成 {done}
                {progressHint ? <div className="mt-1">{progressHint}{taskStatus ? ` · ${taskStatus}` : ''}</div> : null}
              </div>
            ) : null}
          </div>

          <div className="space-y-2 border-t border-border p-4">
            {generating ? (
              <Button type="outline" long onClick={cancelGenerate}>
                取消生成
              </Button>
            ) : null}
            <Button
              type="primary"
              long
              className="h-11 gap-1.5 text-sm font-semibold"
              leftIcon={generating ? <Loader2 size={16} className="animate-spin" /> : <Sparkles size={16} />}
              disabled={generating}
              onClick={() => void handleGenerate()}
            >
              {generating ? '生成中…' : '开始创作'}
            </Button>
          </div>
        </aside>

        <section className="flex min-w-0 flex-1 flex-col bg-muted/20">
          <div className="flex flex-wrap items-center gap-2 border-b border-border bg-card px-4 py-2.5">
            <select
              value={timeFilter}
              onChange={(e) => setTimeFilter(e.target.value)}
              className="h-8 rounded-lg border border-border bg-background px-2.5 text-xs text-foreground outline-none"
            >
              {['全部时间', '今天', '近 7 天'].map((o) => (
                <option key={o} value={o}>{o}</option>
              ))}
            </select>
            <span className="text-xs text-muted-foreground">{filteredHistory.length} 条</span>
          </div>

          <div className="min-h-0 flex-1 overflow-y-auto p-4">
            {historyLoading && history.length === 0 ? (
              <div className="grid h-full min-h-[280px] place-items-center text-sm text-muted-foreground">加载中…</div>
            ) : filteredHistory.length === 0 ? (
              <div className="grid h-full min-h-[280px] place-items-center">
                <Empty description="开始你的第一次创作 — 在左侧填写提示词并点击开始创作" />
              </div>
            ) : (
              <div className="grid grid-cols-1 gap-3 md:grid-cols-2 2xl:grid-cols-3">
                {filteredHistory.map((item) => {
                  const st = statusMeta(item.status)
                  return (
                    <article key={item.id} className="group flex flex-col overflow-hidden rounded-xl border border-border bg-card transition hover:border-foreground/25">
                      <div className="relative bg-black/85">
                        {item.url && item.status === 'succeeded' ? (
                          <video
                            src={item.url}
                            muted
                            playsInline
                            preload="metadata"
                            className="aspect-video w-full object-contain"
                            onMouseEnter={(e) => {
                              void e.currentTarget.play().catch(() => undefined)
                            }}
                            onMouseLeave={(e) => {
                              e.currentTarget.pause()
                              e.currentTarget.currentTime = 0
                            }}
                          />
                        ) : (
                          <div className="grid aspect-video place-items-center px-4 text-center text-xs text-muted-foreground">
                            {item.errorMessage || (item.status === 'running' ? `渲染中 ${item.progress || 0}%` : '等待处理…')}
                          </div>
                        )}
                        <span className={`absolute left-2 top-2 rounded-md px-1.5 py-0.5 text-[10px] font-medium backdrop-blur ${st.className}`}>
                          {st.label}
                        </span>
                      </div>
                      <div className="flex flex-1 flex-col gap-2 p-3">
                        <div className="flex flex-wrap items-center gap-1.5 text-[10px] text-muted-foreground">
                          <span>{new Date(item.at).toLocaleString()}</span>
                          {item.duration ? <span className="rounded bg-muted px-1.5 py-0.5">{item.duration}</span> : null}
                          {item.ratio ? <span className="rounded bg-muted px-1.5 py-0.5">{item.ratio}</span> : null}
                        </div>
                        <p className="line-clamp-2 min-h-[2.5rem] text-xs leading-relaxed text-foreground/90">
                          {item.prompt || '无提示词'}
                        </p>
                        <div className="mt-auto flex items-center gap-1.5">
                          <Button size="sm" type="outline" className="flex-1 gap-1" leftIcon={<Eye size={12} />} onClick={() => setDetail(item)}>
                            详情
                          </Button>
                          {item.url && item.status === 'succeeded' ? (
                            <a
                              href={item.url}
                              download="generated-video.mp4"
                              target="_blank"
                              rel="noreferrer"
                              className="inline-flex h-8 items-center justify-center gap-1 rounded-md border border-border px-2.5 text-xs hover:bg-muted"
                            >
                              <Download size={12} />
                              下载
                            </a>
                          ) : null}
                        </div>
                      </div>
                    </article>
                  )
                })}
              </div>
            )}
          </div>
        </section>
      </div>

      <Drawer width={560} title="创作详情" visible={!!detail} onCancel={() => setDetail(null)} footer={null} unmountOnExit>
        {detail ? (
          <div className="flex h-full flex-col gap-4">
            {detail.url && detail.status === 'succeeded' ? (
              <div className="overflow-hidden rounded-xl bg-black">
                <video src={detail.url} controls preload="metadata" className="aspect-video w-full object-contain" />
              </div>
            ) : (
              <div className="grid aspect-video place-items-center rounded-xl border border-dashed border-border text-sm text-muted-foreground">
                {detail.errorMessage || '视频尚未就绪'}
              </div>
            )}
            <div className="flex flex-wrap items-center gap-2 text-xs">
              {detailStatus ? <span className={`rounded-md px-2 py-0.5 font-medium ${detailStatus.className}`}>{detailStatus.label}</span> : null}
              <span className="text-muted-foreground">{new Date(detail.at).toLocaleString()}</span>
              {detail.duration ? <span className="rounded bg-muted px-2 py-0.5">{detail.duration}</span> : null}
              {detail.ratio ? <span className="rounded bg-muted px-2 py-0.5">{detail.ratio}</span> : null}
            </div>
            <div className="min-h-0 flex-1">
              <div className="mb-1.5 flex items-center justify-between gap-2">
                <div className="text-sm font-medium text-foreground">提示词</div>
                <div className="flex shrink-0 gap-1.5">
                  <Button size="sm" type="outline" leftIcon={<Copy size={12} />} onClick={() => void copyPrompt(detail.prompt)}>复制</Button>
                  <Button size="sm" type="outline" leftIcon={<RotateCcw size={12} />} onClick={() => reusePrompt(detail)}>填入左侧</Button>
                </div>
              </div>
              <div className="max-h-[40vh] overflow-y-auto rounded-lg border border-border bg-background px-3 py-2 text-sm leading-relaxed whitespace-pre-wrap">
                {detail.prompt || '无提示词'}
              </div>
            </div>
            {detail.errorMessage ? (
              <div className="rounded-lg border border-rose-500/30 bg-rose-500/5 px-3 py-2 text-xs text-rose-600 dark:text-rose-400">{detail.errorMessage}</div>
            ) : null}
            <div className="mt-auto flex justify-end gap-2 border-t border-border pt-3">
              {detail.url && detail.status === 'succeeded' ? (
                <a href={detail.url} download="generated-video.mp4" target="_blank" rel="noreferrer" className="inline-flex h-8 items-center gap-1.5 rounded-md border border-border px-3 text-xs hover:bg-muted">
                  <Download size={12} />下载视频
                </a>
              ) : null}
              <Button size="sm" onClick={() => setDetail(null)}>关闭</Button>
            </div>
          </div>
        ) : null}
      </Drawer>
    </BaseLayout>
  )
}
