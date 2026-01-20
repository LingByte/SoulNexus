// 服务商配置定义

export interface ProviderField {
  key: string
  label: string
  type: 'text' | 'password' | 'select' | 'number'
  placeholder?: string
  required?: boolean
  options?: { value: string; label: string }[]
  description?: string
}

export interface ProviderConfig {
  name: string
  fields: ProviderField[]
}

// TTS 服务商配置
export const TTS_PROVIDERS: Record<string, ProviderConfig> = {
  qiniu: {
    name: '七牛云',
    fields: [
      {
        key: 'apiKey',
        label: 'API Key',
        type: 'password',
        placeholder: '请输入七牛云 API Key',
        required: true,
        description: '七牛云的 API Key'
      },
      {
        key: 'baseUrl',
        label: 'Base URL',
        type: 'text',
        placeholder: 'https://openai.qiniu.com/v1',
        required: false,
        description: 'API 基础地址，默认为 https://openai.qiniu.com/v1'
      },
      {
        key: 'voiceType',
        label: '音色类型',
        type: 'text',
        placeholder: 'female_cn_001',
        required: false,
        description: '音色类型，默认为 female_cn_001'
      }
    ]
  },
  qcloud: {
    name: '腾讯云',
    fields: [
      {
        key: 'appId',
        label: 'App ID',
        type: 'text',
        placeholder: '请输入腾讯云 App ID',
        required: true,
        description: '腾讯云应用 ID'
      },
      {
        key: 'secretId',
        label: 'Secret ID',
        type: 'password',
        placeholder: '请输入 Secret ID',
        required: true,
        description: '腾讯云 Secret ID'
      },
      {
        key: 'secretKey',
        label: 'Secret Key',
        type: 'password',
        placeholder: '请输入 Secret Key',
        required: true,
        description: '腾讯云 Secret Key'
      },
      {
        key: 'voiceType',
        label: '音色类型',
        type: 'number',
        placeholder: '601002',
        required: false,
        description: '音色类型 ID，例如 601002（爱小辰）'
      }
    ]
  },
  tencent: {
    name: '腾讯云（别名）',
    fields: [
      {
        key: 'appId',
        label: 'App ID',
        type: 'text',
        placeholder: '请输入腾讯云 App ID',
        required: true
      },
      {
        key: 'secretId',
        label: 'Secret ID',
        type: 'password',
        placeholder: '请输入 Secret ID',
        required: true
      },
      {
        key: 'secretKey',
        label: 'Secret Key',
        type: 'password',
        placeholder: '请输入 Secret Key',
        required: true
      }
    ]
  },
  baidu: {
    name: '百度',
    fields: [
      {
        key: 'token',
        label: 'Access Token',
        type: 'password',
        placeholder: '请输入百度 Access Token',
        required: true,
        description: '百度的 Access Token（通过 API Key 和 Secret Key 获取）'
      }
    ]
  },
  azure: {
    name: '微软 Azure',
    fields: [
      {
        key: 'subscriptionKey',
        label: 'Subscription Key',
        type: 'password',
        placeholder: '请输入 Azure Subscription Key',
        required: true,
        description: 'Azure 的订阅密钥'
      },
      {
        key: 'region',
        label: 'Region',
        type: 'text',
        placeholder: 'eastasia',
        required: true,
        description: 'Azure 服务区域，例如 eastasia, eastus 等'
      },
      {
        key: 'voice',
        label: '音色',
        type: 'text',
        placeholder: 'zh-CN-XiaoxiaoNeural',
        required: false,
        description: '语音名称，默认为 zh-CN-XiaoxiaoNeural'
      }
    ]
  },
  xunfei: {
    name: '科大讯飞',
    fields: [
      {
        key: 'appId',
        label: 'App ID',
        type: 'text',
        placeholder: '请输入讯飞 App ID',
        required: true,
        description: '科大讯飞应用 ID'
      },
      {
        key: 'apiKey',
        label: 'API Key',
        type: 'password',
        placeholder: '请输入 API Key',
        required: true,
        description: '科大讯飞 API Key'
      },
      {
        key: 'apiSecret',
        label: 'API Secret',
        type: 'password',
        placeholder: '请输入 API Secret',
        required: true,
        description: '科大讯飞 API Secret'
      }
    ]
  },
  openai: {
    name: 'OpenAI',
    fields: [
      {
        key: 'apiKey',
        label: 'API Key',
        type: 'password',
        placeholder: '请输入 OpenAI API Key',
        required: true,
        description: 'OpenAI API Key'
      },
      {
        key: 'baseUrl',
        label: 'Base URL',
        type: 'text',
        placeholder: 'https://api.openai.com',
        required: false,
        description: 'API 基础地址，默认为 https://api.openai.com'
      }
    ]
  },
  google: {
    name: 'Google Cloud',
    fields: [
      {
        key: 'apiKey',
        label: 'API Key',
        type: 'password',
        placeholder: '请输入 Google API Key',
        required: true
      },
      {
        key: 'projectId',
        label: 'Project ID',
        type: 'text',
        placeholder: '请输入 Project ID',
        required: false
      }
    ]
  },
  aws: {
    name: 'Amazon AWS',
    fields: [
      {
        key: 'accessKeyId',
        label: 'Access Key ID',
        type: 'password',
        placeholder: '请输入 Access Key ID',
        required: true
      },
      {
        key: 'secretAccessKey',
        label: 'Secret Access Key',
        type: 'password',
        placeholder: '请输入 Secret Access Key',
        required: true
      },
      {
        key: 'region',
        label: 'Region',
        type: 'text',
        placeholder: 'us-east-1',
        required: false
      }
    ]
  },
  volcengine: {
    name: '火山引擎',
    fields: [
      {
        key: 'appId',
        label: 'App ID',
        type: 'text',
        placeholder: '请输入火山引擎 App ID',
        required: true,
        description: '火山引擎应用 ID'
      },
      {
        key: 'accessToken',
        label: 'Access Token',
        type: 'password',
        placeholder: '请输入 Access Token',
        required: true,
        description: '火山引擎 Access Token'
      },
      {
        key: 'cluster',
        label: 'Cluster',
        type: 'text',
        placeholder: 'volcano_tts',
        required: false,
        description: '集群名称，默认为 volcano_tts'
      },
      {
        key: 'language',
        label: '语言',
        type: 'text',
        placeholder: 'zh',
        required: false,
        description: '语言代码，如 zh、en 等'
      },
      {
        key: 'rate',
        label: '采样率',
        type: 'number',
        placeholder: '8000',
        required: false,
        description: '音频采样率，默认为 8000'
      },
      {
        key: 'encoding',
        label: '编码格式',
        type: 'text',
        placeholder: 'pcm',
        required: false,
        description: '音频编码格式，默认为 pcm'
      },
      {
        key: 'speedRatio',
        label: '语速',
        type: 'number',
        placeholder: '1.0',
        required: false,
        description: '语速比例，默认为 1.0'
      }
    ]
  },
  minimax: {
    name: 'Minimax',
    fields: [
      {
        key: 'apiKey',
        label: 'API Key',
        type: 'password',
        placeholder: '请输入 Minimax API Key',
        required: true,
        description: 'Minimax API Key'
      },
      {
        key: 'model',
        label: '模型',
        type: 'select',
        placeholder: 'speech-2.5-turbo-preview',
        required: false,
        options: [
          { value: 'speech-2.5-turbo-preview', label: 'speech-2.5-turbo-preview' },
          { value: 'speech-2.5-hd-preview', label: 'speech-2.5-hd-preview' }
        ],
        description: 'TTS 模型，默认为 speech-2.5-turbo-preview'
      },
      {
        key: 'voiceId',
        label: '音色 ID',
        type: 'text',
        placeholder: 'male-qn-qingse',
        required: false,
        description: '音色 ID，默认为 male-qn-qingse'
      },
      {
        key: 'speedRatio',
        label: '语速',
        type: 'number',
        placeholder: '1.0',
        required: false,
        description: '语速比例，默认为 1.0'
      },
      {
        key: 'volume',
        label: '音量',
        type: 'number',
        placeholder: '1.0',
        required: false,
        description: '音量，默认为 1.0'
      },
      {
        key: 'pitch',
        label: '音调',
        type: 'number',
        placeholder: '0.0',
        required: false,
        description: '音调，默认为 0.0'
      },
      {
        key: 'emotion',
        label: '情感',
        type: 'text',
        placeholder: 'neutral',
        required: false,
        description: '情感类型，默认为 neutral'
      },
      {
        key: 'languageBoost',
        label: '语言增强',
        type: 'text',
        placeholder: 'auto',
        required: false,
        description: '语言增强，默认为 auto'
      },
      {
        key: 'sampleRate',
        label: '采样率',
        type: 'number',
        placeholder: '8000',
        required: false,
        description: '音频采样率，默认为 8000'
      },
      {
        key: 'bitrate',
        label: '比特率',
        type: 'number',
        placeholder: '16',
        required: false,
        description: '音频比特率，默认为 16'
      },
      {
        key: 'format',
        label: '格式',
        type: 'text',
        placeholder: 'pcm',
        required: false,
        description: '音频格式，默认为 pcm'
      },
      {
        key: 'channels',
        label: '声道数',
        type: 'number',
        placeholder: '1',
        required: false,
        description: '声道数，默认为 1'
      }
    ]
  },
  elevenlabs: {
    name: 'ElevenLabs',
    fields: [
      {
        key: 'apiKey',
        label: 'API Key',
        type: 'password',
        placeholder: '请输入 ElevenLabs API Key',
        required: true,
        description: 'ElevenLabs API Key'
      },
      {
        key: 'voiceId',
        label: 'Voice ID',
        type: 'text',
        placeholder: '21m00Tcm4TlvDq8ikWAM',
        required: false,
        description: '音色 ID，默认为 21m00Tcm4TlvDq8ikWAM'
      },
      {
        key: 'modelId',
        label: 'Model ID',
        type: 'text',
        placeholder: 'eleven_monolingual_v1',
        required: false,
        description: '模型 ID，默认为 eleven_monolingual_v1'
      },
      {
        key: 'stability',
        label: '稳定性',
        type: 'number',
        placeholder: '0.5',
        required: false,
        description: '稳定性 (0.0-1.0)，默认为 0.5'
      },
      {
        key: 'similarityBoost',
        label: '相似度增强',
        type: 'number',
        placeholder: '0.75',
        required: false,
        description: '相似度增强 (0.0-1.0)，默认为 0.75'
      },
      {
        key: 'style',
        label: '风格',
        type: 'number',
        placeholder: '0.0',
        required: false,
        description: '风格 (0.0-1.0)，默认为 0.0'
      },
      {
        key: 'useSpeakerBoost',
        label: '使用说话人增强',
        type: 'select',
        required: false,
        options: [
          { value: 'true', label: '是' },
          { value: 'false', label: '否' }
        ],
        description: '是否使用说话人增强，默认为 true'
      }
    ]
  },
  local: {
    name: '本地 TTS',
    fields: [
      {
        key: 'command',
        label: '命令',
        type: 'select',
        placeholder: 'say',
        required: false,
        options: [
          { value: 'say', label: 'say (macOS)' },
          { value: 'espeak', label: 'espeak (Linux)' },
          { value: 'festival', label: 'festival (Linux)' }
        ],
        description: 'TTS 命令，默认为 say'
      },
      {
        key: 'voice',
        label: '音色',
        type: 'text',
        placeholder: '',
        required: false,
        description: '音色名称（可选）'
      },
      {
        key: 'sampleRate',
        label: '采样率',
        type: 'number',
        placeholder: '16000',
        required: false,
        description: '音频采样率，默认为 16000'
      },
      {
        key: 'channels',
        label: '声道数',
        type: 'number',
        placeholder: '1',
        required: false,
        description: '声道数，默认为 1'
      },
      {
        key: 'bitDepth',
        label: '位深度',
        type: 'number',
        placeholder: '16',
        required: false,
        description: '位深度，默认为 16'
      },
      {
        key: 'codec',
        label: '编解码器',
        type: 'text',
        placeholder: 'wav',
        required: false,
        description: '音频编解码器，默认为 wav'
      },
      {
        key: 'outputDir',
        label: '输出目录',
        type: 'text',
        placeholder: '/tmp',
        required: false,
        description: '输出目录，默认为 /tmp'
      }
    ]
  },
  fishspeech: {
    name: 'FishSpeech',
    fields: [
      {
        key: 'apiKey',
        label: 'API Key',
        type: 'password',
        placeholder: '请输入 FishSpeech API Key',
        required: true,
        description: 'FishSpeech API Key'
      },
      {
        key: 'referenceId',
        label: 'Reference ID',
        type: 'text',
        placeholder: 'default',
        required: false,
        description: '参考 ID，默认为 default'
      },
      {
        key: 'latency',
        label: '延迟模式',
        type: 'select',
        required: false,
        options: [
          { value: 'normal', label: '普通' },
          { value: 'balanced', label: '平衡' }
        ],
        description: '延迟模式，默认为 normal'
      },
      {
        key: 'version',
        label: '版本',
        type: 'text',
        placeholder: 's1',
        required: false,
        description: '版本，默认为 s1'
      },
      {
        key: 'sampleRate',
        label: '采样率',
        type: 'number',
        placeholder: '24000',
        required: false,
        description: '音频采样率，默认为 24000'
      },
      {
        key: 'codec',
        label: '编解码器',
        type: 'text',
        placeholder: 'wav',
        required: false,
        description: '音频编解码器，默认为 wav'
      }
    ]
  },
  coqui: {
    name: 'Coqui TTS',
    fields: [
      {
        key: 'url',
        label: 'URL',
        type: 'text',
        placeholder: 'http://localhost:5002/api/tts',
        required: true,
        description: 'Coqui TTS 服务地址'
      },
      {
        key: 'language',
        label: '语言',
        type: 'text',
        placeholder: 'en_US',
        required: false,
        description: '语言代码，默认为 en_US'
      },
      {
        key: 'speaker',
        label: '说话人',
        type: 'text',
        placeholder: 'p226',
        required: false,
        description: '说话人 ID，默认为 p226'
      },
      {
        key: 'sampleRate',
        label: '采样率',
        type: 'number',
        placeholder: '16000',
        required: false,
        description: '音频采样率，默认为 16000'
      },
      {
        key: 'channels',
        label: '声道数',
        type: 'number',
        placeholder: '1',
        required: false,
        description: '声道数，默认为 1'
      },
      {
        key: 'bitDepth',
        label: '位深度',
        type: 'number',
        placeholder: '16',
        required: false,
        description: '位深度，默认为 16'
      }
    ]
  }
}

// ASR 服务商配置
export const ASR_PROVIDERS: Record<string, ProviderConfig> = {
  qiniu: {
    name: '七牛云',
    fields: [
      {
        key: 'apiKey',
        label: 'API Key',
        type: 'password',
        placeholder: '请输入七牛云 API Key',
        required: true,
        description: '七牛云的 API Key'
      },
      {
        key: 'baseUrl',
        label: 'Base URL',
        type: 'text',
        placeholder: 'https://asr.qiniu.com',
        required: false,
        description: 'API 基础地址'
      },
      {
        key: 'model',
        label: '模型',
        type: 'text',
        placeholder: 'asr-model',
        required: false,
        description: '识别模型名称'
      },
      {
        key: 'language',
        label: '语言',
        type: 'select',
        required: false,
        options: [
          { value: 'zh', label: '中文' },
          { value: 'en', label: '英文' },
          { value: 'ja', label: '日文' },
          { value: 'ko', label: '韩文' }
        ]
      }
    ]
  },
  qcloud: {
    name: '腾讯云',
    fields: [
      {
        key: 'appId',
        label: 'App ID',
        type: 'text',
        placeholder: '请输入腾讯云 App ID',
        required: true,
        description: '腾讯云应用 ID'
      },
      {
        key: 'secretId',
        label: 'Secret ID',
        type: 'password',
        placeholder: '请输入 Secret ID',
        required: true,
        description: '腾讯云 Secret ID'
      },
      {
        key: 'secretKey',
        label: 'Secret Key',
        type: 'password',
        placeholder: '请输入 Secret Key',
        required: true,
        description: '腾讯云 Secret Key'
      },
      {
        key: 'language',
        label: '语言',
        type: 'select',
        required: false,
        options: [
          { value: 'zh', label: '中文' },
          { value: 'en', label: '英文' },
          { value: 'ja', label: '日文' },
          { value: 'ko', label: '韩文' }
        ]
      }
    ]
  },
  tencent: {
    name: '腾讯云（别名）',
    fields: [
      {
        key: 'appId',
        label: 'App ID',
        type: 'text',
        placeholder: '请输入腾讯云 App ID',
        required: true
      },
      {
        key: 'secretId',
        label: 'Secret ID',
        type: 'password',
        placeholder: '请输入 Secret ID',
        required: true
      },
      {
        key: 'secretKey',
        label: 'Secret Key',
        type: 'password',
        placeholder: '请输入 Secret Key',
        required: true
      }
    ]
  },
  baidu: {
    name: '百度',
    fields: [
      {
        key: 'apiKey',
        label: 'API Key',
        type: 'password',
        placeholder: '请输入百度 API Key',
        required: true
      },
      {
        key: 'secretKey',
        label: 'Secret Key',
        type: 'password',
        placeholder: '请输入 Secret Key',
        required: true
      }
    ]
  },
  azure: {
    name: '微软 Azure',
    fields: [
      {
        key: 'subscriptionKey',
        label: 'Subscription Key',
        type: 'password',
        placeholder: '请输入 Azure Subscription Key',
        required: true
      },
      {
        key: 'region',
        label: 'Region',
        type: 'text',
        placeholder: 'eastasia',
        required: true
      }
    ]
  },
  xunfei: {
    name: '科大讯飞',
    fields: [
      {
        key: 'appId',
        label: 'App ID',
        type: 'text',
        placeholder: '请输入讯飞 App ID',
        required: true
      },
      {
        key: 'apiKey',
        label: 'API Key',
        type: 'password',
        placeholder: '请输入 API Key',
        required: true
      },
      {
        key: 'apiSecret',
        label: 'API Secret',
        type: 'password',
        placeholder: '请输入 API Secret',
        required: true
      }
    ]
  }
}

// 获取所有 TTS 服务商选项
export const getTTSProviderOptions = () => {
  return Object.keys(TTS_PROVIDERS).map(key => ({
    value: key,
    label: TTS_PROVIDERS[key].name
  }))
}

// 获取所有 ASR 服务商选项
export const getASRProviderOptions = () => {
  return Object.keys(ASR_PROVIDERS).map(key => ({
    value: key,
    label: ASR_PROVIDERS[key].name
  }))
}

// 获取 TTS 服务商配置
export const getTTSProviderConfig = (provider: string): ProviderConfig | null => {
  return TTS_PROVIDERS[provider] || null
}

// 获取 ASR 服务商配置
export const getASRProviderConfig = (provider: string): ProviderConfig | null => {
  return ASR_PROVIDERS[provider] || null
}

