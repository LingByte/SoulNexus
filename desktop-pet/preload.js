const { contextBridge, ipcRenderer } = require('electron')

contextBridge.exposeInMainWorld('electronPet', {
  getConfig: () => ipcRenderer.invoke('pet:get-config'),
  saveConfig: (cfg) => ipcRenderer.invoke('pet:save-config', cfg),
  openSettings: () => ipcRenderer.invoke('pet:open-settings'),
  setIgnoreMouseEvents: (ignore, forward = true) =>
    ipcRenderer.invoke('pet:set-ignore-mouse', ignore, forward),
  pickLocalFolder: () => ipcRenderer.invoke('pet:pick-local-folder'),
  readLocalPackage: () => ipcRenderer.invoke('pet:read-local-package'),
  validateLocalPackage: (rootPath) => ipcRenderer.invoke('pet:validate-local-package', rootPath),
  platform: process.platform,
})
