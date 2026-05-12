// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 厂商配置 schema 集中注册：
//   - 每个 provider 描述「需要填什么字段」+「字段类型 / 是否必填 / 是否敏感」
//   - 表单组件根据 schema 渲染对应输入，再序列化为 channel.config_json
//   - 未在注册表中的 provider 退化为「原始 JSON」编辑
//
// 命名约定：
//   - field.name 是落到 config_json 的 key（保持后端 snake_case 风格）
//   - field.secret=true 表示「敏感字段」（密码/密钥），UI 用 password input
//   - field.placeholder 仅做提示，不参与默认值
//   - 默认值放在 field.default 中
//
// 后续要新增厂商：在对应分类追加条目即可，无需改 UI。

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
  /** 唯一标识，对应 channel.provider */
  provider: string
  /** 人类可读名 */
  label: string
  /** 文档链接（可选） */
  docs?: string
  /** 字段列表 */
  fields: ProviderField[]
  /** 备注 / 使用说明 */
  notes?: string
  /** 该 provider 默认使用的模型 / 音色提示，UI 把它注入 models 字段的 placeholder。 */
  modelsHint?: string
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

const FIELD_REGION: ProviderField = {
  name: 'region',
  label: 'Region',
  type: 'string',
  placeholder: '如 ap-shanghai / eastus',
}

// ============================================================
// ASR Providers
// ============================================================
export const ASR_PROVIDERS: ProviderSchema[] = [
  {
    provider: 'tencent_asr',
    label: '腾讯云 · 语音识别',
    docs: 'https://cloud.tencent.com/document/product/1093',
    modelsHint: '16k_zh,16k_zh_video,16k_en',
    fields: [
      { name: 'app_id', label: 'AppID', type: 'string', required: true, placeholder: '腾讯云 AppID' },
      { name: 'secret_id', label: 'SecretId', type: 'string', required: true, secret: true },
      { name: 'secret_key', label: 'SecretKey', type: 'password', required: true, secret: true },
      { ...FIELD_REGION, default: 'ap-shanghai' },
    ],
  },
  {
    provider: 'aliyun_funasr',
    label: '阿里云 · 智能语音交互（FunASR / Paraformer）',
    docs: 'https://help.aliyun.com/zh/isi',
    modelsHint: 'paraformer-realtime-v2,paraformer-v2',
    fields: [
      { name: 'app_key', label: 'AppKey', type: 'string', required: true },
      { name: 'access_key_id', label: 'AccessKeyId', type: 'string', required: true, secret: true },
      { name: 'access_key_secret', label: 'AccessKeySecret', type: 'password', required: true, secret: true },
      { ...FIELD_ENDPOINT_OPTIONAL, placeholder: 'wss://nls-gateway.aliyuncs.com/ws/v1' },
    ],
  },
  {
    provider: 'xfyun_asr',
    label: '讯飞 · 语音听写',
    docs: 'https://www.xfyun.cn/doc/asr/voicedictation/API.html',
    modelsHint: 'iat,rtasr',
    fields: [
      { name: 'app_id', label: 'AppID', type: 'string', required: true },
      { name: 'api_key', label: 'APIKey', type: 'password', required: true, secret: true },
      { name: 'api_secret', label: 'APISecret', type: 'password', required: true, secret: true },
    ],
  },
  {
    provider: 'baidu_asr',
    label: '百度智能云 · 短语音识别',
    docs: 'https://ai.baidu.com/ai-doc/SPEECH/Vk38lxily',
    modelsHint: 'pro,standard',
    fields: [
      { name: 'app_id', label: 'AppID', type: 'string', required: true },
      { name: 'api_key', label: 'API Key', type: 'password', required: true, secret: true },
      { name: 'secret_key', label: 'Secret Key', type: 'password', required: true, secret: true },
    ],
  },
  {
    provider: 'azure',
    label: 'Microsoft Azure · Speech Services',
    docs: 'https://learn.microsoft.com/azure/cognitive-services/speech-service/',
    modelsHint: 'zh-CN,en-US',
    fields: [
      { name: 'subscription_key', label: 'Subscription Key', type: 'password', required: true, secret: true },
      { ...FIELD_REGION, required: true, placeholder: '如 eastasia / eastus' },
      { name: 'endpoint', label: '自定义 Endpoint', type: 'string', help: '留空走 Region 默认' },
    ],
  },
  {
    provider: 'whisper',
    label: 'OpenAI · Whisper（兼容端点）',
    docs: 'https://platform.openai.com/docs/api-reference/audio',
    modelsHint: 'whisper-1',
    fields: [
      { ...FIELD_API_KEY },
      { ...FIELD_ENDPOINT_OPTIONAL, placeholder: 'https://api.openai.com/v1' },
      { name: 'organization', label: 'OpenAI Organization', type: 'string' },
    ],
  },
  {
    provider: 'volcengine_asr',
    label: '火山引擎 · 语音识别',
    docs: 'https://www.volcengine.com/docs/6561',
    modelsHint: 'bigmodel,streaming',
    fields: [
      { name: 'app_id', label: 'AppID', type: 'string', required: true },
      { name: 'access_token', label: 'Access Token', type: 'password', required: true, secret: true },
      { name: 'cluster', label: 'Cluster', type: 'string', placeholder: 'volcengine_streaming_common' },
    ],
  },
]

// ============================================================
// TTS Providers
// ============================================================
export const TTS_PROVIDERS: ProviderSchema[] = [
  {
    provider: 'tencent_tts',
    label: '腾讯云 · 语音合成',
    docs: 'https://cloud.tencent.com/document/product/1073',
    modelsHint: '101001,101002,201001 (VoiceType)',
    fields: [
      { name: 'app_id', label: 'AppID', type: 'string', required: true },
      { name: 'secret_id', label: 'SecretId', type: 'string', required: true, secret: true },
      { name: 'secret_key', label: 'SecretKey', type: 'password', required: true, secret: true },
      { ...FIELD_REGION, default: 'ap-shanghai' },
      {
        name: 'codec',
        label: '音频编码',
        type: 'select',
        default: 'mp3',
        options: [
          { value: 'mp3', label: 'mp3' },
          { value: 'pcm', label: 'pcm' },
          { value: 'wav', label: 'wav' },
        ],
      },
    ],
  },
  {
    provider: 'aliyun_cosyvoice',
    label: '阿里云 · CosyVoice / 语音合成',
    docs: 'https://help.aliyun.com/zh/dashscope/developer-reference/cosyvoice-quick-start',
    modelsHint: 'cosyvoice-v1,longxiaobai,longxiaochun,longwan',
    fields: [
      { name: 'api_key', label: 'DashScope API Key', type: 'password', required: true, secret: true },
      { ...FIELD_ENDPOINT_OPTIONAL, placeholder: 'https://dashscope.aliyuncs.com/api/v1' },
    ],
  },
  {
    provider: 'xfyun_tts',
    label: '讯飞 · 在线语音合成',
    docs: 'https://www.xfyun.cn/doc/tts/online_tts/API.html',
    modelsHint: 'xiaoyan,aisjiuxu,aisxping',
    fields: [
      { name: 'app_id', label: 'AppID', type: 'string', required: true },
      { name: 'api_key', label: 'APIKey', type: 'password', required: true, secret: true },
      { name: 'api_secret', label: 'APISecret', type: 'password', required: true, secret: true },
    ],
  },
  {
    provider: 'baidu_tts',
    label: '百度智能云 · 语音合成',
    docs: 'https://ai.baidu.com/ai-doc/SPEECH/3k38y8h9b',
    modelsHint: '0,1,3,4,5,103,106 (per_id)',
    fields: [
      { name: 'app_id', label: 'AppID', type: 'string', required: true },
      { name: 'api_key', label: 'API Key', type: 'password', required: true, secret: true },
      { name: 'secret_key', label: 'Secret Key', type: 'password', required: true, secret: true },
    ],
  },
  {
    provider: 'azure',
    label: 'Microsoft Azure · Neural TTS',
    docs: 'https://learn.microsoft.com/azure/cognitive-services/speech-service/',
    modelsHint: 'zh-CN-XiaoxiaoNeural,en-US-AriaNeural',
    fields: [
      { name: 'subscription_key', label: 'Subscription Key', type: 'password', required: true, secret: true },
      { ...FIELD_REGION, required: true, placeholder: '如 eastasia / eastus' },
      { name: 'endpoint', label: '自定义 Endpoint', type: 'string', help: '留空走 Region 默认' },
    ],
  },
  {
    provider: 'openai',
    label: 'OpenAI · TTS（兼容端点）',
    docs: 'https://platform.openai.com/docs/guides/text-to-speech',
    modelsHint: 'tts-1,tts-1-hd',
    fields: [
      { ...FIELD_API_KEY },
      { ...FIELD_ENDPOINT_OPTIONAL, placeholder: 'https://api.openai.com/v1' },
      {
        name: 'voice',
        label: '默认音色（voice）',
        type: 'select',
        default: 'alloy',
        options: [
          { value: 'alloy', label: 'alloy' },
          { value: 'echo', label: 'echo' },
          { value: 'fable', label: 'fable' },
          { value: 'onyx', label: 'onyx' },
          { value: 'nova', label: 'nova' },
          { value: 'shimmer', label: 'shimmer' },
        ],
      },
    ],
  },
  {
    provider: 'edge',
    label: 'Edge TTS（免费 · 无需鉴权）',
    docs: 'https://github.com/rany2/edge-tts',
    modelsHint: 'zh-CN-XiaoxiaoNeural,zh-CN-YunxiNeural',
    fields: [
      {
        name: 'proxy',
        label: '可选代理 URL',
        type: 'string',
        placeholder: '形如 http://127.0.0.1:7890',
        help: '部分网络环境直连受限时填写',
      },
    ],
    notes: 'Edge TTS 调用微软线上 Neural 音色接口，免密钥；仅依赖出网。',
  },
  {
    provider: 'minimax',
    label: 'MiniMax · TTS',
    docs: 'https://www.minimaxi.com/document/guides/T2A-model',
    modelsHint: 'speech-01,speech-02',
    fields: [
      { name: 'group_id', label: 'Group ID', type: 'string', required: true },
      { name: 'api_key', label: 'API Key', type: 'password', required: true, secret: true },
    ],
  },
  {
    provider: 'elevenlabs',
    label: 'ElevenLabs',
    docs: 'https://elevenlabs.io/docs/api-reference/overview',
    modelsHint: 'eleven_multilingual_v2,eleven_turbo_v2',
    fields: [
      { name: 'api_key', label: 'API Key', type: 'password', required: true, secret: true },
      { ...FIELD_ENDPOINT_OPTIONAL, placeholder: 'https://api.elevenlabs.io' },
    ],
  },
  {
    provider: 'volcengine_tts',
    label: '火山引擎 · 语音合成',
    docs: 'https://www.volcengine.com/docs/6561',
    modelsHint: 'BV001_streaming,BV002_streaming',
    fields: [
      { name: 'app_id', label: 'AppID', type: 'string', required: true },
      { name: 'access_token', label: 'Access Token', type: 'password', required: true, secret: true },
      { name: 'cluster', label: 'Cluster', type: 'string', placeholder: 'volcano_tts' },
    ],
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
