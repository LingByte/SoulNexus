import { blobToImage, canvasToBlob } from '@/lib/pixel/canvasUtils'

export type EdgeCrop = { left: number; top: number; right: number; bottom: number }

export async function cropImageBlob(blob: Blob, crop: EdgeCrop): Promise<Blob> {
  const { left, top, right, bottom } = crop
  if (left === 0 && top === 0 && right === 0 && bottom === 0) return blob
  const img = await blobToImage(blob)
  const dstW = Math.max(1, img.naturalWidth - left - right)
  const dstH = Math.max(1, img.naturalHeight - top - bottom)
  const canvas = document.createElement('canvas')
  canvas.width = dstW
  canvas.height = dstH
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('无法创建画布')
  ctx.drawImage(img, left, top, dstW, dstH, 0, 0, dstW, dstH)
  return canvasToBlob(canvas)
}

export async function cropRectBlob(
  blob: Blob,
  rect: { x: number; y: number; width: number; height: number },
): Promise<Blob> {
  const img = await blobToImage(blob)
  const x = Math.max(0, Math.floor(rect.x))
  const y = Math.max(0, Math.floor(rect.y))
  const width = Math.max(1, Math.min(img.naturalWidth - x, Math.floor(rect.width)))
  const height = Math.max(1, Math.min(img.naturalHeight - y, Math.floor(rect.height)))
  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('无法创建画布')
  ctx.drawImage(img, x, y, width, height, 0, 0, width, height)
  return canvasToBlob(canvas)
}
