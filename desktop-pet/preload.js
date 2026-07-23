const { contextBridge, ipcRenderer } = require('electron')

contextBridge.exposeInMainWorld('electronPet', {
  getConfig: () => ipcRenderer.invoke('pet:get-config'),
  saveConfig: (cfg) => ipcRenderer.invoke('pet:save-config', cfg),
  openSettings: () => ipcRenderer.invoke('pet:open-settings'),
  setIgnoreMouseEvents: (ignore, forward = true) =>
    ipcRenderer.invoke('pet:set-ignore-mouse', ignore, forward),
  togglePanel: () => ipcRenderer.invoke('pet:toggle-panel'),
  setOpenAtLogin: (enabled) => ipcRenderer.invoke('pet:set-open-at-login', enabled),
  openEmbedPreview: (petId) => ipcRenderer.invoke('pet:open-embed-preview', petId),
  previewDesktop: () => ipcRenderer.invoke('pet:preview-desktop'),
  tidyDesktop: () => ipcRenderer.invoke('pet:tidy-desktop'),
  /** Subscribe to global coding key events from the main-process hook. Returns unsubscribe. */
  onCodingKey: (handler) => {
    if (typeof handler !== 'function') return () => {}
    const listener = (_event, payload) => {
      try {
        handler(payload || {})
      } catch (e) {
        console.warn('[electronPet] onCodingKey handler error', e)
      }
    }
    ipcRenderer.on('pet:coding-key', listener)
    return () => {
      ipcRenderer.removeListener('pet:coding-key', listener)
    }
  },
  platform: process.platform,
})
