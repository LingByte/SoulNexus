import { useCallback, useEffect, useState } from 'react'
import {
  Space,
  Tabs,
  Typography,
} from '@arco-design/web-react'
import { Button, Link, Select, Card, Input } from '@/components/ui'
import { Loading } from '@/components/ui/loading'
import { IconLeft } from '@arco-design/web-react/icon'
import { useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from '@/i18n'
import BaseLayout from '@/components/Layout/BaseLayout'
import { getTenant, updateTenantPlatform, testTenantLLMStream, type TenantLLMTestResult } from '@/api/tenants'
import {
  TENANT_LLM_RATE_FIELD,
  type AiTab,
  defaultDraft,
  draftToPayload,
  mergeDraft,
  providerRulesFor,
  ruleFor,
  validateDraft,
} from '@/constants/tenantAiConfigRules'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'

const TabPane = Tabs.TabPane

function renderAiFields(
  tab: AiTab,
  draft: Record<string, unknown>,
  setDraft: (next: Record<string, unknown>) => void,
  t: (key: string) => string,
) {
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

export default function TenantAiConfig() {
  const { tenantId } = useParams<{ tenantId: string }>()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const [tenantName, setTenantName] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [aiTab, setAiTab] = useState<AiTab>('asr')
  const [draftAsr, setDraftAsr] = useState<Record<string, unknown>>(() => defaultDraft('asr'))
  const [draftTts, setDraftTts] = useState<Record<string, unknown>>(() => defaultDraft('tts'))
  const [draftLlm, setDraftLlm] = useState<Record<string, unknown>>(() => defaultDraft('llm'))
  const [draftRealtime, setDraftRealtime] = useState<Record<string, unknown>>(() => defaultDraft('realtime'))
  const [voiceMode, setVoiceMode] = useState<'pipeline' | 'realtime'>('pipeline')
  const [llmTesting, setLlmTesting] = useState(false)
  const [llmTestPrompt, setLlmTestPrompt] = useState('请用一句话自我介绍。')
  const [llmTestResult, setLlmTestResult] = useState<TenantLLMTestResult | null>(null)

  const load = useCallback(async () => {
    if (!tenantId) return
    setLoading(true)
    try {
      const r = await getTenant(tenantId)
      if (r.code !== 200 || !r.data?.tenant) {
        showAlert(r.msg || t('tenantAiConfig.loadFailed'), 'error')
        navigate('/tenant-management', { replace: true })
        return
      }
      const tenant = r.data.tenant
      setTenantName(tenant.name)
      setDraftAsr(mergeDraft('asr', tenant.asrConfig))
      setDraftTts(mergeDraft('tts', tenant.ttsConfig))
      setDraftLlm(mergeDraft('llm', tenant.llmConfig))
      setDraftRealtime(mergeDraft('realtime', tenant.realtimeConfig))
      setVoiceMode(tenant.voiceMode === 'realtime' ? 'realtime' : 'pipeline')
      if (tenant.voiceMode === 'realtime') {
        setAiTab('realtime')
      }
    } finally {
      setLoading(false)
    }
  }, [navigate, tenantId])

  useEffect(() => {
    void load()
  }, [load])

  const onSave = async () => {
    if (!tenantId) return
    const err =
      voiceMode === 'realtime'
        ? validateDraft('realtime', draftRealtime)
        : validateDraft('asr', draftAsr) || validateDraft('tts', draftTts) || validateDraft('llm', draftLlm)
    if (err) {
      showAlert(err, 'error')
      return
    }
    setSaving(true)
    try {
      const r = await updateTenantPlatform(tenantId, {
        asrConfig: draftToPayload('asr', draftAsr),
        ttsConfig: draftToPayload('tts', draftTts),
        llmConfig: draftToPayload('llm', draftLlm),
        voiceMode,
        realtimeConfig: draftToPayload('realtime', draftRealtime),
      })
      if (r.code !== 200) {
        showAlert(r.msg || t('tenantAiConfig.saveFailed'), 'error')
        return
      }
      showAlert(t('tenantAiConfig.saveSuccess'), 'success')
    } finally {
      setSaving(false)
    }
  }

  const onTestLLM = async () => {
    if (!tenantId) return
    const err = validateDraft('llm', draftLlm)
    if (err) {
      showAlert(err, 'error')
      return
    }
    setLlmTesting(true)
    setLlmTestResult(null)
    try {
      const r = await testTenantLLMStream(tenantId, {
        prompt: llmTestPrompt.trim() || undefined,
        llmConfig: draftToPayload('llm', draftLlm),
      })
      if (r.code !== 200 || !r.data) {
        showAlert(r.msg || 'LLM 测试失败', 'error')
        return
      }
      setLlmTestResult(r.data)
      showAlert(
        `流式连通：首 token ${r.data.firstTokenMs ?? '—'} ms · 整轮 ${r.data.wallMs ?? '—'} ms`,
        'success',
      )
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, 'LLM 测试失败'), 'error')
    } finally {
      setLlmTesting(false)
    }
  }

  const title = tenantName ? `${t('tenantAiConfig.title')} — ${tenantName}` : t('tenantAiConfig.title')

  return (
    <BaseLayout
      title={title}
      description={t('tenantAiConfig.description')}
      actions={
        <Space>
          <Link to="/tenant-management">
            <Button icon={<IconLeft />}>{t('tenantAiConfig.backToList')}</Button>
          </Link>
          <Button type="primary" loading={saving} disabled={loading} onClick={() => void onSave()}>
            {t('tenantAiConfig.saveConfig')}
          </Button>
        </Space>
      }
    >
      <Card bordered={false} style={{ maxWidth: 960 }}>
        {loading ? (
          <Loading block tip={t('tenantAiConfig.loadingConfig')} />
        ) : (
          <Space direction="vertical" size={20} style={{ width: '100%' }}>
            <div>
              <Typography.Text style={{ display: 'block', marginBottom: 8 }}>
                <strong>{t('tenantAiConfig.voiceMode')}</strong>
              </Typography.Text>
              <Select
                style={{ width: 360 }}
                value={voiceMode}
                options={[
                  { value: 'pipeline', label: t('tenantAiConfig.pipelineLabel') },
                  { value: 'realtime', label: t('tenantAiConfig.realtimeLabel') },
                ]}
                onChange={(v) => {
                  const mode = v as 'pipeline' | 'realtime'
                  setVoiceMode(mode)
                  if (mode === 'realtime') setAiTab('realtime')
                }}
              />
              <Typography.Paragraph type="secondary" style={{ marginBottom: 0, marginTop: 8, fontSize: 13 }}>
                {voiceMode === 'realtime'
                  ? t('tenantAiConfig.realtimeHint')
                  : t('tenantAiConfig.pipelineHint')}
              </Typography.Paragraph>
            </div>
            <Tabs activeTab={aiTab} onChange={(k) => setAiTab(k as AiTab)} type="rounded">
              <TabPane key="asr" title="ASR">
                {renderAiFields('asr', draftAsr, setDraftAsr, t)}
              </TabPane>
              <TabPane key="tts" title="TTS">
                {renderAiFields('tts', draftTts, setDraftTts, t)}
              </TabPane>
              <TabPane key="llm" title="LLM">
                <Space direction="vertical" size={14} style={{ width: '100%' }}>
                  <Typography.Paragraph type="secondary" style={{ marginBottom: 0, fontSize: 12 }}>
                    仅配置连接参数（API Key / Base URL / 模型）。系统提示词请在各智能体「对话」Tab 的 Prompt 中维护。
                  </Typography.Paragraph>
                  <div>
                    <Typography.Text style={{ display: 'block', marginBottom: 6 }}>
                      {TENANT_LLM_RATE_FIELD.label}
                    </Typography.Text>
                    <Input
                      type="number"
                      style={{ maxWidth: 320 }}
                      placeholder={TENANT_LLM_RATE_FIELD.placeholder}
                      value={
                        draftLlm.ratePer1kTokens === undefined || draftLlm.ratePer1kTokens === ''
                          ? ''
                          : String(draftLlm.ratePer1kTokens)
                      }
                      onChange={(val) => setDraftLlm({ ...draftLlm, ratePer1kTokens: val })}
                    />
                    <Typography.Paragraph type="secondary" style={{ marginBottom: 0, marginTop: 6, fontSize: 12 }}>
                      平台 LLM 成本核算使用此单价（人民币）。
                    </Typography.Paragraph>
                  </div>
                  {renderAiFields('llm', draftLlm, setDraftLlm, t)}
                  <div className="rounded-xl border border-border bg-muted/20 p-4">
                    <Typography.Text style={{ display: 'block', marginBottom: 8 }}>
                      <strong>流式连通测试</strong>
                    </Typography.Text>
                    <Typography.Paragraph type="secondary" style={{ marginBottom: 10, fontSize: 12 }}>
                      使用上方表单中的配置（可先改再测，不必保存）发起一次流式请求，返回首 token 耗时。
                    </Typography.Paragraph>
                    <Input.TextArea
                      value={llmTestPrompt}
                      onChange={setLlmTestPrompt}
                      autoSize={{ minRows: 2, maxRows: 4 }}
                      placeholder="测试提示词"
                      style={{ marginBottom: 10 }}
                    />
                    <Button type="outline" loading={llmTesting} disabled={loading} onClick={() => void onTestLLM()}>
                      测试流式链接
                    </Button>
                    {llmTestResult ? (
                      <div className="mt-3 space-y-1 text-sm text-neutral-700">
                        <div>
                          首 token：<strong>{llmTestResult.firstTokenMs ?? '—'} ms</strong>
                          <span className="mx-2 text-neutral-300">·</span>
                          整轮：<strong>{llmTestResult.wallMs ?? '—'} ms</strong>
                          <span className="mx-2 text-neutral-300">·</span>
                          {llmTestResult.provider}/{llmTestResult.model}
                        </div>
                        <div className="whitespace-pre-wrap rounded-lg bg-white px-3 py-2 text-neutral-800 dark:bg-neutral-900">
                          {llmTestResult.reply || '（空回复）'}
                        </div>
                      </div>
                    ) : null}
                  </div>
                </Space>
              </TabPane>
              <TabPane key="realtime" title={t('tenantAiConfig.realtimeTab')}>
                {renderAiFields('realtime', draftRealtime, setDraftRealtime, t)}
              </TabPane>
            </Tabs>
          </Space>
        )}
      </Card>
    </BaseLayout>
  )
}
