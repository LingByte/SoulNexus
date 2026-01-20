import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { translations, type Language } from '@/locales'

export type { Language }

interface I18nState {
  language: Language
  setLanguage: (lang: Language) => void
  t: (key: string, params?: Record<string, string | number>) => string
}

// 翻译资源已迁移到 @/locales/translations.ts

export const useI18nStore = create<I18nState>()(
  persist(
    (set, get) => ({
      language: 'en',
      
      setLanguage: (lang: Language) => {
        set({ language: lang })
      },
      
      t: (key: string, params?: Record<string, string | number>) => {
        const lang = get().language
        let text = translations[lang]?.[key] || key
        
        // 如果有参数，替换占位符
        if (params) {
          Object.keys(params).forEach((paramKey) => {
            const value = params[paramKey]
            // 替换 {key} 格式的占位符
            text = text.replace(new RegExp(`\\{${paramKey}\\}`, 'g'), String(value))
          })
        }
        
        return text
      },
    }),
    {
      name: 'i18n-storage',
    }
  )
)
