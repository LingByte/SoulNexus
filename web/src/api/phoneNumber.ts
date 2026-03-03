import { get, post, put, del, ApiResponse } from '@/utils/request'
import type { PhoneNumber, CreatePhoneNumberRequest, UpdatePhoneNumberRequest, PhoneNumberStats } from '@/types/phoneNumber'

/**
 * 获取号码列表
 */
export const getPhoneNumbers = async (): Promise<ApiResponse<PhoneNumber[]>> => {
  return get('/phone-numbers')
}

/**
 * 获取号码详情
 */
export const getPhoneNumberDetail = async (id: number): Promise<ApiResponse<PhoneNumber>> => {
  return get(`/phone-numbers/${id}`)
}

/**
 * 添加号码
 */
export const createPhoneNumber = async (data: CreatePhoneNumberRequest): Promise<ApiResponse<PhoneNumber>> => {
  return post('/phone-numbers', data)
}

/**
 * 更新号码
 */
export const updatePhoneNumber = async (id: number, data: UpdatePhoneNumberRequest): Promise<ApiResponse<PhoneNumber>> => {
  return put(`/phone-numbers/${id}`, data)
}

/**
 * 删除号码
 */
export const deletePhoneNumber = async (id: number): Promise<ApiResponse<void>> => {
  return del(`/phone-numbers/${id}`)
}

/**
 * 设置主号码
 */
export const setPrimaryPhoneNumber = async (id: number): Promise<ApiResponse<void>> => {
  return post(`/phone-numbers/${id}/set-primary`)
}

/**
 * 绑定方案
 */
export const bindScheme = async (id: number, schemeId: number): Promise<ApiResponse<void>> => {
  return post(`/phone-numbers/${id}/bind-scheme`, { schemeId })
}

/**
 * 解绑方案
 */
export const unbindScheme = async (id: number): Promise<ApiResponse<void>> => {
  return post(`/phone-numbers/${id}/unbind-scheme`)
}

/**
 * 获取号码统计
 */
export const getPhoneNumberStats = async (id: number): Promise<ApiResponse<PhoneNumberStats>> => {
  return get(`/phone-numbers/${id}/stats`)
}
