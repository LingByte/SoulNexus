import { get, post, ApiResponse } from '@/utils/request'

/**
 * 设置呼叫转移请求
 */
export interface SetupRequest {
  phoneNumberId: number
  targetNumber: string
  carrier?: string
}

/**
 * 设置步骤
 */
export interface SetupStep {
  order: number
  description: string
  action?: string
}

/**
 * 设置指引响应
 */
export interface SetupInstructions {
  code: string
  description: string
  steps: SetupStep[]
}

/**
 * 验证响应
 */
export interface VerifyResponse {
  isEnabled: boolean
  status: string
  targetNumber: string
  verifiedAt: string
  message: string
}

/**
 * 运营商代码
 */
export interface CarrierCodes {
  enable: string
  disable: string
  query: string
}

/**
 * 获取呼叫转移设置指引
 */
export const getSetupInstructions = async (data: SetupRequest): Promise<ApiResponse<SetupInstructions>> => {
  return post('/call-forward/setup-instructions', data)
}

/**
 * 获取取消呼叫转移指引
 */
export const getDisableInstructions = async (phoneNumberId: number): Promise<ApiResponse<SetupInstructions>> => {
  return get(`/call-forward/${phoneNumberId}/disable-instructions`)
}

/**
 * 更新呼叫转移状态
 */
export const updateCallForwardStatus = async (
  phoneNumberId: number,
  enabled: boolean,
  targetNumber: string
): Promise<ApiResponse<void>> => {
  return post(`/call-forward/${phoneNumberId}/status`, {
    enabled,
    targetNumber
  })
}

/**
 * 验证呼叫转移状态
 */
export const verifyCallForwardStatus = async (phoneNumberId: number): Promise<ApiResponse<VerifyResponse>> => {
  return post(`/call-forward/${phoneNumberId}/verify`)
}

/**
 * 测试呼叫转移
 */
export const testCallForward = async (phoneNumberId: number): Promise<ApiResponse<void>> => {
  return post(`/call-forward/${phoneNumberId}/test`)
}

/**
 * 获取运营商代码
 */
export const getCarrierCodes = async (carrier: string): Promise<ApiResponse<CarrierCodes>> => {
  return get(`/call-forward/carrier-codes?carrier=${carrier}`)
}
