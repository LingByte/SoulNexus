import type { DesktopPetConfig, PetEntry } from '@/vite-env'

export const DEFAULT_SERVER = 'https://soulmy.top/api'
export const DEFAULT_JS_SOURCE = 'js_75cad2ab4f9f142a'

export function newPetId() {
  return 'pet_' + Math.random().toString(36).slice(2, 10)
}

export function createPetEntry(partial?: Partial<PetEntry>): PetEntry {
  return {
    id: partial?.id || newPetId(),
    name: partial?.name || '新桌宠',
    jsSourceId: partial?.jsSourceId || DEFAULT_JS_SOURCE,
    enabled: partial?.enabled !== false,
    title: partial?.title || partial?.name || '懒懒',
    position: partial?.position || 'right',
    size: partial?.size && partial.size > 0 ? partial.size : 160,
    autoWander: partial?.autoWander !== false,
    autoChat: partial?.autoChat !== false,
    watchCoding: partial?.watchCoding !== false,
  }
}

export const EMPTY_CONFIG: DesktopPetConfig = {
  serverBase: DEFAULT_SERVER,
  assistantId: '8859281265343332864',
  apiKey: 'soulnexus_user_PI2mRsBxioqpTkAS3K3yG3Z_YzY2smCsidEWJRuamMI',
  transport: 'websocket',
  primaryColor: '#18181B',
  settingsHotkey: 'CommandOrControl+Alt+P',
  panelHotkey: 'CommandOrControl+Alt+V',
  voiceHotkey: 'Alt+Shift+V',
  talkHotkey: 'Alt+Shift+T',
  openAtLogin: false,
  primaryPetId: '',
  pets: [createPetEntry({ name: '懒懒', title: '懒懒', jsSourceId: DEFAULT_JS_SOURCE })],
}

export function embedUrl(serverBase: string, jsSourceId: string) {
  const base = String(serverBase || '').replace(/\/+$/, '')
  const id = String(jsSourceId || '').trim()
  if (!id || id === 'YOUR_JS_SOURCE_ID' || id === 'default') {
    return `${base}/lingecho/embed/v1/embed.js`
  }
  return `${base}/lingecho/embed/v1/t/${encodeURIComponent(id)}/embed.js`
}

/** Merge API response (flat legacy or pets[]) into DesktopPetConfig */
export function normalizeConfig(raw: Record<string, unknown>): DesktopPetConfig {
  const base: DesktopPetConfig = {
    ...EMPTY_CONFIG,
    ...(raw as Partial<DesktopPetConfig>),
  }
  if (Array.isArray(raw.pets) && raw.pets.length > 0) {
    base.pets = (raw.pets as PetEntry[]).map((p) => createPetEntry(p))
  } else {
    const legacyId = String(raw.jsSourceId || DEFAULT_JS_SOURCE).trim()
    base.pets = [
      createPetEntry({
        id: newPetId(),
        name: String(raw.title || '懒懒'),
        title: String(raw.title || '懒懒'),
        jsSourceId: legacyId || DEFAULT_JS_SOURCE,
        enabled: true,
        position: (raw.position as 'left' | 'right') || 'right',
        size: Number(raw.size) || 160,
        autoWander: raw.autoWander !== false,
        autoChat: raw.autoChat !== false,
        watchCoding: raw.watchCoding !== false,
      }),
    ]
  }
  if (!base.primaryPetId) {
    const first = base.pets.find((p) => p.enabled) || base.pets[0]
    base.primaryPetId = first?.id || ''
  }
  return base
}

export function payloadForSave(form: DesktopPetConfig): DesktopPetConfig {
  const pets = form.pets.map((p) => ({
    ...p,
    name: p.name.trim() || '桌宠',
    jsSourceId: p.jsSourceId.trim(),
    title: (p.title || p.name).trim() || '懒懒',
    size: Math.max(96, Math.min(256, Number(p.size) || 160)),
    enabled: p.enabled !== false,
  }))
  let primaryPetId = form.primaryPetId
  if (!pets.some((p) => p.id === primaryPetId)) {
    primaryPetId = pets.find((p) => p.enabled)?.id || pets[0]?.id || ''
  }
  return {
    serverBase: form.serverBase.trim(),
    assistantId: form.assistantId.trim(),
    apiKey: form.apiKey.trim(),
    transport: form.transport || 'websocket',
    primaryColor: form.primaryColor.trim() || '#18181B',
    settingsHotkey: form.settingsHotkey.trim(),
    panelHotkey: form.panelHotkey.trim(),
    voiceHotkey: form.voiceHotkey.trim(),
    talkHotkey: form.talkHotkey.trim(),
    openAtLogin: Boolean(form.openAtLogin),
    primaryPetId,
    pets,
  }
}
