import { common, Language } from './modules/common'
import { pages } from './modules/pages'
import { auth } from './modules/auth'
import { assistant } from './modules/assistant'
import { voice } from './modules/voice'
import { knowledge } from './modules/knowledge'
import { workflow } from './modules/workflow'
import { billing } from './modules/billing'
import { groups } from './modules/groups'
import { alerts } from './modules/alerts'
import { notification } from './modules/notification'
import { credential } from './modules/credential'
import { device } from './modules/device'
import { jsTemplate } from './modules/jsTemplate'
import { quota } from './modules/quota'
import { resetPassword } from './modules/resetPassword'
import { animation } from './modules/animation'

// 合并所有翻译模块
function mergeTranslations(...modules: Record<Language, Record<string, string>>[]) {
  const result: Record<Language, Record<string, string>> = {
    zh: {},
    en: {},
    ja: {}
  }

  for (const module of modules) {
    for (const lang of Object.keys(module) as Language[]) {
      Object.assign(result[lang], module[lang])
    }
  }

  return result
}

export const translations = mergeTranslations(
  common,
  pages,
  auth,
  assistant,
  voice,
  knowledge,
  workflow,
  billing,
  groups,
  alerts,
  notification,
  credential,
  device,
  jsTemplate,
  quota,
  resetPassword,
  animation
)

export type { Language }