import { Modal, Select, Typography } from '@arco-design/web-react'
import { Input } from '@/components/ui'
import { Button } from '@/components/ui'
import { AssistantAvatarPicker } from '@/components/assistant/AssistantFileUploadModal'
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
  onAvatarPick?: (file: File) => void
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
        <div className="flex items-center gap-4">
          <JSTemplateAvatar src={avatarUrl} name={name} size="xl" />
          {onAvatarPick ? (
            <div className="flex flex-col gap-2">
              <Typography.Text className="!text-xs">{t('jsTemplate.avatar')}</Typography.Text>
              <AssistantAvatarPicker avatarUrl={avatarUrl} onPick={onAvatarPick} />
              {avatarUploading ? (
                <Typography.Text type="secondary" className="!text-[11px]">
                  {t('jsTemplate.avatarUploading')}
                </Typography.Text>
              ) : null}
              <Typography.Text type="secondary" className="!text-[11px]">
                {t('jsTemplate.avatarHint')}
              </Typography.Text>
            </div>
          ) : (
            <div className="flex-1">
              <Typography.Text className="!text-xs">{t('jsTemplate.avatar')}</Typography.Text>
              <Input
                value={avatarUrl || ''}
                onChange={(v) => onChange({ avatarUrl: v })}
                placeholder={t('jsTemplate.avatarUrlPlaceholder')}
              />
            </div>
          )}
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
          <Input value={usage} onChange={(v) => onChange({ usage: v })} />
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
