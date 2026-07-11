;(async function bootSoulPetDesktop() {
  const boot = document.getElementById('boot')
  let clickThrough = true
  let dragging = false
  let passthroughBound = false

  function setClickThrough(on) {
    clickThrough = on
    if (window.electronPet && window.electronPet.setIgnoreMouseEvents) {
      window.electronPet.setIgnoreMouseEvents(on, true)
    }
  }

  setClickThrough(true)

  function fail(msg) {
    boot.style.display = 'block'
    boot.textContent = msg + '\n\n请用托盘图标打开「控制面板」'
    boot.classList.add('err')
    setClickThrough(true)
    console.error('[soul-pet-desktop]', msg)
  }

  function hitTestVoiceFab(e) {
    const fab = document.getElementById('soul-pet-voice-fab')
    if (!fab || fab.style.display === 'none') return false
    const rect = fab.getBoundingClientRect()
    if (rect.width <= 0 || rect.height <= 0) return false
    return (
      e.clientX >= rect.left &&
      e.clientX <= rect.right &&
      e.clientY >= rect.top &&
      e.clientY <= rect.bottom
    )
  }

  function hitTestPetCanvas(e) {
    if (hitTestVoiceFab(e)) return true
    const canvas = document.querySelector('.soul-pet-canvas')
    if (!canvas || !canvas.width || !canvas.height) return false
    const rect = canvas.getBoundingClientRect()
    if (rect.width <= 0 || rect.height <= 0) return false
    const scaleX = canvas.width / rect.width
    const scaleY = canvas.height / rect.height
    const cx = Math.floor((e.clientX - rect.left) * scaleX)
    const cy = Math.floor((e.clientY - rect.top) * scaleY)
    if (cx < 0 || cy < 0 || cx >= canvas.width || cy >= canvas.height) return false
    try {
      const ctx = canvas.getContext('2d', { willReadFrequently: true })
      if (!ctx) return false
      return ctx.getImageData(cx, cy, 1, 1).data[3] > 16
    } catch (_) {
      return false
    }
  }

  function syncPointerCapture(e) {
    if (dragging) {
      if (clickThrough) setClickThrough(false)
      return
    }
    setClickThrough(!hitTestPetCanvas(e))
  }

  function bindPassthroughHandlers() {
    if (passthroughBound) return
    passthroughBound = true
    document.addEventListener('mousemove', syncPointerCapture, { passive: true })
    document.addEventListener('mousedown', function (e) {
      if (hitTestPetCanvas(e)) {
        dragging = true
        setClickThrough(false)
      }
    })
    document.addEventListener('mouseup', function () {
      dragging = false
      setClickThrough(true)
    })
    window.addEventListener('blur', function () {
      dragging = false
      setClickThrough(true)
    })
  }

  function petReadyWatch(extraCheck) {
    const watch = window.setInterval(function () {
      const sprite = window.__PET_SPRITE__
      const live2d = window.__PET_LIVE2D__
      if (sprite || live2d || (extraCheck && extraCheck())) {
        window.clearInterval(watch)
        boot.style.display = 'none'
        bindPassthroughHandlers()
        setClickThrough(true)
        console.log('[soul-pet-desktop] pet ready')
      }
    }, 200)
    window.setTimeout(function () {
      window.clearInterval(watch)
      if (!window.__PET_SPRITE__ && !window.__PET_LIVE2D__) {
        fail('桌宠脚本已运行但未就绪')
      }
    }, 30000)
  }

  function applyPetConfig(cfg) {
    window.__PET_EMBED_MODE__ = 'desktop'
    const petCfg = { mode: 'desktop' }
    if (cfg.agentId) petCfg.agentId = cfg.agentId
    if (cfg.apiKey) petCfg.apiKey = cfg.apiKey
    if (cfg.apiSecret) petCfg.apiSecret = cfg.apiSecret
    if (cfg.cmdVoiceBase) petCfg.cmdVoiceBase = String(cfg.cmdVoiceBase).replace(/\/+$/, '')
    window.__AIPetConfig = petCfg
  }

  function injectStyle(css) {
    if (!css) return
    const s = document.createElement('style')
    s.textContent = css
    document.head.appendChild(s)
  }

  async function loadScriptUrl(url) {
    await new Promise(function (resolve, reject) {
      const s = document.createElement('script')
      s.src = url
      s.onload = resolve
      s.onerror = reject
      document.body.appendChild(s)
    })
  }

  async function bootLocal(cfg, serverBase) {
    const pkg = await window.electronPet.readLocalPackage()
    if (!pkg || !pkg.entry) {
      fail('本地包无效或缺少 pet.js\n' + (cfg.localPackagePath || ''))
      return
    }
    window.SERVER_BASE = serverBase
    applyPetConfig(cfg)
    window.__PET_MANIFEST__ = JSON.parse(pkg.manifest || '{}')
    window.__PET_PROJECT_BASE__ = 'soulpet-local:///'

    if (window.__PET_SPRITE__?.stop) try { window.__PET_SPRITE__.stop() } catch (_) { /* ignore */ }
    document.getElementById('soul-pet-desktop-root')?.remove()
    injectStyle(pkg.style)

    boot.textContent = '正在加载本地桌宠…'
    try {
      const base = serverBase.replace(/\/api\/?$/, '')
      await loadScriptUrl(base + '/api/static/js/soul-pet-sdk.js')
      await loadScriptUrl(base + '/api/static/js/pet-voice-bridge.js')
    } catch (e) {
      console.warn('[soul-pet-desktop] SDK load optional', e)
    }

    const s = document.createElement('script')
    s.textContent = pkg.entry
    document.body.appendChild(s)
    petReadyWatch()
  }

  async function bootCloud(cfg, serverBase, jsSourceId) {
    window.SERVER_BASE = serverBase
    applyPetConfig(cfg)

    if (window.__PET_SPRITE__?.stop) try { window.__PET_SPRITE__.stop() } catch (_) { /* ignore */ }
    document.querySelectorAll('script[data-soul-pet-loader]').forEach((n) => n.remove())
    document.getElementById('soul-pet-desktop-root')?.remove()

    const loaderUrl = serverBase + '/js-templates/embed/' + encodeURIComponent(jsSourceId) + '/loader.js'
    boot.textContent = '正在加载桌宠…'

    const script = document.createElement('script')
    script.dataset.soulPetLoader = '1'
    script.src = loaderUrl + '?_=' + Date.now()
    script.onload = function () {
      petReadyWatch()
    }
    script.onerror = function () {
      fail('loader 加载失败:\n' + loaderUrl)
    }
    document.body.appendChild(script)
  }

  if (!window.electronPet) {
    fail('Electron preload 未就绪')
    return
  }

  let cfg
  try {
    cfg = await window.electronPet.getConfig()
  } catch (e) {
    fail('读取配置失败\n' + (e && e.message ? e.message : String(e)))
    return
  }

  const serverBase = String(cfg.serverBase || 'http://127.0.0.1:7072/api').replace(/\/+$/, '')
  const loadMode = cfg.loadMode || 'cloud'
  const localPath = String(cfg.localPackagePath || '').trim()
  const jsSourceId = String(cfg.jsSourceId || '').trim()

  boot.style.display = 'block'
  boot.classList.remove('err')
  setClickThrough(true)

  if (loadMode === 'local') {
    if (!localPath) {
      fail('本地模式：请在控制面板选择 .soulpet 文件夹并保存')
      return
    }
    try {
      await bootLocal(cfg, serverBase)
    } catch (e) {
      fail('本地包加载失败:\n' + (e && e.message ? e.message : String(e)))
    }
    return
  }

  if (!jsSourceId || jsSourceId === 'YOUR_JS_SOURCE_ID') {
    fail('请选择本地包或填写 jsSourceId')
    return
  }

  await bootCloud(cfg, serverBase, jsSourceId)
})()
