import { useCallback, useEffect, useRef, useState } from 'react'
import { Button, Empty, Input } from '@/components/ui'
import { Progress, Slider } from '@arco-design/web-react'
import { Download, Grid2x2, Upload } from 'lucide-react'
import {
  mergeSpritesheet,
  revokeSheetTiles,
  splitSpritesheet,
  type SheetTile,
} from '@/lib/pixel/spritesheet'
import { triggerDownload, zipBlobs } from '@/lib/pixel/imageExport'

export default function SheetWorkspace() {
  const [rows, setRows] = useState(4)
  const [cols, setCols] = useState(4)
  const [padding, setPadding] = useState(0)
  const [sheetName, setSheetName] = useState('spritesheet')
  const [sourceUrl, setSourceUrl] = useState<string | null>(null)
  const [mergedUrl, setMergedUrl] = useState<string | null>(null)
  const [mergedBlob, setMergedBlob] = useState<Blob | null>(null)
  const [tiles, setTiles] = useState<SheetTile[]>([])
  const [selected, setSelected] = useState(0)
  const [mergeFiles, setMergeFiles] = useState<File[]>([])
  const [loading, setLoading] = useState(false)
  const [progress, setProgress] = useState({ done: 0, total: 0 })
  const [error, setError] = useState('')
  const sourceFileRef = useRef<File | null>(null)

  const resetTiles = useCallback(() => {
    setTiles((prev) => {
      revokeSheetTiles(prev)
      return []
    })
    setSelected(0)
  }, [])

  useEffect(() => () => {
    revokeSheetTiles(tiles)
    if (sourceUrl) URL.revokeObjectURL(sourceUrl)
    if (mergedUrl) URL.revokeObjectURL(mergedUrl)
  }, [])

  const runSplit = async (file: File, nextRows = rows, nextCols = cols, nextPadding = padding) => {
    setError('')
    resetTiles()
    if (sourceUrl) URL.revokeObjectURL(sourceUrl)
    setSourceUrl(URL.createObjectURL(file))
    sourceFileRef.current = file
    setSheetName(file.name.replace(/\.[^.]+$/, '') || 'spritesheet')
    setLoading(true)
    try {
      const { tiles: result } = await splitSpritesheet(file, {
        rows: nextRows,
        cols: nextCols,
        padding: nextPadding,
        prefix: file.name.replace(/\.[^.]+$/, '') || 'tile',
        onProgress: (d, t) => setProgress({ done: d, total: t }),
      })
      setTiles(result)
    } catch (e) {
      setError((e as Error).message || '拆分失败')
    } finally {
      setLoading(false)
    }
  }

  const onSplit = async (file: File) => {
    await runSplit(file)
  }

  const updateGrid = async (next: { rows?: number; cols?: number; padding?: number }) => {
    const nextRows = next.rows ?? rows
    const nextCols = next.cols ?? cols
    const nextPadding = next.padding ?? padding
    if (next.rows != null) setRows(nextRows)
    if (next.cols != null) setCols(nextCols)
    if (next.padding != null) setPadding(nextPadding)
    if (sourceFileRef.current) {
      await runSplit(sourceFileRef.current, nextRows, nextCols, nextPadding)
    }
  }

  const onMerge = async () => {
    if (!mergeFiles.length) {
      setError('请先选择要合并的帧图片')
      return
    }
    setLoading(true)
    setError('')
    try {
      if (mergedUrl) URL.revokeObjectURL(mergedUrl)
      const blobs = mergeFiles.map((f) => f as Blob)
      const { dataUrl, blob } = await mergeSpritesheet(blobs, { rows, cols, padding })
      setMergedUrl(dataUrl)
      setMergedBlob(blob)
    } catch (e) {
      setError((e as Error).message || '合并失败')
    } finally {
      setLoading(false)
    }
  }

  const onExportZip = async () => {
    if (!tiles.length) return
    await zipBlobs(tiles.map((t) => ({ name: t.name, blob: t.blob })), `${sheetName}_tiles.zip`)
  }

  const onExportMerged = () => {
    if (!mergedBlob) return
    triggerDownload(mergedBlob, `${sheetName}_merged.png`)
  }

  const current = tiles[selected]

  return (
    <div className="grid gap-4 xl:grid-cols-[320px_minmax(0,1fr)_320px]">
      <aside className="space-y-4 rounded-2xl border border-border/60 bg-card/80 p-4 shadow-sm">
        <div>
          <h3 className="text-sm font-semibold text-foreground">精灵图拆分 / 合并</h3>
          <p className="mt-1 text-xs text-muted-foreground">行列拆分、序列帧合成与间隙设置。</p>
        </div>
        <label
          className="grid cursor-pointer gap-2 rounded-2xl border border-dashed border-border/70 bg-background/70 p-4 text-center"
          onDragOver={(e) => { e.preventDefault(); e.stopPropagation() }}
          onDrop={(e) => {
            e.preventDefault()
            const file = e.dataTransfer.files?.[0]
            if (file) void onSplit(file)
          }}
        >
          <Upload className="mx-auto size-6 text-primary" />
          <span className="text-sm font-medium text-foreground">导入精灵图 PNG</span>
          <input hidden type="file" accept="image/*" onChange={(e) => e.target.files?.[0] && void onSplit(e.target.files[0])} />
        </label>
        <label className="grid cursor-pointer gap-2 rounded-xl border border-border/60 bg-background/70 p-3 text-center text-xs text-muted-foreground">
          或选择多帧合并
          <input hidden multiple type="file" accept="image/*" onChange={(e) => setMergeFiles(Array.from(e.target.files ?? []))} />
          {mergeFiles.length ? `${mergeFiles.length} 张已选` : '未选择'}
        </label>
        <Input placeholder="输出名称" value={sheetName} onChange={setSheetName} />
        <div>
          <div className="mb-2 flex justify-between text-xs text-muted-foreground"><span>行数</span><span>{rows}</span></div>
          <Slider min={1} max={16} value={rows} onChange={(v) => void updateGrid({ rows: Number(v) })} />
        </div>
        <div>
          <div className="mb-2 flex justify-between text-xs text-muted-foreground"><span>列数</span><span>{cols}</span></div>
          <Slider min={1} max={16} value={cols} onChange={(v) => void updateGrid({ cols: Number(v) })} />
        </div>
        <div>
          <div className="mb-2 flex justify-between text-xs text-muted-foreground"><span>间隙</span><span>{padding}px</span></div>
          <Slider min={0} max={64} value={padding} onChange={(v) => void updateGrid({ padding: Number(v) })} />
        </div>
      </aside>

      <section className="space-y-4 rounded-2xl border border-border/60 bg-card/80 p-4 shadow-sm">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h3 className="text-sm font-semibold text-foreground">拆分工作区</h3>
            <p className="text-xs text-muted-foreground">{tiles.length ? `${tiles.length} 个切片` : '预览拆分或合并结果'}</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button size="sm" variant="outline" loading={loading} disabled={!tiles.length} leftIcon={<Download size={14} />} onClick={() => void onExportZip()}>ZIP 切片</Button>
            <Button size="sm" variant="outline" loading={loading} disabled={!mergeFiles.length} onClick={() => void onMerge()}>合并帧</Button>
            <Button size="sm" variant="outline" disabled={!mergedBlob} leftIcon={<Download size={14} />} onClick={onExportMerged}>导出合并 PNG</Button>
          </div>
        </div>
        {loading ? <Progress percent={progress.total ? Math.round((progress.done / progress.total) * 100) : 0} size="small" /> : null}
        {error ? <p className="text-xs text-red-600">{error}</p> : null}
        <div
          className="grid min-h-[420px] place-items-center rounded-2xl border border-dashed border-border/70 bg-background/70 p-4"
          style={{ backgroundImage: 'conic-gradient(hsl(var(--muted)) 90deg, hsl(var(--background)) 90deg 180deg, hsl(var(--muted)) 180deg 270deg, hsl(var(--background)) 270deg)', backgroundSize: '20px 20px' }}
        >
          {current ? (
            <img src={current.dataUrl} alt="" className="max-h-[480px] max-w-full object-contain" />
          ) : mergedUrl ? (
            <img src={mergedUrl} alt="" className="max-h-[480px] max-w-full object-contain" />
          ) : sourceUrl ? (
            <img src={sourceUrl} alt="" className="max-h-[480px] max-w-full object-contain" />
          ) : (
            <Empty preset="no-data" description="导入精灵图或帧序列" />
          )}
        </div>
        {tiles.length > 0 ? (
          <div className="flex gap-2 overflow-x-auto pb-1">
            {tiles.map((t, idx) => (
              <button key={t.name} type="button" className={`h-12 w-[72px] shrink-0 overflow-hidden rounded-md border-2 ${idx === selected ? 'border-primary' : 'border-transparent'}`} onClick={() => setSelected(idx)}>
                <img src={t.dataUrl} alt="" className="h-full w-full object-cover" />
              </button>
            ))}
          </div>
        ) : null}
      </section>

      <aside className="space-y-4 rounded-2xl border border-border/60 bg-background/60 p-4">
        <div className="rounded-2xl border border-border/60 bg-card/80 p-4">
          <h4 className="flex items-center gap-2 text-sm font-semibold text-foreground"><Grid2x2 size={14} />导出参数</h4>
          <p className="mt-2 text-sm text-muted-foreground">{sheetName} · {rows} × {cols} · 间隙 {padding}px</p>
        </div>
      </aside>
    </div>
  )
}
