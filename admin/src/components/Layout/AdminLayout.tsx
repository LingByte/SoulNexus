import { Suspense, useEffect } from 'react'
import { Outlet, useNavigate } from 'react-router-dom'
import { Layout, Button } from '@arco-design/web-react'
import { Bell, Moon, Sun, Settings, PanelLeftClose, PanelLeft } from 'lucide-react'
import AdminSidebar from './AdminSidebar'
import { useThemeStore } from '@/stores/themeStore'
import { useSidebar } from '@/contexts/SidebarContext'

const Sider = Layout.Sider
const Header = Layout.Header
const Content = Layout.Content

const AdminLayout = () => {
  const navigate = useNavigate()
  const { toggleMode, isDark } = useThemeStore()
  const { isCollapsed, toggleCollapse } = useSidebar()

  // 同步 Arco 暗色主题：通过 document.body 的 data-theme 属性
  useEffect(() => {
    document.body.setAttribute('arco-theme', isDark ? 'dark' : 'light')
  }, [isDark])

  return (
    <Layout className="min-h-screen">
      <Sider
        collapsed={isCollapsed}
        collapsible={false}
        width={220}
        collapsedWidth={64}
        breakpoint="lg"
        style={{ position: 'fixed', left: 0, top: 0, bottom: 0, zIndex: 30 }}
      >
        <AdminSidebar />
      </Sider>
      <Layout style={{ marginLeft: isCollapsed ? 64 : 220, transition: 'margin-left 0.2s' }}>
        <Header
          style={{
            position: 'sticky',
            top: 0,
            zIndex: 20,
            height: 56,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '0 16px',
            background: 'var(--color-bg-2)',
            borderBottom: '1px solid var(--color-border-2)',
          }}
        >
          <Button
            type="text"
            icon={isCollapsed ? <PanelLeft size={16} /> : <PanelLeftClose size={16} />}
            onClick={toggleCollapse}
          />
          <div className="flex items-center gap-1">
            <Button
              type="text"
              icon={<Bell size={16} />}
              onClick={() => navigate('/notifications')}
            />
            <Button
              type="text"
              icon={isDark ? <Sun size={16} /> : <Moon size={16} />}
              onClick={toggleMode}
            />
            <Button
              type="text"
              icon={<Settings size={16} />}
              onClick={() => navigate('/settings')}
            />
          </div>
        </Header>
        <Content style={{ padding: 20 }}>
          <Suspense fallback={<div className="p-8 text-center text-[var(--color-text-3)]">页面加载中...</div>}>
            <Outlet />
          </Suspense>
        </Content>
      </Layout>
    </Layout>
  )
}

export default AdminLayout
