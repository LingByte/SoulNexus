import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type AppLocale = 'zh-CN' | 'zh-TW' | 'en-US' | 'ja-JP'

interface LocaleState {
  locale: AppLocale
  setLocale: (locale: AppLocale) => void
  toggleLocale: () => void
}

export const useLocaleStore = create<LocaleState>()(
  persist(
    (set, get) => ({
      locale: 'zh-CN',
      setLocale: (locale) => set({ locale }),
      toggleLocale: () => {
        const current = get().locale
        const next: AppLocale = current === 'zh-CN' ? 'zh-TW' : current === 'zh-TW' ? 'en-US' : current === 'en-US' ? 'ja-JP' : 'zh-CN'
        set({ locale: next })
      },
    }),
    { name: 'locale-storage' }
  )
)
