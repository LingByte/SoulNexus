import { Outlet, NavLink, useLocation } from 'react-router-dom'
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
} from 'lucide-react'
import { useAuthStore } from '@/stores/authStore'
import { cn } from '@/utils/cn'
import packageJson from '../../../package.json'

const nav = [
  { to: '/profile/personal', label: '个人信息', icon: User },
  { to: '/profile/teams', label: '团队管理', icon: Users },
  { to: '/profile/activity', label: '使用记录', icon: Clock },
  { to: '/profile/billing', label: '账单与发票', icon: Receipt },
  { to: '/profile/user-devices', label: '用户设备', icon: Smartphone },
  { to: '/profile/integrations', label: '第三方账号', icon: Settings },
  { to: '/profile/credential', label: 'API 密钥', icon: Key },
  { to: '/profile/llm-tokens', label: 'LLM Token', icon: Key },
  { to: '/profile/notifications', label: '通知', icon: Bell },
  { to: '/profile/locale', label: '语言与时区', icon: Globe2 },
  { to: '/profile/security', label: '账户与安全', icon: Shield },
] as const

const ProfileLayout = () => {
  const { logout } = useAuthStore()
  const location = useLocation()

  return (
    <div className="min-h-screen md:h-[calc(100dvh)] md:min-h-0 bg-slate-50 dark:bg-gray-950 flex flex-col">
      <div className="flex flex-1 flex-col md:flex-row min-h-0 md:min-h-0">
        <aside className="shrink-0 border-b md:border-b-0 md:border-r border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 md:w-[12rem] md:h-screen md:sticky md:top-0 flex flex-col md:min-h-0">
          <nav className="flex md:flex-col gap-1 px-1.5 pb-2 md:pt-6 md:pb-2 pt-4 overflow-x-auto md:overflow-y-auto md:flex-1 md:min-h-0">
            {nav.map(({ to, label, icon: Icon }) => (
              <NavLink
                key={to}
                to={to}
                className={({ isActive }) =>
                  cn(
                    'flex items-center gap-1.5 px-2 py-2 rounded-lg text-sm whitespace-nowrap md:w-full transition-colors',
                    isActive || location.pathname.startsWith(`${to}/`)
                      ? 'bg-sky-50 text-sky-700 dark:bg-sky-950/50 dark:text-sky-300'
                      : 'text-slate-600 hover:bg-slate-50 dark:text-gray-300 dark:hover:bg-gray-800'
                  )
                }
              >
                <Icon className="w-3.5 h-3.5 shrink-0" />
                {label}
              </NavLink>
            ))}
          </nav>
          <div className="hidden md:flex flex-col gap-2 p-3 border-t border-slate-200 dark:border-gray-800 shrink-0">
            <button
              type="button"
              onClick={() => void logout()}
              className="flex items-center gap-2 px-3 py-2.5 rounded-lg text-sm w-full text-left text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-950/30 transition-colors"
            >
              <LogOut className="w-4 h-4 shrink-0" />
              登出
            </button>
            <div className="flex items-center gap-2 px-1 text-xs text-slate-500 dark:text-slate-400">
              <span className="w-2 h-2 rounded-full bg-emerald-500 shrink-0" aria-hidden />
              <span className="font-mono font-medium text-slate-700 dark:text-slate-200">
                v{packageJson.version}
              </span>
            </div>
          </div>
          <div className="flex md:hidden items-center justify-between gap-2 px-3 py-2 border-t border-slate-200 dark:border-gray-800">
            <div className="flex items-center gap-2 text-xs text-slate-500 dark:text-slate-400 min-w-0">
              <span className="w-2 h-2 rounded-full bg-emerald-500 shrink-0" aria-hidden />
              <span className="font-mono font-semibold truncate text-slate-700 dark:text-slate-200">
                v{packageJson.version}
              </span>
            </div>
            <button
              type="button"
              onClick={() => void logout()}
              className="flex items-center gap-1.5 px-2 py-1.5 rounded-lg text-xs shrink-0 text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-950/30 transition-colors"
            >
              <LogOut className="w-3.5 h-3.5 shrink-0" />
              登出
            </button>
          </div>
        </aside>

        <div className="flex-1 min-w-0 flex flex-col bg-white dark:bg-gray-950 min-h-0">
          <main className="flex-1 overflow-y-auto min-h-0">
            <div className="mx-auto w-full max-w-none px-4 sm:px-6 lg:px-8 py-4 md:py-6">
              <Outlet />
            </div>
          </main>
        </div>
      </div>
    </div>
  )
}

export default ProfileLayout
