import { useEffect, useRef, useState } from 'react'
import { LayoutDashboard, LogOut, User } from 'lucide-react'
import { useTranslation } from '@/i18n'
import type { User as AuthUser } from '@/stores/authStore'
import { cn } from '@/utils/cn'

type LandingUserMenuProps = {
  user: AuthUser
  onConsole: () => void
  onLogout: () => void
  className?: string
}

function avatarSrc(user: AuthUser): string {
  const url = String(user.avatarUrl || user.avatar || '').trim()
  if (url) return url
  const name = encodeURIComponent(String(user.displayName || user.username || user.email || 'U').slice(0, 2))
  return `https://ui-avatars.com/api/?name=${name}&background=7c3aed&color=fff&size=128`
}

export default function LandingUserMenu({ user, onConsole, onLogout, className }: LandingUserMenuProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    return () => document.removeEventListener('mousedown', onDoc)
  }, [open])

  const label = String(user.displayName || user.username || user.email || t('nav.me'))
  const email = String(user.email || '')

  return (
    <div ref={ref} className={cn('relative', className)}>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex items-center gap-2 rounded-full border border-[hsl(var(--border))] bg-[hsl(var(--background)/0.9)] py-1 pl-1 pr-3 shadow-sm transition hover:border-violet-400/50 hover:shadow-md"
        aria-expanded={open}
      >
        <img src={avatarSrc(user)} alt="" className="h-8 w-8 rounded-full object-cover ring-2 ring-violet-100 dark:ring-violet-900" />
        <span className="hidden max-w-[120px] truncate text-sm font-medium lg:inline">{label}</span>
      </button>
      {open ? (
        <div className="absolute right-0 top-[calc(100%+8px)] z-50 w-56 overflow-hidden rounded-xl border border-[hsl(var(--border))] bg-[hsl(var(--popover))] shadow-xl">
          <div className="border-b border-[hsl(var(--border))] px-4 py-3">
            <p className="truncate text-sm font-semibold">{label}</p>
            {email ? <p className="truncate text-xs text-[hsl(var(--muted-foreground))]">{email}</p> : null}
          </div>
          <div className="p-1.5">
            <button
              type="button"
              className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm hover:bg-[hsl(var(--muted))]"
              onClick={() => {
                setOpen(false)
                onConsole()
              }}
            >
              <LayoutDashboard className="h-4 w-4 text-violet-600" />
              {t('landing.ctaConsole')}
            </button>
            <button
              type="button"
              className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-950/30"
              onClick={() => {
                setOpen(false)
                onLogout()
              }}
            >
              <LogOut className="h-4 w-4" />
              {t('profile.logout')}
            </button>
          </div>
        </div>
      ) : null}
    </div>
  )
}

/** Compact login chip when guest */
export function LandingLoginChip({ onLogin }: { onLogin: () => void }) {
  const { t } = useTranslation()
  return (
    <button
      type="button"
      onClick={onLogin}
      className="inline-flex h-9 items-center gap-2 rounded-full bg-gradient-to-r from-violet-600 to-purple-600 px-4 text-sm font-medium shadow-md shadow-violet-500/25 transition hover:brightness-110"
    >
      <User className="h-4 w-4" />
      {t('auth.login')}
    </button>
  )
}
