import axiosInstance from '@/utils/axios'
import { t } from '@/i18n'
import { InternalAxiosRequestConfig, AxiosResponse } from 'axios'

// 通用响应类型
export interface ApiResponse<T = any> {
  code: number
  msg: string
  data: T
}

// 请求函数 - 返回完整的响应结构
const request = async <T = any>(
  url: string,
  options: Partial<InternalAxiosRequestConfig> = {}
): Promise<ApiResponse<T>> => {
  try {
    const response: AxiosResponse<ApiResponse<T>> = await axiosInstance({
      url,
      ...options,
    })
    
    // 返回完整的响应结构，让业务层处理
    return response.data
  } catch (error: any) {
    // 如果是axios错误，尝试从响应中获取错误信息
    if (error.response?.data) {
      const errorData = error.response.data as Record<string, unknown>
      const status = error.response.status as number
      const msg =
        (typeof errorData.msg === 'string' && errorData.msg.trim()) ||
        (typeof errorData.message === 'string' && errorData.message.trim()) ||
        (typeof errorData.error === 'string' && errorData.error.trim()) ||
        (status === 502 || status === 503 || status === 504
          ? t('request.badGateway')
          : t('request.failed'))
      const code =
        typeof errorData.code === 'number'
          ? errorData.code
          : status || 502
      throw {
        code,
        msg,
        error: typeof errorData.error === 'string' ? errorData.error : undefined,
        data: errorData.data ?? null,
      }
    }

    // 无响应体：连接被拒 / 代理失败 / 超时
    let errorMessage = t('request.networkFailed')
    let code = -1
    if (
      error.code === 'ERR_CONNECTION_REFUSED' ||
      error.code === 'ECONNREFUSED' ||
      /ECONNREFUSED|ENOTFOUND|ECONNRESET/i.test(String(error.message || ''))
    ) {
      errorMessage = t('request.connectionRefused')
      code = 502
    } else if (error.code === 'ECONNABORTED') {
      errorMessage = t('request.timeout')
      code = 504
    } else if (error.message) {
      errorMessage = error.message
    }

    throw {
      code,
      msg: errorMessage,
      data: null,
    }
  }
}

// GET 请求
export const get = <T = any>(url: string, config?: Partial<InternalAxiosRequestConfig>): Promise<ApiResponse<T>> => {
  return request<T>(url, { ...config, method: 'GET' })
}

// POST 请求
export const post = <T = any>(url: string, data?: any, config?: Partial<InternalAxiosRequestConfig>): Promise<ApiResponse<T>> => {
  return request<T>(url, {
    ...config,
    method: 'POST',
    data,
  })
}

// PUT 请求
export const put = <T = any>(url: string, data?: any, config?: Partial<InternalAxiosRequestConfig>): Promise<ApiResponse<T>> => {
  return request<T>(url, {
    ...config,
    method: 'PUT',
    data,
  })
}

// DELETE 请求
export const del = <T = any>(url: string, config?: Partial<InternalAxiosRequestConfig>): Promise<ApiResponse<T>> => {
  return request<T>(url, { ...config, method: 'DELETE' })
}

// PATCH 请求
export const patch = <T = any>(url: string, data?: any, config?: Partial<InternalAxiosRequestConfig>): Promise<ApiResponse<T>> => {
  return request<T>(url, {
    ...config,
    method: 'PATCH',
    data,
  })
}

// 导出 request 对象和类型
export { request }
export default request
