import { useRef } from 'react'
import { Modal, Select, Typography } from '@arco-design/web-react'
import { Input, Button } from '@/components/ui'
import JSTemplateAvatar from '@/components/js-template/JSTemplateAvatar'
import { useTranslation } from '@/i18n'

export type JSTemplateSettingsModalProps = {
  visible: boolean
  name: string
  usage: string
  status: string
  avatarUrl?: string
  jsSourceId?: string
  avatarUploading?: boolean
  onChange: (patch: { name?: string; usage?: string; status?: string; avatarUrl?: string }) => void
  onAvatarPick: (file: File) => void
  onClose: () => void
  onConfirm: () => void
}

export default function JSTemplateSettingsModal({
  visible,
  name,
  usage,
  status,
  avatarUrl,
  jsSourceId,
  avatarUploading,
  onChange,
  onAvatarPick,
  onClose,
  onConfirm,
}: JSTemplateSettingsModalProps) {
  const { t } = useTranslation()
  const fileRef = useRef<HTMLInputElement>(null)

  const pickFile = () => {
    if (!avatarUploading) fileRef.current?.click()
  }

  return (
    <Modal
      visible={visible}
      title={t('jsTemplate.settingsTitle')}
      onCancel={onClose}
      footer={
        <>
          <Button onClick={onClose}>{t('common.cancel')}</Button>
          <Button type="primary" onClick={onConfirm}>
            {t('common.confirm')}
          </Button>
        </>
      }
      style={{ width: 520 }}
    >
      <div className="space-y-4 py-1">
        <div className="flex items-start gap-4">
          <button type="button" className="shrink-0 rounded-xl focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/40" onClick={pickFile}>
            <JSTemplateAvatar src={avatarUrl} name={name} size="xl" />
          </button>
          <div className="flex min-w-0 flex-1 flex-col gap-2">
            <Typography.Text className="!text-xs font-medium">{t('jsTemplate.avatar')}</Typography.Text>
            <Button type="outline" size="small" loading={avatarUploading} onClick={pickFile}>
              {avatarUploading ? t('jsTemplate.avatarUploading') : t('jsTemplate.avatarUpload')}
            </Button>
            <input
              ref={fileRef}
              type="file"
              accept="image/*"
              className="hidden"
              onChange={(e) => {
                const f = e.target.files?.[0]
                if (f) onAvatarPick(f)
                e.target.value = ''
              }}
            />
            <Typography.Text type="secondary" className="!text-[11px] leading-relaxed">
              {t('jsTemplate.avatarHint')}
            </Typography.Text>
          </div>
        </div>
        <div>
          <Typography.Text className="!text-xs">{t('jsTemplate.name')}</Typography.Text>
          <Input
            value={name}
            onChange={(v) => onChange({ name: v })}
            placeholder={t('jsTemplate.namePlaceholder')}
          />
        </div>
        <div>
          <Typography.Text className="!text-xs">{t('jsTemplate.usage')}</Typography.Text>
          <Input value={usage} onChange={(v) => onChange({ usage: v })} placeholder={t('jsTemplate.usagePlaceholder')} />
        </div>
        <div>
          <Typography.Text className="!text-xs">{t('jsTemplate.status')}</Typography.Text>
          <Select
            value={status}
            onChange={(v) => onChange({ status: v })}
            options={[
              { value: 'active', label: t('jsTemplate.statusActive') },
              { value: 'draft', label: t('jsTemplate.statusDraft') },
            ]}
            style={{ width: '100%' }}
          />
        </div>
        {jsSourceId ? (
          <div>
            <Typography.Text className="!text-xs">{t('jsTemplate.sourceId')}</Typography.Text>
            <Input value={jsSourceId} readOnly />
          </div>
        ) : null}
        <Typography.Text type="secondary" className="!text-xs block leading-relaxed">
          {t('jsTemplate.hint')}
        </Typography.Text>
      </div>
    </Modal>
  )
}
