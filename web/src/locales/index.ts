import { common, Language as ModuleLanguage } from './modules/common'
import { pages } from './modules/pages'
import { auth } from './modules/auth'
import { assistant } from './modules/assistant'
import { voice } from './modules/voice'
import { workflow } from './modules/workflow'
import { billing } from './modules/billing'
import { groups } from './modules/groups'
import { notification } from './modules/notification'
import { credential } from './modules/credential'
import { device } from './modules/device'
import { jsTemplate } from './modules/jsTemplate'
import { resetPassword } from './modules/resetPassword'
import { animation } from './modules/animation'
import { zhTWOverrides } from './overrides/zhTW'
import { frOverrides } from './overrides/fr'

// 合并所有翻译模块
export type Language = ModuleLanguage | 'zh-TW' | 'fr'

function mergeTranslations(...modules: Record<ModuleLanguage, Record<string, string>>[]) {
  const result: Record<Language, Record<string, string>> = {
    zh: {},
    en: {},
    ja: {},
    'zh-TW': {},
    fr: {},
  }

  for (const module of modules) {
    for (const lang of Object.keys(module) as ModuleLanguage[]) {
      Object.assign(result[lang], module[lang])
    }
  }

  // Fallback strategy for newly added locales.
  // zh-TW uses zh baseline; fr uses en baseline until dedicated translations are added.
  result['zh-TW'] = { ...result.zh, ...result['zh-TW'] }
  result.fr = { ...result.en, ...result.fr }
  result['zh-TW'] = { ...result['zh-TW'], ...zhTWOverrides }
  result.fr = { ...result.fr, ...frOverrides }

  return result
}

export const translations = mergeTranslations(
  common,
  pages,
  auth,
  assistant,
  voice,
  workflow,
  billing,
  groups,
  notification,
  credential,
  device,
  jsTemplate,
  resetPassword,
  animation
)