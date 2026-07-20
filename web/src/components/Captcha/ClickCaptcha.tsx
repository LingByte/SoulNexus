import { useCallback, useEffect, useRef, useState } from 'react'
import type { CaptchaGenerateResult, CaptchaCharMarker, CaptchaProof } from '@/api/captcha'
import { useTranslation } from '@/i18n'

type Props = {
  session: CaptchaGenerateResult
  onChange: (proof: CaptchaProof | null) => void
  onReload?: () => void | Promise<void>
}

export default function ClickCaptcha({ session, onChange }: Props) {
  const { t } = useTranslation()
  const boardRef = useRef<HTMLDivElement>(null)
  const onChangeRef = useRef(onChange)
  onChangeRef.current = onChange
  const [width, setWidth] = useState(300)
  const [height, setHeight] = useState(200)
  const [background, setBackground] = useState('')
  const [targets, setTargets] = useState<string[]>([])
  const [chars, setChars] = useState<CaptchaCharMarker[]>([])
  const [clicks, setClicks] = useState<Array<{ x: number; y: number }>>([])
  const [done, setDone] = useState(false)

  const applySession = useCallback((next: CaptchaGenerateResult) => {
    setWidth(next.data.width ?? 300)
    setHeight(next.data.height ?? 200)
    setBackground(next.data.background ?? '')
    setTargets(next.data.targets ?? [])
    setChars(next.data.chars ?? [])
    setClicks([])
    setDone(false)
    onChangeRef.current(null)
  }, [])

  useEffect(() => {
    applySession(session)
  }, [session.id, applySession])

  const handleBoardClick = (e: React.MouseEvent<HTMLDivElement>) => {
    if (done || !boardRef.current) return
    const rect = boardRef.current.getBoundingClientRect()
    const scaleX = width / Math.max(1, rect.width)
    const scaleY = height / Math.max(1, rect.height)
    const x = Math.round((e.clientX - rect.left) * scaleX)
    const y = Math.round((e.clientY - rect.top) * scaleY)
    const next = [...clicks, { x, y }]
    setClicks(next)
    if (next.length >= targets.length) {
      setDone(true)
      onChangeRef.current({
        captchaId: session.id,
        captchaType: 'click',
        captchaValue: next,
      })
    }
  }

  const targetHint = targets.join(' → ')

  return (
    <div className="space-y-2">
      <p className="text-sm text-neutral-600">{t('captcha.clickHint', { sequence: targetHint })}</p>
      <div
        ref={boardRef}
        role="button"
        tabIndex={0}
        onClick={handleBoardClick}
        className="captcha-click-board relative overflow-hidden rounded-xl border border-neutral-200"
        style={{
          width: '100%',
          maxWidth: width,
          aspectRatio: `${width} / ${height}`,
          margin: '0 auto',
          backgroundImage: background ? `url(${background})` : undefined,
          backgroundSize: 'cover',
          backgroundPosition: 'center',
          backgroundColor: background ? undefined : 'rgb(245 245 244)',
        }}
      >
        {chars.map((item, idx) => (
          <span
            key={`${item.char}-${idx}`}
            className="pointer-events-none absolute select-none text-2xl font-semibold text-neutral-800 drop-shadow-[0_1px_1px_rgba(255,255,255,0.9)]"
            style={{ left: `${(item.x / width) * 100}%`, top: `${(item.y / height) * 100}%` }}
          >
            {item.char}
          </span>
        ))}
        {clicks.map((p, idx) => (
          <span
            key={`${p.x}-${p.y}-${idx}`}
            className="pointer-events-none absolute flex h-6 w-6 -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full bg-neutral-900/85 text-xs font-medium text-white shadow-sm"
            style={{ left: `${(p.x / width) * 100}%`, top: `${(p.y / height) * 100}%` }}
          >
            {idx + 1}
          </span>
        ))}
      </div>
      <p className="text-xs text-neutral-500">
        {done ? t('captcha.clickDone') : t('captcha.clickProgress', { current: clicks.length, total: targets.length })}
      </p>
    </div>
  )
}
