import { useEffect, useRef, useState } from 'react'
import { Modal, Button, Slider, Space, message } from 'antd'

type Props = {
  open: boolean
  title?: string
  /** Template image (edge context + transparent center) */
  imageUrl: string | null
  width: number
  height: number
  onCancel: () => void
  /** Returns painted PNG blob (white = fill regions for AI) */
  onConfirm: (blob: Blob) => void
}

/** Paint white fill regions on a transparent template before API expand. */
export default function MapStitchRegionEditor({
  open,
  title = '区域绘制',
  imageUrl,
  width,
  height,
  onCancel,
  onConfirm,
}: Props) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const drawing = useRef(false)
  const [brush, setBrush] = useState(24)
  const [mode, setMode] = useState<'paint' | 'erase'>('paint')

  useEffect(() => {
    if (!open || !imageUrl) return
    const canvas = canvasRef.current
    if (!canvas) return
    canvas.width = Math.max(1, width)
    canvas.height = Math.max(1, height)
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    ctx.clearRect(0, 0, canvas.width, canvas.height)
    const img = new Image()
    img.onload = () => {
      ctx.imageSmoothingEnabled = false
      ctx.drawImage(img, 0, 0, canvas.width, canvas.height)
    }
    img.src = imageUrl
  }, [open, imageUrl, width, height])

  const paintAt = (clientX: number, clientY: number) => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    const rect = canvas.getBoundingClientRect()
    const x = ((clientX - rect.left) / rect.width) * canvas.width
    const y = ((clientY - rect.top) / rect.height) * canvas.height
    ctx.beginPath()
    ctx.arc(x, y, brush / 2, 0, Math.PI * 2)
    if (mode === 'erase') {
      ctx.globalCompositeOperation = 'destination-out'
      ctx.fill()
      ctx.globalCompositeOperation = 'source-over'
    } else {
      ctx.fillStyle = 'rgba(255,255,255,0.92)'
      ctx.fill()
    }
  }

  const confirm = () => {
    const canvas = canvasRef.current
    if (!canvas) return
    canvas.toBlob((blob) => {
      if (!blob) {
        message.error('导出绘制失败')
        return
      }
      onConfirm(blob)
    }, 'image/png')
  }

  return (
    <Modal
      title={title}
      open={open}
      onCancel={onCancel}
      width={860}
      destroyOnClose
      footer={
        <Space>
          <Button onClick={onCancel}>取消</Button>
          <Button type="primary" onClick={confirm}>
            用于 API 生成
          </Button>
        </Space>
      }
    >
      <p style={{ marginTop: 0, color: '#666', fontSize: 13 }}>
        在透明区域涂白标记「需要生成」的范围；边缘已有像素请尽量保留，便于模型延续风格。
      </p>
      <Space wrap style={{ marginBottom: 12 }}>
        <Button type={mode === 'paint' ? 'primary' : 'default'} onClick={() => setMode('paint')}>
          涂白区域
        </Button>
        <Button type={mode === 'erase' ? 'primary' : 'default'} onClick={() => setMode('erase')}>
          擦除
        </Button>
        <span style={{ fontSize: 12 }}>笔刷 {brush}px</span>
        <Slider style={{ width: 160 }} min={4} max={96} value={brush} onChange={(v) => setBrush(Number(v))} />
      </Space>
      <div
        style={{
          border: '1px solid #ddd',
          borderRadius: 8,
          overflow: 'auto',
          maxHeight: '55vh',
          background: 'repeating-conic-gradient(#d9d0c2 0% 25%, #eee7dc 0% 50%) 50% / 16px 16px',
        }}
      >
        <canvas
          ref={canvasRef}
          style={{ display: 'block', maxWidth: '100%', cursor: 'crosshair', imageRendering: 'pixelated' }}
          onPointerDown={(e) => {
            drawing.current = true
            ;(e.target as HTMLCanvasElement).setPointerCapture(e.pointerId)
            paintAt(e.clientX, e.clientY)
          }}
          onPointerMove={(e) => {
            if (!drawing.current) return
            paintAt(e.clientX, e.clientY)
          }}
          onPointerUp={() => {
            drawing.current = false
          }}
          onPointerLeave={() => {
            drawing.current = false
          }}
        />
      </div>
    </Modal>
  )
}
