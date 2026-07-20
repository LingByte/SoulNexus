import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type ThemeMode = 'light' | 'dark' | 'system'

export type Theme = {
  mode: ThemeMode
}

interface ThemeState {
  theme: Theme
  setTheme: (theme: Theme) => void
  setMode: (mode: ThemeMode) => void
  isDark: boolean
  toggleMode: () => void
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set, get) => ({
      theme: { mode: 'system' },
      isDark: false,

      setTheme: (theme: Theme) => {
        set({ theme })
        const isDark =
          theme.mode === 'dark' ||
          (theme.mode === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)
        set({ isDark })
        updateThemeClasses(isDark)
      },

      setMode: (mode: ThemeMode) => {
        get().setTheme({ mode })
      },

      toggleMode: () => {
        const { theme, isDark } = get()
        const newMode =
          theme.mode === 'system' ? (isDark ? 'light' : 'dark') : theme.mode === 'light' ? 'dark' : 'light'
        get().setMode(newMode)
      },
    }),
    {
      name: 'theme-storage',
      partialize: (state) => ({ theme: { mode: state.theme.mode } }),
      onRehydrateStorage: () => (state) => {
        if (state) {
          const isDark =
            state.theme.mode === 'dark' ||
            (state.theme.mode === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)
          state.isDark = isDark
          updateThemeClasses(isDark)
        }
      },
    },
  ),
)

function updateThemeClasses(isDark: boolean) {
  const root = document.documentElement
  const body = document.body
  root.classList.remove('dark', 'light', 'arco-theme-dark', 'cherry', 'ocean', 'nature', 'fresh', 'sunset', 'lavender')
  if (isDark) {
    root.classList.add('dark')
    body.setAttribute('arco-theme', 'dark')
  } else {
    root.classList.add('light')
    body.removeAttribute('arco-theme')
  }
}

if (typeof window !== 'undefined') {
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    const state = useThemeStore.getState()
    if (state.theme.mode === 'system') {
      state.setTheme({ mode: 'system' })
    }
  })
}
