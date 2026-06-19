import { ReactNode, useEffect, useState } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '@/stores/authStore'

interface ProtectedRouteProps {
  children: ReactNode
}

const ProtectedRoute = ({ children }: ProtectedRouteProps) => {
  const { isAuthenticated, token, refreshUserInfo } = useAuthStore()
  const [hydrated, setHydrated] = useState(false)
  const location = useLocation()

  // Wait for zustand persist hydration to complete
  useEffect(() => {
    // zustand persist middleware hydration is async; give it a tick
    const t = setTimeout(() => setHydrated(true), 0)
    return () => clearTimeout(t)
  }, [])

  // Refresh user info on mount if we have a token
  useEffect(() => {
    if (isAuthenticated && token) {
      refreshUserInfo().catch(() => {
        // Token refresh failed — handled by axios interceptor (401 → clearUser)
      })
    }
  }, [isAuthenticated, token, refreshUserInfo])

  if (!hydrated) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500" />
      </div>
    )
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  return <>{children}</>
}

export default ProtectedRoute
