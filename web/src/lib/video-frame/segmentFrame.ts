import { preload, removeBackground } from '@imgly/background-removal'
import {
  blobToImage,
  loadImageToCanvas,
  readAlphaMask,
  resizeAlphaMask,
} from '@/lib/pixel/canvasUtils'

export type SegmentModel = 'isnet' | 'isnet_fp16' | 'isnet_quint8'

let preloadCache = new Map<SegmentModel, Promise<void>>()

export async function preloadSegmentModel(model: SegmentModel = 'isnet_fp16') {
  if (!preloadCache.has(model)) {
    preloadCache.set(model, preload({ model }))
  }
  await preloadCache.get(model)
}

export async function segmentFrameBlob(
  blob: Blob,
  model: SegmentModel = 'isnet_fp16',
): Promise<{ alpha: Float32Array; width: number; height: number }> {
  await preloadSegmentModel(model)

  const origImg = await blobToImage(blob)
  const origW = origImg.naturalWidth
  const origH = origImg.naturalHeight

  const url = URL.createObjectURL(blob)
  try {
    const outBlob = await removeBackground(url, {
      model,
      output: { format: 'image/png', quality: 1 },
    })
    const segImg = await blobToImage(outBlob)
    const segCanvas = loadImageToCanvas(segImg)
    let alpha = readAlphaMask(segCanvas)

    if (segCanvas.width !== origW || segCanvas.height !== origH) {
      alpha = resizeAlphaMask(alpha, segCanvas.width, segCanvas.height, origW, origH)
    }

    return { alpha, width: origW, height: origH }
  } finally {
    URL.revokeObjectURL(url)
  }
}

export const SEGMENT_MODEL_OPTIONS = [
  { label: 'isnet（默认，动漫/通用）', value: 'isnet' as SegmentModel },
  { label: 'isnet_fp16（更快）', value: 'isnet_fp16' as SegmentModel },
  { label: 'isnet_quint8（最快）', value: 'isnet_quint8' as SegmentModel },
]
