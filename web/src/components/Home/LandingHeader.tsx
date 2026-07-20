import { Link } from 'react-router-dom'
import { IconMoon, IconSun } from '@arco-design/web-react/icon'
import { Menu, X } from 'lucide-react'
import { Button } from '@/components/ui'
import { BRAND_LOGO_SRC } from '@/utils/brandLogo'
import { useTranslation } from '@/i18n'
import { useThemeStore } from '@/stores/themeStore'
import type { User } from '@/stores/authStore'
import LocaleMenu from '@/components/Home/LocaleMenu'
import LandingUserMenu, { LandingLoginChip } from '@/components/Home/LandingUserMenu'

type LandingHeaderProps = {
  loggedIn: boolean
  user: User | null
  mobileOpen: boolean
  onToggleMobile: () => void
  onCloseMobile: () => void
  onConsole: () => void
  onLogout: () => void
  onLogin: () => void
}

export default function LandingHeader({
  loggedIn,
  user,
  mobileOpen,
  onToggleMobile,
  onCloseMobile,
  onConsole,
  onLogout,
  onLogin,
}: LandingHeaderProps) {
  const { t } = useTranslation()
  const isDark = useThemeStore((s) => s.isDark)
  const toggleMode = useThemeStore((s) => s.toggleMode)

  return (
    <header className="sticky top-0 z-40 border-b border-[hsl(var(--border)/0.6)] bg-[hsl(var(--background)/0.82)] backdrop-blur-xl">
      <div className="mx-auto flex h-[4.25rem] max-w-7xl items-center justify-between gap-4 px-4 sm:px-6 lg:px-8">
        <Link to="/" className="flex shrink-0 items-center gap-2.5">
          <img src={BRAND_LOGO_SRC} alt="" className="logo-brand h-9 w-9 rounded-lg object-contain" />
          <span className="font-display text-lg font-bold tracking-tight text-violet-600 dark:text-violet-300">
            {t('layout.siteName')}
          </span>
        </Link>

        <div className="hidden items-center gap-2 md:flex">
          <LocaleMenu />
          <button
            type="button"
            onClick={toggleMode}
            className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-[hsl(var(--border))] text-[hsl(var(--muted-foreground))] transition hover:border-violet-400/50 hover:bg-[hsl(var(--muted))]"
            aria-label={isDark ? t('landing.themeLight') : t('landing.themeDark')}
          >
            {isDark ? <IconSun /> : <IconMoon />}
          </button>
          {loggedIn && user ? (
            <LandingUserMenu user={user} onConsole={onConsole} onLogout={onLogout} />
          ) : (
            <LandingLoginChip onLogin={onLogin} />
          )}
        </div>

        <button
          type="button"
          className="rounded-lg p-2 md:hidden"
          onClick={onToggleMobile}
          aria-label={t('landing.menuAria')}
        >
          {mobileOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
        </button>
      </div>

      {mobileOpen ? (
        <div className="border-t border-[hsl(var(--border))] px-4 py-4 md:hidden">
          <div className="mb-3 flex items-center justify-between gap-2">
            <LocaleMenu align="left" />
            <button
              type="button"
              onClick={toggleMode}
              className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-[hsl(var(--border))]"
            >
              {isDark ? <IconSun /> : <IconMoon />}
            </button>
          </div>
          <div className="border-t border-[hsl(var(--border))] pt-3">
            {loggedIn && user ? (
              <div className="flex flex-col gap-2">
                <Button
                  variant="outline"
                  onClick={() => {
                    onCloseMobile()
                    onConsole()
                  }}
                >
                  {t('landing.ctaConsole')}
                </Button>
                <Button
                  variant="ghost"
                  onClick={() => {
                    onCloseMobile()
                    onLogout()
                  }}
                >
                  {t('profile.logout')}
                </Button>
              </div>
            ) : (
              <Button
                variant="primary"
                className="w-full"
                onClick={() => {
                  onCloseMobile()
                  onLogin()
                }}
              >
                {t('auth.login')}
              </Button>
            )}
          </div>
        </div>
      ) : null}
    </header>
  )
}
