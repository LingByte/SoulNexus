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
  Smartphone, // 设备管理图标
  GitBranch, // 工作流图标
  Package, // 插件市场图标
  Mic, // 声纹识别图标
  BookOpen,
} from 'lucide-react'
import { useAuthStore } from '@/stores/authStore'
import { useI18nStore } from '@/stores/i18nStore'
import Button from '../UI/Button'
import ConfirmDialog from '../UI/ConfirmDialog'
import { getSystemInit, type SystemInitInfo } from '@/api/system'
import { beginSSOLogin } from '@/utils/sso'

const Sidebar = () => {
  const [isCollapsed, setIsCollapsed] = useState(false)
  const location = useLocation()
  const { user, isAuthenticated, logout } = useAuthStore()
  const { t } = useI18nStore()
  const [showDropdown, setShowDropdown] = useState(false)
  const [confirmLogoutOpen, setConfirmLogoutOpen] = useState(false)
  const [systemInfo, setSystemInfo] = useState<SystemInitInfo | null>(null)
  const navigate = useNavigate()
  const dropdownRef = useRef<HTMLDivElement>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const mobileDropdownRef = useRef<HTMLDivElement>(null)
  const mobileButtonRef = useRef<HTMLButtonElement>(null)
  const dropdownContainerRef = useRef<HTMLDivElement>(null)
  const hoverTimeoutRef = useRef<NodeJS.Timeout | null>(null)

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as Node
      const inDesktopDropdown = dropdownRef.current?.contains(target) || buttonRef.current?.contains(target)
      const inMobileDropdown = mobileDropdownRef.current?.contains(target) || mobileButtonRef.current?.contains(target)
      if (!inDesktopDropdown && !inMobileDropdown) {
        setShowDropdown(false)
      }
    }

    if (showDropdown) {
      const timer = setTimeout(() => {
        document.addEventListener('mousedown', handleClickOutside)
      }, 100)
      return () => {
        clearTimeout(timer)
        document.removeEventListener('mousedown', handleClickOutside)
      }
    }
  }, [showDropdown])

  // 获取系统初始化信息
  useEffect(() => {
    fetchSystemInfo()
  }, [])

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
    { name: t('nav.sidebar.smartAssistant'), href: '/voice-assistant', icon: Bot },
    { name: t('nav.sidebar.voiceTraining'), href: '/voice-training', icon: Settings },
    { name: t('nav.sidebar.knowledgeBase'), href: '/knowledge', icon: BookOpen },
    ...(systemInfo?.features?.voiceprintEnabled ? [{ name: t('voiceprint.title'), href: '/voiceprint-management', icon: Mic }] : []),
    { name: t('nav.sidebar.workflow'), href: '/workflows', icon: GitBranch },
    { name: t('nav.sidebar.pluginMarket'), href: '/node-plugins', icon: Package },
    { name: t('nav.sidebar.jsTemplate'), href: '/js-templates', icon: Component },
    { name: t('nav.sidebar.deviceManagement'), href: '/devices', icon: Smartphone },
  ]

  const navigationSections = [
    {
      title: '核心功能',
      items: navigation.filter((item) => ['/voice-assistant', '/voice-training', '/voiceprint-management', '/knowledge'].includes(item.href)),
    },
    {
      title: '工作流',
      items: navigation.filter((item) => ['/workflows', '/node-plugins'].includes(item.href)),
    },
    {
      title: '运营与管理',
      items: navigation.filter((item) => ['/alerts', '/js-templates', '/devices'].includes(item.href)),
    },
  ].filter((section) => section.items.length > 0)

  const publicNavs = [t('nav.docs')]
  // 受保护页面名称
  const privateNavs = [t('nav.sidebar.smartAssistant'), t('nav.sidebar.voiceTraining'), t('nav.sidebar.knowledgeBase'), t('voiceprint.title'), t('nav.sidebar.workflow'), t('nav.sidebar.pluginMarket'), t('nav.sidebar.jsTemplate'), t('nav.sidebar.deviceManagement')]

  const isActive = (path: string) => location.pathname === path
  const isExternalHref = (href: string) => href.startsWith('http')
  const handleSidebarLogout = () => {
    setShowDropdown(false)
    setConfirmLogoutOpen(true)
  }

  const performLogout = async () => {
    await logout()
  }

  const desktopSidebarContent = (
    <>
      {/* 顶部 LOGO + 折叠：收起时仅显示 logo，展开按钮在 logo 下方 */}
      <div className="border-b border-border flex-shrink-0">
        {isCollapsed ? (
          <div className="flex flex-col items-center gap-2 py-3 px-2">
            <Link
              to="/"
              className="flex justify-center rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              title={t('brand.name')}
            >
              <img src="/SoulMy.png" alt="" className="w-9 h-9 rounded" />
            </Link>
            <button
              type="button"
              onClick={() => setIsCollapsed(false)}
              className="inline-flex items-center justify-center w-8 h-8 rounded-full hover:bg-accent text-muted-foreground hover:text-foreground transition-colors"
              title={t('theme.expand')}
              aria-label={t('theme.expand')}
            >
              <ChevronRight className="w-4 h-4" />
            </button>
          </div>
        ) : (
          <div className="h-14 flex items-center px-3 relative">
            <Link to="/" className="flex items-center gap-2 min-w-0 pr-10">
              <img src="/SoulMy.png" alt="SoulNexus Logo" className="w-8 h-8 rounded shrink-0" />
              <span className="relative inline-block text-sm font-extrabold tracking-wider truncate">
                <span className="block">{t('brand.name')}</span>
                <span className="absolute inset-0 bg-gradient-to-r via-violet-400 bg-clip-text pointer-events-none select-none">
                  {t('brand.name')}
                </span>
              </span>
            </Link>
            <button
              type="button"
              onClick={() => setIsCollapsed(true)}
              className="absolute right-2 top-1/2 -translate-y-1/2 inline-flex items-center justify-center w-7 h-7 rounded-full hover:bg-accent text-muted-foreground hover:text-foreground transition-colors"
              title={t('theme.collapse')}
              aria-label={t('theme.collapse')}
            >
              <ChevronLeft className="w-4 h-4" />
            </button>
          </div>
        )}
      </div>

      {/* 导航 */}
      <nav className="flex-1 p-4 space-y-4 overflow-y-auto">
        {navigationSections.map((section) => (
          <div key={section.title} className="space-y-1">
            {!isCollapsed && (
              <div className="px-2 pb-1 text-[10px] uppercase tracking-wide text-muted-foreground/80">
                {section.title}
              </div>
            )}
            {section.items.filter(item => {
              if (publicNavs.includes(item.name)) return true;
              if (privateNavs.includes(item.name)) return isAuthenticated;
              return true;
            }).map(item => {
          const Icon = item.icon
          return (
            isExternalHref(item.href) ? (
              <a
                key={item.name}
                href={item.href}
                target="_blank"
                rel="noreferrer"
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
                    className="absolute right-0 w-1 h-6 bg-primary rounded-l-full pointer-events-none"
                    transition={{ type: 'spring', bounce: 0.2, duration: 0.3 }}
                  />
                )}
              </a>
            ) : (
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
                    className="absolute right-0 w-1 h-6 bg-primary rounded-l-full pointer-events-none"
                    transition={{ type: 'spring', bounce: 0.2, duration: 0.3 }}
                  />
                )}
              </Link>
            )
          )
        })}
          </div>
        ))}
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
                }, 280)
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
                  className="absolute right-0 bottom-full w-40 bg-popover rounded-md shadow-lg border z-50"
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
                    }, 280)
                  }}
                >
                  <div className="flex flex-col p-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      className="flex items-center gap-2 w-full justify-start text-sm px-3 py-2"
                      onClick={() => { setShowDropdown(false); navigate('/profile/personal') }}
                      leftIcon={<UserIcon className="w-4 h-4" />}
                    >
                      {t('nav.sidebar.profile')}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="flex items-center gap-2 w-full justify-start text-sm px-3 py-2"
                      onClick={handleSidebarLogout}
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
              onClick={() => beginSSOLogin(location.pathname + location.search)}
              leftIcon={<UserIcon className="w-4 h-4" />}
            >
              {!isCollapsed && t('nav.loginRegister')}
            </Button>
          )}
        </div>
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
        className="hidden lg:flex flex-col h-screen min-h-0 bg-background border-r border-border relative"
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
              alt="SoulNexus Logo"
              className="w-6 h-8 rounded"
            />
            <span className="text-sm font-extrabold tracking-wider">
              {t('brand.name')}
            </span>
          </Link>
          
          {/* 用户信息 */}
          <div className="flex items-center gap-2">
            {isAuthenticated && user ? (
              <div className="relative" ref={mobileDropdownRef}>
                <button
                  ref={mobileButtonRef}
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
                        onClick={() => { setShowDropdown(false); navigate('/profile/personal') }}
                        leftIcon={<UserIcon className="w-4 h-4" />}
                      >
                        {t('nav.sidebar.profile')}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="flex items-center gap-2 w-full justify-start text-sm px-3 py-2"
                        onClick={handleSidebarLogout}
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
                onClick={() => beginSSOLogin(location.pathname + location.search)}
                leftIcon={<UserIcon className="w-4 h-4" />}
              >
                {t('nav.loginRegister')}
              </Button>
            )}
          </div>
        </div>

        {/* 导航项横向滚动 */}
        <div className="h-12 flex items-center overflow-x-auto [&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none] bg-background border-b border-border">
          <nav className="flex items-center gap-1 px-2 min-w-max">
            {filteredNavigation.map(item => {
              const Icon = item.icon
              const active = isActive(item.href) || location.pathname.startsWith(item.href + '/')
              return (
                isExternalHref(item.href) ? (
                  <a
                    key={item.name}
                    href={item.href}
                    target="_blank"
                    rel="noreferrer"
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
                        className="absolute bottom-0 left-0 right-0 h-0.5 bg-primary rounded-t-full pointer-events-none"
                        transition={{ type: 'spring', bounce: 0.2, duration: 0.3 }}
                      />
                    )}
                  </a>
                ) : (
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
                        className="absolute bottom-0 left-0 right-0 h-0.5 bg-primary rounded-t-full pointer-events-none"
                        transition={{ type: 'spring', bounce: 0.2, duration: 0.3 }}
                      />
                    )}
                  </Link>
                )
              )
            })}
          </nav>
        </div>
      </header>
      <ConfirmDialog
        isOpen={confirmLogoutOpen}
        onClose={() => setConfirmLogoutOpen(false)}
        onConfirm={performLogout}
        title={t('nav.logoutConfirmTitle')}
        message={t('nav.logoutConfirmMessage')}
        confirmText={t('nav.logout')}
        cancelText={t('profile.cancel')}
        type="warning"
      />
    </>
  )
}

export default Sidebar