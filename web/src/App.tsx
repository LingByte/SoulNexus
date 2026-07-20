import { Suspense } from 'react'
import { Loading } from '@/components/ui/loading'
import { createBrowserRouter, RouterProvider, useLocation } from 'react-router-dom'
import ErrorBoundary from '@/components/error-boundary/ErrorBoundary'
import PWAInstaller from '@/components/PWA/PWAInstaller'
import DevErrorHandler from '@/components/dev/DevErrorHandler'
import { SidebarProvider } from '@/contexts/SidebarContext'
import { SiteConfigProvider } from '@/contexts/siteConfig'
import { AppRoutes } from '@/AppRoutes'
import { useTranslation } from '@/i18n'

function AuthPageLoading() {
  const { t } = useTranslation()
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-background">
      <Loading block tip={t('layout.loadingPage')} />
    </div>
  )
}

function AppShell() {
  const location = useLocation()
  const hidePwa = ['/', '/login', '/register'].includes(location.pathname)

  return (
    <div className="min-h-screen bg-background text-foreground">
      <Suspense fallback={<AuthPageLoading />}>
        <AppRoutes />
      </Suspense>
      {!hidePwa ? <PWAInstaller showOnLoad delay={5000} position="bottom-right" /> : null}
      <DevErrorHandler />
    </div>
  )
}

/** Data router so useBlocker works for in-app navigation guards. */
const router = createBrowserRouter([
  {
    path: '*',
    element: <AppShell />,
  },
])

function App() {
  return (
    <SiteConfigProvider>
      <ErrorBoundary>
        <SidebarProvider>
          <RouterProvider router={router} />
        </SidebarProvider>
      </ErrorBoundary>
    </SiteConfigProvider>
  )
}

export default App
