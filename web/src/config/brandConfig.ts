/**
 * Brand display on auth pages.
 * - Header / login title: backend SITE_NAME
 * - Footer copyright company: VITE_COMPANY_NAME (fallback SITE_NAME)
 * - Logo URL: backend SITE_LOGO_URL
 */

export function copyrightCompanyFromEnv(): string | undefined {
  const v = import.meta.env.VITE_COMPANY_NAME
  if (typeof v !== 'string') return undefined
  const s = v.trim()
  return s || undefined
}

export function resolveCompanyName(siteConfigName?: string, fallback = 'SoulNexus'): string {
  return siteConfigName?.trim() || fallback
}

/** Footer © line — legal/display name from env, else site config. */
export function resolveCopyrightCompany(siteConfigName?: string, fallback = 'SoulNexus'): string {
  return copyrightCompanyFromEnv() || siteConfigName?.trim() || fallback
}

export function resolveLogoUrl(siteConfigLogo?: string, fallback = '/icon-lingyu.png'): string {
  return siteConfigLogo?.trim() || fallback
}

export function formatAuthCopyright(company: string, year = new Date().getFullYear()): string {
  return `© ${year} ${company}`
}

function trimEnvString(v: unknown): string | undefined {
  if (typeof v !== 'string') return undefined
  const s = v.trim()
  return s || undefined
}

/** ICP 备案号（如 沪 ICP 备 xxxxx 号） */
export function icpNumberFromEnv(): string | undefined {
  return trimEnvString(import.meta.env.VITE_ICP_NUMBER)
}

/** ICP 查询链接，默认工信部备案系统 */
export function icpLinkFromEnv(): string {
  return trimEnvString(import.meta.env.VITE_ICP_LINK) || 'https://beian.miit.gov.cn/'
}

/** 公安备案号（可选） */
export function publicSecurityRecordFromEnv(): string | undefined {
  return trimEnvString(import.meta.env.VITE_PUBLIC_SECURITY_RECORD)
}

export function publicSecurityLinkFromEnv(): string | undefined {
  return trimEnvString(import.meta.env.VITE_PUBLIC_SECURITY_LINK)
}

export function contactEmailFromEnv(): string | undefined {
  return trimEnvString(import.meta.env.VITE_CONTACT_EMAIL)
}

export function githubUrlFromEnv(): string | undefined {
  return trimEnvString(import.meta.env.VITE_GITHUB_URL)
}

export function privacyUrlFromEnv(): string | undefined {
  return trimEnvString(import.meta.env.VITE_PRIVACY_URL)
}

export function termsUrlFromEnv(): string | undefined {
  return trimEnvString(import.meta.env.VITE_TERMS_URL)
}
