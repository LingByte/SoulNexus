import { useEffect, useRef } from 'react'
import { cn } from '@/utils/cn'

type Accent = 'signal' | 'rx'

const accentStyles: Record<
  Accent,
  { border: string; glow: string; text: string; bar: string; dot: string }
> = {
  signal: {
    border: 'border-emerald-500/35',
    glow: 'shadow-[0_0_24px_rgba(16,185,129,0.12),inset_0_1px_0_rgba(16,185,129,0.08)]',
    text: 'text-emerald-400/95',
    bar: 'border-emerald-500/25 bg-emerald-950/40',
    dot: 'bg-emerald-400',
  },
  rx: {
    border: 'border-cyan-500/35',
    glow: 'shadow-[0_0_24px_rgba(34,211,238,0.12),inset_0_1px_0_rgba(34,211,238,0.08)]',
    text: 'text-cyan-300/95',
    bar: 'border-cyan-500/25 bg-cyan-950/40',
    dot: 'bg-cyan-400',
  },
}

/**
 * Sci‑fi console: monospace stream, scanlines, auto-scroll. Replaces plain log blocks on Web 坐席 tab.
 */
export function WebSeatTerminalLog({
  title,
  body,
  hint,
  accent = 'signal',
  className,
}: {
  title: string
  body: string
  hint?: string
  accent?: Accent
  className?: string
}) {
  const preRef = useRef<HTMLPreElement>(null)
  const a = accentStyles[accent]

  useEffect(() => {
    const el = preRef.current
    if (!el) return
    el.scrollTop = el.scrollHeight
  }, [body])

  const display = (body || '').trim() || '—'

  return (
    <div
      className={cn(
        'relative overflow-hidden rounded-lg border bg-[#070b10]',
        a.border,
        a.glow,
        className
      )}
    >
      {/* Scanlines */}
      <div
        className="pointer-events-none absolute inset-0 z-[1] opacity-[0.45]"
        style={{
          backgroundImage:
            'repeating-linear-gradient(0deg, transparent, transparent 2px, rgba(255,255,255,0.025) 2px, rgba(255,255,255,0.025) 4px)',
        }}
      />
      {/* Vignette */}
      <div className="pointer-events-none absolute inset-0 z-[1] bg-[radial-gradient(ellipse_at_center,transparent_40%,rgba(0,0,0,0.55)_100%)]" />

      <div className={cn('relative z-[2] flex items-center gap-2 border-b px-3 py-2', a.bar)}>
        <span className="flex gap-1.5">
          <span className="h-2.5 w-2.5 rounded-full bg-red-500/90 shadow-[0_0_6px_rgba(239,68,68,0.6)]" />
          <span className="h-2.5 w-2.5 rounded-full bg-amber-400/90 shadow-[0_0_6px_rgba(251,191,36,0.5)]" />
          <span className="h-2.5 w-2.5 rounded-full bg-emerald-500/80 shadow-[0_0_6px_rgba(34,197,94,0.45)]" />
        </span>
        <span
          className={cn(
            'font-mono text-[10px] font-semibold uppercase tracking-[0.2em] text-white/50',
            accent === 'rx' && 'text-cyan-200/45'
          )}
        >
          {title}
        </span>
        <span className="ml-auto flex items-center gap-1.5 font-mono text-[9px] text-white/35">
          <span className={cn('h-1 w-1 animate-pulse rounded-full', a.dot)} />
          STREAM
        </span>
      </div>

      {hint ? (
        <p className="relative z-[2] border-b border-white/5 px-3 py-1.5 font-mono text-[10px] leading-snug text-white/40">
          <span className="text-white/25"># </span>
          {hint}
        </p>
      ) : null}

      <pre
        ref={preRef}
        className={cn(
          'relative z-[2] max-h-[min(14rem,42vh)] overflow-x-auto overflow-y-auto px-3 py-2.5 font-mono text-[11px] leading-relaxed',
          '[&::-webkit-scrollbar]:h-1.5 [&::-webkit-scrollbar]:w-1.5',
          '[&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-thumb]:bg-white/15',
          '[&::-webkit-scrollbar-track]:bg-transparent',
          a.text,
          'selection:bg-white/20 selection:text-white'
        )}
        style={{
          textShadow:
            accent === 'rx'
              ? '0 0 14px rgba(34,211,238,0.2)'
              : '0 0 14px rgba(16,185,129,0.18)',
        }}
      >
        {display}
      </pre>

      {/* Cursor blink line */}
      <div className="relative z-[2] border-t border-white/5 px-3 py-1 font-mono text-[10px] text-white/25">
        <span className={cn(accent === 'rx' ? 'text-cyan-500/70' : 'text-emerald-500/65')}>{'>'}</span>{' '}
        <span
          className={cn(
            'inline-block h-3 w-2 animate-pulse opacity-80',
            accent === 'rx' ? 'bg-cyan-400' : 'bg-emerald-400'
          )}
        />
      </div>
    </div>
  )
}
