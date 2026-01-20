// LLM服务商配置和提示
export interface LLMProviderOption {
  value: string
  label: string
  description?: string
}

export const LLM_PROVIDER_SUGGESTIONS: LLMProviderOption[] = [
  {
    value: 'openai',
    label: 'OpenAI',
    description: 'GPT-4, GPT-3.5等模型，API地址: https://api.openai.com/v1',
  },
  {
    value: 'anthropic',
    label: 'Anthropic',
    description: 'Claude系列模型，API地址: https://api.anthropic.com',
  },
  {
    value: 'deepseek',
    label: 'DeepSeek',
    description: 'DeepSeek系列模型，API地址: https://api.deepseek.com/v1',
  },
  {
    value: 'groq',
    label: 'Groq',
    description: 'Groq快速推理API，API地址: https://api.groq.com/openai/v1',
  },
  {
    value: 'together',
    label: 'Together AI',
    description: 'Together AI推理服务，API地址: https://api.together.xyz/v1',
  },
  {
    value: 'replicate',
    label: 'Replicate',
    description: 'Replicate模型托管服务，API地址: https://api.replicate.com/v1',
  },
  {
    value: 'cohere',
    label: 'Cohere',
    description: 'Cohere语言模型，API地址: https://api.cohere.ai/v1',
  },
  {
    value: 'mistral',
    label: 'Mistral AI',
    description: 'Mistral系列模型，API地址: https://api.mistral.ai/v1',
  },
  {
    value: 'perplexity',
    label: 'Perplexity',
    description: 'Perplexity搜索增强模型，API地址: https://api.perplexity.ai',
  },
  {
    value: 'openrouter',
    label: 'OpenRouter',
    description: '统一API接口，支持多种模型，API地址: https://openrouter.ai/api/v1',
  },
  {
    value: 'azure',
    label: 'Azure OpenAI',
    description: '微软Azure OpenAI服务，需要配置Azure端点',
  },
  {
    value: 'aliyun',
    label: '阿里云通义千问',
    description: '通义千问模型，API地址: https://dashscope.aliyuncs.com/compatible-mode/v1',
  },
  {
    value: 'baidu',
    label: '百度文心一言',
    description: '文心一言模型，API地址: https://aip.baidubce.com/rpc/2.0/ai_custom/v1',
  },
  {
    value: 'tencent',
    label: '腾讯混元',
    description: '腾讯混元模型，API地址: https://hunyuan.tencentcloudapi.com',
  },
  {
    value: 'zhipu',
    label: '智谱AI',
    description: 'GLM系列模型，API地址: https://open.bigmodel.cn/api/paas/v4',
  },
  {
    value: 'moonshot',
    label: 'Moonshot AI',
    description: 'Moonshot模型，API地址: https://api.moonshot.cn/v1',
  },
  {
    value: 'qwen',
    label: 'Qwen (通义千问)',
    description: '通义千问开源模型，API地址: https://dashscope.aliyuncs.com/compatible-mode/v1',
  },
  {
    value: 'ollama',
    label: 'Ollama',
    description: '本地部署模型，API地址: http://localhost:11434/v1',
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

