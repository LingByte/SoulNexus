import { useEffect, useState, useCallback } from 'react'
import { Avatar, Drawer, Layout } from '@arco-design/web-react'
import { useLocation, useNavigate } from 'react-router-dom'
import {
  FileText,
  UserCircle,
  Shield,
  Briefcase,
  Contact,
  Settings2,
  BookOpen,
  BrainCircuit,
  GitBranch,
  Bot,
  Fingerprint,
  Lock,
  ListTodo,
  Mail,
  MessageSquare,
  Activity,
  Package,
  AudioWaveform,
  Wrench,
  Store,
} from 'lucide-react'
import { useSidebar } from '@/contexts/SidebarContext'
import {
  SIDEBAR_COLLAPSED_WIDTH,
  SIDEBAR_DEFAULT_WIDTH,
  SIDEBAR_MIN_WIDTH,
  SIDEBAR_MAX_WIDTH,
  clampSidebarWidth,
} from '@/constants/sidebar'
import { useSiteConfig } from '@/contexts/siteConfig'
import { useAuthStore } from '@/stores/authStore'
import { useTranslation } from '@/i18n'
import { Link } from '@/components/ui'
import { cn } from '@/utils/cn'
import { PLATFORM_HOME_PATH, TENANT_HOME_PATH } from '@/constants/appPaths'

const Sider = Layout.Sider

const DOCS_CENTER_URL = 'https://docs.lingecho.com'

type NavDef = {
  labelKey: string
  href: string
  icon: typeof Bot
  /** 租户登录：仅当 effective 权限含该菜单码时展示（平台管理员不按此项过滤） */
  tenantMenuCode?: string
  /** 满足任一权限码即可展示（用于菜单/API 权限并存） */
  tenantMenuAnyOf?: string[]
  /** 租户登录：对所有租户用户展示（不校验菜单权限） */
  tenantAllUsers?: boolean
  /** 租户登录：NLU 开启时对所有租户用户展示 */
  tenantNluWhenEnabled?: boolean
}

type NavGroup = {
  labelKey: string
  items: NavDef[]
}

// 分组导航
const navGroups: NavGroup[] = [
  {
    labelKey: 'nav.groupAiAgents',
    items: [
      { labelKey: 'nav.assistantManager', href: '/assistant-manager', icon: Bot, tenantMenuCode: 'menu.res.assistant' },
      { labelKey: 'nav.mcpMarket', href: '/mcp-market', icon: Store, tenantMenuCode: 'menu.res.assistant' },
      { labelKey: 'nav.myMcp', href: '/mcp', icon: Wrench, tenantMenuCode: 'menu.res.assistant' },
      { labelKey: 'nav.voiceCloneManager', href: '/voice-clone-manager', icon: AudioWaveform, tenantMenuCode: 'menu.res.assistant' },
      { labelKey: 'nav.voiceprintManager', href: '/voiceprint-manager', icon: Fingerprint, tenantMenuCode: 'menu.res.assistant' },
      { labelKey: 'nav.nluLab', href: '/nlu-models', icon: BrainCircuit, tenantNluWhenEnabled: true },
    ],
  },
  {
    labelKey: 'nav.groupKnowledge',
    items: [
      { labelKey: 'nav.knowledgeBase', href: '/knowledge-base', icon: BookOpen, tenantAllUsers: true },
      { labelKey: 'nav.workflows', href: '/workflows', icon: GitBranch, tenantMenuAnyOf: ['menu.res.workflow', 'api.workflow.read', 'menu.res.assistant', 'api.assistants.read'] },
      { labelKey: 'nav.pluginMarket', href: '/plugin-market', icon: Package, tenantMenuAnyOf: ['menu.res.workflow', 'api.workflow.read', 'menu.res.assistant', 'api.assistants.read'] },
      { labelKey: 'nav.jsTemplates', href: '/js-templates', icon: FileText, tenantMenuAnyOf: ['menu.res.assistant', 'api.assistants.read'] },
    ],
  },
  {
    labelKey: 'nav.groupOrganization',
    items: [
      { labelKey: 'nav.tenantMembers', href: '/tenant-members', icon: Contact, tenantMenuCode: 'menu.org.members' },
      { labelKey: 'nav.rolePermissions', href: '/role-permissions', icon: Lock, tenantMenuCode: 'menu.org.role' },
    ],
  },
  {
    labelKey: 'nav.groupSystem',
    items: [
      { labelKey: 'nav.tenantManagement', href: '/tenant-management', icon: Briefcase },
      { labelKey: 'nav.platformAdmins', href: '/platform-admins', icon: Shield },
      { labelKey: 'nav.platformVoiceManagement', href: '/platform/voice-management', icon: Bot },
      { labelKey: 'nav.platformAiPools', href: '/platform/ai-pools', icon: AudioWaveform },
      { labelKey: 'nav.platformVoiceprintManagement', href: '/platform/voiceprint-management', icon: Fingerprint },
      { labelKey: 'nav.systemConfigs', href: '/system-configs', icon: Settings2 },
      { labelKey: 'nav.systemStatus', href: '/platform/system-status', icon: Activity },
    ],
  },
  {
    labelKey: 'nav.groupNotifications',
    items: [
      { labelKey: 'nav.notificationChannels', href: '/platform/notification-channels', icon: MessageSquare },
      { labelKey: 'nav.mailTemplates', href: '/platform/mail-templates', icon: Mail },
      { labelKey: 'nav.mailLogs', href: '/platform/mail-logs', icon: Mail },
      { labelKey: 'nav.smsLogs', href: '/platform/sms-logs', icon: MessageSquare },
      { labelKey: 'nav.aiInvocations', href: '/platform/ai-invocations', icon: Activity },
      { labelKey: 'nav.executionTasks', href: '/platform/execution-tasks', icon: ListTodo },
    ],
  },
  {
    labelKey: 'nav.groupMarketplace',
    items: [
      { labelKey: 'nav.platformNluModels', href: '/platform/nlu-models', icon: BrainCircuit },
      { labelKey: 'nav.platformPluginMarket', href: '/platform/plugin-market', icon: Package },
      { labelKey: 'nav.platformMcpMarket', href: '/platform/mcp-market', icon: Store },
    ],
  },
]

// 展平所有导航项（用于路由匹配等）
const allNavigation: NavDef[] = navGroups.flatMap((g) => g.items)

function tenantMaySeeItem(menuCodes: readonly string[] | undefined, item: NavDef): boolean {
  const required = item.tenantMenuAnyOf?.length
    ? item.tenantMenuAnyOf
    : item.tenantMenuCode
      ? [item.tenantMenuCode]
      : []
  if (!required.length) return false
  const list = menuCodes ?? []
  if (!list.length) return false
  return required.some((code) => list.includes(code))
}

const platformAdminMenuHrefs = new Set([
  '/tenant-management',
  '/platform-admins',
  '/platform/voice-management',
  '/platform/ai-pools',
  '/platform/voiceprint-management',
  '/system-configs',
  '/platform/system-status',
  '/platform/nlu-models',
  '/platform/notification-channels',
  '/platform/mail-templates',
  '/platform/mail-logs',
  '/platform/ai-invocations',
  '/platform/sms-logs',
  '/platform/execution-tasks',
  '/platform/plugin-market',
  '/platform/mcp-market',
])

const tenantHiddenHrefs = new Set([
  '/tenant-management',
])

function selectedMenuKey(pathname: string, items: NavDef[]): string {
  const exact = items.find((n) => pathname === n.href)
  if (exact) return exact.href
  const sorted = [...items].sort((a, b) => b.href.length - a.href.length)
  const hit = sorted.find((n) => pathname.startsWith(`${n.href}/`)) ?? items[0] ?? allNavigation[0]
  return hit.href
}

function NavMenuBody({
  collapsed,
  onNavigate,
}: {
  collapsed: boolean
  onNavigate?: () => void
}) {
  const location = useLocation()
  const navigate = useNavigate()
  const { config, loading: siteConfigLoading } = useSiteConfig()
  const { t } = useTranslation()
  const user = useAuthStore((s) => s.user)
  const refreshUserInfo = useAuthStore((s) => s.refreshUserInfo)
  const isPlatformAdmin = Boolean(user?.isPlatformAdmin || user?.principal === 'platform')
  const isTenantUser = user?.principal === 'tenant'

  useEffect(() => {
    if (isTenantUser) void refreshUserInfo()
  }, [isTenantUser, refreshUserInfo])

  const dn = String(user?.displayName || '').trim()
  const un = String(user?.username || '').trim()
  const em = String(user?.email || '').trim()
  const sidebarUserLabel = dn || un || em || t('nav.me')
  const avatarUrl = isTenantUser ? String(user?.avatarUrl || '').trim() : ''
  const menuCodes = (user?.permissionCodes as string[] | undefined) ?? []
  const voiceCloneEnabled = Boolean((config.VOICE_CLONE_PROVIDER || '').trim())
  const voiceprintEnabled = Boolean((config.VOICEPRINT_PROVIDER || '').trim())
  const nluEnabled = Boolean(config.nluEnabled)
  const isCommunity = config.deploymentMode === 'community'
  const visibleNavigation = isPlatformAdmin
    ? allNavigation.filter((n) => platformAdminMenuHrefs.has(n.href))
    : allNavigation.filter((n) => {
        if (tenantHiddenHrefs.has(n.href)) return false
        if (isCommunity && n.href === '/role-permissions') {
          return false
        }
        if (n.href === '/voice-clone-manager' && !voiceCloneEnabled) return false
        if (n.href === '/voiceprint-manager' && !voiceprintEnabled) return false
        if (n.tenantNluWhenEnabled) return nluEnabled
        if (n.tenantAllUsers) return true
        return tenantMaySeeItem(menuCodes, n)
      })
  const visibleHrefSet = new Set(visibleNavigation.map((n) => n.href))

  // 过滤分组：仅保留至少有一个可见项的组
  const visibleGroups = navGroups
    .map((g) => ({
      ...g,
      items: g.items.filter((item) => visibleHrefSet.has(item.href)),
    }))
    .filter((g) => g.items.length > 0)

  const siteName = config.SITE_NAME || t('layout.siteName')
  const logoUrl = config.SITE_LOGO_URL || '/icon-lingyu.png'
  const siteUrl = String(config.SITE_URL || '').trim()
  const logoLinkExternal = /^https?:\/\//i.test(siteUrl)
  const brandKey = `${siteName}|${logoUrl}`
  const brandInner = (
    <>
      <img
        src={logoUrl}
        alt={siteName}
        className={`logo-brand site-brand-logo${siteConfigLoading ? ' site-brand-logo--loading' : ''}`}
        style={{ width: collapsed ? 28 : 36, height: collapsed ? 28 : 36, flexShrink: 0 }}
      />
      {!collapsed && (
        <span style={{ fontWeight: 600, fontSize: 17, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
          {siteName}
        </span>
      )}
    </>
  )
  const brandLinkStyle = {
    display: 'flex',
    alignItems: 'center',
    gap: 12,
    textDecoration: 'none',
    color: 'var(--color-text-1)',
    overflow: 'hidden',
    justifyContent: collapsed ? 'center' : 'flex-start',
    flex: collapsed ? undefined : '1 1 auto',
    minWidth: 0,
  } as const
  const selected =
    visibleNavigation.length > 0 ? selectedMenuKey(location.pathname, visibleNavigation) : location.pathname

  const pinnedHref = isPlatformAdmin ? PLATFORM_HOME_PATH : TENANT_HOME_PATH
  const pinnedItem = visibleNavigation.find((n) => n.href === pinnedHref)
  const navGroupsFiltered = visibleGroups
    .map((g) => ({
      ...g,
      items: g.items.filter((item) => item.href !== pinnedHref),
    }))
    .filter((g) => g.items.length > 0)

  const renderNavItem = (item: NavDef, compact?: boolean) => {
    const Icon = item.icon
    const label = t(item.labelKey)
    const active = selected === item.href
    return (
      <button
        key={item.href}
        type="button"
        title={compact ? label : undefined}
        aria-label={label}
        aria-current={active ? 'page' : undefined}
        className={cn(
          'ling-sidebar-nav-item',
          compact && 'ling-sidebar-nav-item--compact',
          active && 'ling-sidebar-nav-item--active',
        )}
        onClick={() => {
          navigate(item.href)
          onNavigate?.()
        }}
      >
        <Icon size={18} strokeWidth={2} aria-hidden className="shrink-0" />
        {!compact && <span className="ling-sidebar-nav-label truncate">{label}</span>}
      </button>
    )
  }

  return (
    <>
      <div
        style={{
          height: collapsed ? 72 : 64,
          display: 'flex',
          flexDirection: collapsed ? 'column' : 'row',
          alignItems: 'center',
          justifyContent: collapsed ? 'center' : 'space-between',
          padding: collapsed ? '8px 4px' : '0 12px 0 16px',
          borderBottom: '1px solid var(--color-border)',
          flexShrink: 0,
          gap: collapsed ? 4 : 8,
          boxSizing: 'border-box',
        }}
      >
        {logoLinkExternal ? (
          <a
            key={brandKey}
            href={siteUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="site-brand-block"
            style={brandLinkStyle}
            onClick={() => onNavigate?.()}
          >
            {brandInner}
          </a>
        ) : (
        <Link
          key={brandKey}
          to={isPlatformAdmin ? PLATFORM_HOME_PATH : TENANT_HOME_PATH}
          className="site-brand-block"
          style={brandLinkStyle}
          onClick={() => onNavigate?.()}
        >
          {brandInner}
        </Link>
        )}
      </div>

      <div className="ling-sidebar-scroll flex-1 overflow-y-auto overflow-x-hidden">
        {collapsed ? (
          <div className="ling-sidebar-nav ling-sidebar-nav--collapsed">
            {pinnedItem ? renderNavItem(pinnedItem, true) : null}
            {visibleNavigation
              .filter((n) => n.href !== pinnedHref)
              .map((item) => renderNavItem(item, true))}
          </div>
        ) : (
          <nav className="ling-sidebar-nav px-2 py-3" aria-label="Main navigation">
            {pinnedItem ? (
              <div className="mb-3 px-1">{renderNavItem(pinnedItem)}</div>
            ) : null}
            {navGroupsFiltered.map((group) => (
              <div key={group.labelKey} className="mb-4">
                <div className="ling-sidebar-group-label">{t(group.labelKey)}</div>
                <div className="flex flex-col gap-0.5">
                  {group.items.map((item) => renderNavItem(item))}
                </div>
              </div>
            ))}
          </nav>
        )}
      </div>

      <div
        style={{
          padding: collapsed ? '10px 0' : '10px 12px',
          borderTop: '1px solid var(--color-border)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: collapsed ? 'center' : 'space-between',
          width: '100%',
          boxSizing: 'border-box',
          flexShrink: 0,
          gap: collapsed ? 0 : 6,
        }}
      >
        <div
          role="button"
          tabIndex={0}
          className="ling-sidebar-profile-hit"
          onClick={() => {
            navigate('/profile/info')
            onNavigate?.()
          }}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault()
              navigate('/profile/info')
              onNavigate?.()
            }
          }}
          style={{
            display: 'flex',
            flexDirection: 'row',
            alignItems: 'center',
            gap: 10,
            flex: collapsed ? '0 0 auto' : '1 1 auto',
            minWidth: 0,
            padding: collapsed ? 0 : '4px 8px',
            cursor: 'pointer',
            borderRadius: 8,
            boxSizing: 'border-box',
          }}
        >
          <Avatar size={32} style={{ flexShrink: 0, backgroundColor: 'var(--color-fill-3)' }}>
            {isTenantUser && avatarUrl ? (
              <img alt="" src={avatarUrl} style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
            ) : (
              <UserCircle size={22} strokeWidth={2} color="var(--color-text-2)" />
            )}
          </Avatar>
          {!collapsed && (
            <span
              style={{
                flex: 1,
                minWidth: 0,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
                textAlign: 'left',
                fontSize: 13,
                fontWeight: 500,
                color: 'var(--color-text-1)',
              }}
            >
              {sidebarUserLabel}
            </span>
          )}
        </div>

        {!collapsed && (
          <a
            href={DOCS_CENTER_URL}
            target="_blank"
            rel="noopener noreferrer"
            title={t('nav.docsCenter')}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 4,
              flexShrink: 0,
              fontSize: 12,
              fontWeight: 500,
              color: 'var(--color-text-3)',
              textDecoration: 'none',
              padding: '6px 6px',
              borderRadius: 6,
              transition: 'color 0.15s ease, background-color 0.15s ease',
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.color = 'rgb(var(--primary-6))'
              e.currentTarget.style.backgroundColor = 'var(--color-fill-2)'
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.color = 'var(--color-text-3)'
              e.currentTarget.style.backgroundColor = 'transparent'
            }}
          >
            <BookOpen size={14} strokeWidth={2} aria-hidden />
          </a>
        )}
      </div>
    </>
  )
}

const Sidebar = () => {
  const { isCollapsed, sidebarWidth, setSidebarWidth, setIsCollapsed, mobileOpen, setMobileOpen } = useSidebar()
  const [resizing, setResizing] = useState(false)

  const startResize = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault()
      e.stopPropagation()
      if (isCollapsed) setIsCollapsed(false)
      setResizing(true)

      const onMove = (ev: MouseEvent) => {
        setSidebarWidth(clampSidebarWidth(ev.clientX))
      }
      const onUp = () => {
        setResizing(false)
        document.body.classList.remove('ling-sidebar-resizing')
        document.body.style.cursor = ''
        document.body.style.userSelect = ''
        document.removeEventListener('mousemove', onMove)
        document.removeEventListener('mouseup', onUp)
      }

      document.body.classList.add('ling-sidebar-resizing')
      document.body.style.cursor = 'col-resize'
      document.body.style.userSelect = 'none'
      document.addEventListener('mousemove', onMove)
      document.addEventListener('mouseup', onUp)
    },
    [isCollapsed, setIsCollapsed, setSidebarWidth],
  )

  const siderWidth = isCollapsed ? SIDEBAR_COLLAPSED_WIDTH : sidebarWidth

  return (
    <>
      <Drawer
        title={null}
        visible={mobileOpen}
        placement="left"
        width={280}
        footer={null}
        closable
        onCancel={() => setMobileOpen(false)}
        className="lg:hidden"
      >
        <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
          <NavMenuBody collapsed={false} onNavigate={() => setMobileOpen(false)} />
        </div>
      </Drawer>

      <Sider
        className="ling-sidebar-sider hidden lg:block"
        collapsible
        trigger={null}
        collapsed={isCollapsed}
        width={siderWidth}
        collapsedWidth={SIDEBAR_COLLAPSED_WIDTH}
        style={{
          height: '100vh',
          position: 'fixed',
          left: 0,
          top: 0,
          boxSizing: 'border-box',
          transition: resizing ? 'none' : 'width 0.2s ease',
          zIndex: 90,
        }}
      >
        <div className="ling-sidebar-root" style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
          <NavMenuBody collapsed={isCollapsed} />
        </div>
        {!isCollapsed && (
          <div
            role="separator"
            aria-orientation="vertical"
            aria-valuemin={SIDEBAR_MIN_WIDTH}
            aria-valuemax={SIDEBAR_MAX_WIDTH}
            aria-valuenow={sidebarWidth}
            aria-label="调整侧栏宽度"
            className={`ling-sidebar-resize-handle${resizing ? ' ling-sidebar-resize-handle--active' : ''}`}
            onMouseDown={startResize}
            onDoubleClick={() => setSidebarWidth(SIDEBAR_DEFAULT_WIDTH)}
          />
        )}
      </Sider>
    </>
  )
}

export default Sidebar
