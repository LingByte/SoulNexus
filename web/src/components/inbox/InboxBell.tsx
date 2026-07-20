import { useCallback, useEffect, useState } from 'react'
import { Bell } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { fetchInboxUnreadCount } from '@/api/inbox'
import { useAuthStore } from '@/stores/authStore'
import { useTranslation } from '@/i18n'
import { Button } from '@/components/ui'

const POLL_MS = 60_000

export default function InboxBell() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const isTenant = useAuthStore((s) => {
    const u = s.user
    if (!s.isAuthenticated || !u) return false
    if (u.principal === 'platform' || u.isPlatformAdmin) return false
    return u.principal === 'tenant' || u.tenantId != null
  })
  const [count, setCount] = useState(0)

  const load = useCallback(async () => {
    if (!isAuthenticated || !isTenant) {
      setCount(0)
      return
    }
    try {
      const res = await fetchInboxUnreadCount()
      if (res.code === 200) setCount(Math.max(0, Number(res.data ?? 0)))
    } catch {
      /* ignore poll errors */
    }
  }, [isAuthenticated, isTenant])

    useEffect(() => {
    void load()
    const id = window.setInterval(() => void load(), POLL_MS)
    const onInboxUpdated = () => void load()
    window.addEventListener('inbox:updated', onInboxUpdated)
    return () => {
      window.clearInterval(id)
      window.removeEventListener('inbox:updated', onInboxUpdated)
    }
  }, [load])

  useEffect(() => {
    const onFocus = () => void load()
    window.addEventListener('focus', onFocus)
    return () => window.removeEventListener('focus', onFocus)
  }, [load])

  if (!isAuthenticated || !isTenant) return null

  const badge = count > 0 ? (count > 99 ? '99+' : String(count)) : null

  return (
    <Button
      type="text"
      size="small"
      title={t('inbox.title')}
      aria-label={badge ? t('inbox.unreadCount', { count }) : t('inbox.title')}
      className="relative !px-2"
      onClick={() => navigate('/profile/inbox')}
    >
      <Bell size={18} strokeWidth={1.75} />
      {badge && (
        <span className="absolute -right-0.5 -top-0.5 flex h-[18px] min-w-[18px] items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-semibold leading-none text-white">
          {badge}
        </span>
      )}
    </Button>
  )
}
