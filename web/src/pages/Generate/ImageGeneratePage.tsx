import { useCallback, useEffect, useRef, useState } from 'react'
import { Drawer } from '@arco-design/web-react'
import { Button, Card, Empty, Input, Select } from '@/components/ui'
import BaseLayout from '@/components/Layout/BaseLayout'
import {
  Copy,
  Download,
  Eye,
  History,
  ImageIcon,
  Loader2,
  RotateCcw,
  Sparkles,
} from 'lucide-react'
import {
  generateImageToImage,
  generateTextToImage,
  getMediaGenerateConfig,
  friendlyMediaError,
  listMediaJobs,
  parseImageSize,
  pollMediaJob,
  resolveMediaUrl,
  type ImageGenerateResult,
  type VideoTaskResult,
} from '@/api/mediaGenerate'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'

type HistoryItem = {
  id: string
  url: string
  prompt: string
  size: string
  style: string
  status: string
  progress: number
  at: number
  errorMessage?: string
}

const STYLE_LABELS: Record<string, string> = {
  pixel: '像素风',
  cartoon: '卡通',
  realistic: '写实',
  anime: '二次元',
}

function jobToHistoryItem(job: VideoTaskResult): HistoryItem {
  const w = job.width || 0
  const h = job.height || 0
  return {
    id: job.jobId || job.taskId,
    url: resolveMediaUrl(job.url),
    prompt: job.prompt || '',
    size: w > 0 && h > 0 ? `${w}x${h}` : '',
    style: job.style || '',
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
      return { label: '生成中', className: 'bg-violet-500/15 text-violet-600 dark:text-violet-400' }
    default:
      return { label: '排队中', className: 'bg-amber-500/15 text-amber-700 dark:text-amber-400' }
  }
}

export default function ImageGeneratePage() {
  const [prompt, setPrompt] = useState('')
  const [negative, setNegative] = useState('')
  const [size, setSize] = useState('1024x1024')
  const [style, setStyle] = useState('pixel')
  const [count, setCount] = useState('1')
  const [results, setResults] = useState<ImageGenerateResult[]>([])
  const [history, setHistory] = useState<HistoryItem[]>([])
  const [historyLoading, setHistoryLoading] = useState(true)
  const [generating, setGenerating] = useState(false)
  const [queued, setQueued] = useState(0)
  const [activeCount, setActiveCount] = useState(0)
  const [doneCount, setDoneCount] = useState(0)
  const [referenceFile, setReferenceFile] = useState<File | null>(null)
  const [referencePreview, setReferencePreview] = useState('')
  const [detail, setDetail] = useState<HistoryItem | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const abortRef = useRef<AbortController | null>(null)

  const refreshHistory = useCallback(async () => {
    try {
      const res = await listMediaJobs({ kind: 'image', page: 1, pageSize: 40 })
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

  const pickReference = () => fileInputRef.current?.click()

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

  const reusePrompt = (item: HistoryItem) => {
    setPrompt(item.prompt)
    if (item.size) setSize(item.size)
    if (item.style) setStyle(item.style)
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

    const total = Math.max(1, parseInt(count, 10) || 1)
    const { width, height } = parseImageSize(size)
    setGenerating(true)
    setQueued(total)
    setActiveCount(0)
    setDoneCount(0)
    const batch: ImageGenerateResult[] = []

    try {
      for (let i = 0; i < total; i += 1) {
        if (controller.signal.aborted) throw new DOMException('已取消', 'AbortError')
        setQueued(total - i)
        setActiveCount(1)
        const createRes = referenceFile
          ? await generateImageToImage({
              image: referenceFile,
              prompt: trimmed,
              negative,
              width,
              height,
              style,
              category: 'CHARACTER',
            })
          : await generateTextToImage({
              prompt: trimmed,
              negative,
              width,
              height,
              style,
              category: 'CHARACTER',
            })
        const jobId = createRes.data?.jobId || createRes.data?.taskId
        if (createRes.code !== 200 || !jobId) {
          throw new Error(createRes.msg || '创建图片任务失败')
        }
        void refreshHistory()

        const task = await pollMediaJob(jobId, {
          signal: controller.signal,
          onProgress: () => setActiveCount(1),
        })
        const url = resolveMediaUrl(task.url)
        if (!url) throw new Error('图片生成成功但未返回地址')
        batch.push({ jobId, url, status: task.status })
        setResults([...batch])
        setDoneCount(i + 1)
        void refreshHistory()
      }
      showAlert(`已生成 ${batch.length} 张图片`, 'success')
    } catch (e) {
      if (e instanceof DOMException && e.name === 'AbortError') return
      showAlert(friendlyMediaError(extractApiErrorMessage(e, '生成失败请联系管理员'), '生成失败请联系管理员'), 'error')
      if (batch.length > 0) setResults(batch)
      void refreshHistory()
    } finally {
      setGenerating(false)
      setQueued(0)
      setActiveCount(0)
      abortRef.current = null
    }
  }

  const clearResults = () => {
    setResults([])
    setDoneCount(0)
  }

  const detailStatus = detail ? statusMeta(detail.status) : null

  return (
    <BaseLayout
      title="图片生成"
      description="文生图 / 图生图"
      contentPadding="0"
      contentClassName="overflow-hidden"
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="outline" size="sm" onClick={clearResults} disabled={generating || results.length === 0}>
            清空结果
          </Button>
          <Button size="sm" className="gap-1.5" leftIcon={generating ? <Loader2 size={16} className="animate-spin" /> : <Sparkles size={16} />} disabled={generating} onClick={() => void handleGenerate()}>
            {generating ? '生成中…' : '开始生成'}
          </Button>
        </div>
      }
    >
      <div className="h-[calc(100vh-4rem)] bg-[radial-gradient(circle_at_top,rgba(168,85,247,0.12),transparent_40%)]">
        <div className="mx-auto flex h-full max-w-[1600px] flex-col gap-3 p-3 sm:p-4 xl:flex-row xl:gap-4">
          <aside className="flex max-h-[42vh] shrink-0 flex-col overflow-y-auto xl:max-h-none xl:h-full xl:w-[320px]">
            <Card className="flex flex-1 flex-col gap-3 border border-border/60 bg-card/85 p-3 shadow-sm backdrop-blur">
              <div>
                <div className="mb-1.5 text-xs font-medium text-foreground">参考图（可选）</div>
                {referencePreview ? (
                  <div className="overflow-hidden rounded-lg border border-border/60">
                    <img src={referencePreview} alt="参考图" className="aspect-video max-h-28 w-full object-cover" />
                    <div className="flex justify-end gap-1.5 border-t border-border/60 bg-background/70 p-1.5">
                      <Button size="sm" variant="outline" onClick={() => onReferenceChange(null)} disabled={generating}>移除</Button>
                      <Button size="sm" variant="outline" onClick={pickReference} disabled={generating}>更换</Button>
                    </div>
                  </div>
                ) : (
                  <button
                    type="button"
                    className="grid w-full cursor-pointer place-items-center gap-1 rounded-lg border border-dashed border-border/70 bg-background/60 px-3 py-5 text-center transition hover:border-primary/40 hover:bg-primary/5 disabled:opacity-50"
                    disabled={generating}
                    onClick={pickReference}
                    onDragOver={(e) => { e.preventDefault(); e.stopPropagation() }}
                    onDrop={(e) => {
                      e.preventDefault()
                      const file = e.dataTransfer.files?.[0]
                      if (file?.type.startsWith('image/')) onReferenceChange(file)
                    }}
                  >
                    <span className="text-[11px] font-medium text-foreground">拖拽或点击导入</span>
                    <span className="text-[10px] text-muted-foreground">有参考图时走图生图</span>
                  </button>
                )}
                <input ref={fileInputRef} type="file" accept="image/*" className="hidden" onChange={(e) => onReferenceChange(e.target.files?.[0] ?? null)} />
              </div>

              <div>
                <div className="mb-1.5 flex items-center justify-between">
                  <div className="text-xs font-medium text-foreground">提示词</div>
                  <span className="text-[10px] text-muted-foreground">{prompt.length} 字</span>
                </div>
                <Input.TextArea
                  value={prompt}
                  onChange={setPrompt}
                  placeholder="像素风格女战士半身像，侧光勾边，干净背景，游戏立绘"
                  autoSize={{ minRows: 3, maxRows: 5 }}
                  className="text-sm"
                />
              </div>

              <div>
                <div className="mb-1.5 text-xs font-medium text-foreground">反向提示词</div>
                <Input.TextArea
                  value={negative}
                  onChange={setNegative}
                  placeholder="模糊，低质量，文字，水印"
                  autoSize={{ minRows: 2, maxRows: 3 }}
                  className="text-sm"
                />
              </div>

              <div className="grid grid-cols-2 gap-2">
                <div className="col-span-2">
                  <div className="mb-1 text-[11px] text-muted-foreground">尺寸</div>
                  <Select
                    value={size}
                    onChange={setSize}
                    options={[
                      { label: '512 × 512', value: '512x512' },
                      { label: '768 × 768', value: '768x768' },
                      { label: '1024 × 1024', value: '1024x1024' },
                      { label: '1280 × 720', value: '1280x720' },
                    ]}
                  />
                </div>
                <div>
                  <div className="mb-1 text-[11px] text-muted-foreground">风格</div>
                  <Select
                    value={style}
                    onChange={setStyle}
                    options={[
                      { label: '像素风', value: 'pixel' },
                      { label: '卡通', value: 'cartoon' },
                      { label: '写实', value: 'realistic' },
                      { label: '二次元', value: 'anime' },
                    ]}
                  />
                </div>
                <div>
                  <div className="mb-1 text-[11px] text-muted-foreground">数量</div>
                  <Select
                    value={count}
                    onChange={setCount}
                    options={[
                      { label: '1 张', value: '1' },
                      { label: '2 张', value: '2' },
                      { label: '4 张', value: '4' },
                    ]}
                  />
                </div>
              </div>

              <Button block size="sm" leftIcon={generating ? <Loader2 size={14} className="animate-spin" /> : <Sparkles size={14} />} disabled={generating} onClick={() => void handleGenerate()}>
                {generating ? '生成中…' : referenceFile ? '图生图' : '文生图'}
              </Button>

              <div className="grid grid-cols-3 gap-1.5">
                {[
                  { label: '排队', value: queued },
                  { label: '生成中', value: activeCount },
                  { label: '完成', value: doneCount },
                ].map((s) => (
                  <div key={s.label} className="rounded-lg bg-background/70 px-2 py-1.5 text-center">
                    <div className="text-[9px] text-muted-foreground">{s.label}</div>
                    <div className="text-sm font-semibold tabular-nums text-foreground">{s.value}</div>
                  </div>
                ))}
              </div>

              {generating ? (
                <div className="rounded-lg border border-primary/25 bg-primary/5 px-2.5 py-2">
                  <div className="flex items-center gap-1.5 text-xs font-medium text-foreground">
                    <ImageIcon className="size-3.5 text-violet-400" />
                    正在生成 {count} 张…
                  </div>
                  <p className="mt-0.5 line-clamp-2 text-[10px] text-muted-foreground">
                    {prompt.slice(0, 48) || '未填写'}{prompt.length > 48 ? '…' : ''}
                  </p>
                </div>
              ) : null}
            </Card>
          </aside>

          <section className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
            <Card className="flex h-full min-h-0 flex-col overflow-hidden border border-border/60 bg-card/75 p-0 shadow-sm">
              <div className="flex shrink-0 items-center justify-between gap-3 border-b border-border/50 px-4 py-3">
                <div className="flex items-center gap-2 text-sm font-semibold text-foreground">
                  <History size={16} className="text-violet-400" />
                  历史创作
                </div>
                <span className="text-xs text-muted-foreground">{history.length} 条</span>
              </div>

              <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain p-3 sm:p-4">
                {results.length > 0 ? (
                  <div className="mb-4">
                    <div className="mb-2 text-xs font-medium text-foreground">本次结果</div>
                    <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-4">
                      {results.map((item, idx) => (
                        <div key={(item.jobId || item.url) + idx} className="overflow-hidden rounded-xl border border-border/60 bg-background/70">
                          <img src={resolveMediaUrl(item.url)} alt={`结果 ${idx + 1}`} className="aspect-square w-full object-cover" />
                          <div className="flex items-center justify-between border-t border-border/50 px-2 py-1.5 text-xs">
                            <span className="text-muted-foreground">#{idx + 1}</span>
                            <a
                              href={resolveMediaUrl(item.url)}
                              download={`generated-${idx + 1}.png`}
                              target="_blank"
                              rel="noreferrer"
                              className="inline-flex items-center gap-1 text-foreground hover:text-primary"
                            >
                              <Download size={12} />
                              下载
                            </a>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                ) : null}

                {historyLoading && history.length === 0 ? (
                  <div className="grid h-full min-h-[240px] place-items-center text-sm text-muted-foreground">加载历史中…</div>
                ) : history.length === 0 && results.length === 0 ? (
                  <div className="grid h-full min-h-[240px] place-items-center rounded-xl border border-dashed border-border/70 bg-[linear-gradient(145deg,rgba(168,85,247,0.06),transparent_50%),hsl(var(--background)/0.7)] p-8">
                    <Empty description="还没有创作记录，先在左侧输入提示词并生成。" />
                  </div>
                ) : history.length > 0 ? (
                  <div className="grid grid-cols-2 gap-3 md:grid-cols-3 2xl:grid-cols-4">
                    {history.map((item) => {
                      const st = statusMeta(item.status)
                      return (
                        <article
                          key={item.id}
                          className="group flex flex-col overflow-hidden rounded-xl border border-border/60 bg-background/70 transition hover:border-violet-400/35 hover:shadow-sm"
                        >
                          <div className="relative bg-muted/30">
                            {item.url && item.status === 'succeeded' ? (
                              <button type="button" className="block w-full" onClick={() => setDetail(item)}>
                                <img src={item.url} alt="" className="aspect-square w-full object-cover" />
                              </button>
                            ) : (
                              <div className="grid aspect-square place-items-center px-3 text-center text-xs text-muted-foreground">
                                {item.errorMessage || (item.status === 'running' ? `生成中 ${item.progress || 0}%` : '等待处理…')}
                              </div>
                            )}
                            <span className={`absolute left-2 top-2 rounded-md px-1.5 py-0.5 text-[10px] font-medium backdrop-blur ${st.className}`}>
                              {st.label}
                            </span>
                          </div>
                          <div className="flex flex-1 flex-col gap-1.5 p-2.5">
                            <div className="flex flex-wrap gap-1 text-[10px] text-muted-foreground">
                              <span>{STYLE_LABELS[item.style] || item.style || '默认'}</span>
                              {item.size ? <span className="rounded bg-muted/60 px-1.5 py-0.5">{item.size}</span> : null}
                            </div>
                            <p className="line-clamp-2 min-h-[2.25rem] text-[11px] leading-relaxed text-foreground/90">
                              {item.prompt || '无提示词'}
                            </p>
                            <div className="mt-auto flex items-center gap-1.5">
                              <Button size="sm" variant="outline" className="flex-1 gap-1" leftIcon={<Eye size={12} />} onClick={() => setDetail(item)}>
                                详情
                              </Button>
                              {item.url && item.status === 'succeeded' ? (
                                <a
                                  href={item.url}
                                  download="generated.png"
                                  target="_blank"
                                  rel="noreferrer"
                                  className="inline-flex h-8 items-center justify-center gap-1 rounded-md border border-border px-2.5 text-xs hover:bg-muted"
                                >
                                  <Download size={12} />
                                </a>
                              ) : null}
                            </div>
                          </div>
                        </article>
                      )
                    })}
                  </div>
                ) : null}
              </div>
            </Card>
          </section>
        </div>
      </div>

      <Drawer
        width={480}
        title="创作详情"
        visible={!!detail}
        onCancel={() => setDetail(null)}
        footer={null}
        unmountOnExit
      >
        {detail ? (
          <div className="flex h-full flex-col gap-4">
            {detail.url && detail.status === 'succeeded' ? (
              <div className="overflow-hidden rounded-xl border border-border/60 bg-muted/20">
                <img src={detail.url} alt="" className="mx-auto max-h-[42vh] w-full object-contain" />
              </div>
            ) : (
              <div className="grid aspect-square max-h-[280px] place-items-center rounded-xl border border-dashed border-border/70 bg-muted/30 text-sm text-muted-foreground">
                {detail.errorMessage || '图片尚未就绪'}
              </div>
            )}

            <div className="flex flex-wrap items-center gap-2 text-xs">
              {detailStatus ? (
                <span className={`rounded-md px-2 py-0.5 font-medium ${detailStatus.className}`}>{detailStatus.label}</span>
              ) : null}
              <span className="text-muted-foreground">{new Date(detail.at).toLocaleString()}</span>
              {detail.style ? (
                <span className="rounded bg-muted px-2 py-0.5">{STYLE_LABELS[detail.style] || detail.style}</span>
              ) : null}
              {detail.size ? <span className="rounded bg-muted px-2 py-0.5">{detail.size}</span> : null}
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
              <div className="max-h-[36vh] overflow-y-auto rounded-lg border border-border/60 bg-background/80 px-3 py-2 text-sm leading-relaxed whitespace-pre-wrap text-foreground">
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
                  download="generated.png"
                  target="_blank"
                  rel="noreferrer"
                  className="inline-flex h-8 items-center gap-1.5 rounded-md border border-border px-3 text-xs hover:bg-muted"
                >
                  <Download size={12} />
                  下载图片
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
