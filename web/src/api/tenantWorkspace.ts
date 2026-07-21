import { get, put, type ApiResponse } from '@/utils/request'

export interface WorkspaceAIConfig {
  voiceMode?: string
  asrConfig?: Record<string, unknown> | null
  ttsConfig?: Record<string, unknown> | null
  llmConfig?: Record<string, unknown> | null
  realtimeConfig?: Record<string, unknown> | null
}

export async function getWorkspaceAIConfig(): Promise<ApiResponse<WorkspaceAIConfig>> {
  return get('/tenant/workspace/ai')
}

export async function updateWorkspaceAIConfig(body: WorkspaceAIConfig): Promise<ApiResponse<WorkspaceAIConfig>> {
  return put('/tenant/workspace/ai', body)
}
