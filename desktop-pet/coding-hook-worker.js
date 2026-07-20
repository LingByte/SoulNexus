/**
 * Global keyboard hook worker (Electron utilityProcess).
 * Observes keydowns without consuming them, so coding in other apps
 * can drive the desktop pet's watchCoding state machine.
 *
 * Isolated from the main process so a missing Accessibility grant on macOS
 * only kills this worker instead of crashing the whole app.
 */
'use strict'

// Always use platform prebuilds — never host build/Release from cross-packaging.
process.env.PREBUILDS_ONLY = '1'

let uIOhook
let UiohookKey
try {
  ;({ uIOhook, UiohookKey } = require('uiohook-napi'))
} catch (err) {
  if (process.parentPort) {
    process.parentPort.postMessage({
      type: 'error',
      message: 'uiohook-napi 未安装或无法加载: ' + (err && err.message ? err.message : String(err)),
    })
  }
  process.exit(1)
}

const IGNORE = new Set(
  [
    UiohookKey.Shift,
    UiohookKey.ShiftRight,
    UiohookKey.Ctrl,
    UiohookKey.CtrlRight,
    UiohookKey.Alt,
    UiohookKey.AltRight,
    UiohookKey.Meta,
    UiohookKey.MetaRight,
    UiohookKey.CapsLock,
    UiohookKey.Escape,
    UiohookKey.F1,
    UiohookKey.F2,
    UiohookKey.F3,
    UiohookKey.F4,
    UiohookKey.F5,
    UiohookKey.F6,
    UiohookKey.F7,
    UiohookKey.F8,
    UiohookKey.F9,
    UiohookKey.F10,
    UiohookKey.F11,
    UiohookKey.F12,
    UiohookKey.PrintScreen,
    UiohookKey.ScrollLock,
    UiohookKey.Pause,
  ].filter((v) => v != null),
)

function isCodingKey(e) {
  return !!(e && !IGNORE.has(e.keycode))
}

try {
  uIOhook.on('keydown', (e) => {
    if (!isCodingKey(e)) return
    if (process.parentPort) {
      process.parentPort.postMessage({ type: 'coding-key', t: Date.now() })
    }
  })
  uIOhook.start()
  if (process.parentPort) {
    process.parentPort.postMessage({ type: 'ready' })
  }
} catch (err) {
  if (process.parentPort) {
    process.parentPort.postMessage({
      type: 'error',
      message: err && err.message ? err.message : String(err),
    })
  }
  process.exit(1)
}

function shutdown() {
  try {
    uIOhook.stop()
  } catch (_) {
    /* ignore */
  }
  process.exit(0)
}

if (process.parentPort) {
  process.parentPort.on('message', (e) => {
    if (e && e.data && e.data.type === 'stop') shutdown()
  })
}

process.on('disconnect', shutdown)
