import React from 'react'
import { CardContent, CardHeader, CardTitle } from '@/components/UI/Card'
import { Activity } from 'lucide-react'
import { WidgetConfig } from '@/types/overview'

interface ActivityFeedWidgetProps {
  config: WidgetConfig
  data?: Array<{
    type: string
    description: string
    time: string
    user?: string
  }>
}

const ActivityFeedWidget: React.FC<ActivityFeedWidgetProps> = ({ config, data = [] }) => {
  const { title, style, props } = config

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
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Activity className="w-5 h-5" />
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {data.length > 0 ? (
          <div className="space-y-4 max-h-[400px] overflow-y-auto">
            {data.map((activity, index) => (
              <div key={index} className="flex items-start gap-3 pb-4 border-b last:border-0">
                <div className="w-2 h-2 rounded-full bg-primary mt-2 flex-shrink-0"></div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium">{activity.description}</p>
                  <div className="flex items-center gap-2 mt-1">
                    {activity.user && (
                      <span className="text-xs text-muted-foreground">{activity.user}</span>
                    )}
                    <span className="text-xs text-muted-foreground">{activity.time}</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="flex items-center justify-center h-64 text-muted-foreground">
            <div className="text-center">
              <Activity className="w-12 h-12 mx-auto mb-2 opacity-50" />
              <p className="text-sm">暂无活动记录</p>
            </div>
          </div>
        )}
      </CardContent>
    </div>
  )
}

export default ActivityFeedWidget

