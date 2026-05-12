import { useEffect, useState, lazy, Suspense } from 'react'
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom'
import ErrorBoundary from '@/components/ErrorBoundary/ErrorBoundary'
import PWAInstaller from '@/components/PWA/PWAInstaller'
import NotificationContainer from '@/components/UI/NotificationContainer'
import GlobalSearch from '@/components/UI/GlobalSearch'
import DevErrorHandler from '@/components/Dev/DevErrorHandler'
import ProtectedRoute from '@/components/Auth/ProtectedRoute'
import AdminLayout from '@/components/Layout/AdminLayout'
import { SidebarProvider } from '@/contexts/SidebarContext'
import { SiteConfigProvider } from '@/contexts/SiteConfigContext'
import { useAuthStore } from '@/stores/authStore'

const Login = lazy(() => import('@/pages/Login'))
const Settings = lazy(() => import('@/pages/Settings'))
const Profile = lazy(() => import('@/pages/Profile'))
const Notifications = lazy(() => import('@/pages/Notifications'))
const Configs = lazy(() => import('@/pages/Configs'))
const LlmChannels = lazy(() => import('@/pages/LlmChannels'))
const LlmAbilities = lazy(() => import('@/pages/LlmAbilities'))
const LlmModelMetas = lazy(() => import('@/pages/LlmModelMetas'))
const LlmUsage = lazy(() => import('@/pages/LlmUsage'))
const LlmTokens = lazy(() => import('@/pages/LlmTokens'))
const SpeechChannels = lazy(() => import('@/pages/SpeechChannels'))
const SpeechUsage = lazy(() => import('@/pages/SpeechUsage'))
const OAuthClients = lazy(() => import('@/pages/OAuthClients'))
const Users = lazy(() => import('@/pages/Users'))
const Permissions = lazy(() => import('@/pages/Permissions'))
const Roles = lazy(() => import('@/pages/Roles'))
const UserAccess = lazy(() => import('@/pages/UserAccess'))
const Assistants = lazy(() => import('@/pages/Assistants'))
const OperationLogs = lazy(() => import('@/pages/OperationLogs'))
const AccountLocks = lazy(() => import('@/pages/AccountLocks'))
const Groups = lazy(() => import('@/pages/Groups'))
const Credentials = lazy(() => import('@/pages/Credentials'))
const JSTemplates = lazy(() => import('@/pages/JSTemplates'))
const Bills = lazy(() => import('@/pages/Bills'))
const VoiceTraining = lazy(() => import('@/pages/VoiceTraining'))
const Workflows = lazy(() => import('@/pages/Workflows'))
const WorkflowPlugins = lazy(() => import('@/pages/WorkflowPlugins'))
const NodePlugins = lazy(() => import('@/pages/NodePlugins'))
const NotificationCenter = lazy(() => import('@/pages/NotificationCenter'))
const NotificationChannels = lazy(() => import('@/pages/NotificationChannels'))
const NotificationChannelEdit = lazy(() => import('@/pages/NotificationChannelEdit'))
const MailTemplates = lazy(() => import('@/pages/MailTemplates'))
const MailTemplateEdit = lazy(() => import('@/pages/MailTemplateEdit'))
const MailLogs = lazy(() => import('@/pages/MailLogs'))
const SMSLogs = lazy(() => import('@/pages/SMSLogs'))
const KnowledgeBases = lazy(() => import('@/pages/KnowledgeBases'))
const Devices = lazy(() => import('@/pages/Devices'))
const ChatData = lazy(() => import('@/pages/ChatData'))

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

  useEffect(() => {
    if (!authHydrated) return
    const token = localStorage.getItem('auth_token')
    if (token && !useAuthStore.getState().isAuthenticated) {
      void refreshUserInfo()
    }
  }, [authHydrated, refreshUserInfo])

  if (!authHydrated) {
    return (
      <div className="min-h-screen flex items-center justify-center text-slate-500">
        正在恢复登录状态…
      </div>
    )
  }

  return (
    <ErrorBoundary>
      <SiteConfigProvider>
        <SidebarProvider>
          <Router>
            <Routes>
              {/* 登录页：独立无布局 */}
              <Route
                path="/login"
                element={
                  isAuthenticated ? (
                    <Navigate to="/users" replace />
                  ) : (
                    <Suspense fallback={<div className="p-8 text-center text-slate-500">页面加载中...</div>}>
                      <Login />
                    </Suspense>
                  )
                }
              />

              {/* 后台主路由：所有页面共享 AdminLayout（持久化 sidebar，修复点击刷新回顶问题） */}
                <Route
                  element={
                    <ProtectedRoute>
                      <AdminLayout />
                    </ProtectedRoute>
                  }
                >
                  <Route path="/settings" element={<Settings />} />
                  <Route path="/profile" element={<Profile />} />
                  <Route path="/notifications" element={<Notifications />} />
                  <Route path="/configs" element={<Configs />} />
                  <Route path="/llm-channels" element={<LlmChannels />} />
                  <Route path="/llm-abilities" element={<LlmAbilities />} />
                  <Route path="/llm-model-metas" element={<LlmModelMetas />} />
                  <Route path="/llm-usage" element={<LlmUsage />} />
                  <Route path="/llm-tokens" element={<LlmTokens />} />
                  <Route path="/speech-channels" element={<SpeechChannels />} />
                  <Route path="/speech-usage" element={<SpeechUsage />} />
                  <Route path="/oauth-clients" element={<OAuthClients />} />
                  <Route path="/users" element={<Users />} />
                  <Route path="/permissions" element={<Permissions />} />
                  <Route path="/roles" element={<Roles />} />
                  <Route path="/user-access" element={<UserAccess />} />
                  <Route path="/assistants" element={<Assistants />} />
                  <Route path="/operation-logs" element={<OperationLogs />} />
                  <Route path="/account-locks" element={<AccountLocks />} />
                  <Route path="/groups" element={<Groups />} />
                  <Route path="/credentials" element={<Credentials />} />
                  <Route path="/js-templates" element={<JSTemplates />} />
                  <Route path="/bills" element={<Bills />} />
                  <Route path="/voice-training" element={<VoiceTraining />} />
                  <Route path="/workflows" element={<Workflows />} />
                  <Route path="/workflow-plugins" element={<WorkflowPlugins />} />
                  <Route path="/node-plugins" element={<NodePlugins />} />
                  <Route path="/notification-center" element={<NotificationCenter />} />
                  <Route path="/notification-channels" element={<NotificationChannels />} />
                  <Route path="/notification-channels/new" element={<NotificationChannelEdit />} />
                  <Route path="/notification-channels/:id/edit" element={<NotificationChannelEdit />} />
                  <Route path="/mail-templates" element={<MailTemplates />} />
                  <Route path="/mail-templates/new" element={<MailTemplateEdit />} />
                  <Route path="/mail-templates/:id/edit" element={<MailTemplateEdit />} />
                  <Route path="/mail-logs" element={<MailLogs />} />
                  <Route path="/sms-logs" element={<SMSLogs />} />
                  <Route path="/knowledge-bases" element={<KnowledgeBases />} />
                  <Route path="/devices" element={<Devices />} />
                  <Route path="/chat-data" element={<ChatData />} />
                </Route>

              <Route path="/" element={<Navigate to="/users" replace />} />
              <Route path="*" element={<Navigate to="/users" replace />} />
            </Routes>

            {/* 全局组件 */}
            <PWAInstaller showOnLoad={true} delay={5000} position="bottom-right" />
            <NotificationContainer />
            <DevErrorHandler />
            <GlobalSearch />
          </Router>
        </SidebarProvider>
      </SiteConfigProvider>
    </ErrorBoundary>
  )
}

export default App
