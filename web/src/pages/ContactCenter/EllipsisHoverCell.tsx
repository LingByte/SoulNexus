import Popover from '@/components/UI/Popover.tsx'
import { cn } from '@/utils/cn.ts'

interface EllipsisHoverCellProps {
  /** Raw text; empty → em dash */
  text: string | null | undefined
  className?: string
  /** Max lines in table cell before ellipsis */
  lines?: 2 | 3
  mono?: boolean
}

/**
 * Table cell: show up to `lines` with line-clamp; hover opens popover with full text (pre-wrap, break-words).
 */
export function EllipsisHoverCell({ text, className, lines = 2, mono }: EllipsisHoverCellProps) {
  const raw = text?.trim() ?? ''
  if (!raw) {
    return <span className={cn('text-muted-foreground', className)}>—</span>
  }

  return (
    <Popover
      trigger="hover"
      placement="top"
      className={cn('block w-full min-w-0 max-w-full')}
      contentClassName="max-w-[min(24rem,calc(100vw-2rem))] bg-card border-border shadow-xl"
      content={
        <div className="whitespace-pre-wrap break-words text-sm text-foreground leading-relaxed">
          {raw}
        </div>
      }
    >
      <span
        className={cn(
          'block w-full min-w-0 overflow-hidden text-left',
          lines === 2 && 'line-clamp-2',
          lines === 3 && 'line-clamp-3',
          mono && 'font-mono text-xs break-all',
          !mono && 'break-words',
          className
        )}
      >
        {raw}
      </span>
    </Popover>
  )
}

interface LogHoverBlockProps {
  title: string
  body: string
  hint?: string
}

/** Web 坐席：列表区限制约 8 行，悬停浮层展示全部（可滚动、换行） */
export function LogHoverBlock({ title, body, hint }: LogHoverBlockProps) {
  const raw = body || '—'
  return (
    <div>
      <h3 className="text-sm font-semibold mb-1">{title}</h3>
      {hint ? <p className="text-xs text-muted-foreground mb-1">{hint}</p> : null}
      <Popover
        trigger="hover"
        placement="left"
        className="block w-full"
        contentClassName="max-w-[min(42rem,90vw)] max-h-[70vh] overflow-y-auto bg-card border-border shadow-xl"
        content={
          <pre className="whitespace-pre-wrap break-words font-mono text-xs p-1 text-foreground">
            {raw}
          </pre>
        }
      >
        <pre className="max-h-[9.5rem] cursor-default overflow-hidden text-ellipsis line-clamp-[8] rounded-md border border-border bg-muted/40 p-3 font-mono text-xs">
          {raw}
        </pre>
      </Popover>
    </div>
  )
}
