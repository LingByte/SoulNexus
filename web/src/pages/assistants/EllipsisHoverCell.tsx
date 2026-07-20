import { Popover } from '@arco-design/web-react'
import { cn } from '@/utils/cn'

interface EllipsisHoverCellProps {
  text: string | null | undefined
  className?: string
  lines?: 1 | 2 | 3
  mono?: boolean
}

export function EllipsisHoverCell({ text, className, lines = 2, mono }: EllipsisHoverCellProps) {
  const raw = text?.trim() ?? ''
  if (!raw) return <span className={cn('text-muted-foreground', className)}>—</span>

  return (
    <Popover
      content={
        <div style={{ maxWidth: 'min(24rem, calc(100vw - 2rem))', whiteSpace: 'pre-wrap', wordBreak: 'break-word', fontSize: 13 }}>
          {raw}
        </div>
      }
    >
        <span
          className={cn(
            'block w-full min-w-0 overflow-hidden text-left cursor-default',
            lines === 1 && 'truncate whitespace-nowrap',
            lines === 2 && 'line-clamp-2',
            lines === 3 && 'line-clamp-3',
            mono && lines !== 1 && 'font-mono text-xs break-all',
            mono && lines === 1 && 'font-mono text-xs',
            !mono && lines !== 1 && 'break-words',
            className
          )}
        >
          {raw}
        </span>
    </Popover>
  )
}
