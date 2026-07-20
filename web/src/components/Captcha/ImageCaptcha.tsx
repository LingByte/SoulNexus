import { useEffect, useRef, useState } from 'react'
import { Input } from '@/components/ui'
import type { CaptchaGenerateResult, CaptchaProof } from '@/api/captcha'
import { useTranslation } from '@/i18n'

type Props = {
  session: CaptchaGenerateResult
  onChange: (proof: CaptchaProof | null) => void
  onReload?: () => void | Promise<void>
}

export default function ImageCaptcha({ session, onChange }: Props) {
  const { t } = useTranslation()
  const onChangeRef = useRef(onChange)
  onChangeRef.current = onChange
  const [code, setCode] = useState('')
  const length = session.data.length ?? 4

  useEffect(() => {
    setCode('')
    onChangeRef.current(null)
  }, [session.id])

  const handleChange = (value: string) => {
    const next = value.trim()
    setCode(next)
    if (next.length >= length) {
      onChangeRef.current({
        captchaId: session.id,
        captchaType: 'image',
        captchaValue: next,
      })
      return
    }
    onChangeRef.current(null)
  }

  return (
    <div className="space-y-3">
      <p className="text-sm text-neutral-600">{t('captcha.imageHint')}</p>
      {session.data.image ? (
        <div className="overflow-hidden rounded-xl border border-neutral-200 bg-neutral-50 p-2">
          <img
            src={session.data.image}
            alt={t('captcha.typeImage')}
            className="mx-auto block h-[60px] w-full max-w-[240px] object-contain"
            draggable={false}
          />
        </div>
      ) : null}
      <Input
        size="lg"
        variant="filled"
        value={code}
        maxLength={length + 2}
        placeholder={t('captcha.imagePlaceholder', { length: String(length) })}
        autoComplete="off"
        onChange={(v) => handleChange(String(v))}
      />
    </div>
  )
}
