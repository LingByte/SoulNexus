import React from 'react'
import { WidgetConfig } from '@/types/overview'

interface Feature {
  title: string
  description: string
  icon?: string
}

interface FeatureGridWidgetProps {
  config: WidgetConfig
  data?: Feature[]
}

const FeatureGridWidget: React.FC<FeatureGridWidgetProps> = ({ config, data }) => {
  const { title, style, props } = config
  
  // 确保 features 是数组
  let features: Feature[] = []
  if (Array.isArray(data) && data.length > 0) {
    features = data
  } else if (Array.isArray(props?.features)) {
    features = props.features
  } else if (data && typeof data === 'object' && !Array.isArray(data)) {
    const dataObj = data as any
    features = (Array.isArray(dataObj.features) ? dataObj.features : []) || 
               (Array.isArray(dataObj.items) ? dataObj.items : [])
  } else {
    features = [
      { title: '功能1', description: '描述1' },
      { title: '功能2', description: '描述2' },
      { title: '功能3', description: '描述3' },
    ]
  }

  // 确保是数组
  if (!Array.isArray(features)) {
    features = []
  }

  // 处理 padding（支持字符串或对象格式）
  const getPadding = () => {
    if (!style?.padding) return undefined
    if (typeof style.padding === 'object') {
      return `${style.padding.top || 0}px ${style.padding.right || 0}px ${style.padding.bottom || 0}px ${style.padding.left || 0}px`
    }
    return style.padding
  }

  const cardStyle: React.CSSProperties = {
    ...style,
    padding: getPadding(),
  }

  return (
    <div 
      className="h-full rounded-lg border bg-card text-card-foreground shadow-sm"
      style={cardStyle}
    >
      <h3 className="text-lg font-semibold mb-4">{title}</h3>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {features.map((feature: Feature, index: number) => (
          <div key={index} className="p-4 border rounded-lg hover:shadow-md transition-shadow">
            {feature.icon && (
              <div className="text-2xl mb-2">{feature.icon}</div>
            )}
            <h4 className="font-medium mb-1">{feature.title}</h4>
            <p className="text-sm text-muted-foreground">{feature.description}</p>
          </div>
        ))}
      </div>
    </div>
  )
}

export default FeatureGridWidget

