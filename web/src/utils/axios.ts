import axios, { AxiosInstance, InternalAxiosRequestConfig, AxiosResponse } from 'axios'
import { useAuthStore } from '../stores/authStore'
import { syncAuthToken } from '@/utils/authToken'
import { useLocaleStore } from '../stores/localeStore'
import { getApiBaseURL } from '../config/apiConfig'
import { getOrCreateDeviceId } from '@/utils/deviceId'
import { genReqId, X_REQ_ID_HEADER } from '@/utils/reqId'

// 创建axios实例
const axiosInstance: AxiosInstance = axios.create({
  // 统一走配置的后端地址；当 url 为绝对地址时 axios 会优先使用 url 本身
  baseURL: getApiBaseURL(),
  // 30s default; long-running endpoints should pass their own timeout via config.
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true, // 重要：允许发送和接收 cookies（session）
})

// 请求拦截器
axiosInstance.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const locale = useLocaleStore.getState().locale
    config.headers['Accept-Language'] = locale

    const token = useAuthStore.getState().token
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
      syncAuthToken(token)
    } else {
      delete config.headers.Authorization
      syncAuthToken(null)
    }
    
    // 如果是FormData，让浏览器自动设置Content-Type（包含boundary）
    if (config.data instanceof FormData) {
      delete config.headers['Content-Type']
    }

    const deviceId = getOrCreateDeviceId()
    if (deviceId) {
      config.headers['X-Device-Id'] = deviceId
    }

    // 与后端 RequestIDMiddleware 对齐，便于日志串联（未显式传入时自动生成）
    const headers = config.headers as Record<string, string | undefined>
    if (!headers[X_REQ_ID_HEADER] && !headers['X-Req-ID']) {
      headers[X_REQ_ID_HEADER] = genReqId()
    }
    
    // 添加请求时间戳
    if (config.params) {
      config.params._t = Date.now()
    } else {
      config.params = { _t: Date.now() }
    }
    
    // 调试信息仅在开发模式下打印，避免生产环境泄漏请求细节。
    if (import.meta.env.DEV) {
      // @ts-expect-error config.baseURL may be undefined in Axios types when using absolute URLs
      console.debug('[req]', config.method, config.baseURL + config.url, {
        'x-reqid': headers[X_REQ_ID_HEADER] ?? headers['X-Req-ID'],
      })
    }

    return config
  },
  (error) => {
    if (import.meta.env.DEV) console.error('Request interceptor error:', error)
    return Promise.reject(error)
  }
)

// 响应拦截器 - 只处理通用错误，不处理业务逻辑
axiosInstance.interceptors.response.use(
  (response: AxiosResponse) => {
    const rid =
      response.headers[X_REQ_ID_HEADER] ??
      response.headers['x-reqid'] ??
      response.headers['X-Req-ID']
    if (rid && import.meta.env.DEV) {
      console.debug('[x-reqid]', rid, response.config.method, response.config.url)
    }
    return response
  },
  (error) => {
    if (import.meta.env.DEV) console.error('Response interceptor error:', error)
    if (error.response) {
      const status = error.response.status
      if (status === 401) {
        const payload = error.response.data as { data?: { sessionRevoked?: boolean } } | undefined
        const sessionRevoked = payload?.data?.sessionRevoked === true
        useAuthStore.getState().clearUser()
        if (typeof window !== 'undefined') {
          const onAuthPage =
            window.location.pathname === '/login' ||
            window.location.pathname === '/register' ||
            window.location.pathname.startsWith('/account/deletion/revoke')
          if (!onAuthPage) {
            if (sessionRevoked) {
              try {
                sessionStorage.setItem('lingecho.auth.sessionRevoked', '1')
              } catch {
                /* ignore */
              }
            }
            const target = sessionRevoked ? '/login?reason=session_revoked' : '/login'
            window.location.assign(target)
          }
        }
      } else if (import.meta.env.DEV) {
        console.error(`HTTP ${status}:`, error.response.data)
      }
    } else if (import.meta.env.DEV) {
      console.error('Network/Other Error:', error.message)
    }

    return Promise.reject(error)
  }
)

export default axiosInstance
