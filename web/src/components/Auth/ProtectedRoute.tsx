import { ReactNode, useEffect } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '@/stores/authStore'
import { showAlert } from '@/utils/notification'

interface ProtectedRouteProps {
  children: ReactNode
  requireAuth?: boolean
}

const ProtectedRoute = ({ children, requireAuth = true }: ProtectedRouteProps) => {
  const { isAuthenticated, isLoading } = useAuthStore()
  const navigate = useNavigate()
  const location = useLocation()

  useEffect(() => {
    // 如果正在加载，等待加载完成
    if (isLoading) return

    // 如果需要认证但用户未登录
    if (requireAuth && !isAuthenticated) {
      // 保存当前路径，登录后可以跳转回来
      const currentPath = location.pathname + location.search
      if (currentPath !== '/') {
        localStorage.setItem('redirectAfterLogin', currentPath)
      }
      
      // 跳转到首页并显示登录窗口
      navigate('/', { replace: true })
      showAlert('请先登录后再访问此页面', 'warning')
      
      // 延迟打开登录窗口，确保页面已经跳转
      setTimeout(() => {
        // 触发登录窗口打开事件
        const event = new CustomEvent('openAuthModal')
        window.dispatchEvent(event)
      }, 100)
    }
  }, [isAuthenticated, isLoading, requireAuth, navigate, location])

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
