import { getApiBaseURL } from '@/config/apiConfig'
import { defaultGhostSpriteStaticBase } from '@/pages/pet-market/defaultGhostSprites'
import { PANDA_SEQUENCE_MARKER } from '@/pages/pet-market/pandaSpriteCatalog'
import { defaultPandaSpriteStaticBase } from '@/pages/pet-market/defaultPandaSprites'

const SPRITE_LAZY_HELPERS = `
  function ensureLazyFrame(entry, idx) {
    if (!entry.lazyFiles) return entry.imgs && entry.imgs[idx]
    if (!entry.imgs) entry.imgs = []
    if (entry.imgs[idx]) return entry.imgs[idx]
    if (!entry._loading) entry._loading = {}
    if (entry._loading[idx]) return null
    var path = entry.lazyFiles[idx]
    if (path == null) return null
    entry._loading[idx] = true
    var img = new Image()
    img.crossOrigin = 'anonymous'
    img.onload = function () { entry.imgs[idx] = img; delete entry._loading[idx] }
    img.onerror = function () { loadErrors.push(path); delete entry._loading[idx] }
    img.src = assetUrl(path)
    return null
  }
  function prefetchLazyFrames(entry, idx, ahead) {
    var max = Math.max(1, (entry.def && entry.def.frames) || (entry.lazyFiles && entry.lazyFiles.length) || 1)
    for (var d = 0; d < ahead; d++) {
      var i = idx + d
      if (i >= 0 && i < max) ensureLazyFrame(entry, i)
    }
  }
`

function patchSpriteLazyFrames(entry: string): string {
  if (!entry.includes('function loadSheet') || entry.includes('LAZY_FRAME_THRESHOLD')) return entry
  let out = entry.replace('var loadErrors = []', 'var loadErrors = []\n  var LAZY_FRAME_THRESHOLD = 16')
  const idx = out.indexOf('function loadSheet')
  if (idx > 0) {
    out = out.slice(0, idx) + SPRITE_LAZY_HELPERS + '\n' + out.slice(idx)
  }
  out = out.replace(
    'if (def.files && def.files.length) {\n        Promise.all(def.files.map(function (path) {',
    'if (def.files && def.files.length) {\n        if (def.files.length > LAZY_FRAME_THRESHOLD) {\n          resolve({ name: name, def: def, lazyFiles: def.files, imgs: [] })\n          return\n        }\n        Promise.all(def.files.map(function (path) {',
  )
  out = out.replace(
    'var count = Math.max(1, def.frames || (entry.imgs && entry.imgs.length) || 1)',
    'var count = Math.max(1, def.frames || (entry.lazyFiles && entry.lazyFiles.length) || (entry.imgs && entry.imgs.length) || 1)',
  )
  out = out.replace(
    'if (layout.offsetY) dy += layout.offsetY\n    if (entry.imgs && entry.imgs.length) {',
    'if (layout.offsetY) dy += layout.offsetY\n    if (entry.lazyFiles && entry.lazyFiles.length) {\n      prefetchLazyFrames(entry, idx, 4)\n      var lazyImg = ensureLazyFrame(entry, idx)\n      if (lazyImg) ctx.drawImage(lazyImg, 0, 0, fw, fh, dx, dy, dw, dh)\n      return\n    }\n    if (entry.imgs && entry.imgs.length) {',
  )
  out = out.replace(
    'if (entry && (entry.img || (entry.imgs && entry.imgs.length))) drawFrameSheet(entry, w, h)',
    'if (entry && (entry.img || (entry.imgs && entry.imgs.length) || (entry.lazyFiles && entry.lazyFiles.length))) drawFrameSheet(entry, w, h)',
  )
  return out
}

/** Guard legacy pet.js + patch sprite frame timing (dt was ~0, sequences never advanced). */
export function guardLegacyPetJs(entry: string): string {
  if (!entry || entry.includes('liveDestroyed')) return entry
  let out = entry.replace(/app\.renderer\.resize\(/g, '(app&&app.renderer)&&app.renderer.resize(')
  if (out.includes('drawFrameSheet') && !out.includes('drawDt = lastTs')) {
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
  }
  return patchSpriteLazyFrames(out)
}

/** Patch ghost/sprite manifest to load frames from bundled static sprites (unsaved Studio preview). */
export function patchGhostManifestForPreview(manifestRaw: string, apiBase?: string): string {
  try {
    const m = JSON.parse(manifestRaw) as {
      type?: string
      assets?: { sprite?: { baseUrl?: string } }
    }
    if (m.type === 'live2d') return manifestRaw
    const baseUrl = m.assets?.sprite?.baseUrl || ''
    if (baseUrl.startsWith('http')) return manifestRaw
    if (!m.assets?.sprite) return manifestRaw
    const staticBase = apiBase
      ? `${apiBase.replace(/\/$/, '')}/static/pet/examples/sprites/`
      : defaultGhostSpriteStaticBase()
    m.assets.sprite.baseUrl = previewRewriteApiUrl(staticBase)
    return JSON.stringify(m)
  } catch {
    return manifestRaw
  }
}

/** Patch panda manifest to load frames from bundled static sprites. */
export function patchPandaManifestForPreview(manifestRaw: string, apiBase?: string): string {
  try {
    const m = JSON.parse(manifestRaw) as {
      assets?: { sprite?: { baseUrl?: string; animations?: { idle?: { files?: string[] } } } }
    }
    const baseUrl = m.assets?.sprite?.baseUrl || ''
    if (baseUrl.startsWith('http')) return manifestRaw
    const sample = m.assets?.sprite?.animations?.idle?.files?.[0] || ''
    if (!sample.includes(PANDA_SEQUENCE_MARKER)) return manifestRaw
    const staticBase = apiBase
      ? `${apiBase.replace(/\/$/, '')}/static/pet/examples/sprites/`
      : defaultPandaSpriteStaticBase()
    m.assets!.sprite!.baseUrl = previewRewriteApiUrl(staticBase)
    return JSON.stringify(m)
  } catch {
    return manifestRaw
  }
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
