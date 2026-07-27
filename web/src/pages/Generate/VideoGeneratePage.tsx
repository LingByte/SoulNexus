import { useCallback, useEffect, useRef, useState } from 'react'
import { Drawer } from '@arco-design/web-react'
import { Button, Card, Empty, Input, Select } from '@/components/ui'
import BaseLayout from '@/components/Layout/BaseLayout'
import {
  Clapperboard,
  Copy,
  Download,
  Film,
  History,
  Loader2,
  RotateCcw,
  Sparkles,
  Eye,
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
      return { label: '渲染中', className: 'bg-indigo-500/15 text-indigo-600 dark:text-indigo-400' }
    default:
      return { label: '排队中', className: 'bg-amber-500/15 text-amber-700 dark:text-amber-400' }
  }
}

function FrameSlot({
  label,
  hint,
  preview,
  disabled,
  onPick,
  onClear,
  onDropFile,
}: {
  label: string
  hint: string
  preview: string
  disabled?: boolean
  onPick: () => void
  onClear: () => void
  onDropFile: (file: File) => void
}) {
  if (preview) {
    return (
      <div className="group relative overflow-hidden rounded-lg border border-border/60 bg-background/70">
        <img src={preview} alt={label} className="aspect-video w-full object-cover" />
        <div className="absolute inset-x-0 bottom-0 flex items-center justify-between bg-gradient-to-t from-black/70 to-transparent px-2 py-1.5 text-[10px] text-white">
          <span>{label}</span>
          <button type="button" className="hover:underline disabled:opacity-50" disabled={disabled} onClick={onClear}>
            移除
          </button>
        </div>
      </div>
    )
  }
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onPick}
      onDragOver={(e) => {
        e.preventDefault()
        e.stopPropagation()
      }}
      onDrop={(e) => {
        e.preventDefault()
        const file = e.dataTransfer.files?.[0]
        if (file?.type.startsWith('image/')) onDropFile(file)
      }}
      className="grid aspect-video w-full cursor-pointer place-items-center gap-0.5 rounded-lg border border-dashed border-border/70 bg-background/60 px-2 text-center transition hover:border-indigo-400/40 hover:bg-indigo-500/5 disabled:cursor-not-allowed disabled:opacity-50"
    >
      <span className="text-[11px] font-medium text-foreground">{label}</span>
      <span className="text-[10px] text-muted-foreground">{hint}</span>
    </button>
  )
}

export default function VideoGeneratePage() {
  const [prompt, setPrompt] = useState('')
  const [duration, setDuration] = useState('5s')
  const [ratio, setRatio] = useState('16:9')
  const [motion, setMotion] = useState('medium')
  const [fps, setFps] = useState('24')
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
    if (item.duration) setDuration(item.duration.endsWith('s') ? item.duration : `${item.duration}s`)
    if (item.ratio) setRatio(item.ratio)
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

  const detailStatus = detail ? statusMeta(detail.status) : null

  return (
    <BaseLayout
      title="视频生成"
      description="文生视频 / 图生视频"
      contentPadding="0"
      contentClassName="overflow-hidden"
      actions={
        <div className="flex flex-wrap items-center gap-2">
          {generating ? (
            <Button variant="outline" size="sm" onClick={cancelGenerate}>取消</Button>
          ) : (
            <Button size="sm" leftIcon={<Sparkles size={16} />} onClick={() => void handleGenerate()}>
              开始生成
            </Button>
          )}
        </div>
      }
    >
      <div className="h-[calc(100vh-4rem)] bg-[radial-gradient(circle_at_top_right,rgba(99,102,241,0.12),transparent_36%)]">
        <div className="mx-auto flex h-full max-w-[1600px] flex-col gap-3 p-3 sm:p-4 xl:flex-row xl:gap-4">
          {/* Left: fixed pane, scrolls itself only if needed */}
          <aside className="flex max-h-[42vh] shrink-0 flex-col overflow-y-auto xl:max-h-none xl:h-full xl:w-[320px]">
            <Card className="flex flex-1 flex-col gap-3 border border-border/60 bg-card/85 p-3 shadow-sm backdrop-blur">
              <div>
                <div className="mb-1.5 text-xs font-medium text-foreground">首帧 / 尾帧</div>
                <div className="grid grid-cols-2 gap-2">
                  <FrameSlot
                    label="上传首帧"
                    hint="图生视频"
                    preview={referencePreview}
                    disabled={generating}
                    onPick={() => fileInputRef.current?.click()}
                    onClear={() => onReferenceChange(null)}
                    onDropFile={onReferenceChange}
                  />
                  <FrameSlot
                    label="上传尾帧"
                    hint={referenceFile ? '可选' : '需先传首帧'}
                    preview={lastPreview}
                    disabled={generating || !referenceFile}
                    onPick={() => lastInputRef.current?.click()}
                    onClear={() => onLastChange(null)}
                    onDropFile={onLastChange}
                  />
                </div>
                <input ref={fileInputRef} type="file" accept="image/*" className="hidden" onChange={(e) => onReferenceChange(e.target.files?.[0] ?? null)} />
                <input ref={lastInputRef} type="file" accept="image/*" className="hidden" onChange={(e) => onLastChange(e.target.files?.[0] ?? null)} />
              </div>

              <div className="min-h-0 flex-1">
                <div className="mb-1.5 flex items-center justify-between">
                  <div className="text-xs font-medium text-foreground">提示词</div>
                  <span className="text-[10px] text-muted-foreground">{prompt.length} 字</span>
                </div>
                <Input.TextArea
                  value={prompt}
                  onChange={setPrompt}
                  placeholder={'角色从左侧冲入，镜头跟随推进，粒子散开，电影感光影'}
                  autoSize={{ minRows: 4, maxRows: 6 }}
                  className="text-sm"
                />
              </div>

              <div className="grid grid-cols-2 gap-2">
                <div>
                  <div className="mb-1 text-[11px] text-muted-foreground">时长</div>
                  <Select
                    value={duration}
                    onChange={setDuration}
                    options={[
                      { label: '5 秒', value: '5s' },
                      { label: '6 秒', value: '6s' },
                      { label: '8 秒', value: '8s' },
                      { label: '10 秒', value: '10s' },
                    ]}
                  />
                </div>
                <div>
                  <div className="mb-1 text-[11px] text-muted-foreground">比例</div>
                  <Select
                    value={ratio}
                    onChange={setRatio}
                    options={[
                      { label: '16:9', value: '16:9' },
                      { label: '1:1', value: '1:1' },
                      { label: '9:16', value: '9:16' },
                    ]}
                  />
                </div>
                <div>
                  <div className="mb-1 text-[11px] text-muted-foreground">运动</div>
                  <Select
                    value={motion}
                    onChange={setMotion}
                    options={[
                      { label: '轻微', value: 'low' },
                      { label: '中等', value: 'medium' },
                      { label: '强烈', value: 'high' },
                    ]}
                  />
                </div>
                <div>
                  <div className="mb-1 text-[11px] text-muted-foreground">帧率</div>
                  <Select
                    value={fps}
                    onChange={setFps}
                    options={[
                      { label: '12', value: '12' },
                      { label: '24', value: '24' },
                      { label: '30', value: '30' },
                    ]}
                  />
                </div>
              </div>

              <Button block size="sm" leftIcon={generating ? <Loader2 size={14} className="animate-spin" /> : <Film size={14} />} disabled={generating} onClick={() => void handleGenerate()}>
                {generating ? '生成中…' : referenceFile ? '图生视频' : '文生视频'}
              </Button>

              <div className="grid grid-cols-3 gap-1.5">
                {[
                  { label: '排队', value: queued },
                  { label: '渲染', value: rendering },
                  { label: '完成', value: done },
                ].map((s) => (
                  <div key={s.label} className="rounded-lg bg-background/70 px-2 py-1.5 text-center">
                    <div className="text-[9px] text-muted-foreground">{s.label}</div>
                    <div className="text-sm font-semibold tabular-nums text-foreground">{s.value}</div>
                  </div>
                ))}
              </div>

              {generating ? (
                <div className="rounded-lg border border-indigo-400/25 bg-indigo-500/5 px-2.5 py-2">
                  <div className="flex items-center gap-1.5 text-xs font-medium text-foreground">
                    <Clapperboard className="size-3.5 text-indigo-400" />
                    正在生成
                  </div>
                  <p className="mt-0.5 line-clamp-2 text-[10px] text-muted-foreground">
                    {progressHint || '任务处理中…'}
                    {taskStatus ? ` · ${taskStatus}` : ''}
                  </p>
                </div>
              ) : null}
            </Card>
          </aside>

          {/* Right: independent scroll */}
          <section className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
            <Card className="flex h-full min-h-0 flex-col overflow-hidden border border-border/60 bg-card/75 p-0 shadow-sm">
              <div className="flex shrink-0 items-center justify-between gap-3 border-b border-border/50 px-4 py-3">
                <div className="flex items-center gap-2 text-sm font-semibold text-foreground">
                  <History size={16} className="text-indigo-400" />
                  历史创作
                </div>
                <span className="text-xs text-muted-foreground">{history.length} 条</span>
              </div>

              <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain p-3 sm:p-4">
                {historyLoading && history.length === 0 ? (
                  <div className="grid h-full min-h-[240px] place-items-center text-sm text-muted-foreground">加载历史中…</div>
                ) : history.length === 0 ? (
                  <div className="grid h-full min-h-[240px] place-items-center rounded-xl border border-dashed border-border/70 bg-[linear-gradient(160deg,rgba(99,102,241,0.08),transparent_45%),hsl(var(--background)/0.7)] p-8">
                    <Empty description="还没有历史视频，先在左侧生成一条。" />
                  </div>
                ) : (
                  <div className="grid grid-cols-1 gap-3 md:grid-cols-2 2xl:grid-cols-3">
                    {history.map((item) => {
                      const st = statusMeta(item.status)
                      return (
                        <article
                          key={item.id}
                          className="group flex flex-col overflow-hidden rounded-xl border border-border/60 bg-background/70 transition hover:border-indigo-400/35 hover:shadow-sm"
                        >
                          <div className="relative bg-black/85">
                            {item.url && item.status === 'succeeded' ? (
                              <video
                                src={item.url}
                                muted
                                playsInline
                                preload="metadata"
                                className="aspect-video w-full object-contain"
                                onMouseEnter={(e) => {
                                  const v = e.currentTarget
                                  void v.play().catch(() => undefined)
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
                              {item.duration ? <span className="rounded bg-muted/60 px-1.5 py-0.5">{item.duration}</span> : null}
                              {item.ratio ? <span className="rounded bg-muted/60 px-1.5 py-0.5">{item.ratio}</span> : null}
                              {item.resolution ? <span className="rounded bg-muted/60 px-1.5 py-0.5">{item.resolution}</span> : null}
                            </div>
                            <p className="line-clamp-2 min-h-[2.5rem] text-xs leading-relaxed text-foreground/90">
                              {item.prompt || '无提示词'}
                            </p>
                            <div className="mt-auto flex items-center gap-1.5">
                              <Button size="sm" variant="outline" className="flex-1 gap-1" leftIcon={<Eye size={12} />} onClick={() => setDetail(item)}>
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
            </Card>
          </section>
        </div>
      </div>

      <Drawer
        width={560}
        title="创作详情"
        visible={!!detail}
        onCancel={() => setDetail(null)}
        footer={null}
        unmountOnExit
      >
        {detail ? (
          <div className="flex h-full flex-col gap-4">
            {detail.url && detail.status === 'succeeded' ? (
              <div className="overflow-hidden rounded-xl bg-black">
                <video src={detail.url} controls preload="metadata" className="aspect-video w-full object-contain" />
              </div>
            ) : (
              <div className="grid aspect-video place-items-center rounded-xl border border-dashed border-border/70 bg-muted/30 text-sm text-muted-foreground">
                {detail.errorMessage || '视频尚未就绪'}
              </div>
            )}

            <div className="flex flex-wrap items-center gap-2 text-xs">
              {detailStatus ? (
                <span className={`rounded-md px-2 py-0.5 font-medium ${detailStatus.className}`}>{detailStatus.label}</span>
              ) : null}
              <span className="text-muted-foreground">{new Date(detail.at).toLocaleString()}</span>
              {detail.duration ? <span className="rounded bg-muted px-2 py-0.5">{detail.duration}</span> : null}
              {detail.ratio ? <span className="rounded bg-muted px-2 py-0.5">{detail.ratio}</span> : null}
              {detail.resolution ? <span className="rounded bg-muted px-2 py-0.5">{detail.resolution}</span> : null}
              <span className="font-mono text-[10px] text-muted-foreground">{detail.id}</span>
            </div>

            <div className="min-h-0 flex-1">
              <div className="mb-1.5 flex items-center justify-between gap-2">
                <div className="text-sm font-medium text-foreground">提示词</div>
                <div className="flex shrink-0 gap-1.5">
                  <Button size="sm" variant="outline" leftIcon={<Copy size={12} />} onClick={() => void copyPrompt(detail.prompt)}>
                    复制
                  </Button>
                  <Button size="sm" variant="outline" leftIcon={<RotateCcw size={12} />} onClick={() => reusePrompt(detail)}>
                    填入左侧
                  </Button>
                </div>
              </div>
              <div className="max-h-[40vh] overflow-y-auto rounded-lg border border-border/60 bg-background/80 px-3 py-2 text-sm leading-relaxed whitespace-pre-wrap text-foreground">
                {detail.prompt || '无提示词'}
              </div>
            </div>

            {detail.errorMessage ? (
              <div className="rounded-lg border border-rose-500/30 bg-rose-500/5 px-3 py-2 text-xs text-rose-600 dark:text-rose-400">
                {detail.errorMessage}
              </div>
            ) : null}

            <div className="mt-auto flex justify-end gap-2 border-t border-border/50 pt-3">
              {detail.url && detail.status === 'succeeded' ? (
                <a
                  href={detail.url}
                  download="generated-video.mp4"
                  target="_blank"
                  rel="noreferrer"
                  className="inline-flex h-8 items-center gap-1.5 rounded-md border border-border px-3 text-xs hover:bg-muted"
                >
                  <Download size={12} />
                  下载视频
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
