import { useState, useEffect, useRef } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import {
  Component, 
  Settings,
  ChevronLeft,
  ChevronRight,
  Bot,
  User as UserIcon,
  LogOut,
  Bell,
  BookOpen, // 加入文档图标
  Key, // 新增密钥图标
  FileText, // 账单图标
  Users, // 组织管理图标
  AlertTriangle, // 告警图标
  Database, // 配额管理图标
  Smartphone, // 设备管理图标
  Library, // 知识库图标
  GitBranch, // 工作流图标
  LayoutDashboard, // 概览图标
  Phone, // 外呼中心图标
  Info, // 关于我们图标
  Package, // 插件市场图标
  Mic, // 声纹识别图标
  Store, // MCP 广场图标
} from 'lucide-react'
import { useAuthStore } from '@/stores/authStore'
import { useI18nStore } from '@/stores/i18nStore'
import AuthModal from '../Auth/AuthModal'
import Button from '../UI/Button'
import { getGroupList, type Group } from '@/api/group'
import { getSystemInit, type SystemInitInfo } from '@/api/system'
import { prefetch } from '@/utils/prefetch'

const Sidebar = () => {
  const [isCollapsed, setIsCollapsed] = useState(false)
  const location = useLocation()
  const { user, isAuthenticated, logout, currentOrganizationId, setCurrentOrganization } = useAuthStore()
  const { t } = useI18nStore()
  const [showAuthModal, setShowAuthModal] = useState(false)
  const [showDropdown, setShowDropdown] = useState(false)
  const [groups, setGroups] = useState<Group[]>([])
  const [systemInfo, setSystemInfo] = useState<SystemInitInfo | null>(null)
  const navigate = useNavigate()
  const dropdownRef = useRef<HTMLDivElement>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const dropdownContainerRef = useRef<HTMLDivElement>(null)
  const hoverTimeoutRef = useRef<NodeJS.Timeout | null>(null)

  // 点击外部关闭下拉菜单
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node) &&
          buttonRef.current && !buttonRef.current.contains(event.target as Node)) {
        setShowDropdown(false)
      }
    }

    if (showDropdown) {
      // 使用延迟，避免在鼠标移动时立即关闭
      const timer = setTimeout(() => {
        document.addEventListener('mousedown', handleClickOutside)
      }, 100)
      return () => {
        clearTimeout(timer)
        document.removeEventListener('mousedown', handleClickOutside)
      }
    }
  }, [showDropdown])

  // 获取组织列表
  useEffect(() => {
    if (isAuthenticated) {
      fetchGroups()
    } else {
      setGroups([])
    }
  }, [isAuthenticated])

  // 获取系统初始化信息
  useEffect(() => {
    fetchSystemInfo()
  }, [])

  const fetchGroups = async () => {
    try {
      const res = await getGroupList()
      setGroups(res.data || [])
    } catch (err) {
      console.error('获取组织列表失败', err)
    }
  }

  const fetchSystemInfo = async () => {
    try {
      const res = await getSystemInit()
      if (res.code === 200) {
        setSystemInfo(res.data)
      }
    } catch (err) {
      console.error('获取系统信息失败', err)
    }
  }

  const navigation = [
    ...(isAuthenticated && groups.length > 0 ? [{ name: t('nav.sidebar.overview'), href: '/overview', icon: LayoutDashboard }] : []),
    { name: t('nav.sidebar.smartAssistant'), href: '/voice-assistant', icon: Bot },
    { name: t('nav.sidebar.voiceTraining'), href: '/voice-training', icon: Settings },
    ...(systemInfo?.features?.voiceprintEnabled ? [{ name: t('voiceprint.title'), href: '/voiceprint-management', icon: Mic }] : []),
    { name: t('nav.sidebar.mcpManagement'), href: '/mcp-management', icon: Package },
    { name: t('nav.sidebar.mcpMarketplace'), href: '/mcp-marketplace', icon: Store },
    { name: t('nav.sidebar.knowledgeBase'), href: '/knowledge', icon: Library },
    { name: t('nav.sidebar.workflow'), href: '/workflows', icon: GitBranch },
    { name: t('nav.sidebar.pluginMarket'), href: '/node-plugins', icon: Package },
    { name: t('nav.sidebar.notification'), href: '/notification', icon: Bell},
    { name: t('nav.sidebar.alerts'), href: '/alerts', icon: AlertTriangle },
    { name: t('nav.sidebar.jsTemplate'), href: '/js-templates', icon: Component },
    { name: t('nav.sidebar.billing'), href: '/billing', icon: FileText },
    { name: t('nav.sidebar.groups'), href: '/groups', icon: Users },
    { name: t('nav.sidebar.deviceManagement'), href: '/devices', icon: Smartphone },
    { name: t('nav.sidebar.callCenter'), href: '/call-center', icon: Phone },
    { name: t('nav.about'), href: '/about', icon: Info },
    { name: t('nav.docs'), href: '/docs', icon: BookOpen },
  ]

  const publicNavs = [t('nav.docs'), t('nav.about')]
  // 受保护页面名称
  const privateNavs = [t('nav.sidebar.overview'), t('nav.sidebar.smartAssistant'), t('nav.sidebar.voiceTraining'), t('voiceprint.title'), t('nav.sidebar.knowledgeBase'), t('nav.sidebar.workflow'), t('nav.sidebar.pluginMarket'), t('nav.sidebar.notification'), t('nav.sidebar.alerts'), t('nav.sidebar.jsTemplate'), t('nav.sidebar.billing'), t('nav.sidebar.groups'), t('nav.sidebar.deviceManagement'), t('nav.sidebar.callCenter'), t('nav.sidebar.mcpManagement'), t('nav.sidebar.mcpMarketplace')]

  const isActive = (path: string) => location.pathname === path

  // 桌面端侧边栏内容
  const desktopSidebarContent = (
    <>
      {/* 顶部 LOGO 区域 */}
      <div className="h-14 flex items-center border-b border-border px-3 relative">
        {!isCollapsed && (
          <Link to="/" className="flex items-center gap-2">
            <img
              src="https://cetide-1325039295.cos.ap-chengdu.myqcloud.com/folder/icon-192x192.ico"
              alt="LingEcho Logo"
              className="w-6 h-8 rounded"
            />
            <span
              className="relative inline-block text-sm font-extrabold tracking-wider"
            >
              <span className="block">{t('brand.name')}</span>
              <span className="absolute inset-0 bg-gradient-to-r  via-violet-400  bg-clip-text pointer-events-none select-none">
                {t('brand.name')}
              </span>
            </span>
          </Link>
        )}
        {/* 桌面端折叠/展开按钮 */}
        <button
          onClick={() => setIsCollapsed(!isCollapsed)}
          className="absolute right-2 top-3 inline-flex items-center justify-center w-7 h-7 rounded-full hover:bg-accent text-muted-foreground hover:text-foreground transition-colors"
          title={isCollapsed ? t('theme.expand') : t('theme.collapse')}
        >
          {isCollapsed ? (
            <ChevronRight className="w-4 h-4" />
          ) : (
            <ChevronLeft className="w-4 h-4" />
          )}
        </button>
      </div>

      {/* 导航 */}
      <nav className="flex-1 p-4 space-y-1 overflow-y-auto">
        {navigation.filter(item => {
          if (publicNavs.includes(item.name)) return true;
          if (privateNavs.includes(item.name)) return isAuthenticated;
          return true;
        }).map(item => {
          const Icon = item.icon
          return (
            <Link
              key={item.name}
              to={item.href}
              className={`group relative flex items-center rounded-md font-medium transition-colors ${
                isActive(item.href) || location.pathname.startsWith(item.href + '/')
                  ? 'text-foreground bg-accent'
                  : 'text-muted-foreground hover:text-foreground hover:bg-accent'
              } ${isCollapsed ? 'justify-center px-2 py-3 hover:bg-accent/50' : 'px-3 py-2'}`}
              title={isCollapsed ? item.name : ''}
            >
              <Icon
                className={`${
                  isCollapsed 
                    ? 'w-5 h-5' 
                    : 'w-4 h-4 mr-3'
                } ${
                  isActive(item.href)
                    ? 'text-foreground'
                    : isCollapsed
                      ? 'text-foreground group-hover:text-foreground'
                      : 'text-muted-foreground group-hover:text-foreground'
                }`}
                style={{ 
                  display: 'block',
                  minWidth: isCollapsed ? '20px' : '16px',
                  minHeight: isCollapsed ? '20px' : '16px'
                }}
              />
              {!isCollapsed && (
                <motion.span
                  initial={false}
                  animate={{ opacity: 1 }}
                  transition={{ duration: 0.2 }}
                  className="text-xs whitespace-nowrap"
                >
                  {item.name}
                </motion.span>
              )}
              {(isActive(item.href) || location.pathname.startsWith(item.href + '/')) && !isCollapsed && (
                <motion.div
                  layoutId="activeSidebarItem"
                  className="absolute right-0 w-1 h-6 bg-primary rounded-l-full"
                  transition={{ type: 'spring', bounce: 0.2, duration: 0.3 }}
                />
              )}
            </Link>
          )
        })}
      </nav>
      {/* 底部功能区 */}
      <div className="mt-auto p-2 flex flex-col gap-2 relative">
        {/* 通知中心按钮已移除，只保留用户区相关 */}
        <div>
          {isAuthenticated && user ? (
            <div 
              className="relative"
              ref={dropdownContainerRef}
              onMouseEnter={() => {
                if (hoverTimeoutRef.current) {
                  clearTimeout(hoverTimeoutRef.current)
                  hoverTimeoutRef.current = null
                }
                if (!isCollapsed) {
                  setShowDropdown(true)
                }
              }}
              onMouseLeave={() => {
                // 延迟关闭，给鼠标移动到菜单的时间
                if (hoverTimeoutRef.current) {
                  clearTimeout(hoverTimeoutRef.current)
                }
                hoverTimeoutRef.current = setTimeout(() => {
                  setShowDropdown(false)
                  hoverTimeoutRef.current = null
                }, 150)
              }}
            >
              <button
                ref={buttonRef}
                className={`flex items-center w-full p-1 rounded hover:bg-accent transition-colors group text-muted-foreground hover:text-foreground ${isCollapsed ? 'justify-center' : ''}`}
                onClick={() => setShowDropdown((open) => !open)}
              >
                <img
                  src={user.avatar || `https://ui-avatars.com/api/?name=${user.displayName || 'U'}&background=0ea5e9&color=fff`}
                  alt={user.displayName}
                  className={`rounded-full ${isCollapsed ? 'w-9 h-9' : 'w-8 h-8 mr-2'}`}
                />
                {!isCollapsed && (
                  <span className="text-sm font-medium truncate max-w-[80px]">{user.displayName}</span>
                )}
              </button>
              {showDropdown && !isCollapsed && (
                <div 
                  ref={dropdownRef}
                  className="absolute right-0 bottom-full mb-2 w-40 bg-popover rounded-md shadow-lg border z-50"
                  onMouseEnter={() => {
                    if (hoverTimeoutRef.current) {
                      clearTimeout(hoverTimeoutRef.current)
                      hoverTimeoutRef.current = null
                    }
                    setShowDropdown(true)
                  }}
                  onMouseLeave={() => {
                    if (hoverTimeoutRef.current) {
                      clearTimeout(hoverTimeoutRef.current)
                    }
                    hoverTimeoutRef.current = setTimeout(() => {
                      setShowDropdown(false)
                      hoverTimeoutRef.current = null
                    }, 150)
                  }}
                >
                  <div className="flex flex-col p-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      className="flex items-center gap-2 w-full justify-start text-sm px-3 py-2"
                      onClick={() => { setShowDropdown(false); navigate('/profile') }}
                      leftIcon={<UserIcon className="w-4 h-4" />}
                    >
                      {t('nav.sidebar.profile')}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="flex items-center gap-2 w-full justify-start text-sm px-3 py-2"
                      onClick={() => { setShowDropdown(false); navigate('/credential') }}
                      leftIcon={<Key className="w-4 h-4" />}
                    >
                      {t('nav.sidebar.credential')}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="flex items-center gap-2 w-full justify-start text-sm px-3 py-2"
                      onClick={() => { setShowDropdown(false); navigate('/quotas') }}
                      leftIcon={<Database className="w-4 h-4" />}
                    >
                      {t('nav.sidebar.quotas')}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="flex items-center gap-2 w-full justify-start text-sm px-3 py-2"
                      onClick={async () => { setShowDropdown(false); await logout() }}
                      leftIcon={<LogOut className="w-4 h-4" />}
                    >
                      {t('nav.logout')}
                    </Button>
                  </div>
                </div>
              )}
            </div>
          ) : (
            <Button
              variant="primary"
              className="w-full justify-center"
              onClick={() => setShowAuthModal(true)}
              leftIcon={<UserIcon className="w-4 h-4" />}
            >
              {!isCollapsed && t('nav.loginRegister')}
            </Button>
          )}
          {/* 登录弹窗AuthModal */}
          <AuthModal isOpen={showAuthModal} onClose={() => setShowAuthModal(false)} />
        </div>
        
        {/* 组织选择器 - 横向图标展示，放在最底部 */}
        {isAuthenticated && groups.length > 0 && (
          <div className="mt-1 pt-1 border-t border-border">
            {!isCollapsed ? (
              <div 
                className="flex items-center gap-2 overflow-x-auto [&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none]"
              >
                {groups.map((group) => {
                  const isSelected = currentOrganizationId === group.id
                  const groupInitial = group.name.charAt(0).toUpperCase()
                  const avatarUrl = group.avatar || `https://ui-avatars.com/api/?name=${encodeURIComponent(group.name)}&background=6366f1&color=fff&size=32`
                  
                  return (
                    <motion.button
                      key={group.id}
                      onClick={() => {
                        setCurrentOrganization(group.id)
                        // 预加载该组织的概览数据
                        prefetch.prefetchOverview(group.id)
                      }}
                      className={`
                        relative flex-shrink-0 w-6 h-6 rounded-lg overflow-hidden
                        transition-all duration-200
                        ${isSelected 
                          ? 'ring-2 ring-blue-400/40 ring-offset-1 ring-offset-background' 
                          : 'hover:ring-2 hover:ring-muted'
                        }
                      `}
                      title={group.name}
                      whileHover={{ scale: 1.05 }}
                      whileTap={{ scale: 0.95 }}
                    >
                      <img
                        src={avatarUrl}
                        alt={group.name}
                        className="w-full h-full object-cover"
                        onError={(e) => {
                          // 如果图片加载失败，显示文字头像
                          const target = e.target as HTMLImageElement
                          target.style.display = 'none'
                          const parent = target.parentElement
                          if (parent && !parent.querySelector('.fallback-avatar')) {
                            const fallback = document.createElement('div')
                            fallback.className = 'fallback-avatar w-full h-full flex items-center justify-center bg-primary text-primary-foreground text-xs font-semibold'
                            fallback.textContent = groupInitial
                            parent.appendChild(fallback)
                          }
                        }}
                      />
                      {isSelected && (
                        <motion.div
                          className="absolute inset-0 bg-primary/20"
                          initial={{ opacity: 0 }}
                          animate={{ opacity: 1 }}
                          transition={{ duration: 0.2 }}
                        />
                      )}
                      {isSelected && (
                        <motion.div
                          className="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-400/40"
                          initial={{ scaleX: 0 }}
                          animate={{ scaleX: 1 }}
                          transition={{ duration: 0.2 }}
                        />
                      )}
                    </motion.button>
                  )
                })}
              </div>
            ) : (
              <div className="flex flex-col items-center gap-1.5">
                {groups.slice(0, 3).map((group) => {
                  const isSelected = currentOrganizationId === group.id
                  const groupInitial = group.name.charAt(0).toUpperCase()
                  const avatarUrl = group.avatar || `https://ui-avatars.com/api/?name=${encodeURIComponent(group.name)}&background=6366f1&color=fff&size=24`
                  
                  return (
                    <motion.button
                      key={group.id}
                      onClick={() => {
                        setCurrentOrganization(group.id)
                        // 预加载该组织的概览数据
                        prefetch.prefetchOverview(group.id)
                      }}
                      className={`
                        relative w-7 h-7 rounded-lg overflow-hidden
                        transition-all duration-200
                        ${isSelected 
                          ? 'ring-2 ring-blue-400/40' 
                          : 'hover:ring-2 hover:ring-muted'
                        }
                      `}
                      title={group.name}
                      whileHover={{ scale: 1.1 }}
                      whileTap={{ scale: 0.9 }}
                    >
                      <img
                        src={avatarUrl}
                        alt={group.name}
                        className="w-full h-full object-cover"
                        onError={(e) => {
                          const target = e.target as HTMLImageElement
                          target.style.display = 'none'
                          const parent = target.parentElement
                          if (parent && !parent.querySelector('.fallback-avatar')) {
                            const fallback = document.createElement('div')
                            fallback.className = 'fallback-avatar w-full h-full flex items-center justify-center bg-primary text-primary-foreground text-[10px] font-semibold'
                            fallback.textContent = groupInitial
                            parent.appendChild(fallback)
                          }
                        }}
                      />
                      {isSelected && (
                        <motion.div
                          className="absolute inset-0 bg-primary/20"
                          initial={{ opacity: 0 }}
                          animate={{ opacity: 1 }}
                        />
                      )}
                    </motion.button>
                  )
                })}
                {groups.length > 3 && (
                  <div className="text-[10px] text-muted-foreground">+{groups.length - 3}</div>
                )}
              </div>
            )}
          </div>
        )}
      </div>
    </>
  )

  // 过滤后的导航项
  const filteredNavigation = navigation.filter(item => {
    if (publicNavs.includes(item.name)) return true;
    if (privateNavs.includes(item.name)) return isAuthenticated;
    return true;
  })

  return (
    <>
      {/* 桌面端侧边栏 */}
      <motion.aside
        initial={false}
        animate={{ width: isCollapsed ? 72 : 192 }}
        transition={{ duration: 0.3, ease: 'easeInOut' }}
        className="hidden lg:flex flex-col bg-background border-r border-border relative"
      >
        {desktopSidebarContent}
      </motion.aside>

      {/* 移动端顶部 Header */}
      <header className="lg:hidden fixed top-0 left-0 right-0 z-50 bg-background border-b border-border">
        {/* LOGO 和用户信息行 */}
        <div className="h-14 flex items-center justify-between px-4 border-b border-border">
          <Link to="/" className="flex items-center gap-2 flex-shrink-0">
            <img
              src="https://cetide-1325039295.cos.ap-chengdu.myqcloud.com/folder/icon-192x192.ico"
              alt="LingEcho Logo"
              className="w-6 h-8 rounded"
            />
            <span className="text-sm font-extrabold tracking-wider">
              {t('brand.name')}
            </span>
          </Link>
          
          {/* 用户信息 */}
          <div className="flex items-center gap-2">
            {isAuthenticated && groups.length > 0 && (
              <div className="flex items-center gap-1.5 overflow-x-auto [&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none] max-w-[120px]">
                {groups.map((group) => {
                  const isSelected = currentOrganizationId === group.id
                  const avatarUrl = group.avatar || `https://ui-avatars.com/api/?name=${encodeURIComponent(group.name)}&background=6366f1&color=fff&size=24`
                  
                  return (
                    <button
                      key={group.id}
                      onClick={() => {
                        setCurrentOrganization(group.id)
                        prefetch.prefetchOverview(group.id)
                      }}
                      className={`relative flex-shrink-0 w-6 h-6 rounded-lg overflow-hidden transition-all ${
                        isSelected ? 'ring-2 ring-primary' : ''
                      }`}
                      title={group.name}
                    >
                      <img
                        src={avatarUrl}
                        alt={group.name}
                        className="w-full h-full object-cover"
                      />
                    </button>
                  )
                })}
              </div>
            )}
            
            {isAuthenticated && user ? (
              <div className="relative" ref={dropdownRef}>
                <button
                  onClick={() => setShowDropdown(!showDropdown)}
                  className="flex items-center gap-2 p-1 rounded hover:bg-accent transition-colors"
                >
                  <img
                    src={user.avatar || `https://ui-avatars.com/api/?name=${user.displayName || 'U'}&background=0ea5e9&color=fff`}
                    alt={user.displayName}
                    className="w-8 h-8 rounded-full"
                  />
                </button>
                {showDropdown && (
                  <div className="absolute right-0 top-full mt-2 w-40 bg-popover rounded-md shadow-lg border z-50">
                    <div className="flex flex-col p-2">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="flex items-center gap-2 w-full justify-start text-sm px-3 py-2"
                        onClick={() => { setShowDropdown(false); navigate('/profile') }}
                        leftIcon={<UserIcon className="w-4 h-4" />}
                      >
                        {t('nav.sidebar.profile')}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="flex items-center gap-2 w-full justify-start text-sm px-3 py-2"
                        onClick={() => { setShowDropdown(false); navigate('/credential') }}
                        leftIcon={<Key className="w-4 h-4" />}
                      >
                        {t('nav.sidebar.credential')}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="flex items-center gap-2 w-full justify-start text-sm px-3 py-2"
                        onClick={() => { setShowDropdown(false); navigate('/quotas') }}
                        leftIcon={<Database className="w-4 h-4" />}
                      >
                        {t('nav.sidebar.quotas')}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="flex items-center gap-2 w-full justify-start text-sm px-3 py-2"
                        onClick={async () => { setShowDropdown(false); await logout() }}
                        leftIcon={<LogOut className="w-4 h-4" />}
                      >
                        {t('nav.logout')}
                      </Button>
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <Button
                variant="primary"
                size="sm"
                onClick={() => setShowAuthModal(true)}
                leftIcon={<UserIcon className="w-4 h-4" />}
              >
                {t('nav.loginRegister')}
              </Button>
            )}
            <AuthModal isOpen={showAuthModal} onClose={() => setShowAuthModal(false)} />
          </div>
        </div>

        {/* 导航项横向滚动 */}
        <div className="h-12 flex items-center overflow-x-auto [&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none] bg-background border-b border-border">
          <nav className="flex items-center gap-1 px-2 min-w-max">
            {filteredNavigation.map(item => {
              const Icon = item.icon
              const active = isActive(item.href) || location.pathname.startsWith(item.href + '/')
              return (
                <Link
                  key={item.name}
                  to={item.href}
                  className={`group relative flex items-center gap-2 px-3 py-2 rounded-md font-medium text-xs whitespace-nowrap transition-colors ${
                    active
                      ? 'text-foreground bg-accent'
                      : 'text-muted-foreground hover:text-foreground hover:bg-accent/50'
                  }`}
                >
                  <Icon className="w-4 h-4 flex-shrink-0" />
                  <span>{item.name}</span>
                  {active && (
                    <motion.div
                      layoutId="activeMobileNavItem"
                      className="absolute bottom-0 left-0 right-0 h-0.5 bg-primary rounded-t-full"
                      transition={{ type: 'spring', bounce: 0.2, duration: 0.3 }}
                    />
                  )}
                </Link>
              )
            })}
          </nav>
        </div>
      </header>
    </>
  )
}

export default Sidebar