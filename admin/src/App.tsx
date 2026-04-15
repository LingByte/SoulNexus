import { useEffect, lazy, Suspense } from 'react'
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom'
import ErrorBoundary from '@/components/ErrorBoundary/ErrorBoundary'
import PWAInstaller from '@/components/PWA/PWAInstaller'
import NotificationContainer from '@/components/UI/NotificationContainer'
import GlobalSearch from '@/components/UI/GlobalSearch'
import DevErrorHandler from '@/components/Dev/DevErrorHandler'
import ProtectedRoute from '@/components/Auth/ProtectedRoute'
import { SidebarProvider } from '@/contexts/SidebarContext'
import { SiteConfigProvider } from '@/contexts/SiteConfigContext'
import { useAuthStore } from '@/stores/authStore'

const Login = lazy(() => import('@/pages/Login'))
const Settings = lazy(() => import('@/pages/Settings'))
const Profile = lazy(() => import('@/pages/Profile'))
const Notifications = lazy(() => import('@/pages/Notifications'))
const Configs = lazy(() => import('@/pages/Configs'))
const Users = lazy(() => import('@/pages/Users'))
const OperationLogs = lazy(() => import('@/pages/OperationLogs'))
const LoginHistory = lazy(() => import('@/pages/LoginHistory'))

function App() {
  const { refreshUserInfo, isAuthenticated } = useAuthStore()

  // 初始化时检查用户登录状态
  useEffect(() => {
    const token = localStorage.getItem('auth_token')
    if (token && !isAuthenticated) {
      refreshUserInfo()
    }
  }, [])

  return (
    <ErrorBoundary>
      <SiteConfigProvider>
        <SidebarProvider>
          <Router>
          <div className="min-h-screen bg-[#F7F9FC] dark:bg-slate-950">
          <Suspense fallback={<div className="p-8 text-center text-slate-500">页面加载中...</div>}>
          <Routes>
            {/* 登录页 */}
            <Route
              path="/login"
              element={
                isAuthenticated ? <Navigate to="/wordbooks" replace /> : <Login />
              }
            />

            {/* 受保护的路由 */}
            <Route
              path="/settings"
              element={
                <ProtectedRoute>
                  <Settings />
                </ProtectedRoute>
              }
            />
            <Route
              path="/profile"
              element={
                <ProtectedRoute>
                  <Profile />
                </ProtectedRoute>
              }
            />
            <Route
              path="/notifications"
              element={
                <ProtectedRoute>
                  <Notifications />
                </ProtectedRoute>
              }
            />
            <Route
              path="/configs"
              element={
                <ProtectedRoute>
                  <Configs />
                </ProtectedRoute>
              }
            />
            <Route
              path="/users"
              element={
                <ProtectedRoute>
                  <Users />
                </ProtectedRoute>
              }
            />
            <Route
              path="/operation-logs"
              element={
                <ProtectedRoute>
                  <OperationLogs />
                </ProtectedRoute>
              }
            />
            <Route
              path="/login-history"
              element={
                <ProtectedRoute>
                  <LoginHistory />
                </ProtectedRoute>
              }
            />

            {/* 默认重定向 */}
            <Route path="/" element={<Navigate to="/users" replace />} />
            <Route path="*" element={<Navigate to="/users" replace />} />
          </Routes>
          </Suspense>

          {/* 全局组件 */}
          <PWAInstaller
            showOnLoad={true}
            delay={5000}
            position="bottom-right"
          />
          <NotificationContainer />
          <DevErrorHandler />
          <GlobalSearch />
          </div>
        </Router>
      </SidebarProvider>
      </SiteConfigProvider>
    </ErrorBoundary>
  )
}

export default App
