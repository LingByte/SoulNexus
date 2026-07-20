import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { fetchMe, logoutApi } from '@/api/me'
import { syncAuthToken } from '@/utils/authToken'

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

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      isAuthenticated: false,
      isLoading: false,
      token: null,

      login: async (token: string, user?: User) => {
        syncAuthToken(token)
        set({ isLoading: false, isAuthenticated: true, token, user: user || null })
        return true
      },

      logout: async () => {
        try {
          await logoutApi()
        } catch {
          // ignore — clear local state regardless
        } finally {
          syncAuthToken(null)
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
        syncAuthToken(null)
        set({ user: null, isAuthenticated: false, token: null })
      },

      // 从 token 恢复，并尽量使用 /me 刷新当前用户信息
      refreshUserInfo: async () => {
        const state = get()
        const token = state.token
        if (!token) return
        try {
          const res = await fetchMe()
          if (res.code === 200 && res.data) {
            const data = res.data
            if (data.principal === 'platform' && data.platformAdmin) {
              const a = data.platformAdmin
              syncAuthToken(token)
              set({
                user: {
                  id: a.id,
                  email: a.email,
                  displayName: a.displayName,
                  isPlatformAdmin: true,
                  principal: 'platform' as const,
                  permissionCodes: [],
                },
                isAuthenticated: true,
                token,
              })
              return
            }
            if (data.principal === 'tenant' && data.user) {
              syncAuthToken(token)
              set({
                user: {
                  ...data.user,
                  tenantSlug: data.tenant?.slug,
                  tenantName: data.tenant?.name,
                  tenantVoiceMode: data.tenant?.voiceMode,
                  principal: 'tenant' as const,
                  permissionCodes: data.permissionCodes ?? [],
                },
                isAuthenticated: true,
                token,
              })
              return
            }
          }
        } catch {
          // fallthrough — zustand persist keeps last known state
        }
      }
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
        token: state.token,
      }),
      onRehydrateStorage: () => (state) => {
        syncAuthToken(state?.token ?? null)
      },
    }
  )
)
