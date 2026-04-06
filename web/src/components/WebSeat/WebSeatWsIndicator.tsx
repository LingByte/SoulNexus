import Popover from '@/components/UI/Popover.tsx'
import Button from '@/components/UI/Button'
import { useI18nStore } from '@/stores/i18nStore'
import { cn } from '@/utils/cn.ts'
import type { WebSeatWsState } from './WebSeatContext'

interface WebSeatWsIndicatorProps {
  wsState: WebSeatWsState
  wsStatusText: string
  presenceWsClients: number
  onGoOnline: () => void
  onGoOffline: () => void
  onReconnect: () => void
  className?: string
}

function dotClass(wsState: WebSeatWsState): string {
  if (wsState === 'open') {
    return 'bg-emerald-500 shadow-[0_0_0_3px_rgba(16,185,129,0.35)]'
  }
  if (wsState === 'connecting') {
    return 'bg-amber-400 animate-pulse shadow-[0_0_0_3px_rgba(251,191,36,0.35)]'
  }
  if (wsState === 'disabled') {
    return 'bg-muted-foreground/40'
  }
  return 'bg-red-500 shadow-[0_0_0_3px_rgba(239,68,68,0.35)]'
}

export function WebSeatWsIndicator({
  wsState,
  wsStatusText,
  presenceWsClients,
  onGoOnline,
  onGoOffline,
  onReconnect,
  className,
}: WebSeatWsIndicatorProps) {
  const { t } = useI18nStore()
  const onlineBusy = wsState === 'open' || wsState === 'connecting'

  return (
    <Popover
      placement="bottom"
      trigger="click"
      contentClassName="min-w-[240px] p-0 overflow-hidden rounded-xl border border-border bg-card shadow-lg"
      content={
        <div className="p-3 text-sm space-y-3">
          <div>
            <p className="font-medium text-foreground leading-snug">{wsStatusText}</p>
            {wsState === 'open' && (
              <p className="mt-1.5 text-xs text-muted-foreground">
                {t('webseat.wsClientCountLabel')}
                <span className="font-mono text-foreground">{presenceWsClients}</span>
              </p>
            )}
          </div>
          <div className="flex flex-col gap-2">
            <Button
              type="button"
              size="sm"
              className="w-full"
              disabled={onlineBusy}
              onClick={() => onGoOnline()}
            >
              {t('webseat.goOnline')}
            </Button>
            <Button
              type="button"
              size="sm"
              variant="secondary"
              className="w-full"
              onClick={() => onGoOffline()}
            >
              {t('webseat.goOffline')}
            </Button>
            <Button
              type="button"
              size="sm"
              variant="outline"
              className="w-full"
              disabled={wsState === 'connecting'}
              onClick={() => onReconnect()}
            >
              {t('webseat.reconnectWs')}
            </Button>
          </div>
        </div>
      }
    >
      <button
        type="button"
        className={cn(
          'inline-flex items-center gap-2 rounded-full border border-border bg-card/95 px-2.5 py-1.5 text-xs font-medium text-foreground shadow-md backdrop-blur-sm transition hover:bg-accent/50',
          className
        )}
        aria-label={t('webseat.lineTitle')}
      >
        <span
          className={cn('h-2.5 w-2.5 shrink-0 rounded-full', dotClass(wsState))}
          aria-hidden
        />
        <span className="hidden sm:inline">{t('webseat.lineTitle')}</span>
      </button>
    </Popover>
  )
}
