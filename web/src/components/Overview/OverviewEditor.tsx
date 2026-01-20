import React, { useState, Suspense, lazy, useCallback, useEffect, useRef } from 'react'
import { Plus, Trash2, Eye, EyeOff, GripVertical, Palette, Layout, Settings, Maximize2, Minimize2 } from 'lucide-react'
import Button from '@/components/UI/Button'
import Card, { CardContent, CardHeader, CardTitle } from '@/components/UI/Card'
import { OverviewConfig, WidgetConfig, WidgetType, WidgetSize, widgetSizeMap, widgetHeightMap } from '@/types/overview'
import DragSort from '@/components/UI/DragSort'
import ThemeSelector from './ThemeSelector'
import StyleEditor from './StyleEditor'

// 预加载Monaco Editor（优化加载性能）
let MonacoEditorCache: any = null
if (typeof window !== 'undefined') {
  // 在浏览器环境中预加载
  import('@monaco-editor/react').then(module => {
    MonacoEditorCache = module.default
  }).catch(() => {
    // 预加载失败时使用懒加载
  })
}

// Markdown代码编辑器组件
const MarkdownCodeEditor: React.FC<{ value: string; onChange: (value: string) => void }> = ({ value, onChange }) => {
  const [isFullscreen, setIsFullscreen] = useState(false)
  const [MonacoEditor, setMonacoEditor] = useState<any>(MonacoEditorCache)
  
  // 预加载Monaco Editor
  React.useEffect(() => {
    if (!MonacoEditor) {
      import('@monaco-editor/react').then(module => {
        setMonacoEditor(() => module.default)
        MonacoEditorCache = module.default
      })
    }
  }, [MonacoEditor])

  if (isFullscreen) {
    return (
      <div className="fixed inset-0 z-50 bg-background flex flex-col">
        <div className="flex items-center justify-between p-4 border-b">
          <h3 className="text-sm font-semibold">编辑 Markdown</h3>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setIsFullscreen(false)}
            leftIcon={<Minimize2 className="w-4 h-4" />}
          >
            退出全屏
          </Button>
        </div>
        <div className="flex-1 overflow-hidden">
          {MonacoEditor ? (
            <MonacoEditor
              height="100%"
              language="markdown"
              value={value}
              onChange={(val: string | undefined) => onChange(val || '')}
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
          ) : (
            <Suspense fallback={
              <div className="h-full flex items-center justify-center">
                <div className="text-center">
                  <div className="animate-spin rounded-full h-8 w-8 border-4 border-muted border-t-primary mx-auto mb-3"></div>
                  <p className="text-sm text-muted-foreground">加载代码编辑器...</p>
                </div>
              </div>
            }>
              {React.createElement(lazy(() => import('@monaco-editor/react')), {
                height: "100%",
                language: "markdown",
                value: value,
                onChange: (val: string | undefined) => onChange(val || ''),
                theme: "vs-dark",
                options: {
                  minimap: { enabled: true },
                  scrollBeyondLastLine: false,
                  fontSize: 14,
                  lineNumbers: 'on',
                  wordWrap: 'on',
                  automaticLayout: true,
                  tabSize: 2,
                }
              })}
            </Suspense>
          )}
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <span className="text-xs text-muted-foreground">Markdown编辑器</span>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setIsFullscreen(true)}
          leftIcon={<Maximize2 className="w-3 h-3" />}
        >
          全屏
        </Button>
      </div>
      <div className="border rounded overflow-hidden" style={{ height: '300px' }}>
        {MonacoEditor ? (
          <MonacoEditor
            height="100%"
            language="markdown"
            value={value}
            onChange={(val: string | undefined) => onChange(val || '')}
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
        ) : (
          <Suspense fallback={
            <div className="h-full flex items-center justify-center bg-muted">
              <div className="text-center">
                <div className="animate-spin rounded-full h-6 w-6 border-2 border-muted-foreground border-t-primary mx-auto mb-2"></div>
                <p className="text-xs text-muted-foreground">加载中...</p>
              </div>
            </div>
          }>
            {React.createElement(lazy(() => import('@monaco-editor/react')), {
              height: "100%",
              language: "markdown",
              value: value,
              onChange: (val: string | undefined) => onChange(val || ''),
              theme: "vs-dark",
              options: {
                minimap: { enabled: false },
                scrollBeyondLastLine: false,
                fontSize: 12,
                lineNumbers: 'on',
                wordWrap: 'on',
                automaticLayout: true,
                tabSize: 2,
              }
            })}
          </Suspense>
        )}
      </div>
    </div>
  )
}

// HTML代码编辑器组件
const HtmlCodeEditor: React.FC<{ value: string; onChange: (value: string) => void }> = ({ value, onChange }) => {
  const [isFullscreen, setIsFullscreen] = useState(false)
  const [MonacoEditor, setMonacoEditor] = useState<any>(MonacoEditorCache)
  
  // 预加载Monaco Editor
  React.useEffect(() => {
    if (!MonacoEditor) {
      import('@monaco-editor/react').then(module => {
        setMonacoEditor(() => module.default)
        MonacoEditorCache = module.default
      })
    }
  }, [MonacoEditor])

  if (isFullscreen) {
    return (
      <div className="fixed inset-0 z-50 bg-background flex flex-col">
        <div className="flex items-center justify-between p-4 border-b">
          <h3 className="text-sm font-semibold">编辑 HTML 代码</h3>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setIsFullscreen(false)}
            leftIcon={<Minimize2 className="w-4 h-4" />}
          >
            退出全屏
          </Button>
        </div>
        <div className="flex-1 overflow-hidden">
          {MonacoEditor ? (
            <MonacoEditor
              height="100%"
              language="html"
              value={value}
              onChange={(val: string | undefined) => onChange(val || '')}
              theme="vs-dark"
              options={{
                minimap: { enabled: true },
                scrollBeyondLastLine: false,
                fontSize: 14,
                lineNumbers: 'on',
                wordWrap: 'on',
                automaticLayout: true,
                tabSize: 2,
                formatOnPaste: true,
                formatOnType: true,
              }}
            />
          ) : (
            <Suspense fallback={
              <div className="h-full flex items-center justify-center">
                <div className="text-center">
                  <div className="animate-spin rounded-full h-8 w-8 border-4 border-muted border-t-primary mx-auto mb-3"></div>
                  <p className="text-sm text-muted-foreground">加载代码编辑器...</p>
                </div>
              </div>
            }>
              {React.createElement(lazy(() => import('@monaco-editor/react')), {
                height: "100%",
                language: "html",
                value: value,
                onChange: (val: string | undefined) => onChange(val || ''),
                theme: "vs-dark",
                options: {
                  minimap: { enabled: true },
                  scrollBeyondLastLine: false,
                  fontSize: 14,
                  lineNumbers: 'on',
                  wordWrap: 'on',
                  automaticLayout: true,
                  tabSize: 2,
                  formatOnPaste: true,
                  formatOnType: true,
                }
              })}
            </Suspense>
          )}
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <span className="text-xs text-muted-foreground">代码编辑器</span>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setIsFullscreen(true)}
          leftIcon={<Maximize2 className="w-3 h-3" />}
        >
          全屏
        </Button>
      </div>
      <div className="border rounded overflow-hidden" style={{ height: '300px' }}>
        {MonacoEditor ? (
          <MonacoEditor
            height="100%"
            language="html"
            value={value}
            onChange={(val: string | undefined) => onChange(val || '')}
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
        ) : (
          <Suspense fallback={
            <div className="h-full flex items-center justify-center bg-muted">
              <div className="text-center">
                <div className="animate-spin rounded-full h-6 w-6 border-2 border-muted-foreground border-t-primary mx-auto mb-2"></div>
                <p className="text-xs text-muted-foreground">加载中...</p>
              </div>
            </div>
          }>
            {React.createElement(lazy(() => import('@monaco-editor/react')), {
              height: "100%",
              language: "html",
              value: value,
              onChange: (val: string | undefined) => onChange(val || ''),
              theme: "vs-dark",
              options: {
                minimap: { enabled: false },
                scrollBeyondLastLine: false,
                fontSize: 12,
                lineNumbers: 'on',
                wordWrap: 'on',
                automaticLayout: true,
                tabSize: 2,
              }
            })}
          </Suspense>
        )}
      </div>
    </div>
  )
}

interface OverviewEditorProps {
  config: OverviewConfig
  onSave: (config: OverviewConfig) => void
  onCancel: () => void
}

const OverviewEditor: React.FC<OverviewEditorProps> = ({ config, onSave, onCancel }) => {
  const [editedConfig, setEditedConfig] = useState<OverviewConfig>(config)
  const [selectedWidget, setSelectedWidget] = useState<WidgetConfig | null>(null)
  const [activeTab, setActiveTab] = useState<'widgets' | 'theme' | 'layout'>('widgets')
  const saveTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  
  // 当外部config变化时更新内部状态
  useEffect(() => {
    setEditedConfig(config)
  }, [config])

  const availableWidgetTypes: { type: WidgetType; name: string; category: string }[] = [
    { type: 'stat-card', name: '统计卡片', category: '数据' },
    { type: 'hero-banner', name: '英雄横幅', category: '展示' },
    { type: 'chart-line', name: '折线图', category: '图表' },
    { type: 'chart-bar', name: '柱状图', category: '图表' },
    { type: 'chart-pie', name: '饼图', category: '图表' },
    { type: 'chart-area', name: '面积图', category: '图表' },
    { type: 'chart-radar', name: '雷达图', category: '图表' },
    { type: 'activity-feed', name: '活动流', category: '内容' },
    { type: 'table', name: '数据表格', category: '数据' },
    { type: 'feature-grid', name: '功能网格', category: '展示' },
    { type: 'testimonial', name: '客户评价', category: '展示' },
    { type: 'timeline', name: '时间线', category: '内容' },
    { type: 'progress-ring', name: '进度环', category: '数据' },
    { type: 'metric-comparison', name: '指标对比', category: '数据' },
    { type: 'image', name: '图片展示', category: '媒体' },
    { type: 'video', name: '视频嵌入', category: '媒体' },
    { type: 'iframe', name: '嵌入页面', category: '媒体' },
    { type: 'markdown', name: 'Markdown', category: '内容' },
    { type: 'custom-html', name: '自定义HTML', category: '自定义' },
  ]

  const widgetCategories = Array.from(new Set(availableWidgetTypes.map(w => w.category)))

  const addWidget = (type: WidgetType) => {
    const newWidget: WidgetConfig = {
      id: `widget-${Date.now()}`,
      type,
      title: `新建${availableWidgetTypes.find(w => w.type === type)?.name || type}`,
      size: 'medium',
      position: {
        x: 0,
        y: 0,
        w: widgetSizeMap.medium,
        h: widgetHeightMap[type]
      },
      props: {},
      visible: true
    }
    setEditedConfig({
      ...editedConfig,
      widgets: [...editedConfig.widgets, newWidget]
    })
    setSelectedWidget(newWidget)
  }

  const removeWidget = (widgetId: string) => {
    setEditedConfig({
      ...editedConfig,
      widgets: editedConfig.widgets.filter(w => w.id !== widgetId)
    })
    if (selectedWidget?.id === widgetId) {
      setSelectedWidget(null)
    }
  }

  const updateWidget = (widgetId: string, updates: Partial<WidgetConfig>) => {
    setEditedConfig({
      ...editedConfig,
      widgets: editedConfig.widgets.map(w => 
        w.id === widgetId ? { ...w, ...updates } : w
      )
    })
    if (selectedWidget?.id === widgetId) {
      setSelectedWidget({ ...selectedWidget, ...updates })
    }
  }

  const handleSort = (sortedWidgets: WidgetConfig[]) => {
    setEditedConfig({
      ...editedConfig,
      widgets: sortedWidgets
    })
  }

  const handleSave = useCallback(() => {
    if (saveTimeoutRef.current) {
      clearTimeout(saveTimeoutRef.current)
    }
    onSave(editedConfig)
  }, [editedConfig, onSave])

  return (
    <div className="space-y-4">
      {/* 工具栏 */}
      <div className="flex items-center justify-between p-4 bg-muted rounded-lg">
        <div className="flex items-center gap-2">
          <h2 className="text-lg font-semibold">编辑概览页面</h2>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="ghost" onClick={onCancel}>
            取消
          </Button>
          <Button variant="primary" onClick={handleSave}>
            保存配置
          </Button>
        </div>
      </div>

      {/* 配置基本信息 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">基本信息</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <label className="text-sm font-medium mb-1 block">配置名称</label>
            <input
              type="text"
              value={editedConfig.name}
              onChange={(e) => setEditedConfig({ ...editedConfig, name: e.target.value })}
              className="w-full px-3 py-2 text-sm border rounded-lg"
              placeholder="输入配置名称"
            />
          </div>
          <div>
            <label className="text-sm font-medium mb-1 block">描述（可选）</label>
            <textarea
              value={editedConfig.description || ''}
              onChange={(e) => setEditedConfig({ ...editedConfig, description: e.target.value })}
              className="w-full px-3 py-2 text-sm border rounded-lg h-20"
              placeholder="输入配置描述"
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-sm font-medium mb-1 block">网格列数</label>
              <input
                type="number"
                min="1"
                max="24"
                value={editedConfig.layout.columns}
                onChange={(e) => setEditedConfig({
                  ...editedConfig,
                  layout: { ...editedConfig.layout, columns: parseInt(e.target.value) || 12 }
                })}
                className="w-full px-3 py-2 text-sm border rounded-lg"
              />
            </div>
            <div>
              <label className="text-sm font-medium mb-1 block">间距 (px)</label>
              <input
                type="number"
                min="0"
                value={editedConfig.layout.gap}
                onChange={(e) => setEditedConfig({
                  ...editedConfig,
                  layout: { ...editedConfig.layout, gap: parseInt(e.target.value) || 16 }
                })}
                className="w-full px-3 py-2 text-sm border rounded-lg"
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 标签页切换 */}
      <div className="flex gap-2 border-b">
        <button
          onClick={() => setActiveTab('widgets')}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
            activeTab === 'widgets' 
              ? 'border-primary text-primary' 
              : 'border-transparent text-muted-foreground hover:text-foreground'
          }`}
        >
          <Layout className="w-4 h-4 inline mr-2" />
          Widgets
        </button>
        <button
          onClick={() => setActiveTab('theme')}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
            activeTab === 'theme' 
              ? 'border-primary text-primary' 
              : 'border-transparent text-muted-foreground hover:text-foreground'
          }`}
        >
          <Palette className="w-4 h-4 inline mr-2" />
          主题样式
        </button>
        <button
          onClick={() => setActiveTab('layout')}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
            activeTab === 'layout' 
              ? 'border-primary text-primary' 
              : 'border-transparent text-muted-foreground hover:text-foreground'
          }`}
        >
          <Settings className="w-4 h-4 inline mr-2" />
          布局设置
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-4">
        {/* 左侧：Widget列表或主题/布局配置 */}
        <div className="lg:col-span-1">
          {activeTab === 'widgets' && (
            <>
              <Card>
                <CardHeader>
                  <CardTitle className="text-sm">添加Widget</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3 max-h-[600px] overflow-y-auto">
                  {widgetCategories.map(category => (
                    <div key={category} className="space-y-2">
                      <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">
                        {category}
                      </div>
                      {availableWidgetTypes
                        .filter(w => w.category === category)
                        .map(widgetType => (
                          <Button
                            key={widgetType.type}
                            variant="outline"
                            className="w-full justify-start text-xs"
                            onClick={() => addWidget(widgetType.type)}
                            leftIcon={<Plus className="w-3 h-3" />}
                          >
                            {widgetType.name}
                          </Button>
                        ))}
                    </div>
                  ))}
                </CardContent>
              </Card>
            </>
          )}

          {activeTab === 'theme' && (
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">主题配置</CardTitle>
              </CardHeader>
              <CardContent>
                <ThemeSelector
                  currentTheme={editedConfig.theme}
                  onThemeChange={(theme) => setEditedConfig({
                    ...editedConfig,
                    theme: { ...editedConfig.theme, ...theme }
                  })}
                />
                <div className="mt-6">
                  <StyleEditor
                    theme={editedConfig.theme}
                    onThemeChange={(theme) => setEditedConfig({
                      ...editedConfig,
                      theme: { ...editedConfig.theme, ...theme }
                    })}
                  />
                </div>
              </CardContent>
            </Card>
          )}

          {activeTab === 'layout' && (
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">布局配置</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <label className="text-xs font-medium mb-1 block">布局类型</label>
                  <select
                    value={editedConfig.layout.type}
                    onChange={(e) => setEditedConfig({
                      ...editedConfig,
                      layout: { ...editedConfig.layout, type: e.target.value as any }
                    })}
                    className="w-full px-2 py-1 text-xs border rounded"
                  >
                    <option value="grid">网格布局</option>
                    <option value="masonry">瀑布流</option>
                    <option value="flex">弹性布局</option>
                    <option value="carousel">轮播</option>
                    <option value="tabs">标签页</option>
                  </select>
                </div>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="text-xs font-medium mb-1 block">列数</label>
                    <input
                      type="number"
                      min="1"
                      max="24"
                      value={editedConfig.layout.columns}
                      onChange={(e) => setEditedConfig({
                        ...editedConfig,
                        layout: { ...editedConfig.layout, columns: parseInt(e.target.value) || 12 }
                      })}
                      className="w-full px-2 py-1 text-xs border rounded"
                    />
                  </div>
                  <div>
                    <label className="text-xs font-medium mb-1 block">间距 (px)</label>
                    <input
                      type="number"
                      min="0"
                      value={editedConfig.layout.gap}
                      onChange={(e) => setEditedConfig({
                        ...editedConfig,
                        layout: { ...editedConfig.layout, gap: parseInt(e.target.value) || 16 }
                      })}
                      className="w-full px-2 py-1 text-xs border rounded"
                    />
                  </div>
                </div>
                <div>
                  <label className="text-xs font-medium mb-1 block">最大宽度 (px)</label>
                  <input
                    type="number"
                    value={editedConfig.layout.maxWidth || ''}
                    onChange={(e) => setEditedConfig({
                      ...editedConfig,
                      layout: { ...editedConfig.layout, maxWidth: e.target.value ? parseInt(e.target.value) : undefined }
                    })}
                    className="w-full px-2 py-1 text-xs border rounded"
                    placeholder="留空为全宽"
                  />
                </div>
                
                {/* 页面边距设置 */}
                <div className="border-t pt-4 mt-4">
                  <label className="text-xs font-medium mb-2 block">页面边距 (px)</label>
                  <p className="text-xs text-muted-foreground mb-2">支持负数，例如: -5, 0, 10</p>
                  <div className="grid grid-cols-2 gap-2">
                    <div>
                      <label className="text-xs text-muted-foreground mb-1 block">上</label>
                      <input
                        type="number"
                        value={editedConfig.layout.padding?.top ?? 24}
                        onChange={(e) => {
                          const currentPadding = editedConfig.layout.padding || { top: 24, right: 24, bottom: 24, left: 24 }
                          const value = e.target.value === '' ? 0 : parseInt(e.target.value) || 0
                          setEditedConfig({
                            ...editedConfig,
                            layout: {
                              ...editedConfig.layout,
                              padding: {
                                top: value,
                                right: currentPadding.right ?? 24,
                                bottom: currentPadding.bottom ?? 24,
                                left: currentPadding.left ?? 24
                              }
                            }
                          })
                        }}
                        className="w-full px-2 py-1 text-xs border rounded"
                      />
                    </div>
                    <div>
                      <label className="text-xs text-muted-foreground mb-1 block">右</label>
                      <input
                        type="number"
                        value={editedConfig.layout.padding?.right ?? 24}
                        onChange={(e) => {
                          const currentPadding = editedConfig.layout.padding || { top: 24, right: 24, bottom: 24, left: 24 }
                          const value = e.target.value === '' ? 0 : parseInt(e.target.value) || 0
                          setEditedConfig({
                            ...editedConfig,
                            layout: {
                              ...editedConfig.layout,
                              padding: {
                                top: currentPadding.top ?? 24,
                                right: value,
                                bottom: currentPadding.bottom ?? 24,
                                left: currentPadding.left ?? 24
                              }
                            }
                          })
                        }}
                        className="w-full px-2 py-1 text-xs border rounded"
                      />
                    </div>
                    <div>
                      <label className="text-xs text-muted-foreground mb-1 block">下</label>
                      <input
                        type="number"
                        value={editedConfig.layout.padding?.bottom ?? 24}
                        onChange={(e) => {
                          const currentPadding = editedConfig.layout.padding || { top: 24, right: 24, bottom: 24, left: 24 }
                          const value = e.target.value === '' ? 0 : parseInt(e.target.value) || 0
                          setEditedConfig({
                            ...editedConfig,
                            layout: {
                              ...editedConfig.layout,
                              padding: {
                                top: currentPadding.top ?? 24,
                                right: currentPadding.right ?? 24,
                                bottom: value,
                                left: currentPadding.left ?? 24
                              }
                            }
                          })
                        }}
                        className="w-full px-2 py-1 text-xs border rounded"
                      />
                    </div>
                    <div>
                      <label className="text-xs text-muted-foreground mb-1 block">左</label>
                      <input
                        type="number"
                        value={editedConfig.layout.padding?.left ?? 24}
                        onChange={(e) => {
                          const currentPadding = editedConfig.layout.padding || { top: 24, right: 24, bottom: 24, left: 24 }
                          const value = e.target.value === '' ? 0 : parseInt(e.target.value) || 0
                          setEditedConfig({
                            ...editedConfig,
                            layout: {
                              ...editedConfig.layout,
                              padding: {
                                top: currentPadding.top ?? 24,
                                right: currentPadding.right ?? 24,
                                bottom: currentPadding.bottom ?? 24,
                                left: value
                              }
                            }
                          })
                        }}
                        className="w-full px-2 py-1 text-xs border rounded"
                      />
                    </div>
                  </div>
                </div>

                {/* 预览区域内边距设置 */}
                <div className="border-t pt-4 mt-4">
                  <label className="text-xs font-medium mb-2 block">预览区域内边距 (px)</label>
                  <p className="text-xs text-muted-foreground mb-2">控制编辑页面右侧预览区域的内边距，支持负数，例如: -5, 0, 10</p>
                  <div className="grid grid-cols-2 gap-2">
                    <div>
                      <label className="text-xs text-muted-foreground mb-1 block">上</label>
                      <input
                        type="number"
                        value={editedConfig.layout.previewPadding?.top ?? 16}
                        onChange={(e) => {
                          const currentPadding = editedConfig.layout.previewPadding || { top: 16, right: 16, bottom: 16, left: 16 }
                          const value = e.target.value === '' ? 0 : parseInt(e.target.value) || 0
                          setEditedConfig({
                            ...editedConfig,
                            layout: {
                              ...editedConfig.layout,
                              previewPadding: {
                                top: value,
                                right: currentPadding.right ?? 16,
                                bottom: currentPadding.bottom ?? 16,
                                left: currentPadding.left ?? 16
                              }
                            }
                          })
                        }}
                        className="w-full px-2 py-1 text-xs border rounded"
                      />
                    </div>
                    <div>
                      <label className="text-xs text-muted-foreground mb-1 block">右</label>
                      <input
                        type="number"
                        value={editedConfig.layout.previewPadding?.right ?? 16}
                        onChange={(e) => {
                          const currentPadding = editedConfig.layout.previewPadding || { top: 16, right: 16, bottom: 16, left: 16 }
                          const value = e.target.value === '' ? 0 : parseInt(e.target.value) || 0
                          setEditedConfig({
                            ...editedConfig,
                            layout: {
                              ...editedConfig.layout,
                              previewPadding: {
                                top: currentPadding.top ?? 16,
                                right: value,
                                bottom: currentPadding.bottom ?? 16,
                                left: currentPadding.left ?? 16
                              }
                            }
                          })
                        }}
                        className="w-full px-2 py-1 text-xs border rounded"
                      />
                    </div>
                    <div>
                      <label className="text-xs text-muted-foreground mb-1 block">下</label>
                      <input
                        type="number"
                        value={editedConfig.layout.previewPadding?.bottom ?? 16}
                        onChange={(e) => {
                          const currentPadding = editedConfig.layout.previewPadding || { top: 16, right: 16, bottom: 16, left: 16 }
                          const value = e.target.value === '' ? 0 : parseInt(e.target.value) || 0
                          setEditedConfig({
                            ...editedConfig,
                            layout: {
                              ...editedConfig.layout,
                              previewPadding: {
                                top: currentPadding.top ?? 16,
                                right: currentPadding.right ?? 16,
                                bottom: value,
                                left: currentPadding.left ?? 16
                              }
                            }
                          })
                        }}
                        className="w-full px-2 py-1 text-xs border rounded"
                      />
                    </div>
                    <div>
                      <label className="text-xs text-muted-foreground mb-1 block">左</label>
                      <input
                        type="number"
                        value={editedConfig.layout.previewPadding?.left ?? 16}
                        onChange={(e) => {
                          const currentPadding = editedConfig.layout.previewPadding || { top: 16, right: 16, bottom: 16, left: 16 }
                          const value = e.target.value === '' ? 0 : parseInt(e.target.value) || 0
                          setEditedConfig({
                            ...editedConfig,
                            layout: {
                              ...editedConfig.layout,
                              previewPadding: {
                                top: currentPadding.top ?? 16,
                                right: currentPadding.right ?? 16,
                                bottom: currentPadding.bottom ?? 16,
                                left: value
                              }
                            }
                          })
                        }}
                        className="w-full px-2 py-1 text-xs border rounded"
                      />
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Widget列表 - 始终显示 */}
          {activeTab === 'widgets' && (
            <Card className="mt-4">
              <CardHeader>
                <CardTitle className="text-sm">Widget列表 ({editedConfig.widgets.length})</CardTitle>
              </CardHeader>
              <CardContent>
                <DragSort
                  items={editedConfig.widgets.map(w => ({ id: w.id, data: w }))}
                  onSort={(sorted) => {
                    const sortedWidgets = sorted.map(item => item.data as WidgetConfig)
                    handleSort(sortedWidgets)
                  }}
                  className="space-y-2"
                >
                  {(item, _, isDragging) => {
                    const widget = item.data as WidgetConfig
                    return (
                    <div
                      className={`p-2 border rounded-lg cursor-move ${
                        selectedWidget?.id === widget.id ? 'border-primary bg-primary/10' : ''
                      } ${isDragging ? 'opacity-50' : ''}`}
                      onClick={() => setSelectedWidget(widget)}
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2 flex-1 min-w-0">
                          <GripVertical className="w-4 h-4 text-muted-foreground flex-shrink-0" />
                          <span className="text-xs font-medium truncate">{widget.title}</span>
                        </div>
                        <div className="flex items-center gap-1 flex-shrink-0">
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-6 w-6 p-0"
                            onClick={(e) => {
                              e.stopPropagation()
                              updateWidget(widget.id, { visible: !widget.visible })
                            }}
                          >
                            {widget.visible ? (
                              <Eye className="w-3 h-3" />
                            ) : (
                              <EyeOff className="w-3 h-3" />
                            )}
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-6 w-6 p-0"
                            onClick={(e) => {
                              e.stopPropagation()
                              removeWidget(widget.id)
                            }}
                          >
                            <Trash2 className="w-3 h-3" />
                          </Button>
                        </div>
                      </div>
                    </div>
                    )
                  }}
                </DragSort>
              </CardContent>
            </Card>
          )}
        </div>

        {/* 中间：预览区域 */}
        <div className="lg:col-span-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">实时预览</CardTitle>
            </CardHeader>
            <CardContent>
              <div 
                className="grid gap-4 bg-muted/30 rounded-lg"
                style={{
                  gridTemplateColumns: `repeat(${editedConfig.layout.columns}, minmax(0, 1fr))`,
                  gap: `${editedConfig.layout.gap}px`,
                  backgroundColor: editedConfig.theme.backgroundColor,
                  padding: editedConfig.layout.previewPadding
                    ? `${editedConfig.layout.previewPadding.top || 16}px ${editedConfig.layout.previewPadding.right || 16}px ${editedConfig.layout.previewPadding.bottom || 16}px ${editedConfig.layout.previewPadding.left || 16}px`
                    : '16px'
                }}
              >
                {editedConfig.widgets
                  .filter(w => w.visible)
                  .sort((a, b) => {
                    if (a.position.y !== b.position.y) return a.position.y - b.position.y
                    return a.position.x - b.position.x
                  })
                  .map(widget => (
                    <div
                      key={widget.id}
                      className={`border-2 rounded-lg p-2 cursor-pointer transition-all ${
                        selectedWidget?.id === widget.id 
                          ? 'border-primary ring-2 ring-primary/20' 
                          : 'border-dashed border-muted-foreground/30 hover:border-primary/50'
                      }`}
                      style={{
                        gridColumn: `span ${widget.position.w}`,
                        gridRow: `span ${widget.position.h}`,
                        minHeight: `${widget.position.h * 40}px`,
                        backgroundColor: widget.style?.backgroundColor,
                        borderRadius: widget.style?.borderRadius
                      }}
                      onClick={() => setSelectedWidget(widget)}
                    >
                      <div className="text-xs font-medium mb-1" style={{ color: widget.style?.textColor }}>
                        {widget.title}
                      </div>
                      <div className="text-xs text-muted-foreground/50">{widget.type}</div>
                    </div>
                  ))}
                {editedConfig.widgets.filter(w => w.visible).length === 0 && (
                  <div className="col-span-full text-center py-12 text-muted-foreground">
                    <p className="text-sm">暂无Widget，从左侧添加</p>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* 右侧：属性编辑 */}
        <div className="lg:col-span-1">
          {selectedWidget ? (
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Widget属性</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <label className="text-xs font-medium mb-1 block">标题</label>
                  <input
                    type="text"
                    value={selectedWidget.title}
                    onChange={(e) => updateWidget(selectedWidget.id, { title: e.target.value })}
                    className="w-full px-2 py-1 text-sm border rounded"
                  />
                </div>

                <div>
                  <label className="text-xs font-medium mb-1 block">尺寸</label>
                  <select
                    value={selectedWidget.size}
                    onChange={(e) => {
                      const size = e.target.value as WidgetSize
                      updateWidget(selectedWidget.id, {
                        size,
                        position: {
                          ...selectedWidget.position,
                          w: widgetSizeMap[size]
                        }
                      })
                    }}
                    className="w-full px-2 py-1 text-sm border rounded"
                  >
                    <option value="small">小 (1/4)</option>
                    <option value="medium">中 (1/2)</option>
                    <option value="large">大 (3/4)</option>
                    <option value="full">全宽</option>
                  </select>
                </div>

                <div>
                  <label className="text-xs font-medium mb-1 block">高度 (行数)</label>
                  <input
                    type="number"
                    min="1"
                    max="20"
                    value={selectedWidget.position.h}
                    onChange={(e) => {
                      const h = parseInt(e.target.value) || 1
                      updateWidget(selectedWidget.id, {
                        position: {
                          ...selectedWidget.position,
                          h: Math.max(1, Math.min(20, h))
                        }
                      })
                    }}
                    className="w-full px-2 py-1 text-sm border rounded"
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    当前高度: {selectedWidget.position.h} 行 (约 {selectedWidget.position.h * 60}px)
                  </p>
                </div>

                <div>
                  <label className="text-xs font-medium mb-1 block">可见性</label>
                  <label className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={selectedWidget.visible}
                      onChange={(e) => updateWidget(selectedWidget.id, { visible: e.target.checked })}
                    />
                    <span className="text-sm">显示</span>
                  </label>
                </div>

                {/* Widget样式编辑 */}
                <div className="border-t pt-4 mt-4">
                  <label className="text-xs font-medium mb-2 block">样式配置</label>
                  <div className="space-y-2">
                    <div>
                      <label className="text-xs text-muted-foreground mb-1 block">背景色</label>
                      <div className="flex gap-2">
                        <input
                          type="color"
                          value={selectedWidget.style?.backgroundColor || '#ffffff'}
                          onChange={(e) => updateWidget(selectedWidget.id, {
                            style: { ...selectedWidget.style, backgroundColor: e.target.value }
                          })}
                          className="w-10 h-8 rounded border cursor-pointer"
                        />
                        <input
                          type="text"
                          value={selectedWidget.style?.backgroundColor || ''}
                          onChange={(e) => updateWidget(selectedWidget.id, {
                            style: { ...selectedWidget.style, backgroundColor: e.target.value }
                          })}
                          className="flex-1 px-2 py-1 text-xs border rounded"
                          placeholder="#ffffff"
                        />
                      </div>
                    </div>
                    <div>
                      <label className="text-xs text-muted-foreground mb-1 block">圆角</label>
                      <select
                        value={selectedWidget.style?.borderRadius || '8px'}
                        onChange={(e) => {
                          const value = e.target.value
                          updateWidget(selectedWidget.id, {
                            style: { 
                              ...selectedWidget.style, 
                              borderRadius: value === 'custom' ? selectedWidget.props?.customBorderRadius || '8px' : value
                            }
                          })
                        }}
                        className="w-full px-2 py-1 text-xs border rounded"
                      >
                        <option value="0px">无圆角 (0px)</option>
                        <option value="4px">小 (4px)</option>
                        <option value="8px">中 (8px)</option>
                        <option value="12px">大 (12px)</option>
                        <option value="16px">超大 (16px)</option>
                        <option value="20px">圆形 (20px)</option>
                        <option value="custom">自定义</option>
                      </select>
                      {(selectedWidget.style?.borderRadius === 'custom' || 
                        (selectedWidget.style?.borderRadius && 
                         !['0px', '4px', '8px', '12px', '16px', '20px'].includes(selectedWidget.style.borderRadius))) && (
                        <input
                          type="text"
                          value={selectedWidget.style?.borderRadius === 'custom' 
                            ? (selectedWidget.props?.customBorderRadius || '') 
                            : (selectedWidget.style?.borderRadius || '8px')}
                          onChange={(e) => updateWidget(selectedWidget.id, {
                            style: { 
                              ...selectedWidget.style, 
                              borderRadius: e.target.value || '8px'
                            },
                            props: {
                              ...selectedWidget.props,
                              customBorderRadius: e.target.value
                            }
                          })}
                          className="w-full px-2 py-1 text-xs border rounded mt-2"
                          placeholder="例如: 8px, 50%"
                        />
                      )}
                    </div>
                    <div>
                      <label className="text-xs font-medium mb-2 block">内边距 (px)</label>
                      <p className="text-xs text-muted-foreground mb-2">支持负数，例如: -5, 0, 10</p>
                      <div className="grid grid-cols-2 gap-2">
                        <div>
                          <label className="text-xs text-muted-foreground mb-1 block">上</label>
                          <input
                            type="number"
                            value={
                              typeof selectedWidget.style?.padding === 'object' 
                                ? (selectedWidget.style.padding.top ?? 16)
                                : (typeof selectedWidget.style?.padding === 'string' 
                                    ? parseInt(selectedWidget.style.padding) || 16 
                                    : 16)
                            }
                            onChange={(e) => {
                              const value = e.target.value === '' ? 0 : parseInt(e.target.value) || 0
                              const currentPadding = typeof selectedWidget.style?.padding === 'object'
                                ? selectedWidget.style.padding
                                : { top: 16, right: 16, bottom: 16, left: 16 }
                              updateWidget(selectedWidget.id, {
                                style: {
                                  ...selectedWidget.style,
                                  padding: {
                                    top: value,
                                    right: currentPadding.right ?? 16,
                                    bottom: currentPadding.bottom ?? 16,
                                    left: currentPadding.left ?? 16
                                  }
                                }
                              })
                            }}
                            className="w-full px-2 py-1 text-xs border rounded"
                          />
                        </div>
                        <div>
                          <label className="text-xs text-muted-foreground mb-1 block">右</label>
                          <input
                            type="number"
                            value={
                              typeof selectedWidget.style?.padding === 'object' 
                                ? (selectedWidget.style.padding.right ?? 16)
                                : (typeof selectedWidget.style?.padding === 'string' 
                                    ? parseInt(selectedWidget.style.padding) || 16 
                                    : 16)
                            }
                            onChange={(e) => {
                              const value = e.target.value === '' ? 0 : parseInt(e.target.value) || 0
                              const currentPadding = typeof selectedWidget.style?.padding === 'object'
                                ? selectedWidget.style.padding
                                : { top: 16, right: 16, bottom: 16, left: 16 }
                              updateWidget(selectedWidget.id, {
                                style: {
                                  ...selectedWidget.style,
                                  padding: {
                                    top: currentPadding.top ?? 16,
                                    right: value,
                                    bottom: currentPadding.bottom ?? 16,
                                    left: currentPadding.left ?? 16
                                  }
                                }
                              })
                            }}
                            className="w-full px-2 py-1 text-xs border rounded"
                          />
                        </div>
                        <div>
                          <label className="text-xs text-muted-foreground mb-1 block">下</label>
                          <input
                            type="number"
                            value={
                              typeof selectedWidget.style?.padding === 'object' 
                                ? (selectedWidget.style.padding.bottom ?? 16)
                                : (typeof selectedWidget.style?.padding === 'string' 
                                    ? parseInt(selectedWidget.style.padding) || 16 
                                    : 16)
                            }
                            onChange={(e) => {
                              const value = e.target.value === '' ? 0 : parseInt(e.target.value) || 0
                              const currentPadding = typeof selectedWidget.style?.padding === 'object'
                                ? selectedWidget.style.padding
                                : { top: 16, right: 16, bottom: 16, left: 16 }
                              updateWidget(selectedWidget.id, {
                                style: {
                                  ...selectedWidget.style,
                                  padding: {
                                    top: currentPadding.top ?? 16,
                                    right: currentPadding.right ?? 16,
                                    bottom: value,
                                    left: currentPadding.left ?? 16
                                  }
                                }
                              })
                            }}
                            className="w-full px-2 py-1 text-xs border rounded"
                          />
                        </div>
                        <div>
                          <label className="text-xs text-muted-foreground mb-1 block">左</label>
                          <input
                            type="number"
                            value={
                              typeof selectedWidget.style?.padding === 'object' 
                                ? (selectedWidget.style.padding.left ?? 16)
                                : (typeof selectedWidget.style?.padding === 'string' 
                                    ? parseInt(selectedWidget.style.padding) || 16 
                                    : 16)
                            }
                            onChange={(e) => {
                              const value = e.target.value === '' ? 0 : parseInt(e.target.value) || 0
                              const currentPadding = typeof selectedWidget.style?.padding === 'object'
                                ? selectedWidget.style.padding
                                : { top: 16, right: 16, bottom: 16, left: 16 }
                              updateWidget(selectedWidget.id, {
                                style: {
                                  ...selectedWidget.style,
                                  padding: {
                                    top: currentPadding.top ?? 16,
                                    right: currentPadding.right ?? 16,
                                    bottom: currentPadding.bottom ?? 16,
                                    left: value
                                  }
                                }
                              })
                            }}
                            className="w-full px-2 py-1 text-xs border rounded"
                          />
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                {/* 根据Widget类型显示不同的属性编辑 */}
                {selectedWidget.type === 'stat-card' && (
                  <div className="border-t pt-4 mt-4">
                    <label className="text-xs font-medium mb-2 block">数据配置</label>
                    <div>
                      <label className="text-xs text-muted-foreground mb-1 block">数据键</label>
                      <select
                        value={selectedWidget.props?.dataKey || ''}
                        onChange={(e) => updateWidget(selectedWidget.id, {
                          props: { ...selectedWidget.props, dataKey: e.target.value }
                        })}
                        className="w-full px-2 py-1 text-xs border rounded"
                      >
                        <option value="">-- 选择数据键 --</option>
                        <option value="totalMembers">成员总数 (totalMembers)</option>
                        <option value="totalAssistants">助手总数 (totalAssistants)</option>
                        <option value="totalKnowledgeBases">知识库总数 (totalKnowledgeBases)</option>
                        <option value="totalCalls">通话总数 (totalCalls)</option>
                        <option value="totalWorkflows">工作流总数 (totalWorkflows)</option>
                        <option value="totalScripts">脚本总数 (totalScripts)</option>
                        <option value="totalDevices">设备总数 (totalDevices)</option>
                        <option value="totalVoices">音色总数 (totalVoices)</option>
                        <option value="custom">自定义</option>
                      </select>
                      {selectedWidget.props?.dataKey === 'custom' && (
                        <input
                          type="text"
                          value={selectedWidget.props?.customDataKey || ''}
                          onChange={(e) => updateWidget(selectedWidget.id, {
                            props: { 
                              ...selectedWidget.props, 
                              customDataKey: e.target.value,
                              dataKey: e.target.value || 'custom'
                            }
                          })}
                          className="w-full px-2 py-1 text-xs border rounded mt-2"
                          placeholder="输入自定义数据键"
                        />
                      )}
                    </div>
                    <div className="mt-2">
                      <label className="text-xs text-muted-foreground mb-1 block">默认值</label>
                      <input
                        type="text"
                        value={selectedWidget.props?.defaultValue || ''}
                        onChange={(e) => updateWidget(selectedWidget.id, {
                          props: { ...selectedWidget.props, defaultValue: e.target.value }
                        })}
                        className="w-full px-2 py-1 text-xs border rounded"
                        placeholder="当数据键无数据时显示"
                      />
                    </div>
                  </div>
                )}

                {selectedWidget.type === 'custom-html' && (
                  <div className="border-t pt-4 mt-4">
                    <label className="text-xs font-medium mb-2 block">HTML内容</label>
                    <HtmlCodeEditor
                      value={selectedWidget.props?.html || ''}
                      onChange={(value) => updateWidget(selectedWidget.id, {
                        props: { ...selectedWidget.props, html: value }
                      })}
                    />
                  </div>
                )}

                {selectedWidget.type === 'hero-banner' && (
                  <div className="border-t pt-4 mt-4">
                    <label className="text-xs font-medium mb-2 block">横幅配置</label>
                    <div>
                      <label className="text-xs text-muted-foreground mb-1 block">标题</label>
                      <input
                        type="text"
                        value={selectedWidget.props?.title || ''}
                        onChange={(e) => updateWidget(selectedWidget.id, {
                          props: { ...selectedWidget.props, title: e.target.value }
                        })}
                        className="w-full px-2 py-1 text-xs border rounded"
                      />
                    </div>
                    <div className="mt-2">
                      <label className="text-xs text-muted-foreground mb-1 block">副标题</label>
                      <input
                        type="text"
                        value={selectedWidget.props?.subtitle || ''}
                        onChange={(e) => updateWidget(selectedWidget.id, {
                          props: { ...selectedWidget.props, subtitle: e.target.value }
                        })}
                        className="w-full px-2 py-1 text-xs border rounded"
                      />
                    </div>
                    <div className="mt-2">
                      <label className="text-xs text-muted-foreground mb-1 block">背景图片URL</label>
                      <input
                        type="text"
                        value={selectedWidget.props?.backgroundImage || ''}
                        onChange={(e) => updateWidget(selectedWidget.id, {
                          props: { ...selectedWidget.props, backgroundImage: e.target.value }
                        })}
                        className="w-full px-2 py-1 text-xs border rounded"
                        placeholder="https://..."
                      />
                    </div>
                  </div>
                )}

                {selectedWidget.type.startsWith('chart-') && (
                  <div className="border-t pt-4 mt-4">
                    <label className="text-xs font-medium mb-2 block">图表配置</label>
                    <div>
                      <label className="text-xs text-muted-foreground mb-1 block">数据键</label>
                      <select
                        value={selectedWidget.props?.dataKey || 'chartData'}
                        onChange={(e) => updateWidget(selectedWidget.id, {
                          props: { ...selectedWidget.props, dataKey: e.target.value }
                        })}
                        className="w-full px-2 py-1 text-xs border rounded"
                      >
                        <option value="chartData">图表数据 (chartData)</option>
                        <option value="usageTrend">使用趋势 (usageTrend)</option>
                        <option value="activityData">活动数据 (activityData)</option>
                        <option value="custom">自定义</option>
                      </select>
                      {selectedWidget.props?.dataKey === 'custom' && (
                        <input
                          type="text"
                          value={selectedWidget.props?.customDataKey || ''}
                          onChange={(e) => updateWidget(selectedWidget.id, {
                            props: { 
                              ...selectedWidget.props, 
                              customDataKey: e.target.value,
                              dataKey: e.target.value || 'custom'
                            }
                          })}
                          className="w-full px-2 py-1 text-xs border rounded mt-2"
                          placeholder="输入自定义数据键"
                        />
                      )}
                    </div>
                    <div className="mt-2">
                      <label className="text-xs text-muted-foreground mb-1 block">X轴键</label>
                      <select
                        value={selectedWidget.props?.xAxisKey || 'name'}
                        onChange={(e) => updateWidget(selectedWidget.id, {
                          props: { ...selectedWidget.props, xAxisKey: e.target.value }
                        })}
                        className="w-full px-2 py-1 text-xs border rounded"
                      >
                        <option value="name">名称 (name)</option>
                        <option value="date">日期 (date)</option>
                        <option value="time">时间 (time)</option>
                        <option value="label">标签 (label)</option>
                        <option value="custom">自定义</option>
                      </select>
                      {selectedWidget.props?.xAxisKey === 'custom' && (
                        <input
                          type="text"
                          value={selectedWidget.props?.customXAxisKey || ''}
                          onChange={(e) => updateWidget(selectedWidget.id, {
                            props: { 
                              ...selectedWidget.props, 
                              customXAxisKey: e.target.value,
                              xAxisKey: e.target.value || 'custom'
                            }
                          })}
                          className="w-full px-2 py-1 text-xs border rounded mt-2"
                          placeholder="输入自定义X轴键"
                        />
                      )}
                    </div>
                  </div>
                )}

                {selectedWidget.type === 'metric-comparison' && (
                  <div className="border-t pt-4 mt-4">
                    <label className="text-xs font-medium mb-2 block">指标对比配置</label>
                    <div>
                      <label className="text-xs text-muted-foreground mb-1 block">数据键</label>
                      <select
                        value={selectedWidget.props?.dataKey || ''}
                        onChange={(e) => updateWidget(selectedWidget.id, {
                          props: { ...selectedWidget.props, dataKey: e.target.value }
                        })}
                        className="w-full px-2 py-1 text-xs border rounded"
                      >
                        <option value="">-- 选择数据键 --</option>
                        <option value="billStatistics">账单统计 (billStatistics)</option>
                        <option value="metrics">指标数据 (metrics)</option>
                        <option value="custom">自定义</option>
                      </select>
                      {selectedWidget.props?.dataKey === 'custom' && (
                        <input
                          type="text"
                          value={selectedWidget.props?.customDataKey || ''}
                          onChange={(e) => updateWidget(selectedWidget.id, {
                            props: { 
                              ...selectedWidget.props, 
                              customDataKey: e.target.value,
                              dataKey: e.target.value || 'custom'
                            }
                          })}
                          className="w-full px-2 py-1 text-xs border rounded mt-2"
                          placeholder="输入自定义数据键"
                        />
                      )}
                    </div>
                  </div>
                )}

                {selectedWidget.type === 'progress-ring' && (
                  <div className="border-t pt-4 mt-4">
                    <label className="text-xs font-medium mb-2 block">进度环配置</label>
                    <div>
                      <label className="text-xs text-muted-foreground mb-1 block">数据键 (值)</label>
                      <select
                        value={selectedWidget.props?.dataKey || ''}
                        onChange={(e) => updateWidget(selectedWidget.id, {
                          props: { ...selectedWidget.props, dataKey: e.target.value }
                        })}
                        className="w-full px-2 py-1 text-xs border rounded"
                      >
                        <option value="">-- 选择数据键 --</option>
                        <option value="value">进度值 (value)</option>
                        <option value="totalMembers">成员总数 (totalMembers)</option>
                        <option value="totalAssistants">助手总数 (totalAssistants)</option>
                        <option value="totalLLMCalls">LLM调用次数 (billStatistics.totalLLMCalls)</option>
                        <option value="totalCallCount">通话次数 (billStatistics.totalCallCount)</option>
                        <option value="totalASRCount">ASR次数 (billStatistics.totalASRCount)</option>
                        <option value="totalTTSCount">TTS次数 (billStatistics.totalTTSCount)</option>
                        <option value="totalAPICalls">API调用次数 (billStatistics.totalAPICalls)</option>
                        <option value="custom">自定义</option>
                      </select>
                      {selectedWidget.props?.dataKey === 'custom' && (
                        <input
                          type="text"
                          value={selectedWidget.props?.customDataKey || ''}
                          onChange={(e) => updateWidget(selectedWidget.id, {
                            props: { 
                              ...selectedWidget.props, 
                              customDataKey: e.target.value,
                              dataKey: e.target.value || 'custom'
                            }
                          })}
                          className="w-full px-2 py-1 text-xs border rounded mt-2"
                          placeholder="输入自定义数据键"
                        />
                      )}
                    </div>
                    <div className="mt-2">
                      <label className="text-xs text-muted-foreground mb-1 block">当前值</label>
                      <input
                        type="number"
                        min="0"
                        value={selectedWidget.props?.value || 0}
                        onChange={(e) => updateWidget(selectedWidget.id, {
                          props: { ...selectedWidget.props, value: parseInt(e.target.value) || 0 }
                        })}
                        className="w-full px-2 py-1 text-xs border rounded"
                      />
                    </div>
                    <div className="mt-2">
                      <label className="text-xs text-muted-foreground mb-1 block">最大值</label>
                      <select
                        value={selectedWidget.props?.max || 100}
                        onChange={(e) => updateWidget(selectedWidget.id, {
                          props: { ...selectedWidget.props, max: parseInt(e.target.value) || 100 }
                        })}
                        className="w-full px-2 py-1 text-xs border rounded"
                      >
                        {[50, 100, 200, 500, 1000, 5000, 10000].map(num => (
                          <option key={num} value={num}>{num}</option>
                        ))}
                        <option value="custom">自定义</option>
                      </select>
                      {selectedWidget.props?.max === 'custom' || (selectedWidget.props?.max && ![50, 100, 200, 500, 1000, 5000, 10000].includes(selectedWidget.props.max)) && (
                        <input
                          type="number"
                          min="1"
                          value={selectedWidget.props?.max === 'custom' ? '' : (selectedWidget.props?.max || 100)}
                          onChange={(e) => updateWidget(selectedWidget.id, {
                            props: { ...selectedWidget.props, max: parseInt(e.target.value) || 100 }
                          })}
                          className="w-full px-2 py-1 text-xs border rounded mt-2"
                          placeholder="输入最大值"
                        />
                      )}
                    </div>
                    <div className="mt-2">
                      <label className="text-xs text-muted-foreground mb-1 block">颜色</label>
                      <div className="flex gap-2">
                        <input
                          type="color"
                          value={selectedWidget.props?.color || '#6366f1'}
                          onChange={(e) => updateWidget(selectedWidget.id, {
                            props: { ...selectedWidget.props, color: e.target.value }
                          })}
                          className="w-12 h-8 border rounded cursor-pointer"
                        />
                        <select
                          value={selectedWidget.props?.color || '#6366f1'}
                          onChange={(e) => updateWidget(selectedWidget.id, {
                            props: { ...selectedWidget.props, color: e.target.value }
                          })}
                          className="flex-1 px-2 py-1 text-xs border rounded"
                        >
                          <option value="#6366f1">靛蓝 (Indigo)</option>
                          <option value="#8b5cf6">紫色 (Purple)</option>
                          <option value="#ec4899">粉色 (Pink)</option>
                          <option value="#f59e0b">橙色 (Orange)</option>
                          <option value="#10b981">绿色 (Green)</option>
                          <option value="#3b82f6">蓝色 (Blue)</option>
                          <option value="#ef4444">红色 (Red)</option>
                          <option value="#f97316">橙红 (Orange Red)</option>
                        </select>
                      </div>
                    </div>
                  </div>
                )}

                {selectedWidget.type === 'image' && (
                  <div className="border-t pt-4 mt-4">
                    <label className="text-xs font-medium mb-2 block">图片配置</label>
                    <div>
                      <label className="text-xs text-muted-foreground mb-1 block">图片URL</label>
                      <input
                        type="text"
                        value={selectedWidget.props?.imageUrl || selectedWidget.props?.url || ''}
                        onChange={(e) => updateWidget(selectedWidget.id, {
                          props: { ...selectedWidget.props, imageUrl: e.target.value, url: e.target.value }
                        })}
                        className="w-full px-2 py-1 text-xs border rounded"
                        placeholder="https://..."
                      />
                    </div>
                    <div className="mt-2">
                      <label className="text-xs text-muted-foreground mb-1 block">替代文本</label>
                      <input
                        type="text"
                        value={selectedWidget.props?.alt || ''}
                        onChange={(e) => updateWidget(selectedWidget.id, {
                          props: { ...selectedWidget.props, alt: e.target.value }
                        })}
                        className="w-full px-2 py-1 text-xs border rounded"
                      />
                    </div>
                  </div>
                )}

                {selectedWidget.type === 'video' && (
                  <div className="border-t pt-4 mt-4">
                    <label className="text-xs font-medium mb-2 block">视频配置</label>
                    <div>
                      <label className="text-xs text-muted-foreground mb-1 block">视频URL</label>
                      <input
                        type="text"
                        value={selectedWidget.props?.videoUrl || selectedWidget.props?.url || ''}
                        onChange={(e) => updateWidget(selectedWidget.id, {
                          props: { ...selectedWidget.props, videoUrl: e.target.value, url: e.target.value }
                        })}
                        className="w-full px-2 py-1 text-xs border rounded"
                        placeholder="https://..."
                      />
                    </div>
                    <div className="mt-2">
                      <label className="text-xs text-muted-foreground mb-1 block">视频类型</label>
                      <select
                        value={selectedWidget.props?.type || 'direct'}
                        onChange={(e) => updateWidget(selectedWidget.id, {
                          props: { ...selectedWidget.props, type: e.target.value }
                        })}
                        className="w-full px-2 py-1 text-xs border rounded"
                      >
                        <option value="direct">直接链接</option>
                        <option value="youtube">YouTube</option>
                        <option value="vimeo">Vimeo</option>
                        <option value="embed">嵌入代码</option>
                      </select>
                    </div>
                  </div>
                )}

                {selectedWidget.type === 'iframe' && (
                  <div className="border-t pt-4 mt-4">
                    <label className="text-xs font-medium mb-2 block">嵌入页面配置</label>
                    <div>
                      <label className="text-xs text-muted-foreground mb-1 block">页面URL</label>
                      <input
                        type="text"
                        value={selectedWidget.props?.iframeUrl || selectedWidget.props?.url || ''}
                        onChange={(e) => updateWidget(selectedWidget.id, {
                          props: { ...selectedWidget.props, iframeUrl: e.target.value, url: e.target.value }
                        })}
                        className="w-full px-2 py-1 text-xs border rounded"
                        placeholder="https://..."
                      />
                    </div>
                  </div>
                )}

                {selectedWidget.type === 'markdown' && (
                  <div className="border-t pt-4 mt-4">
                    <label className="text-xs font-medium mb-2 block">Markdown内容</label>
                    <MarkdownCodeEditor
                      value={selectedWidget.props?.content || selectedWidget.props?.markdown || ''}
                      onChange={(value) => updateWidget(selectedWidget.id, {
                        props: { ...selectedWidget.props, content: value, markdown: value }
                      })}
                    />
                  </div>
                )}

                {selectedWidget.type === 'table' && (
                  <div className="border-t pt-4 mt-4">
                    <label className="text-xs font-medium mb-2 block">表格配置</label>
                    <div>
                      <label className="text-xs text-muted-foreground mb-1 block">列数</label>
                      <select
                        value={Array.isArray(selectedWidget.props?.columns) ? selectedWidget.props.columns.length : 3}
                        onChange={(e) => {
                          const colCount = parseInt(e.target.value) || 3
                          const currentCols = Array.isArray(selectedWidget.props?.columns) ? selectedWidget.props.columns : []
                          const newCols = Array.from({ length: colCount }, (_, i) => 
                            currentCols[i] || `列${i + 1}`
                          )
                          // 更新行数据以匹配新的列数
                          const currentRows = Array.isArray(selectedWidget.props?.rows) ? selectedWidget.props.rows : []
                          const newRows = currentRows.map(row => {
                            if (Array.isArray(row)) {
                              return Array.from({ length: colCount }, (_, i) => row[i] || '')
                            }
                            return Array(colCount).fill('')
                          })
                          updateWidget(selectedWidget.id, {
                            props: { 
                              ...selectedWidget.props, 
                              columns: newCols,
                              rows: newRows
                            }
                          })
                        }}
                        className="w-full px-2 py-1 text-xs border rounded"
                      >
                        {[1, 2, 3, 4, 5, 6, 7, 8].map(num => (
                          <option key={num} value={num}>{num} 列</option>
                        ))}
                      </select>
                    </div>
                    <div className="mt-2">
                      <label className="text-xs text-muted-foreground mb-1 block">列名配置</label>
                      <div className="space-y-1">
                        {Array.isArray(selectedWidget.props?.columns) && selectedWidget.props.columns.map((col: string, index: number) => (
                          <div key={index} className="flex gap-1">
                            <select
                              value={col}
                              onChange={(e) => {
                                const newCols = [...(selectedWidget.props?.columns || [])]
                                newCols[index] = e.target.value
                                updateWidget(selectedWidget.id, {
                                  props: { ...selectedWidget.props, columns: newCols }
                                })
                              }}
                              className="flex-1 px-2 py-1 text-xs border rounded"
                            >
                              <option value={`列${index + 1}`}>列{index + 1}</option>
                              <option value="名称">名称</option>
                              <option value="类型">类型</option>
                              <option value="状态">状态</option>
                              <option value="数量">数量</option>
                              <option value="日期">日期</option>
                              <option value="时间">时间</option>
                              <option value="用户">用户</option>
                              <option value="操作">操作</option>
                              <option value="描述">描述</option>
                              <option value="备注">备注</option>
                              <option value="ID">ID</option>
                              <option value="标题">标题</option>
                              <option value="内容">内容</option>
                              <option value="价格">价格</option>
                              <option value="金额">金额</option>
                              <option value="百分比">百分比</option>
                              <option value="进度">进度</option>
                              <option value="评分">评分</option>
                              <option value="优先级">优先级</option>
                              <option value="分类">分类</option>
                              <option value="标签">标签</option>
                              <option value="custom">自定义</option>
                            </select>
                            {col === 'custom' && (
                              <input
                                type="text"
                                value={selectedWidget.props?.customColumns?.[index] || ''}
                                onChange={(e) => {
                                  const newCols = [...(selectedWidget.props?.columns || [])]
                                  const customCols = [...(selectedWidget.props?.customColumns || [])]
                                  customCols[index] = e.target.value
                                  newCols[index] = e.target.value || 'custom'
                                  updateWidget(selectedWidget.id, {
                                    props: { 
                                      ...selectedWidget.props, 
                                      columns: newCols,
                                      customColumns: customCols
                                    }
                                  })
                                }}
                                className="flex-1 px-2 py-1 text-xs border rounded"
                                placeholder="输入自定义列名"
                              />
                            )}
                          </div>
                        ))}
                      </div>
                    </div>
                    <div className="mt-2">
                      <label className="text-xs text-muted-foreground mb-1 block">数据行数</label>
                      <select
                        value={Array.isArray(selectedWidget.props?.rows) ? selectedWidget.props.rows.length : 0}
                        onChange={(e) => {
                          const rowCount = parseInt(e.target.value) || 0
                          const currentRows = Array.isArray(selectedWidget.props?.rows) ? selectedWidget.props.rows : []
                          const colCount = Array.isArray(selectedWidget.props?.columns) ? selectedWidget.props.columns.length : 3
                          const newRows = Array.from({ length: rowCount }, (_, i) => 
                            currentRows[i] || Array(colCount).fill('')
                          )
                          updateWidget(selectedWidget.id, {
                            props: { ...selectedWidget.props, rows: newRows }
                          })
                        }}
                        className="w-full px-2 py-1 text-xs border rounded"
                      >
                        {[0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10].map(num => (
                          <option key={num} value={num}>{num} 行</option>
                        ))}
                      </select>
                    </div>
                    {Array.isArray(selectedWidget.props?.rows) && selectedWidget.props.rows.length > 0 && (
                      <div className="mt-2">
                        <label className="text-xs text-muted-foreground mb-1 block">行数据编辑</label>
                        <div className="space-y-1 max-h-32 overflow-y-auto border rounded p-2">
                          {selectedWidget.props.rows.map((row: any[], rowIndex: number) => (
                            <div key={rowIndex} className="flex gap-1">
                              {Array.isArray(row) && row.map((cell: any, cellIndex: number) => (
                                <input
                                  key={cellIndex}
                                  type="text"
                                  value={cell || ''}
                                  onChange={(e) => {
                                    const newRows = [...(selectedWidget.props?.rows || [])]
                                    if (!Array.isArray(newRows[rowIndex])) {
                                      newRows[rowIndex] = Array(selectedWidget.props?.columns?.length || 3).fill('')
                                    }
                                    newRows[rowIndex][cellIndex] = e.target.value
                                    updateWidget(selectedWidget.id, {
                                      props: { ...selectedWidget.props, rows: newRows }
                                    })
                                  }}
                                  className="flex-1 px-1 py-0.5 text-xs border rounded"
                                  placeholder={`行${rowIndex + 1}, 列${cellIndex + 1}`}
                                />
                              ))}
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </CardContent>
            </Card>
          ) : (
            <Card>
              <CardContent className="py-8 text-center text-muted-foreground">
                <p className="text-sm">选择一个Widget进行编辑</p>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  )
}

export default OverviewEditor

