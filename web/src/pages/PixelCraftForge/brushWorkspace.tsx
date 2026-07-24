import { useEffect, useMemo, useRef, useState } from 'react'
import { Brush, Eye, EyeOff, GripVertical, Lock, Move, Redo2, Trash2, Undo2, Unlock, Upload } from 'lucide-react'
import { IconFont } from '@/pages/PixelCraftForge/iconfont'
import { Button, Card, Input, Select, Tooltip } from '@/components/UI'
import { Slider } from '@arco-design/web-react'
import { cn } from '@/utils/cn'
import { addAssetFromFile, saveCanvasToLibrary, zipBlobs, triggerDownload } from '@/lib/pixelcraft/imageExport'
import { saveCanvasToLibrary as saveCanvasToLocalLibrary } from '@/lib/pixelcraft/localAssetStore'

const CANVAS_SIZE = { w: 512, h: 512 }
const MAX_UNDO = 32

export type Layer = {
  id: number
  name: string
  canvas: HTMLCanvasElement
  visible: boolean
  locked: boolean
  opacity: number
  blendMode: GlobalCompositeOperation
  transform: { x: number; y: number }
}

type BrushWorkspaceProps = {
  onOpenLibrary: (mode: 'layer' | 'ref') => void
}

function makeCanvas() {
  const canvas = document.createElement('canvas')
  canvas.width = CANVAS_SIZE.w
  canvas.height = CANVAS_SIZE.h
  return canvas
}

function createEmptyLayer(name: string, id = Date.now()): Layer {
  return { id, name, canvas: makeCanvas(), visible: true, locked: false, opacity: 1, blendMode: 'source-over', transform: { x: 0, y: 0 } }
}

function snapshotCanvas(canvas: HTMLCanvasElement) {
  const snap = makeCanvas()
  snap.getContext('2d')?.drawImage(canvas, 0, 0)
  return snap
}

function floodFillCanvas(canvas: HTMLCanvasElement, startX: number, startY: number, fillStyle: string, tolerance = 32) {
  const ctx = canvas.getContext('2d')
  if (!ctx) return false
  const { width, height } = canvas
  const img = ctx.getImageData(0, 0, width, height)
  const data = img.data
  const idx = (y: number, x: number) => (y * width + x) * 4
  const sx = Math.floor(startX)
  const sy = Math.floor(startY)
  if (sx < 0 || sy < 0 || sx >= width || sy >= height) return false
  const base = idx(sy, sx)
  const match = (i: number) => Math.abs(data[i] - data[base]) <= tolerance && Math.abs(data[i + 1] - data[base + 1]) <= tolerance && Math.abs(data[i + 2] - data[base + 2]) <= tolerance && Math.abs(data[i + 3] - data[base + 3]) <= tolerance
  if (!match(base)) return false
  const stack: Array<[number, number]> = [[sx, sy]]
  const visited = new Uint8Array(width * height)
  const fill = document.createElement('canvas').getContext('2d')
  if (!fill) return false
  fill.fillStyle = fillStyle
  fill.fillRect(0, 0, 1, 1)
  const fr = fill.getImageData(0, 0, 1, 1).data
  while (stack.length) {
    const [x, y] = stack.pop()!
    const p = idx(y, x)
    if (visited[y * width + x]) continue
    if (!match(p)) continue
    visited[y * width + x] = 1
    data[p] = fr[0]
    data[p + 1] = fr[1]
    data[p + 2] = fr[2]
    data[p + 3] = 255
    if (x > 0) stack.push([x - 1, y])
    if (x < width - 1) stack.push([x + 1, y])
    if (y > 0) stack.push([x, y - 1])
    if (y < height - 1) stack.push([x, y + 1])
  }
  ctx.putImageData(img, 0, 0)
  return true
}

const brushTools = [
  { id: 'brush', label: '画笔', icon: 'brush' },
  { id: 'eraser', label: '橡皮', icon: 'eraser' },
  { id: 'fill', label: '填充', icon: 'fill' },
  { id: 'move', label: '移动', icon: 'move' },
  { id: 'picker', label: '吸管', icon: 'picker' },
  { id: 'hand', label: '抓手', icon: 'hand' },
] as const

export default function BrushWorkspace({ onOpenLibrary }: BrushWorkspaceProps) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null)
  const [layers, setLayers] = useState<Layer[]>([createEmptyLayer('图层 1')])
  const [activeId, setActiveId] = useState<number>(layers[0].id)
  const [brushSize, setBrushSize] = useState(12)
  const [brushOpacity, setBrushOpacity] = useState(1)
  const [color, setColor] = useState('#a855f7')
  const [tool, setTool] = useState<'brush' | 'eraser' | 'fill' | 'move' | 'picker' | 'hand'>('brush')
  const [drawing, setDrawing] = useState(false)
  const [panning, setPanning] = useState(false)
  const [refOpacity, setRefOpacity] = useState(0.35)
  const [showGrid, setShowGrid] = useState(true)
  const [undoStack, setUndoStack] = useState<Array<{ layerId: number; canvas: HTMLCanvasElement }>>([])
  const [redoStack, setRedoStack] = useState<Array<{ layerId: number; canvas: HTMLCanvasElement }>>([])
  const [layersRevision, setLayersRevision] = useState(0)
  const [zoom, setZoom] = useState(1)
  const [pan, setPan] = useState({ x: 0, y: 0 })
  const refImgRef = useRef<HTMLImageElement | null>(null)
  const strokeStartedRef = useRef(false)
  const panStartRef = useRef<{ x: number; y: number } | null>(null)
  const moveStartRef = useRef<{ x: number; y: number; tx: number; ty: number } | null>(null)

  const activeLayer = layers.find((l) => l.id === activeId) ?? layers[0]
  const bumpRevision = () => setLayersRevision((v) => v + 1)

  const composite = () => {
    const out = makeCanvas()
    const ctx = out.getContext('2d')
    if (!ctx) return out
    for (const layer of layers) {
      if (!layer.visible) continue
      ctx.globalAlpha = layer.opacity ?? 1
      ctx.globalCompositeOperation = layer.blendMode || 'source-over'
      ctx.drawImage(layer.canvas, layer.transform?.x ?? 0, layer.transform?.y ?? 0)
    }
    ctx.globalAlpha = 1
    ctx.globalCompositeOperation = 'source-over'
    return out
  }

  const redrawPreview = () => {
    const canvas = canvasRef.current
    if (!canvas) return
    canvas.width = CANVAS_SIZE.w
    canvas.height = CANVAS_SIZE.h
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    ctx.clearRect(0, 0, CANVAS_SIZE.w, CANVAS_SIZE.h)
    if (showGrid) {
      ctx.fillStyle = '#ffffff'
      ctx.fillRect(0, 0, CANVAS_SIZE.w, CANVAS_SIZE.h)
    }
    if (refImgRef.current) {
      ctx.globalAlpha = refOpacity
      ctx.drawImage(refImgRef.current, 0, 0, CANVAS_SIZE.w, CANVAS_SIZE.h)
      ctx.globalAlpha = 1
    }
    ctx.drawImage(composite(), 0, 0)
  }

  useEffect(() => { redrawPreview() }, [layersRevision, showGrid, refOpacity, layers])

  const pushUndo = (layerId: number, canvasSnap: HTMLCanvasElement) => {
    setUndoStack((prev) => [...prev.slice(-MAX_UNDO + 1), { layerId, canvas: canvasSnap }])
    setRedoStack([])
  }

  const restoreLayerCanvas = (layerId: number, snap: HTMLCanvasElement) => {
    setLayers((prev) => prev.map((l) => (l.id === layerId ? { ...l, canvas: snapshotCanvas(snap) } : l)))
    bumpRevision()
  }

  const undo = () => {
    if (!undoStack.length) return
    const last = undoStack[undoStack.length - 1]
    const layer = layers.find((l) => l.id === last.layerId)
    if (layer) {
      setRedoStack((r) => [...r, { layerId: last.layerId, canvas: snapshotCanvas(layer.canvas) }])
      restoreLayerCanvas(last.layerId, last.canvas)
      setUndoStack((u) => u.slice(0, -1))
    }
  }

  const redo = () => {
    if (!redoStack.length) return
    const last = redoStack[redoStack.length - 1]
    const layer = layers.find((l) => l.id === last.layerId)
    if (layer) {
      setUndoStack((u) => [...u, { layerId: last.layerId, canvas: snapshotCanvas(layer.canvas) }])
      restoreLayerCanvas(last.layerId, last.canvas)
      setRedoStack((r) => r.slice(0, -1))
    }
  }

  const addLayer = (name?: string) => {
    const layer = createEmptyLayer(name ?? `图层 ${layers.length + 1}`)
    setLayers((prev) => [...prev, layer])
    setActiveId(layer.id)
    bumpRevision()
  }

  const duplicateLayer = (id: number) => {
    const src = layers.find((l) => l.id === id)
    if (!src) return
    const layer: Layer = { ...src, id: Date.now(), name: `${src.name} 副本`, canvas: snapshotCanvas(src.canvas), locked: false }
    const idx = layers.findIndex((l) => l.id === id)
    setLayers((prev) => [...prev.slice(0, idx + 1), layer, ...prev.slice(idx + 1)])
    setActiveId(layer.id)
    bumpRevision()
  }

  const deleteLayer = (id: number) => {
    if (layers.length <= 1) return
    setLayers((prev) => prev.filter((l) => l.id !== id))
    if (activeId === id) setActiveId(layers.find((l) => l.id !== id)?.id ?? layers[0].id)
    bumpRevision()
  }

  const moveLayerOrder = (id: number, dir: number) => {
    setLayers((prev) => {
      const idx = prev.findIndex((l) => l.id === id)
      if (idx < 0) return prev
      const next = [...prev]
      const swap = dir < 0 ? idx + 1 : idx - 1
      if (swap < 0 || swap >= next.length) return prev
      ;[next[idx], next[swap]] = [next[swap], next[idx]]
      return next
    })
    bumpRevision()
  }

  const clientToCanvas = (clientX: number, clientY: number) => {
    const canvas = canvasRef.current
    if (!canvas) return null
    const rect = canvas.getBoundingClientRect()
    const x = ((clientX - rect.left) / rect.width) * CANVAS_SIZE.w
    const y = ((clientY - rect.top) / rect.height) * CANVAS_SIZE.h
    if (x < 0 || y < 0 || x >= CANVAS_SIZE.w || y >= CANVAS_SIZE.h) return null
    return { x, y }
  }

  const paint = (clientX: number, clientY: number) => {
    if (!drawing || !activeLayer || activeLayer.locked) return
    if (tool === 'hand' || tool === 'picker' || tool === 'move' || tool === 'fill') return
    if (!strokeStartedRef.current) {
      pushUndo(activeLayer.id, snapshotCanvas(activeLayer.canvas))
      strokeStartedRef.current = true
    }
    const pt = clientToCanvas(clientX, clientY)
    if (!pt) return
    const ctx = activeLayer.canvas.getContext('2d')
    if (!ctx) return
    ctx.fillStyle = color
    ctx.globalAlpha = brushOpacity
    const size = Math.max(1, brushSize)
    if (tool === 'eraser') ctx.clearRect(pt.x - size / 2, pt.y - size / 2, size, size)
    else ctx.fillRect(pt.x - size / 2, pt.y - size / 2, size, size)
    ctx.globalAlpha = 1
    bumpRevision()
  }

  const pickColor = (clientX: number, clientY: number) => {
    const pt = clientToCanvas(clientX, clientY)
    if (!pt) return
    const merged = composite()
    const ctx = merged.getContext('2d')
    if (!ctx) return
    const p = ctx.getImageData(Math.floor(pt.x), Math.floor(pt.y), 1, 1).data
    if (p[3] === 0) return
    const hex = `#${[p[0], p[1], p[2]].map((v) => v.toString(16).padStart(2, '0')).join('')}`
    setColor(hex)
    setTool('brush')
  }

  const handlePointerDown = (e: React.PointerEvent<HTMLCanvasElement>) => {
    if (tool === 'hand') { setPanning(true); panStartRef.current = { x: e.clientX - pan.x, y: e.clientY - pan.y }; return }
    if (tool === 'picker') { pickColor(e.clientX, e.clientY); return }
    if (tool === 'move' && activeLayer) { moveStartRef.current = { x: e.clientX, y: e.clientY, tx: activeLayer.transform?.x ?? 0, ty: activeLayer.transform?.y ?? 0 }; return }
    if (tool === 'fill' && activeLayer && !activeLayer.locked) { const pt = clientToCanvas(e.clientX, e.clientY); if (!pt) return; pushUndo(activeLayer.id, snapshotCanvas(activeLayer.canvas)); floodFillCanvas(activeLayer.canvas, pt.x, pt.y, color); bumpRevision(); redrawPreview(); return }
    setDrawing(true)
    strokeStartedRef.current = false
    paint(e.clientX, e.clientY)
  }

  const handlePointerMove = (e: React.PointerEvent<HTMLCanvasElement>) => {
    if (panning && panStartRef.current) { setPan({ x: e.clientX - panStartRef.current.x, y: e.clientY - panStartRef.current.y }); return }
    if (moveStartRef.current && activeLayer) { updateLayer(activeLayer.id, { transform: { x: Math.round(moveStartRef.current.tx + (e.clientX - moveStartRef.current.x)), y: Math.round(moveStartRef.current.ty + (e.clientY - moveStartRef.current.y)) } }); return }
    if (drawing) paint(e.clientX, e.clientY)
  }

  const handlePointerUp = () => { setDrawing(false); setPanning(false); panStartRef.current = null; moveStartRef.current = null }
  const updateLayer = (id: number, patch: Partial<Layer>) => { setLayers((prev) => prev.map((l) => (l.id === id ? { ...l, ...patch } : l))); bumpRevision() }

  return <div className="space-y-4 rounded-2xl border border-border/60 bg-card/80 p-4 shadow-sm"><div className="flex flex-wrap items-center justify-between gap-3"><div><h3 className="text-sm font-semibold text-foreground">画布工作区</h3><p className="text-xs text-muted-foreground">这里接入完整 Canvas 编辑逻辑、图层管理、快捷键与导出。</p></div><div className="flex flex-wrap gap-2"><Button size="sm" variant="outline" onClick={() => addLayer()}>新建图层</Button><Button size="sm" variant="outline" onClick={() => undo()}>撤销</Button><Button size="sm" variant="outline" onClick={() => redo()}>重做</Button><Button size="sm" variant="outline" onClick={() => {}}>导出 PNG</Button></div></div><div className="grid gap-4 xl:grid-cols-[72px_minmax(0,1fr)_320px]"><aside className="flex flex-col gap-2 rounded-2xl border border-border/60 bg-background/70 p-3 shadow-sm">{brushTools.map((toolItem) => (<Tooltip key={toolItem.id} content={toolItem.label}><button type="button" className={cn('flex size-11 items-center justify-center rounded-xl border', tool === toolItem.id ? 'border-primary/40 bg-primary/10 text-primary' : 'border-border/60 bg-background/70 text-foreground')} onClick={() => setTool(toolItem.id as typeof tool)}><IconFont type={toolItem.icon} size={18} /></button></Tooltip>))}</aside><section className="space-y-4 rounded-2xl border border-border/60 bg-card/80 p-4 shadow-sm"><div className="flex flex-wrap gap-2"><Button size="sm" variant="outline" onClick={() => setShowGrid((v) => !v)}>{showGrid ? '隐藏网格' : '显示网格'}</Button><Button size="sm" variant="outline" onClick={() => addLayer()}>添加图层</Button><Button size="sm" variant="outline" onClick={() => {}}>存入素材库</Button></div><div className="flex min-h-[560px] items-center justify-center rounded-2xl border border-dashed border-border/70 bg-background/70 p-4"><canvas ref={canvasRef} width={CANVAS_SIZE.w} height={CANVAS_SIZE.h} onPointerDown={handlePointerDown} onPointerMove={handlePointerMove} onPointerUp={handlePointerUp} onPointerLeave={handlePointerUp} className="max-w-full rounded-xl border border-border/50 bg-white shadow-sm" style={{ transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})`, imageRendering: 'pixelated' }} /></div></section><aside className="space-y-4 rounded-2xl border border-border/60 bg-background/60 p-4"><div><h4 className="text-sm font-semibold text-foreground">工具属性</h4><div className="mt-3 space-y-3"><div><div className="mb-2 flex items-center justify-between text-xs text-muted-foreground"><span>画笔大小</span><span>{brushSize}px</span></div><Slider min={1} max={64} value={brushSize} onChange={setBrushSize} /></div><div><div className="mb-2 flex items-center justify-between text-xs text-muted-foreground"><span>透明度</span><span>{Math.round(brushOpacity * 100)}%</span></div><Slider min={0} max={1} step={0.01} value={brushOpacity} onChange={setBrushOpacity} /></div><div><div className="mb-2 flex items-center justify-between text-xs text-muted-foreground"><span>参考图透明度</span><span>{Math.round(refOpacity * 100)}%</span></div><Slider min={0} max={1} step={0.01} value={refOpacity} onChange={setRefOpacity} /></div></div></div><div><h4 className="text-sm font-semibold text-foreground">图层</h4><div className="mt-3 space-y-2">{layers.map((layer) => (<div key={layer.id} className={cn('rounded-xl border p-3', layer.id === activeId ? 'border-primary/40 bg-primary/10' : 'border-border/60 bg-background/70')} onClick={() => setActiveId(layer.id)}><div className="flex items-center gap-2"><GripVertical size={14} /><span className="flex-1 text-sm">{layer.name}</span><button onClick={(e) => { e.stopPropagation(); updateLayer(layer.id, { visible: !layer.visible }) }}>{layer.visible ? <Eye size={14} /> : <EyeOff size={14} />}</button><button onClick={(e) => { e.stopPropagation(); updateLayer(layer.id, { locked: !layer.locked }) }}>{layer.locked ? <Lock size={14} /> : <Unlock size={14} />}</button></div><div className="mt-2 flex items-center gap-2 text-xs text-muted-foreground"><button onClick={(e) => { e.stopPropagation(); duplicateLayer(layer.id) }}>复制</button><button onClick={(e) => { e.stopPropagation(); moveLayerOrder(layer.id, -1) }}>上移</button><button onClick={(e) => { e.stopPropagation(); moveLayerOrder(layer.id, 1) }}>下移</button><button onClick={(e) => { e.stopPropagation(); deleteLayer(layer.id) }}>删除</button></div></div>))}</div></div></aside></div></div>
}
