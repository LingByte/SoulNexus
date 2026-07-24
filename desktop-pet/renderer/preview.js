;(async function previewEmbed() {
  const boot = document.getElementById('boot')

  function fail(msg) {
    boot.textContent = msg
    boot.classList.add('err')
  }

  function applyLingEchoConfig(cfg, serverBase) {
    window.__LINGECHO_EMBED_MODE__ = 'preview'
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
    window.__LanlanConfig = Object.assign({}, window.__LanlanConfig || {}, next, {
      name: cfg.title || next.title || '懒懒',
      persist: false,
      autoWander: cfg.autoWander !== false,
      autoChat: false,
      watchCoding: false,
    })
  }

  function embedScriptUrl(serverBase, jsSourceId) {
    const id = String(jsSourceId || '').trim()
    if (!id || id === 'YOUR_JS_SOURCE_ID' || id === 'default') {
      return serverBase + '/lingecho/embed/v1/embed.js'
    }
    return serverBase + '/lingecho/embed/v1/t/' + encodeURIComponent(id) + '/embed.js'
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

  const serverBase = String(cfg.serverBase || '').replace(/\/+$/, '')
  const jsSourceId = String(cfg.jsSourceId || '').trim()
  const assistantId = String(cfg.assistantId || '').trim()
  if (!assistantId || assistantId === 'YOUR_ASSISTANT_ID') {
    fail('请先在控制面板填写 assistantId')
    return
  }

  applyLingEchoConfig(cfg, serverBase)
  const scriptUrl = embedScriptUrl(serverBase, jsSourceId)
  boot.textContent = '加载中…\n' + scriptUrl

  const script = document.createElement('script')
  script.src = scriptUrl + (scriptUrl.includes('?') ? '&' : '?') + '_=' + Date.now()
  script.onload = function () {
    const watch = window.setInterval(function () {
      const root =
        document.getElementById('lingecho-embed-root') || document.getElementById('lanlan-pet-root')
      if (root && (window.LingEchoWidget || window.LanlanPet || root.querySelector('.ll-pet'))) {
        window.clearInterval(watch)
        boot.style.display = 'none'
      }
    }, 200)
    window.setTimeout(function () {
      window.clearInterval(watch)
      if (boot.style.display !== 'none') fail('预览超时，请检查 jsSourceId 与网络')
    }, 25000)
  }
  script.onerror = function () {
    fail('embed.js 加载失败:\n' + scriptUrl)
  }
  document.body.appendChild(script)
})()
