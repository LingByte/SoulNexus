import { ReactNode, useEffect } from 'react'
import { useLocation } from 'react-router-dom'
import { useAuthStore } from '@/stores/authStore'
import { beginSSOLogin } from '@/utils/sso'

interface ProtectedRouteProps {
  children: ReactNode
  requireAuth?: boolean
}

const ProtectedRoute = ({ children, requireAuth = true }: ProtectedRouteProps) => {
  const { isAuthenticated, isLoading } = useAuthStore()
  const location = useLocation()

  useEffect(() => {
    // 如果正在加载，等待加载完成
    if (isLoading) return

    // 如果需要认证但用户未登录
    if (requireAuth && !isAuthenticated) {
      const currentPath = location.pathname + location.search
      beginSSOLogin(currentPath)
    }
  }, [isAuthenticated, isLoading, requireAuth, location])

  // 如果正在加载，显示加载状态
  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    )
  }

  // 如果需要认证但用户未登录，不渲染内容
  if (requireAuth && !isAuthenticated) {
    return null
  }

  return <>{children}</>
}

export default ProtectedRoute
