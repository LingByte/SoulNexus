import React, { useMemo, memo } from 'react'
import { OverviewConfig } from '@/types/overview'
import WidgetRenderer from './WidgetRenderer'
import { motion } from 'framer-motion'

interface DynamicOverviewProps {
  config: OverviewConfig
  data?: Record<string, any>
}

const DynamicOverview: React.FC<DynamicOverviewProps> = ({ config, data = {} }) => {
  const { layout, widgets, theme, header, footer } = config

  // 使用 useMemo 缓存计算结果
  const backgroundStyle = useMemo(() => {
    if (!theme.backgroundColor) return undefined
    const bg = theme.backgroundColor
    if (typeof bg === 'string') {
      if (bg.includes('gradient') || bg.includes('linear-gradient') || bg.includes('radial-gradient')) {
        return bg
      }
      return bg
    }
    return undefined
  }, [theme.backgroundColor])

  const pageStyle: React.CSSProperties = useMemo(() => ({
    ...(backgroundStyle?.includes('gradient') 
      ? { background: backgroundStyle }
      : { backgroundColor: backgroundStyle as string | undefined }
    ),
    color: theme.textColor || '#1f2937',
    fontFamily: theme.fontFamily || 'system-ui, -apple-system, sans-serif',
    fontSize: theme.fontSize || '16px',
    padding: layout.padding 
      ? `${layout.padding.top || 24}px ${layout.padding.right || 24}px ${layout.padding.bottom || 24}px ${layout.padding.left || 24}px`
      : undefined
  }), [backgroundStyle, theme.textColor, theme.fontFamily, theme.fontSize, layout.padding])

  const containerStyle: React.CSSProperties = useMemo(() => ({
    maxWidth: layout.maxWidth ? `${layout.maxWidth}px` : '100%',
    margin: '0 auto'
  }), [layout.maxWidth])

  // 应用主题类名
  const themeClass = `theme-${theme.style || 'modern'}`
  
  // 预计算可见的 widgets 和排序
  const visibleWidgets = useMemo(() => {
    return widgets
      .filter(w => w.visible)
      .sort((a, b) => {
        if (a.position.y !== b.position.y) {
          return a.position.y - b.position.y
        }
        return a.position.x - b.position.x
      })
  }, [widgets])
  
  return (
    <div 
      className={`w-full min-h-screen ${themeClass}`}
      style={pageStyle}
    >
      {/* Header */}
      {header?.enabled && (header.title || header.subtitle) && (
        <div 
          className="w-full flex items-center justify-center"
          style={{
            height: `${header.height || 200}px`,
            background: header.background || `linear-gradient(135deg, ${theme.primaryColor} 0%, ${theme.secondaryColor || theme.primaryColor} 100%)`
          }}
        >
          <div className="text-center text-white">
            {header.title && (
              <h1 className="text-4xl font-bold mb-2">{header.title}</h1>
            )}
            {header.subtitle && (
              <p className="text-xl opacity-90">{header.subtitle}</p>
            )}
          </div>
        </div>
      )}

      {/* Main Content */}
      <div style={containerStyle} className="p-4">
        {visibleWidgets.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 min-h-[400px]">
            <div className="text-center max-w-md">
              <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-muted flex items-center justify-center">
                <svg className="w-8 h-8 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z" />
                </svg>
              </div>
              <h3 className="text-lg font-semibold mb-2">暂无Widget</h3>
              <p className="text-sm text-muted-foreground mb-4">
                当前概览页面还没有配置任何Widget组件
              </p>
              <p className="text-xs text-muted-foreground">
                点击右上角的"编辑页面"按钮开始添加Widget
              </p>
            </div>
          </div>
        ) : (
          <div 
            className="grid"
            style={{
              gridTemplateColumns: `repeat(${layout.columns || 12}, minmax(0, 1fr))`,
              gap: `${layout.gap || 16}px`
            }}
          >
            {visibleWidgets.map((widget, index) => (
              <motion.div
                key={widget.id}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ 
                  delay: Math.min(index * 0.05, 0.5), // 减少延迟，最多0.5秒
                  duration: 0.2 // 减少动画时长
                }}
                style={{
                  gridColumn: `span ${widget.position.w}`,
                  gridRow: `span ${widget.position.h}`,
                  minHeight: `${widget.position.h * 60}px`
                }}
              >
                <WidgetRenderer 
                  config={widget} 
                  data={data[widget.id] || data}
                />
              </motion.div>
            ))}
          </div>
        )}
      </div>

      {/* Footer */}
      {footer?.enabled && footer?.content && (
        <div className="mt-8 py-6 text-center text-muted-foreground border-t">
          <div dangerouslySetInnerHTML={{ __html: footer.content }} />
        </div>
      )}
    </div>
  )
}

// 使用 memo 优化，避免不必要的重渲染
export default memo(DynamicOverview, (prevProps, nextProps) => {
  // 如果 config 和 data 的引用相同，则不重新渲染
  return (
    prevProps.config === nextProps.config &&
    prevProps.data === nextProps.data
  )
})

