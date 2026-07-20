/** Matches `.logo-brand` in index.css — keep native purple brand mark */
export const BRAND_LOGO_FILTER = 'none'
export const BRAND_LOGO_SRC = '/icon-lingyu.png'

let tintedDataUrlCache: string | null = null

function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => resolve(img)
    img.onerror = () => reject(new Error(`Failed to load image: ${src}`))
    img.src = src
  })
}

/** Renders the logo with the brand CSS filter; result is cached for favicon / PWA icons. */
export async function getBrandTintedImageDataUrl(
  src = BRAND_LOGO_SRC,
  cssFilter = BRAND_LOGO_FILTER,
): Promise<string> {
  if (tintedDataUrlCache) return tintedDataUrlCache
  const img = await loadImage(src)
  const canvas = document.createElement('canvas')
  canvas.width = img.naturalWidth || img.width
  canvas.height = img.naturalHeight || img.height
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('Canvas not supported')
  ctx.filter = cssFilter
  ctx.drawImage(img, 0, 0, canvas.width, canvas.height)
  tintedDataUrlCache = canvas.toDataURL('image/png')
  return tintedDataUrlCache
}

export function clearBrandTintedImageCache(): void {
  tintedDataUrlCache = null
}
