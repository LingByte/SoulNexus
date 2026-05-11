import { useEffect, useMemo, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { Menu, Dropdown, Avatar } from '@arco-design/web-react'
import { User as UserIcon, LogOut } from 'lucide-react'
import { useAuthStore } from '@/stores/authStore'
import {
  Users,
  Sliders,
  FileText,
  Lock,
  KeyRound,
  Building2,
  Key,
  Code2,
  Receipt,
  Mic,
  Bot,
  Boxes,
  Puzzle,
  Bell,
  Mail,
  Send,
  AlertTriangle,
  BookOpen,
  Smartphone,
  MessageSquare,
  Activity,
  Shield,
  UserCog,
  Settings,
} from 'lucide-react'
import faviconUrl from '/favicon.png'
import { useSiteConfig } from '@/contexts/SiteConfigContext'
import { useSidebar } from '@/contexts/SidebarContext'

const MenuItem = Menu.Item
const SubMenu = Menu.SubMenu

type IconType = React.ComponentType<{ size?: number | string; className?: string }>

interface NavItem {
  key: string
  name: string
  icon: IconType
  children?: NavItem[]
}

const NAVIGATION: NavItem[] = [
  { key: '/users', name: '用户管理', icon: Users },
  {
    key: 'access',
    name: '访问控制',
    icon: Shield,
    children: [
      { key: '/permissions', name: '权限', icon: KeyRound },
      { key: '/roles', name: '角色', icon: Users },
      { key: '/user-access', name: '用户授权', icon: UserCog },
    ],
  },
  { key: '/assistants', name: '智能体管理', icon: Bot },
  { key: '/groups', name: '企业管理', icon: Building2 },
  { key: '/credentials', name: '密钥管理', icon: Key },
  { key: '/oauth-clients', name: 'OAuth 客户端', icon: KeyRound },
  { key: '/js-templates', name: 'JS 模版', icon: Code2 },
  { key: '/bills', name: '账单管理', icon: Receipt },
  { key: '/voice-training', name: '音色训练', icon: Mic },
  {
    key: 'workflow',
    name: '工作流',
    icon: Boxes,
    children: [
      { key: '/workflows', name: '工作流管理', icon: Boxes },
      { key: '/workflow-plugins', name: '插件市场', icon: Puzzle },
    ],
  },
  { key: '/notification-center', name: '通知中心', icon: Bell },
  {
    key: 'notification',
    name: '通知设置',
    icon: Send,
    children: [
      { key: '/notification-channels', name: '渠道供应商', icon: Send },
      { key: '/mail-templates', name: '邮件模板', icon: Mail },
      { key: '/mail-logs', name: '邮件日志', icon: FileText },
      { key: '/sms-logs', name: '短信日志', icon: FileText },
    ],
  },
  { key: '/alerts', name: '告警管理', icon: AlertTriangle },
  { key: '/knowledge-bases', name: '知识库', icon: BookOpen },
  { key: '/devices', name: '设备管理', icon: Smartphone },
  { key: '/chat-data', name: '会话与用量', icon: MessageSquare },
  { key: '/configs', name: '配置管理', icon: Sliders },
  {
    key: 'security',
    name: '安全管理',
    icon: Lock,
    children: [
      { key: '/operation-logs', name: '操作日志', icon: FileText },
      { key: '/account-locks', name: '账号锁定', icon: Lock },
    ],
  },
  {
    key: 'settings',
    name: '系统设置',
    icon: Settings,
    children: [
      { key: '/settings', name: '常规设置', icon: Settings },
      { key: '/service-health', name: '服务存活', icon: Activity },
    ],
  },
]

// 收集所有可点击的叶子路径，用于反向定位父级 submenu。
const findParentKey = (path: string): string | undefined => {
  for (const item of NAVIGATION) {
    if (item.children?.some((c) => path === c.key || path.startsWith(c.key + '/'))) {
      return item.key
    }
  }
  return undefined
}

const renderIcon = (Icon: IconType) => <Icon size={16} />

const AdminSidebar = () => {
  const location = useLocation()
  const navigate = useNavigate()
  const { isCollapsed } = useSidebar()
  const { config } = useSiteConfig()
  const { user, logout } = useAuthStore()
  const siteName = config?.SITE_NAME || 'SoulNexus 管理'

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  const userMenu = (
    <Menu onClickMenuItem={(key) => {
      if (key === 'profile') navigate('/profile')
      if (key === 'logout') handleLogout()
    }}>
      <Menu.Item key="profile">
        <span className="inline-flex items-center gap-2">
          <UserIcon size={14} /> 个人中心
        </span>
      </Menu.Item>
      <Menu.Item key="logout">
        <span className="inline-flex items-center gap-2 text-red-500">
          <LogOut size={14} /> 退出登录
        </span>
      </Menu.Item>
    </Menu>
  )

  const selectedKeys = useMemo(() => {
    // 命中规则：完整路径匹配优先，否则按前缀。
    const path = location.pathname
    for (const item of NAVIGATION) {
      if (!item.children && (path === item.key || path.startsWith(item.key + '/'))) {
        return [item.key]
      }
      const hit = item.children?.find((c) => path === c.key || path.startsWith(c.key + '/'))
      if (hit) return [hit.key]
    }
    return []
  }, [location.pathname])

  const [openKeys, setOpenKeys] = useState<string[]>(() => {
    const p = findParentKey(location.pathname)
    return p ? [p] : []
  })

  // 路由切换时自动展开匹配的父菜单（但不会主动关闭已展开的，保持用户操作意图）。
  useEffect(() => {
    const p = findParentKey(location.pathname)
    if (p) {
      setOpenKeys((prev) => (prev.includes(p) ? prev : [...prev, p]))
    }
  }, [location.pathname])

  const handleClickMenuItem = (key: string) => {
    if (key.startsWith('/')) {
      navigate(key)
    }
  }

  return (
    <div className="h-full flex flex-col bg-white dark:bg-[#17171a] border-r border-[var(--color-border-2)]">
      <div className="h-14 flex items-center gap-2 px-4 border-b border-[var(--color-border-2)] shrink-0">
        <img src={faviconUrl} alt="logo" className="w-7 h-7 rounded" />
        {!isCollapsed && (
          <span className="text-sm font-semibold text-[var(--color-text-1)] truncate">
            {siteName}
          </span>
        )}
      </div>
      <div className="flex-1 overflow-y-auto">
        <Menu
          mode="vertical"
          collapse={isCollapsed}
          selectedKeys={selectedKeys}
          openKeys={openKeys}
          onClickMenuItem={handleClickMenuItem}
          onClickSubMenu={(key, currentOpenKeys) => setOpenKeys(currentOpenKeys)}
          style={{ width: '100%', border: 'none' }}
        >
          {NAVIGATION.map((item) => {
            if (item.children?.length) {
              return (
                <SubMenu
                  key={item.key}
                  title={
                    <span className="inline-flex items-center gap-2">
                      {renderIcon(item.icon)}
                      <span>{item.name}</span>
                    </span>
                  }
                >
                  {item.children.map((child) => (
                    <MenuItem key={child.key}>
                      <span className="inline-flex items-center gap-2">
                        {renderIcon(child.icon)}
                        <span>{child.name}</span>
                      </span>
                    </MenuItem>
                  ))}
                </SubMenu>
              )
            }
            return (
              <MenuItem key={item.key}>
                <span className="inline-flex items-center gap-2">
                  {renderIcon(item.icon)}
                  <span>{item.name}</span>
                </span>
              </MenuItem>
            )
          })}
        </Menu>
      </div>
      {/* 底部用户信息 */}
      <div className="border-t border-[var(--color-border-2)] px-2 py-2 shrink-0">
        <Dropdown droplist={userMenu} position="tr" trigger="click">
          <button className="w-full flex items-center gap-2 px-2 py-2 rounded hover:bg-[var(--color-fill-2)] transition-colors">
            <Avatar size={28} style={{ backgroundColor: '#4ECDC4' }}>
              {(user?.displayName || user?.username || user?.email || 'A').slice(0, 1).toUpperCase()}
            </Avatar>
            {!isCollapsed && (
              <div className="flex-1 min-w-0 text-left">
                <div className="text-sm font-medium text-[var(--color-text-1)] truncate">
                  {user?.displayName || user?.username || '管理员'}
                </div>
                <div className="text-xs text-[var(--color-text-3)] truncate">
                  {user?.email || ''}
                </div>
              </div>
            )}
          </button>
        </Dropdown>
      </div>
    </div>
  )
}

export default AdminSidebar
