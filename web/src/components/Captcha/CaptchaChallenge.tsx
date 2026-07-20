import { useCallback, useEffect, useRef, useState } from 'react'
import { Loader2, RefreshCw } from 'lucide-react'
import {
  captchaTypeLabel,
  type CaptchaGenerateResult,
  type CaptchaProof,
} from '@/api/captcha'
import { generateCaptcha } from '@/api/common'
import ClickCaptcha from '@/components/Captcha/ClickCaptcha'
import ImageCaptcha from '@/components/Captcha/ImageCaptcha'
import SliderCaptcha from '@/components/Captcha/SliderCaptcha'
import { useTranslation } from '@/i18n'

type Props = {
  active: boolean
  onChange: (proof: CaptchaProof | null) => void
}

export default function CaptchaChallenge({ active, onChange }: Props) {
  const { t } = useTranslation()
  const onChangeRef = useRef(onChange)
  onChangeRef.current = onChange
  const [session, setSession] = useState<CaptchaGenerateResult | null>(null)
  const [loading, setLoading] = useState(false)
  const [loadError, setLoadError] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    setLoadError('')
    setSession(null)
    onChangeRef.current(null)
    try {
      const res = await generateCaptcha()
      if (res.code === 429) {
        setLoadError(t('captcha.rateLimited'))
        return
      }
      if (res.code !== 200 || !res.data?.id) {
        setLoadError(res.msg || t('captcha.loadFailed'))
        return
      }
      setSession(res.data)
    } catch (e: unknown) {
      const msg = typeof e === 'object' && e && 'msg' in e ? String((e as { msg?: string }).msg) : t('captcha.loadFailed')
      setLoadError(msg)
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    if (active) void load()
  }, [active, load])

  if (loading && !session) {
    return (
      <div className="flex h-32 items-center justify-center gap-2 text-sm text-neutral-500">
        <Loader2 className="h-4 w-4 animate-spin" />
        {t('common.loading')}
      </div>
    )
  }

  if (loadError && !session) {
    return (
      <div className="space-y-3 rounded-xl border border-red-200 bg-red-50 px-4 py-6 text-center">
        <p className="text-sm text-red-600">{loadError}</p>
        <button
          type="button"
          className="inline-flex items-center gap-1 text-sm text-neutral-600 hover:text-neutral-900"
          onClick={() => void load()}
        >
          <RefreshCw className="h-3.5 w-3.5" />
          {t('captcha.refresh')}
        </button>
      </div>
    )
  }

  if (!session) return null

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between gap-2">
        <span className="inline-flex items-center rounded-full bg-neutral-100 px-2.5 py-1 text-xs font-medium text-neutral-600">
          {captchaTypeLabel(session.type, t)}
        </span>
        <button
          type="button"
          className="inline-flex items-center gap-1 text-xs text-neutral-500 hover:text-neutral-900"
          onClick={() => void load()}
        >
          <RefreshCw className="h-3.5 w-3.5" />
          {t('captcha.refresh')}
        </button>
      </div>

      {session.type === 'slider' && (
        <SliderCaptcha session={session} onChange={onChange} onReload={load} />
      )}
      {session.type === 'click' && (
        <ClickCaptcha session={session} onChange={onChange} onReload={load} />
      )}
      {session.type === 'image' && (
        <ImageCaptcha session={session} onChange={onChange} onReload={load} />
      )}
    </div>
  )
}
