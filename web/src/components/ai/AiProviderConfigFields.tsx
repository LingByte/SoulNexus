import { Space, Typography } from '@arco-design/web-react'
import { Input, Select } from '@/components/ui'
import {
  type AiTab,
  defaultDraft,
  providerRulesFor,
  ruleFor,
} from '@/constants/tenantAiConfigRules'

export function AiProviderConfigFields({
  tab,
  draft,
  setDraft,
  t,
}: {
  tab: AiTab
  draft: Record<string, unknown>
  setDraft: (next: Record<string, unknown>) => void
  t: (key: string) => string
}) {
  const provider = String(draft.provider ?? '')
  const def = ruleFor(tab, provider)
  const opts = providerRulesFor(tab).map((x) => ({ value: x.provider, label: x.label }))
  return (
    <Space direction="vertical" size={14} style={{ width: '100%' }}>
      <div>
        <Typography.Text style={{ display: 'block', marginBottom: 6 }}>{t('tenantAiConfig.providerLabel')}</Typography.Text>
        <Select
          style={{ width: '100%', maxWidth: 480 }}
          value={provider}
          options={opts}
          onChange={(v) => setDraft({ ...defaultDraft(tab), provider: String(v) })}
        />
      </div>
      {def?.fields.map((f) => (
        <div key={f.key}>
          <Typography.Text style={{ display: 'block', marginBottom: 6 }}>
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
              style={{ maxWidth: 320 }}
              placeholder={f.placeholder}
              value={draft[f.key] === undefined || draft[f.key] === '' ? '' : String(draft[f.key])}
              onChange={(val) => setDraft({ ...draft, [f.key]: val })}
            />
          ) : f.type === 'textarea' ? (
            <Input.TextArea
              placeholder={f.placeholder}
              value={String(draft[f.key] ?? '')}
              autoSize={{ minRows: f.textareaMinRows ?? 10, maxRows: 32 }}
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
    </Space>
  )
}

export type { AiTab }
