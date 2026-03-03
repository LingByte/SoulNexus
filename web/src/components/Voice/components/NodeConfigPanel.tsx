import React, { Suspense } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { X, AlertCircle, Package, Maximize2, Minimize2 } from 'lucide-react'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import type { WorkflowNode } from '../types'

// Lazy load Monaco editor
const MonacoEditor = React.lazy(() => import('@monaco-editor/react'))

interface NodeConfigPanelProps {
  isOpen: boolean
  node: WorkflowNode | null
  nodeConfig: any
  onClose: () => void
  onNodeLabelChange: (label: string) => void
  onNodeConfigChange: (config: any) => void
  getIconComponent: (iconName: string) => React.ReactNode
  availableEventTypes: string[]
  installedPlugins: any[]
  loadingPlugins: boolean
  isCodeEditorFullscreen?: boolean
  setIsCodeEditorFullscreen?: (value: boolean) => void
}

/**
 * 节点配置面板组件
 * 用于配置工作流节点的参数
 */
export const NodeConfigPanel: React.FC<NodeConfigPanelProps> = ({
  isOpen,
  node,
  nodeConfig,
  onClose,
  onNodeLabelChange,
  onNodeConfigChange,
  getIconComponent,
  availableEventTypes,
  installedPlugins,
  loadingPlugins,
  isCodeEditorFullscreen = false,
  setIsCodeEditorFullscreen = () => {}
}) => {
  const [internalIsCodeEditorFullscreen, setInternalIsCodeEditorFullscreen] = React.useState(false)
  const fullscreenState = isCodeEditorFullscreen !== undefined ? isCodeEditorFullscreen : internalIsCodeEditorFullscreen
  const setFullscreenState = setIsCodeEditorFullscreen !== undefined ? setIsCodeEditorFullscreen : setInternalIsCodeEditorFullscreen
  
  if (!isOpen || !node) return null

  return (
    <AnimatePresence>
      <>
        {/* 背景遮罩 */}
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          onClick={onClose}
          className="fixed inset-0 bg-black/50 z-40"
        />
        
        {/* 抽屉内容 */}
        <motion.div
          initial={{ x: '100%' }}
          animate={{ x: 0 }}
          exit={{ x: '100%' }}
          transition={{ type: 'spring', damping: 25, stiffness: 200 }}
          className="fixed top-0 right-0 bottom-0 w-96 bg-white dark:bg-gray-800 shadow-2xl z-50 flex flex-col"
        >

          {/* 抽屉头部 */}
          <div className="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700 bg-gradient-to-r from-gray-50 to-white dark:from-gray-800 dark:to-gray-900">
            <div className="flex items-center gap-3">
              <div 
                className="p-2 rounded-lg"
                style={{ 
                  backgroundColor: `${nodeConfig.color || '#64748b'}15`,
                  color: nodeConfig.color || '#64748b'
                }}
              >
                {getIconComponent(nodeConfig.iconName)}
              </div>
              <div>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                  {node.data.label}
                </h3>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
                  {nodeConfig.label}
                </p>
              </div>
            </div>
            <Button
              variant="ghost"
              size="icon"
              onClick={onClose}
            >
              <X className="w-5 h-5" />
            </Button>
          </div>
          
          {/* 配置内容 */}
          <div className="flex-1 overflow-y-auto p-6 space-y-6">
            {/* 节点基本信息 */}
            <div className="space-y-4">
              <div>
                <h4 className="text-sm font-semibold text-gray-900 dark:text-white mb-3 flex items-center gap-2">
                  <div className="w-1 h-4 rounded-full" style={{ backgroundColor: nodeConfig.color || '#64748b' }} />
                  基本信息
                </h4>
                <Input
                  label="节点名称"
                  size="sm"
                  value={node.data.label}
                  onChange={(e) => onNodeLabelChange(e.target.value)}
                  placeholder="输入节点名称"
                />
              </div>
            </div>

            {/* 节点类型特定配置 */}
            <div className="space-y-4">
              <h4 className="text-sm font-semibold text-gray-900 dark:text-white mb-3 flex items-center gap-2">
                <div className="w-1 h-4 rounded-full" style={{ backgroundColor: nodeConfig.color || '#64748b' }} />
                配置参数
              </h4>

              {/* 根据节点类型渲染不同的配置UI */}
              <NodeTypeConfig
                node={node}
                onConfigChange={onNodeConfigChange}
                availableEventTypes={availableEventTypes}
                installedPlugins={installedPlugins}
                loadingPlugins={loadingPlugins}
                isCodeEditorFullscreen={fullscreenState}
                setIsCodeEditorFullscreen={setFullscreenState}
              />
            </div>
          </div>

          {/* 底部操作按钮 */}
          <div className="flex items-center justify-end gap-3 p-6 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900">
            <Button
              variant="outline"
              size="sm"
              onClick={onClose}
            >
              取消
            </Button>
            <Button
              variant="primary"
              size="sm"
              onClick={onClose}
            >
              保存
            </Button>
          </div>
        </motion.div>
      </>
    </AnimatePresence>
  )
}

/**
 * 节点类型配置组件
 * 根据不同的节点类型渲染不同的配置UI
 */
interface NodeTypeConfigProps {
  node: WorkflowNode
  onConfigChange: (config: any) => void
  availableEventTypes: string[]
  installedPlugins: any[]
  loadingPlugins: boolean
  isCodeEditorFullscreen: boolean
  setIsCodeEditorFullscreen: (value: boolean) => void
}

const NodeTypeConfig: React.FC<NodeTypeConfigProps> = ({
  node,
  onConfigChange,
  availableEventTypes,
  installedPlugins,
  loadingPlugins,
  isCodeEditorFullscreen,
  setIsCodeEditorFullscreen
}) => {
  const [newParam, setNewParam] = React.useState('')

  // 根据节点类型渲染不同的配置
  switch (node.type) {
    case 'start':
      return (
        <div className="space-y-4">
          <div className="p-3 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
            <div className="text-xs text-green-800 dark:text-green-200">
              <strong>开始节点：</strong>工作流的入口点。定义工作流需要接收的输入参数。
            </div>
          </div>

          <div className="space-y-3">
            <div className="flex gap-2">
              <Input
                value={newParam}
                onChange={(e) => setNewParam(e.target.value)}
                onKeyPress={(e) => {
                  if (e.key === 'Enter' && newParam.trim()) {
                    const inputs = node.inputs || []
                    if (!inputs.includes(newParam.trim())) {
                      onConfigChange({ inputs: [...inputs, newParam.trim()] })
                      setNewParam('')
                    }
                  }
                }}
                placeholder="输入参数名称"
                size="sm"
                className="flex-1"
              />
              <Button
                onClick={() => {
                  if (newParam.trim()) {
                    const inputs = node.inputs || []
                    if (!inputs.includes(newParam.trim())) {
                      onConfigChange({ inputs: [...inputs, newParam.trim()] })
                      setNewParam('')
                    }
                  }
                }}
                variant="success"
                size="sm"
              >
                添加
              </Button>
            </div>

            {node.inputs && node.inputs.length > 0 && (
              <div className="space-y-2">
                <p className="text-xs font-medium text-gray-700 dark:text-gray-300">输入参数列表：</p>
                <div className="space-y-1">
                  {node.inputs.map((input, idx) => (
                    <div key={idx} className="flex items-center justify-between p-2 bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded">
                      <span className="text-sm text-gray-900 dark:text-white">{input}</span>
                      <Button
                        onClick={() => {
                          const inputs = node.inputs.filter((_: string, i: number) => i !== idx)
                          onConfigChange({ inputs })
                        }}
                        variant="destructive"
                        size="xs"
                      >
                        删除
                      </Button>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      )

    case 'end':
      return (
        <div className="space-y-4">
          <div className="p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
            <div className="text-xs text-blue-800 dark:text-blue-200">
              <strong>结束节点：</strong>工作流的出口点。定义工作流的输出参数。
            </div>
          </div>

          <div className="space-y-3">
            <div className="flex gap-2">
              <Input
                value={newParam}
                onChange={(e) => setNewParam(e.target.value)}
                onKeyPress={(e) => {
                  if (e.key === 'Enter' && newParam.trim()) {
                    const outputs = node.outputs || []
                    if (!outputs.includes(newParam.trim())) {
                      onConfigChange({ outputs: [...outputs, newParam.trim()] })
                      setNewParam('')
                    }
                  }
                }}
                placeholder="输入参数名称"
                size="sm"
                className="flex-1"
              />
              <Button
                onClick={() => {
                  if (newParam.trim()) {
                    const outputs = node.outputs || []
                    if (!outputs.includes(newParam.trim())) {
                      onConfigChange({ outputs: [...outputs, newParam.trim()] })
                      setNewParam('')
                    }
                  }
                }}
                variant="primary"
                size="sm"
              >
                添加
              </Button>
            </div>

            {node.outputs && node.outputs.length > 0 && (
              <div className="space-y-2">
                <p className="text-xs font-medium text-gray-700 dark:text-gray-300">输出参数列表：</p>
                <div className="space-y-1">
                  {node.outputs.map((output, idx) => (
                    <div key={idx} className="flex items-center justify-between p-2 bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded">
                      <span className="text-sm text-gray-900 dark:text-white">{output}</span>
                      <Button
                        onClick={() => {
                          const outputs = node.outputs.filter((_: string, i: number) => i !== idx)
                          onConfigChange({ outputs })
                        }}
                        variant="destructive"
                        size="xs"
                      >
                        删除
                      </Button>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      )

    case 'event':
      return (
        <EventNodeConfig
          node={node}
          onConfigChange={onConfigChange}
          availableEventTypes={availableEventTypes}
        />
      )
    case 'task':
      return (
        <TaskNodeConfig
          node={node}
          onConfigChange={onConfigChange}
        />
      )
    case 'gateway':
      return (
        <GatewayNodeConfig
          node={node}
          onConfigChange={onConfigChange}
        />
      )
    case 'script':
      return (
        <ScriptNodeConfig
          node={node}
          onConfigChange={onConfigChange}
          isCodeEditorFullscreen={isCodeEditorFullscreen}
          setIsCodeEditorFullscreen={setIsCodeEditorFullscreen}
        />
      )
    case 'ai_chat':
      return (
        <AIChatNodeConfig
          node={node}
          onConfigChange={onConfigChange}
        />
      )
    case 'workflow_plugin':
      return (
        <WorkflowPluginNodeConfig
          node={node}
          onConfigChange={onConfigChange}
          installedPlugins={installedPlugins}
          loadingPlugins={loadingPlugins}
        />
      )
    case 'wait':
      return (
        <WaitNodeConfig
          node={node}
          onConfigChange={onConfigChange}
        />
      )
    default:
      return (
        <div className="text-sm text-gray-500 dark:text-gray-400">
          此节点类型暂无特殊配置
        </div>
      )
  }
}

/**
 * 事件节点配置
 */
const EventNodeConfig: React.FC<{
  node: WorkflowNode
  onConfigChange: (config: any) => void
  availableEventTypes: string[]
}> = ({ node, onConfigChange, availableEventTypes }) => {
  return (
    <div className="space-y-4">
      <div className="p-3 bg-orange-50 dark:bg-orange-900/20 border border-orange-200 dark:border-orange-800 rounded-lg">
        <div className="text-xs text-orange-800 dark:text-orange-200 space-y-2">
          <div>
            <strong>事件节点：</strong>发布事件到事件总线，自动触发其他工作流执行。
          </div>
          <div className="mt-2 space-y-2">
            <div>
              <strong>工作原理：</strong>
              <ol className="ml-4 mt-1 space-y-1 list-decimal">
                <li>当前工作流执行到事件节点时，发布指定类型的事件</li>
                <li>系统自动查找所有配置了<strong>事件触发器</strong>的工作流</li>
                <li>如果其他工作流监听该事件类型，会自动触发执行</li>
                <li>事件数据会作为参数传递给被触发的工作流</li>
              </ol>
            </div>
            <div className="mt-2 p-2 bg-orange-100 dark:bg-orange-900 rounded">
              <strong>如何让其他工作流响应事件：</strong>
              <div className="mt-1 text-xs">
                在工作流管理页面，为其他工作流配置<strong>事件触发器</strong>，设置监听的事件类型（如：user_action），当事件发布时，该工作流会自动执行。
              </div>
            </div>
          </div>
        </div>
      </div>
      
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          工作模式
        </label>
        <select
          value={node.data.config?.mode || 'publish'}
          onChange={(e) => onConfigChange({ ...(node.data.config || {}), mode: e.target.value })}
          className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="publish">发布事件（推荐）</option>
          <option value="wait">等待事件（待完善）</option>
        </select>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          事件类型（必填）
        </label>
        <div className="relative">
          <input
            type="text"
            list={`event-types-${node.id}`}
            value={node.data.config?.eventType || node.data.config?.event_type || ''}
            onChange={(e) => onConfigChange({ ...(node.data.config || {}), eventType: e.target.value, event_type: e.target.value })}
            placeholder="user.created, order.paid, workflow.completed"
            className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <datalist id={`event-types-${node.id}`}>
            {availableEventTypes.map((type, idx) => (
              <option key={idx} value={type} />
            ))}
          </datalist>
        </div>
        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
          事件类型标识符，用于触发其他工作流
          {availableEventTypes.length > 0 && (
            <span className="ml-2 text-blue-600 dark:text-blue-400">
              （已注册 {availableEventTypes.length} 个事件类型，输入时会有提示）
            </span>
          )}
        </p>
      </div>

      {node.data.config?.mode === 'publish' && (
        <div className="space-y-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
          <div>
            <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
              事件数据 (JSON)
            </label>
            <textarea
              value={node.data.config?.eventData ? (typeof node.data.config.eventData === 'string' ? node.data.config.eventData : JSON.stringify(node.data.config.eventData, null, 2)) : (node.data.config?.event_data || '{}')}
              onChange={(e) => {
                try {
                  const eventData = JSON.parse(e.target.value)
                  onConfigChange({ ...(node.data.config || {}), eventData, event_data: JSON.stringify(eventData) })
                } catch {
                  // Keep as string if not valid JSON
                  onConfigChange({ ...(node.data.config || {}), eventData: e.target.value, event_data: e.target.value })
                }
              }}
              className="w-full px-2 py-1.5 text-xs font-mono border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700"
              rows={4}
              placeholder='{"userId": 123, "action": "login"}'
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              事件数据会与输入参数合并。支持模板变量：{'{{'}variable{'}}'} 或 {'{{'}context.key{'}}'}
            </p>
          </div>
          <div className="p-2 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded text-xs">
            <div className="font-semibold text-blue-800 dark:text-blue-200 mb-1">使用场景：</div>
            <div className="text-blue-700 dark:text-blue-300 space-y-1">
              <div>• 工作流完成后通知外部系统</div>
              <div>• 触发其他工作流执行</div>
              <div>• 记录重要业务事件</div>
              <div>• 与消息队列、Webhook 等集成</div>
            </div>
          </div>
        </div>
      )}

      {node.data.config?.mode === 'wait' && (
        <div className="p-3 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg">
          <div className="text-xs text-yellow-800 dark:text-yellow-200">
            <strong>注意：</strong>等待事件功能正在开发中。当前会立即继续执行。
          </div>
        </div>
      )}
    </div>
  )
}

/**
 * 任务节点配置
 */
const TaskNodeConfig: React.FC<{
  node: WorkflowNode
  onConfigChange: (config: any) => void
}> = ({ node, onConfigChange }) => {
  const taskType = node.data.config?.task_type || node.data.config?.type || 'log'
  
  return (
    <div className="space-y-4">
      <div className="p-3 bg-purple-50 dark:bg-purple-900/20 border border-purple-200 dark:border-purple-800 rounded-lg">
        <div className="text-xs text-purple-800 dark:text-purple-200">
          <strong>任务节点：</strong>执行各种操作（HTTP请求、数据转换、变量设置等），输出数据存储在上下文中，可在后续节点通过 <code className="bg-purple-100 dark:bg-purple-900 px-1 rounded">context.xxx</code> 访问。
        </div>
      </div>
      
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          任务类型
        </label>
        <select
          value={taskType}
          onChange={(e) => onConfigChange({ ...(node.data.config || {}), task_type: e.target.value, type: e.target.value })}
          className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="http">HTTP 请求</option>
          <option value="transform">数据转换</option>
          <option value="set_variable">设置变量</option>
          <option value="delay">延迟等待</option>
          <option value="log">日志记录</option>
        </select>
      </div>

      {/* HTTP 任务配置 */}
      {taskType === 'http' && (
        <div className="space-y-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
          <div className="p-2 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded text-xs">
            <div className="font-semibold text-blue-800 dark:text-blue-200 mb-1">输出数据访问：</div>
            <div className="text-blue-700 dark:text-blue-300 space-y-1">
              <div>• <code className="bg-blue-100 dark:bg-blue-900 px-1 rounded">context.response.body</code> - 响应体（JSON 对象）</div>
              <div>• <code className="bg-blue-100 dark:bg-blue-900 px-1 rounded">context.response.body.data.user.name</code> - 嵌套字段访问</div>
              <div>• <code className="bg-blue-100 dark:bg-blue-900 px-1 rounded">context.response.statusCode</code> - HTTP 状态码</div>
              <div>• <code className="bg-blue-100 dark:bg-blue-900 px-1 rounded">context.{'{'}nodeId{'}'}.response</code> - 通过节点ID访问</div>
            </div>
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
              请求方法
            </label>
            <select
              value={node.data.config?.method || 'GET'}
              onChange={(e) => onConfigChange({ ...(node.data.config || {}), method: e.target.value })}
              className="w-full px-2 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700"
            >
              <option value="GET">GET</option>
              <option value="POST">POST</option>
              <option value="PUT">PUT</option>
              <option value="PATCH">PATCH</option>
              <option value="DELETE">DELETE</option>
            </select>
          </div>
          <Input
            label="请求 URL"
            size="sm"
            value={node.data.config?.url || ''}
            onChange={(e) => onConfigChange({ ...(node.data.config || {}), url: e.target.value })}
            placeholder="https://api.example.com/data"
            helperText={'支持模板变量：{{variable}} 或 {{context.key}}'}
          />
          <Input
            label="超时时间"
            size="sm"
            type="text"
            value={node.data.config?.timeout || '10s'}
            onChange={(e) => onConfigChange({ ...(node.data.config || {}), timeout: e.target.value })}
            placeholder="10s"
            helperText="例如：10s, 30s, 1m"
          />
          <div>
            <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
              请求头 (JSON)
            </label>
            <textarea
              value={node.data.config?.headers ? JSON.stringify(node.data.config.headers, null, 2) : '{}'}
              onChange={(e) => {
                try {
                  const headers = JSON.parse(e.target.value)
                  onConfigChange({ ...(node.data.config || {}), headers })
                } catch {
                  // Invalid JSON, ignore
                }
              }}
              className="w-full px-2 py-1.5 text-xs font-mono border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700"
              rows={3}
              placeholder='{"Content-Type": "application/json"}'
            />
          </div>
          {(node.data.config?.method === 'POST' || node.data.config?.method === 'PUT' || node.data.config?.method === 'PATCH') && (
            <div>
              <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
                请求体 (JSON)
              </label>
              <textarea
                value={node.data.config?.body ? (typeof node.data.config.body === 'string' ? node.data.config.body : JSON.stringify(node.data.config.body, null, 2)) : '{}'}
                onChange={(e) => {
                  try {
                    const body = JSON.parse(e.target.value)
                    onConfigChange({ ...(node.data.config || {}), body })
                  } catch {
                    // Keep as string if not valid JSON
                    onConfigChange({ ...(node.data.config || {}), body: e.target.value })
                  }
                }}
                className="w-full px-2 py-1.5 text-xs font-mono border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700"
                rows={4}
                placeholder='{"key": "value"}'
              />
            </div>
          )}
        </div>
      )}

      {/* 数据转换任务配置 */}
      {taskType === 'transform' && (
        <div className="space-y-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
          <div>
            <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
              转换操作
            </label>
            <select
              value={node.data.config?.operation || 'copy'}
              onChange={(e) => onConfigChange({ ...(node.data.config || {}), operation: e.target.value })}
              className="w-full px-2 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700"
            >
              <option value="copy">复制所有字段</option>
              <option value="select">选择字段</option>
              <option value="map">字段映射</option>
              <option value="merge">合并数据</option>
              <option value="filter">过滤数据</option>
            </select>
          </div>
        </div>
      )}

      {/* 设置变量任务配置 */}
      {taskType === 'set_variable' && (
        <div className="space-y-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
          <div>
            <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
              变量 (JSON)
            </label>
            <textarea
              value={node.data.config?.variables ? JSON.stringify(node.data.config.variables, null, 2) : '{}'}
              onChange={(e) => {
                try {
                  const variables = JSON.parse(e.target.value)
                  onConfigChange({ ...(node.data.config || {}), variables })
                } catch {
                  // Invalid JSON, ignore
                }
              }}
              className="w-full px-2 py-1.5 text-xs font-mono border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700"
              rows={4}
              placeholder='{"key": "value"}'
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              支持模板变量：{'{'}{'{'}variable{'}'}{'}'} 或 {'{'}{'{'}context.key{'}'}{'}'}
            </p>
          </div>
        </div>
      )}

      {/* 延迟任务配置 */}
      {taskType === 'delay' && (
        <div className="space-y-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
          <Input
            label="延迟时间"
            size="sm"
            type="text"
            value={node.data.config?.duration || '1s'}
            onChange={(e) => onConfigChange({ ...(node.data.config || {}), duration: e.target.value })}
            placeholder="1s"
            helperText="例如：1s, 5s, 1m, 30s"
          />
        </div>
      )}

      {/* 日志任务配置 */}
      {taskType === 'log' && (
        <div className="space-y-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
          <div>
            <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
              日志级别
            </label>
            <select
              value={node.data.config?.level || 'info'}
              onChange={(e) => onConfigChange({ ...(node.data.config || {}), level: e.target.value })}
              className="w-full px-2 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700"
            >
              <option value="debug">Debug</option>
              <option value="info">Info</option>
              <option value="warning">Warning</option>
              <option value="error">Error</option>
            </select>
          </div>
          <Input
            label="日志消息"
            size="sm"
            value={node.data.config?.message || ''}
            onChange={(e) => onConfigChange({ ...(node.data.config || {}), message: e.target.value })}
            placeholder="日志内容"
            helperText={'支持模板变量：{{variable}} 或 {{context.key}}'}
          />
        </div>
      )}
    </div>
  )
}

/**
 * 条件分支节点配置
 */
const GatewayNodeConfig: React.FC<{
  node: WorkflowNode
  onConfigChange: (config: any) => void
}> = ({ node, onConfigChange }) => {
  return (
    <div className="space-y-4">
      <div className="p-3 bg-purple-50 dark:bg-purple-900/20 border border-purple-200 dark:border-purple-800 rounded-lg">
        <div className="text-xs text-purple-800 dark:text-purple-200">
          <strong>条件分支：</strong>根据条件判断结果选择不同的执行路径
        </div>
      </div>
      
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          条件表达式
        </label>
        <textarea
          value={node.data.config?.condition || 'context.value > 0'}
          onChange={(e) => onConfigChange({ ...(node.data.config || {}), condition: e.target.value })}
          placeholder="context.value > 0"
          className="w-full px-3 py-2 text-sm font-mono border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
          rows={3}
        />
      </div>
    </div>
  )
}

/**
 * 脚本节点配置
 */
const ScriptNodeConfig: React.FC<{
  node: WorkflowNode
  onConfigChange: (config: any) => void
  isCodeEditorFullscreen: boolean
  setIsCodeEditorFullscreen: (value: boolean) => void
}> = ({ node, onConfigChange, isCodeEditorFullscreen, setIsCodeEditorFullscreen }) => {
  return (
    <div className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          脚本语言
        </label>
        <select
          value={node.data.config?.language || 'go'}
          onChange={(e) => onConfigChange({ ...(node.data.config || {}), language: e.target.value })}
          className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
          disabled
        >
          <option value="go">Go</option>
        </select>
        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
          目前仅支持 Go 语言脚本
        </p>
      </div>
      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            脚本代码
          </label>
          <Button
            variant="ghost"
            size="xs"
            onClick={() => setIsCodeEditorFullscreen(!isCodeEditorFullscreen)}
          >
            {isCodeEditorFullscreen ? (
              <Minimize2 className="w-4 h-4" />
            ) : (
              <Maximize2 className="w-4 h-4" />
            )}
          </Button>
        </div>
        <div className="mb-2 p-2 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded text-xs text-blue-800 dark:text-blue-200">
          <strong>提示：</strong>脚本必须定义一个 <code className="px-1 py-0.5 bg-blue-100 dark:bg-blue-900/40 rounded">Run</code> 函数，签名：
          <code className="block mt-1 px-2 py-1 bg-blue-100 dark:bg-blue-900/40 rounded font-mono">
            func Run(inputs map[string]interface{}) (map[string]interface{}, error)
          </code>
        </div>
        <div className="relative border border-gray-300 dark:border-gray-600 rounded-md overflow-hidden bg-gray-900">
          <Suspense fallback={
            <div className="h-[400px] flex items-center justify-center bg-gray-900">
              <div className="text-center">
                <div className="animate-spin rounded-full h-8 w-8 border-4 border-gray-700 border-t-blue-500 mx-auto mb-3"></div>
                <p className="text-sm text-gray-400">加载代码编辑器...</p>
              </div>
            </div>
          }>
            <MonacoEditor
              height="400px"
              language={node.data.config?.language || 'go'}
              value={node.data.config?.code || ''}
              onChange={(value) => onConfigChange({ ...(node.data.config || {}), code: value || '' })}
              theme="vs-dark"
              options={{
                minimap: { enabled: false },
                scrollBeyondLastLine: false,
                fontSize: 14,
                lineNumbers: 'on',
                wordWrap: 'on',
                automaticLayout: true,
                tabSize: 2,
                formatOnPaste: true,
                formatOnType: true,
                suggestOnTriggerCharacters: true,
                quickSuggestions: true,
              }}
            />
          </Suspense>
        </div>
      </div>
    </div>
  )
}

/**
 * AI对话节点配置
 */
const AIChatNodeConfig: React.FC<{
  node: WorkflowNode
  onConfigChange: (config: any) => void
}> = ({ node, onConfigChange }) => {
  return (
    <div className="space-y-4">
      <div className="p-3 bg-indigo-50 dark:bg-indigo-900/20 border border-indigo-200 dark:border-indigo-800 rounded-lg">
        <div className="text-xs text-indigo-800 dark:text-indigo-200">
          <strong>AI对话节点：</strong>与AI模型进行对话交互
        </div>
      </div>
      
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          AI提供商
        </label>
        <select
          value={node.data.config?.provider || 'openai'}
          onChange={(e) => onConfigChange({ ...(node.data.config || {}), provider: e.target.value })}
          className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="openai">OpenAI</option>
          <option value="anthropic">Anthropic</option>
          <option value="local">本地模型</option>
        </select>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          模型
        </label>
        <Input
          size="sm"
          value={node.data.config?.model || 'gpt-4'}
          onChange={(e) => onConfigChange({ ...(node.data.config || {}), model: e.target.value })}
          placeholder="gpt-4"
        />
      </div>
    </div>
  )
}

/**
 * 等待节点配置
 */
const WaitNodeConfig: React.FC<{
  node: WorkflowNode
  onConfigChange: (config: any) => void
}> = ({ node, onConfigChange }) => {
  return (
    <div className="space-y-4">
      <Input
        label="等待时长 (毫秒)"
        size="sm"
        type="number"
        min="0"
        value={node.data.config?.duration?.toString() || '5000'}
        onChange={(e) => onConfigChange({ ...(node.data.config || {}), duration: parseInt(e.target.value) || 5000 })}
      />
      <Input
        label="等待事件"
        size="sm"
        value={node.data.config?.untilEvent || ''}
        onChange={(e) => onConfigChange({ ...(node.data.config || {}), untilEvent: e.target.value })}
        placeholder="可选：等待特定事件"
        helperText="留空则按时长等待"
      />
    </div>
  )
}

/**
 * 工作流插件节点配置
 */
const WorkflowPluginNodeConfig: React.FC<{
  node: WorkflowNode
  onConfigChange: (config: any) => void
  installedPlugins: any[]
  loadingPlugins: boolean
}> = ({ node, onConfigChange, installedPlugins, loadingPlugins }) => {
  if (loadingPlugins) {
    return (
      <div className="p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
        <div className="flex items-center gap-2 text-blue-700 dark:text-blue-400">
          <div className="animate-spin rounded-full h-4 w-4 border-2 border-blue-500 border-t-transparent"></div>
          <span className="text-sm font-medium">加载插件信息中...</span>
        </div>
      </div>
    )
  }

  const plugin = installedPlugins.find(p => p.pluginId === node.data.pluginId)

  if (!plugin) {
    return (
      <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
        <div className="flex items-center gap-2 text-red-700 dark:text-red-400">
          <AlertCircle className="w-4 h-4" />
          <span className="text-sm font-medium">工作流插件未找到</span>
        </div>
        <p className="text-xs text-red-600 dark:text-red-500 mt-1">
          该节点关联的工作流插件可能已被卸载或不存在
        </p>
        <p className="text-xs text-red-600 dark:text-red-500 mt-2">
          插件ID: {node.data.pluginId}
        </p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* 工作流插件基本信息 */}
      <div className="p-4 bg-gradient-to-r from-purple-50 to-blue-50 dark:from-purple-900/20 dark:to-blue-900/20 border border-purple-200 dark:border-purple-800 rounded-lg">
        <div className="flex items-center gap-3 mb-2">
          <div 
            className="p-2 rounded-lg bg-gradient-to-r from-purple-500 to-blue-500 text-white"
            style={{ 
              background: plugin.plugin?.color 
                ? `linear-gradient(to right, ${plugin.plugin.color}, ${plugin.plugin.color}CC)` 
                : undefined
            }}
          >
            <Package className="w-4 h-4" />
          </div>
          <div>
            <h4 className="text-sm font-semibold text-gray-900 dark:text-white">
              {plugin.plugin?.displayName || plugin.plugin?.name}
            </h4>
            <p className="text-xs text-gray-500 dark:text-gray-400">
              工作流插件 • v{plugin.version} • {plugin.plugin?.category}
            </p>
          </div>
        </div>
        {plugin.plugin?.description && (
          <p className="text-xs text-gray-600 dark:text-gray-400">
            {plugin.plugin.description}
          </p>
        )}
        <div className="mt-2 p-2 bg-purple-100 dark:bg-purple-900/30 border border-purple-200 dark:border-purple-700 rounded text-xs text-purple-800 dark:text-purple-200">
          <strong>说明：</strong>工作流插件节点会执行一个完整的子工作流，输入参数会传递给子工作流的开始节点，输出参数来自子工作流的结束节点。
        </div>
      </div>
      
      {/* 输入参数配置 */}
      {plugin.plugin?.inputSchema?.parameters?.length > 0 && (
        <div>
          <h5 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3 flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-gradient-to-r from-emerald-400 to-emerald-500"></div>
            输入参数配置
          </h5>
          <div className="space-y-3">
            {plugin.plugin.inputSchema.parameters.map((param: any, index: number) => (
              <div key={index} className="space-y-2">
                <label className="block text-xs font-medium text-gray-600 dark:text-gray-400">
                  {param.name}
                  {param.required && <span className="text-red-500 ml-1">*</span>}
                </label>
                {param.type === 'boolean' ? (
                  <div className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={node.data.config?.parameters?.[param.name] || false}
                      onChange={(e) => onConfigChange({
                        ...(node.data.config || {}),
                        parameters: {
                          ...(node.data.config?.parameters || {}),
                          [param.name]: e.target.checked
                        }
                      })}
                      className="rounded border-gray-300 dark:border-gray-600 text-purple-600 focus:ring-purple-500"
                    />
                    <span className="text-xs text-gray-500 dark:text-gray-400">
                      {param.description}
                    </span>
                  </div>
                ) : param.type === 'number' ? (
                  <input
                    type="number"
                    value={node.data.config?.parameters?.[param.name] || param.default || ''}
                    onChange={(e) => onConfigChange({
                      ...(node.data.config || {}),
                      parameters: {
                        ...(node.data.config?.parameters || {}),
                        [param.name]: parseFloat(e.target.value) || 0
                      }
                    })}
                    placeholder={param.example || param.default}
                    className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                  />
                ) : (
                  <input
                    type="text"
                    value={node.data.config?.parameters?.[param.name] || param.default || ''}
                    onChange={(e) => onConfigChange({
                      ...(node.data.config || {}),
                      parameters: {
                        ...(node.data.config?.parameters || {}),
                        [param.name]: e.target.value
                      }
                    })}
                    placeholder={param.example || param.default}
                    className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                  />
                )}
                {param.description && (
                  <p className="text-xs text-gray-500 dark:text-gray-400">
                    {param.description}
                  </p>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
      
      {/* 输出参数说明 */}
      {plugin.plugin?.outputSchema?.parameters?.length > 0 && (
        <div>
          <h5 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3 flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-gradient-to-r from-blue-400 to-blue-500"></div>
            输出参数说明
          </h5>
          <div className="space-y-2">
            {plugin.plugin.outputSchema.parameters.map((param: any, index: number) => (
              <div key={index} className="flex items-center gap-2 text-xs">
                <span className="font-medium text-gray-600 dark:text-gray-400">
                  {param.name}
                </span>
                <span className="px-2 py-1 bg-gradient-to-r from-blue-100 to-purple-100 dark:from-blue-900/30 dark:to-purple-900/30 text-blue-700 dark:text-blue-400 rounded border border-blue-200 dark:border-blue-700">
                  {param.type}
                </span>
                {param.description && (
                  <span className="text-gray-500 dark:text-gray-400">
                    - {param.description}
                  </span>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
