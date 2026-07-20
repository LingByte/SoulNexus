import { lazy } from 'react'
import { Routes, Route, Navigate, Outlet } from 'react-router-dom'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout'
import { useAuthStore } from '@/stores/authStore'

const LandingPage = lazy(() => import('@/pages/LandingPage'))
const ComponentsDemo = lazy(() => import('@/pages/ComponentsDemo'))
const OverviewDashboard = lazy(() => import('@/pages/OverviewDashboard'))
const TenantBills = lazy(() => import('@/pages/TenantBills'))
const UsageMetrics = lazy(() => import('@/pages/UsageMetrics'))
const AssistantManager = lazy(() => import('@/pages/AssistantManager'))
const AssistantManagerNew = lazy(() => import('@/pages/AssistantManagerNew'))
const AssistantManagerCreate = lazy(() => import('@/pages/AssistantManagerCreate'))
const AssistantManagerEdit = lazy(() => import('@/pages/AssistantManagerEdit'))
const AssistantManagerDebug = lazy(() => import('@/pages/AssistantManagerDebug'))
const AssistantTools = lazy(() => import('@/pages/AssistantTools'))
const McpMarket = lazy(() => import('@/pages/McpMarket'))
const VoiceCloneManager = lazy(() => import('@/pages/VoiceCloneManager'))
const VoiceprintManager = lazy(() => import('@/pages/VoiceprintManager'))
const JSTemplateManager = lazy(() => import('@/pages/JSTemplateManager'))
const JSTemplateNew = lazy(() => import('@/pages/JSTemplateNew'))
const JSTemplateEdit = lazy(() => import('@/pages/JSTemplateEdit'))
const WorkflowManager = lazy(() => import('@/pages/WorkflowManager'))
const WorkflowEditorPage = lazy(() => import('@/pages/WorkflowEditorPage'))
const WorkflowPluginMarket = lazy(() => import('@/pages/WorkflowPluginMarket'))
const PlatformPluginMarket = lazy(() => import('@/pages/platform/PlatformPluginMarket'))
const PlatformMcpMarket = lazy(() => import('@/pages/platform/PlatformMcpMarket'))
const TenantLogin = lazy(() => import('@/pages/TenantLogin'))
const TenantRegister = lazy(() => import('@/pages/TenantRegister'))
const AccountDeletionRevoke = lazy(() => import('@/pages/AccountDeletionRevoke'))
const Profile = lazy(() => import('@/pages/Profile'))
const TenantRolePermissions = lazy(() => import('@/pages/TenantRolePermissions'))
const TenantManagement = lazy(() => import('@/pages/TenantManagement'))
const PlatformAdmins = lazy(() => import('@/pages/PlatformAdmins'))
const PlatformVoiceManagement = lazy(() => import('@/pages/PlatformVoiceManagement'))
const PlatformVoiceprintManagement = lazy(() => import('@/pages/PlatformVoiceprintManagement'))
const SystemConfigs = lazy(() => import('@/pages/SystemConfigs'))
const TenantAiConfig = lazy(() => import('@/pages/TenantAiConfig'))
const TenantMembers = lazy(() => import('@/pages/TenantMembers'))
const BillingPlan = lazy(() => import('@/pages/BillingPlan'))
const KnowledgeNamespaces = lazy(() => import('@/pages/KnowledgeNamespaces'))
const KnowledgeBaseDetail = lazy(() => import('@/pages/KnowledgeBaseDetail'))
const KnowledgeDocumentEdit = lazy(() => import('@/pages/KnowledgeDocumentEdit'))
const KnowledgeDocumentChunks = lazy(() => import('@/pages/KnowledgeDocumentChunks'))
const NotFound = lazy(() => import('@/pages/NotFound'))
const NotificationChannels = lazy(() => import('@/pages/platform/NotificationChannels'))
const NotificationChannelEdit = lazy(() => import('@/pages/platform/NotificationChannelEdit'))
const MailTemplates = lazy(() => import('@/pages/platform/MailTemplates'))
const NluModelsPage = lazy(() => import('@/pages/NluModelsPage'))
const NluModelDetailPage = lazy(() => import('@/pages/NluModelDetailPage'))
const PlatformNluModels = lazy(() => import('@/pages/platform/PlatformNluModels'))
const MailTemplateEdit = lazy(() => import('@/pages/platform/MailTemplateEdit'))
const MailLogs = lazy(() => import('@/pages/platform/MailLogs'))
const AIInvocationLogs = lazy(() => import('@/pages/AIInvocationLogs'))
const ExecutionTasks = lazy(() => import('@/pages/platform/ExecutionTasks'))
const SMSLogs = lazy(() => import('@/pages/platform/SMSLogs'))
const SystemStatus = lazy(() => import('@/pages/platform/SystemStatus'))

function RequireAuth({ children }: { children: JSX.Element }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const token = useAuthStore((s) => s.token)
  if (!isAuthenticated || !token) {
    return <Navigate to="/login" replace />
  }
  return children
}

function hasPermission(codes: readonly string[] | undefined, code: string): boolean {
  const list = codes ?? []
  return list.includes('*') || list.includes(code)
}

function RequireBillingView({ children }: { children: JSX.Element }) {
  const user = useAuthStore((s) => s.user)
  const isPlatform = Boolean(user?.isPlatformAdmin || user?.principal === 'platform')
  if (isPlatform) return children
  const codes = user?.permissionCodes as string[] | undefined
  const canView =
    hasPermission(codes, 'menu.acc.billing') || hasPermission(codes, 'api.billing.read')
  if (!canView) {
    return <Navigate to="/overview" replace />
  }
  return children
}

function RequireUsageMetricsView({ children }: { children: JSX.Element }) {
  const user = useAuthStore((s) => s.user)
  const isPlatform = Boolean(user?.isPlatformAdmin || user?.principal === 'platform')
  if (isPlatform) {
    return <Navigate to="/overview" replace />
  }
  const codes = user?.permissionCodes as string[] | undefined
  const canView =
    hasPermission(codes, 'menu.acc.usage_metrics') || hasPermission(codes, 'api.billing.read')
  if (!canView) {
    return <Navigate to="/overview" replace />
  }
  return children
}
function RequirePlatform({ children }: { children: JSX.Element }) {
  const user = useAuthStore((s) => s.user)
  const isPlatform = Boolean(user?.isPlatformAdmin || user?.principal === 'platform')
  if (!isPlatform) {
    return <Navigate to="/overview" replace />
  }
  return children
}

function RequireTenant({ children }: { children: JSX.Element }) {
  const user = useAuthStore((s) => s.user)
  const isPlatform = Boolean(user?.isPlatformAdmin || user?.principal === 'platform')
  if (isPlatform) {
    return <Navigate to="/tenant-management" replace />
  }
  return children
}

function HomeRedirect() {
  const user = useAuthStore((s) => s.user)
  const isPlatform = Boolean(user?.isPlatformAdmin || user?.principal === 'platform')
  return <Navigate to={isPlatform ? '/tenant-management' : '/overview'} replace />
}

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<LandingPage />} />
      <Route path="/components-demo" element={<ComponentsDemo />} />
      <Route path="/login" element={<TenantLogin />} />
      <Route path="/register" element={<TenantRegister />} />
      <Route path="/account/deletion/revoke" element={<AccountDeletionRevoke />} />
      <Route path="/tenant/login" element={<Navigate to="/login" replace />} />
      <Route path="/tenant/register" element={<Navigate to="/register" replace />} />

      <Route
        element={
          <RequireAuth>
            <AuthenticatedLayout />
          </RequireAuth>
        }
      >
        <Route path="/app" element={<HomeRedirect />} />
        <Route path="/overview" element={<OverviewDashboard />} />
        <Route path="/profile" element={<Navigate to="/profile/info" replace />} />
        <Route path="/profile/:section" element={<Profile />} />
        <Route path="/inbox" element={<Navigate to="/profile/inbox" replace />} />
        <Route
          path="/billing"
          element={
            <RequireBillingView>
              <TenantBills />
            </RequireBillingView>
          }
        />
        <Route
          path="/usage-metrics"
          element={
            <RequireUsageMetricsView>
              <UsageMetrics />
            </RequireUsageMetricsView>
          }
        />
        <Route path="/operation-logs" element={<Navigate to="/profile/logs" replace />} />
        <Route path="/ai-invocations" element={<Navigate to="/profile/ai-invocations" replace />} />
        <Route
          path="/knowledge-base"
          element={
            <RequireTenant>
              <KnowledgeNamespaces />
            </RequireTenant>
          }
        />
        <Route
          path="/knowledge-base/:nsId"
          element={
            <RequireTenant>
              <KnowledgeBaseDetail />
            </RequireTenant>
          }
        />
        <Route
          path="/knowledge-base/:nsId/documents/:docId/edit"
          element={
            <RequireTenant>
              <KnowledgeDocumentEdit />
            </RequireTenant>
          }
        />
        <Route
          path="/knowledge-base/:nsId/documents/:docId/chunks"
          element={
            <RequireTenant>
              <KnowledgeDocumentChunks />
            </RequireTenant>
          }
        />
        <Route
          path="/voice-clone-manager"
          element={
            <RequireTenant>
              <VoiceCloneManager />
            </RequireTenant>
          }
        />
        <Route
          path="/voiceprint-manager"
          element={
            <RequireTenant>
              <VoiceprintManager />
            </RequireTenant>
          }
        />
        <Route
          path="/js-templates/new"
          element={
            <RequireTenant>
              <JSTemplateNew />
            </RequireTenant>
          }
        />
        <Route
          path="/js-templates/:id/edit"
          element={
            <RequireTenant>
              <JSTemplateEdit />
            </RequireTenant>
          }
        />
        <Route
          path="/js-templates"
          element={
            <RequireTenant>
              <JSTemplateManager />
            </RequireTenant>
          }
        />
        <Route
          path="/workflows"
          element={
            <RequireTenant>
              <WorkflowManager />
            </RequireTenant>
          }
        />
        <Route
          path="/workflows/:id"
          element={
            <RequireTenant>
              <WorkflowEditorPage />
            </RequireTenant>
          }
        />
        <Route
          path="/plugin-market"
          element={
            <RequireTenant>
              <WorkflowPluginMarket />
            </RequireTenant>
          }
        />
        <Route
          path="/assistant-manager/new"
          element={
            <RequireTenant>
              <AssistantManagerNew />
            </RequireTenant>
          }
        />
        <Route
          path="/assistant-manager/create"
          element={
            <RequireTenant>
              <AssistantManagerCreate />
            </RequireTenant>
          }
        />
        <Route
          path="/assistant-manager/:id/debug"
          element={
            <RequireTenant>
              <AssistantManagerDebug />
            </RequireTenant>
          }
        />
        <Route
          path="/assistant-manager/:id/edit"
          element={
            <RequireTenant>
              <AssistantManagerEdit />
            </RequireTenant>
          }
        />
        <Route
          path="/assistant-manager"
          element={
            <RequireTenant>
              <AssistantManager />
            </RequireTenant>
          }
        />
        <Route
          path="/mcp"
          element={
            <RequireTenant>
              <AssistantTools />
            </RequireTenant>
          }
        />
        <Route
          path="/assistant-tools"
          element={<Navigate to="/mcp" replace />}
        />
        <Route
          path="/mcp-market"
          element={
            <RequireTenant>
              <McpMarket />
            </RequireTenant>
          }
        />
        <Route path="/nlu-models" element={<Outlet />}>
          <Route
            index
            element={
              <RequireTenant>
                <NluModelsPage />
              </RequireTenant>
            }
          />
          <Route
            path=":id"
            element={
              <RequireTenant>
                <NluModelDetailPage />
              </RequireTenant>
            }
          />
        </Route>
        <Route
          path="/tenant-management"
          element={
            <RequirePlatform>
              <TenantManagement />
            </RequirePlatform>
          }
        />
        <Route
          path="/platform-admins"
          element={
            <RequirePlatform>
              <PlatformAdmins />
            </RequirePlatform>
          }
        />
        <Route
          path="/platform/voice-management"
          element={
            <RequirePlatform>
              <PlatformVoiceManagement />
            </RequirePlatform>
          }
        />
        <Route
          path="/platform/voiceprint-management"
          element={
            <RequirePlatform>
              <PlatformVoiceprintManagement />
            </RequirePlatform>
          }
        />
        <Route
          path="/system-configs"
          element={
            <RequirePlatform>
              <SystemConfigs />
            </RequirePlatform>
          }
        />
        <Route path="/platform/notification-channels" element={<RequirePlatform><NotificationChannels /></RequirePlatform>} />
        <Route path="/platform/notification-channels/new" element={<RequirePlatform><NotificationChannelEdit /></RequirePlatform>} />
        <Route path="/platform/notification-channels/:id/edit" element={<RequirePlatform><NotificationChannelEdit /></RequirePlatform>} />
        <Route path="/platform/mail-templates" element={<RequirePlatform><MailTemplates /></RequirePlatform>} />
        <Route path="/platform/nlu-models" element={<RequirePlatform><PlatformNluModels /></RequirePlatform>} />
        <Route path="/platform/nlu-lab" element={<Navigate to="/platform/nlu-models" replace />} />
        <Route path="/platform/mail-templates/new" element={<RequirePlatform><MailTemplateEdit /></RequirePlatform>} />
        <Route path="/platform/mail-templates/:id/edit" element={<RequirePlatform><MailTemplateEdit /></RequirePlatform>} />
        <Route path="/platform/mail-logs" element={<RequirePlatform><MailLogs /></RequirePlatform>} />
        <Route path="/platform/ai-invocations" element={<RequirePlatform><AIInvocationLogs platform /></RequirePlatform>} />
        <Route path="/platform/sms-logs" element={<RequirePlatform><SMSLogs /></RequirePlatform>} />
        <Route path="/platform/execution-tasks" element={<RequirePlatform><ExecutionTasks /></RequirePlatform>} />
        <Route path="/platform/plugin-market" element={<RequirePlatform><PlatformPluginMarket /></RequirePlatform>} />
        <Route path="/platform/mcp-market" element={<RequirePlatform><PlatformMcpMarket /></RequirePlatform>} />
        <Route path="/platform/system-status" element={<RequirePlatform><SystemStatus /></RequirePlatform>} />
        <Route
          path="/tenant-management/:tenantId/ai"
          element={
            <RequirePlatform>
              <TenantAiConfig />
            </RequirePlatform>
          }
        />
        <Route path="/access-keys" element={<Navigate to="/profile/access-keys" replace />} />
        <Route
          path="/tenant-members"
          element={
            <RequireTenant>
              <TenantMembers />
            </RequireTenant>
          }
        />
        <Route
          path="/departments"
          element={<Navigate to="/tenant-members" replace />}
        />
        <Route
          path="/role-permissions"
          element={
            <RequireTenant>
              <TenantRolePermissions />
            </RequireTenant>
          }
        />
        <Route
          path="/billingplan"
          element={
            <RequirePlatform>
              <BillingPlan />
            </RequirePlatform>
          }
        />
        <Route path="/404" element={<NotFound />} />
        <Route path="*" element={<NotFound />} />
      </Route>
    </Routes>
  )
}
