import { useState, useEffect, useRef } from 'react'
import faviconUrl from '/favicon.png'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { motion, AnimatePresence } from 'framer-motion'
import {
  Settings,
  LogOut,
  Menu,
  X,
  ChevronRight,
  User as UserIcon,
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
  AlertTriangle,
  BookOpen,
  Smartphone,
  MessageSquareText,
} from 'lucide-react'
import { useAuthStore } from '@/stores/authStore'
import { useSidebar } from '@/contexts/SidebarContext'
import { useSiteConfig } from '@/contexts/SiteConfigContext'
import { cn } from '@/utils/cn'

interface NavItem {
  name: string
  href: string
  icon: React.ComponentType<{ className?: string }>
  badge?: number
  children?: NavItem[]
}

const AdminSidebar = () => {
  const { isCollapsed, toggleCollapse } = useSidebar()
  const [isMobileOpen, setIsMobileOpen] = useState(false)
  const [expandedItems, setExpandedItems] = useState<string[]>([])
  const [showDropdown, setShowDropdown] = useState(false)
  const location = useLocation()
  const navigate = useNavigate()
  const { user, logout } = useAuthStore()
  const { config } = useSiteConfig()
  const dropdownRef = useRef<HTMLDivElement>(null)
  const siteName = config?.SITE_NAME || 'SoulNexus管理'
  const logoUrl = faviconUrl

  // 点击外部关闭下拉菜单
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setShowDropdown(false)
      }
    }

    if (showDropdown) {
      document.addEventListener('mousedown', handleClickOutside)
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [showDropdown])

  const navigation: NavItem[] = [
    { name: '用户管理', href: '/users', icon: Users },
    { name: '助手管理', href: '/assistants', icon: Bot },
    { name: '企业管理', href: '/groups', icon: Building2 },
    { name: '密钥管理', href: '/credentials', icon: Key },
    { name: 'OAuth 客户端', href: '/oauth-clients', icon: KeyRound },
    { name: 'JS 模版', href: '/js-templates', icon: Code2 },
    { name: '账单管理', href: '/bills', icon: Receipt },
    { name: '音色训练', href: '/voice-training', icon: Mic },
    {
      name: 'MCP 栏',
      href: '/mcp-marketplace',
      icon: Boxes,
      children: [
        { name: 'MCP 官方', href: '/mcp-marketplace', icon: Boxes },
        { name: 'MCP 管理', href: '/mcp-servers', icon: Bot },
      ],
    },
    {
      name: '工作流',
      href: '/workflows',
      icon: Boxes,
      children: [
        { name: '工作流管理', href: '/workflows', icon: Boxes },
        { name: '插件市场', href: '/workflow-plugins', icon: Puzzle },
      ],
    },
    { name: '通知中心', href: '/notification-center', icon: Bell },
    { name: '告警管理', href: '/alerts', icon: AlertTriangle },
    { name: '知识库', href: '/knowledge-bases', icon: BookOpen },
    { name: '设备管理', href: '/devices', icon: Smartphone },
    { name: '会话与用量', href: '/chat-data', icon: MessageSquareText },
    { name: '配置管理', href: '/configs', icon: Sliders },
    {
      name: '安全管理',
      href: '/security',
      icon: Lock,
      children: [
        { name: '操作日志', href: '/operation-logs', icon: FileText },
        { name: '账号锁定', href: '/account-locks', icon: Lock },
      ],
    },
    { name: '系统设置', href: '/settings', icon: Settings },
  ]

  const toggleExpand = (itemName: string) => {
    setExpandedItems((prev) =>
      prev.includes(itemName)
        ? prev.filter((name) => name !== itemName)
        : [...prev, itemName]
    )
  }

  const isActive = (path: string) => {
    return location.pathname === path || location.pathname.startsWith(path + '/')
  }

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  const SidebarContent = ({ showLogo = true }: { showLogo?: boolean }) => {
    const { config: sidebarConfig } = useSiteConfig()
    const currentSiteName = sidebarConfig?.SITE_NAME || 'SoulNexus管理后台'
    const sidebarLogoUrl = faviconUrl
    
    return (
      <>
        {/* Logo区域 - 只在桌面端显示，移动端不显示（因为移动端侧边栏已经有logo了） */}
        {showLogo && (
          <div className="h-16 flex items-center justify-between px-4 border-b border-slate-200 dark:border-slate-700">
            {!isCollapsed && (
              <Link to="/wordbooks" className="flex items-center gap-3 group">
                <div className="relative">
                  <div className="absolute inset-0 bg-gradient-to-br from-[#4ECDC4] to-[#45b8b0] rounded-lg blur-sm opacity-50 group-hover:opacity-75 transition-opacity" />
                  <div className="relative w-10 h-10 rounded-lg bg-gradient-to-br from-[#4ECDC4] to-[#45b8b0] flex items-center justify-center shadow-lg">
                    <img 
                      src={sidebarLogoUrl} 
                      alt={currentSiteName} 
                      className="w-7 h-7 object-contain"
                      onError={(e) => {
                        const target = e.target as HTMLImageElement
                        target.style.display = 'none'
                        const parent = target.parentElement
                        if (parent) {
                          parent.innerHTML = '<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" /></svg>'
                        }
                      }}
                    />
                  </div>
                </div>
                <span className="font-semibold text-xs whitespace-nowrap leading-none">
                  {currentSiteName}
                </span>
              </Link>
            )}
          {isCollapsed && (
            <div className="relative w-8 h-10 rounded-lg flex items-center justify-center mx-auto">
              <img 
                src={sidebarLogoUrl} 
                alt={currentSiteName} 
                className="w-7 h-7 object-contain"
                onError={(e) => {
                  const target = e.target as HTMLImageElement
                  target.style.display = 'none'
                  const parent = target.parentElement
                  if (parent) {
                    parent.innerHTML = '<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" /></svg>'
                  }
                }}
              />
            </div>
          )}
          <button
            onClick={toggleCollapse}
            className="hidden lg:flex items-center justify-center w-8 h-8 rounded-md hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-600 dark:text-slate-400 transition-colors"
            title={isCollapsed ? '展开' : '折叠'}
          >
            {isCollapsed ? (
              <ChevronRight className="w-4 h-4" />
            ) : (
              <X className="w-4 h-4" />
            )}
          </button>
        </div>
      )}

      {/* 导航菜单 */}
      <nav className="flex-1 px-3 py-4 space-y-1 overflow-y-auto">
        {navigation.map((item) => {
          const Icon = item.icon
          const hasChildren = item.children && item.children.length > 0
          const isExpanded = expandedItems.includes(item.name)
          const itemActive = isActive(item.href)

          if (hasChildren) {
            return (
              <div key={item.name}>
                <button
                  onClick={() => !isCollapsed && toggleExpand(item.name)}
                  className={cn(
                    'w-full flex items-center justify-between px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
                    itemActive
                      ? 'bg-teal-50 dark:bg-teal-900/20 text-teal-700 dark:text-teal-300'
                      : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800',
                    isCollapsed && 'justify-center'
                  )}
                >
                  <div className="flex items-center gap-3">
                    <Icon className="w-5 h-5" />
                    {!isCollapsed && <span>{item.name}</span>}
                  </div>
                  {!isCollapsed && (
                    <ChevronRight
                      className={cn(
                        'w-4 h-4 transition-transform',
                        isExpanded && 'rotate-90'
                      )}
                    />
                  )}
                </button>
                <AnimatePresence>
                  {isExpanded && !isCollapsed && (
                    <motion.div
                      initial={{ height: 0, opacity: 0 }}
                      animate={{ height: 'auto', opacity: 1 }}
                      exit={{ height: 0, opacity: 0 }}
                      className="ml-4 mt-1 space-y-1 border-l-2 border-slate-200 dark:border-slate-700 pl-4"
                    >
                      {item.children?.map((child) => {
                        const ChildIcon = child.icon
                        const childActive = isActive(child.href)
                        return (
                          <Link
                            key={child.name}
                            to={child.href}
                            className={cn(
                              'flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors',
                              childActive
                                ? 'bg-teal-50 dark:bg-teal-900/20 text-teal-700 dark:text-teal-300'
                                : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800'
                            )}
                          >
                            <ChildIcon className="w-4 h-4" />
                            <span>{child.name}</span>
                          </Link>
                        )
                      })}
                    </motion.div>
                  )}
                </AnimatePresence>
              </div>
            )
          }

          return (
            <Link
              key={item.name}
              to={item.href}
              className={cn(
                'flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
                itemActive
                  ? 'bg-teal-50 dark:bg-teal-900/20 text-teal-700 dark:text-teal-300'
                  : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800',
                isCollapsed && 'justify-center'
              )}
              title={isCollapsed ? item.name : ''}
            >
              <Icon className="w-5 h-5" />
              {!isCollapsed && (
                <span className="flex-1">{item.name}</span>
              )}
              {item.badge && !isCollapsed && (
                <span className="px-2 py-0.5 text-xs font-medium bg-teal-100 dark:bg-teal-900 text-teal-700 dark:text-teal-300 rounded-full">
                  {item.badge}
                </span>
              )}
            </Link>
          )
        })}
      </nav>

      {/* 底部用户信息 */}
      {!isCollapsed && (
        <div className="px-3 py-4 border-t border-slate-200 dark:border-slate-800">
          <div className="relative" ref={dropdownRef}>
            <button
              onClick={() => setShowDropdown(!showDropdown)}
              className="flex items-center gap-3 w-full p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
            >
              <img
                src={user?.avatar || '/favicon.png'}
                alt="avatar"
                className="w-9 h-9 rounded-full object-cover border border-slate-200 dark:border-slate-700"
              />
              <div className="flex-1 min-w-0 text-left">
                <p className="text-sm font-medium text-slate-900 truncate">
                  {user?.displayName || user?.email || '管理员'}
                </p>
                <p className="text-xs text-slate-500 dark:text-slate-400 truncate">
                  {user?.email || 'admin@example.com'}
                </p>
              </div>
              <ChevronRight className={cn(
                "w-4 h-4 text-slate-400 transition-transform",
                showDropdown && "rotate-90"
              )} />
            </button>
            
            {/* 下拉菜单 */}
            {showDropdown && (
              <div className="absolute bottom-full left-0 right-0 mb-2 bg-white dark:bg-slate-900 rounded-lg shadow-lg border border-slate-200 dark:border-slate-800 overflow-hidden">
                <button
                  onClick={() => {
                    setShowDropdown(false)
                    navigate('/profile')
                  }}
                  className="w-full flex items-center gap-3 px-4 py-3 text-left hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors"
                >
                  <UserIcon className="w-4 h-4 text-slate-600 dark:text-slate-400" />
                  <span className="text-sm text-slate-900">个人中心</span>
                </button>
                <button
                  onClick={() => {
                    setShowDropdown(false)
                    handleLogout()
                  }}
                  className="w-full flex items-center gap-3 px-4 py-3 text-left hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors text-red-600 dark:text-red-400"
                >
                  <LogOut className="w-4 h-4" />
                  <span className="text-sm">退出登录</span>
                </button>
              </div>
            )}
          </div>
        </div>
      )}
    </>
    )
  }

  return (
    <>
      {/* 移动端菜单按钮 */}
      <button
        onClick={() => setIsMobileOpen(true)}
        className="lg:hidden fixed top-4 left-4 z-50 p-2 rounded-lg bg-white dark:bg-slate-800 shadow-lg border border-slate-200 dark:border-slate-700"
      >
        <Menu className="w-5 h-5 text-slate-700 dark:text-slate-300" />
      </button>

      {/* 移动端遮罩 */}
      <AnimatePresence>
        {isMobileOpen && (
          <>
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              onClick={() => setIsMobileOpen(false)}
              className="lg:hidden fixed inset-0 bg-black/50 z-40"
            />
            <motion.aside
              initial={{ x: -280 }}
              animate={{ x: 0 }}
              exit={{ x: -280 }}
              transition={{ type: 'spring', damping: 25, stiffness: 200 }}
              className="lg:hidden fixed left-0 top-0 bottom-0 w-70 bg-white/95 dark:bg-slate-950/95 backdrop-blur-xl border-r border-slate-200/50 dark:border-slate-800/50 shadow-xl z-50 flex flex-col"
            >
              <div className="h-16 flex items-center justify-between px-4 border-b border-slate-200 dark:border-slate-700">
                <div className="flex items-center gap-3">
                  <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-[#4ECDC4] to-[#45b8b0] flex items-center justify-center shadow-lg">
                    <img 
                      src={logoUrl} 
                      alt={siteName} 
                      className="w-6 h-6 object-contain"
                      onError={(e) => {
                        const target = e.target as HTMLImageElement
                        target.style.display = 'none'
                        const parent = target.parentElement
                        if (parent) {
                          parent.innerHTML = '<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" /></svg>'
                        }
                      }}
                    />
                  </div>
                  <span className="font-semibold text-xs whitespace-nowrap leading-none">
                    {siteName}
                  </span>
                </div>
                <button
                  onClick={() => setIsMobileOpen(false)}
                  className="p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800"
                >
                  <X className="w-5 h-5 text-slate-700 dark:text-slate-300" />
                </button>
              </div>
              <SidebarContent showLogo={false} />
            </motion.aside>
          </>
        )}
      </AnimatePresence>

      {/* 桌面端侧边栏 */}
      <motion.aside
        initial={false}
        animate={{ width: isCollapsed ? 80 : 220 }}
        transition={{ duration: 0.2, ease: 'easeInOut' }}
        className="hidden lg:flex flex-col bg-white/95 dark:bg-slate-950/95 backdrop-blur-xl border-r border-slate-200/50 dark:border-slate-800/50 shadow-lg fixed left-0 top-0 bottom-0 z-30"
      >
        <SidebarContent showLogo={true} />
      </motion.aside>

    </>
  )
}

export default AdminSidebar

