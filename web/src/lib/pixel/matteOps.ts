import { blobToImage, canvasToBlob } from '@/lib/pixel/canvasUtils'

export type ChromaPreset = 'green' | 'blue' | 'custom'

export const CHROMA_PRESETS: Record<'green' | 'blue', { r: number; g: number; b: number }> = {
  green: { r: 0, g: 177, b: 64 },
  blue: { r: 0, g: 71, b: 187 },
}

export async function applyChromaKey(
  blob: Blob,
  bg: { r: number; g: number; b: number },
  tolerance: number,
  feather: number,
): Promise<Blob> {
  const img = await blobToImage(blob)
  const canvas = document.createElement('canvas')
  canvas.width = img.naturalWidth
  canvas.height = img.naturalHeight
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('无法创建画布')
  ctx.drawImage(img, 0, 0)
  const id = ctx.getImageData(0, 0, canvas.width, canvas.height)
  const d = id.data
  const { r: bgR, g: bgG, b: bgB } = bg
  for (let i = 0; i < d.length; i += 4) {
    const r = d[i]!
    const g = d[i + 1]!
    const b = d[i + 2]!
    const dist = Math.sqrt((r - bgR) ** 2 + (g - bgG) ** 2 + (b - bgB) ** 2)
    if (dist <= tolerance) {
      d[i + 3] = 0
    } else if (feather > 0 && dist < tolerance + feather) {
      const t = (dist - tolerance) / feather
      d[i + 3] = Math.round(255 * Math.min(1, t))
    }
  }
  ctx.putImageData(id, 0, 0)
  return canvasToBlob(canvas)
}

/** Sample top-left pixel as key color then chroma. */
export async function applyChromaFromTopLeft(
  blob: Blob,
  tolerance: number,
  feather: number,
): Promise<Blob> {
  const img = await blobToImage(blob)
  const canvas = document.createElement('canvas')
  canvas.width = img.naturalWidth
  canvas.height = img.naturalHeight
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('无法创建画布')
  ctx.drawImage(img, 0, 0)
  const { data } = ctx.getImageData(0, 0, 1, 1)
  return applyChromaKey(blob, { r: data[0]!, g: data[1]!, b: data[2]! }, tolerance, feather)
}
