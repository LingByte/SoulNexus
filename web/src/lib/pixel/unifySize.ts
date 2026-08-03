import { canvasToBlob, blobToImage } from '@/lib/pixel/canvasUtils'

export type UnifyPadH = 'left' | 'center' | 'right'
export type UnifyPadV = 'top' | 'center' | 'bottom'

function cellInnerOffset(max: number, img: number, align: 'start' | 'center' | 'end'): number {
  if (align === 'start') return 0
  if (align === 'center') return Math.floor((max - img) / 2)
  return max - img
}

/** Pad images to shared max size and pack into a sheet. */
export async function unifySizeSheet(
  files: Blob[],
  cols: number,
  padH: UnifyPadH = 'center',
  padV: UnifyPadV = 'bottom',
): Promise<{ blob: Blob; maxW: number; maxH: number }> {
  if (!files.length) throw new Error('没有图片')
  const imgs = await Promise.all(files.map((f) => blobToImage(f)))
  const maxW = Math.max(...imgs.map((i) => i.naturalWidth))
  const maxH = Math.max(...imgs.map((i) => i.naturalHeight))
  const c = Math.max(1, Math.min(64, cols))
  const rows = Math.ceil(imgs.length / c)
  const canvas = document.createElement('canvas')
  canvas.width = c * maxW
  canvas.height = rows * maxH
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('无法创建画布')
  ctx.imageSmoothingEnabled = false
  ctx.clearRect(0, 0, canvas.width, canvas.height)
  const hAlign: 'start' | 'center' | 'end' = padH === 'left' ? 'start' : padH === 'right' ? 'end' : 'center'
  const vAlign: 'start' | 'center' | 'end' = padV === 'top' ? 'start' : padV === 'bottom' ? 'end' : 'center'
  for (let i = 0; i < imgs.length; i++) {
    const img = imgs[i]!
    const row = Math.floor(i / c)
    const col = i % c
    const dx = col * maxW + cellInnerOffset(maxW, img.naturalWidth, hAlign)
    const dy = row * maxH + cellInnerOffset(maxH, img.naturalHeight, vAlign)
    ctx.drawImage(img, 0, 0, img.naturalWidth, img.naturalHeight, dx, dy, img.naturalWidth, img.naturalHeight)
  }
  return { blob: await canvasToBlob(canvas), maxW, maxH }
}

/** Detect suggested uniform grid from transparent gaps. */
export function detectAutoSplit(imageData: ImageData): { cols: number; rows: number } {
  const { data, width, height } = imageData
  const transparentRows: number[] = []
  for (let y = 0; y < height; y++) {
    let all = true
    for (let x = 0; x < width; x++) {
      if (data[(y * width + x) * 4 + 3] !== 0) {
        all = false
        break
      }
    }
    if (all) transparentRows.push(y)
  }
  const transparentCols: number[] = []
  for (let x = 0; x < width; x++) {
    let all = true
    for (let y = 0; y < height; y++) {
      if (data[(y * width + x) * 4 + 3] !== 0) {
        all = false
        break
      }
    }
    if (all) transparentCols.push(x)
  }
  const runs = (arr: number[]) => {
    if (!arr.length) return [] as [number, number][]
    const out: [number, number][] = []
    let s = arr[0]!
    let e = s
    for (let i = 1; i < arr.length; i++) {
      if (arr[i] === e + 1) e = arr[i]!
      else {
        out.push([s, e])
        s = e = arr[i]!
      }
    }
    out.push([s, e])
    return out
  }
  const gaps = (r: [number, number][], total: number) => {
    if (!r.length) return [[0, total - 1] as [number, number]]
    const regions: [number, number][] = [[0, r[0]![0] - 1]]
    for (let i = 0; i < r.length - 1; i++) regions.push([r[i]![1] + 1, r[i + 1]![0] - 1])
    regions.push([r[r.length - 1]![1] + 1, total - 1])
    return regions.filter(([a, b]) => a <= b)
  }
  return {
    cols: Math.max(1, gaps(runs(transparentCols), width).length),
    rows: Math.max(1, gaps(runs(transparentRows), height).length),
  }
}

export async function detectAutoSplitFromBlob(blob: Blob): Promise<{ cols: number; rows: number }> {
  const img = await blobToImage(blob)
  const canvas = document.createElement('canvas')
  canvas.width = img.naturalWidth
  canvas.height = img.naturalHeight
  const ctx = canvas.getContext('2d')!
  ctx.drawImage(img, 0, 0)
  return detectAutoSplit(ctx.getImageData(0, 0, canvas.width, canvas.height))
}
