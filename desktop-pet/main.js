const {
  app,
  BrowserWindow,
  screen,
  Tray,
  Menu,
  nativeImage,
  ipcMain,
  shell,
  globalShortcut,
  session,
  protocol,
  dialog,
} = require('electron')
const path = require('path')
const fs = require('fs')

let petWindow = null
let settingsWindow = null
let tray = null

const DEFAULTS = {
  serverBase: 'http://127.0.0.1:7072/api',
  jsSourceId: '',
  cmdVoiceBase: 'http://127.0.0.1:7080',
  agentId: '',
  apiKey: '',
  apiSecret: '',
  settingsHotkey: 'CommandOrControl+Alt+P',
  voiceHotkey: 'CommandOrControl+Alt+V',
  loadMode: 'cloud',
  localPackagePath: '',
}

function settingsPath() {
  return path.join(app.getPath('userData'), 'settings.json')
}

function loadConfig() {
  const merged = { ...DEFAULTS }
  try {
    const p = settingsPath()
    if (fs.existsSync(p)) {
      Object.assign(merged, JSON.parse(fs.readFileSync(p, 'utf8')))
    } else {
      const legacy = path.join(__dirname, 'config.local.json')
      const example = path.join(__dirname, 'config.example.json')
      const file = fs.existsSync(legacy) ? legacy : example
      if (fs.existsSync(file)) Object.assign(merged, JSON.parse(fs.readFileSync(file, 'utf8')))
    }
  } catch (e) {
    console.warn('[soul-pet-desktop] loadConfig failed', e)
  }
  if (merged.agentId == null) merged.agentId = ''
  return merged
}

function saveConfig(cfg) {
  const next = { ...DEFAULTS, ...cfg }
  fs.mkdirSync(path.dirname(settingsPath()), { recursive: true })
  fs.writeFileSync(settingsPath(), JSON.stringify(next, null, 2), 'utf8')
  return next
}

function hasValidPetConfig(cfg) {
  const c = cfg || loadConfig()
  if (c.loadMode === 'local') {
    const local = String(c.localPackagePath || '').trim()
    if (!local || !fs.existsSync(local)) return false
    return (
      fs.existsSync(path.join(local, 'manifest.json')) &&
      fs.existsSync(path.join(local, 'pet.js'))
    )
  }
  const id = String(c.jsSourceId || '').trim()
  return id.length > 0 && id !== 'YOUR_JS_SOURCE_ID'
}

function readLocalPackageMeta(root) {
  const readText = (rel) => {
    const p = path.join(root, rel)
    return fs.existsSync(p) ? fs.readFileSync(p, 'utf8') : ''
  }
  if (!fs.existsSync(path.join(root, 'manifest.json'))) {
    throw new Error('missing manifest.json')
  }
  if (!fs.existsSync(path.join(root, 'pet.js'))) {
    throw new Error('missing pet.js')
  }
  return {
    manifest: readText('manifest.json'),
    entry: readText('pet.js'),
    style: readText('style.css'),
    kind: readText('soulpet.yaml'),
  }
}

function applyPetClickThrough() {
  if (!petWindow || petWindow.isDestroyed()) return
  petWindow.setIgnoreMouseEvents(true, { forward: true })
}

function createPetWindow() {
  if (petWindow && !petWindow.isDestroyed()) return petWindow

  const display = screen.getPrimaryDisplay()
  const { width, height } = display.workAreaSize
  const { x, y } = display.workArea

  petWindow = new BrowserWindow({
    x,
    y,
    width,
    height,
    transparent: true,
    frame: false,
    hasShadow: false,
    alwaysOnTop: true,
    resizable: false,
    movable: false,
    fullscreenable: false,
    show: false,
    focusable: false,
    backgroundColor: '#00000000',
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: false,
      backgroundThrottling: false,
    },
  })

  // Normal floating level — avoid blocking system UI above screen-saver tier.
  petWindow.setAlwaysOnTop(true, 'floating')
  petWindow.setVisibleOnAllWorkspaces(true, { visibleOnFullScreen: true })
  if (process.platform === 'darwin') petWindow.setHiddenInMissionControl(true)

  petWindow.webContents.on('did-finish-load', applyPetClickThrough)

  petWindow.loadFile(path.join(__dirname, 'renderer', 'index.html'))
  petWindow.once('ready-to-show', () => {
    if (petWindow.isDestroyed()) return
    applyPetClickThrough()
    petWindow.showInactive()
  })
  petWindow.webContents.setWindowOpenHandler(({ url }) => {
    shell.openExternal(url)
    return { action: 'deny' }
  })
  petWindow.on('closed', () => {
    petWindow = null
  })

  return petWindow
}

function destroyPetWindow() {
  if (petWindow && !petWindow.isDestroyed()) {
    petWindow.close()
  }
  petWindow = null
}

function toggleVoiceCall() {
  if (!petWindow || petWindow.isDestroyed()) {
    createSettingsWindow()
    return
  }
  petWindow.webContents
    .executeJavaScript(
      `(function () {
        if (window.__SOUL_PET_VOICE__ && typeof window.__SOUL_PET_VOICE__.toggleCall === 'function') {
          window.__SOUL_PET_VOICE__.toggleCall()
          return 'ok'
        }
        return 'missing'
      })()`,
      true,
    )
    .then((result) => {
      if (result === 'missing') createSettingsWindow()
    })
    .catch((err) => {
      console.warn('[soul-pet-desktop] toggleVoiceCall failed', err)
    })
}

function registerGlobalShortcuts() {
  globalShortcut.unregisterAll()
  const cfg = loadConfig()

  const panelKey = String(cfg.settingsHotkey || DEFAULTS.settingsHotkey).trim()
  if (panelKey) {
    try {
      if (!globalShortcut.register(panelKey, createSettingsWindow)) {
        console.warn('[soul-pet-desktop] panel hotkey not registered:', panelKey)
      }
    } catch (e) {
      console.warn('[soul-pet-desktop] panel hotkey invalid:', panelKey, e.message)
    }
  }

  const voiceKey = String(cfg.voiceHotkey || DEFAULTS.voiceHotkey).trim()
  if (voiceKey) {
    try {
      if (!globalShortcut.register(voiceKey, toggleVoiceCall)) {
        console.warn('[soul-pet-desktop] voice hotkey not registered:', voiceKey)
      }
    } catch (e) {
      console.warn('[soul-pet-desktop] voice hotkey invalid:', voiceKey, e.message)
    }
  }
}

function setupMediaPermissions() {
  session.defaultSession.setPermissionRequestHandler((_wc, permission, callback, details) => {
    const mediaTypes = details && details.mediaTypes
    if (
      permission === 'media' ||
      permission === 'microphone' ||
      (Array.isArray(mediaTypes) && mediaTypes.includes('audio'))
    ) {
      callback(true)
      return
    }
    callback(false)
  })
}

function createSettingsWindow() {
  if (settingsWindow && !settingsWindow.isDestroyed()) {
    settingsWindow.show()
    settingsWindow.focus()
    return
  }

  settingsWindow = new BrowserWindow({
    width: 440,
    height: 560,
    minWidth: 380,
    minHeight: 480,
    title: 'Soul Pet 控制面板',
    show: false,
    autoHideMenuBar: true,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: false,
    },
  })

  settingsWindow.loadFile(path.join(__dirname, 'renderer', 'settings.html'))
  settingsWindow.once('ready-to-show', () => {
    if (!settingsWindow.isDestroyed()) settingsWindow.show()
  })
  settingsWindow.on('closed', () => {
    settingsWindow = null
  })
}

function reloadPet() {
  const cfg = loadConfig()
  if (!hasValidPetConfig(cfg)) {
    destroyPetWindow()
    createSettingsWindow()
    return
  }
  if (petWindow && !petWindow.isDestroyed()) {
    petWindow.webContents.reloadIgnoringCache()
    return
  }
  createPetWindow()
}

function hotkeyLabel(key) {
  const k = String(key || '').trim()
  if (!k) return ''
  if (process.platform === 'darwin') {
    return k
      .replace(/CommandOrControl/g, '⌘')
      .replace(/Command/g, '⌘')
      .replace(/Control/g, '⌃')
      .replace(/Alt/g, '⌥')
      .replace(/Shift/g, '⇧')
      .replace(/\+/g, '')
  }
  return k.replace(/CommandOrControl/g, 'Ctrl')
}

function rebuildTray() {
  if (!tray) return
  const cfg = loadConfig()
  const panelHint = hotkeyLabel(cfg.settingsHotkey || DEFAULTS.settingsHotkey)
  const voiceHint = hotkeyLabel(cfg.voiceHotkey || DEFAULTS.voiceHotkey)
  const menu = Menu.buildFromTemplate([
    {
      label: panelHint ? `控制面板 (${panelHint})` : '控制面板',
      click: createSettingsWindow,
    },
    {
      label: voiceHint ? `语音对话 (${voiceHint})` : '语音对话',
      click: toggleVoiceCall,
    },
    { label: '重新加载桌宠', click: reloadPet },
    {
      label: '打开 SoulNexus',
      click: () => {
        const base = (loadConfig().serverBase || DEFAULTS.serverBase).replace(/\/api\/?$/, '')
        shell.openExternal(base)
      },
    },
    { type: 'separator' },
    { label: '退出', click: () => app.quit() },
  ])
  tray.setContextMenu(menu)
}

function createTray() {
  const icon = nativeImage.createEmpty()
  tray = new Tray(icon)
  tray.setToolTip('Soul Pet')
  tray.on('click', createSettingsWindow)
  rebuildTray()
}

ipcMain.handle('pet:get-config', () => loadConfig())
ipcMain.handle('pet:pick-local-folder', async () => {
  const result = await dialog.showOpenDialog({
    properties: ['openDirectory'],
    title: '选择 .soulpet 项目文件夹',
  })
  if (result.canceled || !result.filePaths?.[0]) return null
  return result.filePaths[0]
})
ipcMain.handle('pet:validate-local-package', (_e, rootPath) => {
  const root = String(rootPath || loadConfig().localPackagePath || '').trim()
  if (!root || !fs.existsSync(root)) {
    return { ok: false, error: '目录不存在' }
  }
  try {
    const meta = readLocalPackageMeta(root)
    JSON.parse(meta.manifest || '{}')
    return { ok: true, root }
  } catch (e) {
    return { ok: false, error: e.message || String(e) }
  }
})
ipcMain.handle('pet:read-local-package', () => {
  const cfg = loadConfig()
  const root = String(cfg.localPackagePath || '').trim()
  if (!root || !fs.existsSync(root)) return null
  try {
    const meta = readLocalPackageMeta(root)
    return { root, ...meta }
  } catch (e) {
    console.error('[soul-pet-desktop] readLocalPackageMeta failed', e)
    return null
  }
})
ipcMain.handle('pet:save-config', (_e, cfg) => {
  const saved = saveConfig(cfg || {})
  registerGlobalShortcuts()
  rebuildTray()
  reloadPet()
  return saved
})
ipcMain.handle('pet:toggle-voice', () => {
  toggleVoiceCall()
})
ipcMain.handle('pet:open-settings', () => {
  createSettingsWindow()
})
ipcMain.handle('pet:set-ignore-mouse', (_e, ignore, forward = true) => {
  if (!petWindow || petWindow.isDestroyed()) return false
  if (ignore) {
    petWindow.setIgnoreMouseEvents(true, forward ? { forward: true } : undefined)
  } else {
    petWindow.setIgnoreMouseEvents(false)
  }
  return true
})

app.whenReady().then(() => {
  protocol.registerFileProtocol('soulpet-local', (request, callback) => {
    const cfg = loadConfig()
    const root = String(cfg.localPackagePath || '').trim()
    if (!root) {
      callback({ error: -2 })
      return
    }
    let rel = decodeURIComponent(request.url.replace(/^soulpet-local:\/\/?/i, ''))
    rel = rel.replace(/^\/+/, '')
    const rootResolved = path.resolve(root)
    const filePath = path.resolve(rootResolved, rel)
    const rootPrefix = rootResolved.endsWith(path.sep) ? rootResolved : rootResolved + path.sep
    if (filePath !== rootResolved && !filePath.startsWith(rootPrefix)) {
      callback({ error: -6 })
      return
    }
    if (!fs.existsSync(filePath) || fs.statSync(filePath).isDirectory()) {
      callback({ error: -6 })
      return
    }
    callback({ path: filePath })
  })

  setupMediaPermissions()
  createTray()
  registerGlobalShortcuts()

  const cfg = loadConfig()
  if (hasValidPetConfig(cfg)) {
    createPetWindow()
  } else {
    createSettingsWindow()
  }
})

app.on('will-quit', () => {
  globalShortcut.unregisterAll()
})

app.on('activate', () => {
  if (BrowserWindow.getAllWindows().length === 0) {
    createTray()
    if (hasValidPetConfig(loadConfig())) {
      createPetWindow()
    } else {
      createSettingsWindow()
    }
  }
})

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') app.quit()
})
