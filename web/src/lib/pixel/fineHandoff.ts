/** sessionStorage key for process → fine editor handoff (data URL). */
export const PIXEL_FINE_HANDOFF_KEY = 'soulnexus_pixel_fine_handoff_v1'

export function setFineHandoffDataUrl(dataUrl: string) {
  try {
    sessionStorage.setItem(PIXEL_FINE_HANDOFF_KEY, dataUrl)
  } catch {
    /* quota */
  }
}

export function takeFineHandoffDataUrl(): string | null {
  try {
    const v = sessionStorage.getItem(PIXEL_FINE_HANDOFF_KEY)
    if (v) sessionStorage.removeItem(PIXEL_FINE_HANDOFF_KEY)
    return v
  } catch {
    return null
  }
}
