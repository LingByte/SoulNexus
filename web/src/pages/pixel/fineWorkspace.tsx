import { useCallback, useEffect, useRef, useState } from 'react'
import { Slider } from '@arco-design/web-react'
import { Button, Empty } from '@/components/ui'
import { Download, Eraser, Paintbrush, Pipette, Upload } from 'lucide-react'
import { canvasToBlob } from '@/lib/pixel/canvasUtils'
import { paintDisk, superEraseAt } from '@/lib/pixel/finePaint'
import { takeFineHandoffDataUrl } from '@/lib/pixel/fineHandoff'
import { triggerDownload } from '@/lib/pixel/imageExport'
import { showAlert } from '@/utils/notification'

type Tool = 'brush' | 'eraser' | 'superEraser' | 'eyedropper'

const MAX_UNDO = 30

export default function FineWorkspace() {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [hasImage, setHasImage] = useState(false)
  const [tool, setTool] = useState<Tool>('brush')
  const [brushSize, setBrushSize] = useState(8)
  const [tolerance, setTolerance] = useState(30)
  const [color, setColor] = useState({ r: 0, g: 0, b: 0, a: 255 })
  const [scale, setScale] = useState(1)
  const [offset, setOffset] = useState({ x: 0, y: 0 })
  const drawing = useRef(false)
  const panning = useRef(false)
  const panStart = useRef({ x: 0, y: 0, ox: 0, oy: 0 })
  const history = useRef<ImageData[]>([])
  const [historyLen, setHistoryLen] = useState(0)

  const getCtx = () => canvasRef.current?.getContext('2d', { willReadFrequently: true }) ?? null

  const pushHistory = useCallback(() => {
    const canvas = canvasRef.current
    const ctx = getCtx()
    if (!canvas || !ctx) return
    const snap = ctx.getImageData(0, 0, canvas.width, canvas.height)
    history.current.push(snap)
    if (history.current.length > MAX_UNDO) history.current.shift()
    setHistoryLen(history.current.length)
  }, [])

  const undo = () => {
    const canvas = canvasRef.current
    const ctx = getCtx()
    if (!canvas || !ctx || history.current.length < 2) return
    history.current.pop()
    const prev = history.current[history.current.length - 1]
    if (prev) ctx.putImageData(prev, 0, 0)
    setHistoryLen(history.current.length)
  }

  const paintImage = useCallback(
    async (url: string) => {
      const img = new Image()
      await new Promise<void>((resolve, reject) => {
        img.onload = () => resolve()
        img.onerror = () => reject(new Error('图片加载失败'))
        img.src = url
      })
      const canvas = canvasRef.current
      if (!canvas) throw new Error('画布未就绪')
      canvas.width = img.naturalWidth
      canvas.height = img.naturalHeight
      const ctx = canvas.getContext('2d', { willReadFrequently: true })
      if (!ctx) throw new Error('无法创建画布上下文')
      ctx.clearRect(0, 0, canvas.width, canvas.height)
      ctx.drawImage(img, 0, 0)
      history.current = []
      const snap = ctx.getImageData(0, 0, canvas.width, canvas.height)
      history.current.push(snap)
      setHistoryLen(1)
      setHasImage(true)
      setScale(Math.min(1, 480 / Math.max(img.naturalWidth, img.naturalHeight)))
      setOffset({ x: 0, y: 0 })
    },
    [],
  )

  useEffect(() => {
    const handoff = takeFineHandoffDataUrl()
    if (!handoff) return
    void paintImage(handoff).catch((e) => showAlert((e as Error).message, 'error'))
  }, [paintImage])

  const onUpload = (file: File) => {
    const url = URL.createObjectURL(file)
    void paintImage(url)
      .catch((e) => showAlert((e as Error).message, 'error'))
      .finally(() => URL.revokeObjectURL(url))
  }

  const screenToImage = (clientX: number, clientY: number) => {
    const canvas = canvasRef.current
    if (!canvas) return null
    const rect = canvas.getBoundingClientRect()
    const x = (clientX - rect.left) / scale
    const y = (clientY - rect.top) / scale
    return { x, y }
  }

  const applyAt = (x: number, y: number) => {
    const canvas = canvasRef.current
    const ctx = getCtx()
    if (!canvas || !ctx) return
    const id = ctx.getImageData(0, 0, canvas.width, canvas.height)
    if (tool === 'superEraser') {
      superEraseAt(id, x, y, tolerance)
    } else if (tool === 'brush') {
      paintDisk(id, x, y, brushSize / 2, color, false)
    } else if (tool === 'eraser') {
      paintDisk(id, x, y, brushSize / 2, color, true)
    } else if (tool === 'eyedropper') {
      const ix = Math.floor(x)
      const iy = Math.floor(y)
      if (ix >= 0 && iy >= 0 && ix < canvas.width && iy < canvas.height) {
        const o = (iy * canvas.width + ix) * 4
        setColor({ r: id.data[o]!, g: id.data[o + 1]!, b: id.data[o + 2]!, a: id.data[o + 3]! })
      }
      return
    }
    ctx.putImageData(id, 0, 0)
  }

  const onPointerDown = (e: React.PointerEvent) => {
    if (!hasImage) return
    if (e.button === 2 || e.altKey) {
      panning.current = true
      panStart.current = { x: e.clientX, y: e.clientY, ox: offset.x, oy: offset.y }
      return
    }
    const pt = screenToImage(e.clientX, e.clientY)
    if (!pt) return
    pushHistory()
    drawing.current = tool !== 'eyedropper' && tool !== 'superEraser'
    applyAt(pt.x, pt.y)
  }

  const onPointerMove = (e: React.PointerEvent) => {
    if (panning.current) {
      setOffset({
        x: panStart.current.ox + (e.clientX - panStart.current.x),
        y: panStart.current.oy + (e.clientY - panStart.current.y),
      })
      return
    }
    if (!drawing.current) return
    const pt = screenToImage(e.clientX, e.clientY)
    if (pt) applyAt(pt.x, pt.y)
  }

  const onPointerUp = () => {
    drawing.current = false
    panning.current = false
  }

  const onWheel = (e: React.WheelEvent) => {
    e.preventDefault()
    const delta = e.deltaY > 0 ? 0.9 : 1.1
    setScale((s) => Math.min(8, Math.max(0.1, s * delta)))
  }

  const download = async () => {
    const canvas = canvasRef.current
    if (!canvas || !hasImage) return
    const blob = await canvasToBlob(canvas)
    triggerDownload(blob, 'fine_edit.png')
  }

  const hex = `#${[color.r, color.g, color.b].map((v) => v.toString(16).padStart(2, '0')).join('')}`

  return (
    <div className="grid gap-4 xl:grid-cols-[300px_1fr]">
      <div className="space-y-3 rounded-xl border border-border bg-card p-4">
        <label className="flex cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-border px-4 py-6 text-sm text-muted-foreground hover:bg-muted/40">
          <Upload size={18} />
          上传图片
          <input hidden type="file" accept="image/*" onChange={(e) => e.target.files?.[0] && onUpload(e.target.files[0])} />
        </label>

        <div className="flex flex-wrap gap-2">
          <Button size="sm" variant={tool === 'brush' ? 'default' : 'outline'} icon={<Paintbrush size={14} />} onClick={() => setTool('brush')}>画笔</Button>
          <Button size="sm" variant={tool === 'eraser' ? 'default' : 'outline'} icon={<Eraser size={14} />} onClick={() => setTool('eraser')}>橡皮</Button>
          <Button size="sm" variant={tool === 'superEraser' ? 'default' : 'outline'} onClick={() => setTool('superEraser')}>超级橡皮</Button>
          <Button size="sm" variant={tool === 'eyedropper' ? 'default' : 'outline'} icon={<Pipette size={14} />} onClick={() => setTool('eyedropper')}>取色</Button>
        </div>

        <div>
          <div className="mb-1 flex justify-between text-xs text-muted-foreground">
            <span>笔刷大小</span>
            <span>{brushSize}</span>
          </div>
          <Slider min={1} max={64} value={brushSize} onChange={(v) => setBrushSize(Number(v))} />
        </div>

        {tool === 'superEraser' ? (
          <div>
            <div className="mb-1 flex justify-between text-xs text-muted-foreground">
              <span>容差</span>
              <span>{tolerance}</span>
            </div>
            <Slider min={1} max={100} value={tolerance} onChange={(v) => setTolerance(Number(v))} />
          </div>
        ) : null}

        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">颜色</span>
          <input
            type="color"
            value={hex}
            onChange={(e) => {
              const h = e.target.value
              setColor({
                r: parseInt(h.slice(1, 3), 16),
                g: parseInt(h.slice(3, 5), 16),
                b: parseInt(h.slice(5, 7), 16),
                a: 255,
              })
            }}
            className="h-8 w-12 cursor-pointer rounded border border-border bg-transparent"
          />
        </div>

        <Button variant="outline" disabled={historyLen < 2} onClick={undo}>撤销</Button>
        <Button type="primary" disabled={!hasImage} icon={<Download size={14} />} onClick={() => void download()}>
          下载 PNG
        </Button>
        <p className="text-xs text-muted-foreground">滚轮缩放 · Alt/右键拖拽平移 · 超级橡皮点击连通域去背</p>
      </div>

      <div
        className="relative min-h-[480px] overflow-hidden rounded-xl border border-border bg-[repeating-conic-gradient(#e5e5e5_0%_25%,#fff_0%_50%)] bg-[length:16px_16px]"
        onWheel={onWheel}
        onContextMenu={(e) => e.preventDefault()}
      >
        {!hasImage ? (
          <div className="pointer-events-none absolute inset-0 z-10 grid place-items-center">
            <Empty description="上传图片或从「像素处理」送来" />
          </div>
        ) : null}
        <div style={{ transform: `translate(${offset.x}px, ${offset.y}px)` }}>
          <canvas
            ref={canvasRef}
            className="origin-top-left cursor-crosshair"
            style={{
              transform: `scale(${scale})`,
              imageRendering: 'pixelated',
              visibility: hasImage ? 'visible' : 'hidden',
            }}
            onPointerDown={onPointerDown}
            onPointerMove={onPointerMove}
            onPointerUp={onPointerUp}
            onPointerLeave={onPointerUp}
          />
        </div>
      </div>
    </div>
  )
}
