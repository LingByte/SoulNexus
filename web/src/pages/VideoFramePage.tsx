import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Button, Empty, Input, Select } from '@/components/UI'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Progress, Slider } from '@arco-design/web-react'
import {
  Film,
  Grid3x3,
  Image as ImageIcon,
  LayoutGrid,
  Play,
  PlaySquare,
  Upload,
} from 'lucide-react'
import FrameAnimationPreview from '@/components/pixel/FrameAnimationPreview'
import {
  disposeVideo,
  extractVideoFrames,
  loadVideoFile,
  revokeFrameUrls,
  type ExtractMode,
  type VideoMeta,
} from '@/lib/video-frame/extractFrames'
import {
  revokeMattingFrames,
  runVideoMattingPipeline,
  type MattingFrame,
} from '@/lib/video-frame/videoPipeline'
import { SEGMENT_MODEL_OPTIONS, type SegmentModel } from '@/lib/video-frame/segmentFrame'
import { buildFrameSheet, type FrameSheetLayout } from '@/lib/pixel/frameSheet'
import { triggerDownload } from '@/lib/pixel/imageExport'

type DisplayFrame = {
  index: number
  name: string
  blob: Blob
  dataUrl: string
}

type PipelineMode = 'extract' | 'matte'
type PreviewMode = 'animate' | 'sheet' | 'single'

type ProgressState = {
  done: number
  total: number
  stage: string
}

export default function VideoFramePage() {
  const [videoUrl, setVideoUrl] = useState<string | null>(null)
  const [videoMeta, setVideoMeta] = useState<VideoMeta | null>(null)
  const [fileName, setFileName] = useState('')
  const [frames, setFrames] = useState<DisplayFrame[]>([])
  const [selectedIndex, setSelectedIndex] = useState(0)
  const [previewMode, setPreviewMode] = useState<PreviewMode>('animate')
  const [sheetPreviewUrl, setSheetPreviewUrl] = useState<string | null>(null)
  const [sheetBlob, setSheetBlob] = useState<Blob | null>(null)
  const [sheetLayout, setSheetLayout] = useState<FrameSheetLayout>('horizontal')
  const [processing, setProcessing] = useState(false)
  const [buildingSheet, setBuildingSheet] = useState(false)
  const [progress, setProgress] = useState<ProgressState>({ done: 0, total: 0, stage: '' })
  const [error, setError] = useState('')

  const [outputName, setOutputName] = useState('video_frames')
  const [pipelineMode, setPipelineMode] = useState<PipelineMode>('matte')
  const [mode, setMode] = useState<ExtractMode>('fps')
  const [fps, setFps] = useState(8)
  const [maxFrames, setMaxFrames] = useState(48)
  const [frameInterval, setFrameInterval] = useState(2)
  const [segmentModel, setSegmentModel] = useState<SegmentModel>('isnet_fp16')
  const [emaBeta, setEmaBeta] = useState(0.6)

  const videoRef = useRef<HTMLVideoElement | null>(null)
  const loadedRef = useRef<{ video: HTMLVideoElement; objectUrl: string } | null>(null)
  const framesRef = useRef(frames)
  const isMatteRef = useRef(false)
  const sheetPreviewUrlRef = useRef<string | null>(null)
  framesRef.current = frames
  sheetPreviewUrlRef.current = sheetPreviewUrl

  const revokeAll = useCallback((list: DisplayFrame[], matte: boolean) => {
    if (matte) revokeMattingFrames(list as MattingFrame[])
    else revokeFrameUrls(list as Parameters<typeof revokeFrameUrls>[0])
  }, [])

  // 仅在页面卸载时释放帧 / 视频；勿依赖 sheetPreviewUrl，否则重建帧序图会误 revoke 动作预览用的 blob
  useEffect(() => {
    return () => {
      revokeAll(framesRef.current, isMatteRef.current)
      if (loadedRef.current) disposeVideo(loadedRef.current.objectUrl)
      if (sheetPreviewUrlRef.current) URL.revokeObjectURL(sheetPreviewUrlRef.current)
    }
  }, [revokeAll])

  const rebuildSheetPreview = useCallback(async (list: DisplayFrame[], layout: FrameSheetLayout) => {
    if (!list.length) {
      setSheetPreviewUrl((prev) => {
        if (prev) URL.revokeObjectURL(prev)
        return null
      })
      setSheetBlob(null)
      return
    }
    setBuildingSheet(true)
    try {
      const { blob, dataUrl } = await buildFrameSheet(
        list.map((f) => f.blob),
        { layout, padding: 4, maxFrameHeight: pipelineMode === 'matte' ? 160 : 128 },
      )
      setSheetPreviewUrl((prev) => {
        if (prev) URL.revokeObjectURL(prev)
        return dataUrl
      })
      setSheetBlob(blob)
    } catch (e) {
      setError((e as Error).message || '帧序图生成失败')
    } finally {
      setBuildingSheet(false)
    }
  }, [pipelineMode])

  useEffect(() => {
    void rebuildSheetPreview(frames, sheetLayout)
  }, [frames, sheetLayout, rebuildSheetPreview])

  const resetFrames = useCallback((matte: boolean) => {
    setFrames((prev) => {
      revokeAll(prev, matte)
      return []
    })
    setSelectedIndex(0)
  }, [revokeAll])

  const onPickFile = async (file: File | null) => {
    if (!file) return
    setError('')
    resetFrames(isMatteRef.current)

    if (loadedRef.current) {
      disposeVideo(loadedRef.current.objectUrl)
      loadedRef.current = null
    }
    if (videoUrl) URL.revokeObjectURL(videoUrl)

    try {
      const { video, objectUrl, meta } = await loadVideoFile(file)
      loadedRef.current = { video, objectUrl }
      setVideoMeta(meta)
      setFileName(file.name)
      setOutputName(file.name.replace(/\.[^.]+$/, '') || 'video_frames')
      setVideoUrl(URL.createObjectURL(file))
    } catch (e) {
      setError((e as Error).message || '加载视频失败')
      setVideoMeta(null)
      setFileName('')
    }
  }

  const onRun = async () => {
    if (!loadedRef.current || !videoMeta) {
      setError('请先上传视频')
      return
    }
    setError('')
    setProcessing(true)
    isMatteRef.current = pipelineMode === 'matte'
    resetFrames(isMatteRef.current)
    setProgress({ done: 0, total: 0, stage: '准备' })

    const extractOpts = {
      mode,
      fps,
      frameInterval,
      maxFrames,
      onProgress: (done: number, total: number, stage: string) => setProgress({ done, total, stage }),
    }

    try {
      if (pipelineMode === 'matte') {
        const result = await runVideoMattingPipeline(loadedRef.current.video, videoMeta, {
          ...extractOpts,
          segmentModel,
          emaBeta,
          onProgress: (done, total, stage) => setProgress({ done, total, stage }),
        })
        if (!result.length) throw new Error('未生成任何透明帧')
        setFrames(result)
      } else {
        const result = await extractVideoFrames(loadedRef.current.video, videoMeta, extractOpts)
        if (!result.length) throw new Error('未拆出任何帧')
        setFrames(result)
      }
      setSelectedIndex(0)
      setPreviewMode('animate')
    } catch (e) {
      setError((e as Error).message || '处理失败')
    } finally {
      setProcessing(false)
    }
  }

  const exportFrameSheet = () => {
    if (!sheetBlob) return
    triggerDownload(sheetBlob, `${outputName}_sheet.png`)
  }

  const selected = frames[selectedIndex]
  const previewPlaybackFps =
    mode === 'fps' ? fps : Math.max(1, Math.round(30 / Math.max(1, frameInterval)))
  const previewFrameUrls = useMemo(() => frames.map((f) => f.dataUrl), [frames])

  return (
    <BaseLayout title="视频抽帧" description="上传 · 拆帧/抠图 · 预览导出（单页）">
      <div className="mx-auto grid max-w-[1400px] gap-4 lg:grid-cols-[280px_minmax(0,1fr)] lg:items-start">
        {/* Left: upload + params + actions */}
        <aside className="space-y-3 rounded-2xl border border-border/60 bg-card p-3 shadow-sm">
          <label
            className="grid cursor-pointer place-items-center gap-2 rounded-xl border border-dashed border-border bg-background px-3 py-6 text-center transition hover:border-primary/40 hover:bg-primary/5"
            onDragOver={(e) => { e.preventDefault(); e.stopPropagation() }}
            onDrop={(e) => {
              e.preventDefault()
              const file = e.dataTransfer.files?.[0]
              if (file) void onPickFile(file)
            }}
          >
            <Upload className="size-7 text-primary" />
            <div>
              <p className="text-sm font-semibold text-foreground">上传视频</p>
              <p className="mt-0.5 text-[11px] text-muted-foreground">MP4 / MOV / WebM · 可拖拽</p>
              {fileName ? <p className="mt-1 line-clamp-2 break-all text-[10px] font-mono text-muted-foreground">{fileName}</p> : null}
            </div>
            <input hidden type="file" accept="video/*" onChange={(e) => void onPickFile(e.target.files?.[0] ?? null)} />
          </label>

          {videoUrl ? (
            <div className="overflow-hidden rounded-xl border border-border/60 bg-black/90">
              <video ref={videoRef} src={videoUrl} controls className="max-h-[120px] w-full" />
              {videoMeta ? (
                <p className="px-2 py-1 text-[10px] text-muted-foreground">
                  {videoMeta.width}×{videoMeta.height} · {videoMeta.duration.toFixed(1)}s
                </p>
              ) : null}
            </div>
          ) : null}

          <div className="space-y-2.5 border-t border-border/50 pt-3">
            <div className="flex items-center gap-1.5 text-xs font-semibold text-foreground">
              <Film size={14} />
              参数
            </div>
            <Select
              value={pipelineMode}
              onChange={(v) => setPipelineMode(v as PipelineMode)}
              options={[
                { label: '抠图 + 帧序图', value: 'matte' },
                { label: '仅拆帧', value: 'extract' },
              ]}
            />
            <Input value={outputName} onChange={setOutputName} placeholder="输出名称" size="sm" />
            <Select
              value={sheetLayout}
              onChange={(v) => setSheetLayout(v as FrameSheetLayout)}
              options={[
                { label: '横向帧序图', value: 'horizontal' },
                { label: '纵向帧序图', value: 'vertical' },
                { label: '网格帧序图', value: 'grid' },
              ]}
            />
            <Select
              value={mode}
              onChange={(v) => setMode(v as ExtractMode)}
              options={[
                { label: '按目标 FPS', value: 'fps' },
                { label: '按帧间隔', value: 'frame-interval' },
              ]}
            />
            {mode === 'fps' ? (
              <div>
                <div className="mb-1 flex justify-between text-[11px] text-muted-foreground"><span>FPS</span><span>{fps}</span></div>
                <Slider min={1} max={30} value={fps} onChange={(v) => setFps(Number(v))} />
              </div>
            ) : (
              <div>
                <div className="mb-1 flex justify-between text-[11px] text-muted-foreground"><span>帧间隔</span><span>{frameInterval}</span></div>
                <Slider min={1} max={12} value={frameInterval} onChange={(v) => setFrameInterval(Number(v))} />
              </div>
            )}
            <div>
              <div className="mb-1 flex justify-between text-[11px] text-muted-foreground"><span>最大帧数</span><span>{maxFrames}</span></div>
              <Slider min={1} max={120} value={maxFrames} onChange={(v) => setMaxFrames(Number(v))} />
            </div>
            {pipelineMode === 'matte' ? (
              <>
                <Select
                  value={segmentModel}
                  onChange={(v) => setSegmentModel(v as SegmentModel)}
                  options={SEGMENT_MODEL_OPTIONS}
                />
                <div>
                  <div className="mb-1 flex justify-between text-[11px] text-muted-foreground">
                    <span>时序平滑 β</span>
                    <span>{emaBeta.toFixed(2)}</span>
                  </div>
                  <Slider min={0.3} max={0.9} step={0.05} value={emaBeta} onChange={(v) => setEmaBeta(Number(v))} />
                </div>
              </>
            ) : null}
          </div>

          {(processing || buildingSheet) ? (
            <div className="space-y-1.5">
              <Progress percent={progress.total ? Math.round((progress.done / progress.total) * 100) : 0} size="small" />
              <p className="text-[11px] text-muted-foreground">
                {buildingSheet ? '生成帧序图' : progress.stage} · {progress.done}/{progress.total || '—'}
              </p>
            </div>
          ) : (
            <p className="text-[11px] text-muted-foreground">
              {frames.length ? `已就绪 · ${frames.length} 帧` : '上传后点击开始'}
            </p>
          )}

          {error ? (
            <div className="rounded-lg border border-red-200 bg-red-50 px-2 py-1.5 text-[11px] text-red-700 dark:border-red-900/50 dark:bg-red-950/40 dark:text-red-300">
              {error}
            </div>
          ) : null}

          <div className="flex flex-col gap-2 pt-1">
            <Button
              block
              loading={processing}
              disabled={!videoMeta}
              leftIcon={<PlaySquare size={16} />}
              onClick={() => void onRun()}
            >
              {pipelineMode === 'matte' ? '开始抠图' : '开始抽帧'}
            </Button>
            <Button
              block
              variant="outline"
              disabled={!sheetBlob}
              leftIcon={<Grid3x3 size={16} />}
              onClick={exportFrameSheet}
            >
              导出帧序图 PNG
            </Button>
          </div>
        </aside>

        {/* Right: preview fills remaining viewport */}
        <section className="flex min-h-[calc(100vh-9rem)] flex-col rounded-2xl border border-border/60 bg-card p-3 shadow-sm">
          <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
            <div>
              <h3 className="text-sm font-semibold text-foreground">预览</h3>
              <p className="text-[11px] text-muted-foreground">
                {frames.length ? `共 ${frames.length} 帧` : '处理完成后在此预览'}
              </p>
            </div>
            <div className="flex flex-wrap gap-1.5">
              <Button size="sm" variant={previewMode === 'animate' ? 'primary' : 'outline'} leftIcon={<Play size={14} />} onClick={() => setPreviewMode('animate')}>
                动作
              </Button>
              <Button size="sm" variant={previewMode === 'sheet' ? 'primary' : 'outline'} leftIcon={<LayoutGrid size={14} />} onClick={() => setPreviewMode('sheet')}>
                帧序图
              </Button>
              <Button size="sm" variant={previewMode === 'single' ? 'primary' : 'outline'} leftIcon={<ImageIcon size={14} />} onClick={() => setPreviewMode('single')}>
                单帧
              </Button>
            </div>
          </div>

          <div
            className="grid min-h-0 flex-1 place-items-center overflow-auto rounded-xl border border-dashed border-border bg-background p-3"
            style={{
              backgroundImage:
                'conic-gradient(hsl(var(--muted)) 90deg, hsl(var(--background)) 90deg 180deg, hsl(var(--muted)) 180deg 270deg, hsl(var(--background)) 270deg)',
              backgroundSize: '16px 16px',
            }}
          >
            {previewMode === 'animate' ? (
              frames.length > 0 ? (
                <FrameAnimationPreview
                  frames={previewFrameUrls}
                  defaultFps={previewPlaybackFps}
                  className="w-full"
                />
              ) : (
                <Empty preset="no-data" description="上传视频并开始处理" />
              )
            ) : previewMode === 'sheet' ? (
              buildingSheet ? (
                <p className="text-sm text-muted-foreground">正在生成帧序图…</p>
              ) : sheetPreviewUrl ? (
                <img src={sheetPreviewUrl} alt="帧序图" className="max-h-[min(560px,70vh)] max-w-full object-contain" />
              ) : (
                <Empty preset="no-data" description="上传视频并开始处理" />
              )
            ) : selected ? (
              <img src={selected.dataUrl} alt={selected.name} className="max-h-[min(480px,65vh)] max-w-full object-contain" />
            ) : (
              <Empty preset="no-data" description="上传视频并开始处理" />
            )}
          </div>

          {previewMode === 'single' && frames.length > 0 ? (
            <div className="mt-3 flex gap-1.5 overflow-x-auto pb-1">
              {frames.map((frame, idx) => (
                <button
                  key={frame.name}
                  type="button"
                  className={`h-11 w-16 shrink-0 overflow-hidden rounded-md border-2 ${idx === selectedIndex ? 'border-primary' : 'border-transparent'}`}
                  onClick={() => setSelectedIndex(idx)}
                >
                  <img src={frame.dataUrl} alt="" className="h-full w-full object-cover" />
                </button>
              ))}
            </div>
          ) : null}
        </section>
      </div>
    </BaseLayout>
  )
}
