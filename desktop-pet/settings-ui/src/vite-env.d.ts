/// <reference types="vite/client" />

export type PetEntry = {
  id: string
  name: string
  jsSourceId: string
  enabled: boolean
  title: string
  position: 'left' | 'right'
  size: number
  autoWander: boolean
  autoChat: boolean
  watchCoding: boolean
}

/** Settings document (multi-pet) */
export type DesktopPetConfig = {
  serverBase: string
  assistantId: string
  apiKey: string
  transport: string
  primaryColor: string
  settingsHotkey: string
  panelHotkey: string
  voiceHotkey: string
  talkHotkey: string
  openAtLogin: boolean
  primaryPetId: string
  pets: PetEntry[]
  loginItem?: {
    openAtLogin?: boolean
    openAsHidden?: boolean
    wasOpenedAtLogin?: boolean
  }
}

/** Flat config returned to pet renderer boot.js */
export type PetRuntimeConfig = {
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
  voiceHotkey: string
  talkHotkey: string
  openAtLogin: boolean
  petId?: string
  petName?: string
}

export type ElectronPetAPI = {
  getConfig: () => Promise<DesktopPetConfig & Partial<PetRuntimeConfig>>
  saveConfig: (cfg: DesktopPetConfig) => Promise<DesktopPetConfig>
  openSettings: () => Promise<void>
  setIgnoreMouseEvents: (ignore: boolean, forward?: boolean) => Promise<boolean>
  togglePanel: () => Promise<void>
  setOpenAtLogin: (enabled: boolean) => Promise<DesktopPetConfig>
  openEmbedPreview: (petId: string) => Promise<{ ok: boolean; error?: string }>
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
