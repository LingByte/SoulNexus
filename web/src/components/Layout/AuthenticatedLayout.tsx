import { Suspense, useEffect, useState } from 'react'
import { Outlet } from 'react-router-dom'
import { Loading } from '@/components/ui/loading'
import Sidebar from '@/components/layout/Sidebar'
import { AppShellProvider } from '@/contexts/AppShellContext'
import { useSidebar } from '@/contexts/SidebarContext'
import { useTranslation } from '@/i18n'

function MainContentLoading() {
  const { t } = useTranslation()
  return (
    <div className="flex min-h-[50vh] items-center justify-center py-16">
      <Loading block tip={t('layout.loadingPage')} />
    </div>
  )
}

export default function AuthenticatedLayout() {
  const { effectiveSidebarWidth } = useSidebar()
  const [isLg, setIsLg] = useState(() =>
    typeof window !== 'undefined' ? window.matchMedia('(min-width: 1024px)').matches : false
  )

  useEffect(() => {
    const q = window.matchMedia('(min-width: 1024px)')
    const sync = () => setIsLg(q.matches)
    sync()
    q.addEventListener('change', sync)
    return () => q.removeEventListener('change', sync)
  }, [])

  const marginLeft = isLg ? effectiveSidebarWidth : 0

  return (
    <AppShellProvider>
      <div className="min-h-screen bg-background text-foreground">
        <Sidebar />
        <div
          className="min-h-screen bg-background ling-main-with-sidebar"
          data-sidebar-offset
          style={{
            marginLeft,
            transition: 'margin-left 0.2s ease',
          }}
        >
          <Suspense fallback={<MainContentLoading />}>
            <Outlet />
          </Suspense>
        </div>
      </div>
    </AppShellProvider>
  )
}
