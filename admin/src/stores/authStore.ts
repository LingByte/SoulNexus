import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { getCurrentUser } from '@/services/adminApi'

// 用户类型定义（可以根据实际需求修改）
export interface User {
  id: string | number
  username?: string
  email?: string
  avatar?: string
  [key: string]: any
}

interface AuthState {
  user: User | null
  isAuthenticated: boolean
  isLoading: boolean
  token: string | null
  login: (token: string, user?: User) => Promise<boolean>
  logout: () => Promise<void>
  setLoading: (loading: boolean) => void
  updateProfile: (data: Partial<User>) => void
  clearUser: () => void
  refreshUserInfo: () => Promise<void>
}

// @ts-ignore
export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      isAuthenticated: false,
      isLoading: false,
      token: null,

      login: async (token: string, user?: User) => {
        set({ isLoading: true })
        try {
          // 存储token
          localStorage.setItem('auth_token', token)
          
          // 存储用户信息
          if (user) {
            localStorage.setItem('auth_user', JSON.stringify(user))
          }
          
          set({
            isAuthenticated: true, 
            isLoading: false,
            token: token,
            user: user || null
          })
          
          return true
        } catch (error) {
          set({ isLoading: false })
          console.error('Login failed:', error)
          return false
        }
      },

      logout: async () => {
        try {
          const token = localStorage.getItem('auth_token')
          if (token) {
            await fetch(`${import.meta.env.VITE_API_BASE_URL || '/api'}/auth/logout`, {
              method: 'POST',
              headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
            }).catch(() => {}) // 忽略登出接口错误
          }
        } finally {
          localStorage.removeItem('auth_token')
          localStorage.removeItem('auth_user')
          set({ user: null, isAuthenticated: false, token: null })
        }
      },

      setLoading: (loading: boolean) => {
        set({ isLoading: loading })
      },

      updateProfile: (data: Partial<User>) => {
        const { user } = get()
        if (user) {
          set({ user: { ...user, ...data } })
        }
      },

      // 清除用户信息方法
      clearUser: () => {
        localStorage.removeItem('auth_token')
        set({ user: null, isAuthenticated: false, token: null })
      },

      // 刷新用户信息
      refreshUserInfo: async () => {
        const token = localStorage.getItem('auth_token')
        if (!token) return

        try {
          const userData = await getCurrentUser()
          localStorage.setItem('auth_user', JSON.stringify(userData))
          set({ 
            user: userData, 
            isAuthenticated: true, 
            token 
          })
        } catch (error) {
          console.error('Failed to refresh user info:', error)
          // 如果获取失败，尝试从localStorage恢复
          const storedUser = localStorage.getItem('auth_user')
          if (storedUser) {
            try {
              set({ 
                user: JSON.parse(storedUser), 
                isAuthenticated: true, 
                token 
              })
            } catch (e) {
              localStorage.removeItem('auth_token')
              set({ user: null, isAuthenticated: false, token: null })
            }
          } else {
            localStorage.removeItem('auth_token')
            set({ user: null, isAuthenticated: false, token: null })
          }
        }
      }
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({ 
        user: state.user, 
        isAuthenticated: state.isAuthenticated,
        token: state.token
      }),
    }
  )
)

// 导出User类型供其他组件使用（interface 已在上方 export，此处无需重复）
