import React from 'react'
import { WidgetConfig } from '@/types/overview'
import Card, { CardContent, CardHeader, CardTitle } from '@/components/UI/Card'
import { ExternalLink } from 'lucide-react'

interface IframeWidgetProps {
  config: WidgetConfig
  data?: {
    url?: string
  }
}

const IframeWidget: React.FC<IframeWidgetProps> = ({ config, data }) => {
  const { title, props, style } = config
  const iframeUrl = data?.url || props?.iframeUrl || props?.url
  const allowFullscreen = props?.allowFullscreen !== false

  if (!iframeUrl) {
    return (
      <Card className="h-full" style={style}>
        <CardHeader>
          <CardTitle>{title}</CardTitle>
        </CardHeader>
        <CardContent className="h-[calc(100%-60px)] flex items-center justify-center">
          <div className="text-center text-muted-foreground">
            <ExternalLink className="w-12 h-12 mx-auto mb-2 opacity-50" />
            <p className="text-sm">未配置嵌入URL</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="h-full flex flex-col rounded-lg border bg-card text-card-foreground shadow-sm" style={style}>
      <div className="flex items-center justify-between px-6 py-4 border-b flex-shrink-0" style={{ minHeight: '60px' }}>
        <h3 className="text-lg font-semibold">{title}</h3>
        <a
          href={iframeUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="text-muted-foreground hover:text-foreground transition-colors"
        >
          <ExternalLink className="w-4 h-4" />
        </a>
      </div>
      <div className="flex-1 p-0 overflow-hidden min-h-0" style={{ flex: '1 1 auto' }}>
        <iframe
          src={iframeUrl}
          className="w-full h-full border-0"
          style={{ minHeight: '100%', display: 'block' }}
          allowFullScreen={allowFullscreen}
          sandbox="allow-same-origin allow-scripts allow-popups allow-forms"
        />
      </div>
    </div>
  )
}

export default IframeWidget

