import React from 'react'
import { Palette, Sparkles } from 'lucide-react'
import { ThemeStyle, themePresets, PageTheme } from '@/types/overview'
import Card, { CardContent } from '@/components/UI/Card'

interface ThemeSelectorProps {
  currentTheme: PageTheme
  onThemeChange: (theme: Partial<PageTheme>) => void
}

const ThemeSelector: React.FC<ThemeSelectorProps> = ({ currentTheme, onThemeChange }) => {
  const themes: { style: ThemeStyle; name: string; preview: string }[] = [
    { style: 'modern', name: '现代风格', preview: 'bg-gradient-to-br from-indigo-500 to-purple-600' },
    { style: 'minimal', name: '极简风格', preview: 'bg-white border-2 border-black' },
    { style: 'corporate', name: '企业风格', preview: 'bg-gradient-to-br from-blue-600 to-blue-800' },
    { style: 'creative', name: '创意风格', preview: 'bg-gradient-to-br from-pink-400 to-yellow-300' },
    { style: 'dark', name: '深色风格', preview: 'bg-gradient-to-br from-gray-800 to-gray-900' },
    { style: 'gradient', name: '渐变风格', preview: 'bg-gradient-to-br from-purple-500 via-pink-500 to-red-500' },
    { style: 'glassmorphism', name: '玻璃态', preview: 'bg-white/20 backdrop-blur-lg border border-white/30' },
    { style: 'neomorphism', name: '新拟态', preview: 'bg-gray-200 shadow-[inset_5px_5px_10px_#bebebe,inset_-5px_-5px_10px_#ffffff]' },
  ]

  const handleThemeSelect = (style: ThemeStyle) => {
    const preset = themePresets[style]
    onThemeChange({
      ...preset,
      style
    })
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Palette className="w-5 h-5" />
        <h3 className="text-sm font-semibold">选择主题风格</h3>
      </div>
      <div className="grid grid-cols-4 gap-3">
        {themes.map(theme => (
          <button
            key={theme.style}
            onClick={() => handleThemeSelect(theme.style)}
            className={`relative p-4 rounded-lg border-2 transition-all ${
              currentTheme.style === theme.style
                ? 'border-primary ring-2 ring-primary ring-offset-2'
                : 'border-border hover:border-primary/50'
            }`}
          >
            <div className={`w-full h-16 rounded mb-2 ${theme.preview}`} />
            <div className="text-xs font-medium text-center">{theme.name}</div>
            {currentTheme.style === theme.style && (
              <div className="absolute top-1 right-1">
                <div className="w-4 h-4 bg-primary rounded-full flex items-center justify-center">
                  <div className="w-2 h-2 bg-white rounded-full" />
                </div>
              </div>
            )}
          </button>
        ))}
      </div>
    </div>
  )
}

export default ThemeSelector

