import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { registerUser, getUserInfo, logoutUser, type User, type RegisterUserForm } from '../api/auth'

interface AuthState {
  user: User | null
  isAuthenticated: boolean
  isLoading: boolean
  token: string | null
  currentOrganizationId: number | null
  login: (token: string) => Promise<boolean>
  register: (data: RegisterUserForm) => Promise<boolean>
  logout: (next?: string) => Promise<void>
  setLoading: (loading: boolean) => void
  refreshUserInfo: () => Promise<void>
  updateProfile: (data: Partial<User>) => void
  clearUser: () => void
  setCurrentOrganization: (organizationId: number | null) => void
}

// @ts-ignore
export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      isAuthenticated: false,
      isLoading: false,
      token: null,
      currentOrganizationId: null,

      login: async (token: string) => {
        set({ isLoading: true })
        try {
          // 存储token
          localStorage.setItem('auth_token', token)
          
          set({
            isAuthenticated: true, 
            isLoading: false,
            token: token
          })
          
          // 从token中解析用户信息或调用API获取
          const userResponse = await getUserInfo()
          if (userResponse.code === 200) {
            set({
              user: userResponse.data
            })
            return true
          } else {
            throw new Error(userResponse.msg || '获取用户信息失败')
          }
        } catch (error) {
          set({ isLoading: false })
          console.error('Login failed:', error)
          return false
        }
      },

      register: async (data: RegisterUserForm) => {
        set({ isLoading: true })
        try {
          const response = await registerUser(data)
          
          if (response.code === 200) {
            // 注册成功，但通常注册后需要用户主动登录
            set({ isLoading: false })
            return true
          } else {
            throw new Error(response.msg || '注册失败')
          }
        } catch (error) {
          set({ isLoading: false })
          console.error('Registration failed:', error)
          return false
        }
      },

      logout: async (next?: string) => {
        try {
          // 调用登出API
          const response = await logoutUser(next)
          if (response.code !== 200) {
            console.warn('Logout API warning:', response.msg)
          }
        } catch (error) {
          console.error('Logout API error:', error)
        } finally {
          // 清除本地存储
          localStorage.removeItem('auth_token')
          set({ user: null, isAuthenticated: false, token: null, currentOrganizationId: null })
        }
      },

      setLoading: (loading: boolean) => {
        set({ isLoading: loading })
      },

      refreshUserInfo: async () => {
        const token = localStorage.getItem('auth_token')
        if (!token) return

        try {
          const response = await getUserInfo()
          if (response.code === 200) {
            set({ user: response.data, isAuthenticated: true, token })
          } else {
            throw new Error(response.msg || '获取用户信息失败')
          }
        } catch (error) {
          console.error('Failed to refresh user info:', error)
          // 如果获取用户信息失败，清除认证状态
          localStorage.removeItem('auth_token')
          set({ user: null, isAuthenticated: false, token: null })
        }
      },

      updateProfile: (data: Partial<User>) => {
        const { user } = get()
        if (user) {
          set({ user: { ...user, ...data } })
        }
      },

        // 新增的清除用户信息方法
        clearUser: () => {
            localStorage.removeItem('auth_token')
            set({ user: null, isAuthenticated: false, token: null, currentOrganizationId: null })
        },

        // 设置当前选择的组织
        setCurrentOrganization: (organizationId: number | null) => {
            set({ currentOrganizationId: organizationId })
        }
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({ 
        user: state.user, 
        isAuthenticated: state.isAuthenticated,
        token: state.token,
        currentOrganizationId: state.currentOrganizationId
      }),
    }
  )
)

// 导出User类型供其他组件使用
export type { User }