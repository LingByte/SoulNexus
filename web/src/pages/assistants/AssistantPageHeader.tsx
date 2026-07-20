import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Space } from '@arco-design/web-react'
import { IconHome, IconMenu, IconMessage, IconSettings } from '@arco-design/web-react/icon'
import { useSidebar } from '@/contexts/SidebarContext'
import { Button } from '@/components/ui'
import { useTranslation } from '@/i18n'

export type AssistantPageHeaderProps = {
  listPath?: string
  currentLabel: string
  onCancel?: () => void
  cancelLabel?: string
  primaryLabel?: string
  primaryLoading?: boolean
  onPrimary?: () => void
  showPrimary?: boolean
  secondaryLabel?: string
  secondaryLoading?: boolean
  onSecondary?: () => void
  showSecondary?: boolean
  onDebugClick?: () => void
  debugActive?: boolean
  onSettingsClick?: () => void
}

export default function AssistantPageHeader({
  listPath = '/assistant-manager',
  currentLabel,
  onCancel,
  cancelLabel,
  primaryLabel,
  primaryLoading,
  onPrimary,
  showPrimary = false,
  secondaryLabel,
  secondaryLoading,
  onSecondary,
  showSecondary = false,
  onDebugClick,
  debugActive = false,
  onSettingsClick,
}: AssistantPageHeaderProps) {
  const navigate = useNavigate()
  const { setMobileOpen } = useSidebar()
  const { t } = useTranslation()
  const defaultCancelLabel = cancelLabel ?? t('common.cancel')
  const [isLg, setIsLg] = useState(() =>
    typeof window !== 'undefined' ? window.matchMedia('(min-width: 1024px)').matches : false,
  )

  useEffect(() => {
    const q = window.matchMedia('(min-width: 1024px)')
    const sync = () => setIsLg(q.matches)
    sync()
    q.addEventListener('change', sync)
    return () => q.removeEventListener('change', sync)
  }, [])

  return (
    <div
      className="flex items-center justify-between border-b border-border bg-card px-6 py-3"
      style={{ minHeight: 52 }}
    >
      <div className="flex min-w-0 items-center gap-2 text-sm text-muted-foreground">
        {!isLg && (
          <Button
            type="text"
            size="mini"
            icon={<IconMenu />}
            onClick={() => setMobileOpen(true)}
            style={{ padding: 2, marginRight: 4, flexShrink: 0 }}
          />
        )}
        <IconHome style={{ cursor: 'pointer', flexShrink: 0 }} onClick={() => navigate('/')} />
        <span>&gt;</span>
        <a
          className="shrink-0 cursor-pointer transition-colors hover:text-foreground"
          onClick={() => navigate(listPath)}
        >
          {t('assistant.breadcrumbLabel')}
        </a>
        <span>&gt;</span>
        <span className="truncate text-foreground" title={currentLabel}>
          {currentLabel}
        </span>
      </div>
      <Space size="small">
        {onSettingsClick && (
          <Button size="small" type="outline" icon={<IconSettings />} onClick={onSettingsClick}>
            设置
          </Button>
        )}
        {onDebugClick && (
          <Button
            size="small"
            type={debugActive ? 'primary' : 'outline'}
            icon={<IconMessage />}
            onClick={onDebugClick}
          >
            {t('common.debugChat')}
          </Button>
        )}
        <Button size="small" onClick={onCancel ?? (() => navigate(listPath))}>
          {defaultCancelLabel}
        </Button>
        {showSecondary && secondaryLabel && onSecondary && (
          <Button type="outline" size="small" loading={secondaryLoading} onClick={onSecondary}>
            {secondaryLabel}
          </Button>
        )}
        {showPrimary && primaryLabel && onPrimary && (
          <Button type="primary" size="small" loading={primaryLoading} onClick={onPrimary}>
            {primaryLabel}
          </Button>
        )}
      </Space>
    </div>
  )
}
