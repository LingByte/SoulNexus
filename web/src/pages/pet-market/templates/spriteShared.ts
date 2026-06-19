/** Frame-sequence sprite desktop pet — manifest-driven, no Live2D */

import {
  buildGhostAnimations,
  GHOST_BEHAVIORS,
  GHOST_EMOTION_MAP,
} from '../ghostSpriteCatalog'

export const SPRITE_ASSETS_PREFIX = 'assets/sprites/'

export const MANIFEST_FIELD_GUIDE: Record<string, string> = {
  'assets.sprite.baseUrl': '精灵资源目录，对应 assets/sprites/',
  'assets.sprite.animations': '动画名 → 帧序配置（sheet / columns / frameWidth / frames / fps）',
  'assets.sprite.defaultAnimation': '默认循环动画，通常为 idle',
  'behaviors.lipSync': 'volume=按 TTS 音量切换 talk 动画；none=关闭',
  'behaviors.talkAnimation': '说话时播放的动画名，默认 talk',
  'behaviors.dragEnabled': '是否可拖拽桌宠',
  'behaviors.clickActions': '左键单击依次播放的动作名列表（循环）',
  'behaviors.doubleTapAnimation': '双击触发的动作，默认 hide',
  emotionMap: '情绪名 → 动画名（如 joy → tap）',
}

export interface SpriteAnimationDef {
  sheet?: string
  /** Multi-file frame sequence (one PNG per frame). */
  files?: string[]
  frameWidth: number
  frameHeight: number
  frames: number
  fps?: number
  /** 雪碧图网格列数（>1 时按行优先 grid 裁剪） */
  columns?: number
  direction?: 'horizontal' | 'vertical'
  loop?: boolean
}

export function buildDefaultGhostAnimations(): Record<string, SpriteAnimationDef> {
  return buildGhostAnimations()
}

export function buildSpriteManifest(opts: {
  name: string
  animations?: Record<string, SpriteAnimationDef>
  defaultAnimation?: string
  talkAnimation?: string
  emotionMap?: Record<string, string>
  behaviors?: Record<string, unknown>
  layout?: { scale?: number; anchor?: 'bottom-center' | 'center'; offsetY?: number }
}) {
  const animations = opts.animations ?? buildDefaultGhostAnimations()

  return {
    version: 1,
    type: 'sprite',
    name: opts.name,
    fieldGuide: MANIFEST_FIELD_GUIDE,
    assets: {
      sprite: {
        baseUrl: SPRITE_ASSETS_PREFIX,
        animations,
        defaultAnimation: opts.defaultAnimation ?? 'idle',
      },
    },
    layout: {
      scale: 1,
      anchor: 'bottom-center',
      offsetY: 0,
      ...opts.layout,
    },
    emotionMap: opts.emotionMap ?? { ...GHOST_EMOTION_MAP },
    behaviors: {
      ...GHOST_BEHAVIORS,
      talkAnimation: opts.talkAnimation ?? 'talk',
      ...opts.behaviors,
    },
  }
}

export function buildSpritePetJs(name: string): string {
  return `// ${name} — 帧动画桌宠
(function initFrameSpritePet() {
  if (window.__PET_SPRITE__ && window.__PET_SPRITE__.stop) {
    window.__PET_SPRITE__.stop()
    window.__PET_SPRITE__ = null
  }

  var manifest = window.__PET_MANIFEST__ || {}
  var behaviors = manifest.behaviors || {}
  var layout = manifest.layout || {}
  var spriteAssets = (manifest.assets && manifest.assets.sprite) || {}
  var animations = spriteAssets.animations || {}
  var defaultAnim = spriteAssets.defaultAnimation || 'idle'
  var talkAnim = behaviors.talkAnimation || 'talk'
  var lipSync = behaviors.lipSync !== 'none'
  var projectBase = window.__PET_PROJECT_BASE__ || ''

  var root = document.getElementById(window.__PET_MOUNT_ID__ || 'app') || document.body
  root.classList.add('soul-pet-mount')
  root.innerHTML = ''
  if (root.style.position !== 'fixed' && root.style.position !== 'absolute') {
    root.style.position = 'relative'
    root.style.width = root.style.width || '100%'
    root.style.height = root.style.height || '100%'
    root.style.minHeight = root.style.minHeight || '100%'
  }
  root.style.overflow = 'hidden'

  var stage = document.createElement('div')
  stage.className = 'soul-pet-stage'
  stage.style.cssText = 'position:absolute;inset:0;overflow:hidden;touch-action:none;'
  root.appendChild(stage)

  var canvas = document.createElement('canvas')
  canvas.className = 'soul-pet-canvas'
  canvas.style.cssText = 'display:block;width:100%;height:100%;'
  stage.appendChild(canvas)

  var ctx = null
  var raf = 0
  var destroyed = false
  var mouthLevel = 0
  var phase = 'idle'
  var currentAnim = defaultAnim
  var drawDt = 0
  var lastTs = 0
  var lastClickAt = 0
  var onceTimer = null
  var forcedAnim = null
  var forcedUntil = 0
  var clickActionIndex = 0
  var pointerDownX = 0
  var pointerDownY = 0
  var didDragMove = false
  var dragReady = false
  var posX = 0
  var posY = 0
  var dragging = false
  var dragOffX = 0
  var dragOffY = 0
  var bounce = 0
  var sheets = {}
  var loadErrors = []

  function assetUrl(path) {
    if (!path) return path
    if (path.indexOf('http') === 0) return path
    var base = spriteAssets.baseUrl || 'assets/sprites/'
    if (base.indexOf('http') === 0) {
      return (base.endsWith('/') ? base : base + '/') + path.replace(/^\\/+/, '')
    }
    if (projectBase) {
      return projectBase + (base.charAt(0) === '/' ? base.slice(1) : base) + path.replace(/^\\/+/, '')
    }
    var host = window.SERVER_BASE || ''
    var origin = host.match(/^(https?:\\/\\/[^\\/]+)/)
    var o = origin ? origin[1] : window.location.origin
    return o + '/' + (base.charAt(0) === '/' ? base.slice(1) : base) + path.replace(/^\\/+/, '')
  }

  function loadSheet(name, def) {
    return new Promise(function (resolve) {
      if (!def) { resolve(null); return }
      if (def.files && def.files.length) {
        Promise.all(def.files.map(function (path) {
          return new Promise(function (res) {
            var img = new Image()
            img.crossOrigin = 'anonymous'
            img.onload = function () { res(img) }
            img.onerror = function () {
              loadErrors.push(path)
              res(null)
            }
            img.src = assetUrl(path)
          })
        })).then(function (imgs) {
          var list = imgs.filter(Boolean)
          if (!list.length) { resolve(null); return }
          resolve({ name: name, def: def, imgs: list })
        })
        return
      }
      if (!def.sheet) { resolve(null); return }
      var img = new Image()
      img.crossOrigin = 'anonymous'
      img.onload = function () { resolve({ name: name, def: def, img: img }) }
      img.onerror = function () {
        loadErrors.push(def.sheet)
        resolve(null)
      }
      img.src = assetUrl(def.sheet)
    })
  }

  function loadAllSheets() {
    var keys = Object.keys(animations)
    if (!keys.length) return Promise.resolve()
    return Promise.all(keys.map(function (k) { return loadSheet(k, animations[k]) })).then(function (rows) {
      rows.forEach(function (row) {
        if (row) sheets[row.name] = row
      })
    })
  }

  function resize() {
    var w = stage.clientWidth || root.clientWidth || 280
    var h = stage.clientHeight || root.clientHeight || 320
    var dpr = Math.min(window.devicePixelRatio || 1, 2)
    canvas.width = Math.max(1, Math.floor(w * dpr))
    canvas.height = Math.max(1, Math.floor(h * dpr))
    ctx = canvas.getContext('2d')
    if (ctx) ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  }

  function pickAnim() {
    var now = performance.now()
    if (forcedAnim && now < forcedUntil && sheets[forcedAnim]) return forcedAnim
    if (lipSync && mouthLevel > 0.06 && sheets[talkAnim]) return talkAnim
    if (phase === 'speaking' && sheets[talkAnim]) return talkAnim
    if (currentAnim && sheets[currentAnim]) return currentAnim
    if (sheets[defaultAnim]) return defaultAnim
    return Object.keys(sheets)[0] || null
  }

  function drawProcedural(w, h) {
    if (!ctx) return
    var cx = w / 2 + posX
    var cy = h * 0.55 + posY + bounce
    var r = Math.min(w, h) * 0.22
    var pulse = mouthLevel > 0.05 ? Math.sin(Date.now() / 100) * 2 : 0
    var grad = ctx.createRadialGradient(cx, cy - r * 0.2, r * 0.1, cx, cy, r)
    grad.addColorStop(0, '#a78bfa')
    grad.addColorStop(1, '#6366f1')
    ctx.fillStyle = grad
    ctx.beginPath()
    ctx.arc(cx, cy, r + pulse, 0, Math.PI * 2)
    ctx.fill()
    ctx.fillStyle = '#1e1b4b'
    ctx.beginPath()
    ctx.arc(cx - r * 0.28, cy - r * 0.1, r * 0.09, 0, Math.PI * 2)
    ctx.arc(cx + r * 0.28, cy - r * 0.1, r * 0.09, 0, Math.PI * 2)
    ctx.fill()
    var mouthH = 3 + mouthLevel * r * 0.2
    ctx.fillStyle = '#312e81'
    ctx.fillRect(cx - r * 0.2, cy + r * 0.18, r * 0.4, mouthH)
    ctx.fillStyle = 'rgba(255,255,255,0.55)'
    ctx.font = '11px system-ui,sans-serif'
    ctx.textAlign = 'center'
    ctx.fillText(loadErrors.length ? '请上传 assets/sprites/*.png' : '${name}', cx, h - 16)
  }

  function drawFrameSheet(entry, w, h) {
    var def = entry.def
    if (!entry._state) entry._state = { i: 0, acc: 0 }
    var st = entry._state
    var fw = def.frameWidth || 128
    var fh = def.frameHeight || 128
    var count = Math.max(1, def.frames || (entry.imgs && entry.imgs.length) || 1)
    var fps = def.fps || 8
    st.acc += drawDt * fps
    if (st.acc >= 1) {
      st.i += Math.floor(st.acc)
      st.acc -= Math.floor(st.acc)
      if (def.loop !== false) st.i = st.i % count
      else st.i = Math.min(st.i, count - 1)
    }
    var idx = Math.max(0, Math.min(count - 1, st.i))
    var scale = (layout.scale != null ? layout.scale : 1) * Math.min(w / fw, h / fh) * 0.85
    var dw = fw * scale
    var dh = fh * scale
    var anchor = layout.anchor || 'bottom-center'
    var dx = w / 2 + posX - dw / 2
    var dy = h * 0.92 + posY + bounce - dh
    if (anchor === 'center') {
      dx = w / 2 + posX - dw / 2
      dy = h / 2 + posY + bounce - dh / 2
    }
    if (layout.offsetY) dy += layout.offsetY
    if (entry.imgs && entry.imgs.length) {
      var frameImg = entry.imgs[idx] || entry.imgs[0]
      if (frameImg) ctx.drawImage(frameImg, 0, 0, fw, fh, dx, dy, dw, dh)
      return
    }
    var img = entry.img
    if (!img) return
    var col, row, sx, sy
    var gridCols = def.columns || 0
    if (gridCols > 1) {
      col = idx % gridCols
      row = Math.floor(idx / gridCols)
    } else if (def.direction === 'vertical') {
      col = 0
      row = idx
    } else {
      col = idx
      row = 0
    }
    sx = col * fw
    sy = row * fh
    ctx.drawImage(img, sx, sy, fw, fh, dx, dy, dw, dh)
  }

  function draw(ts) {
    if (!ctx || destroyed) return
    drawDt = lastTs ? (ts - lastTs) / 1000 : 0
    lastTs = ts
    var w = canvas.width / (window.devicePixelRatio || 1)
    var h = canvas.height / (window.devicePixelRatio || 1)
    ctx.clearRect(0, 0, w, h)
    if (bounce > 0) bounce = Math.max(0, bounce - 0.6)
    var animName = pickAnim()
    var entry = animName ? sheets[animName] : null
    if (entry && (entry.img || (entry.imgs && entry.imgs.length))) drawFrameSheet(entry, w, h)
    else drawProcedural(w, h)
  }

  function loop(ts) {
    draw(ts)
    raf = requestAnimationFrame(loop)
  }

  function setAnim(name, reset) {
    if (!animations[name] && !sheets[name]) return
    if (currentAnim !== name || reset) {
      currentAnim = name
      if (sheets[name]) sheets[name]._state = { i: 0, acc: 0 }
    }
  }

  function animDurationMs(def) {
    if (!def) return 600
    var n = def.frames || (def.files && def.files.length) || 1
    var fps = def.fps || 8
    return Math.max(300, Math.round((n / fps) * 1000))
  }

  function playOnceAnim(name) {
    if (!animations[name] && !sheets[name]) {
      console.warn('[${name}] unknown animation:', name)
      return
    }
    if (onceTimer) window.clearTimeout(onceTimer)
    var def = animations[name] || {}
    var ms = animDurationMs(def)
    forcedAnim = name
    forcedUntil = performance.now() + ms
    setAnim(name, true)
    onceTimer = window.setTimeout(function () {
      onceTimer = null
      forcedAnim = null
      forcedUntil = 0
      setAnim(defaultAnim, true)
    }, ms)
  }

  function nextClickAction() {
    var list = behaviors.clickActions
    if (list && list.length) {
      var name = String(list[clickActionIndex % list.length])
      clickActionIndex += 1
      return name
    }
    return (manifest.emotionMap && manifest.emotionMap.joy) || 'tap'
  }

  function scheduleAmbient() {
    var list = behaviors.ambientAnimations
    if (!list || !list.length || destroyed) return
    var range = behaviors.ambientIntervalMs || [8000, 22000]
    var lo = range[0] || 8000
    var hi = range[1] || lo + 12000
    var wait = lo + Math.random() * Math.max(500, hi - lo)
    window.setTimeout(function () {
      if (destroyed) return
      if (phase === 'speaking' || mouthLevel > 0.06 || dragging) {
        scheduleAmbient()
        return
      }
      var pick = list[Math.floor(Math.random() * list.length)]
      if (pick && pick !== defaultAnim) playOnceAnim(String(pick))
      scheduleAmbient()
    }, wait)
  }

  function setEmotion(key) {
    var map = manifest.emotionMap || {}
    var val = map[key]
    if (val && (animations[val] || sheets[val])) setAnim(String(val), true)
  }

  function setVolume(v) {
    mouthLevel = Math.min(1, Math.max(0, Number(v) || 0))
    if (lipSync && mouthLevel > 0.06) phase = 'speaking'
    else if (phase === 'speaking') phase = 'idle'
  }

  function onPointerDown(e) {
    pointerDownX = e.clientX || 0
    pointerDownY = e.clientY || 0
    didDragMove = false
    dragReady = !!behaviors.dragEnabled
    if (!behaviors.dragEnabled) return
    var rect = stage.getBoundingClientRect()
    dragOffX = (e.clientX || 0) - rect.left - rect.width / 2 - posX
    dragOffY = (e.clientY || 0) - rect.top - rect.height * 0.55 - posY
  }

  function onPointerMove(e) {
    if (!dragReady) return
    var dx = (e.clientX || 0) - pointerDownX
    var dy = (e.clientY || 0) - pointerDownY
    if (!dragging && (dx * dx + dy * dy) > 64) {
      dragging = true
      didDragMove = true
      try { stage.setPointerCapture(e.pointerId) } catch (err) { /* ignore */ }
    }
    if (!dragging) return
    var rect = stage.getBoundingClientRect()
    posX = (e.clientX || 0) - rect.left - rect.width / 2 - dragOffX
    posY = (e.clientY || 0) - rect.top - rect.height * 0.55 - dragOffY
  }

  function onPointerUp(e) {
    dragReady = false
    if (!dragging) return
    dragging = false
    try { stage.releasePointerCapture(e.pointerId) } catch (err) { /* ignore */ }
  }

  function onTap(e) {
    if (didDragMove) {
      didDragMove = false
      return
    }
    if (e && e.detail >= 2) return
    var name = nextClickAction()
    playOnceAnim(String(name))
    if (behaviors.bounceOnTap !== false) bounce = 12
  }

  function onDoubleClick(e) {
    if (didDragMove) return
    e.preventDefault()
    var dbl = behaviors.doubleTapAnimation
    if (dbl && (sheets[dbl] || animations[dbl])) playOnceAnim(String(dbl))
    else playOnceAnim('hide')
    if (behaviors.bounceOnTap !== false) bounce = 12
  }

  stage.addEventListener('pointerdown', onPointerDown)
  stage.addEventListener('pointermove', onPointerMove)
  stage.addEventListener('pointerup', onPointerUp)
  stage.addEventListener('pointercancel', onPointerUp)
  stage.addEventListener('click', onTap)
  stage.addEventListener('dblclick', onDoubleClick)

  function stop() {
    destroyed = true
    if (onceTimer) window.clearTimeout(onceTimer)
    cancelAnimationFrame(raf)
    window.removeEventListener('resize', resize)
    stage.removeEventListener('pointerdown', onPointerDown)
    stage.removeEventListener('pointermove', onPointerMove)
    stage.removeEventListener('pointerup', onPointerUp)
    stage.removeEventListener('pointercancel', onPointerUp)
    stage.removeEventListener('click', onTap)
    stage.removeEventListener('dblclick', onDoubleClick)
    root.innerHTML = ''
  }

  resize()
  window.addEventListener('resize', resize)

  loadAllSheets().finally(function () {
    if (destroyed) return
    raf = requestAnimationFrame(loop)
    scheduleAmbient()
    console.log('[${name}] Sprite pet ready', Object.keys(sheets), loadErrors.length ? 'loadErrors:' + loadErrors.join(',') : '')
  })

  window.__PET_SPRITE__ = {
    setPhase: function (p) { phase = p || 'idle' },
    setVolume: setVolume,
    setEmotion: function (key) {
      var map = manifest.emotionMap || {}
      var val = map[key]
      if (val) playOnceAnim(String(val))
    },
    playAnimation: function (n) { playOnceAnim(String(n)) },
    listAnimations: function () { return Object.keys(animations) },
    stop: stop,
  }
  window.__SOUL_PET_LIP_SYNC__ = { setVolume: setVolume }
})();

`
}

export function buildSpriteStyle(): string {
  return `.soul-pet-mount {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
  background: transparent;
}
.soul-pet-mount .soul-pet-stage,
.soul-pet-mount .soul-pet-canvas {
  touch-action: none;
}
`
}

export function buildSpriteReadme(name: string, variant = '帧动画桌宠'): string {
  return `# ${name}

## 模板：${variant}

使用 **PNG 帧图** 播放动画（支持雪碧图 sheet 或逐帧 files 列表）。

默认资源来自仓库根目录 sprites/（Ghost 角色）：

\`\`\`
assets/sprites/
  ghost_idle.png
  ghost_sing_1.png … ghost_sing_7.png   # 说话
  ghost_daze_1.png … ghost_daze_8.png     # 点击
  ghost_sad_1.png … ghost_sad_4.png     # 悲伤
\`\`\`

## manifest.json → assets.sprite.animations

每个动画：

| 字段 | 说明 |
|------|------|
| sheet | PNG 文件名 |
| frameWidth / frameHeight | 单帧宽高（像素） |
| frames | 总帧数 |
| fps | 播放帧率 |
| direction | horizontal（默认）或 vertical |
| loop | 是否循环，默认 true |

## API

\`\`\`js
window.__PET_SPRITE__.playAnimation('tap')
window.__PET_SPRITE__.setEmotion('joy')
window.__PET_SPRITE__.setVolume(0.8)  // TTS 对口型
\`\`\`

未上传 PNG 时会显示占位精灵，并在底部提示上传路径。

## 点击交互

- **左键单击**：按 behaviors.clickActions 循环播放（tap → sad → cry → hide → falldown）
- **双击**：doubleTapAnimation（默认 hide）
- 控制台：__PET_SPRITE__.playAnimation('cry') 或 listAnimations()
`
}