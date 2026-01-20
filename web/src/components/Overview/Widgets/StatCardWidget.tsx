import React from 'react'
import Card, { CardContent } from '@/components/UI/Card'
import { LucideIcon } from 'lucide-react'
import { WidgetConfig } from '@/types/overview'

interface StatCardWidgetProps {
  config: WidgetConfig
  data?: {
    value: number | string
    label?: string
    change?: string
    trend?: 'up' | 'down' | 'neutral'
  }
}

const StatCardWidget: React.FC<StatCardWidgetProps> = ({ config, data }) => {
  const { title, props, style } = config
  const iconName = props?.icon as string
  const iconColor = props?.iconColor || 'text-blue-500'
  const bgColor = props?.bgColor || 'bg-blue-50 dark:bg-blue-900/20'
  
  // 动态导入图标
  const IconComponent = React.useMemo(() => {
    if (!iconName) return null
    try {
      // 这里需要根据实际使用的图标库来动态导入
      // 暂时返回null，实际使用时需要传入Icon组件
      return null
    } catch {
      return null
    }
  }, [iconName])

  const value = data?.value ?? props?.defaultValue ?? 0
  const label = data?.label ?? props?.label ?? title

  // 处理 padding（支持字符串或对象格式）
  const getPadding = () => {
    if (!style?.padding) return undefined
    if (typeof style.padding === 'object') {
      return `${style.padding.top || 0}px ${style.padding.right || 0}px ${style.padding.bottom || 0}px ${style.padding.left || 0}px`
    }
    return style.padding
  }

  // 应用样式
  const cardStyle: React.CSSProperties = {
    ...style,
    backgroundColor: style?.backgroundColor || undefined,
    color: style?.textColor || undefined,
    borderRadius: style?.borderRadius || undefined,
    padding: getPadding(),
  }

  return (
    <div 
      className="h-full rounded-lg border bg-card text-card-foreground shadow-sm"
      style={cardStyle}
    >
      <CardContent className="p-6 h-full flex items-center justify-between">
        <div className="flex-1">
          <p className="text-sm font-medium text-muted-foreground mb-1">
            {label}
          </p>
          <p className="text-2xl font-bold" style={{ color: style?.textColor || undefined }}>
            {typeof value === 'number' ? value.toLocaleString() : value}
          </p>
          {data?.change && (
            <p className={`text-xs mt-1 ${
              data.trend === 'up' ? 'text-green-500' : 
              data.trend === 'down' ? 'text-red-500' : 
              'text-muted-foreground'
            }`}>
              {data.change}
            </p>
          )}
        </div>
        {IconComponent && (
          <div className={`${bgColor} p-3 rounded-lg`}>
            <IconComponent className={`w-6 h-6 ${iconColor}`} />
          </div>
        )}
      </CardContent>
    </div>
  )
}

export default StatCardWidget

