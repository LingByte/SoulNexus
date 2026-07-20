import { useMemo } from 'react'
import { cn } from '@/utils/cn'

export type LineDiffKind = 'same' | 'add' | 'del' | 'change'

export type LineDiff = {
  left?: string
  right?: string
  kind: LineDiffKind
}

/** Line-oriented LCS diff for side-by-side JSON / text comparison. */
export function buildLineDiff(leftRaw: string, rightRaw: string): LineDiff[] {
  const a = leftRaw.replace(/\r\n/g, '\n').split('\n')
  const b = rightRaw.replace(/\r\n/g, '\n').split('\n')
  const n = a.length
  const m = b.length
  const dp: number[][] = Array.from({ length: n + 1 }, () => Array(m + 1).fill(0))
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      if (a[i] === b[j]) dp[i][j] = dp[i + 1][j + 1] + 1
      else dp[i][j] = Math.max(dp[i + 1][j], dp[i][j + 1])
    }
  }
  const out: LineDiff[] = []
  let i = 0
  let j = 0
  while (i < n && j < m) {
    if (a[i] === b[j]) {
      out.push({ left: a[i], right: b[j], kind: 'same' })
      i++
      j++
    } else if (dp[i + 1][j] >= dp[i][j + 1]) {
      out.push({ left: a[i], kind: 'del' })
      i++
    } else {
      out.push({ right: b[j], kind: 'add' })
      j++
    }
  }
  while (i < n) {
    out.push({ left: a[i++], kind: 'del' })
  }
  while (j < m) {
    out.push({ right: b[j++], kind: 'add' })
  }
  return out
}

function pretty(raw: unknown): string {
  if (raw == null || raw === '') return ''
  if (typeof raw === 'string') {
    try {
      return JSON.stringify(JSON.parse(raw), null, 2)
    } catch {
      return raw
    }
  }
  try {
    return JSON.stringify(raw, null, 2)
  } catch {
    return String(raw)
  }
}

type Props = {
  left: unknown
  right: unknown
  leftTitle?: string
  rightTitle?: string
  className?: string
}

/** VSCode-like side-by-side textual diff. */
export default function SideBySideDiff({ left, right, leftTitle = 'A', rightTitle = 'B', className }: Props) {
  const leftText = useMemo(() => pretty(left), [left])
  const rightText = useMemo(() => pretty(right), [right])
  const rows = useMemo(() => buildLineDiff(leftText, rightText), [leftText, rightText])

  return (
    <div className={cn('overflow-hidden rounded-md border border-border', className)}>
      <div className="grid grid-cols-2 border-b border-border bg-muted/40 text-xs font-medium">
        <div className="border-r border-border px-3 py-2">{leftTitle}</div>
        <div className="px-3 py-2">{rightTitle}</div>
      </div>
      <div className="max-h-[min(70vh,560px)] overflow-auto font-mono text-[11px] leading-5">
        {rows.length === 0 ? (
          <div className="px-3 py-6 text-center text-muted-foreground">无差异</div>
        ) : (
          rows.map((r, idx) => (
            <div key={idx} className="grid grid-cols-2 border-b border-border/40 last:border-b-0">
              <div
                className={cn(
                  'whitespace-pre-wrap break-all border-r border-border/40 px-2 py-0.5',
                  r.kind === 'del' && 'bg-red-500/10 text-red-700 dark:text-red-300',
                  r.kind === 'same' && 'text-foreground/80',
                  r.kind === 'add' && 'bg-transparent text-transparent',
                )}
              >
                {r.left ?? '\u00a0'}
              </div>
              <div
                className={cn(
                  'whitespace-pre-wrap break-all px-2 py-0.5',
                  r.kind === 'add' && 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-300',
                  r.kind === 'same' && 'text-foreground/80',
                  r.kind === 'del' && 'bg-transparent text-transparent',
                )}
              >
                {r.right ?? '\u00a0'}
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  )
}
