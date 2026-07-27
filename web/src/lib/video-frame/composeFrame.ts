import { applyAlphaMask, blobToImage, canvasToBlob, loadImageToCanvas } from '@/lib/pixel/canvasUtils'

export async function composeTransparentPng(
  rgbBlob: Blob,
  alpha: Float32Array,
  width: number,
  height: number,
): Promise<{ blob: Blob; dataUrl: string }> {
  const img = await blobToImage(rgbBlob)
  const rgbCanvas = loadImageToCanvas(img)
  const out = applyAlphaMask(rgbCanvas, alpha, width, height)
  const blob = await canvasToBlob(out)
  return { blob, dataUrl: URL.createObjectURL(blob) }
}

export function revokeComposedUrl(dataUrl: string) {
  URL.revokeObjectURL(dataUrl)
}
