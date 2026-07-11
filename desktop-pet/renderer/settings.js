;(async function () {
  const $ = (id) => document.getElementById(id)
  const status = $('status')

  function showStatus(msg, ok) {
    status.textContent = msg
    status.className = 'status ' + (ok ? 'ok' : 'err')
  }

  function readForm() {
    return {
      serverBase: $('serverBase').value.trim(),
      loadMode: $('loadMode').value || 'cloud',
      localPackagePath: $('localPackagePath').value.trim(),
      jsSourceId: $('jsSourceId').value.trim(),
      settingsHotkey: $('settingsHotkey').value.trim(),
      voiceHotkey: $('voiceHotkey').value.trim(),
      cmdVoiceBase: $('cmdVoiceBase').value.trim(),
      agentId: $('agentId').value.trim(),
      apiKey: $('apiKey').value.trim(),
      apiSecret: $('apiSecret').value.trim(),
    }
  }

  function fillForm(cfg) {
    $('serverBase').value = cfg.serverBase || 'http://127.0.0.1:7072/api'
    $('loadMode').value = cfg.loadMode || 'cloud'
    $('localPackagePath').value = cfg.localPackagePath || ''
    $('jsSourceId').value = cfg.jsSourceId || ''
    $('settingsHotkey').value = cfg.settingsHotkey || 'CommandOrControl+Alt+P'
    $('voiceHotkey').value = cfg.voiceHotkey || 'CommandOrControl+Alt+V'
    $('cmdVoiceBase').value = cfg.cmdVoiceBase || 'http://127.0.0.1:7080'
    $('agentId').value = cfg.agentId || ''
    $('apiKey').value = cfg.apiKey || ''
    $('apiSecret').value = cfg.apiSecret || ''
  }

  if (!window.electronPet) {
    showStatus('Electron preload 未就绪', false)
    return
  }

  try {
    fillForm(await window.electronPet.getConfig())
  } catch (e) {
    showStatus('读取配置失败: ' + (e.message || e), false)
  }

  $('btnTest').addEventListener('click', async () => {
    const cfg = readForm()
    if (cfg.loadMode === 'local') {
      showStatus('云端测试仅用于 jsSourceId 模式，本地包请点「校验本地包」', false)
      return
    }
    const id = cfg.jsSourceId
    if (!id) {
      showStatus('请先填写 jsSourceId', false)
      return
    }
    const base = (cfg.serverBase || 'http://127.0.0.1:7072/api').replace(/\/+$/, '')
    const url = base + '/js-templates/embed/' + encodeURIComponent(id) + '/loader.js'
    showStatus('正在测试 ' + url + ' …', true)
    try {
      const res = await fetch(url, { method: 'GET', cache: 'no-store' })
      if (!res.ok) throw new Error('HTTP ' + res.status)
      const text = await res.text()
      if (!text.includes('__PET_MANIFEST__') && !text.includes('__PET_SPRITE__')) {
        throw new Error('响应不像有效的 pet loader')
      }
      showStatus('连接成功，可以启动桌宠', true)
    } catch (e) {
      showStatus('连接失败: ' + (e.message || e) + '\n请确认后端已启动且项目已保存', false)
    }
  })

  $('btnPickFolder').addEventListener('click', async () => {
    if (!window.electronPet?.pickLocalFolder) return
    const picked = await window.electronPet.pickLocalFolder()
    if (picked) {
      $('localPackagePath').value = picked
      $('loadMode').value = 'local'
      showStatus('已选择本地包，请点击「启动桌宠」保存并加载', true)
    }
  })

  $('btnTestLocal').addEventListener('click', async () => {
    const cfg = readForm()
    const root = cfg.localPackagePath
    if (!root) {
      showStatus('请先选择本地 .soulpet 文件夹', false)
      return
    }
    $('loadMode').value = 'local'
    showStatus('正在校验本地包…', true)
    const res = await window.electronPet.validateLocalPackage(root)
    if (res?.ok) {
      showStatus('本地包有效，点击「启动桌宠」加载', true)
    } else {
      showStatus('本地包无效: ' + (res?.error || '未知错误'), false)
    }
  })

  $('btnSave').addEventListener('click', async () => {
    const cfg = readForm()
    if (cfg.loadMode === 'local') {
      if (!cfg.localPackagePath) {
        showStatus('请选择本地 .soulpet 文件夹', false)
        return
      }
    } else if (!cfg.jsSourceId) {
      showStatus('请填写 jsSourceId 或改用本地包模式', false)
      $('jsSourceId').focus()
      return
    }
    try {
      await window.electronPet.saveConfig(cfg)
      showStatus('已保存并重新加载桌宠', true)
    } catch (e) {
      showStatus('保存失败: ' + (e.message || e), false)
    }
  })
})()
