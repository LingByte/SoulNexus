import axios, { AxiosInstance, InternalAxiosRequestConfig, AxiosResponse } from 'axios'
import { useAuthStore } from '../stores/authStore'
import {
  getAccountDeletionRevokePageURL,
  getApiBaseURL,
  getUserServiceBaseURL,
  isAccountDeletionRevokeStandalonePage,
} from '../config/apiConfig'

// 获取API基础URL
const getApiBaseUrl = () => {
  return getApiBaseURL()
}

// 创建axios实例
const axiosInstance: AxiosInstance = axios.create({
  baseURL: getApiBaseUrl(),
  timeout: 100000,
  headers: {
    'Content-Type': 'application/json',
  },
})

let isRefreshingToken = false
let refreshPromise: Promise<string> | null = null

const requestTokenRefresh = async (): Promise<string> => {
  const refreshToken = localStorage.getItem('refresh_token')
  if (!refreshToken) {
    throw new Error('missing refresh token')
  }
  const resp = await axios.post(`${getUserServiceBaseURL()}/auth/refresh`, {
    refresh_token: refreshToken,
  }, {
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: 100000,
  })
  const data = resp.data
  const nextAccessToken = data?.data?.token || data?.data?.access_token
  const nextRefreshToken = data?.data?.refreshToken || data?.data?.refresh_token
  if (!nextAccessToken || !nextRefreshToken) {
    throw new Error('invalid refresh response')
  }
  localStorage.setItem('auth_token', nextAccessToken)
  localStorage.setItem('refresh_token', nextRefreshToken)
  return nextAccessToken
}

// 请求拦截器
axiosInstance.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    // 添加认证token
    const token = localStorage.getItem('auth_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    // 移除测试token逻辑，让需要认证的接口正确返回401
    
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
    
    // 添加调试信息
    // @ts-ignore
      console.log('Making request to:', config.baseURL + config.url, {
      method: config.method,
      headers: config.headers,
      params: config.params
    })
    
    return config
  },
  (error) => {
    console.error('Request interceptor error:', error)
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
      console.error('Response interceptor error:', error)
    // 处理网络错误和HTTP状态码错误
    if (error.response) {
        console.log('Response status:', error.response.status)
      // 服务器返回了错误状态码
      const status = error.response.status

      switch (status) {
        case 401:
          {
            const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean }
            if (!originalRequest?._retry && !String(originalRequest?.url || '').includes('/auth/refresh')) {
              originalRequest._retry = true
              if (!isRefreshingToken) {
                isRefreshingToken = true
                refreshPromise = requestTokenRefresh()
                  .finally(() => {
                    isRefreshingToken = false
                  })
              }
              return refreshPromise!
                .then((nextAccessToken) => {
                  originalRequest.headers.Authorization = `Bearer ${nextAccessToken}`
                  return axiosInstance(originalRequest)
                })
                .catch(() => {
                  useAuthStore.getState().clearUser()
                  return Promise.reject(error)
                })
            }
            console.log('Unauthorized')
            useAuthStore.getState().clearUser()
            console.log('Unauthorized: Please log in')
            break
          }
        case 403: {
          const body = error.response?.data as any
          const payload = body?.data
          if (payload?.accountDeletionPending) {
            useAuthStore
              .getState()
              .refreshUserInfo()
              .catch(() => {})
            if (!isAccountDeletionRevokeStandalonePage()) {
              const email = useAuthStore.getState().user?.email
              window.location.assign(getAccountDeletionRevokePageURL(email))
            }
          }
          console.error('Forbidden: Access denied')
          break
        }
        case 404:
          console.error('Not Found: API endpoint not found')
          break
        case 500:
          console.error('Internal Server Error')
          break
        default:
          console.error(`HTTP Error ${status}:`, error.response.data)
      }
    } else if (error.request) {
      // 网络错误 - 连接被拒绝或超时
      console.error('Network Error:', error.message)
    } else {
      // 其他错误
      console.error('Error:', error.message)
    }
    
    return Promise.reject(error)
  }
)

export default axiosInstance
