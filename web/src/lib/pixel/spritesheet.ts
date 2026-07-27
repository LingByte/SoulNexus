import { blobToImage, canvasToBlob, fileToImage, loadImageToCanvas } from '@/lib/pixel/canvasUtils'

export type SheetTile = {
  index: number
  row: number
  col: number
  blob: Blob
  dataUrl: string
  name: string
}

export type SplitSheetOptions = {
  rows: number
  cols: number
  padding?: number
  prefix?: string
  onProgress?: (done: number, total: number) => void
}

export async function splitSpritesheet(file: File, opts: SplitSheetOptions): Promise<{ source: HTMLCanvasElement; tiles: SheetTile[] }> {
  const img = await fileToImage(file)
  const source = loadImageToCanvas(img)
  const { rows, cols } = opts
  const padding = opts.padding ?? 0
  const prefix = opts.prefix ?? 'tile'

  const tileW = Math.floor((source.width - padding * (cols - 1)) / cols)
  const tileH = Math.floor((source.height - padding * (rows - 1)) / rows)
  if (tileW <= 0 || tileH <= 0) throw new Error('行列或间隙参数无效')

  const tiles: SheetTile[] = []
  const total = rows * cols
  let idx = 0

  for (let r = 0; r < rows; r++) {
    for (let c = 0; c < cols; c++) {
      const sx = c * (tileW + padding)
      const sy = r * (tileH + padding)
      const tileCanvas = document.createElement('canvas')
      tileCanvas.width = tileW
      tileCanvas.height = tileH
      const ctx = tileCanvas.getContext('2d')
      if (!ctx) throw new Error('无法创建画布')
      ctx.drawImage(source, sx, sy, tileW, tileH, 0, 0, tileW, tileH)
      const blob = await canvasToBlob(tileCanvas)
      idx++
      tiles.push({
        index: idx,
        row: r + 1,
        col: c + 1,
        blob,
        dataUrl: URL.createObjectURL(blob),
        name: `${prefix}_${String(idx).padStart(3, '0')}.png`,
      })
      opts.onProgress?.(idx, total)
    }
  }

  return { source, tiles }
}

export type MergeSheetOptions = {
  rows: number
  cols: number
  padding?: number
  tileWidth?: number
  tileHeight?: number
}

export async function mergeSpritesheet(
  blobs: Blob[],
  opts: MergeSheetOptions,
): Promise<{ blob: Blob; dataUrl: string; canvas: HTMLCanvasElement }> {
  const { rows, cols } = opts
  const padding = opts.padding ?? 0
  if (blobs.length > rows * cols) throw new Error('帧数量超过行列容量')

  const images = await Promise.all(blobs.map(async (b) => loadImageToCanvas(await blobToImage(b))))
  const tileW = opts.tileWidth ?? Math.max(...images.map((c) => c.width))
  const tileH = opts.tileHeight ?? Math.max(...images.map((c) => c.height))

  const canvas = document.createElement('canvas')
  canvas.width = cols * tileW + padding * (cols - 1)
  canvas.height = rows * tileH + padding * (rows - 1)
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('无法创建画布')
  ctx.clearRect(0, 0, canvas.width, canvas.height)

  for (let i = 0; i < blobs.length; i++) {
    const r = Math.floor(i / cols)
    const c = i % cols
    const x = c * (tileW + padding)
    const y = r * (tileH + padding)
    ctx.drawImage(images[i], x, y, tileW, tileH)
  }

  const blob = await canvasToBlob(canvas)
  return { blob, dataUrl: URL.createObjectURL(blob), canvas }
}

export function revokeSheetTiles(tiles: SheetTile[]) {
  for (const t of tiles) URL.revokeObjectURL(t.dataUrl)
}
