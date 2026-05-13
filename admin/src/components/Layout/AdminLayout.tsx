import { Suspense, useEffect } from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { Layout, Button } from '@arco-design/web-react'
import { Bell, Moon, Sun, Settings, PanelLeftClose, PanelLeft } from 'lucide-react'
import AdminSidebar from './AdminSidebar'
import { useThemeStore } from '@/stores/themeStore'
import { useSidebar } from '@/contexts/SidebarContext'
import { useMediaQuery } from '@/hooks/useMediaQuery'

const Sider = Layout.Sider
const Header = Layout.Header
const Content = Layout.Content

const AdminLayout = () => {
  const navigate = useNavigate()
  const location = useLocation()
  const { toggleMode, isDark } = useThemeStore()
  const { isCollapsed, toggleCollapse, setIsCollapsed } = useSidebar()
  const isNarrow = useMediaQuery('(max-width: 991px)')

  useEffect(() => {
    document.body.setAttribute('arco-theme', isDark ? 'dark' : 'light')
  }, [isDark])

  // 窄屏下路由切换后收起侧栏，避免挡住内容
  useEffect(() => {
    if (isNarrow) {
      setIsCollapsed(true)
    }
  }, [location.pathname, isNarrow, setIsCollapsed])

  const mainMarginLeft = isNarrow ? 0 : isCollapsed ? 64 : 220

  const siderTransform =
    isNarrow && isCollapsed ? 'translate3d(-100%,0,0)' : 'translate3d(0,0,0)'

  return (
    <Layout className="min-h-screen">
      {isNarrow && !isCollapsed && (
        <button
          type="button"
          aria-label="关闭菜单"
          className="fixed inset-0 z-[29] border-0 bg-black/40 p-0"
          onClick={() => setIsCollapsed(true)}
        />
      )}
      <Sider
        collapsed={isCollapsed}
        collapsible={false}
        width={220}
        collapsedWidth={isNarrow ? 0 : 64}
        breakpoint="lg"
        className="!border-r !border-[var(--color-border-2)]"
        style={{
          position: 'fixed',
          left: 0,
          top: 0,
          bottom: 0,
          zIndex: 30,
          transform: siderTransform,
          transition: 'transform 0.2s ease, width 0.2s ease',
          boxShadow: isNarrow && !isCollapsed ? '4px 0 24px rgba(0,0,0,0.12)' : undefined,
        }}
      >
        <AdminSidebar />
      </Sider>
      <Layout
        className="min-w-0"
        style={{
          marginLeft: mainMarginLeft,
          transition: 'margin-left 0.2s ease',
        }}
      >
        <Header
          className="px-3 sm:px-4"
          style={{
            position: 'sticky',
            top: 0,
            zIndex: 20,
            height: 56,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            background: 'var(--color-bg-2)',
            borderBottom: '1px solid var(--color-border-2)',
          }}
        >
          <Button
            type="text"
            icon={isCollapsed ? <PanelLeft size={16} /> : <PanelLeftClose size={16} />}
            onClick={toggleCollapse}
          />
          <div className="flex items-center gap-0.5 sm:gap-1">
            <Button type="text" icon={<Bell size={16} />} onClick={() => navigate('/notifications')} />
            <Button
              type="text"
              icon={isDark ? <Sun size={16} /> : <Moon size={16} />}
              onClick={toggleMode}
            />
            <Button type="text" icon={<Settings size={16} />} onClick={() => navigate('/settings')} />
          </div>
        </Header>
        <Content
          style={{
            padding: isNarrow ? 12 : 20,
            maxWidth: '100%',
            overflowX: 'hidden',
          }}
        >
          <Suspense fallback={<div className="p-8 text-center text-[var(--color-text-3)]">页面加载中...</div>}>
            <Outlet />
          </Suspense>
        </Content>
      </Layout>
    </Layout>
  )
}

export default AdminLayout
