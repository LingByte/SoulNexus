import { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { IconLanguage } from '@arco-design/web-react/icon'
import { useSiteConfig } from '@/contexts/siteConfig'
import { formatAuthCopyright, resolveCompanyName, resolveCopyrightCompany, resolveLogoUrl } from '@/config/brandConfig'
import { useLocaleStore } from '@/stores/localeStore'
import { useTranslation } from '@/i18n'
import { Button, Select } from '@/components/ui'
import AuthCarousel from '@/components/Auth/AuthCarousel'

export type AuthPageMode = 'login' | 'register'

type AuthSplitLayoutProps = {
  children: ReactNode
  mode?: AuthPageMode
  onSwitchMode?: (mode: AuthPageMode) => void
  /** Standalone auth subpage (e.g. account deletion revoke) — hides login/register switch. */
  subpage?: boolean
}

export default function AuthSplitLayout({ children, mode = 'login', onSwitchMode, subpage = false }: AuthSplitLayoutProps) {
  const { t } = useTranslation()
  const { config } = useSiteConfig()
  const locale = useLocaleStore((s) => s.locale)
  const setLocale = useLocaleStore((s) => s.setLocale)

  const siteName = resolveCompanyName(config.SITE_NAME, t('layout.siteName'))
  const copyrightCompany = resolveCopyrightCompany(config.SITE_NAME, t('layout.siteName'))
  const logoUrl = resolveLogoUrl(config.SITE_LOGO_URL)
  const year = new Date().getFullYear()

  return (
    <div className="flex h-screen overflow-hidden bg-white text-neutral-900">
      <div className="flex h-screen w-full shrink-0 flex-col overflow-hidden lg:w-[44%] xl:w-[42%]">
        <header className="flex shrink-0 items-center justify-between gap-4 px-6 py-4 sm:px-10 lg:px-12">
          <Link to="/" className="flex min-w-0 items-center gap-3 hover:opacity-90">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center overflow-hidden rounded-xl bg-neutral-50 ring-1 ring-neutral-100">
              <img
                src={logoUrl}
                alt={siteName}
                className="logo-brand h-7 w-7 object-contain"
                draggable={false}
              />
            </div>
            <span className="truncate text-lg font-semibold tracking-tight">{siteName}</span>
          </Link>
          <div className="flex shrink-0 items-center gap-2 sm:gap-3">
            <a
              href="https://docs.lingecho.com"
              target="_blank"
              rel="noopener noreferrer"
              className="hidden text-sm text-neutral-500 transition-colors hover:text-neutral-900 sm:inline"
            >
              {t('nav.docsCenter')}
            </a>
            <div className="flex items-center gap-1 text-sm text-neutral-500">
              <IconLanguage style={{ fontSize: 16 }} />
              <Select
                size="small"
                bordered={false}
                allowClear={false}
                value={locale}
                style={{ width: 88 }}
                options={[
                  { value: 'zh-CN', label: t('locale.zh') },
                  { value: 'zh-TW', label: t('locale.tw') },
                  { value: 'en-US', label: t('locale.en') },
                  { value: 'ja-JP', label: t('locale.ja') },
                ]}
                onChange={(v) => {
                  if (!v) return
                  setLocale(v as 'zh-CN' | 'zh-TW' | 'en-US' | 'ja-JP')
                }}
              />
            </div>
            {!subpage && onSwitchMode && mode === 'register' ? (
              <>
                <span className="hidden text-sm text-neutral-500 md:inline">{t('auth.hasAccount')}</span>
                <Button type="outline" size="small" onClick={() => onSwitchMode('login')}>
                  {t('auth.goLogin')}
                </Button>
              </>
            ) : null}
          </div>
        </header>

        <main className="flex min-h-0 flex-1 flex-col justify-center overflow-y-auto overscroll-contain px-6 py-2 sm:px-10 lg:px-12 xl:px-16 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
          <div className="mx-auto w-full max-w-[420px] py-2">{children}</div>
        </main>

        <footer className="shrink-0 px-6 pb-4 text-xs text-neutral-400 sm:px-10 lg:px-12">
          {formatAuthCopyright(copyrightCompany, year)}
        </footer>
      </div>

      <aside className="hidden h-screen shrink-0 flex-1 p-6 pl-0 lg:flex">
        <div className="h-full w-full">
          <AuthCarousel />
        </div>
      </aside>
    </div>
  )
}
