import { get, post, put, ApiResponse } from '@/utils/request'

// 搜索配置接口
export interface SearchConfig {
  enabled: boolean
  searchPath: string
  batchSize: number
  schedule: string
}

// 获取搜索状态
export const getSearchStatus = async (): Promise<ApiResponse<SearchConfig>> => {
  return get('/system/search/status')
}

// 更新搜索配置
export const updateSearchConfig = async (config: Partial<SearchConfig>): Promise<ApiResponse<void>> => {
  return put('/system/search/config', config)
}

// 启用搜索
export const enableSearch = async (): Promise<ApiResponse<void>> => {
  return post('/system/search/enable')
}

// 禁用搜索
export const disableSearch = async (): Promise<ApiResponse<void>> => {
  return post('/system/search/disable')
}

// 系统初始化信息
export interface SystemInitInfo {
  database: {
    driver: string
    isMemoryDB: boolean
  }
  email: {
    configured: boolean
  }
  voiceClone: {
    xunfei: {
      configured: boolean
      config?: {
        app_id?: string
        api_key?: string
        base_url?: string
        ws_app_id?: string
        ws_api_key?: string
        ws_api_secret?: string
      }
    }
    volcengine: {
      configured: boolean
      config?: {
        app_id?: string
        token?: string
        cluster?: string
        voice_type?: string
        encoding?: string
        frame_duration?: string
        sample_rate?: number
        bit_depth?: number
        channels?: number
        speed_ratio?: number
        training_times?: number
      }
    }
  }
}

// 获取系统初始化信息
export const getSystemInit = async (): Promise<ApiResponse<SystemInitInfo>> => {
  return get('/system/init')
}

// 保存音色克隆配置
export interface SaveVoiceCloneConfigRequest {
  provider: 'xunfei' | 'volcengine'
  config: Record<string, any>
}

export const saveVoiceCloneConfig = async (data: SaveVoiceCloneConfigRequest): Promise<ApiResponse<void>> => {
  return post('/system/voice-clone/config', data)
}

