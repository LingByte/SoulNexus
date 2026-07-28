import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Drawer } from '@arco-design/web-react'
import { Button, Empty, Input } from '@/components/ui'
import BaseLayout from '@/components/Layout/BaseLayout'
import {
  ChevronRight,
  Copy,
  Download,
  Eye,
  Image as ImageIcon,
  ImagePlus,
  Loader2,
  RotateCcw,
  Sparkles,
  Trash2,
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
import { ChipGroup, sizeFromResRatio } from './seaartLayout'

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

const RESOLUTIONS = ['1K', '2K', '4K'] as const
const RATIOS = ['1:1', '2:3', '3:2', '3:4', '4:3', '4:5', '16:9', '9:16'] as const
const COUNTS = ['1', '2', '3', '4'] as const
const STYLES = [
  { value: 'pixel', label: '像素' },
  { value: 'cartoon', label: '卡通' },
  { value: 'realistic', label: '写实' },
  { value: 'anime', label: '二次元' },
] as const

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
      return { label: '生成中', className: 'bg-sky-500/15 text-sky-600 dark:text-sky-400' }
    default:
      return { label: '排队中', className: 'bg-amber-500/15 text-amber-700 dark:text-amber-400' }
  }
}

export default function ImageGeneratePage() {
  const [prompt, setPrompt] = useState('')
  const [negative, setNegative] = useState('')
  const [resolution, setResolution] = useState<(typeof RESOLUTIONS)[number]>('1K')
  const [ratio, setRatio] = useState<(typeof RATIOS)[number]>('1:1')
  const [style, setStyle] = useState<(typeof STYLES)[number]['value']>('pixel')
  const [count, setCount] = useState<(typeof COUNTS)[number]>('1')
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
  const [timeFilter, setTimeFilter] = useState('全部时间')
  const [typeFilter, setTypeFilter] = useState('全部')
  const fileInputRef = useRef<HTMLInputElement>(null)
  const abortRef = useRef<AbortController | null>(null)

  const size = useMemo(() => sizeFromResRatio(resolution, ratio), [resolution, ratio])

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

  const resetForm = () => {
    setPrompt('')
    setNegative('')
    setResolution('1K')
    setRatio('1:1')
    setStyle('pixel')
    setCount('1')
    onReferenceChange(null)
  }

  const reusePrompt = (item: HistoryItem) => {
    setPrompt(item.prompt)
    if (item.style) setStyle(item.style as typeof style)
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

  const filteredHistory = useMemo(() => {
    const now = Date.now()
    return history.filter((item) => {
      if (timeFilter === '今天' && now - item.at > 24 * 3600 * 1000) return false
      if (timeFilter === '近 7 天' && now - item.at > 7 * 24 * 3600 * 1000) return false
      return true
    })
  }, [history, timeFilter])

  const detailStatus = detail ? statusMeta(detail.status) : null
  const modeLabel = referenceFile ? '图生图' : '文生图'

  return (
    <BaseLayout title="图片生成" description="文生图 / 图生图" contentPadding="0" contentClassName="overflow-hidden">
      <div className="flex h-[calc(100vh-4rem)] min-h-[560px] overflow-hidden bg-background">
        {/* 创作侧栏 — 参考海艺；不改动应用 Sidebar / Header */}
        <aside className="flex w-[340px] shrink-0 flex-col border-r border-border bg-card">
          <div className="flex items-center justify-between border-b border-border px-4 py-3">
            <h2 className="text-sm font-semibold text-foreground">图片生成</h2>
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
                  <ImageIcon size={18} className="text-muted-foreground" />
                )}
              </div>
              <div className="min-w-0 flex-1">
                <div className="truncate text-sm font-medium text-foreground">Seedream</div>
                <div className="text-xs text-muted-foreground">{modeLabel} · {size}</div>
              </div>
              <ChevronRight size={16} className="shrink-0 text-muted-foreground" />
            </div>

            <div>
              <div className="mb-2 flex items-center justify-between">
                <span className="text-xs font-medium text-muted-foreground">{modeLabel}</span>
                <button
                  type="button"
                  className="text-xs text-muted-foreground transition hover:text-foreground disabled:opacity-50"
                  disabled={generating}
                  onClick={pickReference}
                >
                  {referenceFile ? '更换参考图' : '导入参考图'}
                </button>
              </div>
              <div className="overflow-hidden rounded-xl border border-border bg-muted/20">
                <Input.TextArea
                  value={prompt}
                  onChange={setPrompt}
                  disabled={generating}
                  placeholder="请详细描述您想要生成的画面内容（场景、人物、动作、光影等）。"
                  autoSize={{ minRows: 5, maxRows: 10 }}
                  className="!border-0 !bg-transparent !shadow-none"
                />
                <div className="flex items-center justify-between border-t border-border/60 px-2 py-1.5">
                  <button
                    type="button"
                    className="rounded-md p-1.5 text-muted-foreground transition hover:bg-muted hover:text-foreground disabled:opacity-50"
                    title="参考图"
                    disabled={generating}
                    onClick={pickReference}
                  >
                    <ImagePlus size={15} />
                  </button>
                  <div className="flex items-center gap-0.5">
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
              <input
                ref={fileInputRef}
                type="file"
                accept="image/*"
                className="hidden"
                onChange={(e) => onReferenceChange(e.target.files?.[0] ?? null)}
              />
              {referenceFile ? (
                <div className="mt-2 flex items-center justify-between rounded-lg border border-border bg-muted/30 px-2.5 py-1.5 text-xs">
                  <span className="truncate text-muted-foreground">{referenceFile.name}</span>
                  <button type="button" className="shrink-0 text-foreground hover:underline" disabled={generating} onClick={() => onReferenceChange(null)}>
                    移除
                  </button>
                </div>
              ) : null}
            </div>

            <div>
              <div className="mb-2 text-xs font-medium text-muted-foreground">反向提示词（可选）</div>
              <Input.TextArea
                value={negative}
                onChange={setNegative}
                disabled={generating}
                placeholder="模糊，低质量，文字，水印"
                autoSize={{ minRows: 2, maxRows: 3 }}
              />
            </div>

            <ChipGroup label="分辨率" options={RESOLUTIONS} value={resolution} onChange={setResolution} disabled={generating} />
            <ChipGroup label="比例" options={RATIOS} value={ratio} onChange={setRatio} disabled={generating} />
            <ChipGroup label="生成数量" options={COUNTS} value={count} onChange={setCount} disabled={generating} />
            <ChipGroup
              label="风格"
              options={STYLES.map((s) => s.value)}
              value={style}
              onChange={(v) => setStyle(v as typeof style)}
              renderLabel={(v) => STYLES.find((s) => s.value === v)?.label ?? v}
              disabled={generating}
            />

            {generating ? (
              <div className="rounded-lg border border-border bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
                排队 {queued} · 生成中 {activeCount} · 完成 {doneCount}
              </div>
            ) : null}
          </div>

          <div className="border-t border-border p-4">
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

        {/* 结果画廊 */}
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
            <select
              value={typeFilter}
              onChange={(e) => setTypeFilter(e.target.value)}
              className="h-8 rounded-lg border border-border bg-background px-2.5 text-xs text-foreground outline-none"
            >
              {['全部', '文生图', '图生图'].map((o) => (
                <option key={o} value={o}>{`生成类型: ${o}`}</option>
              ))}
            </select>
            <span className="text-xs text-muted-foreground">{filteredHistory.length} 条</span>
            <div className="ml-auto">
              <Button type="outline" size="small" onClick={clearResults} disabled={generating || results.length === 0}>
                清空本次结果
              </Button>
            </div>
          </div>

          <div className="min-h-0 flex-1 overflow-y-auto p-4">
            {results.length > 0 ? (
              <div className="mb-6">
                <div className="mb-2 text-xs font-medium text-muted-foreground">本次结果</div>
                <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-4">
                  {results.map((item, idx) => (
                    <div key={(item.jobId || item.url) + idx} className="overflow-hidden rounded-xl border border-border bg-card">
                      <img src={resolveMediaUrl(item.url)} alt={`结果 ${idx + 1}`} className="aspect-square w-full object-cover" />
                      <div className="flex items-center justify-between border-t border-border px-2 py-1.5 text-xs">
                        <span className="text-muted-foreground">#{idx + 1}</span>
                        <a
                          href={resolveMediaUrl(item.url)}
                          download={`generated-${idx + 1}.png`}
                          target="_blank"
                          rel="noreferrer"
                          className="inline-flex items-center gap-1 text-foreground hover:underline"
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
              <div className="grid h-full min-h-[280px] place-items-center text-sm text-muted-foreground">加载中…</div>
            ) : filteredHistory.length === 0 && results.length === 0 ? (
              <div className="grid h-full min-h-[280px] place-items-center">
                <Empty description="开始你的第一次创作 — 在左侧填写提示词并点击开始创作" />
              </div>
            ) : filteredHistory.length > 0 ? (
              <div className="grid grid-cols-2 gap-3 md:grid-cols-3 2xl:grid-cols-4">
                {filteredHistory.map((item) => {
                  const st = statusMeta(item.status)
                  return (
                    <article key={item.id} className="group flex flex-col overflow-hidden rounded-xl border border-border bg-card transition hover:border-foreground/25">
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
                          {item.size ? <span className="rounded bg-muted px-1.5 py-0.5">{item.size}</span> : null}
                        </div>
                        <p className="line-clamp-2 min-h-[2.25rem] text-[11px] leading-relaxed text-foreground/90">
                          {item.prompt || '无提示词'}
                        </p>
                        <div className="mt-auto flex items-center gap-1.5">
                          <Button size="sm" type="outline" className="flex-1 gap-1" leftIcon={<Eye size={12} />} onClick={() => setDetail(item)}>
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
        </section>
      </div>

      <Drawer width={480} title="创作详情" visible={!!detail} onCancel={() => setDetail(null)} footer={null} unmountOnExit>
        {detail ? (
          <div className="flex h-full flex-col gap-4">
            {detail.url && detail.status === 'succeeded' ? (
              <div className="overflow-hidden rounded-xl border border-border bg-muted/20">
                <img src={detail.url} alt="" className="mx-auto max-h-[42vh] w-full object-contain" />
              </div>
            ) : (
              <div className="grid aspect-square max-h-[280px] place-items-center rounded-xl border border-dashed border-border text-sm text-muted-foreground">
                {detail.errorMessage || '图片尚未就绪'}
              </div>
            )}
            <div className="flex flex-wrap items-center gap-2 text-xs">
              {detailStatus ? <span className={`rounded-md px-2 py-0.5 font-medium ${detailStatus.className}`}>{detailStatus.label}</span> : null}
              <span className="text-muted-foreground">{new Date(detail.at).toLocaleString()}</span>
              {detail.style ? <span className="rounded bg-muted px-2 py-0.5">{STYLE_LABELS[detail.style] || detail.style}</span> : null}
              {detail.size ? <span className="rounded bg-muted px-2 py-0.5">{detail.size}</span> : null}
            </div>
            <div className="min-h-0 flex-1">
              <div className="mb-1.5 flex items-center justify-between gap-2">
                <div className="text-sm font-medium text-foreground">提示词</div>
                <div className="flex shrink-0 gap-1.5">
                  <Button size="sm" type="outline" leftIcon={<Copy size={12} />} onClick={() => void copyPrompt(detail.prompt)}>复制</Button>
                  <Button size="sm" type="outline" leftIcon={<RotateCcw size={12} />} onClick={() => reusePrompt(detail)}>填入左侧</Button>
                </div>
              </div>
              <div className="max-h-[36vh] overflow-y-auto rounded-lg border border-border bg-background px-3 py-2 text-sm leading-relaxed whitespace-pre-wrap">
                {detail.prompt || '无提示词'}
              </div>
            </div>
            {detail.errorMessage ? (
              <div className="rounded-lg border border-rose-500/30 bg-rose-500/5 px-3 py-2 text-xs text-rose-600 dark:text-rose-400">{detail.errorMessage}</div>
            ) : null}
            <div className="mt-auto flex justify-end gap-2 border-t border-border pt-3">
              {detail.url && detail.status === 'succeeded' ? (
                <a href={detail.url} download="generated.png" target="_blank" rel="noreferrer" className="inline-flex h-8 items-center gap-1.5 rounded-md border border-border px-3 text-xs hover:bg-muted">
                  <Download size={12} />下载图片
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
