import React, { useState, Suspense, lazy } from 'react'
import { WidgetConfig } from '@/types/overview'
import Card, { CardContent, CardHeader, CardTitle } from '@/components/UI/Card'
import { Edit, Eye, Maximize2, Minimize2 } from 'lucide-react'
import Button from '@/components/UI/Button'
import MarkdownPreview from '@uiw/react-markdown-preview'

// 懒加载Monaco Editor
const MonacoEditor = lazy(() => import('@monaco-editor/react'))

interface MarkdownWidgetProps {
  config: WidgetConfig
  data?: {
    content?: string
  }
}

const MarkdownWidget: React.FC<MarkdownWidgetProps> = ({ config, data }) => {
  const { title, props, style } = config
  const [isEditing, setIsEditing] = useState(false)
  const [isFullscreen, setIsFullscreen] = useState(false)
  const [content, setContent] = useState(
    data?.content || props?.content || props?.markdown || '# 标题\n\n这里是内容...'
  )

  // 更新内容时同步到props（如果需要保存）
  React.useEffect(() => {
    if (props && typeof props === 'object') {
      (props as any).content = content
      ;(props as any).markdown = content
    }
  }, [content, props])

  if (isFullscreen) {
    return (
      <div className="fixed inset-0 z-50 bg-background flex flex-col">
        <div className="flex items-center justify-between p-4 border-b flex-shrink-0">
          <h3 className="text-sm font-semibold">编辑 Markdown</h3>
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setIsEditing(!isEditing)}
              leftIcon={isEditing ? <Eye className="w-4 h-4" /> : <Edit className="w-4 h-4" />}
            >
              {isEditing ? '预览' : '编辑'}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setIsFullscreen(false)}
              leftIcon={<Minimize2 className="w-4 h-4" />}
            >
              退出全屏
            </Button>
          </div>
        </div>
        <div className="flex-1 overflow-hidden">
          {isEditing ? (
            <Suspense fallback={
              <div className="h-full flex items-center justify-center">
                <div className="text-center">
                  <div className="animate-spin rounded-full h-8 w-8 border-4 border-muted border-t-primary mx-auto mb-3"></div>
                  <p className="text-sm text-muted-foreground">加载代码编辑器...</p>
                </div>
              </div>
            }>
              <MonacoEditor
                height="100%"
                language="markdown"
                value={content}
                onChange={(val) => setContent(val || '')}
                theme="vs-dark"
                options={{
                  minimap: { enabled: true },
                  scrollBeyondLastLine: false,
                  fontSize: 14,
                  lineNumbers: 'on',
                  wordWrap: 'on',
                  automaticLayout: true,
                  tabSize: 2,
                }}
              />
            </Suspense>
          ) : (
            <div className="h-full overflow-y-auto p-4">
              <MarkdownPreview 
                source={content}
                wrapperElement={{
                  'data-color-mode': 'light'
                }}
              />
            </div>
          )}
        </div>
      </div>
    )
  }

  return (
    <Card className="h-full flex flex-col" style={style}>
      <CardHeader className="flex items-center justify-between flex-shrink-0">
        <CardTitle>{title}</CardTitle>
        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setIsEditing(!isEditing)}
            leftIcon={isEditing ? <Eye className="w-4 h-4" /> : <Edit className="w-4 h-4" />}
          >
            {isEditing ? '预览' : '编辑'}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setIsFullscreen(true)}
            leftIcon={<Maximize2 className="w-4 h-4" />}
          >
            全屏
          </Button>
        </div>
      </CardHeader>
      <CardContent className="flex-1 overflow-hidden min-h-0">
        {isEditing ? (
          <div className="h-full border rounded">
            <Suspense fallback={
              <div className="h-full flex items-center justify-center bg-muted">
                <div className="text-center">
                  <div className="animate-spin rounded-full h-6 w-6 border-2 border-muted-foreground border-t-primary mx-auto mb-2"></div>
                  <p className="text-xs text-muted-foreground">加载中...</p>
                </div>
              </div>
            }>
              <MonacoEditor
                height="100%"
                language="markdown"
                value={content}
                onChange={(val) => setContent(val || '')}
                theme="vs-dark"
                options={{
                  minimap: { enabled: false },
                  scrollBeyondLastLine: false,
                  fontSize: 12,
                  lineNumbers: 'on',
                  wordWrap: 'on',
                  automaticLayout: true,
                  tabSize: 2,
                }}
              />
            </Suspense>
          </div>
        ) : (
          <div className="h-full overflow-y-auto">
            <MarkdownPreview 
              source={content}
              wrapperElement={{
                'data-color-mode': 'light'
              }}
            />
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export default MarkdownWidget
