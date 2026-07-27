import { useMemo, useState } from 'react'
import { Button, Empty, Input } from '@/components/UI'
import { Slider } from '@arco-design/web-react'

export default function GifWorkspace() {
  const [frameGap, setFrameGap] = useState(1)
  const [startFrame, setStartFrame] = useState(0)
  const [endFrame, setEndFrame] = useState(24)
  const [size, setSize] = useState('512')

  const exportHint = useMemo(() => `${startFrame} - ${endFrame} 帧 · 间隔 ${frameGap} 帧`, [startFrame, endFrame, frameGap])

  return (
    <div className="space-y-5">
      <div className="rounded-2xl border border-border/60 bg-card/80 p-4 shadow-sm">
        <div className="grid gap-5 xl:grid-cols-[320px_minmax(0,1fr)_280px]">
          <aside className="space-y-4 rounded-2xl border border-border/60 bg-background/70 p-4">
            <div>
              <h3 className="text-sm font-semibold text-foreground">GIF 拆帧 / 合成</h3>
              <p className="mt-1 text-xs text-muted-foreground">支持 GIF、PNG 序列、ZIP 帧包。</p>
            </div>
            <label className="grid cursor-pointer gap-2 rounded-2xl border border-dashed border-border/70 bg-card/70 p-4 text-center">
              <span className="text-sm font-medium text-foreground">拖拽或点击导入 GIF</span>
              <span className="text-xs text-muted-foreground">支持 GIF、PNG 序列、ZIP 帧包</span>
              <input hidden type="file" accept="image/gif,image/png,.zip" multiple />
            </label>
            <Input placeholder="输出名称" defaultValue="animation" />
            <div className="space-y-2">
              <div className="mb-2 flex items-center justify-between text-xs text-muted-foreground"><span>画布尺寸</span><span>{size}</span></div>
              <div className="flex gap-2">
                {['256', '512', '1024'].map((item) => (
                  <button
                    key={item}
                    type="button"
                    className={`rounded-full border px-3 py-1 text-xs ${size === item ? 'border-primary bg-primary/10 text-primary' : 'border-border/60 bg-background/70 text-foreground'}`}
                    onClick={() => setSize(item)}
                  >
                    {item}
                  </button>
                ))}
              </div>
            </div>
          </aside>

          <section className="space-y-4 rounded-2xl border border-border/60 bg-background/70 p-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <h3 className="text-sm font-semibold text-foreground">帧工作区</h3>
                <p className="text-xs text-muted-foreground">拆帧结果与时间轴预览显示在这里。</p>
              </div>
              <div className="flex flex-wrap gap-2">
                <Button size="sm" variant="outline">拆帧</Button>
                <Button size="sm" variant="outline">合成 GIF</Button>
                <Button size="sm" variant="outline">导出 ZIP</Button>
              </div>
            </div>
            <div className="grid min-h-[560px] place-items-center rounded-2xl border border-dashed border-border/70 bg-card/60 p-8 text-center text-muted-foreground">
              <Empty title="GIF 帧列表" description="拆出来的帧会显示在这里。" />
            </div>
          </section>

          <aside className="space-y-4 rounded-2xl border border-border/60 bg-background/60 p-4">
            <div className="space-y-3">
              <div>
                <div className="mb-2 flex items-center justify-between text-xs text-muted-foreground"><span>帧间隔</span><span>{frameGap}</span></div>
                <Slider min={1} max={12} value={frameGap} onChange={setFrameGap} />
              </div>
              <div>
                <div className="mb-2 flex items-center justify-between text-xs text-muted-foreground"><span>开始帧</span><span>{startFrame}</span></div>
                <Slider min={0} max={100} value={startFrame} onChange={setStartFrame} />
              </div>
              <div>
                <div className="mb-2 flex items-center justify-between text-xs text-muted-foreground"><span>结束帧</span><span>{endFrame}</span></div>
                <Slider min={1} max={120} value={endFrame} onChange={setEndFrame} />
              </div>
            </div>
            <div className="rounded-2xl border border-border/60 bg-card/80 p-4">
              <h4 className="text-sm font-semibold text-foreground">当前配置</h4>
              <p className="mt-2 text-sm text-muted-foreground">{exportHint}</p>
            </div>
          </aside>
        </div>
      </div>
    </div>
  )
}
