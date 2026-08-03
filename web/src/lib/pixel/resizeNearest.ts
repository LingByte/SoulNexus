import { blobToImage, canvasToBlob } from '@/lib/pixel/canvasUtils'

/** PS-style nearest-neighbor hard resize into target box. */
export async function resizeNearest(
  blob: Blob,
  targetW: number,
  targetH: number,
  keepAspect = true,
): Promise<Blob> {
  const img = await blobToImage(blob)
  const srcW = img.naturalWidth
  const srcH = img.naturalHeight
  const tmp = document.createElement('canvas')
  tmp.width = srcW
  tmp.height = srcH
  const tmpCtx = tmp.getContext('2d')
  if (!tmpCtx) throw new Error('无法创建画布')
  tmpCtx.drawImage(img, 0, 0)
  const srcData = tmpCtx.getImageData(0, 0, srcW, srcH).data

  let cw: number
  let ch: number
  let cx: number
  let cy: number
  if (keepAspect) {
    const scale = Math.min(targetW / srcW, targetH / srcH)
    cw = Math.max(1, Math.round(srcW * scale))
    ch = Math.max(1, Math.round(srcH * scale))
    cx = Math.round((targetW - cw) / 2)
    cy = Math.round((targetH - ch) / 2)
  } else {
    cw = targetW
    ch = targetH
    cx = 0
    cy = 0
  }

  const out = document.createElement('canvas')
  out.width = targetW
  out.height = targetH
  const outCtx = out.getContext('2d')
  if (!outCtx) throw new Error('无法创建画布')
  const outImg = outCtx.createImageData(targetW, targetH)
  const dst = outImg.data

  for (let dy = 0; dy < targetH; dy++) {
    for (let dx = 0; dx < targetW; dx++) {
      const dstIdx = (dy * targetW + dx) * 4
      if (dx < cx || dx >= cx + cw || dy < cy || dy >= cy + ch) {
        dst[dstIdx + 3] = 0
        continue
      }
      const rx = dx - cx
      const ry = dy - cy
      const sx = Math.min(srcW - 1, Math.max(0, Math.floor(((rx + 0.5) * srcW) / cw)))
      const sy = Math.min(srcH - 1, Math.max(0, Math.floor(((ry + 0.5) * srcH) / ch)))
      const srcIdx = (sy * srcW + sx) * 4
      dst[dstIdx] = srcData[srcIdx]
      dst[dstIdx + 1] = srcData[srcIdx + 1]
      dst[dstIdx + 2] = srcData[srcIdx + 2]
      dst[dstIdx + 3] = srcData[srcIdx + 3]
    }
  }
  outCtx.putImageData(outImg, 0, 0)
  return canvasToBlob(out)
}

export async function resizeSmooth(
  blob: Blob,
  targetW: number,
  targetH: number,
  keepAspect = true,
): Promise<Blob> {
  const img = await blobToImage(blob)
  const srcW = img.naturalWidth
  const srcH = img.naturalHeight
  let dw = targetW
  let dh = targetH
  let dx = 0
  let dy = 0
  if (keepAspect) {
    const scale = Math.min(targetW / srcW, targetH / srcH)
    dw = Math.max(1, Math.round(srcW * scale))
    dh = Math.max(1, Math.round(srcH * scale))
    dx = Math.round((targetW - dw) / 2)
    dy = Math.round((targetH - dh) / 2)
  }
  const canvas = document.createElement('canvas')
  canvas.width = targetW
  canvas.height = targetH
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('无法创建画布')
  ctx.imageSmoothingEnabled = true
  ctx.drawImage(img, dx, dy, dw, dh)
  return canvasToBlob(canvas)
}
