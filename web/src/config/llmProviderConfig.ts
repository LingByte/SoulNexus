// LLM服务商配置和提示
export interface LLMProviderOption {
  value: string
  label: string
  description?: string
}

export const LLM_PROVIDER_SUGGESTIONS: LLMProviderOption[] = [
  {
    value: 'openi',
    label: 'OpenI (OpenAI Compatible)',
    description: 'OpenAI兼容协议，默认地址: https://api.openai.com/v1',
  },
  {
    value: 'openai',
    label: 'OpenAI',
    description: 'GPT-4, GPT-3.5等模型，API地址: https://api.openai.com/v1',
  },
  {
    value: 'alibaba',
    label: '阿里云通义千问',
    description: '阿里云 DashScope，API地址: https://dashscope.aliyuncs.com',
  },
  {
    value: 'anthropic',
    label: 'Anthropic',
    description: 'Claude系列模型，API地址: https://api.anthropic.com',
  },
  {
    value: 'ollama',
    label: 'Ollama',
    description: '本地部署模型，API地址: http://localhost:11434/v1',
  },
  {
    value: 'lmstudio',
    label: 'LM Studio',
    description: '本地LM Studio OpenAI兼容服务，API地址: http://localhost:1234/v1',
  },
  {
    value: 'localai',
    label: 'LocalAI',
    description: '本地AI服务，API地址: http://localhost:8080/v1',
  },
  {
    value: 'google',
    label: 'Google Gemini',
    description: 'Gemini系列模型，API地址: https://generativelanguage.googleapis.com/v1',
  },
  {
    value: 'xai',
    label: 'xAI (Grok)',
    description: 'Grok模型，API地址: https://api.x.ai/v1',
  },
  {
    value: 'coze',
    label: 'Coze',
    description: 'Coze智能体平台，需要配置Bot ID',
  },
]

/**
 * 根据provider值获取默认的API URL
 */
export const getDefaultApiUrl = (provider: string): string => {
  if (!provider) return ''
  
  const providerLower = provider.toLowerCase()
  
  // Coze 不需要默认 URL，返回空字符串
  if (providerLower === 'coze') {
    return ''
  }
  
  // Ollama 使用默认本地地址
  if (providerLower === 'ollama') {
    return 'http://localhost:11434/v1'
  }

  // LM Studio 使用默认本地地址
  if (providerLower === 'lmstudio') {
    return 'http://localhost:1234/v1'
  }

  if (providerLower === 'openi') {
    return 'https://api.openai.com/v1'
  }
  
  const suggestion = LLM_PROVIDER_SUGGESTIONS.find(
    p => p.value.toLowerCase() === providerLower
  )
  
  if (suggestion && suggestion.description) {
    // 从description中提取URL
    const urlMatch = suggestion.description.match(/https?:\/\/[^\s]+/)
    if (urlMatch) {
      return urlMatch[0]
    }
  }
  
  // 默认返回空字符串，让用户自己填写
  return ''
}

/**
 * 检查是否为 Coze 提供者
 */
export const isCozeProvider = (provider: string): boolean => {
  return provider?.toLowerCase() === 'coze'
}

/**
 * 检查是否为 Ollama 提供者
 */
export const isOllamaProvider = (provider: string): boolean => {
  return provider?.toLowerCase() === 'ollama'
}

/**
 * 检查是否为 LM Studio 提供者
 */
export const isLMStudioProvider = (provider: string): boolean => {
  return provider?.toLowerCase() === 'lmstudio'
}

/**
 * 根据provider值获取提示信息
 */
export const getProviderInfo = (provider: string): LLMProviderOption | undefined => {
  if (!provider) return undefined
  
  const providerLower = provider.toLowerCase()
  return LLM_PROVIDER_SUGGESTIONS.find(
    p => p.value.toLowerCase() === providerLower ||
         p.label.toLowerCase() === providerLower
  )
}

