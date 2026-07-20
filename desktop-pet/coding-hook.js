/**
 * Global keyboard hook for watchCoding.
 * Prefer running in the Electron main process (simpler native-module resolution).
 * Falls back to a utilityProcess worker if main-process start fails.
 *
 * Always load N-API prebuilds (not build/Release). Cross-packaging from macOS
 * otherwise ships a darwin .node that node-gyp-build prefers on Windows and
 * silently breaks the hook.
 */
'use strict'

const path = require('path')
const fs = require('fs')

const IGNORE_NAMES = [
  'Shift',
  'ShiftRight',
  'Ctrl',
  'CtrlRight',
  'Alt',
  'AltRight',
  'Meta',
  'MetaRight',
  'CapsLock',
  'Escape',
  'F1',
  'F2',
  'F3',
  'F4',
  'F5',
  'F6',
  'F7',
  'F8',
  'F9',
  'F10',
  'F11',
  'F12',
  'PrintScreen',
  'ScrollLock',
  'Pause',
]

function loadUiohook() {
  // Prefer prebuilds/<platform>-<arch> over leftover host build/Release/*.node
  const prev = process.env.PREBUILDS_ONLY
  process.env.PREBUILDS_ONLY = '1'
  try {
    return require('uiohook-napi')
  } finally {
    if (prev === undefined) delete process.env.PREBUILDS_ONLY
    else process.env.PREBUILDS_ONLY = prev
  }
}

function buildIgnoreSet(UiohookKey) {
  return new Set(IGNORE_NAMES.map((n) => UiohookKey[n]).filter((v) => v != null))
}

function attachHook(uIOhook, UiohookKey, onKey) {
  const ignore = buildIgnoreSet(UiohookKey)
  const handler = (e) => {
    if (!e || ignore.has(e.keycode)) return
    onKey()
  }
  uIOhook.on('keydown', handler)
  uIOhook.start()
  return () => {
    try {
      if (typeof uIOhook.removeListener === 'function') uIOhook.removeListener('keydown', handler)
      else if (typeof uIOhook.off === 'function') uIOhook.off('keydown', handler)
    } catch (_) {
      /* ignore */
    }
    try {
      uIOhook.stop()
    } catch (_) {
      /* ignore */
    }
  }
}

function startInMainProcess(onKey) {
  const mod = loadUiohook()
  const stop = attachHook(mod.uIOhook, mod.UiohookKey, onKey)
  return { mode: 'main', stop }
}

function startInUtilityProcess(utilityProcess, onKey, onLog) {
  const workerPath = path.join(__dirname, 'coding-hook-worker.js')
  if (!fs.existsSync(workerPath)) {
    throw new Error('coding-hook-worker.js missing')
  }
  // IMPORTANT: args is the 2nd parameter; options is the 3rd.
  const child = utilityProcess.fork(workerPath, [], {
    serviceName: 'lingecho-coding-hook',
    env: {
      ...process.env,
      PREBUILDS_ONLY: '1',
    },
  })
  const onMessage = (msg) => {
    if (!msg || typeof msg !== 'object') return
    if (msg.type === 'coding-key') onKey()
    else if (msg.type === 'ready' && onLog) onLog('worker ready')
    else if (msg.type === 'error' && onLog) onLog('worker error: ' + msg.message)
  }
  child.on('message', onMessage)
  child.on('exit', (code) => {
    if (onLog) onLog('worker exited: ' + code)
  })
  return {
    mode: 'worker',
    stop: () => {
      try {
        child.postMessage({ type: 'stop' })
      } catch (_) {
        /* ignore */
      }
      try {
        child.kill()
      } catch (_) {
        /* ignore */
      }
    },
  }
}

module.exports = {
  startInMainProcess,
  startInUtilityProcess,
  loadUiohook,
}
