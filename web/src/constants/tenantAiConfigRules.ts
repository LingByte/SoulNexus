/**
 * 租户 ASR / TTS / LLM 的 JSON 字段规则（仅前端校验与表单渲染，后端只存 JSON）。
 * 统一使用顶层字段 provider 标识厂商。
 * ASR/TTS 厂商 slug 对齐 lingllm/recognizer、lingllm/synthesizer；LLM 对齐 lingllm/protocol。
 * 音色在智能体上配置；租户 AI 仅维护密钥与厂商选择。
 */

/** JSON keys stripped from tenant config — timbre lives on assistant. */
const TENANT_VOICE_STRIP_KEYS = ['voice', 'voiceType', 'voiceId', 'voice_id', 'assetId', 'asset_id', 'speaker'] as const

function stripTenantVoiceFields(draft: Record<string, unknown>): Record<string, unknown> {
  const out = { ...draft }
  for (const k of TENANT_VOICE_STRIP_KEYS) {
    delete out[k]
  }
  return out
}

export type AiFieldType = 'text' | 'password' | 'number' | 'textarea'

export interface AiFieldRule {
  key: string
  label: string
  type: AiFieldType
  required?: boolean
  placeholder?: string
  /** 仅 type=textarea：最小可见行数 */
  textareaMinRows?: number
}

export interface AiProviderRule {
  provider: string
  label: string
  fields: AiFieldRule[]
}

const qcloudAsrFields: AiFieldRule[] = [
  { key: 'appId', label: 'AppId', type: 'text', required: true, placeholder: '控制台 AppId' },
  { key: 'secretId', label: 'SecretId', type: 'text', required: true },
  { key: 'secretKey', label: 'SecretKey', type: 'password', required: true },
  { key: 'modelType', label: '模型 modelType', type: 'text', placeholder: '默认 16k_zh' },
  {
    key: 'vadSilenceTime',
    label: '断句静音阈值 vadSilenceTime (ms)',
    type: 'number',
    placeholder: '默认 300；云侧未设时为 1000。范围 240–2000',
  },
]

const qcloudTtsFields: AiFieldRule[] = [
  { key: 'appId', label: 'AppId', type: 'text', required: true },
  { key: 'secretId', label: 'SecretId', type: 'text', required: true },
  { key: 'secretKey', label: 'SecretKey', type: 'password', required: true },
  { key: 'speed', label: '语速', type: 'number', placeholder: '-2~6' },
  { key: 'sampleRate', label: '采样率 Hz', type: 'number', placeholder: '0=跟随会话 PCM' },
]

const apiKeyModel: AiFieldRule[] = [
  { key: 'apiKey', label: 'API Key', type: 'password', required: true },
  { key: 'model', label: '模型', type: 'text', placeholder: '可选' },
]

const apiKeyBase: AiFieldRule[] = [
  { key: 'apiKey', label: 'API Key', type: 'password', required: true },
  { key: 'baseUrl', label: 'Base URL', type: 'text', placeholder: '服务端点' },
]

/** lingllm recognizer: volcengine / volcllmasr */
const volcengineAsrFields: AiFieldRule[] = [
  { key: 'appId', label: 'App ID', type: 'text', required: true },
  { key: 'token', label: 'Access Token', type: 'password', required: true },
  { key: 'cluster', label: 'Cluster', type: 'text', placeholder: 'volcengine_input_common（默认）' },
  { key: 'format', label: '音频格式', type: 'text', placeholder: 'raw（默认）' },
  {
    key: 'endWindowSize',
    label: '断句静音 endWindowSize (ms)',
    type: 'number',
    placeholder: '默认 300；云侧未设约 800，最小 200',
  },
]

const volcengineLLMAsrFields: AiFieldRule[] = [
  { key: 'appId', label: 'App ID', type: 'text', required: true },
  { key: 'token', label: 'Access Token', type: 'password', required: true },
  { key: 'resourceId', label: 'Resource ID', type: 'text', placeholder: 'volc.bigasr.sauc.duration（默认）' },
  {
    key: 'endWindowSize',
    label: '断句静音 endWindowSize (ms)',
    type: 'number',
    placeholder: '默认 300；云侧未设约 800，最小 200',
  },
]

/** lingllm synthesizer: volcengine（含 WS 流式，无单独 stream provider） */
const volcengineTtsFields: AiFieldRule[] = [
  { key: 'appId', label: 'App ID', type: 'text', required: true },
  { key: 'accessToken', label: 'Access Token', type: 'password', required: true },
  { key: 'cluster', label: 'Cluster', type: 'text', placeholder: 'volcano_tts（默认）' },
  { key: 'sampleRate', label: '采样率 Hz', type: 'number', placeholder: '16000（默认）' },
  { key: 'speedRatio', label: '语速', type: 'number', placeholder: '1.0（默认）' },
]

const volcengineCloneTtsFields: AiFieldRule[] = [
  { key: 'appId', label: 'App ID', type: 'text', required: true },
  { key: 'accessToken', label: 'Access Token', type: 'password', required: true },
  { key: 'cluster', label: 'Cluster', type: 'text', placeholder: 'volcano_icl（默认）' },
  { key: 'sampleRate', label: '采样率 Hz', type: 'number', placeholder: '16000（默认）' },
]

export const TENANT_ASR_PROVIDER_RULES: AiProviderRule[] = [
  { provider: 'qcloud', label: '腾讯云 ASR', fields: [...qcloudAsrFields] },
  { provider: 'google', label: 'Google Speech', fields: [...apiKeyBase, { key: 'projectId', label: 'Project ID', type: 'text' }] },
  { provider: 'aliyun', label: '阿里云 ASR', fields: [...apiKeyBase, { key: 'appKey', label: 'AppKey', type: 'text', required: true }] },
  { provider: 'qiniu', label: 'SoulNexus ASR', fields: [...apiKeyModel] },
  { provider: 'funasr', label: 'FunASR', fields: [...apiKeyBase, { key: 'endpoint', label: '服务地址', type: 'text' }] },
  { provider: 'volcengine', label: '火山引擎 ASR', fields: [...volcengineAsrFields] },
  { provider: 'volcllmasr', label: '火山 LLM ASR', fields: [...volcengineLLMAsrFields] },
  {
    provider: 'xfyun_mul',
    label: '科大讯飞多语言',
    fields: [
      { key: 'appId', label: 'AppId', type: 'text', required: true },
      { key: 'apiKey', label: 'API Key', type: 'password', required: true },
      { key: 'apiSecret', label: 'API Secret', type: 'password', required: true },
    ],
  },
  { provider: 'gladia', label: 'Gladia', fields: [...apiKeyModel] },
  { provider: 'funasr_realtime', label: 'FunASR 实时', fields: [...apiKeyBase] },
  { provider: 'whisper', label: 'Whisper', fields: [...apiKeyBase] },
  { provider: 'deepgram', label: 'Deepgram', fields: [...apiKeyModel, { key: 'language', label: '语言', type: 'text' }] },
  { provider: 'aws', label: 'AWS Transcribe', fields: [...apiKeyBase, { key: 'region', label: 'Region', type: 'text', required: true }] },
  { provider: 'baidu', label: '百度 ASR', fields: [{ key: 'apiKey', label: 'API Key', type: 'password', required: true }, { key: 'secretKey', label: 'Secret Key', type: 'password', required: true }] },
  { provider: 'voiceapi', label: 'VoiceAPI', fields: [...apiKeyBase] },
  { provider: 'local', label: '本地 ASR', fields: [{ key: 'endpoint', label: '服务地址', type: 'text', required: true }] },
  { provider: 'openai', label: 'OpenAI Whisper', fields: [...apiKeyBase] },
]

export const TENANT_TTS_PROVIDER_RULES: AiProviderRule[] = [
  { provider: 'qcloud', label: '腾讯云 TTS', fields: [...qcloudTtsFields] },
  { provider: 'qiniu', label: 'SoulNexus TTS', fields: [...apiKeyModel] },
  { provider: 'xunfei', label: '讯飞 TTS', fields: [{ key: 'appId', label: 'AppId', type: 'text', required: true }, { key: 'apiKey', label: 'API Key', type: 'password', required: true }, { key: 'apiSecret', label: 'API Secret', type: 'password', required: true }] },
  {
    provider: 'aliyun',
    label: '阿里云 Qwen-TTS 实时',
    fields: [
      { key: 'apiKey', label: 'DashScope API Key', type: 'password', required: true, placeholder: 'sk-xxx (DASHSCOPE_API_KEY)' },
      { key: 'model', label: '模型', type: 'text', placeholder: 'qwen3-tts-flash-realtime（默认）/ qwen3-tts-instruct-flash-realtime' },
      { key: 'languageType', label: '语言', type: 'text', placeholder: 'Auto / Chinese / English / Japanese ...' },
      { key: 'mode', label: '模式', type: 'text', placeholder: 'server_commit（默认）/ commit' },
      { key: 'sampleRate', label: '采样率 Hz', type: 'number', placeholder: '24000（默认）/ 22050 / 16000' },
      { key: 'baseUrl', label: 'WS 端点（可选）', type: 'text', placeholder: '留空=北京区；新加坡区 wss://dashscope-intl.aliyuncs.com/api-ws/v1/realtime' },
      { key: 'instructions', label: '风格指令（可选，需 instruct 模型）', type: 'textarea', textareaMinRows: 6, placeholder: '语速快、上扬，适合介绍时尚单品' },
    ],
  },
  { provider: 'baidu', label: '百度 TTS', fields: [{ key: 'apiKey', label: 'API Key', type: 'password', required: true }, { key: 'secretKey', label: 'Secret Key', type: 'password', required: true }] },
  { provider: 'azure', label: 'Azure TTS', fields: [...apiKeyBase, { key: 'region', label: 'Region', type: 'text', required: true }] },
  { provider: 'google', label: 'Google Cloud TTS', fields: [...apiKeyBase] },
  { provider: 'aws', label: 'AWS Polly', fields: [...apiKeyBase, { key: 'region', label: 'Region', type: 'text', required: true }] },
  { provider: 'openai', label: 'OpenAI TTS', fields: [...apiKeyBase, { key: 'model', label: '模型', type: 'text', placeholder: 'tts-1' }] },
  { provider: 'elevenlabs', label: 'ElevenLabs', fields: [...apiKeyModel] },
  { provider: 'local', label: '本地 TTS', fields: [{ key: 'endpoint', label: '服务地址', type: 'text', required: true }] },
  { provider: 'local_gospeech', label: '本地 go-speech', fields: [] },
  { provider: 'fishspeech', label: 'FishSpeech', fields: [...apiKeyBase] },
  { provider: 'fishaudio', label: 'Fish Audio', fields: [...apiKeyBase] },
  { provider: 'coqui', label: 'Coqui TTS', fields: [{ key: 'modelPath', label: '模型路径', type: 'text', required: true }] },
  { provider: 'volcengine', label: '火山引擎 TTS', fields: [...volcengineTtsFields] },
  { provider: 'volcengine_clone', label: '火山声音复刻 TTS', fields: [...volcengineCloneTtsFields] },
  { provider: 'minimax', label: 'Minimax', fields: [...apiKeyModel, { key: 'groupId', label: 'Group ID', type: 'text' }] },
]

/** LLM 成本单价（元/千 token），全租户 LLM 配置共用，与厂商无关。 */
export const TENANT_LLM_RATE_FIELD: AiFieldRule = {
  key: 'ratePer1kTokens',
  label: 'LLM 单价（元/千 token）',
  type: 'number',
  placeholder: '例如 0.02，用于平台 LLM 成本核算',
}

export const TENANT_LLM_PROVIDER_RULES: AiProviderRule[] = [
  {
    provider: 'openai',
    label: 'OpenAI 兼容',
    fields: [
      { key: 'apiKey', label: 'API Key', type: 'password', required: true },
      { key: 'baseUrl', label: 'Base URL', type: 'text', placeholder: 'https://api.openai.com/v1' },
      { key: 'model', label: '模型', type: 'text' },
    ],
  },
  {
    provider: 'anthropic',
    label: 'Anthropic',
    fields: [
      { key: 'apiKey', label: 'API Key', type: 'password', required: true },
      { key: 'baseUrl', label: 'Base URL', type: 'text', placeholder: 'https://api.anthropic.com' },
      { key: 'model', label: '模型', type: 'text', placeholder: 'claude-sonnet-4-20250514' },
    ],
  },
  {
    provider: 'ollama',
    label: 'Ollama',
    fields: [
      { key: 'baseUrl', label: 'Base URL', type: 'text', required: true, placeholder: 'http://127.0.0.1:11434' },
      { key: 'apiKey', label: 'API Key（可选）', type: 'password' },
      { key: 'model', label: '模型（可选）', type: 'text', placeholder: 'llama3.2' },
    ],
  },
]

/**
 * 实时多模态供应商（Qwen-Omni / GPT-4o realtime / …）。
 * 选 voiceMode='realtime' 时走 pkg/realtime 单条 WS，跳过 ASR/TTS/LLM 三层。
 * 后端字段对齐 pkg/realtime/aliyunomni/client.go 的 Config 解析。
 */
export const TENANT_REALTIME_PROVIDER_RULES: AiProviderRule[] = [
  {
    provider: 'aliyun_omni',
    label: '阿里云 Qwen-Omni 实时多模态',
    fields: [
      { key: 'apiKey', label: 'DashScope API Key', type: 'password', required: true, placeholder: 'sk-xxx (DASHSCOPE_API_KEY)' },
      { key: 'model', label: '模型', type: 'text', placeholder: 'qwen3.5-omni-flash-realtime-2026-03-15（默认）' },
      { key: 'baseUrl', label: 'WS 端点（可选）', type: 'text', placeholder: '留空=北京区；新加坡区 wss://dashscope-intl.aliyuncs.com/api-ws/v1/realtime' },
    ],
  },
  {
    provider: 'volcengine_dialogue',
    label: '火山豆包 端到端实时语音',
    fields: [
      { key: 'appId', label: 'App ID', type: 'text', required: true, placeholder: '控制台 X-Api-App-ID' },
      { key: 'accessKey', label: 'Access Token', type: 'password', required: true, placeholder: '控制台 X-Api-Access-Key' },
      { key: 'model', label: '模型版本', type: 'text', placeholder: 'O2.0: 1.2.1.1（默认）/ SC2.0: 2.2.0.0' },
      {
        key: 'systemRole',
        label: '系统人设（O/O2 版）',
        type: 'textarea',
        textareaMinRows: 10,
        placeholder: '例：你是客服助手，回答简洁专业',
      },
      { key: 'botName', label: '机器人名称（O 版可选）', type: 'text', placeholder: '默认豆包' },
      { key: 'speakingStyle', label: '说话风格（O 版可选）', type: 'text', placeholder: '专业、简洁、友好' },
      {
        key: 'characterManifest',
        label: '角色描述（SC/SC2 版）',
        type: 'textarea',
        textareaMinRows: 8,
        placeholder: 'SC 版本使用；填写后优先于 botName/systemRole',
      },
      { key: 'resourceId', label: 'Resource ID（可选）', type: 'text', placeholder: '默认 volc.speech.dialog' },
      { key: 'appKey', label: 'App Key（可选）', type: 'text', placeholder: '默认 PlgvMymc7f3tQnJ6' },
      { key: 'baseUrl', label: 'WS 端点（可选）', type: 'text', placeholder: 'wss://openspeech.bytedance.com/api/v3/realtime/dialogue' },
    ],
  },
]

export type AiTab = 'asr' | 'tts' | 'llm' | 'realtime'

export function providerRulesFor(tab: AiTab): AiProviderRule[] {
  if (tab === 'asr') return TENANT_ASR_PROVIDER_RULES
  if (tab === 'tts') return TENANT_TTS_PROVIDER_RULES
  if (tab === 'realtime') return TENANT_REALTIME_PROVIDER_RULES
  return TENANT_LLM_PROVIDER_RULES
}

export function ruleFor(tab: AiTab, provider: string): AiProviderRule | undefined {
  const p = String(provider || '').trim().toLowerCase()
  return providerRulesFor(tab).find((x) => x.provider.toLowerCase() === p)
}

export function defaultDraft(tab: AiTab): Record<string, unknown> {
  const first = providerRulesFor(tab)[0]
  return { provider: first?.provider ?? 'qcloud' }
}

/** Default provider slug for assistant voice picker (pipeline → TTS, realtime → Realtime). */
export function defaultVoiceProvider(mode: 'tts' | 'realtime'): string {
  const tab: AiTab = mode === 'realtime' ? 'realtime' : 'tts'
  return String(defaultDraft(tab).provider ?? (mode === 'realtime' ? 'aliyun_omni' : 'qcloud'))
}

export function voiceProviderLabel(mode: 'tts' | 'realtime', provider: string): string {
  const slug = provider.trim() || defaultVoiceProvider(mode)
  return ruleFor(mode === 'realtime' ? 'realtime' : 'tts', slug)?.label ?? slug
}

export function mergeDraft(tab: AiTab, raw: unknown): Record<string, unknown> {
  const base = defaultDraft(tab)
  if (raw && typeof raw === 'object' && !Array.isArray(raw)) {
    const merged = { ...base, ...(raw as Record<string, unknown>) }
    const prov = String(merged.provider ?? '').trim().toLowerCase()
    if (prov) {
      merged.provider = prov
    }
    if (tab === 'llm') {
      delete merged.instructions
      delete merged.systemPrompt
      delete merged.system_prompt
    }
    if (tab === 'tts' || tab === 'realtime') {
      const stripped = stripTenantVoiceFields(merged)
      if (tab === 'realtime') {
        delete stripped.instructions
        delete stripped.temperature
      }
      return stripped
    }
    return merged
  }
  return { ...base }
}

export function validateDraft(tab: AiTab, draft: Record<string, unknown>): string | null {
  const prov = String(draft.provider ?? '')
  const def = ruleFor(tab, prov)
  if (!def) return `不支持的 ${tab} 厂商：${prov || '（空）'}`
  for (const f of def.fields) {
    if (!f.required) continue
    const v = draft[f.key]
    if (v === undefined || v === null || String(v).trim() === '') {
      return `「${def.label}」请填写：${f.label}`
    }
  }
  return null
}

/** 提交给后端的 JSON：含 provider + 各厂商字段（空字符串省略） */
export function draftToPayload(tab: AiTab, draft: Record<string, unknown>): Record<string, unknown> {
  const prov = String(draft.provider ?? '').trim().toLowerCase()
  const def = ruleFor(tab, prov)
  if (!def) return { provider: prov }
  const out: Record<string, unknown> = { provider: def.provider }
  for (const f of def.fields) {
    const v = draft[f.key]
    if (v === undefined || v === null || v === '') continue
    if (f.type === 'number') {
      const n = typeof v === 'number' ? v : Number(String(v).trim())
      if (!Number.isFinite(n)) continue
      out[f.key] = n
    } else {
      out[f.key] = String(v).trim()
    }
  }
  if (tab === 'llm') {
    const rate = draft.ratePer1kTokens
    if (rate !== undefined && rate !== null && rate !== '') {
      const n = typeof rate === 'number' ? rate : Number(String(rate).trim())
      if (Number.isFinite(n)) out.ratePer1kTokens = n
    }
    delete out.instructions
    delete out.systemPrompt
    delete out.system_prompt
  }
  if (tab === 'tts' || tab === 'realtime') {
    const stripped = stripTenantVoiceFields(out)
    if (tab === 'realtime') {
      delete stripped.instructions
      delete stripped.temperature
    }
    return stripped
  }
  return out
}
