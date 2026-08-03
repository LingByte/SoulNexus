import { blobToImage, canvasToBlob } from '@/lib/pixel/canvasUtils'

const YIELD_BATCH = 8000

function yieldToMain(): Promise<void> {
  return new Promise((r) => setTimeout(r, 0))
}

/** Paint an inner stroke along opaque→transparent edges. */
export async function applyInnerStroke(
  blob: Blob,
  strokeWidth: number,
  strokeColor: string,
): Promise<Blob> {
  if (strokeWidth <= 0) return blob
  const img = await blobToImage(blob)
  const w = img.naturalWidth
  const h = img.naturalHeight
  const canvas = document.createElement('canvas')
  canvas.width = w
  canvas.height = h
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('无法创建画布')
  ctx.drawImage(img, 0, 0)
  const imageData = ctx.getImageData(0, 0, w, h)
  const data = imageData.data

  const alphaTransparent = 5
  const m = strokeColor.match(/^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i)
  const sr = m ? parseInt(m[1], 16) : 0
  const sg = m ? parseInt(m[2], 16) : 0
  const sb = m ? parseInt(m[3], 16) : 0

  const INF = 0xffff
  const dist = new Uint16Array(w * h)
  const total = w * h
  for (let i = 0; i < total; i += YIELD_BATCH) {
    const end = Math.min(i + YIELD_BATCH, total)
    for (let j = i; j < end; j++) {
      dist[j] = data[j * 4 + 3]! <= alphaTransparent ? 0 : INF
    }
    if (end < total) await yieldToMain()
  }

  const queue: number[] = []
  for (let i = 0; i < total; i++) {
    if (dist[i] === 0) queue.push(i)
  }
  const dx = [-1, -1, -1, 0, 0, 1, 1, 1]
  const dy = [-1, 0, 1, -1, 1, -1, 0, 1]
  while (queue.length > 0) {
    let processed = 0
    while (queue.length > 0 && processed < YIELD_BATCH) {
      const idx = queue.shift()!
      const d = dist[idx]!
      const x = idx % w
      const y = (idx / w) | 0
      for (let k = 0; k < 8; k++) {
        const nx = x + dx[k]!
        const ny = y + dy[k]!
        if (nx < 0 || nx >= w || ny < 0 || ny >= h) continue
        const ni = ny * w + nx
        if (dist[ni] !== INF) continue
        dist[ni] = d + 1
        queue.push(ni)
      }
      processed++
    }
    if (queue.length > 0) await yieldToMain()
  }

  for (let i = 0; i < total; i += YIELD_BATCH) {
    const end = Math.min(i + YIELD_BATCH, total)
    for (let j = i; j < end; j++) {
      const d = dist[j]!
      if (d > 0 && d <= strokeWidth) {
        const o = j * 4
        data[o] = sr
        data[o + 1] = sg
        data[o + 2] = sb
        data[o + 3] = 255
      }
    }
    if (end < total) await yieldToMain()
  }

  ctx.putImageData(imageData, 0, 0)
  return canvasToBlob(canvas)
}
