import { useEffect, useState } from 'react'
import { Collapse, Drawer, Space, Switch, Typography } from '@arco-design/web-react'
import { Button, Input, Select } from '@/components/ui'
import { useTranslation } from '@/i18n'
import {
  type AiTab,
  defaultDraft,
  draftToPayload,
  mergeDraft,
  providerRulesFor,
  ruleFor,
  validateDraft,
} from '@/constants/tenantAiConfigRules'
import {
  createAIProviderPool,
  updateAIProviderPool,
  type AIProviderPoolRow,
} from '@/api/platformAiPools'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'

const modalities = ['asr', 'tts', 'llm', 'realtime'] as const
export type PoolModality = (typeof modalities)[number]

function modalityToTab(m: PoolModality): AiTab {
  return m
}

function renderProviderFields(
  tab: AiTab,
  draft: Record<string, unknown>,
  setDraft: (next: Record<string, unknown>) => void,
  providerLabel: string,
) {
  const provider = String(draft.provider ?? '')
  const def = ruleFor(tab, provider)
  const opts = providerRulesFor(tab).map((x) => ({ value: x.provider, label: x.label }))
  return (
    <>
      <div>
        <Typography.Text style={{ display: 'block', marginBottom: 6, fontSize: 12 }}>
          {providerLabel}
        </Typography.Text>
        <Select
          style={{ width: '100%' }}
          value={provider || undefined}
          options={opts}
          onChange={(v) => setDraft({ ...defaultDraft(tab), provider: String(v) })}
        />
      </div>
      {def?.fields.map((f) => (
        <div key={f.key}>
          <Typography.Text style={{ display: 'block', marginBottom: 6, fontSize: 12 }}>
            {f.label}
            {f.required ? ' *' : ''}
          </Typography.Text>
          {f.type === 'password' ? (
            <Input.Password
              autoComplete="new-password"
              placeholder={f.placeholder}
              value={String(draft[f.key] ?? '')}
              onChange={(val) => setDraft({ ...draft, [f.key]: val })}
            />
          ) : f.type === 'number' ? (
            <Input
              type="number"
              placeholder={f.placeholder}
              value={draft[f.key] === undefined || draft[f.key] === '' ? '' : String(draft[f.key])}
              onChange={(val) => setDraft({ ...draft, [f.key]: val })}
            />
          ) : f.type === 'textarea' ? (
            <Input.TextArea
              placeholder={f.placeholder}
              value={String(draft[f.key] ?? '')}
              autoSize={{ minRows: f.textareaMinRows ?? 4, maxRows: 16 }}
              onChange={(val) => setDraft({ ...draft, [f.key]: val })}
            />
          ) : (
            <Input
              placeholder={f.placeholder}
              value={String(draft[f.key] ?? '')}
              allowClear={!f.required}
              onChange={(val) => setDraft({ ...draft, [f.key]: val })}
            />
          )}
        </div>
      ))}
    </>
  )
}

type Props = {
  visible: boolean
  mode: 'create' | 'edit'
  row?: AIProviderPoolRow | null
  onClose: () => void
  onSaved: () => void
}

export default function AiProviderPoolDrawer({ visible, mode, row, onClose, onSaved }: Props) {
  const { t } = useTranslation()
  const [modality, setModality] = useState<PoolModality>('tts')
  const [name, setName] = useState('')
  const [voiceIds, setVoiceIds] = useState('')
  const [priority, setPriority] = useState('0')
  const [quotaLimit, setQuotaLimit] = useState('0')
  const [description, setDescription] = useState('')
  const [enabled, setEnabled] = useState(true)
  const [draft, setDraft] = useState<Record<string, unknown>>(() => defaultDraft('tts'))
  const [advancedJson, setAdvancedJson] = useState('')
  const [useAdvanced, setUseAdvanced] = useState(false)
  const [saving, setSaving] = useState(false)

  const tab = modalityToTab(modality)
  const showVoiceIds = modality === 'tts' || modality === 'realtime'

  useEffect(() => {
    if (!visible) return
    if (mode === 'edit' && row) {
      const m = (row.modality || 'tts') as PoolModality
      setModality(modalities.includes(m) ? m : 'tts')
      setName(row.name || '')
      setVoiceIds((row.voiceIds || []).join(', '))
      setPriority(String(row.priority ?? 0))
      setQuotaLimit(String(row.quotaLimit ?? 0))
      setDescription(row.description || '')
      setEnabled(row.enabled ?? true)
      setDraft(mergeDraft(modalityToTab(m), row.config))
      setAdvancedJson(JSON.stringify(row.config ?? {}, null, 2))
      setUseAdvanced(false)
      return
    }
    setModality('tts')
    setName('')
    setVoiceIds('')
    setPriority('0')
    setQuotaLimit('0')
    setDescription('')
    setEnabled(true)
    setDraft(defaultDraft('tts'))
    setAdvancedJson('')
    setUseAdvanced(false)
  }, [visible, mode, row])

  const onModalityChange = (m: PoolModality) => {
    setModality(m)
    setDraft(defaultDraft(modalityToTab(m)))
  }

  const buildConfig = (): Record<string, unknown> | null => {
    if (useAdvanced) {
      try {
        return JSON.parse(advancedJson) as Record<string, unknown>
      } catch {
        showAlert(t('platformAiPools.invalidJson'), 'error')
        return null
      }
    }
    const err = validateDraft(tab, draft)
    if (err) {
      showAlert(err, 'error')
      return null
    }
    return draftToPayload(tab, draft)
  }

  const submit = async () => {
    const config = buildConfig()
    if (!config) return
    const provider = String(config.provider ?? draft.provider ?? '').trim()
    if (!provider) {
      showAlert(t('platformAiPools.providerRequired'), 'error')
      return
    }
    const ids = voiceIds
      .split(/[,，\s]+/)
      .map((s) => s.trim())
      .filter(Boolean)
    setSaving(true)
    try {
      const body = {
        name: name.trim() || undefined,
        modality,
        provider,
        config,
        voiceIds: showVoiceIds ? ids : [],
        priority: Number(priority) || 0,
        quotaLimit: Number(quotaLimit) || 0,
        description: description.trim() || undefined,
        enabled,
      }
      const res =
        mode === 'edit' && row
          ? await updateAIProviderPool(row.id, body)
          : await createAIProviderPool({ ...body, enabled: mode === 'create' ? true : enabled })
      if (res.code === 200) {
        showAlert(mode === 'edit' ? t('common.saveSuccess') : t('common.createSuccess'), 'success')
        onSaved()
        onClose()
      } else {
        showAlert(res.msg || t('common.saveFailed'), 'error')
      }
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('common.saveFailed')), 'error')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Drawer
      title={mode === 'edit' ? t('platformAiPools.edit') : t('platformAiPools.create')}
      visible={visible}
      placement="right"
      width={640}
      onCancel={() => {
        if (!saving) onClose()
      }}
      footer={
        <Space>
          <Button onClick={onClose} disabled={saving}>
            {t('common.cancel')}
          </Button>
          <Button type="primary" loading={saving} onClick={() => void submit()}>
            {mode === 'edit' ? t('common.save') : t('common.create')}
          </Button>
        </Space>
      }
    >
      <Space direction="vertical" size={14} style={{ width: '100%' }}>
        <div>
          <Typography.Text style={{ display: 'block', marginBottom: 6, fontSize: 12 }}>{t('common.name')}</Typography.Text>
          <Input
            placeholder={t('platformAiPools.namePlaceholder')}
            value={name}
            onChange={setName}
          />
          <Typography.Paragraph type="secondary" style={{ margin: '6px 0 0', fontSize: 11 }}>
            {t('platformAiPools.nameHint')}
          </Typography.Paragraph>
        </div>

        <div>
          <Typography.Text style={{ display: 'block', marginBottom: 6, fontSize: 12 }}>
            {t('platformAiPools.modality')}
          </Typography.Text>
          <Select
            style={{ width: '100%' }}
            value={modality}
            disabled={mode === 'edit'}
            options={modalities.map((m) => ({
              value: m,
              label: t(`platformAiPools.modality_${m}`),
            }))}
            onChange={(v) => onModalityChange(v as PoolModality)}
          />
        </div>

        {mode === 'edit' ? (
          <div className="flex items-center justify-between rounded-lg border border-border px-3 py-2">
            <Typography.Text style={{ fontSize: 12 }}>{t('platformAiPools.enabled')}</Typography.Text>
            <Switch checked={enabled} onChange={setEnabled} />
          </div>
        ) : null}

        <div className="grid grid-cols-2 gap-3">
          <div>
            <Typography.Text style={{ display: 'block', marginBottom: 6, fontSize: 12 }}>
              {t('platformAiPools.priority')}
            </Typography.Text>
            <Input value={priority} onChange={setPriority} />
          </div>
          <div>
            <Typography.Text style={{ display: 'block', marginBottom: 6, fontSize: 12 }}>
              {t('platformAiPools.quotaMinutes')}
            </Typography.Text>
            <Input value={quotaLimit} onChange={setQuotaLimit} placeholder="0 = ∞" />
          </div>
        </div>

        {showVoiceIds ? (
          <div>
            <Typography.Text style={{ display: 'block', marginBottom: 6, fontSize: 12 }}>
              {t('platformAiPools.voiceIds')}
            </Typography.Text>
            <Input.TextArea
              placeholder={t('platformAiPools.voiceIdsPlaceholder')}
              value={voiceIds}
              autoSize={{ minRows: 2, maxRows: 4 }}
              onChange={setVoiceIds}
            />
          </div>
        ) : null}

        <div>
          <Typography.Text style={{ display: 'block', marginBottom: 6, fontSize: 12 }}>
            {t('platformAiPools.descriptionOptional')}
          </Typography.Text>
          <Input value={description} onChange={setDescription} />
        </div>

        {!useAdvanced ? renderProviderFields(tab, draft, setDraft, t('tenantAiConfig.providerLabel')) : null}

        <Collapse
          bordered={false}
          activeKey={useAdvanced ? ['adv'] : []}
          onChange={(k) => {
            const open = Array.isArray(k) ? k.includes('adv') : k === 'adv'
            setUseAdvanced(open)
            if (open && !advancedJson.trim()) {
              const err = validateDraft(tab, draft)
              if (!err) {
                setAdvancedJson(JSON.stringify(draftToPayload(tab, draft), null, 2))
              }
            }
          }}
        >
          <Collapse.Item header={t('platformAiPools.advancedJson')} name="adv">
            <Input.TextArea
              value={advancedJson}
              onChange={setAdvancedJson}
              autoSize={{ minRows: 8, maxRows: 20 }}
              style={{ fontFamily: 'monospace', fontSize: 12 }}
            />
            <Typography.Paragraph type="secondary" style={{ margin: '8px 0 0', fontSize: 11 }}>
              {t('platformAiPools.advancedJsonHint')}
            </Typography.Paragraph>
          </Collapse.Item>
        </Collapse>
      </Space>
    </Drawer>
  )
}
