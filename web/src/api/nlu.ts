import { get, post } from '@/utils/request'

export type NluPrediction = {
  text: string
  intent_index: number
  intent_name: string
  confidence: number
  keyword_bias_applied?: boolean
  used_config_fallback?: boolean
  softmax?: number[]
}

export type NluParseResult = {
  channel: 'intent' | 'llm' | 'unknown'
  reply?: string
  prediction: NluPrediction
}

export type NluStatus = {
  deployEnabled: boolean
  enabled: boolean
  ready: boolean
  modelPath: string
  tokenizerPath: string
  intentsPath?: string
  seqLen: number
  minConfidence: number
  engineOnline: boolean
  numClasses?: number
  intents?: string[]
  engineError?: string
}

export async function fetchNluStatus(): Promise<NluStatus> {
  const res = await get<NluStatus>('/admin/nlu/status')
  return res.data!
}

export async function parseNluText(text: string): Promise<NluParseResult> {
  const res = await post<NluParseResult>('/admin/nlu/parse', { text })
  return res.data!
}

export async function reloadNluEngine(): Promise<{ message?: string; numClasses?: number; intents?: string[] }> {
  const res = await post<{ message?: string; numClasses?: number; intents?: string[] }>('/admin/nlu/reload')
  return res.data!
}
