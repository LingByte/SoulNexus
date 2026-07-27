import { blobToImage, canvasToBlob, loadImageToCanvas } from '@/lib/pixel/canvasUtils'

export type FrameSheetLayout = 'horizontal' | 'vertical' | 'grid'

export type BuildFrameSheetOptions = {
  layout?: FrameSheetLayout
  padding?: number
  maxFrameHeight?: number
  cols?: number
}

export async function buildFrameSheet(
  blobs: Blob[],
  opts: BuildFrameSheetOptions = {},
): Promise<{ blob: Blob; dataUrl: string; canvas: HTMLCanvasElement }> {
  if (!blobs.length) throw new Error('没有可合成的帧')

  const padding = opts.padding ?? 4
  const maxFrameHeight = opts.maxFrameHeight ?? 128
  const layout = opts.layout ?? 'horizontal'

  const images = await Promise.all(blobs.map(async (b) => loadImageToCanvas(await blobToImage(b))))
  const thumbH = maxFrameHeight
  const thumbs = images.map((canvas) => {
    const scale = thumbH / canvas.height
    const w = Math.max(1, Math.round(canvas.width * scale))
    const h = thumbH
    const c = document.createElement('canvas')
    c.width = w
    c.height = h
    const ctx = c.getContext('2d')
    if (!ctx) throw new Error('无法创建画布')
    ctx.drawImage(canvas, 0, 0, w, h)
    return c
  })

  const count = thumbs.length
  let cols = 1
  let rows = 1

  if (layout === 'horizontal') {
    cols = count
    rows = 1
  } else if (layout === 'vertical') {
    cols = 1
    rows = count
  } else {
    cols = opts.cols ?? Math.ceil(Math.sqrt(count))
    rows = Math.ceil(count / cols)
  }

  const cellW = Math.max(...thumbs.map((t) => t.width))
  const cellH = thumbH
  const sheet = document.createElement('canvas')
  sheet.width = cols * cellW + padding * (cols - 1)
  sheet.height = rows * cellH + padding * (rows - 1)
  const ctx = sheet.getContext('2d')
  if (!ctx) throw new Error('无法创建画布')
  ctx.clearRect(0, 0, sheet.width, sheet.height)

  for (let i = 0; i < thumbs.length; i++) {
    const c = i % cols
    const r = Math.floor(i / cols)
    const x = c * (cellW + padding) + Math.floor((cellW - thumbs[i].width) / 2)
    const y = r * (cellH + padding)
    ctx.drawImage(thumbs[i], x, y)
  }

  const blob = await canvasToBlob(sheet)
  return { blob, dataUrl: URL.createObjectURL(blob), canvas: sheet }
}
