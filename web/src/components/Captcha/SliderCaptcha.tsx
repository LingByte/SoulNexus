import { useCallback, useEffect, useRef, useState } from 'react'
import { Check, ChevronsRight, Loader2 } from 'lucide-react'
import type { CaptchaGenerateResult, CaptchaProof } from '@/api/captcha'
import { generateCaptcha } from '@/api/common'
import { useTranslation } from '@/i18n'

const HANDLE_SIZE = 46
const PASS_RATIO = 0.92

type Props = {
  onChange: (proof: CaptchaProof | null) => void
  autoLoad?: boolean
  session?: CaptchaGenerateResult
  onReload?: () => void | Promise<void>
}

export default function SliderCaptcha({ onChange, autoLoad = true, session, onReload }: Props) {
  const { t } = useTranslation()
  const trackRef = useRef<HTMLDivElement>(null)
  const draggingRef = useRef(false)
  const offsetRef = useRef(0)
  const onChangeRef = useRef(onChange)
  onChangeRef.current = onChange

  const [loading, setLoading] = useState(false)
  const [loadError, setLoadError] = useState('')
  const [captchaId, setCaptchaId] = useState('')
  const [trackWidth, setTrackWidth] = useState(300)
  const [offset, setOffset] = useState(0)
  const [dragging, setDragging] = useState(false)
  const [passed, setPassed] = useState(false)
  const [failed, setFailed] = useState(false)

  const setOffsetSafe = useCallback((next: number) => {
    offsetRef.current = next
    setOffset(next)
  }, [])

  const resetInteractive = useCallback(() => {
    setOffsetSafe(0)
    setPassed(false)
    setFailed(false)
    onChangeRef.current(null)
  }, [setOffsetSafe])

  const applySession = useCallback(
    (next: CaptchaGenerateResult) => {
      resetInteractive()
      setLoadError('')
      setCaptchaId(next.id)
      setTrackWidth(next.data.trackWidth ?? 300)
      setLoading(false)
    },
    [resetInteractive],
  )

  const load = useCallback(async () => {
    if (onReload) {
      await onReload()
      return
    }
    setLoading(true)
    setLoadError('')
    resetInteractive()
    try {
      const res = await generateCaptcha('slider')
      if (res.code === 429) {
        setLoadError(t('captcha.rateLimited'))
        return
      }
      if (res.code !== 200 || !res.data?.id) {
        setLoadError(res.msg || t('captcha.loadFailed'))
        return
      }
      applySession(res.data)
    } catch (e: unknown) {
      const msg = typeof e === 'object' && e && 'msg' in e ? String((e as { msg?: string }).msg) : t('captcha.loadFailed')
      setLoadError(msg)
    } finally {
      setLoading(false)
    }
  }, [applySession, onReload, resetInteractive, t])

  useEffect(() => {
    if (session) {
      applySession(session)
      return
    }
    if (autoLoad) void load()
    // Only re-apply when captcha session id changes; avoid onChange identity loops.
    // eslint-disable-next-line react-hooks/exhaustive-deps -- intentional
  }, [session?.id, autoLoad])

  const updateOffsetFromClientX = useCallback(
    (clientX: number) => {
      const rect = trackRef.current?.getBoundingClientRect()
      if (!rect) return
      const max = Math.max(0, rect.width - HANDLE_SIZE)
      const next = Math.min(max, Math.max(0, clientX - rect.left - HANDLE_SIZE / 2))
      setOffsetSafe(next)
    },
    [setOffsetSafe],
  )

  const submitProof = useCallback(
    (nextOffset: number) => {
      const rect = trackRef.current?.getBoundingClientRect()
      if (!rect || !captchaId) return
      const max = Math.max(1, rect.width - HANDLE_SIZE)
      const ratio = (nextOffset + HANDLE_SIZE) / rect.width
      if (ratio >= PASS_RATIO) {
        setPassed(true)
        setFailed(false)
        setOffsetSafe(max)
        const scaled = Math.min(trackWidth, Math.max(0, Math.round(ratio * trackWidth)))
        onChangeRef.current({
          captchaId,
          captchaType: 'slider',
          captchaValue: scaled,
        })
        return
      }
      setFailed(true)
      setOffsetSafe(0)
      onChangeRef.current(null)
      window.setTimeout(() => setFailed(false), 450)
    },
    [captchaId, setOffsetSafe, trackWidth],
  )

  useEffect(() => {
    const onMove = (e: PointerEvent) => {
      if (!draggingRef.current) return
      updateOffsetFromClientX(e.clientX)
    }
    const onUp = () => {
      if (!draggingRef.current) return
      draggingRef.current = false
      setDragging(false)
      submitProof(offsetRef.current)
    }
    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
    window.addEventListener('pointercancel', onUp)
    return () => {
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onUp)
      window.removeEventListener('pointercancel', onUp)
    }
  }, [submitProof, updateOffsetFromClientX])

  const handlePointerDown = (e: React.PointerEvent) => {
    if (loading || passed || loadError || !captchaId) return
    e.preventDefault()
    draggingRef.current = true
    setDragging(true)
    setFailed(false)
    updateOffsetFromClientX(e.clientX)
  }

  const maxOffset = () => {
    const rect = trackRef.current?.getBoundingClientRect()
    if (!rect) return 0
    return Math.max(0, rect.width - HANDLE_SIZE)
  }

  const progress = (() => {
    const max = maxOffset()
    if (max <= 0) return passed ? 100 : 0
    return Math.min(100, Math.round((offset / max) * 100))
  })()

  return (
    <div className="captcha-slider-root select-none">
      {loadError ? <p className="mb-2 text-xs text-red-500">{loadError}</p> : null}
      <div
        ref={trackRef}
        className={`captcha-slider-track ${failed ? 'captcha-slider-track--fail' : ''} ${passed ? 'captcha-slider-track--pass' : ''}`}
      >
        <div
          className="captcha-slider-fill"
          style={{
            width: passed ? '100%' : offset + HANDLE_SIZE,
            transition: dragging ? 'none' : 'width 0.25s cubic-bezier(0.4, 0, 0.2, 1)',
          }}
        />
        <span
          className="captcha-slider-hint"
          style={{ opacity: passed ? 0 : Math.max(0.25, 1 - progress / 100) }}
        >
          {loading ? t('common.loading') : passed ? t('captcha.sliderPassed') : t('captcha.sliderHint')}
        </span>
        <button
          type="button"
          aria-label={t('captcha.sliderHint')}
          disabled={loading || passed || Boolean(loadError) || !captchaId}
          className={`captcha-slider-handle ${dragging ? 'captcha-slider-handle--drag' : ''}`}
          style={{
            width: HANDLE_SIZE,
            height: HANDLE_SIZE,
            transform: `translateX(${offset}px)`,
            transition: dragging ? 'none' : 'transform 0.25s cubic-bezier(0.4, 0, 0.2, 1)',
          }}
          onPointerDown={handlePointerDown}
        >
          {loading ? (
            <Loader2 className="h-5 w-5 animate-spin text-neutral-400" />
          ) : passed ? (
            <Check className="h-5 w-5 text-emerald-600" strokeWidth={2.5} />
          ) : (
            <ChevronsRight className="h-5 w-5 text-neutral-600" strokeWidth={2} />
          )}
        </button>
      </div>
      {!session ? (
        <p className="mt-2 text-xs text-neutral-400">
          {passed ? t('captcha.sliderPassed') : t('captcha.sliderSubHint')}
        </p>
      ) : null}
    </div>
  )
}
