import { useEffect, useMemo, useState } from 'react'
import { Button } from '@/components/UI'
import { Slider } from '@arco-design/web-react'
import { Pause, Play } from 'lucide-react'

type FrameAnimationPreviewProps = {
  frames: string[]
  defaultFps?: number
  className?: string
}

/** 按 FPS 循环播放帧序列，预览连贯动作 */
export default function FrameAnimationPreview({
  frames,
  defaultFps = 8,
  className,
}: FrameAnimationPreviewProps) {
  const [index, setIndex] = useState(0)
  const [playing, setPlaying] = useState(true)
  const [previewFps, setPreviewFps] = useState(defaultFps)
  const frameKey = useMemo(() => frames.join('|'), [frames])

  useEffect(() => {
    setIndex(0)
    setPlaying(true)
  }, [frameKey])

  useEffect(() => {
    setPreviewFps(defaultFps)
  }, [defaultFps])

  useEffect(() => {
    if (!playing || frames.length < 2) return undefined
    const ms = Math.max(16, Math.round(1000 / Math.max(1, previewFps)))
    const id = window.setInterval(() => {
      setIndex((i) => (i + 1) % frames.length)
    }, ms)
    return () => window.clearInterval(id)
  }, [playing, frames.length, previewFps, frameKey])

  if (!frames.length) return null

  const safeIndex = Math.min(index, frames.length - 1)
  const current = frames[safeIndex] ?? frames[0]

  return (
    <div className={className}>
      <div className="relative grid min-h-[280px] place-items-center">
        <img
          src={current}
          alt={`帧 ${safeIndex + 1}/${frames.length}`}
          className="max-h-[420px] max-w-full object-contain"
        />
        <div className="absolute bottom-2 right-2 rounded-md bg-black/60 px-2 py-1 text-[11px] font-mono text-white">
          {safeIndex + 1} / {frames.length}
        </div>
      </div>
      <div className="mt-4 flex flex-wrap items-center gap-3">
        <Button
          size="sm"
          variant="outline"
          leftIcon={playing ? <Pause size={14} /> : <Play size={14} />}
          onClick={() => setPlaying((p) => !p)}
        >
          {playing ? '暂停' : '播放'}
        </Button>
        <div className="min-w-[140px] flex-1">
          <div className="mb-1 flex justify-between text-[11px] text-muted-foreground">
            <span>预览 FPS</span>
            <span>{previewFps}</span>
          </div>
          <Slider min={1} max={30} value={previewFps} onChange={(v) => setPreviewFps(Number(v))} />
        </div>
      </div>
    </div>
  )
}
