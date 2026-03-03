import { get, post, put, del, ApiResponse } from '@/utils/request'
import type { Scheme, CreateSchemeRequest, UpdateSchemeRequest, SchemeStats } from '@/types/scheme'

/**
 * 获取方案列表
 */
export const getSchemes = async (): Promise<ApiResponse<Scheme[]>> => {
  return get('/schemes')
}

/**
 * 获取方案详情
 */
export const getSchemeDetail = async (id: number): Promise<ApiResponse<Scheme>> => {
  return get(`/schemes/${id}`)
}

/**
 * 创建方案
 */
export const createScheme = async (data: CreateSchemeRequest): Promise<ApiResponse<Scheme>> => {
  return post('/schemes', data)
}

/**
 * 更新方案
 */
export const updateScheme = async (id: number, data: UpdateSchemeRequest): Promise<ApiResponse<Scheme>> => {
  return put(`/schemes/${id}`, data)
}

/**
 * 删除方案
 */
export const deleteScheme = async (id: number): Promise<ApiResponse<void>> => {
  return del(`/schemes/${id}`)
}

/**
 * 激活方案
 */
export const activateScheme = async (id: number): Promise<ApiResponse<void>> => {
  return post(`/schemes/${id}/activate`)
}

/**
 * 停用方案
 */
export const deactivateScheme = async (id: number): Promise<ApiResponse<void>> => {
  return post(`/schemes/${id}/deactivate`)
}

/**
 * 获取方案统计
 */
export const getSchemeStats = async (id: number): Promise<ApiResponse<SchemeStats>> => {
  return get(`/schemes/${id}/stats`)
}

/**
 * 获取当前激活的方案
 */
export const getActiveScheme = async (): Promise<ApiResponse<Scheme>> => {
  return get('/schemes/active')
}
