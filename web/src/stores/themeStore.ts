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

        updateThemeClasses(theme.mode, isDark)
      },

      setMode: (mode: ThemeMode) => {
        const { theme } = get()
        const newTheme = { ...theme, mode }
        get().setTheme(newTheme)
      },

      // 从 system 切换时也必须按「当前实际明暗」翻转，否则 mode=system 时一点永远是 light
      toggleMode: () => {
        const { theme } = get()
        const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
        const currentlyDark = theme.mode === 'dark' || (theme.mode === 'system' && prefersDark)
        get().setMode(currentlyDark ? 'light' : 'dark')
      },
    }),
    {
      name: 'theme-storage',
      partialize: (state) => ({ theme: state.theme }),
      onRehydrateStorage: () => (state) => {
        if (state) {
          const isDark =
            state.theme.mode === 'dark' ||
            (state.theme.mode === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)
          state.isDark = isDark

          updateThemeClasses(state.theme.mode, isDark)
        }
      },
    },
  ),
)

/** 在 React 首帧前调用：从 localStorage 同步 `html` 的 class，避免先亮后暗的闪烁 */
export function applyStoredThemeBeforeReact(): void {
  if (typeof window === 'undefined') return
  try {
    const raw = localStorage.getItem('theme-storage')
    if (!raw) return
    const parsed = JSON.parse(raw) as { state?: { theme?: Theme & { color?: string } }; theme?: Theme }
    const theme = parsed.state?.theme ?? parsed.theme
    if (!theme) return
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
    const isDark = theme.mode === 'dark' || (theme.mode === 'system' && prefersDark)
    updateThemeClasses(theme.mode, isDark)
  } catch {
    /* ignore corrupt storage */
  }
}

function updateThemeClasses(_mode: ThemeMode, isDark: boolean) {
  const root = document.documentElement

  root.classList.remove('dark', 'light', 'cherry', 'ocean', 'nature', 'fresh', 'sunset', 'lavender')

  if (isDark) {
    root.classList.add('dark')
  } else {
    root.classList.add('light')
  }
}
