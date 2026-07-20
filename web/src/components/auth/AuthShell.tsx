import { ReactNode } from 'react'
import { IconMoon, IconSun } from '@arco-design/web-react/icon'
import { useThemeStore } from '@/stores/themeStore'
import { useLocaleStore } from '@/stores/localeStore'
import { useTranslation } from '@/i18n'
import { Button, Select } from '@/components/ui'

interface AuthShellProps {
  title: string
  subtitle?: string
  children: ReactNode
  footer?: ReactNode
}

export default function AuthShell({ title, subtitle, children, footer }: AuthShellProps) {
  const { toggleMode, isDark } = useThemeStore()
  const { t } = useTranslation()
  const locale = useLocaleStore((s) => s.locale)
  const setLocale = useLocaleStore((s) => s.setLocale)

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '32px 16px',
        boxSizing: 'border-box',
        position: 'relative',
        overflow: 'hidden',
        background: isDark
          ? 'linear-gradient(145deg, #0b1220 0%, #111827 38%, #0f2744 72%, #0d3a7a 100%)'
          : 'linear-gradient(145deg, #eff6ff 0%, #dbeafe 35%, #e0f2fe 70%, #f0f9ff 100%)',
      }}
    >
      <div
        aria-hidden
        style={{
          position: 'absolute',
          inset: 0,
          pointerEvents: 'none',
          background: isDark
            ? 'radial-gradient(ellipse 80% 55% at 15% 20%, rgba(22,113,239,0.22), transparent 55%), radial-gradient(ellipse 70% 50% at 85% 75%, rgba(56,189,248,0.14), transparent 50%)'
            : 'radial-gradient(ellipse 75% 50% at 12% 18%, rgba(22,113,239,0.18), transparent 52%), radial-gradient(ellipse 65% 45% at 88% 78%, rgba(56,189,248,0.12), transparent 48%)',
        }}
      />
      <div
        aria-hidden
        style={{
          position: 'absolute',
          inset: 0,
          pointerEvents: 'none',
          opacity: isDark ? 0.07 : 0.045,
          backgroundImage: `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='48' height='48' viewBox='0 0 48 48'%3E%3Cg fill='none' stroke='%2394a3b8' stroke-width='0.6'%3E%3Cpath d='M0 24h48M24 0v48'/%3E%3C/g%3E%3C/svg%3E")`,
          backgroundSize: '48px 48px',
        }}
      />
      <div style={{ position: 'fixed', top: 20, right: 20, zIndex: 3 }}>
        <Select
          size="small"
          allowClear={false}
          value={locale}
          style={{ width: 98, marginRight: 8 }}
          options={[
            { value: 'zh-CN', label: t('locale.zh') },
            { value: 'zh-TW', label: t('locale.tw') },
            { value: 'en-US', label: t('locale.en') },
            { value: 'ja-JP', label: t('locale.ja') },
          ]}
          onChange={(v) => {
            if (!v) return
            setLocale(v as 'zh-CN' | 'zh-TW' | 'en-US' | 'ja-JP')
          }}
        />
        <Button
          type="secondary"
          size="small"
          icon={isDark ? <IconSun /> : <IconMoon />}
          onClick={toggleMode}
        />
      </div>
      <div
        style={{
          width: '100%',
          maxWidth: 420,
          borderRadius: 12,
          border: '1px solid var(--color-border)',
          background: isDark ? 'rgba(15,23,42,0.72)' : 'rgba(255,255,255,0.82)',
          backdropFilter: 'blur(14px)',
          WebkitBackdropFilter: 'blur(14px)',
          boxShadow: isDark ? '0 24px 64px rgba(0,0,0,0.35)' : '0 12px 48px rgba(15, 23, 42, 0.08)',
          padding: '40px 36px 36px',
          boxSizing: 'border-box',
          position: 'relative',
          zIndex: 2,
        }}
      >
        <div style={{ marginBottom: 28 }}>
          <h1
            style={{
              margin: 0,
              fontSize: 22,
              fontWeight: 600,
              letterSpacing: '-0.02em',
              color: 'var(--color-text-1)',
            }}
          >
            {title}
          </h1>
          {subtitle ? (
            <p style={{ margin: '10px 0 0', fontSize: 14, color: 'var(--color-text-3)', lineHeight: 1.5 }}>
              {subtitle}
            </p>
          ) : null}
        </div>
        {children}
        {footer ? <div style={{ marginTop: 24 }}>{footer}</div> : null}
      </div>
    </div>
  )
}
