import { useEffect, useState } from 'react'
import { Modal } from '@arco-design/web-react'
import { ShieldCheck } from 'lucide-react'
import CaptchaChallenge from '@/components/Captcha/CaptchaChallenge'
import type { CaptchaProof } from '@/api/captcha'
import { useTranslation } from '@/i18n'
import { Button } from '@/components/ui'

type Props = {
  open: boolean
  onClose: () => void
  onVerified: (proof: CaptchaProof) => void
}

export default function CaptchaVerifyModal({ open, onClose, onVerified }: Props) {
  const { t } = useTranslation()
  const [proof, setProof] = useState<CaptchaProof | null>(null)

  useEffect(() => {
    if (!open) {
      setProof(null)
    }
  }, [open])

  useEffect(() => {
    if (!open || !proof?.captchaId) return
    if (proof.captchaType === 'image') return
    const timer = window.setTimeout(() => onVerified(proof), 380)
    return () => window.clearTimeout(timer)
  }, [open, proof, onVerified])

  const isImage = proof?.captchaType === 'image'

  return (
    <Modal
      visible={open}
      title={null}
      footer={null}
      closable
      maskClosable={false}
      unmountOnExit
      onCancel={onClose}
      className="captcha-verify-modal"
      style={{ width: 420, maxWidth: 'calc(100vw - 32px)' }}
    >
      <div className="px-1 pb-1 pt-2">
        <div className="mb-5 flex items-start gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-neutral-900 text-white">
            <ShieldCheck className="h-5 w-5" strokeWidth={2} />
          </div>
          <div>
            <h3 className="text-base font-semibold text-neutral-900">{t('captcha.modalTitle')}</h3>
            <p className="mt-1 text-sm leading-relaxed text-neutral-500">{t('captcha.modalDescRandom')}</p>
          </div>
        </div>

        <CaptchaChallenge active={open} onChange={setProof} />

        {isImage ? (
          <div className="mt-5 flex justify-end gap-2">
            <Button type="outline" onClick={onClose}>
              {t('common.cancel')}
            </Button>
            <Button
              type="primary"
              disabled={!proof?.captchaId}
              onClick={() => proof && onVerified(proof)}
            >
              {t('captcha.confirm')}
            </Button>
          </div>
        ) : null}
      </div>
    </Modal>
  )
}
