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
  utilityProcess,
  systemPreferences,
  dialog,
} = require('electron')
const path = require('path')
const fs = require('fs')

const gotTheLock = app.requestSingleInstanceLock()
if (!gotTheLock) {
  app.quit()
}

/** @type {Map<string, import('electron').BrowserWindow>} */
const petWindows = new Map()
/** @type {Map<number, string>} webContents.id -> petId */
const webContentsPetId = new Map()
/** @type {Map<number, string>} preview webContents.id -> petId */
const previewWebContentsPetId = new Map()
let settingsWindow = null
let previewWindow = null
let tray = null
let codingHook = null
let codingHookStarting = false
let codingHookPrompted = false
let codingHookRetryTimer = null
let lastCodingKeyAt = 0
const codingHookApi = require('./coding-hook')
const desktopTidy = require('./desktop-tidy')

const SETTINGS_DEV_URL = process.env.LINGECHO_SETTINGS_URL || 'http://127.0.0.1:5174'
const isDev = !app.isPackaged && process.env.LINGECHO_SETTINGS_DEV === '1'

const APP_NAME = 'SoulMy'
const APP_ID = 'com.soulnexus.desktop-pet'

const DEFAULTS = {
  serverBase: 'https://soulmy.top/api',
  jsSourceId: 'js_75cad2ab4f9f142a',
  assistantId: '8859281265343332864',
  apiKey: 'soulnexus_user_PI2mRsBxioqpTkAS3K3yG3Z_YzY2smCsidEWJRuamMI',
  transport: 'websocket',
  title: '懒懒',
  position: 'right',
  primaryColor: '#18181B',
  size: 160,
  autoWander: true,
  autoChat: true,
  watchCoding: true,
  settingsHotkey: 'CommandOrControl+Alt+P',
  panelHotkey: 'CommandOrControl+Alt+V',
  voiceHotkey: 'Alt+Shift+V',
  talkHotkey: 'Alt+Shift+T',
  openAtLogin: false,
  primaryPetId: '',
  pets: [],
}

function newPetId() {
  return 'pet_' + Math.random().toString(36).slice(2, 10)
}

function defaultPetEntry(overrides) {
  return {
    id: newPetId(),
    name: DEFAULTS.title,
    jsSourceId: DEFAULTS.jsSourceId,
    enabled: true,
    title: DEFAULTS.title,
    position: DEFAULTS.position,
    size: DEFAULTS.size,
    autoWander: DEFAULTS.autoWander,
    autoChat: DEFAULTS.autoChat,
    watchCoding: DEFAULTS.watchCoding,
    ...overrides,
  }
}

function normalizePetsArray(cfg) {
  if (Array.isArray(cfg.pets) && cfg.pets.length > 0) {
    return cfg.pets.map((p) => ({
      ...defaultPetEntry({}),
      ...p,
      id: String(p.id || newPetId()),
      size: Math.max(96, Math.min(256, Number(p.size) || DEFAULTS.size)),
      enabled: p.enabled !== false,
    }))
  }
  return [
    defaultPetEntry({
      name: cfg.title || DEFAULTS.title,
      title: cfg.title || DEFAULTS.title,
      jsSourceId: String(cfg.jsSourceId || DEFAULTS.jsSourceId).trim() || DEFAULTS.jsSourceId,
      position: cfg.position || DEFAULTS.position,
      size: Math.max(96, Math.min(256, Number(cfg.size) || DEFAULTS.size)),
      autoWander: cfg.autoWander !== false,
      autoChat: cfg.autoChat !== false,
      watchCoding: cfg.watchCoding !== false,
    }),
  ]
}

function buildRuntimeConfig(doc, petEntry, petIdHint) {
  let p = petEntry
  if (!p && petIdHint) {
    p = doc.pets.find((x) => x.id === petIdHint)
  }
  if (!p) p = normalizePetsArray(doc)[0]
  if (!p) {
    p = defaultPetEntry({ jsSourceId: DEFAULTS.jsSourceId })
  }
  return {
    serverBase: doc.serverBase,
    jsSourceId: String(p.jsSourceId || '').trim() || DEFAULTS.jsSourceId,
    assistantId: doc.assistantId,
    apiKey: doc.apiKey,
    transport: doc.transport,
    title: p.title || p.name || DEFAULTS.title,
    position: p.position || DEFAULTS.position,
    primaryColor: doc.primaryColor,
    size: Math.max(96, Math.min(256, Number(p.size) || DEFAULTS.size)),
    autoWander: p.autoWander !== false,
    autoChat: p.autoChat !== false,
    watchCoding: p.watchCoding !== false,
    voiceHotkey: doc.voiceHotkey,
    talkHotkey: doc.talkHotkey,
    openAtLogin: doc.openAtLogin,
    petId: p.id,
    petName: p.name,
  }
}

function petIdFromWebContents(wc) {
  if (!wc || wc.isDestroyed()) return null
  let id = webContentsPetId.get(wc.id) || previewWebContentsPetId.get(wc.id)
  if (id) return id
  for (const [petId, win] of petWindows) {
    if (win && !win.isDestroyed() && win.webContents.id === wc.id) return petId
  }
  try {
    const url = wc.getURL()
    if (url && url.includes('petId=')) {
      const u = new URL(url)
      const q = u.searchParams.get('petId')
      if (q) return q
    }
  } catch (_) {
    /* ignore */
  }
  return null
}

function getPetIdForWebContents(wc) {
  return petIdFromWebContents(wc)
}

function getPrimaryPetWindow() {
  const cfg = loadConfig()
  const primaryId = String(cfg.primaryPetId || '').trim()
  if (primaryId && petWindows.has(primaryId)) {
    const win = petWindows.get(primaryId)
    if (win && !win.isDestroyed()) return win
  }
  for (const win of petWindows.values()) {
    if (win && !win.isDestroyed()) return win
  }
  return null
}

function assetPath(...parts) {
  return path.join(__dirname, 'assets', ...parts)
}

function loadAppIcon() {
  const candidates =
    process.platform === 'darwin'
      ? [
          path.join(__dirname, 'build', 'icon.icns'),
          path.join(__dirname, 'build', 'icon.png'),
          assetPath('icon.png'),
        ]
      : process.platform === 'win32'
        ? [
            path.join(__dirname, 'build', 'icon.ico'),
            path.join(__dirname, 'build', 'icon.png'),
            assetPath('icon.png'),
          ]
        : [
            path.join(__dirname, 'build', 'icon.png'),
            assetPath('icon.png'),
          ]
  for (const p of candidates) {
    try {
      if (!fs.existsSync(p)) continue
      const img = nativeImage.createFromPath(p)
      if (!img.isEmpty()) return img
    } catch (e) {
      console.warn('[lingecho-desktop] loadAppIcon failed', p, e)
    }
  }
  return nativeImage.createEmpty()
}

function loadTrayIcon() {
  // Prefer small tray assets; fall back to the app icon scaled down.
  const candidates =
    process.platform === 'darwin'
      ? [assetPath('trayTemplate.png'), assetPath('tray.png'), assetPath('icon.png')]
      : [assetPath('tray.png'), assetPath('icon.png')]
  for (const p of candidates) {
    try {
      if (!fs.existsSync(p)) continue
      let img = nativeImage.createFromPath(p)
      if (img.isEmpty()) continue
      if (process.platform === 'darwin' && /Template\.png$/i.test(p)) {
        img.setTemplateImage(true)
      }
      // Keep tray icons small so they don't look blurry / huge.
      const size = process.platform === 'darwin' ? 22 : 16
      if (img.getSize().width > size * 2) {
        img = img.resize({ width: size, height: size, quality: 'best' })
      }
      return img
    } catch (e) {
      console.warn('[lingecho-desktop] loadTrayIcon failed', p, e)
    }
  }
  return nativeImage.createEmpty()
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
    console.warn('[lingecho-desktop] loadConfig failed', e)
  }
  if (!merged.assistantId && merged.agentId) merged.assistantId = merged.agentId
  if (merged.assistantId == null) merged.assistantId = ''
  if (merged.jsSourceId === 'YOUR_JS_SOURCE_ID') merged.jsSourceId = ''
  // Migrate legacy host + fill empty / placeholder credentials from baked-in defaults
  const fromLegacyHost = String(merged.serverBase || '').includes('400.lingecho.com')
  if (!String(merged.serverBase || '').trim() || fromLegacyHost) {
    merged.serverBase = DEFAULTS.serverBase
  }
  if (fromLegacyHost || !String(merged.jsSourceId || '').trim()) {
    merged.jsSourceId = DEFAULTS.jsSourceId
  }
  if (
    fromLegacyHost ||
    !String(merged.assistantId || '').trim() ||
    merged.assistantId === 'YOUR_ASSISTANT_ID'
  ) {
    merged.assistantId = DEFAULTS.assistantId
  }
  if (fromLegacyHost || !String(merged.apiKey || '').trim() || merged.apiKey === 'YOUR_API_KEY') {
    merged.apiKey = DEFAULTS.apiKey
  }
  if (!merged.primaryColor || /^#165DFF$/i.test(String(merged.primaryColor))) {
    merged.primaryColor = DEFAULTS.primaryColor
  }
  if (!merged.title) merged.title = DEFAULTS.title
  delete merged.senseWindows
  delete merged.followWindow
  merged.size = Math.max(96, Math.min(256, Number(merged.size) || DEFAULTS.size))
  merged.autoWander = merged.autoWander !== false
  merged.autoChat = merged.autoChat !== false
  merged.watchCoding = merged.watchCoding !== false
  merged.voiceHotkey = String(merged.voiceHotkey || DEFAULTS.voiceHotkey).trim()
  merged.talkHotkey = String(merged.talkHotkey || DEFAULTS.talkHotkey).trim()
  merged.openAtLogin = Boolean(merged.openAtLogin)
  merged.pets = normalizePetsArray(merged)
  if (!merged.primaryPetId || !merged.pets.some((p) => p.id === merged.primaryPetId)) {
    merged.primaryPetId = (merged.pets.find((p) => p.enabled) || merged.pets[0]).id
  }
  return merged
}

function applyLoginItemSettings(openAtLogin) {
  try {
    app.setLoginItemSettings({
      openAtLogin: Boolean(openAtLogin),
      openAsHidden: true,
      name: APP_NAME,
    })
  } catch (e) {
    console.warn('[lingecho-desktop] setLoginItemSettings failed', e)
  }
}

function saveConfig(cfg) {
  const next = { ...DEFAULTS, ...cfg }
  delete next.agentId
  delete next.apiSecret
  delete next.cmdVoiceBase
  delete next.loadMode
  delete next.localPackagePath
  delete next.senseWindows
  delete next.followWindow
  delete next.jsSourceId
  delete next.title
  delete next.position
  delete next.size
  delete next.autoWander
  delete next.autoChat
  delete next.watchCoding
  next.pets = normalizePetsArray(next)
  if (!next.primaryPetId || !next.pets.some((p) => p.id === next.primaryPetId)) {
    next.primaryPetId = (next.pets.find((p) => p.enabled) || next.pets[0]).id
  }
  next.primaryColor = next.primaryColor || DEFAULTS.primaryColor
  next.voiceHotkey = String(next.voiceHotkey || DEFAULTS.voiceHotkey).trim()
  next.talkHotkey = String(next.talkHotkey || DEFAULTS.talkHotkey).trim()
  next.openAtLogin = Boolean(next.openAtLogin)
  fs.mkdirSync(path.dirname(settingsPath()), { recursive: true })
  fs.writeFileSync(settingsPath(), JSON.stringify(next, null, 2), 'utf8')
  applyLoginItemSettings(next.openAtLogin)
  return next
}

function hasValidPetConfig(cfg) {
  const c = cfg || loadConfig()
  const assistantId = String(c.assistantId || '').trim()
  if (!assistantId || assistantId === 'YOUR_ASSISTANT_ID') return false
  const base = String(c.serverBase || '').trim()
  return base.length > 0
}

function applyPetClickThrough(win) {
  if (!win || win.isDestroyed()) return
  win.setIgnoreMouseEvents(true, { forward: true })
}

function createPetWindowForEntry(petEntry) {
  const petId = petEntry.id
  const existing = petWindows.get(petId)
  if (existing && !existing.isDestroyed()) return existing

  const display = screen.getPrimaryDisplay()
  const { width, height } = display.workAreaSize
  const { x, y } = display.workArea

  const win = new BrowserWindow({
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
    skipTaskbar: true,
    show: false,
    focusable: true,
    backgroundColor: '#00000000',
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: false,
      backgroundThrottling: false,
    },
  })

  win.setAlwaysOnTop(true, 'floating')
  win.setVisibleOnAllWorkspaces(true, { visibleOnFullScreen: true })
  if (process.platform === 'darwin') win.setHiddenInMissionControl(true)

  const wcId = win.webContents.id
  webContentsPetId.set(wcId, petId)
  win.webContents.on('did-finish-load', () => applyPetClickThrough(win))

  win.loadFile(path.join(__dirname, 'renderer', 'index.html'), {
    query: { petId },
  })
  win.once('ready-to-show', () => {
    if (win.isDestroyed()) return
    applyPetClickThrough(win)
    win.showInactive()
  })
  win.webContents.setWindowOpenHandler(({ url }) => {
    shell.openExternal(url)
    return { action: 'deny' }
  })
  win.on('closed', () => {
    webContentsPetId.delete(wcId)
    petWindows.delete(petId)
  })

  petWindows.set(petId, win)
  return win
}

function destroyAllPetWindows() {
  for (const win of petWindows.values()) {
    if (win && !win.isDestroyed()) win.close()
  }
  petWindows.clear()
}

function syncPetWindows(opts) {
  const reloadExisting = opts && opts.reloadExisting
  const cfg = loadConfig()
  if (!hasValidPetConfig(cfg)) {
    destroyAllPetWindows()
    return
  }
  const enabled = cfg.pets.filter((p) => p.enabled !== false)
  for (const id of [...petWindows.keys()]) {
    if (!enabled.some((p) => p.id === id)) {
      const win = petWindows.get(id)
      if (win && !win.isDestroyed()) win.close()
      petWindows.delete(id)
    }
  }
  enabled.forEach((p) => {
    const hadWindow =
      petWindows.has(p.id) &&
      petWindows.get(p.id) &&
      !petWindows.get(p.id).isDestroyed()
    const win = createPetWindowForEntry(p)
    if (reloadExisting && hadWindow && win && !win.isDestroyed()) {
      win.webContents.reloadIgnoringCache()
    }
  })
}

function ensurePetAlive() {
  if (hasValidPetConfig()) syncPetWindows()
}

function withPrimaryPetRenderer(callback) {
  let win = getPrimaryPetWindow()
  if (!win) {
    if (hasValidPetConfig()) {
      syncPetWindows()
      win = getPrimaryPetWindow()
      if (win) {
        win.webContents.once('did-finish-load', () => callback(win))
        return
      }
    }
    createSettingsWindow()
    return
  }
  callback(win)
}

function toggleChatPanel() {
  withPrimaryPetRenderer((win) => {
    win.webContents
      .executeJavaScript(
        `(function () {
          var api = window.LanlanPet || window.LingEchoWidget
          if (!api) return 'missing'
          var inst = api.instance
          if (inst && inst.talkOpen && typeof inst.closeTalk === 'function') {
            inst.closeTalk()
            return 'closed'
          }
          if (typeof api.openTalk === 'function') {
            api.openTalk()
            return 'opened'
          }
          if (typeof api.toggle === 'function') {
            api.toggle()
            return 'opened'
          }
          return 'missing'
        })()`,
        true,
      )
      .then((result) => {
        if (result === 'missing') {
          createSettingsWindow()
          return
        }
        if (win.isDestroyed()) return
        // Keep click-through; only take keyboard focus so the talk input can receive typing.
        if (result === 'opened') {
          win.showInactive()
          win.focus()
          win.webContents
            .executeJavaScript(
              `(function () {
                var root = document.getElementById('lingecho-embed-root') || document.getElementById('lanlan-pet-root')
                var input = root && root.querySelector('.ll-talk input, .ll-talk textarea, input')
                if (input && typeof input.focus === 'function') input.focus()
                return true
              })()`,
              true,
            )
            .catch(() => {})
        }
      })
      .catch((err) => {
        console.warn('[lingecho-desktop] toggleChatPanel failed', err)
      })
  })
}

function toggleVoiceGlobally() {
  withPrimaryPetRenderer((win) => {
    win.webContents
      .executeJavaScript(
        `(function () {
          var api = window.LanlanPet || window.LingEchoWidget
          if (!api || typeof api.toggleVoice !== 'function') return 'missing'
          api.toggleVoice()
          return 'ok'
        })()`,
        true,
      )
      .then((result) => {
        if (result === 'missing') createSettingsWindow()
      })
      .catch((err) => {
        console.warn('[lingecho-desktop] toggleVoiceGlobally failed', err)
      })
  })
}

function registerGlobalShortcuts() {
  globalShortcut.unregisterAll()
  const cfg = loadConfig()
  const registered = new Set()

  function register(name, key, handler) {
    const accelerator = String(key || '').trim()
    if (!accelerator) return
    const signature = accelerator.toLowerCase()
    if (registered.has(signature)) {
      console.warn(`[lingecho-desktop] ${name} hotkey duplicates another shortcut:`, accelerator)
      return
    }
    try {
      if (!globalShortcut.register(accelerator, handler)) {
        console.warn(`[lingecho-desktop] ${name} hotkey not registered:`, accelerator)
        return
      }
      registered.add(signature)
    } catch (e) {
      console.warn(`[lingecho-desktop] ${name} hotkey invalid:`, accelerator, e.message)
    }
  }

  register('settings', cfg.settingsHotkey || DEFAULTS.settingsHotkey, createSettingsWindow)
  register('panel', cfg.panelHotkey || DEFAULTS.panelHotkey, toggleChatPanel)
  register('voice', cfg.voiceHotkey || DEFAULTS.voiceHotkey, toggleVoiceGlobally)
  register('talk', cfg.talkHotkey || DEFAULTS.talkHotkey, toggleChatPanel)
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

function loadSettingsContent(win) {
  if (isDev) {
    win.loadURL(SETTINGS_DEV_URL)
    return
  }
  const built = path.join(__dirname, 'dist-settings', 'index.html')
  if (fs.existsSync(built)) {
    win.loadFile(built)
    return
  }
  // Fallback during migration if build missing
  const legacy = path.join(__dirname, 'renderer', 'settings.html')
  if (fs.existsSync(legacy)) {
    win.loadFile(legacy)
    return
  }
  win.loadURL(
    'data:text/html;charset=utf-8,' +
      encodeURIComponent(
        '<p style="font:14px system-ui;padding:24px">设置页未构建。请运行 <code>npm run build:settings</code></p>',
      ),
  )
}

function createSettingsWindow() {
  if (settingsWindow && !settingsWindow.isDestroyed()) {
    settingsWindow.show()
    settingsWindow.focus()
    return
  }

  settingsWindow = new BrowserWindow({
    width: 920,
    height: 760,
    minWidth: 720,
    minHeight: 560,
    title: APP_NAME,
    icon: loadAppIcon(),
    show: false,
    autoHideMenuBar: true,
    backgroundColor: '#e3e3e6',
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: false,
    },
  })

  loadSettingsContent(settingsWindow)
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
    destroyAllPetWindows()
    createSettingsWindow()
    return
  }
  syncCodingHook()
  syncPetWindows({ reloadExisting: true })
}

function openEmbedPreview(petId) {
  const cfg = loadConfig()
  const entry = cfg.pets.find((p) => p.id === petId)
  if (!entry) return { ok: false, error: '未找到该桌宠' }
  if (previewWindow && !previewWindow.isDestroyed()) {
    previewWindow.close()
    previewWindow = null
  }
  previewWindow = new BrowserWindow({
    width: 420,
    height: 520,
    title: `预览 · ${entry.name || entry.title}`,
    icon: loadAppIcon(),
    show: false,
    autoHideMenuBar: true,
    backgroundColor: '#e8e8eb',
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: false,
    },
  })
  const previewWcId = previewWindow.webContents.id
  previewWebContentsPetId.set(previewWcId, petId)
  previewWindow.on('closed', () => {
    previewWebContentsPetId.delete(previewWcId)
    previewWindow = null
  })
  previewWindow.loadFile(path.join(__dirname, 'renderer', 'preview.html'))
  previewWindow.once('ready-to-show', () => {
    if (!previewWindow.isDestroyed()) previewWindow.show()
  })
  return { ok: true }
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

function setOpenAtLogin(enabled) {
  const cfg = loadConfig()
  cfg.openAtLogin = Boolean(enabled)
  saveConfig(cfg)
  rebuildTray()
}

function rebuildTray() {
  if (!tray) return
  const cfg = loadConfig()
  const settingsHint = hotkeyLabel(cfg.settingsHotkey || DEFAULTS.settingsHotkey)
  const talkHint = hotkeyLabel(cfg.talkHotkey || DEFAULTS.talkHotkey)
  const voiceHint = hotkeyLabel(cfg.voiceHotkey || DEFAULTS.voiceHotkey)
  const menu = Menu.buildFromTemplate([
    {
      label: settingsHint ? `控制面板 (${settingsHint})` : '控制面板',
      click: createSettingsWindow,
    },
    {
      label: talkHint ? `打开/关闭文字对话 (${talkHint})` : '打开/关闭文字对话',
      click: toggleChatPanel,
    },
    {
      label: voiceHint ? `开始/结束语音 (${voiceHint})` : '开始/结束语音',
      click: toggleVoiceGlobally,
    },
    { label: '重新加载挂件', click: reloadPet },
    {
      label: '打开 SoulMy',
      click: () => {
        const base = (loadConfig().serverBase || DEFAULTS.serverBase).replace(/\/api\/?$/, '')
        shell.openExternal(base || 'http://127.0.0.1:8080')
      },
    },
    { type: 'separator' },
    {
      label: '让懒懒整理桌面',
      click: () => void startDesktopTidy(),
    },
    { type: 'separator' },
    {
      label: '开机自启',
      type: 'checkbox',
      checked: Boolean(cfg.openAtLogin),
      click: (item) => setOpenAtLogin(item.checked),
    },
    {
      label: codingHook ? '敲代码监听：已开启' : '敲代码监听：未开启（点击重试）',
      click: () => {
        codingHookPrompted = false
        stopCodingHook()
        void startCodingHook().then((ok) => {
          rebuildTray()
          if (!ok && process.platform === 'darwin' && !hasMacAccessibility()) {
            dialog
              .showMessageBox({
                type: 'warning',
                buttons: ['打开系统设置', '好'],
                defaultId: 0,
                message: '尚未获得「辅助功能」权限',
                detail:
                  '没有该权限就无法在其它 App 打字时驱动桌宠。授权后请完全退出桌宠再打开。',
              })
              .then((r) => {
                if (r.response === 0) {
                  shell.openExternal(
                    'x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility',
                  )
                }
              })
          }
        })
      },
    },
    { type: 'separator' },
    { label: '退出', click: () => app.quit() },
  ])
  tray.setContextMenu(menu)
}

function emitCodingKeyToPet() {
  const now = Date.now()
  // Dedupe bursts from OS key repeat / dual listeners
  if (now - lastCodingKeyAt < 30) return
  lastCodingKeyAt = now
  if (!getPrimaryPetWindow()) return
  const cfg = loadConfig()
  const primary = cfg.pets.find((p) => p.id === cfg.primaryPetId)
  if (primary && primary.watchCoding === false) return
  try {
    const win = getPrimaryPetWindow()
    if (win && !win.isDestroyed()) win.webContents.send('pet:coding-key', { t: now })
  } catch (e) {
    console.warn('[lingecho-desktop] emit coding-key failed', e)
  }
}

function stopCodingHook() {
  if (codingHookRetryTimer) {
    clearInterval(codingHookRetryTimer)
    codingHookRetryTimer = null
  }
  if (!codingHook) {
    codingHookStarting = false
    return
  }
  try {
    codingHook.stop()
  } catch (e) {
    console.warn('[lingecho-desktop] stop coding hook failed', e)
  }
  codingHook = null
  codingHookStarting = false
  console.log('[lingecho-desktop] coding hook stopped')
}

function hasMacAccessibility() {
  if (process.platform !== 'darwin') return true
  try {
    return !!systemPreferences.isTrustedAccessibilityClient(false)
  } catch (_) {
    return true
  }
}

async function promptMacAccessibilityOnce() {
  if (process.platform !== 'darwin') return true
  if (hasMacAccessibility()) return true
  if (codingHookPrompted) return hasMacAccessibility()
  codingHookPrompted = true

  let response = 1
  try {
    const result = await dialog.showMessageBox({
      type: 'info',
      buttons: ['打开系统设置', '暂不开启'],
      defaultId: 0,
      cancelId: 1,
      title: APP_NAME,
      message: '敲代码监听需要「辅助功能」权限',
      detail:
        '桌宠是点击穿透窗口，收不到其它应用的键盘事件。\n请在「系统设置 → 隐私与安全性 → 辅助功能」中勾选 SoulMy（或你用来启动的 Terminal/Electron）。\n授权后请完全退出并重新打开桌宠。',
    })
    response = result.response
  } catch (_) {
    response = 0
  }

  if (response === 0) {
    try {
      systemPreferences.isTrustedAccessibilityClient(true)
    } catch (_) {
      /* ignore */
    }
    try {
      await shell.openExternal(
        'x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility',
      )
    } catch (_) {
      try {
        await shell.openExternal('x-apple.systempreferences:com.apple.preference.security?Privacy')
      } catch (e) {
        console.warn('[lingecho-desktop] open accessibility settings failed', e)
      }
    }
  }
  return hasMacAccessibility()
}

function scheduleCodingHookRetry() {
  if (codingHookRetryTimer) return
  codingHookRetryTimer = setInterval(() => {
    const cfg = loadConfig()
    const primary = cfg.pets.find((p) => p.id === cfg.primaryPetId) || cfg.pets[0]
    if (!primary || primary.watchCoding === false) {
      stopCodingHook()
      return
    }
    if (codingHook || codingHookStarting) return
    if (process.platform === 'darwin' && !hasMacAccessibility()) {
      console.log('[lingecho-desktop] coding hook: waiting for Accessibility…')
      return
    }
    void startCodingHook()
  }, 8000)
}

async function startCodingHook() {
  const cfg = loadConfig()
  const primary = cfg.pets.find((p) => p.id === cfg.primaryPetId) || cfg.pets[0]
  if (!primary || primary.watchCoding === false) {
    stopCodingHook()
    return false
  }
  if (codingHook) return true
  if (codingHookStarting) return false

  codingHookStarting = true
  try {
    const trusted = await promptMacAccessibilityOnce()
    if (!trusted) {
      console.warn(
        '[lingecho-desktop] coding hook blocked: grant Accessibility, then fully quit & relaunch',
      )
      scheduleCodingHookRetry()
      return false
    }

    const onKey = () => emitCodingKeyToPet()
    const onLog = (m) => console.log('[lingecho-desktop] coding hook', m)

    try {
      codingHook = codingHookApi.startInMainProcess(onKey)
      console.log('[lingecho-desktop] global coding hook ready (main process)')
      rebuildTray()
      return true
    } catch (mainErr) {
      console.warn('[lingecho-desktop] main-process coding hook failed:', mainErr && mainErr.message)
      if (!utilityProcess || typeof utilityProcess.fork !== 'function') throw mainErr
      codingHook = codingHookApi.startInUtilityProcess(utilityProcess, onKey, onLog)
      console.log('[lingecho-desktop] global coding hook ready (utilityProcess)')
      rebuildTray()
      return true
    }
  } catch (e) {
    codingHook = null
    console.warn('[lingecho-desktop] start coding hook failed', e)
    scheduleCodingHookRetry()
    return false
  } finally {
    codingHookStarting = false
  }
}

function syncCodingHook() {
  const cfg = loadConfig()
  const primary = cfg.pets.find((p) => p.id === cfg.primaryPetId) || cfg.pets[0]
  if (!primary || primary.watchCoding === false) {
    stopCodingHook()
    return
  }
  void startCodingHook()
  scheduleCodingHookRetry()
}

async function startDesktopTidy() {
  const win = getPrimaryPetWindow()
  if (win && !win.isDestroyed()) {
    try {
      const started = await win.webContents.executeJavaScript(
        `(function () {
          var api = window.LanlanPet || window.LingEchoWidget
          if (!api || typeof api.tidyDesktop !== 'function') return false
          api.tidyDesktop()
          return true
        })()`,
        true,
      )
      if (started) return
    } catch (e) {
      console.warn('[lingecho-desktop] start pet tidy animation failed', e)
    }
  }

  // Old/unavailable template fallback: still perform the requested tidy silently.
  try {
    desktopTidy.tidy()
  } catch (e) {
    console.warn('[lingecho-desktop] tidy desktop failed', e)
  }
}

function createTray() {
  if (tray) return
  const icon = loadTrayIcon()
  tray = new Tray(icon)
  tray.setToolTip(APP_NAME)
  tray.on('click', createSettingsWindow)
  rebuildTray()
}

ipcMain.handle('pet:get-config', (event) => {
  const doc = loadConfig()
  let login = { openAtLogin: doc.openAtLogin, openAsHidden: true }
  try {
    login = { ...login, ...app.getLoginItemSettings() }
  } catch {
    /* ignore */
  }
  const petId = petIdFromWebContents(event.sender)
  if (petId) {
    const entry = doc.pets.find((p) => p.id === petId)
    return { ...buildRuntimeConfig(doc, entry, petId), loginItem: login }
  }
  return { ...doc, loginItem: login }
})
ipcMain.handle('pet:save-config', (_e, cfg) => {
  const saved = saveConfig(cfg || {})
  registerGlobalShortcuts()
  rebuildTray()
  syncCodingHook()
  reloadPet()
  return saved
})
ipcMain.handle('pet:preview-desktop', () => {
  try {
    return { ok: true, ...desktopTidy.preview() }
  } catch (e) {
    return { ok: false, total: 0, groups: {}, error: e && e.message ? e.message : String(e) }
  }
})
ipcMain.handle('pet:tidy-desktop', () => {
  try {
    return desktopTidy.tidy()
  } catch (e) {
    return {
      ok: false,
      moved: 0,
      folders: [],
      errors: [e && e.message ? e.message : String(e)],
      skipped: 0,
    }
  }
})
ipcMain.handle('pet:toggle-panel', () => {
  toggleChatPanel()
})
ipcMain.handle('pet:open-settings', () => {
  createSettingsWindow()
})
ipcMain.handle('pet:set-open-at-login', (_e, enabled) => {
  setOpenAtLogin(enabled)
  return loadConfig()
})
ipcMain.handle('pet:open-embed-preview', (_e, petId) => openEmbedPreview(String(petId || '')))
ipcMain.handle('pet:set-ignore-mouse', (_e, ignore, forward = true) => {
  const win = BrowserWindow.fromWebContents(_e.sender)
  if (!win || win.isDestroyed()) return false
  if (ignore) {
    win.setIgnoreMouseEvents(true, forward ? { forward: true } : undefined)
  } else {
    win.setIgnoreMouseEvents(false)
  }
  return true
})

if (gotTheLock) {
  app.on('second-instance', () => {
    createTray()
    ensurePetAlive()
    createSettingsWindow()
  })

  app.whenReady().then(() => {
    if (process.platform === 'win32') {
      try {
        app.setAppUserModelId(APP_ID)
      } catch (e) {
        console.warn('[lingecho-desktop] setAppUserModelId failed', e)
      }
    }
    const appIcon = loadAppIcon()
    if (!appIcon.isEmpty()) {
      if (process.platform === 'darwin' && app.dock) {
        app.dock.setIcon(appIcon)
      }
      // Ensure Linux / Windows taskbar pick up the icon even before a window opens
      if (process.platform !== 'darwin') {
        try {
          app.setName(APP_NAME)
        } catch (e) {
          /* ignore */
        }
      }
    }
    const cfg = loadConfig()
    applyLoginItemSettings(cfg.openAtLogin)
    setupMediaPermissions()
    createTray()
    registerGlobalShortcuts()

    if (hasValidPetConfig(cfg)) {
      syncPetWindows()
    } else {
      createSettingsWindow()
    }
    syncCodingHook()
  })

  app.on('will-quit', () => {
    globalShortcut.unregisterAll()
    stopCodingHook()
  })

  app.on('activate', () => {
    createTray()
    ensurePetAlive()
    if (!settingsWindow || settingsWindow.isDestroyed()) {
      if (!hasValidPetConfig()) createSettingsWindow()
    } else {
      settingsWindow.show()
    }
  })

  // Tray residency: never quit when all windows are closed
  app.on('window-all-closed', () => {
    /* keep running in tray */
  })
}
