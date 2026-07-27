import { useState } from 'react'
import { Button, Card, Empty, Input } from '@/components/UI'
import { Slider } from '@arco-design/web-react'

export default function SheetWorkspace() {
  const [rows, setRows] = useState(4)
  const [cols, setCols] = useState(4)
  const [padding, setPadding] = useState(0)
  const [sheetName, setSheetName] = useState('spritesheet')

  return (
    <div className="grid gap-4 xl:grid-cols-[320px_minmax(0,1fr)_320px]">
      <aside className="space-y-4 rounded-2xl border border-border/60 bg-card/80 p-4 shadow-sm">
        <div>
          <h3 className="text-sm font-semibold text-foreground">精灵图拆分 / 合并</h3>
          <p className="mt-1 text-xs text-muted-foreground">这里接入行列拆分、序列帧合成、裁切排版与间隙设置。</p>
        </div>
        <Input placeholder="输出名称" value={sheetName} onChange={(e) => setSheetName(e.target.value)} />
        <div>
          <div className="mb-2 flex items-center justify-between text-xs text-muted-foreground"><span>行数</span><span>{rows}</span></div>
          <Slider min={1} max={16} value={rows} onChange={setRows} />
        </div>
        <div>
          <div className="mb-2 flex items-center justify-between text-xs text-muted-foreground"><span>列数</span><span>{cols}</span></div>
          <Slider min={1} max={16} value={cols} onChange={setCols} />
        </div>
        <div>
          <div className="mb-2 flex items-center justify-between text-xs text-muted-foreground"><span>间隙</span><span>{padding}px</span></div>
          <Slider min={0} max={64} value={padding} onChange={setPadding} />
        </div>
      </aside>

      <section className="space-y-4 rounded-2xl border border-border/60 bg-card/80 p-4 shadow-sm">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h3 className="text-sm font-semibold text-foreground">拆分工作区</h3>
            <p className="text-xs text-muted-foreground">当前先提供精灵图工作区壳，后续接入真实切片和合并逻辑。</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button size="sm" variant="outline">拆分图集</Button>
            <Button size="sm" variant="outline">合并帧</Button>
            <Button size="sm" variant="outline">导出 ZIP</Button>
          </div>
        </div>
        <div className="grid min-h-[520px] place-items-center rounded-2xl border border-dashed border-border/70 bg-background/70 p-8 text-center text-muted-foreground">
          <Empty title="精灵图预览区" description="拆出来的帧、切片和合成结果会显示在这里。" />
        </div>
      </section>

      <aside className="space-y-4 rounded-2xl border border-border/60 bg-background/60 p-4">
        <Card className="border border-border/60 bg-card/80 p-4 shadow-sm">
          <h4 className="text-sm font-semibold text-foreground">导出参数</h4>
          <p className="mt-2 text-sm text-muted-foreground">{sheetName} · {rows} × {cols} · 间隙 {padding}px</p>
        </Card>
        <Card className="border border-border/60 bg-card/80 p-4 shadow-sm">
          <h4 className="text-sm font-semibold text-foreground">操作说明</h4>
          <p className="mt-2 text-sm leading-6 text-muted-foreground">这里后续会接入真正的裁切、排版与帧序列导出。 </p>
        </Card>
      </aside>
    </div>
  )
}
