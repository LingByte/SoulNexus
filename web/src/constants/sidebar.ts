export const SIDEBAR_COLLAPSED_WIDTH = 80
export const SIDEBAR_DEFAULT_WIDTH = 220
export const SIDEBAR_MIN_WIDTH = 180
export const SIDEBAR_MAX_WIDTH = 420
export const SIDEBAR_WIDTH_STORAGE_KEY = 'lingecho_sidebar_width'

export function clampSidebarWidth(width: number): number {
  return Math.round(Math.min(SIDEBAR_MAX_WIDTH, Math.max(SIDEBAR_MIN_WIDTH, width)))
}
