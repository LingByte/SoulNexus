import { canvasToBlob, blobToImage } from '@/lib/pixel/canvasUtils'

/** Split into N×M cells; center-crop each to cellW×cellH; reassemble (no scale). */
export async function expandShrinkImage(
  source: Blob | HTMLImageElement,
  cols: number,
  rows: number,
  cellW: number,
  cellH: number,
): Promise<Blob> {
  const img = source instanceof HTMLImageElement ? source : await blobToImage(source)
  const fullW = img.naturalWidth
  const fullH = img.naturalHeight
  const colsNum = Math.max(1, Math.floor(cols))
  const rowsNum = Math.max(1, Math.floor(rows))
  const cellSrcW = fullW / colsNum
  const cellSrcH = fullH / rowsNum
  const out = document.createElement('canvas')
  out.width = colsNum * cellW
  out.height = rowsNum * cellH
  const ctx = out.getContext('2d')
  if (!ctx) throw new Error('无法创建画布')
  for (let row = 0; row < rowsNum; row++) {
    for (let col = 0; col < colsNum; col++) {
      const sx = (col * fullW) / colsNum
      const sy = (row * fullH) / rowsNum
      const cropW = Math.min(cellW, Math.floor(cellSrcW))
      const cropH = Math.min(cellH, Math.floor(cellSrcH))
      const srcX = sx + Math.max(0, (cellSrcW - cropW) / 2)
      const srcY = sy + Math.max(0, (cellSrcH - cropH) / 2)
      const dx = col * cellW + Math.max(0, (cellW - cropW) / 2)
      const dy = row * cellH + Math.max(0, (cellH - cropH) / 2)
      ctx.drawImage(img, srcX, srcY, cropW, cropH, dx, dy, cropW, cropH)
    }
  }
  return canvasToBlob(out)
}
