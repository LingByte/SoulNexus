import { ReactNode, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import AdminSidebar from './AdminSidebar'
import { Bell, Moon, Sun, Settings } from 'lucide-react'
import { useThemeStore } from '@/stores/themeStore'
import { useSidebar } from '@/contexts/SidebarContext'
import Button from '../UI/Button'

interface AdminLayoutProps {
  children: ReactNode
  title?: string
  description?: string
  actions?: ReactNode
}

const AdminLayout = ({ children, title, description, actions }: AdminLayoutProps) => {
  const { toggleMode, isDark } = useThemeStore()
  const { isCollapsed } = useSidebar()
  const navigate = useNavigate()
  const [isDesktop, setIsDesktop] = useState(false)

  useEffect(() => {
    const checkDesktop = () => {
      setIsDesktop(window.innerWidth >= 1024)
    }
    checkDesktop()
    window.addEventListener('resize', checkDesktop)
    return () => window.removeEventListener('resize', checkDesktop)
  }, [])

  return (
    <div className="min-h-screen bg-[#F7F9FC] dark:bg-slate-950">
      <AdminSidebar />
      
      {/* 主内容区 - 移动端无左边距，桌面端根据侧边栏状态调整 */}
      <div
        className="transition-all duration-200 ease-in-out"
        style={{
          marginLeft: isDesktop ? (isCollapsed ? '80px' : '220px') : '0px',
        }}
      >
        {/* 顶部导航栏 */}
        <header className="sticky top-0 z-20 bg-white/80 dark:bg-slate-950/80 backdrop-blur-xl border-b border-slate-200/50 dark:border-slate-800/50 shadow-sm">
          <div className="px-4 sm:px-6 lg:px-8">
            <div className="flex items-center justify-between h-16">
              {/* 左侧：Logo、标题和描述 */}
              <div className="flex items-center gap-4 flex-1 min-w-0">
                {/* 移动端显示Logo */}
                <div className="lg:hidden flex items-center gap-2">
                  <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-[#4ECDC4] to-[#45b8b0] flex items-center justify-center shadow-lg">
                    <img
                      src="/favicon.png"
                      alt="Logo"
                      className="w-6 h-6 object-contain"
                      onError={(e) => {
                        const target = e.target as HTMLImageElement
                        target.style.display = 'none'
                        const parent = target.parentElement
                        if (parent) {
                          parent.innerHTML =
                            '<svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" /></svg>'
                        }
                      }}
                    />
                  </div>
                </div>
                <div className="flex-1 min-w-0">
                  {title && (
                    <h1 className="text-xl font-semibold text-slate-900 dark:text-white">
                      {title}
                    </h1>
                  )}
                  {description && (
                    <p className="text-sm text-slate-500 dark:text-slate-400 mt-0.5">
                      {description}
                    </p>
                  )}
                </div>
              </div>

              {/* 右侧：操作按钮 */}
              <div className="flex items-center gap-2">
                {/* 通知按钮 */}
                <Button
                  variant="ghost"
                  size="sm"
                  className="relative"
                  leftIcon={<Bell className="w-4 h-4" />}
                  onClick={() => navigate('/notifications')}
                >
                  <span className="absolute top-1 right-1 w-2 h-2 bg-red-500 rounded-full" />
                </Button>

                {/* 主题切换 */}
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={toggleMode}
                  leftIcon={isDark ? <Sun className="w-4 h-4" /> : <Moon className="w-4 h-4" />}
                />

                {/* 设置按钮 */}
                <Button
                  variant="ghost"
                  size="sm"
                  leftIcon={<Settings className="w-4 h-4" />}
                  onClick={() => navigate('/settings')}
                />

                {/* 自定义操作 */}
                {actions}
              </div>
            </div>
          </div>
        </header>

        {/* 页面内容 */}
        <main className="px-4 sm:px-6 lg:px-8 py-6 lg:py-8">
          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
          >
            {children}
          </motion.div>
        </main>
      </div>
    </div>
  )
}

export default AdminLayout

