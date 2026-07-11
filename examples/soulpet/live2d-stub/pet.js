/**
 * Live2D-compatible placeholder runtime — manifest-driven.
 * Replace draw() with Cubism Web SDK when model files are present.
 * Exposes window.__PET_LIVE2D__ (aligned with __SOUL_PET__ SDK).
 */
(function () {
  'use strict'
  var manifest = window.__PET_MANIFEST__ || {}
  var cfg = manifest.assets && manifest.assets.live2d
  if (!cfg) {
    console.error('[Live2D Pet] missing assets.live2d in manifest')
    return
  }

  var desktopMode = (window.__PET_EMBED_MODE__ || (window.__AIPetConfig || {}).mode) !== 'widget'
  var mountId = desktopMode ? 'soul-pet-desktop-root' : 'app'
  var root = document.getElementById(mountId) || document.body
  if (desktopMode && root.id !== 'soul-pet-desktop-root') {
    root = document.createElement('div')
    root.id = 'soul-pet-desktop-root'
    root.className = 'soul-pet-mount soul-pet-desktop'
    document.body.appendChild(root)
  }

  var stage = document.createElement('div')
  stage.className = 'soul-pet-stage'
  var canvas = document.createElement('canvas')
  canvas.className = 'soul-pet-canvas'
  stage.appendChild(canvas)
  root.appendChild(stage)

  var ctx = canvas.getContext('2d')
  var destroyed = false
  var currentMotion = 'idle'
  var currentExpr = ''
  var lipLevel = 0
  var posX = 0
  var posY = 0
  var dragging = false
  var dragOffX = 0
  var dragOffY = 0
  var projectBase = window.__PET_PROJECT_BASE__ || ''

  function resize() {
    var w = window.innerWidth
    var h = window.innerHeight
    canvas.width = w
    canvas.height = h
    canvas.style.width = w + 'px'
    canvas.style.height = h + 'px'
  }

  function modelLabel() {
    return (cfg.baseUrl || '') + (cfg.model || 'model.model3.json')
  }

  function draw() {
    if (!ctx || destroyed) return
    var w = canvas.width
    var h = canvas.height
    ctx.clearRect(0, 0, w, h)
    var scale = (manifest.layout && manifest.layout.scale) || 0.35
    var cx = w / 2 + posX
    var cy = h * 0.55 + posY
    var rw = 180 * scale * 4
    var rh = 240 * scale * 4

    ctx.fillStyle = 'rgba(99,102,241,0.15)'
    ctx.strokeStyle = 'rgba(99,102,241,0.6)'
    ctx.lineWidth = 2
    ctx.beginPath()
    ctx.roundRect(cx - rw / 2, cy - rh / 2, rw, rh, 16)
    ctx.fill()
    ctx.stroke()

    ctx.fillStyle = '#4338ca'
    ctx.font = '600 14px system-ui,sans-serif'
    ctx.textAlign = 'center'
    ctx.fillText('Live2D', cx, cy - 20)
    ctx.font = '12px system-ui,sans-serif'
    ctx.fillStyle = '#6366f1'
    ctx.fillText('motion: ' + currentMotion, cx, cy + 4)
    if (currentExpr) ctx.fillText('expr: ' + currentExpr, cx, cy + 22)
    ctx.fillStyle = '#94a3b8'
    ctx.font = '10px monospace'
    var label = modelLabel()
    ctx.fillText(label.length > 42 ? label.slice(0, 40) + '…' : label, cx, cy + 44)
    if (lipLevel > 0.05) {
      ctx.fillStyle = 'rgba(16,185,129,' + Math.min(0.5, lipLevel) + ')'
      ctx.fillRect(cx - rw / 2, cy + rh / 2 - 6, rw * lipLevel, 4)
    }
  }

  function loop() {
    if (destroyed) return
    draw()
    requestAnimationFrame(loop)
  }

  function playMotion(name) {
    currentMotion = String(name || 'idle')
    draw()
  }

  function setExpression(name) {
    currentExpr = String(name || '')
    draw()
  }

  function setVolume(v) {
    lipLevel = Math.min(1, Math.max(0, Number(v) || 0))
    var talk = (manifest.behaviors && manifest.behaviors.talkMotion) || 'idle'
    if (lipLevel > 0.08) playMotion(talk)
    else playMotion(Object.keys(cfg.motions || {})[0] || 'idle')
  }

  function onDown(e) {
    if (!(manifest.behaviors && manifest.behaviors.dragEnabled)) return
    dragging = true
    var rect = canvas.getBoundingClientRect()
    dragOffX = (e.clientX || 0) - rect.left - rect.width / 2 - posX
    dragOffY = (e.clientY || 0) - rect.top - rect.height * 0.55 - posY
  }
  function onMove(e) {
    if (!dragging) return
    var rect = canvas.getBoundingClientRect()
    posX = (e.clientX || 0) - rect.left - rect.width / 2 - dragOffX
    posY = (e.clientY || 0) - rect.top - rect.height * 0.55 - dragOffY
  }
  function onUp() { dragging = false }
  function onTap() {
    var keys = Object.keys(cfg.motions || {})
    if (keys.length > 1) playMotion(keys[1])
  }

  stage.addEventListener('pointerdown', onDown)
  stage.addEventListener('pointermove', onMove)
  stage.addEventListener('pointerup', onUp)
  stage.addEventListener('click', onTap)

  function stop() {
    destroyed = true
    stage.removeEventListener('pointerdown', onDown)
    stage.removeEventListener('pointermove', onMove)
    stage.removeEventListener('pointerup', onUp)
    stage.removeEventListener('click', onTap)
    if (root && root.id === 'soul-pet-desktop-root') root.remove()
    else stage.remove()
  }

  resize()
  window.addEventListener('resize', resize)
  requestAnimationFrame(loop)

  window.__PET_LIVE2D__ = {
    playMotion: playMotion,
    listMotions: function () { return Object.keys(cfg.motions || {}) },
    setExpression: setExpression,
    listExpressions: function () { return Object.keys(cfg.expressions || {}) },
    setVolume: setVolume,
    getModelUrl: function () { return projectBase + modelLabel() },
    stop: stop,
  }
  window.__SOUL_PET_LIP_SYNC__ = { setVolume: setVolume }
  console.log('[Live2D Pet] placeholder ready — drop Cubism model into', cfg.baseUrl)
})()
