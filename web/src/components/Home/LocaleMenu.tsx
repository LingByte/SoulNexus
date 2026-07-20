import { useState, useRef, useEffect } from 'react'
import { Globe, Check } from 'lucide-react'
import { useLocaleStore, type AppLocale } from '@/stores/localeStore'
import { useTranslation } from '@/i18n'
import { cn } from '@/utils/cn'

const LOCALES: { value: AppLocale; labelKey: 'locale.zh' | 'locale.tw' | 'locale.en' | 'locale.ja' }[] = [
  { value: 'zh-CN', labelKey: 'locale.zh' },
  { value: 'zh-TW', labelKey: 'locale.tw' },
  { value: 'en-US', labelKey: 'locale.en' },
  { value: 'ja-JP', labelKey: 'locale.ja' },
]

type LocaleMenuProps = {
  className?: string
  align?: 'left' | 'right'
}

export default function LocaleMenu({ className, align = 'right' }: LocaleMenuProps) {
  const { t } = useTranslation()
  const locale = useLocaleStore((s) => s.locale)
  const setLocale = useLocaleStore((s) => s.setLocale)
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    return () => document.removeEventListener('mousedown', onDoc)
  }, [open])

  const current = LOCALES.find((l) => l.value === locale)

  return (
    <div ref={ref} className={cn('relative', className)}>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="inline-flex h-9 items-center gap-1.5 rounded-lg border border-[hsl(var(--border))] bg-[hsl(var(--background)/0.8)] px-2.5 text-sm text-[hsl(var(--foreground))] shadow-sm transition hover:border-violet-400/60 hover:bg-[hsl(var(--muted)/0.5)]"
        aria-expanded={open}
        aria-haspopup="listbox"
      >
        <Globe className="h-4 w-4 text-violet-600 dark:text-violet-400" />
        <span className="max-w-[4.5rem] truncate font-medium">{current ? t(current.labelKey) : locale}</span>
      </button>
      {open ? (
        <ul
          role="listbox"
          className={cn(
            'absolute top-[calc(100%+6px)] z-50 min-w-[140px] overflow-hidden rounded-xl border border-[hsl(var(--border))] bg-[hsl(var(--popover))] py-1 shadow-lg',
            align === 'right' ? 'right-0' : 'left-0',
          )}
        >
          {LOCALES.map((item) => {
            const active = item.value === locale
            return (
              <li key={item.value}>
                <button
                  type="button"
                  role="option"
                  aria-selected={active}
                  className={cn(
                    'flex w-full items-center justify-between gap-2 px-3 py-2 text-left text-sm transition',
                    active
                      ? 'bg-violet-50 text-violet-800 dark:bg-violet-950/50 dark:text-violet-200'
                      : 'text-[hsl(var(--foreground))] hover:bg-[hsl(var(--muted))]',
                  )}
                  onClick={() => {
                    setLocale(item.value)
                    setOpen(false)
                  }}
                >
                  {t(item.labelKey)}
                  {active ? <Check className="h-4 w-4 shrink-0 text-violet-600" /> : null}
                </button>
              </li>
            )
          })}
        </ul>
      ) : null}
    </div>
  )
}
