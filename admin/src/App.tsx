import { useEffect, useState, lazy, Suspense } from 'react'
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
const OAuthClients = lazy(() => import('@/pages/OAuthClients'))
const Users = lazy(() => import('@/pages/Users'))
const Assistants = lazy(() => import('@/pages/Assistants'))
const OperationLogs = lazy(() => import('@/pages/OperationLogs'))
const AccountLocks = lazy(() => import('@/pages/AccountLocks'))
const Groups = lazy(() => import('@/pages/Groups'))
const Credentials = lazy(() => import('@/pages/Credentials'))
const JSTemplates = lazy(() => import('@/pages/JSTemplates'))
const Bills = lazy(() => import('@/pages/Bills'))
const VoiceTraining = lazy(() => import('@/pages/VoiceTraining'))
const MCPServers = lazy(() => import('@/pages/MCPServers'))
const MCPMarketplace = lazy(() => import('@/pages/MCPMarketplace'))
const Workflows = lazy(() => import('@/pages/Workflows'))
const WorkflowPlugins = lazy(() => import('@/pages/WorkflowPlugins'))
const NodePlugins = lazy(() => import('@/pages/NodePlugins'))
const NotificationCenter = lazy(() => import('@/pages/NotificationCenter'))
const AlertCenter = lazy(() => import('@/pages/AlertCenter'))
const KnowledgeBases = lazy(() => import('@/pages/KnowledgeBases'))
const Devices = lazy(() => import('@/pages/Devices'))
const ChatData = lazy(() => import('@/pages/ChatData'))
const ServiceHealth = lazy(() => import('@/pages/ServiceHealth'))

function App() {
  const { refreshUserInfo, isAuthenticated } = useAuthStore()
  const [authHydrated, setAuthHydrated] = useState(() => {
    const p = (useAuthStore as unknown as { persist?: { hasHydrated?: () => boolean } }).persist
    if (!p?.hasHydrated) return true
    return p.hasHydrated()
  })

  useEffect(() => {
    const p = (useAuthStore as unknown as {
      persist?: {
        hasHydrated?: () => boolean
        onFinishHydration?: (fn: () => void) => () => void
      }
    }).persist
    if (!p?.onFinishHydration) {
      setAuthHydrated(true)
      return
    }
    if (p.hasHydrated?.()) {
      setAuthHydrated(true)
      return
    }
    return p.onFinishHydration(() => setAuthHydrated(true))
  }, [])

  // 持久化恢复完成后再尝试用本地 token 拉取用户信息，避免与 persist 写入竞态
  useEffect(() => {
    if (!authHydrated) return
    const token = localStorage.getItem('auth_token')
    if (token && !useAuthStore.getState().isAuthenticated) {
      void refreshUserInfo()
    }
  }, [authHydrated, refreshUserInfo])

  if (!authHydrated) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[#F7F9FC] dark:bg-slate-950 text-slate-500">
        正在恢复登录状态…
      </div>
    )
  }

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
                isAuthenticated ? <Navigate to="/users" replace /> : <Login />
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
              path="/oauth-clients"
              element={
                <ProtectedRoute>
                  <OAuthClients />
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
            <Route path="/assistants" element={<ProtectedRoute><Assistants /></ProtectedRoute>} />
            <Route
              path="/operation-logs"
              element={
                <ProtectedRoute>
                  <OperationLogs />
                </ProtectedRoute>
              }
            />
            <Route
              path="/account-locks"
              element={
                <ProtectedRoute>
                  <AccountLocks />
                </ProtectedRoute>
              }
            />
            <Route path="/groups" element={<ProtectedRoute><Groups /></ProtectedRoute>} />
            <Route path="/credentials" element={<ProtectedRoute><Credentials /></ProtectedRoute>} />
            <Route path="/js-templates" element={<ProtectedRoute><JSTemplates /></ProtectedRoute>} />
            <Route path="/bills" element={<ProtectedRoute><Bills /></ProtectedRoute>} />
            <Route path="/voice-training" element={<ProtectedRoute><VoiceTraining /></ProtectedRoute>} />
            <Route path="/mcp-servers" element={<ProtectedRoute><MCPServers /></ProtectedRoute>} />
            <Route path="/mcp-marketplace" element={<ProtectedRoute><MCPMarketplace /></ProtectedRoute>} />
            <Route path="/workflows" element={<ProtectedRoute><Workflows /></ProtectedRoute>} />
            <Route path="/workflow-plugins" element={<ProtectedRoute><WorkflowPlugins /></ProtectedRoute>} />
            <Route path="/node-plugins" element={<ProtectedRoute><NodePlugins /></ProtectedRoute>} />
            <Route path="/notification-center" element={<ProtectedRoute><NotificationCenter /></ProtectedRoute>} />
            <Route path="/alerts" element={<ProtectedRoute><AlertCenter /></ProtectedRoute>} />
            <Route path="/knowledge-bases" element={<ProtectedRoute><KnowledgeBases /></ProtectedRoute>} />
            <Route path="/devices" element={<ProtectedRoute><Devices /></ProtectedRoute>} />
            <Route path="/chat-data" element={<ProtectedRoute><ChatData /></ProtectedRoute>} />
            <Route path="/service-health" element={<ProtectedRoute><ServiceHealth /></ProtectedRoute>} />

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
