import { useCallback, useEffect, useMemo, useState } from 'react'
import { Checkbox } from '@arco-design/web-react'
import { Button, Empty, Input } from '@/components/ui'
import { ChevronLeft, ChevronRight, Download, Upload } from 'lucide-react'
import FrameAnimationPreview from '@/components/pixel/FrameAnimationPreview'
import { encodeGif } from '@/lib/pixel/gifCodec'
import { mergeSpritesheet, revokeSheetTiles, splitSpritesheet, type SheetTile } from '@/lib/pixel/spritesheet'
import { triggerDownload, zipBlobs } from '@/lib/pixel/imageExport'
import { showAlert } from '@/utils/notification'

export default function AdjustWorkspace() {
  const [rows, setRows] = useState(4)
  const [cols, setCols] = useState(4)
  const [padding, setPadding] = useState(0)
  const [tiles, setTiles] = useState<SheetTile[]>([])
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [cursor, setCursor] = useState(0)
  const [loading, setLoading] = useState(false)
  const [fps, setFps] = useState(8)
  const [sourceName, setSourceName] = useState('sheet')

  const reset = useCallback(() => {
    setTiles((prev) => {
      revokeSheetTiles(prev)
      return []
    })
    setSelected(new Set())
    setCursor(0)
  }, [])

  useEffect(() => () => reset(), [reset])

  const runSplit = async (file: File) => {
    setLoading(true)
    try {
      reset()
      setSourceName(file.name.replace(/\.[^.]+$/, '') || 'sheet')
      const { tiles: result } = await splitSpritesheet(file, {
        rows,
        cols,
        padding,
        prefix: file.name.replace(/\.[^.]+$/, '') || 'frame',
      })
      setTiles(result)
      setSelected(new Set(result.map((t) => t.index)))
      showAlert(`已切出 ${result.length} 帧`, 'success')
    } catch (e) {
      showAlert((e as Error).message || '拆分失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  const onMultiFrames = async (files: File[]) => {
    setLoading(true)
    try {
      reset()
      setSourceName('frames')
      const next: SheetTile[] = []
      for (let i = 0; i < files.length; i++) {
        const blob = files[i]
        next.push({
          index: i + 1,
          row: 1,
          col: i + 1,
          blob,
          dataUrl: URL.createObjectURL(blob),
          name: files[i].name || `frame_${String(i + 1).padStart(3, '0')}.png`,
        })
      }
      setTiles(next)
      setSelected(new Set(next.map((t) => t.index)))
    } finally {
      setLoading(false)
    }
  }

  const selectedTiles = useMemo(
    () => tiles.filter((t) => selected.has(t.index)),
    [tiles, selected],
  )
  const previewUrls = useMemo(() => selectedTiles.map((t) => t.dataUrl), [selectedTiles])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement)?.tagName
      if (tag === 'INPUT' || tag === 'TEXTAREA') return
      if (e.key === 'a' || e.key === 'A' || e.key === 'ArrowLeft') {
        setCursor((c) => Math.max(0, c - 1))
      }
      if (e.key === 'd' || e.key === 'D' || e.key === 'ArrowRight') {
        setCursor((c) => Math.min(Math.max(0, selectedTiles.length - 1), c + 1))
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [selectedTiles.length])

  const toggle = (index: number) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(index)) next.delete(index)
      else next.add(index)
      return next
    })
  }

  const exportSheet = async () => {
    if (!selectedTiles.length) return
    setLoading(true)
    try {
      const r = Math.max(1, Math.ceil(selectedTiles.length / cols))
      const { blob } = await mergeSpritesheet(
        selectedTiles.map((t) => t.blob),
        { rows: r, cols, padding },
      )
      triggerDownload(blob, `${sourceName}_recombined.png`)
    } catch (e) {
      showAlert((e as Error).message || '导出失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  const exportZip = async () => {
    if (!selectedTiles.length) return
    await zipBlobs(
      selectedTiles.map((t) => ({ name: t.name, blob: t.blob })),
      `${sourceName}_frames.zip`,
    )
  }

  const exportGif = async () => {
    if (!selectedTiles.length) return
    setLoading(true)
    try {
      const blob = await encodeGif(
        selectedTiles.map((t) => ({ blob: t.blob })),
        { fps, loop: 0 },
      )
      triggerDownload(blob, `${sourceName}.gif`)
    } catch (e) {
      showAlert((e as Error).message || 'GIF 导出失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="grid gap-4 xl:grid-cols-[320px_1fr_300px]">
      <div className="space-y-3 rounded-xl border border-border bg-card p-4">
        <label className="flex cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-border px-4 py-6 text-sm text-muted-foreground hover:bg-muted/40">
          <Upload size={18} />
          上传 Sprite Sheet
          <input hidden type="file" accept="image/*" onChange={(e) => e.target.files?.[0] && void runSplit(e.target.files[0])} />
        </label>
        <label className="flex cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-border px-4 py-4 text-sm text-muted-foreground hover:bg-muted/40">
          或批量导入帧
          <input
            hidden
            multiple
            type="file"
            accept="image/*"
            onChange={(e) => e.target.files && void onMultiFrames(Array.from(e.target.files))}
          />
        </label>
        <div className="grid grid-cols-3 gap-2">
          <div>
            <div className="mb-1 text-xs text-muted-foreground">行</div>
            <Input type="number" value={String(rows)} onChange={(v) => setRows(Math.max(1, Number(v) || 1))} />
          </div>
          <div>
            <div className="mb-1 text-xs text-muted-foreground">列</div>
            <Input type="number" value={String(cols)} onChange={(v) => setCols(Math.max(1, Number(v) || 1))} />
          </div>
          <div>
            <div className="mb-1 text-xs text-muted-foreground">间隙</div>
            <Input type="number" value={String(padding)} onChange={(v) => setPadding(Math.max(0, Number(v) || 0))} />
          </div>
        </div>
        <div>
          <div className="mb-1 text-xs text-muted-foreground">导出 GIF FPS</div>
          <Input type="number" value={String(fps)} onChange={(v) => setFps(Math.max(1, Number(v) || 1))} />
        </div>
        <div className="flex flex-wrap gap-2">
          <Button size="sm" onClick={() => setSelected(new Set(tiles.map((t) => t.index)))}>全选</Button>
          <Button size="sm" variant="outline" onClick={() => setSelected(new Set())}>清空</Button>
        </div>
        <Button type="primary" loading={loading} disabled={!selectedTiles.length} icon={<Download size={14} />} onClick={() => void exportSheet()}>
          重组合 Sheet
        </Button>
        <Button variant="outline" disabled={!selectedTiles.length} onClick={() => void exportZip()}>导出 ZIP</Button>
        <Button variant="outline" loading={loading} disabled={!selectedTiles.length} onClick={() => void exportGif()}>导出 GIF</Button>
      </div>

      <div className="space-y-3 rounded-xl border border-border bg-card p-4">
        {previewUrls.length ? (
          <>
            <FrameAnimationPreview frames={previewUrls} defaultFps={fps} />
            <div className="flex items-center justify-center gap-2">
              <Button size="sm" variant="outline" icon={<ChevronLeft size={14} />} onClick={() => setCursor((c) => Math.max(0, c - 1))} />
              <span className="text-xs text-muted-foreground font-mono">
                {Math.min(cursor + 1, selectedTiles.length)} / {selectedTiles.length}（A/D 切换）
              </span>
              <Button
                size="sm"
                variant="outline"
                icon={<ChevronRight size={14} />}
                onClick={() => setCursor((c) => Math.min(selectedTiles.length - 1, c + 1))}
              />
            </div>
            {selectedTiles[cursor] ? (
              <div className="grid place-items-center border-t border-border pt-3">
                <img src={selectedTiles[cursor].dataUrl} alt="" className="max-h-40 object-contain" />
              </div>
            ) : null}
          </>
        ) : (
          <Empty description="上传 Sheet 或帧序列后预览" />
        )}
      </div>

      <div className="rounded-xl border border-border bg-card p-4">
        <div className="mb-2 text-sm font-medium">帧列表（勾选导出）</div>
        <div className="max-h-[560px] space-y-2 overflow-auto">
          {tiles.map((t) => (
            <label key={t.index} className="flex cursor-pointer items-center gap-2 rounded-lg border border-border/60 p-2 hover:bg-muted/30">
              <Checkbox checked={selected.has(t.index)} onChange={() => toggle(t.index)} />
              <img src={t.dataUrl} alt="" className="h-10 w-10 object-contain rounded bg-[repeating-conic-gradient(#e5e5e5_0%_25%,#fff_0%_50%)] bg-[length:8px_8px]" />
              <span className="truncate text-xs font-mono">{t.name}</span>
            </label>
          ))}
          {!tiles.length ? <Empty description="暂无帧" /> : null}
        </div>
      </div>
    </div>
  )
}
