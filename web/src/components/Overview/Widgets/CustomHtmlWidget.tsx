import React from 'react'
import Card, { CardContent, CardHeader, CardTitle } from '@/components/UI/Card'
import { WidgetConfig } from '@/types/overview'

interface CustomHtmlWidgetProps {
  config: WidgetConfig
}

const CustomHtmlWidget: React.FC<CustomHtmlWidgetProps> = ({ config }) => {
  const { title, props, style } = config
  const htmlContent = props?.html || ''

  return (
    <Card className="h-full" style={style}>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent className="h-[calc(100%-60px)] overflow-auto">
        {htmlContent ? (
          <div 
            dangerouslySetInnerHTML={{ __html: htmlContent }}
            className="h-full"
          />
        ) : (
          <div className="flex items-center justify-center h-full text-muted-foreground">
            <div className="text-center">
              <p className="text-sm">暂无HTML内容</p>
              <p className="text-xs opacity-50 mt-1">请在编辑器中添加HTML代码</p>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export default CustomHtmlWidget
