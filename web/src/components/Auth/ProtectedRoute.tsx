import { ReactNode, useEffect, useState } from 'react'
import { useLocation } from 'react-router-dom'
import LoadingAnimation from '@/components/Animations/LoadingAnimation'
import { useAuthStore } from '@/stores/authStore'
import { beginSSOLogin } from '@/utils/sso'
import {
  getAccountDeletionRevokePageURL,
  isAccountDeletionRevokeStandalonePage,
} from '@/config/apiConfig'

interface ProtectedRouteProps {
  children: ReactNode
  requireAuth?: boolean
}

function isDeletionPending(user: { accountDeletionEffectiveAt?: string | null } | null): boolean {
  if (!user?.accountDeletionEffectiveAt) return false
  const t = new Date(user.accountDeletionEffectiveAt).getTime()
  return !Number.isNaN(t) && t > Date.now()
}

function AuthFullscreenLoading({ message }: { message: string }) {
  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex flex-col items-center justify-center px-4">
      <LoadingAnimation type="spinner" size="lg" className="mb-4" />
      <p className="text-sm text-gray-500 dark:text-gray-400 text-center">{message}</p>
    </div>
  )
}

const ProtectedRoute = ({ children, requireAuth = true }: ProtectedRouteProps) => {
  const location = useLocation()
  const { isAuthenticated, isLoading, isLoggingOut, user, refreshUserInfo } = useAuthStore()
  const [authGateReady, setAuthGateReady] = useState(() => !requireAuth || !isAuthenticated)

  useEffect(() => {
    if (isLoading || isLoggingOut) return
    if (requireAuth && !isAuthenticated) {
      const currentPath = location.pathname + location.search
      beginSSOLogin(currentPath)
    }
  }, [isAuthenticated, isLoading, isLoggingOut, requireAuth, location])

  useEffect(() => {
    if (!requireAuth || !isAuthenticated) {
      setAuthGateReady(true)
      return
    }
    setAuthGateReady(false)
    let cancelled = false
    ;(async () => {
      try {
        await refreshUserInfo()
      } catch {
        //
      } finally {
        if (!cancelled) setAuthGateReady(true)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [requireAuth, isAuthenticated, refreshUserInfo])

  useEffect(() => {
    if (!requireAuth || !isAuthenticated || !user || !authGateReady) return
    if (!isDeletionPending(user)) return
    if (isAccountDeletionRevokeStandalonePage()) return
    window.location.replace(getAccountDeletionRevokePageURL(user.email))
  }, [requireAuth, isAuthenticated, user, authGateReady])

  if (isLoading) {
    return <AuthFullscreenLoading message="加载中…" />
  }

  if (requireAuth && !isAuthenticated) {
    return null
  }

  if (requireAuth && isAuthenticated && !authGateReady) {
    return <AuthFullscreenLoading message="加载中…" />
  }

  if (requireAuth && isAuthenticated && isDeletionPending(user)) {
    return (
      <div className="min-h-screen flex items-center justify-center text-gray-600 dark:text-gray-300">
        {isAccountDeletionRevokeStandalonePage()
          ? '账号处于注销冷静期，请在本页完成撤销验证。'
          : '账号处于注销冷静期，正在跳转至用户服务撤销页…'}
      </div>
    )
  }

  return <>{children}</>
}

export default ProtectedRoute
