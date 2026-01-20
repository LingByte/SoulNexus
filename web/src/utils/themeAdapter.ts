// 主题适配器 - 统一管理主题色和样式
export const getThemeColors = (theme: string = 'default') => {
  const themes = {
    default: {
      primary: 'blue',
      primaryHover: 'blue-700',
      primaryForeground: 'blue-50',
      secondary: 'gray',
      secondaryHover: 'gray-700',
      accent: 'blue-100',
      muted: 'gray-100',
      mutedForeground: 'gray-600',
      border: 'gray-200',
      background: 'white',
      foreground: 'gray-900'
    },
    dark: {
      primary: 'blue-400',
      primaryHover: 'blue-300',
      primaryForeground: 'blue-950',
      secondary: 'gray-600',
      secondaryHover: 'gray-500',
      accent: 'blue-900',
      muted: 'gray-800',
      mutedForeground: 'gray-400',
      border: 'gray-700',
      background: 'gray-900',
      foreground: 'gray-100'
    },
    nature: {
      primary: 'green-600',
      primaryHover: 'green-700',
      primaryForeground: 'green-50',
      secondary: 'emerald-600',
      secondaryHover: 'emerald-700',
      accent: 'green-100',
      muted: 'emerald-50',
      mutedForeground: 'green-700',
      border: 'green-200',
      background: 'white',
      foreground: 'green-900'
    },
    ocean: {
      primary: 'cyan-600',
      primaryHover: 'cyan-700',
      primaryForeground: 'cyan-50',
      secondary: 'blue-600',
      secondaryHover: 'blue-700',
      accent: 'cyan-100',
      muted: 'cyan-50',
      mutedForeground: 'cyan-700',
      border: 'cyan-200',
      background: 'white',
      foreground: 'cyan-900'
    },
    sunset: {
      primary: 'orange-600',
      primaryHover: 'orange-700',
      primaryForeground: 'orange-50',
      secondary: 'red-600',
      secondaryHover: 'red-700',
      accent: 'orange-100',
      muted: 'orange-50',
      mutedForeground: 'orange-700',
      border: 'orange-200',
      background: 'white',
      foreground: 'orange-900'
    },
    purple: {
      primary: 'purple-600',
      primaryHover: 'purple-700',
      primaryForeground: 'purple-50',
      secondary: 'violet-600',
      secondaryHover: 'violet-700',
      accent: 'purple-100',
      muted: 'purple-50',
      mutedForeground: 'purple-700',
      border: 'purple-200',
      background: 'white',
      foreground: 'purple-900'
    },
    rose: {
      primary: 'rose-600',
      primaryHover: 'rose-700',
      primaryForeground: 'rose-50',
      secondary: 'pink-600',
      secondaryHover: 'pink-700',
      accent: 'rose-100',
      muted: 'pink-50',
      mutedForeground: 'rose-700',
      border: 'rose-200',
      background: 'white',
      foreground: 'rose-900'
    },
    fresh: {
      primary: 'emerald-600',
      primaryHover: 'emerald-700',
      primaryForeground: 'emerald-50',
      secondary: 'teal-600',
      secondaryHover: 'teal-700',
      accent: 'emerald-100',
      muted: 'teal-50',
      mutedForeground: 'emerald-700',
      border: 'emerald-200',
      background: 'white',
      foreground: 'emerald-900'
    }
  }

  return themes[theme as keyof typeof themes] || themes.default
}

// 获取主题相关的CSS类名
export const getThemeClasses = (theme: string = 'default', variant: 'primary' | 'secondary' | 'accent' | 'muted' = 'primary') => {
  const colors = getThemeColors(theme)
  
  const variants = {
    primary: {
      bg: `bg-${colors.primary}`,
      hover: `hover:bg-${colors.primaryHover}`,
      text: `text-${colors.primaryForeground}`,
      border: `border-${colors.primary}`,
      ring: `ring-${colors.primary}`,
      shadow: `shadow-${colors.primary}/20`
    },
    secondary: {
      bg: `bg-${colors.secondary}`,
      hover: `hover:bg-${colors.secondaryHover}`,
      text: `text-${colors.primaryForeground}`,
      border: `border-${colors.secondary}`,
      ring: `ring-${colors.secondary}`,
      shadow: `shadow-${colors.secondary}/20`
    },
    accent: {
      bg: `bg-${colors.accent}`,
      hover: `hover:bg-${colors.primary}`,
      text: `text-${colors.primary}`,
      border: `border-${colors.accent}`,
      ring: `ring-${colors.primary}`,
      shadow: `shadow-${colors.primary}/10`
    },
    muted: {
      bg: `bg-${colors.muted}`,
      hover: `hover:bg-${colors.secondary}`,
      text: `text-${colors.mutedForeground}`,
      border: `border-${colors.border}`,
      ring: `ring-${colors.primary}`,
      shadow: `shadow-${colors.primary}/5`
    }
  }

  return variants[variant]
}

// 获取当前主题
export const getCurrentTheme = (): string => {
  if (typeof window === 'undefined') return 'default'
  
  const html = document.documentElement
  const theme = html.getAttribute('data-theme') || 
                html.classList.contains('dark') ? 'dark' : 'default'
  
  return theme
}

// 获取当前主题模式
export const getCurrentThemeMode = (): 'light' | 'dark' => {
  if (typeof window === 'undefined') return 'light'
  
  const html = document.documentElement
  return html.classList.contains('dark') ? 'dark' : 'light'
}

// 设置主题
export const setTheme = (theme: string) => {
  if (typeof window === 'undefined') return
  
  const html = document.documentElement
  html.setAttribute('data-theme', theme)
  
  // 更新CSS变量
  const colors = getThemeColors(theme)
  Object.entries(colors).forEach(([key, value]) => {
    html.style.setProperty(`--${key}`, value)
  })
}

// 设置自定义主题色
export const setCustomThemeColor = (color: string) => {
  if (typeof window === 'undefined') return
  
  const html = document.documentElement
  const isDark = html.classList.contains('dark')
  
  // 计算颜色变体
  const primary = color
  const primaryHover = adjustBrightness(color, -20)
  const primaryForeground = isDark ? '#ffffff' : '#ffffff'
  
  // 设置CSS变量
  html.style.setProperty('--color-primary', primary)
  html.style.setProperty('--color-primary-hover', primaryHover)
  html.style.setProperty('--color-primary-foreground', primaryForeground)
}

// 调整颜色亮度
const adjustBrightness = (color: string, amount: number): string => {
  const num = parseInt(color.replace('#', ''), 16)
  const r = Math.max(0, Math.min(255, (num >> 16) + amount))
  const g = Math.max(0, Math.min(255, ((num >> 8) & 0x00FF) + amount))
  const b = Math.max(0, Math.min(255, (num & 0x0000FF) + amount))
  return `#${((r << 16) | (g << 8) | b).toString(16).padStart(6, '0')}`
}

// 主题感知的样式生成器
export const createThemeAwareStyles = (theme: string = 'default') => {
  const colors = getThemeColors(theme)
  
  return {
    // 按钮样式
    button: {
      primary: `bg-${colors.primary} hover:bg-${colors.primaryHover} text-${colors.primaryForeground} border-${colors.primary} ring-${colors.primary}`,
      secondary: `bg-${colors.secondary} hover:bg-${colors.secondaryHover} text-${colors.primaryForeground} border-${colors.secondary} ring-${colors.secondary}`,
      outline: `border-${colors.border} text-${colors.foreground} hover:bg-${colors.muted} hover:border-${colors.primary}`,
      ghost: `text-${colors.foreground} hover:bg-${colors.muted} hover:text-${colors.primary}`
    },
    
    // 卡片样式
    card: {
      default: `bg-${colors.background} border-${colors.border} text-${colors.foreground}`,
      elevated: `bg-${colors.background} border-${colors.border} shadow-lg hover:shadow-xl`,
      filled: `bg-${colors.muted} border-${colors.border} text-${colors.foreground}`,
      glass: `bg-${colors.background}/80 backdrop-blur-sm border-${colors.border}/20`
    },
    
    // 输入框样式
    input: {
      default: `border-${colors.border} bg-${colors.background} text-${colors.foreground} focus:border-${colors.primary} focus:ring-${colors.primary}`,
      filled: `border-${colors.border} bg-${colors.muted} text-${colors.foreground} focus:border-${colors.primary} focus:ring-${colors.primary}`
    },
    
    // 徽章样式
    badge: {
      primary: `bg-${colors.primary} text-${colors.primaryForeground}`,
      secondary: `bg-${colors.secondary} text-${colors.primaryForeground}`,
      accent: `bg-${colors.accent} text-${colors.primary}`,
      muted: `bg-${colors.muted} text-${colors.mutedForeground}`
    }
  }
}

// 响应式主题切换Hook
export const useTheme = () => {
  const [theme, setThemeState] = useState(getCurrentTheme())
  
  useEffect(() => {
    const handleThemeChange = () => {
      setThemeState(getCurrentTheme())
    }
    
    // 监听主题变化
    const observer = new MutationObserver(handleThemeChange)
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['data-theme', 'class']
    })
    
    return () => observer.disconnect()
  }, [])
  
  const changeTheme = (newTheme: string) => {
    setTheme(newTheme)
    setThemeState(newTheme)
  }
  
  return {
    theme,
    changeTheme,
    colors: getThemeColors(theme),
    classes: getThemeClasses(theme)
  }
}

// 导入React hooks
import { useState, useEffect } from 'react'