/// <reference types="vite/client" />

export type PetConfig = {
  serverBase: string
  jsSourceId: string
  assistantId: string
  apiKey: string
  transport: string
  title: string
  position: string
  primaryColor: string
  size: number
  autoWander: boolean
  autoChat: boolean
  watchCoding: boolean
  settingsHotkey: string
  panelHotkey: string
  voiceHotkey: string
  talkHotkey: string
  openAtLogin: boolean
  loginItem?: {
    openAtLogin?: boolean
    openAsHidden?: boolean
    wasOpenedAtLogin?: boolean
  }
}

export type ElectronPetAPI = {
  getConfig: () => Promise<PetConfig>
  saveConfig: (cfg: Partial<PetConfig>) => Promise<PetConfig>
  openSettings: () => Promise<void>
  setIgnoreMouseEvents: (ignore: boolean, forward?: boolean) => Promise<boolean>
  togglePanel: () => Promise<void>
  setOpenAtLogin: (enabled: boolean) => Promise<PetConfig>
  previewDesktop: () => Promise<{
    ok: boolean
    total: number
    groups: Record<string, string[]>
    error?: string
  }>
  tidyDesktop: () => Promise<{
    ok: boolean
    moved: number
    folders: string[]
    errors: string[]
    skipped: number
  }>
  platform: string
}

declare global {
  interface Window {
    electronPet?: ElectronPetAPI
  }
}

export {}
