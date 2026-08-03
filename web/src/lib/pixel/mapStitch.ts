import { canvasToBlob, blobToImage } from '@/lib/pixel/canvasUtils'

/** Grid stitch: tiles laid left-to-right, top-to-bottom with optional pixel overlap. */
export async function stitchMapGrid(tiles: Blob[], cols: number, overlap = 0): Promise<Blob> {
  if (!tiles.length) throw new Error('没有瓦片')
  const c = Math.max(1, cols)
  const imgs = await Promise.all(tiles.map((t) => blobToImage(t)))
  const tw = Math.max(...imgs.map((i) => i.naturalWidth))
  const th = Math.max(...imgs.map((i) => i.naturalHeight))
  const rows = Math.ceil(imgs.length / c)
  const ov = Math.max(0, Math.min(Math.floor(Math.min(tw, th) / 2), overlap))
  const stepX = Math.max(1, tw - ov)
  const stepY = Math.max(1, th - ov)
  const canvas = document.createElement('canvas')
  canvas.width = (c - 1) * stepX + tw
  canvas.height = (rows - 1) * stepY + th
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('无法创建画布')
  ctx.clearRect(0, 0, canvas.width, canvas.height)
  for (let i = 0; i < imgs.length; i++) {
    const img = imgs[i]!
    const col = i % c
    const row = Math.floor(i / c)
    ctx.drawImage(img, 0, 0, img.naturalWidth, img.naturalHeight, col * stepX, row * stepY, tw, th)
  }
  return canvasToBlob(canvas)
}
