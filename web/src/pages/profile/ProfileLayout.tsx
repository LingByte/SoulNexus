import { Outlet, NavLink, useLocation, Link } from 'react-router-dom'
import {
  User,
  Users,
  Clock,
  Receipt,
  Settings,
  Globe2,
  Shield,
  Key,
  Bell,
  LogOut,
  Smartphone,
  ArrowLeft,
  Sparkles,
  ChevronRight,
} from 'lucide-react'
import { useAuthStore } from '@/stores/authStore'
import { useI18nStore } from '@/stores/i18nStore'
import { cn } from '@/utils/cn'
import packageJson from '../../../package.json'
import { useMemo } from 'react'

const nav = [
  { to: '/profile/personal', label: '个人信息', icon: User },
  { to: '/profile/teams', label: '团队管理', icon: Users },
  { to: '/profile/activity', label: '使用记录', icon: Clock },
  { to: '/profile/billing', label: '账单与发票', icon: Receipt },
  { to: '/profile/user-devices', label: '用户设备', icon: Smartphone },
  { to: '/profile/account-security', label: '账号与绑定安全', icon: Shield },
  { to: '/profile/credential', label: 'API 密钥', icon: Key },
  { to: '/profile/llm-tokens', label: 'LLM Token', icon: Sparkles },
  { to: '/profile/notifications', label: '通知', icon: Bell },
  { to: '/profile/locale', label: '语言与时区', icon: Globe2 },
] as const

const ProfileLayout = () => {
  const { logout } = useAuthStore()
  const { t } = useI18nStore()
  const location = useLocation()

  const currentNavLabel = useMemo(() => {
    const path = location.pathname
    const hit = nav.find(
      ({ to }) => path === to || path.startsWith(`${to}/`),
    )
    return hit?.label ?? t('nav.sidebar.profile')
  }, [location.pathname, t])

  const breadcrumbMb =
    location.pathname === '/profile/user-devices'
      ? 'mb-2 shrink-0'
      : location.pathname === '/profile/personal'
        ? 'mb-1.5 shrink-0'
        : 'mb-5'

  return (
    <div className="min-h-screen md:h-[calc(100dvh)] md:min-h-0 bg-slate-50 dark:bg-gray-950 flex flex-col">
      <header className="shrink-0 z-20 flex h-14 items-center gap-3 border-b border-slate-200 bg-white px-4 dark:border-gray-800 dark:bg-gray-900">
        <Link
          to="/assistants"
          className="inline-flex items-center gap-2 rounded-lg px-2 py-1.5 text-sm font-medium text-slate-600 transition-colors hover:bg-slate-100 hover:text-slate-900 dark:text-gray-300 dark:hover:bg-gray-800 dark:hover:text-white"
        >
          <ArrowLeft className="h-4 w-4 shrink-0" aria-hidden />
          {t('nav.enterSystem')}
        </Link>
        <span className="h-5 w-px shrink-0 bg-slate-200 dark:bg-gray-700" aria-hidden />
        <span className="truncate text-sm font-semibold text-slate-800 dark:text-gray-100">
          {t('nav.sidebar.profile')}
        </span>
      </header>

      <div className="flex flex-1 flex-col min-h-0 md:flex-row">
        <aside className="shrink-0 border-b border-slate-200 bg-white dark:border-gray-800 dark:bg-gray-900 md:flex md:h-[calc(100dvh-3.5rem)] md:min-h-0 md:w-[13rem] md:flex-col md:border-b-0 md:border-r">
          <nav className="flex gap-1 overflow-x-auto px-1.5 pb-2 pt-3 md:flex md:flex-1 md:flex-col md:overflow-y-auto md:px-2 md:pb-2 md:pt-4">
            {nav.map(({ to, label, icon: Icon }) => (
              <NavLink
                key={to}
                to={to}
                className={({ isActive }) =>
                  cn(
                    'flex shrink-0 items-center gap-2 rounded-lg px-2.5 py-2 text-sm whitespace-nowrap transition-colors md:w-full',
                    isActive || location.pathname.startsWith(`${to}/`)
                      ? 'bg-sky-50 text-sky-700 dark:bg-sky-950/50 dark:text-sky-300'
                      : 'text-slate-600 hover:bg-slate-50 dark:text-gray-300 dark:hover:bg-gray-800',
                  )
                }
              >
                <Icon className="h-4 w-4 shrink-0 opacity-90" />
                {label}
              </NavLink>
            ))}
          </nav>
          <div className="hidden flex-col gap-2 border-t border-slate-200 p-3 dark:border-gray-800 md:flex shrink-0">
            <button
              type="button"
              onClick={() => void logout()}
              className="flex w-full items-center gap-2 rounded-lg px-3 py-2.5 text-left text-sm text-red-600 transition-colors hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-950/30"
            >
              <LogOut className="h-4 w-4 shrink-0" />
              {t('nav.logout')}
            </button>
            <div className="flex items-center gap-2 px-1 text-xs text-slate-500 dark:text-slate-400">
              <span className="h-2 w-2 shrink-0 rounded-full bg-emerald-500" aria-hidden />
              <span className="font-mono font-medium text-slate-700 dark:text-slate-200">v{packageJson.version}</span>
            </div>
          </div>
          <div className="flex items-center justify-between gap-2 border-t border-slate-200 px-3 py-2 dark:border-gray-800 md:hidden">
            <div className="flex min-w-0 items-center gap-2 text-xs text-slate-500 dark:text-slate-400">
              <span className="h-2 w-2 shrink-0 rounded-full bg-emerald-500" aria-hidden />
              <span className="truncate font-mono font-semibold text-slate-700 dark:text-slate-200">
                v{packageJson.version}
              </span>
            </div>
            <button
              type="button"
              onClick={() => void logout()}
              className="inline-flex shrink-0 items-center gap-1.5 rounded-lg px-2 py-1.5 text-xs text-red-600 transition-colors hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-950/30"
            >
              <LogOut className="h-3.5 w-3.5 shrink-0" />
              {t('nav.logout')}
            </button>
          </div>
        </aside>

        <div className="flex min-h-0 min-w-0 flex-1 flex-col bg-white dark:bg-gray-950">
          <main
            className={cn(
              'min-h-0 flex-1',
              location.pathname === '/profile/personal' ? 'overflow-hidden' : 'overflow-y-auto',
            )}
          >
            <div
              className={cn(
                'mx-auto w-full max-w-none px-4 py-4 sm:px-6 md:py-6 lg:px-8',
                location.pathname === '/profile/personal' &&
                  'flex min-h-0 flex-1 flex-col px-4 py-2 sm:px-5 md:py-3 lg:px-6',
                location.pathname === '/profile/user-devices' && 'py-3 md:py-4',
              )}
            >
              <nav
                aria-label="breadcrumb"
                className={cn('text-sm text-slate-600 dark:text-gray-400', breadcrumbMb)}
              >
                <ol className="flex flex-wrap items-center gap-1.5">
                  <li>
                    <Link
                      to="/profile/personal"
                      className="rounded px-0.5 font-medium text-slate-700 transition-colors hover:text-sky-700 dark:text-gray-300 dark:hover:text-sky-400"
                    >
                      {t('nav.sidebar.profile')}
                    </Link>
                  </li>
                  <li className="flex items-center gap-1.5" aria-hidden>
                    <ChevronRight className="h-3.5 w-3.5 shrink-0 opacity-60" />
                    <span className="font-medium text-slate-900 dark:text-gray-100">{currentNavLabel}</span>
                  </li>
                </ol>
              </nav>
              <div
                className={cn(
                  location.pathname === '/profile/personal' && 'flex min-h-0 flex-1 flex-col',
                )}
              >
                <Outlet />
              </div>
            </div>
          </main>
        </div>
      </div>
    </div>
  )
}

export default ProfileLayout
