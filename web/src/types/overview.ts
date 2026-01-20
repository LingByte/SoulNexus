// 概览页面配置类型定义

// Widget类型
export type WidgetType = 
  | 'stat-card'      // 统计卡片
  | 'chart-line'     // 折线图
  | 'chart-bar'      // 柱状图
  | 'chart-pie'      // 饼图
  | 'chart-area'     // 面积图
  | 'chart-radar'    // 雷达图
  | 'activity-feed'  // 活动流
  | 'table'          // 表格
  | 'image'          // 图片展示
  | 'video'          // 视频嵌入
  | 'iframe'         // 嵌入页面
  | 'markdown'       // Markdown内容
  | 'custom-html'    // 自定义HTML
  | 'hero-banner'    // 英雄横幅
  | 'feature-grid'   // 功能网格
  | 'testimonial'    // 客户评价
  | 'timeline'       // 时间线
  | 'progress-ring'  // 进度环
  | 'metric-comparison' // 指标对比

// Widget尺寸
export type WidgetSize = 'small' | 'medium' | 'large' | 'full' | 'custom'

// 布局类型
export type LayoutType = 'grid' | 'masonry' | 'flex' | 'carousel' | 'tabs'

// 主题风格
export type ThemeStyle = 
  | 'modern'      // 现代风格
  | 'minimal'     // 极简风格
  | 'corporate'   // 企业风格
  | 'creative'    // 创意风格
  | 'dark'        // 深色风格
  | 'gradient'    // 渐变风格
  | 'glassmorphism' // 玻璃态风格
  | 'neomorphism'   // 新拟态风格

// Widget样式配置
export interface WidgetStyle {
  backgroundColor?: string
  backgroundImage?: string
  backgroundGradient?: string
  textColor?: string
  borderColor?: string
  borderWidth?: string
  borderRadius?: string
  padding?: string | { // 支持字符串格式（如 "16px"）或对象格式（分别控制四个方向）
    top: number
    right: number
    bottom: number
    left: number
  }
  margin?: string
  shadow?: string
  opacity?: number
  transform?: string
  animation?: string
  backdropFilter?: string
  // 高级样式
  gradient?: {
    type: 'linear' | 'radial' | 'conic'
    colors: string[]
    direction?: string
  }
  glassmorphism?: {
    blur: number
    opacity: number
    borderColor?: string
  }
}

// Widget配置
export interface WidgetConfig {
  id: string
  type: WidgetType
  title: string
  size: WidgetSize
  position: {
    x: number  // 列位置 (0-based)
    y: number  // 行位置 (0-based)
    w: number  // 宽度 (列数)
    h: number  // 高度 (行数)
  }
  props: Record<string, any>  // Widget特定属性
  style?: WidgetStyle
  visible: boolean
  animation?: {
    type: 'fade' | 'slide' | 'scale' | 'bounce' | 'none'
    duration?: number
    delay?: number
  }
}

// 页面主题配置
export interface PageTheme {
  style: ThemeStyle
  primaryColor: string
  secondaryColor?: string
  backgroundColor: string
  textColor: string
  accentColor?: string
  fontFamily?: string
  fontSize?: string
  cardStyle: 'default' | 'minimal' | 'bordered' | 'shadow' | 'glass' | 'gradient'
  borderRadius?: string
  spacing?: {
    small: number
    medium: number
    large: number
  }
  shadows?: {
    small: string
    medium: string
    large: string
  }
}

// 布局配置
export interface LayoutConfig {
  type: LayoutType
  columns: number  // 网格列数 (默认12)
  gap: number      // 间距 (px)
  padding?: {
    top: number
    right: number
    bottom: number
    left: number
  }
  previewPadding?: { // 预览区域内边距 (px)
    top: number
    right: number
    bottom: number
    left: number
  }
  maxWidth?: number // 最大宽度 (px)
  responsive?: {
    mobile: { columns: number; gap: number }
    tablet: { columns: number; gap: number }
    desktop: { columns: number; gap: number }
  }
}

// 概览页面配置
export interface OverviewConfig {
  id: string
  organizationId: number
  name: string
  description?: string
  layout: LayoutConfig
  widgets: WidgetConfig[]
  theme: PageTheme
  header?: {
    enabled: boolean
    title?: string
    subtitle?: string
    background?: string
    height?: number
  }
  footer?: {
    enabled: boolean
    content?: string
  }
  createdAt?: string
  updatedAt?: string
}

// 默认配置
export const defaultOverviewConfig: OverviewConfig = {
  id: '',
  organizationId: 0,
  name: '默认概览',
  layout: {
    type: 'grid',
    columns: 12,
    gap: 16,
    padding: { top: 24, right: 24, bottom: 24, left: 24 },
    previewPadding: { top: 16, right: 16, bottom: 16, left: 16 },
    responsive: {
      mobile: { columns: 4, gap: 12 },
      tablet: { columns: 8, gap: 16 },
      desktop: { columns: 12, gap: 16 }
    }
  },
  widgets: [],
  theme: {
    style: 'modern',
    primaryColor: '#6366f1',
    secondaryColor: '#8b5cf6',
    backgroundColor: '#ffffff',
    textColor: '#1f2937',
    accentColor: '#ec4899',
    fontFamily: 'system-ui, -apple-system, sans-serif',
    fontSize: '16px',
    cardStyle: 'default',
    borderRadius: '8px',
    spacing: {
      small: 8,
      medium: 16,
      large: 24
    },
    shadows: {
      small: '0 1px 2px rgba(0,0,0,0.05)',
      medium: '0 4px 6px rgba(0,0,0,0.1)',
      large: '0 10px 15px rgba(0,0,0,0.15)'
    }
  },
  header: {
    enabled: true,
    height: 200
  },
  footer: {
    enabled: false
  }
}

// Widget尺寸映射 (列数)
export const widgetSizeMap: Record<WidgetSize, number> = {
  small: 3,    // 1/4 宽度
  medium: 6,   // 1/2 宽度
  large: 9,    // 3/4 宽度
  full: 12     // 全宽
}

// Widget高度映射 (行数)
export const widgetHeightMap: Record<WidgetType, number> = {
  'stat-card': 2,
  'chart-line': 4,
  'chart-bar': 4,
  'chart-pie': 4,
  'chart-area': 4,
  'chart-radar': 4,
  'activity-feed': 6,
  'table': 6,
  'image': 4,
  'video': 6,
  'iframe': 6,
  'markdown': 4,
  'custom-html': 4,
  'hero-banner': 8,
  'feature-grid': 6,
  'testimonial': 4,
  'timeline': 8,
  'progress-ring': 3,
  'metric-comparison': 4
}

// 预设主题模板
export const themePresets: Record<ThemeStyle, Partial<PageTheme>> = {
  modern: {
    style: 'modern',
    primaryColor: '#6366f1',
    backgroundColor: '#ffffff',
    textColor: '#1f2937',
    cardStyle: 'shadow',
    borderRadius: '12px'
  },
  minimal: {
    style: 'minimal',
    primaryColor: '#000000',
    backgroundColor: '#ffffff',
    textColor: '#000000',
    cardStyle: 'bordered',
    borderRadius: '0px'
  },
  corporate: {
    style: 'corporate',
    primaryColor: '#1e40af',
    backgroundColor: '#f8fafc',
    textColor: '#1e293b',
    cardStyle: 'default',
    borderRadius: '4px'
  },
  creative: {
    style: 'creative',
    primaryColor: '#ec4899',
    backgroundColor: '#fef3c7',
    textColor: '#78350f',
    cardStyle: 'gradient',
    borderRadius: '20px'
  },
  dark: {
    style: 'dark',
    primaryColor: '#8b5cf6',
    backgroundColor: '#111827',
    textColor: '#f9fafb',
    cardStyle: 'shadow',
    borderRadius: '8px'
  },
  gradient: {
    style: 'gradient',
    primaryColor: '#6366f1',
    backgroundColor: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
    textColor: '#ffffff',
    cardStyle: 'glass',
    borderRadius: '16px'
  },
  glassmorphism: {
    style: 'glassmorphism',
    primaryColor: '#6366f1',
    backgroundColor: 'rgba(255,255,255,0.1)',
    textColor: '#1f2937',
    cardStyle: 'glass',
    borderRadius: '20px'
  },
  neomorphism: {
    style: 'neomorphism',
    primaryColor: '#6366f1',
    backgroundColor: '#e5e7eb',
    textColor: '#1f2937',
    cardStyle: 'default',
    borderRadius: '20px'
  }
}

