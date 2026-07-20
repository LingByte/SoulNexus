import { ReactNode, useEffect, useState } from 'react'
import { IconMoon, IconSun, IconMenuFold, IconMenuUnfold, IconMenu } from '@arco-design/web-react/icon'
import { Github } from 'lucide-react'
import Sidebar from './Sidebar'
import InboxBell from '@/components/inbox/InboxBell'
import { useInAppShell } from '@/contexts/AppShellContext'
import { useThemeStore } from '@/stores/themeStore'
import { useSidebar } from '@/contexts/SidebarContext'
import { useLocaleStore } from '@/stores/localeStore'
import { useTranslation } from '@/i18n'
import { Button, Select } from '@/components/ui'

interface AdminLayoutProps {
  children: ReactNode
  title?: string
  description?: string
  actions?: ReactNode
  /** When true, omit the sticky top bar (title strip + locale/theme/collapse). Use with Sidebar extras on specific routes. */
  hideHeader?: boolean
  /** Override main content padding (e.g. dense map pages). */
  contentPadding?: string
  /** Extra class on the main content element. */
  contentClassName?: string
}

const GITHUB_REPO_URL = 'https://github.com/LingByte/SoulNexus'

const BaseLayout = ({
  children,
  title,
  description,
  actions,
  hideHeader,
  contentPadding,
  contentClassName,
}: AdminLayoutProps) => {
  const inAppShell = useInAppShell()
  const { toggleMode, isDark } = useThemeStore()
  const { t } = useTranslation()
  const locale = useLocaleStore((s) => s.locale)
  const setLocale = useLocaleStore((s) => s.setLocale)
  const { isCollapsed, toggleCollapse, effectiveSidebarWidth, setMobileOpen } = useSidebar()
  const [isLg, setIsLg] = useState(() =>
    typeof window !== 'undefined' ? window.matchMedia('(min-width: 1024px)').matches : false
  )

  useEffect(() => {
    const q = window.matchMedia('(min-width: 1024px)')
    const sync = () => setIsLg(q.matches)
    sync()
    q.addEventListener('change', sync)
    return () => q.removeEventListener('change', sync)
  }, [])

  const marginLeft = inAppShell ? 0 : isLg ? effectiveSidebarWidth : 0

  const header = !hideHeader ? (
    <header className="sticky top-0 z-10 flex min-h-16 items-center border-b border-border bg-card px-4 sm:px-6 box-border">
      <div className="flex min-w-0 flex-1 items-center justify-between gap-3 sm:gap-4">
        <div className="flex min-w-0 flex-1 items-center gap-2 sm:gap-2.5">
          {!isLg && (
            <Button
              type="text"
              size="small"
              icon={<IconMenu />}
              onClick={() => setMobileOpen(true)}
              className="shrink-0"
            />
          )}
          <div className="min-w-0 flex-1 overflow-hidden">
            {title ? (
              <div
                className="truncate text-base font-semibold text-foreground sm:text-lg"
                title={typeof title === 'string' ? title : undefined}
              >
                {title}
              </div>
            ) : null}
            {description ? (
              <div
                className="mt-0.5 truncate text-xs text-muted-foreground sm:text-[13px]"
                title={typeof description === 'string' ? description : undefined}
              >
                {description}
              </div>
            ) : null}
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-1.5 sm:gap-2">
          <InboxBell />
          <Select
            size="small"
            value={locale}
            allowClear={false}
            style={{ width: 110 }}
            options={[
              { value: 'zh-CN', label: t('locale.zh') },
              { value: 'zh-TW', label: t('locale.tw') },
              { value: 'en-US', label: t('locale.en') },
              { value: 'ja-JP', label: t('locale.ja') },
            ]}
            onChange={(v) => setLocale(v as Parameters<typeof setLocale>[0])}
          />
          <Button
            type="secondary"
            size="small"
            icon={isDark ? <IconSun /> : <IconMoon />}
            onClick={toggleMode}
          />
          <Button
            type="text"
            size="small"
            icon={<Github size={16} strokeWidth={2} />}
            title={t('nav.github')}
            onClick={() => window.open(GITHUB_REPO_URL, '_blank', 'noopener,noreferrer')}
          />
          {actions}
          {isLg && (
            <Button
              type="secondary"
              size="small"
              icon={isCollapsed ? <IconMenuUnfold /> : <IconMenuFold />}
              onClick={toggleCollapse}
              title={isCollapsed ? t('layout.expandSidebar') : t('layout.collapseSidebar')}
            />
          )}
        </div>
      </div>
    </header>
  ) : null

  const main = (
    <main
      className={contentClassName}
      style={
        hideHeader
          ? { padding: contentPadding ?? 0, minHeight: 'calc(100vh - 0px)', width: '100%' }
          : { padding: contentPadding ?? '16px 20px' }
      }
    >
      {children}
    </main>
  )

  if (inAppShell) {
    return (
      <>
        {header}
        {main}
      </>
    )
  }

  return (
    <div className="min-h-screen bg-background text-foreground">
      <Sidebar />
      <div
        className="min-h-screen bg-background ling-main-with-sidebar"
        data-sidebar-offset
        style={{
          marginLeft,
          transition: 'margin-left 0.2s ease',
        }}
      >
        {header}
        {main}
      </div>
    </div>
  )
}

export default BaseLayout
