import { Popover, Typography } from '@arco-design/web-react'
import { Copy } from 'lucide-react'
import { cn } from '@/utils/cn'
import { showAlert } from '@/utils/notification'

type Props = {
  value?: unknown
  /** Max visible chars before ellipsis (monospace ids). */
  maxChars?: number
  className?: string
  copyable?: boolean
}

/** Compact ID cell with hover full value + optional copy (platform admin tables). */
export function CompactIdCell({ value, maxChars = 12, className, copyable = true }: Props) {
  const raw = value == null || value === '' ? '' : String(value).trim()
  if (!raw) return <span className="text-muted-foreground">—</span>

  const short =
    raw.length <= maxChars ? raw : `${raw.slice(0, Math.max(4, maxChars - 1))}…`

  const copy = async (e: React.MouseEvent) => {
    e.stopPropagation()
    try {
      await navigator.clipboard.writeText(raw)
      showAlert('已复制', 'success')
    } catch {
      showAlert('复制失败', 'error')
    }
  }

  return (
    <Popover
      content={
        <div className="max-w-xs break-all font-mono text-xs">
          <Typography.Text copyable={{ text: raw }}>{raw}</Typography.Text>
        </div>
      }
    >
      <span className={cn('inline-flex max-w-full items-center gap-1 font-mono text-xs', className)}>
        <span className="min-w-0 truncate">{short}</span>
        {copyable && raw.length > 0 ? (
          <button
            type="button"
            className="shrink-0 rounded p-0.5 text-muted-foreground hover:bg-muted hover:text-foreground"
            onClick={(e) => void copy(e)}
            aria-label="copy id"
          >
            <Copy className="h-3 w-3" />
          </button>
        ) : null}
      </span>
    </Popover>
  )
}
