import { parseGIF, decompressFrames } from 'gifuct-js'
// @ts-expect-error gifenc has no types
import { GIFEncoder, quantize, applyPalette } from 'gifenc'
import { canvasToBlob } from '@/lib/pixel/canvasUtils'

export type GifFrame = {
  index: number
  blob: Blob
  dataUrl: string
  delayMs: number
  width: number
  height: number
}

export async function decodeGif(file: Blob): Promise<GifFrame[]> {
  const buf = await file.arrayBuffer()
  const gif = parseGIF(buf)
  const frames = decompressFrames(gif, true)
  if (!frames.length) throw new Error('GIF 中没有帧')

  const out: GifFrame[] = []
  const fullW = gif.lsd?.width ?? frames[0].dims.width
  const fullH = gif.lsd?.height ?? frames[0].dims.height
  const canvas = document.createElement('canvas')
  canvas.width = fullW
  canvas.height = fullH
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('无法创建画布')

  for (let i = 0; i < frames.length; i++) {
    const f = frames[i]
    const { width, height, left = 0, top = 0 } = f.dims
    const imageData = ctx.createImageData(width, height)
    imageData.data.set(f.patch)
    if (f.disposalType === 2) {
      ctx.clearRect(0, 0, fullW, fullH)
    }
    const tmp = document.createElement('canvas')
    tmp.width = width
    tmp.height = height
    const tctx = tmp.getContext('2d')
    if (!tctx) throw new Error('无法创建画布')
    tctx.putImageData(imageData, 0, 0)
    ctx.drawImage(tmp, left, top)
    const frameCanvas = document.createElement('canvas')
    frameCanvas.width = fullW
    frameCanvas.height = fullH
    const fctx = frameCanvas.getContext('2d')
    if (!fctx) throw new Error('无法创建画布')
    fctx.drawImage(canvas, 0, 0)
    const blob = await canvasToBlob(frameCanvas)
    out.push({
      index: i,
      blob,
      dataUrl: URL.createObjectURL(blob),
      delayMs: Math.max(20, f.delay || 100),
      width: fullW,
      height: fullH,
    })
  }
  return out
}

export function revokeGifFrames(frames: GifFrame[]) {
  for (const f of frames) {
    if (f.dataUrl.startsWith('blob:')) URL.revokeObjectURL(f.dataUrl)
  }
}

export async function encodeGif(
  frames: Array<{ blob: Blob; delayMs?: number }>,
  opts?: { fps?: number; loop?: number },
): Promise<Blob> {
  if (!frames.length) throw new Error('没有可编码的帧')
  const fps = opts?.fps ?? 10
  const defaultDelay = Math.round(1000 / Math.max(1, fps))
  const loop = opts?.loop ?? 0

  const bitmaps = await Promise.all(
    frames.map(async (f) => {
      if (typeof createImageBitmap === 'function') return createImageBitmap(f.blob)
      const url = URL.createObjectURL(f.blob)
      try {
        const img = await new Promise<HTMLImageElement>((resolve, reject) => {
          const im = new Image()
          im.onload = () => resolve(im)
          im.onerror = reject
          im.src = url
        })
        return img
      } finally {
        URL.revokeObjectURL(url)
      }
    }),
  )

  const w = Math.max(...bitmaps.map((b) => ('width' in b ? b.width : (b as HTMLImageElement).naturalWidth)))
  const h = Math.max(...bitmaps.map((b) => ('height' in b ? b.height : (b as HTMLImageElement).naturalHeight)))
  const canvas = document.createElement('canvas')
  canvas.width = w
  canvas.height = h
  const ctx = canvas.getContext('2d', { willReadFrequently: true })
  if (!ctx) throw new Error('无法创建画布')

  const gif = GIFEncoder()
  for (let i = 0; i < bitmaps.length; i++) {
    ctx.clearRect(0, 0, w, h)
    ctx.drawImage(bitmaps[i] as CanvasImageSource, 0, 0)
    const { data } = ctx.getImageData(0, 0, w, h)
    const palette = quantize(data, 256)
    const index = applyPalette(data, palette)
    const delay = Math.max(2, Math.round((frames[i].delayMs ?? defaultDelay) / 10))
    gif.writeFrame(index, w, h, { palette, delay, repeat: i === 0 ? loop : undefined })
  }
  gif.finish()
  return new Blob([gif.bytes()], { type: 'image/gif' })
}
