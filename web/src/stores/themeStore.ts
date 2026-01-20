import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type ThemeMode = 'light' | 'dark' | 'system'
export type ThemeColor = 'default' | 'cherry' | 'ocean' | 'nature' | 'fresh' | 'sunset' | 'lavender'

export type Theme = {
  mode: ThemeMode
  color: ThemeColor
}

interface ThemeState {
  theme: Theme
  setTheme: (theme: Theme) => void
  setMode: (mode: ThemeMode) => void
  setColor: (color: ThemeColor) => void
  isDark: boolean
  toggleMode: () => void
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set, get) => ({
      theme: { mode: 'system', color: 'default' },
      isDark: false,
      
      setTheme: (theme: Theme) => {
        set({ theme })
        const isDark = theme.mode === 'dark' || (theme.mode === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)
        set({ isDark })
        
        // 更新 DOM
        updateThemeClasses(theme.mode, theme.color, isDark)
      },
      
      setMode: (mode: ThemeMode) => {
        const { theme } = get()
        const newTheme = { ...theme, mode }
        get().setTheme(newTheme)
      },
      
      setColor: (color: ThemeColor) => {
        const { theme } = get()
        const newTheme = { ...theme, color }
        get().setTheme(newTheme)
      },
      
      toggleMode: () => {
        const { theme } = get()
        const newMode = theme.mode === 'light' ? 'dark' : 'light'
        get().setMode(newMode)
      },
    }),
    {
      name: 'theme-storage',
      onRehydrateStorage: () => (state) => {
        if (state) {
          const isDark = state.theme.mode === 'dark' || (state.theme.mode === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)
          state.isDark = isDark
          
          updateThemeClasses(state.theme.mode, state.theme.color, isDark)
        }
      },
    }
  )
)

// 更新主题类名的辅助函数
function updateThemeClasses(_mode: ThemeMode, color: ThemeColor, isDark: boolean) {
  const root = document.documentElement
  
  // 清除所有主题类
  root.classList.remove('dark', 'light', 'cherry', 'ocean', 'nature', 'fresh', 'sunset', 'lavender')
  
  // 添加颜色主题类（必须在模式类之前添加）
  if (color !== 'default') {
    root.classList.add(color)
  }
  
  // 添加模式类
  if (isDark) {
    root.classList.add('dark')
  } else {
    root.classList.add('light')
  }
}
