import axios, { AxiosInstance, InternalAxiosRequestConfig, AxiosResponse } from 'axios'
import { useAuthStore } from '../stores/authStore'
import { getApiBaseURL } from '../config/apiConfig'

// 创建axios实例
const axiosInstance: AxiosInstance = axios.create({
  // 不设置 baseURL，由调用方传入完整路径（如 /api/xxx 或完整 http(s) 地址）
  baseURL: '',
  timeout: 100000,
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true, // 重要：允许发送和接收 cookies（session）
})

// 请求拦截器
axiosInstance.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    // 添加认证token
    const token = localStorage.getItem('auth_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    // 无 token 时不发送 Authorization header，由后端决定是否拒绝
    
    // 如果是FormData，让浏览器自动设置Content-Type（包含boundary）
    if (config.data instanceof FormData) {
      delete config.headers['Content-Type']
    }
    
    // 添加请求时间戳
    if (config.params) {
      config.params._t = Date.now()
    } else {
      config.params = { _t: Date.now() }
    }
    
    return config
  },
  (error) => {
    if (import.meta.env.DEV) {
      console.error('Request interceptor error:', error)
    }
    return Promise.reject(error)
  }
)

// 响应拦截器 - 只处理通用错误，不处理业务逻辑
axiosInstance.interceptors.response.use(
  (response: AxiosResponse) => {
    // 直接返回完整响应，让业务层处理
    return response
  },
  (error) => {
    if (import.meta.env.DEV) {
      console.error('Response interceptor error:', error)
    }
    // 处理网络错误和HTTP状态码错误
    if (error.response) {
      const status = error.response.status

      switch (status) {
        case 401:
          useAuthStore.getState().clearUser()
          break
        case 403:
          console.error('Forbidden: Access denied')
          break
        case 404:
          console.error('Not Found: API endpoint not found')
          break
        case 500:
          console.error('Internal Server Error')
          break
        default:
          if (import.meta.env.DEV) {
            console.error(`HTTP Error ${status}:`, error.response.data)
          }
      }
    } else if (error.request) {
      if (import.meta.env.DEV) {
        console.error('Network Error:', error.message)
      }
    } else {
      if (import.meta.env.DEV) {
        console.error('Error:', error.message)
      }
    }
    
    return Promise.reject(error)
  }
)

export default axiosInstance
