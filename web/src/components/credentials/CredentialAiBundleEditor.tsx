import { useState } from 'react'
import { Tabs, Typography } from '@arco-design/web-react'
import { Button, Input, Select } from '@/components/ui'
import { useTranslation } from '@/i18n'
import {
  type AiTab,
  defaultDraft,
  draftToPayload,
  mergeDraft,
  validateDraft,
} from '@/constants/tenantAiConfigRules'
import { AiProviderConfigFields } from '@/components/ai/AiProviderConfigFields'
import { testCredentialLLMStream, type CredentialLLMTestResult } from '@/api/credentials'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'

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
  credentialId,
}: {
  value: CredentialAiBundleState
  onChange: (next: CredentialAiBundleState) => void
  /** When editing a saved key, allow testing platform pool LLM via id. */
  credentialId?: string
}) {
  const { t } = useTranslation()
  const [aiTab, setAiTab] = useState<AiTab>('llm')
  const [llmTesting, setLlmTesting] = useState(false)
  const [llmTestPrompt, setLlmTestPrompt] = useState('请用一句话自我介绍。')
  const [llmTestResult, setLlmTestResult] = useState<CredentialLLMTestResult | null>(null)

  const patch = (partial: Partial<CredentialAiBundleState>) => onChange({ ...value, ...partial })

  const onTestLLM = async () => {
    const err = validateDraft('llm', value.llm)
    if (err) {
      showAlert(err, 'error')
      return
    }
    setLlmTesting(true)
    setLlmTestResult(null)
    try {
      const res = await testCredentialLLMStream({
        prompt: llmTestPrompt.trim() || undefined,
        llmConfig: draftToPayload('llm', value.llm),
        credentialId,
      })
      if (res.code !== 200 || !res.data) {
        showAlert(res.msg || t('credentialAi.llmTestFailed'), 'error')
        return
      }
      setLlmTestResult(res.data)
      showAlert(
        t('credentialAi.llmTestSuccess', {
          first: res.data.firstTokenMs ?? '—',
          wall: res.data.wallMs ?? '—',
        }),
        'success',
      )
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('credentialAi.llmTestFailed')), 'error')
    } finally {
      setLlmTesting(false)
    }
  }

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
          <div className="space-y-4">
            <AiProviderConfigFields tab="llm" draft={value.llm} setDraft={(llm) => patch({ llm })} t={t} />
            <div className="rounded-xl border border-border bg-muted/20 p-4">
              <Typography.Text style={{ display: 'block', marginBottom: 8 }}>
                <strong>{t('credentialAi.llmTestTitle')}</strong>
              </Typography.Text>
              <Typography.Paragraph type="secondary" style={{ marginBottom: 10, fontSize: 12 }}>
                {t('credentialAi.llmTestHint')}
              </Typography.Paragraph>
              <Input.TextArea
                value={llmTestPrompt}
                onChange={setLlmTestPrompt}
                autoSize={{ minRows: 2, maxRows: 4 }}
                placeholder={t('credentialAi.llmTestPrompt')}
                style={{ marginBottom: 10 }}
              />
              <Button type="outline" loading={llmTesting} onClick={() => void onTestLLM()}>
                {t('credentialAi.llmTestRun')}
              </Button>
              {llmTestResult ? (
                <div className="mt-3 space-y-1 text-sm text-neutral-700 dark:text-neutral-200">
                  <div>
                    {t('credentialAi.llmTestFirstToken')}: <strong>{llmTestResult.firstTokenMs ?? '—'} ms</strong>
                    <span className="mx-2 text-neutral-300">·</span>
                    {t('credentialAi.llmTestWall')}: <strong>{llmTestResult.wallMs ?? '—'} ms</strong>
                    <span className="mx-2 text-neutral-300">·</span>
                    {llmTestResult.provider}/{llmTestResult.model}
                  </div>
                  <div className="whitespace-pre-wrap rounded-lg bg-white px-3 py-2 text-neutral-800 dark:bg-neutral-900">
                    {llmTestResult.reply || '—'}
                  </div>
                </div>
              ) : null}
            </div>
          </div>
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
