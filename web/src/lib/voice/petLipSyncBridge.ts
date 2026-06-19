/**
 * Bridges TTS / WebRTC audio volume → sprite pet mouth / talk animation.
 */

const PET_CHANNEL = 'soul-pet'

type SpritePetApi = {
  setVolume?: (level: number) => void
  setPhase?: (phase: string) => void
}

declare global {
  interface Window {
    __PET_SPRITE__?: SpritePetApi
    __SOUL_PET_LIP_SYNC__?: { setVolume: (level: number) => number }
  }
}

let smoothed = 0
let rafId = 0
let lastActiveMs = 0

const ATTACK = 0.55
const RELEASE = 0.18

function smoothVolume(target: number): number {
  const rate = target > smoothed ? ATTACK : RELEASE
  smoothed += (target - smoothed) * rate
  if (smoothed < 0.02) smoothed = 0
  return Math.min(1, Math.max(0, smoothed))
}

function applyMouthLevel(level: number): void {
  const sprite = window.__PET_SPRITE__
  if (typeof sprite?.setVolume === 'function') {
    sprite.setVolume(level)
    return
  }
  if (window.__SOUL_PET_LIP_SYNC__?.setVolume) {
    window.__SOUL_PET_LIP_SYNC__.setVolume(level)
  }
}

function tickLoop(): void {
  applyMouthLevel(smoothed)
  const idle = performance.now() - lastActiveMs > 250 && smoothed < 0.02
  if (idle) {
    smoothed = 0
    applyMouthLevel(0)
    window.__PET_SPRITE__?.setPhase?.('idle')
    rafId = 0
    return
  }
  rafId = requestAnimationFrame(tickLoop)
}

function ensureLoop(): void {
  if (!rafId) rafId = requestAnimationFrame(tickLoop)
}

export function applyPetLipSync(rawLevel: number): void {
  smoothed = smoothVolume(rawLevel)
  lastActiveMs = performance.now()
  if (window.__SOUL_PET_LIP_SYNC__?.setVolume) {
    window.__SOUL_PET_LIP_SYNC__.setVolume(smoothed)
    return
  }
  ensureLoop()
}

export function resetPetLipSync(): void {
  smoothed = 0
  lastActiveMs = 0
  if (window.__SOUL_PET_LIP_SYNC__?.setVolume) {
    window.__SOUL_PET_LIP_SYNC__.setVolume(0)
  } else {
    applyMouthLevel(0)
  }
  window.__PET_SPRITE__?.setPhase?.('idle')
  if (rafId) {
    cancelAnimationFrame(rafId)
    rafId = 0
  }
}

export function postPetLipSyncToIframe(iframe: HTMLIFrameElement | null | undefined, level: number): void {
  if (!iframe?.contentWindow) return
  iframe.contentWindow.postMessage(
    {
      channel: PET_CHANNEL,
      type: 'host:lipsync',
      payload: { volume: level },
    },
    '*',
  )
}

export function installPetLipSyncListener(): () => void {
  const onMessage = (e: MessageEvent) => {
    const data = e.data as { channel?: string; type?: string; payload?: { volume?: number } }
    if (data?.channel !== PET_CHANNEL || data.type !== 'host:lipsync') return
    const v = Number(data.payload?.volume)
    if (Number.isFinite(v)) applyPetLipSync(Math.min(1, Math.max(0, v)))
  }
  window.addEventListener('message', onMessage)
  return () => window.removeEventListener('message', onMessage)
}
