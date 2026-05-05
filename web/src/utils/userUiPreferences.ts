import type { Language } from '@/stores/i18nStore'
import { useI18nStore } from '@/stores/i18nStore'
import { useThemeStore, type ThemeMode, type ThemeColor } from '@/stores/themeStore'
import { normalizeLegacyLocale } from '@/utils/authUserProfile'

const VALID_LANGS = new Set<Language>(['zh', 'en', 'ja', 'zh-TW', 'fr'])

/** Apply server-stored locale / theme to client stores after login or `/auth/info`. */
export function applyAuthUserUIPreferences(user: Record<string, unknown> | null | undefined): void {
  if (!user) return

  const locRaw =
    (typeof user.preferredLocale === 'string' && user.preferredLocale.trim()) ||
    (typeof user.locale === 'string' && user.locale.trim()) ||
    ''
  if (locRaw) {
    const loc = normalizeLegacyLocale(locRaw) as Language
    if (VALID_LANGS.has(loc)) {
      useI18nStore.getState().setLanguage(loc)
    }
  }

  const modeRaw = typeof user.themeMode === 'string' ? user.themeMode.trim().toLowerCase() : ''
  const colorRaw = typeof user.themeColor === 'string' ? user.themeColor.trim().toLowerCase() : ''
  const mode =
    modeRaw === 'light' || modeRaw === 'dark' || modeRaw === 'system' ? (modeRaw as ThemeMode) : null
  const palette: ThemeColor[] = ['default', 'cherry', 'ocean', 'nature', 'fresh', 'sunset', 'lavender']
  const color = palette.includes(colorRaw as ThemeColor) ? (colorRaw as ThemeColor) : null

  if (mode && color) {
    useThemeStore.getState().setTheme({ mode, color })
  } else if (mode) {
    const cur = useThemeStore.getState().theme
    useThemeStore.getState().setTheme({ mode, color: cur.color })
  } else if (color) {
    const cur = useThemeStore.getState().theme
    useThemeStore.getState().setTheme({ ...cur, color })
  }
}
