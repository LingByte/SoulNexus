import { getApiBaseURL } from '@/config/apiConfig'

/** Guard legacy pet.js + patch sprite frame timing (dt was ~0, sequences never advanced). */
export function guardLegacyPetJs(entry: string): string {
  if (!entry || entry.includes('liveDestroyed')) return entry
  let out = entry.replace(/app\.renderer\.resize\(/g, '(app&&app.renderer)&&app.renderer.resize(')
  if (!out.includes('drawFrameSheet') || out.includes('drawDt = lastTs')) return out

  out = out.replace('var lastTs = 0', 'var lastTs = 0\n  var drawDt = 0\n  var lastRenderedAnim = null')
  out = out.replace(
    /function draw\(ts\) \{\s*\n\s*if \(!ctx \|\| destroyed\) return\s*\n\s*lastTs = ts/,
    `function draw(ts) {
    if (!ctx || destroyed) return
    drawDt = lastTs ? (ts - lastTs) / 1000 : 0
    lastTs = ts`,
  )
  out = out.replace(
    /var dt = lastTs \? \(performance\.now\(\) - lastTs\) \/ 1000 : 0\s*\n\s*frameAcc \+= dt \* fps/,
    'frameAcc += drawDt * fps',
  )
  if (!out.includes('lastRenderedAnim')) {
    out = out.replace(
      /var animName = pickAnim\(\)\s*\n\s*var entry = animName/,
      `var animName = pickAnim()
    if (animName !== lastRenderedAnim) {
      frameIndex = 0
      frameAcc = 0
      lastRenderedAnim = animName
    }
    var entry = animName`,
    )
  }
  return out
}

/** API base for Studio preview iframe — dev uses Vite /api proxy (same origin). */
export function previewApiBase(): string {
  if (import.meta.env.DEV && typeof window !== 'undefined') {
    return `${window.location.origin}/api`
  }
  return getApiBaseURL()
}

/** Rewrite backend absolute URL to current origin (Vite proxy in dev). */
export function previewRewriteApiUrl(url: string): string {
  if (!import.meta.env.DEV || typeof window === 'undefined') return url
  return url.replace(/^https?:\/\/[^/]+/, window.location.origin)
}
