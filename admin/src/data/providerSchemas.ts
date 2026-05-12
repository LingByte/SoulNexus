// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 厂商 schema 注册表。
// **provider 字段的取值必须严格等于后端 pkg/recognizer (ASR) 与 pkg/synthesizer (TTS) 中的注册名**，
// 后端不做任何归一化映射（不再有 tencent_tts → tencent 这种隐式映射）。
//
// - ASR: 见 pkg/recognizer/factory.go Vendor 常量（qcloud / aliyun / xunfei_mul / funasr / whisper / baidu / google / aws / volcengine / qiniu / local …）
// - TTS: 见 pkg/synthesizer/synthesis.go NewSynthesisServiceFromCredential 中 switch 的 case
//        （qcloud | tencent / xunfei / qiniu / baidu / azure / openai / elevenlabs / minimax / volcengine / fishaudio / fishspeech / local …）
//
// 字段说明：
//   - field.name 落到 channel.config_json 的 key；保持后端 snake_case / camelCase 风格
//   - field.secret=true → password 输入；后端编辑时空值默认沿用原值
//   - field.placeholder 仅做 UI 提示
//
// 后续新增厂商：在对应分类追加条目即可，无需改 UI。

export type ProviderFieldType = 'string' | 'number' | 'password' | 'textarea' | 'select' | 'boolean'

export interface ProviderField {
  name: string
  label: string
  type: ProviderFieldType
  required?: boolean
  secret?: boolean
  placeholder?: string
  default?: string | number | boolean
  options?: Array<{ value: string; label: string }>
  help?: string
  /** 仅在 kind 列表中显示；不写则两个 kind 都显示。 */
  kindOnly?: Array<'asr' | 'tts'>
}

export interface ProviderSchema {
  /** 唯一标识，对应 channel.provider；必须与后端注册名一致。 */
  provider: string
  /** 人类可读名 */
  label: string
  /** 文档链接（可选） */
  docs?: string
  /** 字段列表 */
  fields: ProviderField[]
  /** 备注 / 使用说明 */
  notes?: string
}

// ============================================================
// 公共字段（多个厂商复用）
// ============================================================
const FIELD_API_KEY: ProviderField = {
  name: 'api_key',
  label: 'API Key',
  type: 'password',
  required: true,
  secret: true,
  placeholder: 'sk-xxx',
}

const FIELD_ENDPOINT_OPTIONAL: ProviderField = {
  name: 'endpoint',
  label: 'Endpoint / Base URL',
  type: 'string',
  placeholder: '留空使用官方默认',
}

// ============================================================
// ASR Providers — 与 pkg/recognizer/factory.go 的 Vendor 常量一一对应
// ============================================================
export const ASR_PROVIDERS: ProviderSchema[] = [
  {
    provider: 'qcloud',
    label: '腾讯云 · 语音识别',
    docs: 'https://cloud.tencent.com/document/product/1093',
    fields: [
      { name: 'app_id', label: 'AppID', type: 'string', required: true },
      { name: 'secret_id', label: 'SecretId', type: 'string', required: true, secret: true },
      { name: 'secret_key', label: 'SecretKey', type: 'password', required: true, secret: true },
    ],
  },
  {
    provider: 'aliyun',
    label: '阿里云 · 智能语音交互',
    docs: 'https://help.aliyun.com/zh/isi',
    fields: [
      { name: 'app_key', label: 'AppKey', type: 'string', required: true },
      { name: 'access_key_id', label: 'AccessKeyId', type: 'string', required: true, secret: true },
      { name: 'access_key_secret', label: 'AccessKeySecret', type: 'password', required: true, secret: true },
      { ...FIELD_ENDPOINT_OPTIONAL, placeholder: 'wss://nls-gateway.aliyuncs.com/ws/v1' },
    ],
  },
  {
    provider: 'xfyun_mul',
    label: '科大讯飞 · 多语种语音听写',
    docs: 'https://www.xfyun.cn/doc/asr/voicedictation/API.html',
    fields: [
      { name: 'app_id', label: 'AppID', type: 'string', required: true },
      { name: 'api_key', label: 'APIKey', type: 'password', required: true, secret: true },
      { name: 'api_secret', label: 'APISecret', type: 'password', required: true, secret: true },
    ],
  },
  {
    provider: 'baidu',
    label: '百度智能云 · 短语音识别',
    docs: 'https://ai.baidu.com/ai-doc/SPEECH/Vk38lxily',
    fields: [
      { name: 'app_id', label: 'AppID', type: 'string', required: true },
      { name: 'api_key', label: 'API Key', type: 'password', required: true, secret: true },
      { name: 'secret_key', label: 'Secret Key', type: 'password', required: true, secret: true },
    ],
  },
  {
    provider: 'google',
    label: 'Google Cloud Speech-to-Text',
    docs: 'https://cloud.google.com/speech-to-text/docs',
    fields: [
      { name: 'credentials_json', label: 'Service Account JSON', type: 'textarea', required: true, secret: true, help: '直接粘贴 GCP service-account JSON 内容' },
    ],
  },
  {
    provider: 'aws',
    label: 'AWS Transcribe',
    docs: 'https://docs.aws.amazon.com/transcribe/',
    fields: [
      { name: 'access_key_id', label: 'Access Key ID', type: 'string', required: true, secret: true },
      { name: 'secret_access_key', label: 'Secret Access Key', type: 'password', required: true, secret: true },
      { name: 'region', label: 'Region', type: 'string', required: true, placeholder: 'us-east-1' },
    ],
  },
  {
    provider: 'volcengine',
    label: '火山引擎 · 语音识别',
    docs: 'https://www.volcengine.com/docs/6561',
    fields: [
      { name: 'app_id', label: 'AppID', type: 'string', required: true },
      { name: 'access_token', label: 'Access Token', type: 'password', required: true, secret: true },
      { name: 'cluster', label: 'Cluster', type: 'string', placeholder: 'volcengine_streaming_common' },
    ],
  },
  {
    provider: 'volcllmasr',
    label: '火山引擎 · 大模型 ASR',
    docs: 'https://www.volcengine.com/docs/6561',
    fields: [
      { name: 'app_id', label: 'AppID', type: 'string', required: true },
      { name: 'access_token', label: 'Access Token', type: 'password', required: true, secret: true },
    ],
  },
  {
    provider: 'funasr',
    label: 'FunASR（自托管）',
    docs: 'https://github.com/modelscope/FunASR',
    fields: [
      { ...FIELD_ENDPOINT_OPTIONAL, required: true, placeholder: 'ws://127.0.0.1:10095' },
    ],
  },
  {
    provider: 'funasr_realtime',
    label: 'FunASR 实时（自托管）',
    docs: 'https://github.com/modelscope/FunASR',
    fields: [
      { ...FIELD_ENDPOINT_OPTIONAL, required: true, placeholder: 'ws://127.0.0.1:10096' },
    ],
  },
  {
    provider: 'whisper',
    label: 'OpenAI Whisper（兼容端点）',
    docs: 'https://platform.openai.com/docs/api-reference/audio',
    fields: [
      { ...FIELD_API_KEY },
      { ...FIELD_ENDPOINT_OPTIONAL, placeholder: 'https://api.openai.com/v1' },
    ],
  },
  {
    provider: 'deepgram',
    label: 'Deepgram',
    docs: 'https://developers.deepgram.com/',
    fields: [
      { ...FIELD_API_KEY },
      { ...FIELD_ENDPOINT_OPTIONAL, placeholder: 'https://api.deepgram.com' },
    ],
  },
  {
    provider: 'gladia',
    label: 'Gladia',
    docs: 'https://docs.gladia.io/',
    fields: [{ ...FIELD_API_KEY }],
  },
  {
    provider: 'qiniu',
    label: '七牛云 · 语音识别',
    docs: 'https://developer.qiniu.com/',
    fields: [
      { name: 'api_key', label: 'API Key', type: 'password', required: true, secret: true },
      { ...FIELD_ENDPOINT_OPTIONAL },
    ],
  },
  {
    provider: 'voiceapi',
    label: 'VoiceAPI（自定义 HTTP）',
    docs: '',
    fields: [
      { ...FIELD_ENDPOINT_OPTIONAL, required: true, placeholder: 'http://127.0.0.1:8000' },
    ],
  },
  {
    provider: 'local',
    label: '本地 ASR（自部署）',
    fields: [
      { ...FIELD_ENDPOINT_OPTIONAL, required: true, placeholder: 'http://127.0.0.1:8000' },
    ],
    notes: '完全离线 / 私有部署，无需鉴权时填占位 endpoint 即可。',
  },
]

// ============================================================
// TTS Providers — 与 pkg/synthesizer/synthesis.go NewSynthesisServiceFromCredential 的 case 一一对应
// （注：synthesizer 同时接受 "qcloud" 与 "tencent" 两个别名，统一使用 "qcloud"）
// ============================================================
export const TTS_PROVIDERS: ProviderSchema[] = [
  {
    provider: 'qcloud',
    label: '腾讯云 · 语音合成',
    docs: 'https://cloud.tencent.com/document/product/1073',
    // 音色（voiceType）/ 音频编码（format）/ 地域（region） 均由调用方在请求里指定。
    fields: [
      { name: 'app_id', label: 'AppID', type: 'string', required: true },
      { name: 'secret_id', label: 'SecretId', type: 'string', required: true, secret: true },
      { name: 'secret_key', label: 'SecretKey', type: 'password', required: true, secret: true },
    ],
  },
  {
    provider: 'xunfei',
    label: '科大讯飞 · 在线语音合成',
    docs: 'https://www.xfyun.cn/doc/tts/online_tts/API.html',
    fields: [
      { name: 'app_id', label: 'AppID', type: 'string', required: true },
      { name: 'api_key', label: 'APIKey', type: 'password', required: true, secret: true },
      { name: 'api_secret', label: 'APISecret', type: 'password', required: true, secret: true },
    ],
  },
  {
    provider: 'qiniu',
    label: '七牛云 · 语音合成',
    docs: 'https://developer.qiniu.com/',
    fields: [
      { name: 'api_key', label: 'API Key', type: 'password', required: true, secret: true },
      { ...FIELD_ENDPOINT_OPTIONAL },
    ],
  },
  {
    provider: 'baidu',
    label: '百度智能云 · 语音合成',
    docs: 'https://ai.baidu.com/ai-doc/SPEECH/3k38y8h9b',
    fields: [
      { name: 'token', label: 'Access Token', type: 'password', required: true, secret: true },
    ],
  },
  {
    provider: 'google',
    label: 'Google Cloud Text-to-Speech',
    docs: 'https://cloud.google.com/text-to-speech/docs',
    fields: [
      { name: 'credentials_json', label: 'Service Account JSON', type: 'textarea', required: true, secret: true },
    ],
  },
  {
    provider: 'aws',
    label: 'AWS Polly',
    docs: 'https://docs.aws.amazon.com/polly/',
    fields: [
      { name: 'access_key_id', label: 'Access Key ID', type: 'string', required: true, secret: true },
      { name: 'secret_access_key', label: 'Secret Access Key', type: 'password', required: true, secret: true },
      { name: 'region', label: 'Region', type: 'string', required: true, placeholder: 'us-east-1' },
    ],
  },
  {
    provider: 'azure',
    label: 'Microsoft Azure · Neural TTS',
    docs: 'https://learn.microsoft.com/azure/cognitive-services/speech-service/',
    fields: [
      { name: 'subscription_key', label: 'Subscription Key', type: 'password', required: true, secret: true },
      { name: 'region', label: 'Region', type: 'string', required: true, placeholder: '如 eastasia / eastus' },
    ],
  },
  {
    provider: 'openai',
    label: 'OpenAI TTS',
    docs: 'https://platform.openai.com/docs/guides/text-to-speech',
    fields: [
      { ...FIELD_API_KEY },
      { ...FIELD_ENDPOINT_OPTIONAL, placeholder: 'https://api.openai.com/v1' },
    ],
  },
  {
    provider: 'elevenlabs',
    label: 'ElevenLabs',
    docs: 'https://elevenlabs.io/docs/api-reference/overview',
    fields: [
      { ...FIELD_API_KEY },
      { ...FIELD_ENDPOINT_OPTIONAL, placeholder: 'https://api.elevenlabs.io' },
    ],
  },
  {
    provider: 'minimax',
    label: 'MiniMax · TTS',
    docs: 'https://www.minimaxi.com/document/guides/T2A-model',
    fields: [
      { name: 'group_id', label: 'Group ID', type: 'string', required: true },
      { name: 'api_key', label: 'API Key', type: 'password', required: true, secret: true },
    ],
  },
  {
    provider: 'volcengine',
    label: '火山引擎 · 语音合成',
    docs: 'https://www.volcengine.com/docs/6561',
    fields: [
      { name: 'app_id', label: 'AppID', type: 'string', required: true },
      { name: 'access_token', label: 'Access Token', type: 'password', required: true, secret: true },
      { name: 'cluster', label: 'Cluster', type: 'string', placeholder: 'volcano_tts' },
    ],
  },
  {
    provider: 'fishaudio',
    label: 'FishAudio',
    docs: 'https://docs.fish.audio/',
    fields: [{ ...FIELD_API_KEY }],
  },
  {
    provider: 'fishspeech',
    label: 'FishSpeech（自托管）',
    docs: 'https://github.com/fishaudio/fish-speech',
    fields: [{ ...FIELD_ENDPOINT_OPTIONAL, required: true, placeholder: 'http://127.0.0.1:8080' }],
  },
  {
    provider: 'coqui',
    label: 'Coqui TTS（自托管）',
    docs: 'https://github.com/coqui-ai/TTS',
    fields: [{ ...FIELD_ENDPOINT_OPTIONAL, required: true }],
  },
  {
    provider: 'local',
    label: '本地 TTS（自部署）',
    fields: [{ ...FIELD_ENDPOINT_OPTIONAL, required: true }],
  },
  {
    provider: 'local_gospeech',
    label: '本地 GoSpeech（嵌入式）',
    fields: [],
    notes: '嵌入式本地合成，无需额外配置。',
  },
]

// ============================================================
// LLM Providers（协议级）
// 注：LLM 渠道字段大部分已由 channel 表本身承载（base_url/key/openai_organization/models）
// 这里仅为 UI 提供「占位 / 文档链接 / 该协议下额外配置项」。
// ============================================================
export interface LLMProviderHint {
  protocol: string
  label: string
  docs?: string
  baseUrlPlaceholder?: string
  modelsHint?: string
  /** 该协议下「额外」需要的 channel.config_json 字段（OpenAI org 等已在专属列里，不放这里） */
  extraFields?: ProviderField[]
  /** UI 提示 */
  notes?: string
  /** 显示 openai_organization 字段 */
  showOpenAIOrganization?: boolean
}

export const LLM_PROVIDER_HINTS: LLMProviderHint[] = [
  {
    protocol: 'openai',
    label: 'OpenAI',
    docs: 'https://platform.openai.com/docs',
    baseUrlPlaceholder: 'https://api.openai.com',
    modelsHint: 'gpt-4o-mini,gpt-4o,gpt-3.5-turbo',
    showOpenAIOrganization: true,
  },
  {
    protocol: 'azure',
    label: 'Azure OpenAI',
    docs: 'https://learn.microsoft.com/azure/ai-services/openai/',
    baseUrlPlaceholder: 'https://<resource>.openai.azure.com',
    modelsHint: 'gpt-4o,gpt-35-turbo',
    extraFields: [
      { name: 'api_version', label: 'API Version', type: 'string', required: true, default: '2024-06-01', placeholder: '2024-06-01' },
      { name: 'deployment_map', label: 'Deployment Map (JSON)', type: 'textarea', help: '可选：模型名 → 部署名映射，留空走模型名同名部署' },
    ],
  },
  {
    protocol: 'anthropic',
    label: 'Anthropic Claude',
    docs: 'https://docs.anthropic.com',
    baseUrlPlaceholder: 'https://api.anthropic.com',
    modelsHint: 'claude-3-5-sonnet-20241022,claude-3-haiku-20240307',
    extraFields: [
      { name: 'anthropic_version', label: 'Anthropic Version', type: 'string', default: '2023-06-01' },
    ],
  },
  {
    protocol: 'gemini',
    label: 'Google Gemini',
    docs: 'https://ai.google.dev/',
    baseUrlPlaceholder: 'https://generativelanguage.googleapis.com',
    modelsHint: 'gemini-1.5-flash,gemini-1.5-pro',
  },
  {
    protocol: 'qwen',
    label: '阿里通义千问 / DashScope',
    docs: 'https://help.aliyun.com/zh/dashscope/',
    baseUrlPlaceholder: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
    modelsHint: 'qwen-plus,qwen-turbo,qwen-max',
  },
  {
    protocol: 'deepseek',
    label: 'DeepSeek',
    docs: 'https://api-docs.deepseek.com',
    baseUrlPlaceholder: 'https://api.deepseek.com',
    modelsHint: 'deepseek-chat,deepseek-reasoner',
  },
  {
    protocol: 'zhipu',
    label: '智谱 GLM',
    docs: 'https://open.bigmodel.cn/dev/api',
    baseUrlPlaceholder: 'https://open.bigmodel.cn/api/paas/v4',
    modelsHint: 'glm-4-plus,glm-4-flash',
  },
  {
    protocol: 'moonshot',
    label: 'Moonshot / Kimi',
    docs: 'https://platform.moonshot.cn/docs',
    baseUrlPlaceholder: 'https://api.moonshot.cn/v1',
    modelsHint: 'moonshot-v1-8k,moonshot-v1-32k',
  },
  {
    protocol: 'ollama',
    label: 'Ollama 本地',
    docs: 'https://github.com/ollama/ollama/blob/main/docs/api.md',
    baseUrlPlaceholder: 'http://127.0.0.1:11434',
    modelsHint: 'llama3.1,qwen2.5,deepseek-r1',
    notes: '本地 / 私有部署，无需 Key 时可填占位符。',
  },
]

export function findASRSchema(provider: string): ProviderSchema | undefined {
  return ASR_PROVIDERS.find((p) => p.provider === provider)
}

export function findTTSSchema(provider: string): ProviderSchema | undefined {
  return TTS_PROVIDERS.find((p) => p.provider === provider)
}

export function findLLMHint(protocol: string): LLMProviderHint | undefined {
  return LLM_PROVIDER_HINTS.find((p) => p.protocol === protocol)
}
