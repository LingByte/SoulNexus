import { useState } from 'react'
import { Tabs, Typography } from '@arco-design/web-react'
import { Select } from '@/components/ui'
import { useTranslation } from '@/i18n'
import {
  type AiTab,
  defaultDraft,
  draftToPayload,
  mergeDraft,
} from '@/constants/tenantAiConfigRules'
import { AiProviderConfigFields } from '@/components/ai/AiProviderConfigFields'

const TabPane = Tabs.TabPane

export type CredentialAiBundleState = {
  voiceMode: 'pipeline' | 'realtime'
  asr: Record<string, unknown>
  tts: Record<string, unknown>
  llm: Record<string, unknown>
  realtime: Record<string, unknown>
}

export function emptyCredentialAiBundle(): CredentialAiBundleState {
  return {
    voiceMode: 'pipeline',
    asr: defaultDraft('asr'),
    tts: defaultDraft('tts'),
    llm: defaultDraft('llm'),
    realtime: defaultDraft('realtime'),
  }
}

export function credentialAiBundleFromRow(row: {
  voiceMode?: string
  asrConfig?: unknown
  ttsConfig?: unknown
  llmConfig?: unknown
  realtimeConfig?: unknown
}): CredentialAiBundleState {
  return {
    voiceMode: row.voiceMode === 'realtime' ? 'realtime' : 'pipeline',
    asr: mergeDraft('asr', row.asrConfig),
    tts: mergeDraft('tts', row.ttsConfig),
    llm: mergeDraft('llm', row.llmConfig),
    realtime: mergeDraft('realtime', row.realtimeConfig),
  }
}

export function credentialAiBundleToPayload(state: CredentialAiBundleState) {
  return {
    kind: 'user_bundle' as const,
    voiceMode: state.voiceMode,
    asrConfig: draftToPayload('asr', state.asr),
    ttsConfig: draftToPayload('tts', state.tts),
    llmConfig: draftToPayload('llm', state.llm),
    realtimeConfig: draftToPayload('realtime', state.realtime),
  }
}

export default function CredentialAiBundleEditor({
  value,
  onChange,
}: {
  value: CredentialAiBundleState
  onChange: (next: CredentialAiBundleState) => void
}) {
  const { t } = useTranslation()
  const [aiTab, setAiTab] = useState<AiTab>('llm')

  const patch = (partial: Partial<CredentialAiBundleState>) => onChange({ ...value, ...partial })

  return (
    <div className="space-y-4">
      <div>
        <Typography.Text style={{ display: 'block', marginBottom: 6 }}>{t('tenantAiConfig.voiceMode')}</Typography.Text>
        <Select
          style={{ width: 240 }}
          value={value.voiceMode}
          options={[
            { value: 'pipeline', label: 'Pipeline (ASR+LLM+TTS)' },
            { value: 'realtime', label: 'Realtime' },
          ]}
          onChange={(v) => patch({ voiceMode: v === 'realtime' ? 'realtime' : 'pipeline' })}
        />
      </div>
      <Tabs activeTab={aiTab} onChange={(k) => setAiTab(k as AiTab)} type="rounded">
        <TabPane key="llm" title="LLM">
          <AiProviderConfigFields tab="llm" draft={value.llm} setDraft={(llm) => patch({ llm })} t={t} />
        </TabPane>
        <TabPane key="asr" title="ASR">
          <AiProviderConfigFields tab="asr" draft={value.asr} setDraft={(asr) => patch({ asr })} t={t} />
        </TabPane>
        <TabPane key="tts" title="TTS">
          <AiProviderConfigFields tab="tts" draft={value.tts} setDraft={(tts) => patch({ tts })} t={t} />
        </TabPane>
        <TabPane key="realtime" title="Realtime">
          <AiProviderConfigFields tab="realtime" draft={value.realtime} setDraft={(realtime) => patch({ realtime })} t={t} />
        </TabPane>
      </Tabs>
    </div>
  )
}
