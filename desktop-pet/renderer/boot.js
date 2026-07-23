;(async function bootLingEchoDesktop() {
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
    console.error('[lingecho-desktop]', msg)
  }

  function pointInRect(e, rect) {
    if (!rect || rect.width <= 0 || rect.height <= 0) return false
    return (
      e.clientX >= rect.left &&
      e.clientX <= rect.right &&
      e.clientY >= rect.top &&
      e.clientY <= rect.bottom
    )
  }

  function widgetRoot() {
    return document.getElementById('lingecho-embed-root') || document.getElementById('lanlan-pet-root')
  }

  function hitTestInteractive(e) {
    const root = widgetRoot()
    if (!root) return false

    const panel = root.querySelector('.le-panel')
    if (panel && panel.style.display !== 'none' && pointInRect(e, panel.getBoundingClientRect())) {
      return true
    }

    const lanlanUi = root.querySelectorAll('.ll-menu.open, .ll-talk.open, .ll-hint.show')
    for (let i = 0; i < lanlanUi.length; i++) {
      if (pointInRect(e, lanlanUi[i].getBoundingClientRect())) return true
    }

    const petWrap = root.querySelector('.le-pet-wrap, .ll-wrap')
    if (petWrap && !petWrap.classList.contains('hidden') && pointInRect(e, petWrap.getBoundingClientRect())) {
      return true
    }

    const pet = root.querySelector('.le-pet, .ll-pet')
    if (pet && !pet.classList.contains('hidden') && pointInRect(e, pet.getBoundingClientRect())) {
      return true
    }

    return false
  }

  function syncPointerCapture(e) {
    if (dragging) {
      if (clickThrough) setClickThrough(false)
      return
    }
    setClickThrough(!hitTestInteractive(e))
  }

  function bindPassthroughHandlers() {
    if (passthroughBound) return
    passthroughBound = true
    document.addEventListener('mousemove', syncPointerCapture, { passive: true })
    document.addEventListener(
      'mousedown',
      function (e) {
        if (hitTestInteractive(e)) {
          dragging = true
          setClickThrough(false)
        }
      },
      true,
    )
    document.addEventListener('mouseup', function () {
      dragging = false
      setClickThrough(true)
    })
    window.addEventListener('blur', function () {
      dragging = false
      setClickThrough(true)
    })
  }

  function isWidgetReady() {
    const root = widgetRoot()
    if (!root) return false
    const pet = root.querySelector('.le-pet, .ll-pet')
    return !!(window.LingEchoWidget || window.LanlanPet || pet)
  }

  /** Make sure LanlanPet.notifyCodingKey exists even for older remote templates. */
  function ensureCodingNotifyApi() {
    var api = window.LanlanPet || window.LingEchoWidget
    if (!api) return null
    if (typeof api.notifyCodingKey === 'function') {
      return function () {
        api.notifyCodingKey()
      }
    }
    api.notifyCodingKey = function () {
      var inst = api.instance
      if (inst && typeof inst.notifyCodingKey === 'function') {
        inst.notifyCodingKey()
        return
      }
      if (inst && typeof inst.onCodingKey === 'function') {
        inst.onCodingKey()
        return
      }
      window.dispatchEvent(new CustomEvent('lanlan:coding-key'))
    }
    return function () {
      api.notifyCodingKey()
    }
  }

  /** Forward OS-level key activity into lanlan (pet window is click-through / unfocused). */
  function bindGlobalCodingBridge() {
    if (!window.electronPet || typeof window.electronPet.onCodingKey !== 'function') return
    if (window.__lingechoCodingBridgeBound) return
    window.__lingechoCodingBridgeBound = true
    window.electronPet.onCodingKey(function () {
      try {
        var notify = ensureCodingNotifyApi()
        if (notify) {
          notify()
          return
        }
        window.dispatchEvent(new CustomEvent('lanlan:coding-key'))
      } catch (e) {
        console.warn('[lingecho-desktop] coding bridge failed', e)
      }
    })
    console.log('[lingecho-desktop] coding key bridge bound')
  }

  /** Keep desktop tidy usable while a newly edited lanlan.js is waiting to be republished. */
  function ensureTidyApi() {
    var api = window.LanlanPet || window.LingEchoWidget
    if (!api || typeof api.tidyDesktop === 'function') return
    api.tidyDesktop = async function () {
      if (!window.electronPet || typeof window.electronPet.tidyDesktop !== 'function') return false
      var preview =
        typeof window.electronPet.previewDesktop === 'function'
          ? await window.electronPet.previewDesktop()
          : null
      if (preview && preview.ok && !preview.total) {
        if (typeof api.say === 'function') api.say('桌面已经很干净啦～', 1800)
        return true
      }
      var size = Number((window.__LanlanConfig || {}).size) || 160
      var total = preview && preview.total ? preview.total : 1
      var target = {
        left: window.innerWidth - size - 36,
        top: Math.max(54, Math.min(window.innerHeight - size - 40, 54 + Math.min(total, 8) * 16)),
      }
      if (typeof api.say === 'function') api.say('我来收拾一下～', 1800)
      if (typeof api.wanderTo === 'function') api.wanderTo(target, false, true)
      await new Promise((resolve) => window.setTimeout(resolve, 2600))
      if (typeof api.play === 'function') {
        await api.play('shy', { loop: false, force: true, fps: 16 })
      }
      var result = await window.electronPet.tidyDesktop()
      if (typeof api.say === 'function') {
        api.say(result && result.ok ? '都整理好啦～' : '整理时出了点小问题…', 2200)
      }
      if (typeof api.play === 'function') {
        api.play(result && result.ok && result.moved ? 'celebrate' : 'blink')
      }
      return !!(result && result.ok)
    }
  }

  function widgetReadyWatch() {
    const watch = window.setInterval(function () {
      if (isWidgetReady()) {
        window.clearInterval(watch)
        boot.style.display = 'none'
        bindPassthroughHandlers()
        setClickThrough(true)
        ensureCodingNotifyApi()
        ensureTidyApi()
        bindGlobalCodingBridge()
        console.log('[lingecho-desktop] widget ready')
      }
    }, 200)
    window.setTimeout(function () {
      window.clearInterval(watch)
      if (!isWidgetReady()) {
        var hasApi = !!(window.LingEchoWidget || window.LanlanPet)
        var hasLegacyRoot = !!document.getElementById('lanlan-pet-root')
        var msg = '挂件脚本已运行但未就绪（未找到 lingecho-embed-root）'
        if (hasApi && hasLegacyRoot) {
          msg =
            '懒懒挂件已加载，但模版版本较旧（lanlan-pet-root）。\n请在网页控制台重新保存并发布最新 lanlan.js 模版。'
        } else if (hasApi) {
          msg = '挂件 API 已加载，但 DOM 未挂载完成。请检查模版是否设置了 autoMount: true。'
        }
        fail(msg)
      }
    }, 30000)
  }

  function applyLingEchoConfig(cfg, serverBase) {
    window.__LINGECHO_EMBED_MODE__ = 'desktop'
    const next = {
      apiBase: serverBase,
      autoMount: true,
      position: cfg.position || 'right',
    }
    if (cfg.assistantId) next.assistantId = String(cfg.assistantId).trim()
    if (cfg.apiKey) next.apiKey = String(cfg.apiKey).trim()
    if (cfg.transport) next.transport = String(cfg.transport).trim()
    if (cfg.title) next.title = String(cfg.title).trim()
    if (cfg.primaryColor) next.primaryColor = String(cfg.primaryColor).trim()
    if (cfg.size != null && Number(cfg.size) > 0) next.size = Number(cfg.size)
    window.__LingEchoConfig = Object.assign({}, window.__LingEchoConfig || {}, next)
    // 兼容旧版 lanlan.js：只读取 __LanlanConfig
    window.__LanlanConfig = Object.assign({}, window.__LanlanConfig || {}, next, {
      name: cfg.title || next.title || '懒懒',
      persist: true,
      autoWander: cfg.autoWander !== false,
      autoChat: cfg.autoChat !== false,
      watchCoding: cfg.watchCoding !== false,
      voiceHotkey: cfg.voiceHotkey || 'Alt+Shift+V',
      talkHotkey: cfg.talkHotkey || 'Alt+Shift+T',
    })
  }

  function embedScriptUrl(serverBase, jsSourceId) {
    const id = String(jsSourceId || '').trim()
    if (!id || id === 'YOUR_JS_SOURCE_ID' || id === 'default') {
      return serverBase + '/lingecho/embed/v1/embed.js'
    }
    return serverBase + '/lingecho/embed/v1/t/' + encodeURIComponent(id) + '/embed.js'
  }

  async function bootEmbed(cfg, serverBase, jsSourceId) {
    if (window.LingEchoWidget && typeof window.LingEchoWidget.destroy === 'function') {
      try {
        window.LingEchoWidget.destroy()
      } catch (_) {
        /* ignore */
      }
    }
    if (window.LanlanPet && typeof window.LanlanPet.destroy === 'function') {
      try {
        window.LanlanPet.destroy()
      } catch (_) {
        /* ignore */
      }
    }
    document.getElementById('lingecho-embed-root')?.remove()
    document.getElementById('lanlan-pet-root')?.remove()
    document.getElementById('lingecho-embed-css')?.remove()
    document.getElementById('lanlan-pet-css')?.remove()
    document.querySelectorAll('script[data-lingecho-embed]').forEach((n) => n.remove())

    applyLingEchoConfig(cfg, serverBase)

    const scriptUrl = embedScriptUrl(serverBase, jsSourceId)
    boot.textContent = '正在加载挂件…\n' + scriptUrl

    const script = document.createElement('script')
    script.dataset.lingechoEmbed = '1'
    script.src = scriptUrl + (scriptUrl.includes('?') ? '&' : '?') + '_=' + Date.now()
    script.onload = function () {
      widgetReadyWatch()
    }
    script.onerror = function () {
      fail('embed.js 加载失败:\n' + scriptUrl)
    }
    document.body.appendChild(script)
  }

  if (!window.electronPet) {
    fail('Electron preload 未就绪')
    return
  }

  // Bind as early as possible — do not wait for click / widget focus.
  bindGlobalCodingBridge()

  let cfg
  try {
    cfg = await window.electronPet.getConfig()
  } catch (e) {
    fail('读取配置失败\n' + (e && e.message ? e.message : String(e)))
    return
  }

  const serverBase = String(cfg.serverBase || 'http://127.0.0.1:7072/api').replace(/\/+$/, '')
  let jsSourceId = String(cfg.jsSourceId || '').trim()
  if (!jsSourceId && Array.isArray(cfg.pets) && cfg.petId) {
    const pe = cfg.pets.find(function (p) {
      return p && p.id === cfg.petId
    })
    if (pe && pe.jsSourceId) jsSourceId = String(pe.jsSourceId).trim()
  }
  const assistantId = String(cfg.assistantId || '').trim()

  boot.style.display = 'block'
  boot.classList.remove('err')
  setClickThrough(true)

  if (!jsSourceId) {
    fail('缺少 jsSourceId：请在控制面板左侧仓库填写并保存')
    return
  }

  if (!assistantId || assistantId === 'YOUR_ASSISTANT_ID') {
    fail('请在控制面板填写 assistantId（智能体 ID）')
    return
  }

  try {
    await bootEmbed(cfg, serverBase, jsSourceId)
  } catch (e) {
    fail('挂件加载失败:\n' + (e && e.message ? e.message : String(e)))
  }
})()
