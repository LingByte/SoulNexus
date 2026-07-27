export async function blobToImage(blob: Blob): Promise<HTMLImageElement> {
  const url = URL.createObjectURL(blob)
  try {
    return await new Promise((resolve, reject) => {
      const img = new Image()
      img.onload = () => resolve(img)
      img.onerror = reject
      img.src = url
    })
  } finally {
    URL.revokeObjectURL(url)
  }
}

export async function fileToImage(file: File | Blob): Promise<HTMLImageElement> {
  return blobToImage(file)
}

export function canvasToBlob(canvas: HTMLCanvasElement, type = 'image/png', quality = 0.92): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob((b) => (b ? resolve(b) : reject(new Error('导出失败'))), type, quality)
  })
}

export function loadImageToCanvas(img: HTMLImageElement, maxEdge?: number): HTMLCanvasElement {
  let w = img.naturalWidth
  let h = img.naturalHeight
  if (maxEdge && Math.max(w, h) > maxEdge) {
    const s = maxEdge / Math.max(w, h)
    w = Math.round(w * s)
    h = Math.round(h * s)
  }
  const canvas = document.createElement('canvas')
  canvas.width = w
  canvas.height = h
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('无法创建画布')
  ctx.drawImage(img, 0, 0, w, h)
  return canvas
}

export function readAlphaMask(canvas: HTMLCanvasElement): Float32Array {
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('无法读取画布')
  const { width, height } = canvas
  const data = ctx.getImageData(0, 0, width, height).data
  const alpha = new Float32Array(width * height)
  for (let i = 0; i < alpha.length; i++) alpha[i] = data[i * 4 + 3] / 255
  return alpha
}

export function resizeAlphaMask(
  alpha: Float32Array,
  srcW: number,
  srcH: number,
  dstW: number,
  dstH: number,
): Float32Array {
  if (srcW === dstW && srcH === dstH) return alpha

  const srcCanvas = document.createElement('canvas')
  srcCanvas.width = srcW
  srcCanvas.height = srcH
  const srcCtx = srcCanvas.getContext('2d')
  if (!srcCtx) throw new Error('无法创建画布')

  const imgData = srcCtx.createImageData(srcW, srcH)
  for (let i = 0; i < alpha.length; i++) {
    const v = Math.round(Math.min(1, Math.max(0, alpha[i])) * 255)
    const o = i * 4
    imgData.data[o] = v
    imgData.data[o + 1] = v
    imgData.data[o + 2] = v
    imgData.data[o + 3] = 255
  }
  srcCtx.putImageData(imgData, 0, 0)

  const dstCanvas = document.createElement('canvas')
  dstCanvas.width = dstW
  dstCanvas.height = dstH
  const dstCtx = dstCanvas.getContext('2d')
  if (!dstCtx) throw new Error('无法创建画布')
  dstCtx.drawImage(srcCanvas, 0, 0, dstW, dstH)
  return readAlphaMask(dstCanvas)
}

export function applyAlphaMask(
  rgbCanvas: HTMLCanvasElement,
  alpha: Float32Array,
  outW: number,
  outH: number,
): HTMLCanvasElement {
  const out = document.createElement('canvas')
  out.width = outW
  out.height = outH
  const ctx = out.getContext('2d')
  if (!ctx) throw new Error('无法创建画布')
  ctx.drawImage(rgbCanvas, 0, 0, outW, outH)
  const img = ctx.getImageData(0, 0, outW, outH)
  const pixels = outW * outH
  if (alpha.length !== pixels) {
    throw new Error(`Alpha 尺寸不匹配: ${alpha.length} vs ${pixels}`)
  }
  for (let i = 0; i < pixels; i++) {
    img.data[i * 4 + 3] = Math.round(Math.min(1, Math.max(0, alpha[i])) * 255)
  }
  ctx.putImageData(img, 0, 0)
  return out
}
