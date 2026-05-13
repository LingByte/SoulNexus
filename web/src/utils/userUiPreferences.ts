import type { Language } from '@/stores/i18nStore'
import { useI18nStore } from '@/stores/i18nStore'
import { normalizeLegacyLocale } from '@/utils/authUserProfile'

const VALID_LANGS = new Set<Language>(['zh', 'en', 'ja', 'zh-TW', 'fr'])

/** Apply server-stored locale to client after login or `/auth/info`. Theme is not overwritten here (see Profile save). */
export function applyAuthUserUIPreferences(user: Record<string, unknown> | null | undefined): void {
  if (!user) return

  const locRaw =
    (typeof user.locale === 'string' && user.locale.trim()) ||
    (typeof user.preferredLocale === 'string' && user.preferredLocale.trim()) ||
    ''
  if (locRaw) {
    const loc = normalizeLegacyLocale(locRaw) as Language
    if (VALID_LANGS.has(loc)) {
      useI18nStore.getState().setLanguage(loc)
    }
  }

  // 主题（明暗、色板）完全由客户端 theme-storage 与资料页保存驱动；登录/refreshUserInfo 不覆盖，
  // 避免服务端旧值覆盖首页刚切换的暗色。保存资料见 Profile handleSave 中的 setTheme。
}
