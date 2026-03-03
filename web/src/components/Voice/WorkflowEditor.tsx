import React, { useState, useRef, useCallback, useEffect, Suspense } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { 
  Play,
  Square,
  FileText,
  GitBranch,
  Zap,
  Clock,
  Timer,
  Code,
  Settings,
  Package,
  Bot,
  Trash2, 
  ChevronLeft,
  ChevronRight,
  CheckCircle,
  Minimize2
} from 'lucide-react'
import { cn } from '@/utils/cn'
import Modal from '@/components/UI/Modal'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import { showAlert } from '@/utils/notification'
import { workflowService } from '@/api/workflow'
import { workflowPluginService, WorkflowPluginCategory } from '@/api/workflowPlugin'
import { useI18nStore } from '@/stores/i18nStore'
import { getNodeTypes } from './utils'
import type { WorkflowNode, WorkflowConnection, Workflow } from './types'
import { 
  useNodeOperations, 
  useCanvasControl, 
  useConnectionOperations,
  useWorkflowValidation 
} from './hooks'
import { NodeDrawer, CanvasToolbar, HelpModal, RunParametersModal, NodeConfigPanel } from './components'

// 懒加载Monaco Editor
const MonacoEditor = React.lazy(() => import('@monaco-editor/react'))

// 图标映射
const ICON_MAP: Record<string, React.ComponentType<any>> = {
  Play,
  Square,
  FileText,
  GitBranch,
  Zap,
  Clock,
  Timer,
  Code,
  Settings,
  Package,
  Bot
}

// 获取图标组件的辅助函数
const getIconComponent = (iconName: string) => {
  const Icon = ICON_MAP[iconName]
  return Icon ? <Icon className="w-5 h-5" /> : null
}

interface WorkflowEditorProps {
  workflow?: Workflow
  onSave?: (workflow: Workflow) => void
  onRun?: (workflow: Workflow, parameters?: Record<string, any>) => void
  onStop?: (instanceId: number) => Promise<void>
  isRunning?: boolean
  currentInstanceId?: number | null
  workflowId?: number // 工作流ID，用于节点测试
  className?: string
}

const WorkflowEditor: React.FC<WorkflowEditorProps> = ({
  workflow,
  onSave,
  onRun,
  onStop,
  isRunning: propsIsRunning,
  currentInstanceId,
  workflowId,
  className = ''
}) => {
  const { t } = useI18nStore()
  const NODE_TYPES = getNodeTypes(t)
  const canvasRef = useRef<HTMLDivElement>(null)
  const [nodes, setNodes] = useState<WorkflowNode[]>(workflow?.nodes || [])
  const [connections, setConnections] = useState<WorkflowConnection[]>(workflow?.connections || [])
  const [selectedNode, setSelectedNode] = useState<string | null>(null)
  const [draggedNode, setDraggedNode] = useState<string | null>(null)
  const [isConnecting, setIsConnecting] = useState(false)
  const [connectionStart, setConnectionStart] = useState<{ nodeId: string; handle: string } | null>(null)
  const [canvasOffset, setCanvasOffset] = useState({ x: 100000, y: 100000 })
  const [isDragging, setIsDragging] = useState(false)
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 })
  const [selectedConnection, setSelectedConnection] = useState<string | null>(null)
  const [configuringNode, setConfiguringNode] = useState<string | null>(null)
  const [canvasScale, setCanvasScale] = useState(1)
  const [showHelpModal, setShowHelpModal] = useState(false)
  const [showNodeDrawer, setShowNodeDrawer] = useState(false)
  const [installedPlugins, setInstalledPlugins] = useState<any[]>([])
  const [loadingPlugins, setLoadingPlugins] = useState(false)
  const [nodeSearchQuery, setNodeSearchQuery] = useState('')

  // 加载已安装的插件
  const loadInstalledPlugins = useCallback(async () => {
    setLoadingPlugins(true)
    try {
      const response = await workflowPluginService.listInstalledWorkflowPlugins()
      if (response.data) {
        setInstalledPlugins(response.data)
      }
    } catch (error) {
      console.error('加载已安装插件失败:', error)
    } finally {
      setLoadingPlugins(false)
    }
  }, [])

  // 组件挂载时加载插件
  useEffect(() => {
    loadInstalledPlugins()
  }, [loadInstalledPlugins])

  // 当工作流数据改变时，同步节点和连接
  useEffect(() => {
    if (workflow?.nodes) {
      setNodes(workflow.nodes)
    }
    if (workflow?.connections) {
      setConnections(workflow.connections)
    }
  }, [workflow?.id, workflow?.nodes, workflow?.connections])

  const [showPublishModal, setShowPublishModal] = useState(false)
  const [showRunParamsModal, setShowRunParamsModal] = useState(false)
  const [runParameters, setRunParameters] = useState<Record<string, string>>({})
  const [availableEventTypes, setAvailableEventTypes] = useState<string[]>([])
  const [showNodeTestModal, setShowNodeTestModal] = useState(false)
  const [testingNode, setTestingNode] = useState<string | null>(null)
  const [nodeTestParameters, setNodeTestParameters] = useState<Record<string, string>>({})
  const [nodeTestResult, setNodeTestResult] = useState<any>(null)
  const [isTestingNode, setIsTestingNode] = useState(false)
  const [isCodeEditorFullscreen, setIsCodeEditorFullscreen] = useState(false)


  // 使用 hooks 管理节点操作
  const {
    addNode,
    addPluginNode,
    deleteNode,
    updateNodePosition,
    deleteConnection,
    isOutputConnected
  } = useNodeOperations({
    nodes,
    connections,
    setNodes,
    setConnections,
    selectedNode,
    setSelectedNode,
    selectedConnection,
    setSelectedConnection,
    NODE_TYPES
  })

  // 使用 hooks 管理画布控制
  const {
    resetCanvasView,
    zoomIn,
    zoomOut,
    centerOnNodes
  } = useCanvasControl({
    nodes,
    canvasRef,
    canvasOffset,
    canvasScale,
    setCanvasOffset,
    setCanvasScale
  })

  // 使用 hooks 管理连接操作
  const {
    startConnection,
    completeConnection
  } = useConnectionOperations({
    nodes,
    connections,
    setConnections,
    isConnecting,
    connectionStart,
    setIsConnecting,
    setConnectionStart
  })

  // 使用 hooks 验证工作流
  const { validateWorkflow } = useWorkflowValidation({ nodes })

  // 点击画布取消选择

  // 画布拖拽处理
  const handleCanvasMouseDown = useCallback((e: React.MouseEvent) => {
    // 只有在点击画布背景时才开始拖拽（不是节点）
    const target = e.target as HTMLElement
    // 检查是否点击在节点上
    const isNode = target.closest('.absolute.w-\\[180px\\]')
    // 检查是否点击在连接线上
    const isConnection = target.tagName === 'path' || target.tagName === 'line'
    
    if (!isNode && !isConnection) {
      setIsDragging(true)
      setDragStart({ x: e.clientX - canvasOffset.x, y: e.clientY - canvasOffset.y })
      // 只在需要时调用 preventDefault
      if (e.cancelable) {
        e.preventDefault()
      }
    }
  }, [canvasOffset])

  const handleCanvasMouseMove = useCallback((e: React.MouseEvent) => {
    if (isDragging) {
      const newOffset = {
        x: e.clientX - dragStart.x,
        y: e.clientY - dragStart.y
      }
      setCanvasOffset(newOffset)
    }
  }, [isDragging, dragStart])

  const handleCanvasMouseUp = useCallback(() => {
    setIsDragging(false)
  }, [])

  // 节点拖拽处理
  const [dragOffset, setDragOffset] = useState({ x: 0, y: 0 })
  
  const handleNodeMouseDown = useCallback((e: React.MouseEvent, nodeId: string) => {
    e.stopPropagation()
    // 只在需要时调用 preventDefault
    if (e.cancelable) {
      e.preventDefault()
    }
    
    const rect = canvasRef.current?.getBoundingClientRect()
    if (rect) {
      const node = nodes.find(n => n.id === nodeId)
      if (node) {
        // 计算鼠标相对于节点的偏移量
        // 注意：这里要使用鼠标在画布中的实际位置
        const mouseX = e.clientX - rect.left
        const mouseY = e.clientY - rect.top
        
        // 节点在屏幕中的实际位置（考虑画布偏移 -100000px 和缩放）
        // 节点层的 transform: translate(canvasOffset.x, canvasOffset.y) scale(canvasScale)
        // 节点位置: left: node.position.x, top: node.position.y (相对于节点层，节点层从 -100000px 开始)
        // 节点在屏幕中的位置: -100000 + canvasOffset.x + node.position.x * canvasScale
        const nodeScreenX = -100000 + canvasOffset.x + node.position.x * canvasScale
        const nodeScreenY = -100000 + canvasOffset.y + node.position.y * canvasScale
        
        // 计算鼠标相对于节点的偏移量（在画布坐标系中）
        const offsetX = (mouseX - nodeScreenX) / canvasScale
        const offsetY = (mouseY - nodeScreenY) / canvasScale
        
        setDragOffset({ x: offsetX, y: offsetY })
        setDraggedNode(nodeId)
        setSelectedNode(nodeId)
      }
    }
  }, [nodes, canvasOffset, canvasScale])


  // 保存工作流
  const handleSave = useCallback(async () => {
    if (onSave) {
      const savedWorkflow: Workflow = {
        id: workflow?.id || `workflow-${Date.now()}`,
        name: workflow?.name || '未命名工作流',
        description: workflow?.description || '',
        nodes,
        connections,
        createdAt: workflow?.createdAt || new Date().toISOString(),
        updatedAt: new Date().toISOString()
      }
      try {
        await onSave(savedWorkflow)
      } catch (error) {
        console.error('Failed to save workflow:', error)
      }
    } else {
      console.warn('onSave callback is not provided')
    }
  }, [workflow, nodes, connections, onSave])

  // 运行工作流
  const handleRun = useCallback(() => {
    // 找到开始节点，获取其输入参数
    const startNode = nodes.find(n => n.type === 'start')
    if (!startNode) {
      console.warn('工作流中没有开始节点')
      return
    }

    // 如果有输入参数，显示参数输入对话框
    if (startNode.inputs && startNode.inputs.length > 0) {
      // 初始化参数值
      const initialParams: Record<string, string> = {}
      startNode.inputs.forEach(input => {
        if (input && input.trim()) {
          initialParams[input] = ''
        }
      })
      setRunParameters(initialParams)
      setShowRunParamsModal(true)
    } else {
      // 没有输入参数，直接运行
      executeRun({})
    }
  }, [nodes])

  // 执行运行
  const executeRun = useCallback(async (parameters: Record<string, any>) => {
    if (onRun) {
      setShowRunParamsModal(false)
      
      const currentWorkflow: Workflow = {
        id: workflow?.id || `workflow-${Date.now()}`,
        name: workflow?.name || '未命名工作流',
        description: workflow?.description || '',
        nodes,
        connections,
        createdAt: workflow?.createdAt || new Date().toISOString(),
        updatedAt: new Date().toISOString()
      }
      
      try {
        // 将参数传递给 onRun 回调
        await onRun(currentWorkflow, parameters)
      } catch (error) {
        console.error('Error running workflow:', error)
      }
    }
  }, [workflow, nodes, connections, onRun])

  // 停止工作流执行
  const handleStop = useCallback(async () => {
    if (!currentInstanceId || !onStop) {
      console.warn('No running instance to stop or onStop callback not provided')
      return
    }

    try {
      console.log('Stopping workflow instance:', currentInstanceId)
      await onStop(currentInstanceId)
      console.log('Workflow stopped successfully')
    } catch (error) {
      console.error('Error stopping workflow:', error)
    }
  }, [currentInstanceId, onStop])

  // 渲染连接线 - 现代化贝塞尔曲线设计
  const renderConnections = () => {
    return connections.map(connection => {
      const sourceNode = nodes.find(n => n.id === connection.source)
      const targetNode = nodes.find(n => n.id === connection.target)
      
      if (!sourceNode || !targetNode) return null

      // 节点卡片宽度 260px
      const nodeWidth = 260
      const baseConnectionY = 45 // 基础连接点Y位置（从顶部开始）
      
      // 计算源节点连接点位置
      const sourceOutputIndex = connection.sourceHandle ? 
        (sourceNode.outputs?.findIndex(o => o === connection.sourceHandle) ?? 0) : 0
      const sourceX = sourceNode.position.x + nodeWidth // 节点右边缘
      // 根据输出参数数量均匀分布连接点
      const totalSourceOutputs = sourceNode.outputs?.length || 1
      const sourceSpacing = totalSourceOutputs > 1 ? 60 / (totalSourceOutputs - 1) : 0
      const sourceY = sourceNode.position.y + baseConnectionY + (sourceOutputIndex * sourceSpacing)
      
      // 计算目标节点连接点位置
      const targetInputIndex = connection.targetHandle ? 
        (targetNode.inputs?.findIndex(i => i === connection.targetHandle) ?? 0) : 0
      const targetX = targetNode.position.x // 节点左边缘
      // 根据输入参数数量均匀分布连接点
      const totalTargetInputs = targetNode.inputs?.length || 1
      const targetSpacing = totalTargetInputs > 1 ? 60 / (totalTargetInputs - 1) : 0
      const targetY = targetNode.position.y + baseConnectionY + (targetInputIndex * targetSpacing)

      // 计算控制点，创建平滑的贝塞尔曲线
      const dx = targetX - sourceX
      const controlPointOffset = Math.min(Math.abs(dx) * 0.5, 180) // 控制点偏移，最大180px
      const cp1x = sourceX + controlPointOffset
      const cp1y = sourceY
      const cp2x = targetX - controlPointOffset
      const cp2y = targetY

      const isSelected = selectedConnection === connection.id
      
      // 根据连接类型确定颜色和样式
      const getConnectionStyle = () => {
        switch (connection.type) {
          case 'true':
            return {
              color: '#059669', // 绿色 - 真分支
              gradient: 'url(#greenGradient)',
              dashArray: 'none'
            }
          case 'false':
            return {
              color: '#dc2626', // 红色 - 假分支
              gradient: 'url(#redGradient)',
              dashArray: 'none'
            }
          case 'error':
            return {
              color: '#f59e0b', // 橙色 - 错误分支
              gradient: 'url(#orangeGradient)',
              dashArray: '8,4'
            }
          case 'branch':
            return {
              color: '#8b5cf6', // 紫色 - 并行分支
              gradient: 'url(#purpleGradient)',
              dashArray: 'none'
            }
          default:
            return {
              color: '#3b82f6', // 蓝色 - 默认
              gradient: 'url(#blueGradient)',
              dashArray: 'none'
            }
        }
      }

      const connectionStyle = getConnectionStyle()

      return (
        <g key={connection.id}>
          {/* 连接线光晕效果 */}
          <motion.path
            d={`M ${sourceX} ${sourceY} C ${cp1x} ${cp1y}, ${cp2x} ${cp2y}, ${targetX} ${targetY}`}
            stroke={connectionStyle.color}
            strokeWidth={isSelected ? "8" : "6"}
            fill="none"
            className="pointer-events-none opacity-20"
            strokeDasharray={connectionStyle.dashArray}
            initial={{ pathLength: 0, opacity: 0 }}
            animate={{ pathLength: 1, opacity: 0.2 }}
            transition={{ duration: 0.8, ease: "easeInOut" }}
          />
          
          {/* 可点击的连接线背景（更粗，透明） */}
          <path
            d={`M ${sourceX} ${sourceY} C ${cp1x} ${cp1y}, ${cp2x} ${cp2y}, ${targetX} ${targetY}`}
            stroke="transparent"
            strokeWidth="24"
            fill="none"
            className="cursor-pointer"
            style={{ pointerEvents: 'all' }}
            onClick={(e) => {
              e.stopPropagation()
              setSelectedConnection(connection.id)
            }}
          />
          
          {/* 主连接线 - 渐变贝塞尔曲线 */}
          <motion.path
            d={`M ${sourceX} ${sourceY} C ${cp1x} ${cp1y}, ${cp2x} ${cp2y}, ${targetX} ${targetY}`}
            stroke={isSelected ? "#ef4444" : connectionStyle.gradient}
            strokeWidth={isSelected ? "4" : "3"}
            fill="none"
            markerEnd={`url(#arrowhead-${connection.type || 'default'})`}
            className="pointer-events-none"
            strokeDasharray={connectionStyle.dashArray}
            style={{ 
              filter: isSelected 
                ? 'drop-shadow(0 0 8px rgba(239, 68, 68, 0.6))' 
                : `drop-shadow(0 2px 4px ${connectionStyle.color}40)`
            }}
            initial={{ pathLength: 0 }}
            animate={{ pathLength: 1 }}
            transition={{ duration: 0.8, ease: "easeInOut" }}
            whileHover={{ strokeWidth: isSelected ? 5 : 4 }}
          />
          
          {/* 数据流动画效果 */}
          <motion.circle
            r="3"
            fill={connectionStyle.color}
            className="opacity-60"
            initial={{ opacity: 0 }}
            animate={{ opacity: [0, 0.8, 0] }}
            transition={{ 
              duration: 2, 
              repeat: Infinity, 
              ease: "easeInOut",
              delay: Math.random() * 2 
            }}
          >
            <animateMotion
              dur="2s"
              repeatCount="indefinite"
              path={`M ${sourceX} ${sourceY} C ${cp1x} ${cp1y}, ${cp2x} ${cp2y}, ${targetX} ${targetY}`}
            />
          </motion.circle>
        </g>
      )
    })
  }

  const validation = validateWorkflow()


  // 全局鼠标事件监听 - 处理节点拖拽和画布拖拽
  useEffect(() => {
    const handleGlobalMouseMove = (e: MouseEvent) => {
      if (draggedNode) {
        const rect = canvasRef.current?.getBoundingClientRect()
        if (rect) {
          // 计算鼠标在画布中的位置
          const mouseX = e.clientX - rect.left
          const mouseY = e.clientY - rect.top
          
          // 计算新的节点位置，考虑画布偏移 -100000px、缩放和拖拽偏移
          // 节点在屏幕中的位置: -100000 + canvasOffset.x + node.position.x * canvasScale
          // 所以: node.position.x = (mouseX - (-100000 + canvasOffset.x)) / canvasScale - dragOffset.x
          const x = (mouseX + 100000 - canvasOffset.x) / canvasScale - dragOffset.x
          const y = (mouseY + 100000 - canvasOffset.y) / canvasScale - dragOffset.y
          
          // 无限画布，不限制节点位置
          updateNodePosition(draggedNode, { x, y })
        }
      } else if (isDragging) {
        // 画布拖拽
        const newOffset = {
          x: e.clientX - dragStart.x,
          y: e.clientY - dragStart.y
        }
        setCanvasOffset(newOffset)
      }
    }

    const handleGlobalMouseUp = () => {
      setDraggedNode(null)
      setDragOffset({ x: 0, y: 0 })
      setIsDragging(false)
    }

    if (draggedNode || isDragging) {
      document.addEventListener('mousemove', handleGlobalMouseMove)
      document.addEventListener('mouseup', handleGlobalMouseUp)
    }

    return () => {
      document.removeEventListener('mousemove', handleGlobalMouseMove)
      document.removeEventListener('mouseup', handleGlobalMouseUp)
    }
  }, [draggedNode, isDragging, canvasOffset, canvasScale, dragOffset, dragStart, updateNodePosition])

  // 加载可用的事件类型
  useEffect(() => {
    const loadEventTypes = async () => {
      try {
        const response = await workflowService.getAvailableEventTypes()
        if (response.code === 200 && response.data) {
          const types = response.data.event_types.map(et => et.type)
          setAvailableEventTypes(types)
        }
      } catch (error) {
        console.error('Failed to load event types:', error)
      }
    }
    loadEventTypes()
  }, [])

  // 画布滚轮缩放 - 使用非被动事件监听器避免 preventDefault 警告
  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return

    const handleWheel = (e: WheelEvent) => {
      e.preventDefault()
      const delta = e.deltaY > 0 ? 0.9 : 1.1
      setCanvasScale(prev => Math.max(0.3, Math.min(3, prev * delta)))
    }

    canvas.addEventListener('wheel', handleWheel, { passive: false })
    return () => {
      canvas.removeEventListener('wheel', handleWheel)
    }
  }, [])

  return (
    <div className={cn('flex flex-col h-full bg-gray-50 dark:bg-gray-900', className)}>
      {/* 工具栏 */}
      <CanvasToolbar
        validation={validation}
        onAddNode={() => setShowNodeDrawer(true)}
        onHelp={() => setShowHelpModal(true)}
        canvasScale={canvasScale}
        onZoomIn={zoomIn}
        onZoomOut={zoomOut}
        onResetView={resetCanvasView}
        onCenterNodes={centerOnNodes}
        selectedConnection={selectedConnection}
        onDeleteConnection={() => {
          if (selectedConnection) {
            deleteConnection(selectedConnection)
          }
        }}
        isRunning={propsIsRunning || !!currentInstanceId}
        onRun={handleRun}
        onStop={handleStop}
        onSave={handleSave}
        t={t}
      />

      <div className="flex flex-1">
        {/* 画布区域 */}
        <div className="flex-1 relative overflow-hidden">
          <div
            ref={canvasRef}
            className="relative w-full h-full"
            style={{ 
              cursor: isDragging ? 'grabbing' : 'grab'
            }}
            onMouseDown={handleCanvasMouseDown}
            onMouseMove={handleCanvasMouseMove}
            onMouseUp={handleCanvasMouseUp}
            onMouseLeave={handleCanvasMouseUp}
          >
            {/* 画布背景网格 - 无限延伸 */}
            <div 
              className="absolute pointer-events-none"
              style={{
                width: '200000px',
                height: '200000px',
                left: '-100000px',
                top: '-100000px',
                backgroundImage: `
                  linear-gradient(to right, rgba(0,0,0,0.05) 1px, transparent 1px),
                  linear-gradient(to bottom, rgba(0,0,0,0.05) 1px, transparent 1px)
                `,
                backgroundSize: '20px 20px',
                backgroundPosition: `${(canvasOffset.x % 20)}px ${(canvasOffset.y % 20)}px`,
                transform: `translate(${canvasOffset.x}px, ${canvasOffset.y}px) scale(${canvasScale})`,
                transformOrigin: '0 0',
                zIndex: 0
              }}
            />
            {/* SVG 连接线层 - 无限延伸 */}
            <svg
              className="absolute pointer-events-none"
              style={{ 
                width: '200000px',
                height: '200000px',
                left: '-100000px',
                top: '-100000px',
                zIndex: 1,
                transform: `translate(${canvasOffset.x}px, ${canvasOffset.y}px) scale(${canvasScale})`,
                transformOrigin: '0 0'
              }}
            >
              <defs>
                {/* 渐变定义 */}
                <linearGradient id="blueGradient" x1="0%" y1="0%" x2="100%" y2="0%">
                  <stop offset="0%" stopColor="#3b82f6" />
                  <stop offset="100%" stopColor="#1d4ed8" />
                </linearGradient>
                
                <linearGradient id="greenGradient" x1="0%" y1="0%" x2="100%" y2="0%">
                  <stop offset="0%" stopColor="#059669" />
                  <stop offset="100%" stopColor="#047857" />
                </linearGradient>
                
                <linearGradient id="redGradient" x1="0%" y1="0%" x2="100%" y2="0%">
                  <stop offset="0%" stopColor="#dc2626" />
                  <stop offset="100%" stopColor="#b91c1c" />
                </linearGradient>
                
                <linearGradient id="orangeGradient" x1="0%" y1="0%" x2="100%" y2="0%">
                  <stop offset="0%" stopColor="#f59e0b" />
                  <stop offset="100%" stopColor="#d97706" />
                </linearGradient>
                
                <linearGradient id="purpleGradient" x1="0%" y1="0%" x2="100%" y2="0%">
                  <stop offset="0%" stopColor="#8b5cf6" />
                  <stop offset="100%" stopColor="#7c3aed" />
                </linearGradient>

                {/* 箭头标记定义 */}
                <marker
                  id="arrowhead-default"
                  markerWidth="12"
                  markerHeight="8"
                  refX="11"
                  refY="4"
                  orient="auto"
                  markerUnits="strokeWidth"
                >
                  <polygon
                    points="0 0, 12 4, 0 8"
                    fill="url(#blueGradient)"
                    stroke="none"
                  />
                </marker>
                
                <marker
                  id="arrowhead-true"
                  markerWidth="12"
                  markerHeight="8"
                  refX="11"
                  refY="4"
                  orient="auto"
                  markerUnits="strokeWidth"
                >
                  <polygon
                    points="0 0, 12 4, 0 8"
                    fill="url(#greenGradient)"
                    stroke="none"
                  />
                </marker>
                
                <marker
                  id="arrowhead-false"
                  markerWidth="12"
                  markerHeight="8"
                  refX="11"
                  refY="4"
                  orient="auto"
                  markerUnits="strokeWidth"
                >
                  <polygon
                    points="0 0, 12 4, 0 8"
                    fill="url(#redGradient)"
                    stroke="none"
                  />
                </marker>
                
                <marker
                  id="arrowhead-error"
                  markerWidth="12"
                  markerHeight="8"
                  refX="11"
                  refY="4"
                  orient="auto"
                  markerUnits="strokeWidth"
                >
                  <polygon
                    points="0 0, 12 4, 0 8"
                    fill="url(#orangeGradient)"
                    stroke="none"
                  />
                </marker>
                
                <marker
                  id="arrowhead-branch"
                  markerWidth="12"
                  markerHeight="8"
                  refX="11"
                  refY="4"
                  orient="auto"
                  markerUnits="strokeWidth"
                >
                  <polygon
                    points="0 0, 12 4, 0 8"
                    fill="url(#purpleGradient)"
                    stroke="none"
                  />
                </marker>
              </defs>
              {renderConnections()}
            </svg>

            {/* 节点层 - 无限画布 */}
            <div
              className="absolute"
              style={{
                width: '200000px',
                height: '200000px',
                left: '-100000px',
                top: '-100000px',
                transform: `translate(${canvasOffset.x}px, ${canvasOffset.y}px) scale(${canvasScale})`,
                transformOrigin: '0 0',
                zIndex: 2
              }}
            >
              {nodes.map(node => {
                const nodeConfig = NODE_TYPES[node.type]
                if (!nodeConfig) {
                  console.warn(`Unknown node type: ${node.type}, using default config`)
                  return null
                }
                return (
                  <motion.div
                    key={node.id}
                    className={cn(
                      'absolute w-[260px] cursor-move select-none group',
                      draggedNode === node.id ? 'z-50' : 'z-10'
                    )}
                    style={{
                      left: node.position.x,
                      top: node.position.y
                    }}
                    onMouseDown={(e) => handleNodeMouseDown(e, node.id)}
                    initial={{ opacity: 0, scale: 0.95 }}
                    animate={{ 
                      opacity: 1, 
                      scale: draggedNode === node.id ? 1.02 : 1
                    }}
                    transition={{ duration: 0.15 }}
                  >
                    {/* 主卡片 - 全新现代化设计 */}
                    <div className={cn(
                      'relative overflow-hidden transition-all duration-300 transform-gpu',
                      'bg-gradient-to-br from-white via-gray-50 to-white dark:from-gray-800 dark:via-gray-850 dark:to-gray-800',
                      'rounded-2xl border border-gray-200/60 dark:border-gray-700/60',
                      'backdrop-blur-sm shadow-lg hover:shadow-2xl',
                      selectedNode === node.id 
                        ? `border-2 ${nodeConfig.shadowColor} shadow-2xl ring-4 ring-opacity-20` 
                        : 'hover:border-gray-300/80 dark:hover:border-gray-600/80 hover:-translate-y-1',
                      draggedNode === node.id && 'shadow-3xl scale-105'
                    )}
                    style={{
                      boxShadow: selectedNode === node.id 
                        ? `0 20px 40px -12px ${nodeConfig.color}40, 0 8px 16px -8px ${nodeConfig.color}20`
                        : undefined
                    }}>
                      {/* 渐变顶部装饰条 */}
                      <div 
                        className={cn(
                          'absolute top-0 left-0 right-0 h-1.5 bg-gradient-to-r',
                          nodeConfig.gradient || 'from-gray-400 to-gray-600'
                        )}
                      />
                      
                      {/* 背景装饰 - 微妙的几何图案 */}
                      <div className="absolute top-0 right-0 w-24 h-24 opacity-5 dark:opacity-10">
                        <div 
                          className="w-full h-full rounded-full blur-xl"
                          style={{ backgroundColor: nodeConfig.color }}
                        />
                      </div>
                      
                      {/* 内容区域 */}
                      <div className="relative pt-4 pb-4 px-5">
                        {/* 头部：图标和标题 */}
                        <div className="flex items-center justify-between gap-4 mb-4">
                          <div className="flex items-center gap-4 flex-1 min-w-0">
                            {/* 现代化图标容器 */}
                            <div className="relative flex-shrink-0">
                              <div 
                                className={cn(
                                  'p-3 rounded-xl shadow-md transition-all duration-300',
                                  'bg-gradient-to-br border border-white/20',
                                  nodeConfig.gradient || 'from-gray-400 to-gray-600',
                                  'hover:scale-110 hover:rotate-3'
                                )}
                                style={{
                                  boxShadow: `0 8px 16px -4px ${nodeConfig.color}30`
                                }}
                              >
                                <div className="text-white drop-shadow-sm">
                                  {getIconComponent(nodeConfig.iconName)}
                                </div>
                              </div>
                              {/* 图标光晕效果 */}
                              <div 
                                className="absolute inset-0 rounded-xl blur-md opacity-30 -z-10"
                                style={{ backgroundColor: nodeConfig.color }}
                              />
                            </div>
                            
                            {/* 标题和类型 */}
                            <div className="flex-1 min-w-0">
                              <div className="text-lg font-bold text-gray-900 dark:text-gray-100 truncate leading-tight">
                                {node.data.label}
                              </div>
                              <div className="text-sm text-gray-600 dark:text-gray-400 mt-1 font-medium">
                                {nodeConfig.label}
                              </div>
                            </div>
                          </div>
                          
                          {/* 现代化操作按钮 */}
                          <div className={cn(
                            'flex items-center gap-2 transition-all duration-300 flex-shrink-0',
                            selectedNode === node.id ? 'opacity-100 scale-100' : 'opacity-0 scale-95 group-hover:opacity-100 group-hover:scale-100'
                          )}>
                            <motion.button
                              whileHover={{ scale: 1.1 }}
                              whileTap={{ scale: 0.95 }}
                              onClick={(e) => {
                                e.stopPropagation()
                                setConfiguringNode(node.id)
                              }}
                              className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-xl transition-all duration-200 shadow-sm hover:shadow-md"
                              title="配置"
                            >
                              <Settings className="w-4 h-4 text-gray-600 dark:text-gray-400" />
                            </motion.button>
                            <motion.button
                              whileHover={{ scale: 1.1 }}
                              whileTap={{ scale: 0.95 }}
                              onClick={(e) => {
                                e.stopPropagation()
                                deleteNode(node.id)
                              }}
                              className="p-2 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-xl transition-all duration-200 shadow-sm hover:shadow-md"
                              title="删除"
                            >
                              <Trash2 className="w-4 h-4 text-red-500 dark:text-red-400" />
                            </motion.button>
                          </div>
                        </div>

                        {/* 现代化参数展示区域 */}
                        {(() => {
                          // 开始节点：只显示输入参数
                          if (node.type === 'start' && node.inputs.length > 0) {
                            return (
                              <div className="pt-3 border-t border-gray-100/60 dark:border-gray-700/60">
                                <div className="space-y-3">
                                  <div className="flex items-center gap-2">
                                    <div className="w-3 h-3 rounded-full bg-gradient-to-r from-emerald-400 to-emerald-500 shadow-sm"></div>
                                    <span className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                                      输入参数 ({node.inputs.length})
                                    </span>
                                  </div>
                                  <div className="flex flex-wrap gap-2">
                                    {node.inputs.slice(0, 3).map((input, idx) => (
                                      <motion.span
                                        key={idx}
                                        initial={{ opacity: 0, scale: 0.8 }}
                                        animate={{ opacity: 1, scale: 1 }}
                                        transition={{ delay: idx * 0.1 }}
                                        className="px-3 py-1.5 text-xs font-medium bg-gradient-to-r from-emerald-50 to-emerald-100 dark:from-emerald-900/30 dark:to-emerald-800/30 text-emerald-700 dark:text-emerald-400 rounded-lg border border-emerald-200/60 dark:border-emerald-700/60 shadow-sm hover:shadow-md transition-all duration-200 truncate max-w-[120px]"
                                        title={input}
                                      >
                                        {input}
                                      </motion.span>
                                    ))}
                                    {node.inputs.length > 3 && (
                                      <span className="px-3 py-1.5 text-xs font-medium text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-700 rounded-lg">
                                        +{node.inputs.length - 3}
                                      </span>
                                    )}
                                  </div>
                                </div>
                              </div>
                            )
                          }
                          
                          // 结束节点：只显示输出参数
                          if (node.type === 'end' && node.outputs.length > 0) {
                            return (
                              <div className="pt-3 border-t border-gray-100/60 dark:border-gray-700/60">
                                <div className="space-y-3">
                                  <div className="flex items-center gap-2">
                                    <div className="w-3 h-3 rounded-full bg-gradient-to-r from-blue-400 to-blue-500 shadow-sm"></div>
                                    <span className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                                      输出参数 ({node.outputs.length})
                                    </span>
                                  </div>
                                  <div className="flex flex-wrap gap-2">
                                    {node.outputs.slice(0, 3).map((output, idx) => (
                                      <motion.span
                                        key={idx}
                                        initial={{ opacity: 0, scale: 0.8 }}
                                        animate={{ opacity: 1, scale: 1 }}
                                        transition={{ delay: idx * 0.1 }}
                                        className="px-3 py-1.5 text-xs font-medium bg-gradient-to-r from-blue-50 to-blue-100 dark:from-blue-900/30 dark:to-blue-800/30 text-blue-700 dark:text-blue-400 rounded-lg border border-blue-200/60 dark:border-blue-700/60 shadow-sm hover:shadow-md transition-all duration-200 truncate max-w-[120px]"
                                        title={output}
                                      >
                                        {output}
                                      </motion.span>
                                    ))}
                                    {node.outputs.length > 3 && (
                                      <span className="px-3 py-1.5 text-xs font-medium text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-700 rounded-lg">
                                        +{node.outputs.length - 3}
                                      </span>
                                    )}
                                  </div>
                                </div>
                              </div>
                            )
                          }
                          
                          // 其他节点：显示输入和输出参数
                          if (node.type !== 'start' && node.type !== 'end' && (node.inputs.length > 0 || node.outputs.length > 0)) {
                            return (
                              <div className="pt-3 border-t border-gray-100/60 dark:border-gray-700/60 space-y-4">
                                {/* 输入参数 */}
                                {node.inputs.length > 0 && (
                                  <div className="space-y-2">
                                    <div className="flex items-center gap-2">
                                      <div className="w-3 h-3 rounded-full bg-gradient-to-r from-emerald-400 to-emerald-500 shadow-sm"></div>
                                      <span className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                                        输入 ({node.inputs.length})
                                      </span>
                                    </div>
                                    <div className="flex flex-wrap gap-2">
                                      {node.inputs.slice(0, 3).map((input, idx) => (
                                        <motion.span
                                          key={idx}
                                          initial={{ opacity: 0, scale: 0.8 }}
                                          animate={{ opacity: 1, scale: 1 }}
                                          transition={{ delay: idx * 0.1 }}
                                          className="px-3 py-1.5 text-xs font-medium bg-gradient-to-r from-emerald-50 to-emerald-100 dark:from-emerald-900/30 dark:to-emerald-800/30 text-emerald-700 dark:text-emerald-400 rounded-lg border border-emerald-200/60 dark:border-emerald-700/60 shadow-sm hover:shadow-md transition-all duration-200 truncate max-w-[120px]"
                                          title={input}
                                        >
                                          {input}
                                        </motion.span>
                                      ))}
                                      {node.inputs.length > 3 && (
                                        <span className="px-3 py-1.5 text-xs font-medium text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-700 rounded-lg">
                                          +{node.inputs.length - 3}
                                        </span>
                                      )}
                                    </div>
                                  </div>
                                )}

                                {/* 输出参数 - gateway 节点不显示输出参数标签，因为它们是逻辑分支而不是数据输出 */}
                                {node.outputs.length > 0 && node.type !== 'gateway' && (
                                  <div className="space-y-2">
                                    <div className="flex items-center gap-2">
                                      <div className="w-3 h-3 rounded-full bg-gradient-to-r from-blue-400 to-blue-500 shadow-sm"></div>
                                      <span className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                                        输出 ({node.outputs.length})
                                      </span>
                                    </div>
                                    <div className="flex flex-wrap gap-2">
                                      {node.outputs.slice(0, 3).map((output, idx) => (
                                        <motion.span
                                          key={idx}
                                          initial={{ opacity: 0, scale: 0.8 }}
                                          animate={{ opacity: 1, scale: 1 }}
                                          transition={{ delay: idx * 0.1 }}
                                          className="px-3 py-1.5 text-xs font-medium bg-gradient-to-r from-blue-50 to-blue-100 dark:from-blue-900/30 dark:to-blue-800/30 text-blue-700 dark:text-blue-400 rounded-lg border border-blue-200/60 dark:border-blue-700/60 shadow-sm hover:shadow-md transition-all duration-200 truncate max-w-[120px]"
                                          title={output}
                                        >
                                          {output}
                                        </motion.span>
                                      ))}
                                      {node.outputs.length > 3 && (
                                        <span className="px-3 py-1.5 text-xs font-medium text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-700 rounded-lg">
                                          +{node.outputs.length - 3}
                                        </span>
                                      )}
                                    </div>
                                  </div>
                                )}
                              </div>
                            )
                          }
                          
                          return null
                        })()}
                      </div>
                    </div>

                    {/* 现代化输入连接点 - 渐变设计 */}
                    {node.inputs.map((input, index) => {
                      // 计算连接点位置：在节点左侧，根据输入参数数量均匀分布
                      const totalInputs = node.inputs.length
                      const spacing = totalInputs > 1 ? 60 / (totalInputs - 1) : 0
                      const topPosition = 45 + (index * spacing)
                      
                      return (
                        <motion.div
                          key={input}
                          className="absolute z-30"
                          style={{
                            left: -10,
                            top: `${topPosition}px`
                          }}
                          whileHover={{ scale: 1.2 }}
                          whileTap={{ scale: 0.9 }}
                        >
                          {/* 连接点光晕效果 */}
                          <div 
                            className={cn(
                              "absolute inset-0 rounded-full blur-sm opacity-60 transition-all duration-300",
                              isConnecting && connectionStart?.nodeId !== node.id
                                ? "bg-emerald-400 scale-150"
                                : "bg-emerald-300 scale-100"
                            )}
                          />
                          {/* 主连接点 */}
                          <div
                            className={cn(
                              "relative w-5 h-5 rounded-full cursor-pointer transition-all duration-300 transform-gpu",
                              "bg-gradient-to-br from-emerald-400 to-emerald-600",
                              "border-2 border-white dark:border-gray-800 shadow-lg",
                              "hover:shadow-emerald-300/50 hover:shadow-xl",
                              isConnecting && connectionStart?.nodeId !== node.id
                                ? "ring-4 ring-emerald-300/60 scale-110 animate-pulse"
                                : "hover:scale-110"
                            )}
                            onMouseDown={(e) => {
                              e.stopPropagation()
                              if (isConnecting) {
                                completeConnection(node.id, input)
                              }
                            }}
                            title={`输入: ${input}`}
                          >
                            {/* 内部高光 */}
                            <div className="absolute top-0.5 left-0.5 w-2 h-2 bg-white/40 rounded-full blur-sm" />
                          </div>
                        </motion.div>
                      )
                    })}

                    {/* 现代化输出连接点 - 渐变设计 */}
                    {node.outputs.map((output, index) => {
                      // 计算连接点位置：在节点右侧，根据输出参数数量均匀分布
                      const totalOutputs = node.outputs.length
                      const spacing = totalOutputs > 1 ? 60 / (totalOutputs - 1) : 0
                      const topPosition = 45 + (index * spacing)
                      
                      // 检查此输出点是否已经连接
                      const isConnected = isOutputConnected(node.id, output)
                      // 检查是否可以开始连接（未连接且不在连接模式中）
                      const canStartConnection = !isConnected && !isConnecting
                      
                      return (
                        <motion.div
                          key={output}
                          className="absolute z-30"
                          style={{
                            right: -10,
                            top: `${topPosition}px`
                          }}
                          whileHover={{ scale: canStartConnection ? 1.2 : 1.1 }}
                          whileTap={{ scale: canStartConnection ? 0.9 : 1 }}
                        >
                          {/* 连接点光晕效果 */}
                          <div 
                            className={cn(
                              "absolute inset-0 rounded-full blur-sm opacity-60 transition-all duration-300",
                              canStartConnection
                                ? "bg-blue-300 scale-100"
                                : isConnected
                                ? "bg-green-300 scale-90"
                                : "bg-gray-300 scale-75"
                            )}
                          />
                          
                          {/* 主连接点 */}
                          <div
                            className={cn(
                              "relative w-5 h-5 rounded-full transition-all duration-300 transform-gpu",
                              "border-2 border-white dark:border-gray-800 shadow-lg",
                              canStartConnection
                                ? "bg-gradient-to-br from-blue-400 to-blue-600 hover:shadow-blue-300/50 hover:shadow-xl cursor-pointer"
                                : isConnected
                                ? "bg-gradient-to-br from-green-400 to-green-600 shadow-green-300/50 cursor-default"
                                : "bg-gradient-to-br from-gray-300 to-gray-500 cursor-not-allowed opacity-60"
                            )}
                            onMouseDown={(e) => {
                              e.stopPropagation()
                              if (canStartConnection) {
                                startConnection(node.id, output)
                              }
                            }}
                            title={
                              isConnected 
                                ? `输出: ${output} (已连接)` 
                                : canStartConnection 
                                ? `输出: ${output}` 
                                : `输出: ${output} (连接模式中)`
                            }
                          >
                            {/* 内部高光 */}
                            <div className="absolute top-0.5 left-0.5 w-2 h-2 bg-white/40 rounded-full blur-sm" />
                          </div>
                        </motion.div>
                      )
                    })}
                  </motion.div>
                )
              })}
            </div>
          </div>
        </div>
      </div>
      
      {/* 节点配置面板 */}
      <NodeConfigPanel
        isOpen={!!configuringNode}
        node={configuringNode ? nodes.find(n => n.id === configuringNode) || null : null}
        nodeConfig={configuringNode ? NODE_TYPES[nodes.find(n => n.id === configuringNode)?.type as keyof typeof NODE_TYPES] : null}
        onClose={() => setConfiguringNode(null)}
        onNodeLabelChange={(label: string) => {
          if (configuringNode) {
            setNodes(prev => prev.map(n => 
              n.id === configuringNode ? { ...n, data: { ...n.data, label } } : n
            ))
          }
        }}
        onNodeConfigChange={(config: any) => {
          if (configuringNode) {
            setNodes(prev => prev.map(n => 
              n.id === configuringNode ? { 
                ...n, 
                data: { ...n.data, config },
                inputs: config.inputs !== undefined ? config.inputs : n.inputs,
                outputs: config.outputs !== undefined ? config.outputs : n.outputs,
              } : n
            ))
          }
        }}
        getIconComponent={getIconComponent}
        availableEventTypes={availableEventTypes}
        installedPlugins={installedPlugins}
        loadingPlugins={loadingPlugins}
        isCodeEditorFullscreen={isCodeEditorFullscreen}
        setIsCodeEditorFullscreen={setIsCodeEditorFullscreen}
      />

      {/* 运行参数模态框 */}
      <RunParametersModal
        isOpen={showRunParamsModal}
        onClose={() => setShowRunParamsModal(false)}
        nodes={nodes}
        parameters={runParameters}
        onParameterChange={(key, value) => {
          setRunParameters(prev => ({
            ...prev,
            [key]: value
          }))
        }}
        onRun={executeRun}
        isRunning={propsIsRunning || false}
      />

      {/* 节点测试对话框 */}
      <Modal
        isOpen={showNodeTestModal}
        onClose={() => {
          setShowNodeTestModal(false)
          setTestingNode(null)
          setNodeTestResult(null)
        }}
        title="测试节点"
        size="lg"
      >
        <div className="space-y-4">
          {(() => {
            const node = nodes.find(n => n.id === testingNode)
            if (!node || !node.inputs || node.inputs.length === 0) {
              return (
                <p className="text-sm text-gray-500 dark:text-gray-400 italic">
                  该节点没有定义输入参数
                </p>
              )
            }
            return (
              <>
                <div className="space-y-3">
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    请填写节点的输入参数值：
                  </p>
                  {node.inputs.map((input, idx) => (
                    <div key={idx}>
                      <Input
                        label={input || `参数 ${idx + 1}`}
                        size="sm"
                        value={nodeTestParameters[input] || ''}
                        onChange={(e) => {
                          setNodeTestParameters(prev => ({
                            ...prev,
                            [input]: e.target.value
                          }))
                        }}
                        placeholder={`请输入 ${input || `参数 ${idx + 1}`} 的值`}
                      />
                    </div>
                  ))}
                </div>

                {/* 测试结果 */}
                {nodeTestResult && (
                  <div className="mt-4 p-4 bg-gray-50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700 rounded-lg">
                    <h4 className="text-sm font-semibold text-gray-900 dark:text-white mb-3">
                      测试结果
                    </h4>
                    <div className="space-y-2 text-sm">
                      <div>
                        <span className="font-medium text-gray-700 dark:text-gray-300">状态：</span>
                        <span className={`ml-2 ${
                          nodeTestResult.status === 'completed' 
                            ? 'text-green-600 dark:text-green-400' 
                            : 'text-red-600 dark:text-red-400'
                        }`}>
                          {nodeTestResult.status === 'completed' ? '成功' : '失败'}
                        </span>
                      </div>
                      {nodeTestResult.error && (
                        <div>
                          <span className="font-medium text-gray-700 dark:text-gray-300">错误：</span>
                          <span className="ml-2 text-red-600 dark:text-red-400">{nodeTestResult.error}</span>
                        </div>
                      )}
                      {nodeTestResult.context && Object.keys(nodeTestResult.context).length > 0 && (
                        <div>
                          <span className="font-medium text-gray-700 dark:text-gray-300">输出数据：</span>
                          <pre className="mt-1 p-2 bg-gray-100 dark:bg-gray-900 rounded text-xs overflow-auto max-h-40">
                            {JSON.stringify(nodeTestResult.context, null, 2)}
                          </pre>
                        </div>
                      )}
                      {nodeTestResult.logs && nodeTestResult.logs.length > 0 && (
                        <div>
                          <span className="font-medium text-gray-700 dark:text-gray-300">执行日志：</span>
                          <div className="mt-1 p-2 bg-gray-100 dark:bg-gray-900 rounded text-xs overflow-auto max-h-40 space-y-1">
                            {nodeTestResult.logs.map((log: any, idx: number) => (
                              <div key={idx} className={`${
                                log.level === 'error' ? 'text-red-600 dark:text-red-400' :
                                log.level === 'success' ? 'text-green-600 dark:text-green-400' :
                                log.level === 'warning' ? 'text-yellow-600 dark:text-yellow-400' :
                                'text-gray-700 dark:text-gray-300'
                              }`}>
                                [{log.level.toUpperCase()}] {log.message}
                              </div>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  </div>
                )}

                <div className="flex items-center justify-end gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      setShowNodeTestModal(false)
                      setTestingNode(null)
                      setNodeTestResult(null)
                    }}
                  >
                    关闭
                  </Button>
                  <Button
                    variant="primary"
                    size="sm"
                    onClick={async () => {
                      if (!workflowId || !testingNode) return
                      
                      setIsTestingNode(true)
                      try {
                        // 转换参数值
                        const parsedParams: Record<string, any> = {}
                        Object.entries(nodeTestParameters).forEach(([key, value]) => {
                          if (value.trim() === '') {
                            parsedParams[key] = null
                          } else {
                            try {
                              parsedParams[key] = JSON.parse(value)
                            } catch {
                              parsedParams[key] = value
                            }
                          }
                        })
                        
                        const response = await workflowService.testNode(workflowId, testingNode, parsedParams)
                        if (response.code === 200 && response.data) {
                          setNodeTestResult(response.data)
                        } else {
                          setNodeTestResult({
                            status: 'failed',
                            error: response.msg || '测试失败'
                          })
                        }
                      } catch (error: any) {
                        setNodeTestResult({
                          status: 'failed',
                          error: error.message || '测试失败'
                        })
                      } finally {
                        setIsTestingNode(false)
                      }
                    }}
                    disabled={isTestingNode}
                    loading={isTestingNode}
                  >
                    {isTestingNode ? '测试中...' : '运行测试'}
                  </Button>
                </div>
              </>
            )
          })()}
        </div>
      </Modal>
      
      
      {/* 节点选择抽屉 */}
      <NodeDrawer
        isOpen={showNodeDrawer}
        onClose={() => setShowNodeDrawer(false)}
        nodeSearchQuery={nodeSearchQuery}
        onSearchChange={setNodeSearchQuery}
        NODE_TYPES={NODE_TYPES}
        installedPlugins={installedPlugins}
        loadingPlugins={loadingPlugins}
        onAddNode={(type, position) => {
          addNode(type, position)
          setTimeout(() => {
            centerOnNodes()
          }, 100)
        }}
        onAddPluginNode={(plugin, position) => {
          addPluginNode(plugin, position)
          setTimeout(() => {
            centerOnNodes()
          }, 100)
        }}
        getIconComponent={getIconComponent}
      />
      
      {/* 操作说明弹窗 */}
      {/* 帮助模态框 */}
      <HelpModal
        isOpen={showHelpModal}
        onClose={() => setShowHelpModal(false)}
      />

      {/* 发布为插件模态框 */}
      <Modal
        isOpen={showPublishModal}
        onClose={() => setShowPublishModal(false)}
        title="发布工作流为插件"
        size="lg"
      >
        <PublishWorkflowPluginModal
          workflowId={workflowId!}
          onClose={() => setShowPublishModal(false)}
        />
      </Modal>

      {/* 脚本全屏编辑器 */}
      <AnimatePresence>
        {isCodeEditorFullscreen && configuringNode && (() => {
          const node = nodes.find(n => n.id === configuringNode)
          return node && node.type === 'script' ? (
            <>
              {/* 背景遮罩 */}
              <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                onClick={() => setIsCodeEditorFullscreen(false)}
                className="fixed inset-0 bg-black/80 backdrop-blur-sm z-[100]"
              />
              {/* 全屏编辑器 */}
              <motion.div
                initial={{ opacity: 0, scale: 0.95 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.95 }}
                transition={{ duration: 0.2 }}
                className="fixed inset-4 z-[110] bg-slate-900 rounded-lg shadow-2xl flex flex-col overflow-hidden"
                onClick={(e) => e.stopPropagation()}
              >
                {/* 标题栏 */}
                <div className="h-14 bg-slate-800 border-b border-slate-700 flex items-center justify-between px-6 flex-shrink-0">
                  <div className="flex items-center gap-3">
                    <div className="p-2 rounded-lg bg-blue-500/15">
                      <Package className="w-5 h-5 text-blue-400" />
                    </div>
                    <div>
                      <h3 className="text-base font-semibold text-slate-200">
                        {node.data.label} - Go 脚本编辑器
                      </h3>
                      <p className="text-xs text-slate-400 mt-0.5">
                        全屏编辑模式
                      </p>
                    </div>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setIsCodeEditorFullscreen(false)}
                    className="flex items-center gap-2 text-slate-300 hover:text-white"
                  >
                    <Minimize2 className="w-4 h-4" />
                  </Button>
                </div>

                {/* 编辑器容器 */}
                <div className="flex-1 overflow-hidden">
                  <Suspense fallback={
                    <div className="h-full flex items-center justify-center bg-slate-900">
                      <div className="text-center">
                        <div className="animate-spin rounded-full h-8 w-8 border-4 border-slate-700 border-t-blue-500 mx-auto mb-3"></div>
                        <p className="text-sm text-slate-400">加载代码编辑器...</p>
                      </div>
                    </div>
                  }>
                    <MonacoEditor
                      height="100%"
                      language={node.data.config?.language || 'go'}
                      value={node.data.config?.code || ''}
                      onChange={(value) => {
                        setNodes(prev => prev.map(n => 
                          n.id === configuringNode ? { 
                            ...n, 
                            data: { ...n.data, config: { ...(n.data.config || {}), code: value || '' } }
                          } : n
                        ))
                      }}
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
              </motion.div>
            </>
          ) : null
        })()}
      </AnimatePresence>
    </div>
  )
}

// 发布工作流插件模态框组件
const PublishWorkflowPluginModal: React.FC<{
  workflowId: number
  onClose: () => void
}> = ({ workflowId, onClose }) => {
  const [workflow, setWorkflow] = useState<any>(null)
  const [loading, setLoading] = useState(false)
  const [step, setStep] = useState(1) // 添加步骤状态
  const [inputParameters, setInputParameters] = useState<any[]>([])
  const [outputParameters, setOutputParameters] = useState<any[]>([])
  const [formData, setFormData] = useState({
    name: '',
    displayName: '',
    description: '',
    category: 'utility' as WorkflowPluginCategory,
    icon: '',
    color: '#6366f1',
    tags: '',
    author: '',
    homepage: '',
    repository: '',
    license: 'MIT',
    inputSchema: {
      parameters: [] as any[]
    },
    outputSchema: {
      parameters: [] as any[]
    }
  })

  // 加载工作流详情
  useEffect(() => {
    const loadWorkflow = async () => {
      try {
        const response = await workflowService.getDefinition(workflowId)
        if (response.data) {
          const workflowData = response.data
          setWorkflow(workflowData)
          setFormData(prev => ({
            ...prev,
            name: workflowData.slug || workflowData.name.toLowerCase().replace(/\s+/g, '-'),
            displayName: workflowData.name,
            description: workflowData.description || '',
            inputSchema: {
              parameters: workflowData.inputParameters || []
            },
            outputSchema: {
              parameters: workflowData.outputParameters || []
            }
          }))
          // 初始化参数状态
          setInputParameters(workflowData.inputParameters || [])
          setOutputParameters(workflowData.outputParameters || [])
        }
      } catch (error) {
        console.error('加载工作流失败:', error)
      }
    }
    
    loadWorkflow()
  }, [workflowId])

  // 处理表单提交
  const handleSubmit = async () => {
    if (!workflow) return
    
    setLoading(true)
    try {
      const pluginData = {
        ...formData,
        tags: formData.tags.split(',').map((tag: string) => tag.trim()).filter(Boolean),
        inputSchema: {
          parameters: inputParameters
        },
        outputSchema: {
          parameters: outputParameters
        }
      }

      console.log('发送的插件数据:', pluginData)

      const response = await workflowPluginService.publishWorkflowAsPlugin(workflowId, pluginData)
      if (response.data) {
        // 使用 Toast 显示成功提示
        showAlert('您的工作流插件已成功发布到插件市场', 'success', '插件发布成功！')
        setStep(4) // 显示成功页面
        setTimeout(() => {
          onClose()
        }, 2000)
      }
    } catch (error) {
      console.error('发布工作流失败:', error)
      showAlert('插件发布时发生错误，请稍后重试', 'error', '发布失败')
    } finally {
      setLoading(false)
    }
  }

  const isFormValid = formData.name && formData.displayName && formData.description

  if (!workflow) {
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <div className="animate-spin rounded-full h-12 w-12 border-4 border-blue-500 border-t-transparent mb-4"></div>
        <p className="text-gray-600 dark:text-gray-400">加载工作流信息...</p>
      </div>
    )
  }

  // 成功页面
  if (step === 4) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <div className="w-16 h-16 bg-green-100 dark:bg-green-900/20 rounded-full flex items-center justify-center mb-4">
          <CheckCircle className="w-8 h-8 text-green-600 dark:text-green-400" />
        </div>
        <h3 className="text-xl font-semibold text-gray-900 dark:text-white mb-2">
          插件发布成功！
        </h3>
        <p className="text-gray-600 dark:text-gray-400 mb-4">
          您的工作流插件已成功发布，现在可以在插件市场中找到它。
        </p>
        <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
          <p className="text-sm text-blue-800 dark:text-blue-200">
            <strong>提示：</strong>插件已创建为草稿状态，您可以在插件市场的"我的插件"中查看和管理。
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="max-h-[80vh] overflow-y-auto">
      {/* 步骤指示器 */}
      <div className="flex items-center justify-center mb-8">
        <div className="flex items-center space-x-4">
          <div className={`flex items-center justify-center w-8 h-8 rounded-full text-sm font-medium ${
            step >= 1 ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-600'
          }`}>
            1
          </div>
          <div className={`h-1 w-16 ${step >= 2 ? 'bg-blue-600' : 'bg-gray-200'}`}></div>
          <div className={`flex items-center justify-center w-8 h-8 rounded-full text-sm font-medium ${
            step >= 2 ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-600'
          }`}>
            2
          </div>
          <div className={`h-1 w-16 ${step >= 3 ? 'bg-blue-600' : 'bg-gray-200'}`}></div>
          <div className={`flex items-center justify-center w-8 h-8 rounded-full text-sm font-medium ${
            step >= 3 ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-600'
          }`}>
            3
          </div>
        </div>
      </div>

      {step === 1 && (
        <div className="space-y-6">
          {/* 工作流信息预览 */}
          <div className="bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-blue-900/20 dark:to-indigo-900/20 border border-blue-200 dark:border-blue-800 rounded-xl p-6">
            <div className="flex items-start space-x-4">
              <div className="w-12 h-12 bg-blue-600 rounded-lg flex items-center justify-center">
                <GitBranch className="w-6 h-6 text-white" />
              </div>
              <div className="flex-1">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
                  {workflow.name}
                </h3>
                <p className="text-gray-600 dark:text-gray-400 mb-3">
                  {workflow.description || '暂无描述'}
                </p>
                <div className="flex items-center gap-3">
                  <span className={`px-3 py-1 text-xs font-medium rounded-full ${
                    workflow.status === 'active' 
                      ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
                      : workflow.status === 'draft'
                      ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400'
                      : 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400'
                  }`}>
                    {workflow.status === 'active' ? '已激活' : workflow.status === 'draft' ? '草稿' : '已归档'}
                  </span>
                  <span className="text-xs text-gray-500 bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded">
                    {workflow.definition?.nodes?.length || 0} 个节点
                  </span>
                  <span className="text-xs text-gray-500 bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded">
                    v{workflow.version}
                  </span>
                </div>
              </div>
            </div>
          </div>

          {/* 基本信息表单 */}
          <div className="space-y-6">
            <div>
              <h4 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">基本信息</h4>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    插件名称 <span className="text-red-500">*</span>
                  </label>
                  <Input
                    value={formData.name}
                    onChange={(e) => setFormData({...formData, name: e.target.value})}
                    placeholder="my-awesome-workflow"
                    className="font-mono text-sm"
                  />
                  <p className="text-xs text-gray-500 mt-1">用于标识插件的唯一名称，建议使用小写字母和连字符</p>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    显示名称 <span className="text-red-500">*</span>
                  </label>
                  <Input
                    value={formData.displayName}
                    onChange={(e) => setFormData({...formData, displayName: e.target.value})}
                    placeholder="我的超棒工作流"
                  />
                  <p className="text-xs text-gray-500 mt-1">在插件市场中显示的名称</p>
                </div>
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                插件描述 <span className="text-red-500">*</span>
              </label>
              <textarea
                value={formData.description}
                onChange={(e) => setFormData({...formData, description: e.target.value})}
                placeholder="详细描述这个工作流插件的功能和用途..."
                className="w-full px-3 py-3 border border-gray-300 dark:border-gray-600 rounded-lg resize-none bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                rows={4}
              />
              <p className="text-xs text-gray-500 mt-1">清晰的描述有助于其他用户理解和使用您的插件</p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">分类</label>
                <select
                  value={formData.category}
                  onChange={(e) => setFormData({...formData, category: e.target.value as WorkflowPluginCategory})}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                >
                  <option value="data_processing">📊 数据处理</option>
                  <option value="api_integration">🔗 API集成</option>
                  <option value="ai_service">🤖 AI服务</option>
                  <option value="notification">📢 通知服务</option>
                  <option value="utility">🛠️ 工具类</option>
                  <option value="business">💼 业务逻辑</option>
                  <option value="custom">⚙️ 自定义</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">主题色</label>
                <div className="flex items-center space-x-3">
                  <input
                    type="color"
                    value={formData.color}
                    onChange={(e) => setFormData({...formData, color: e.target.value})}
                    className="w-12 h-10 border border-gray-300 dark:border-gray-600 rounded-lg cursor-pointer"
                  />
                  <Input
                    value={formData.color}
                    onChange={(e) => setFormData({...formData, color: e.target.value})}
                    placeholder="#6366f1"
                    className="flex-1 font-mono text-sm"
                  />
                </div>
              </div>
            </div>
          </div>

          {/* 操作按钮 */}
          <div className="flex justify-between pt-6 border-t border-gray-200 dark:border-gray-700">
            <button
              onClick={onClose}
              className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600 rounded-md transition-colors"
            >
              取消
            </button>
            <button
              onClick={() => setStep(2)}
              disabled={!isFormValid}
              className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              下一步
              <ChevronRight className="w-4 h-4" />
            </button>
          </div>
        </div>
      )}

      {step === 2 && (
        <div className="space-y-6">
          <div>
            <h4 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">可选信息</h4>
            <p className="text-gray-600 dark:text-gray-400 mb-6">
              以下信息是可选的，但填写完整有助于提升插件的专业度和可信度。
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">标签</label>
            <Input
              value={formData.tags}
              onChange={(e) => setFormData({...formData, tags: e.target.value})}
              placeholder="自动化, 数据处理, API"
            />
            <p className="text-xs text-gray-500 mt-1">用逗号分隔多个标签，有助于用户搜索和发现</p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">作者</label>
              <Input
                value={formData.author}
                onChange={(e) => setFormData({...formData, author: e.target.value})}
                placeholder="您的名称"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">主页</label>
              <Input
                value={formData.homepage}
                onChange={(e) => setFormData({...formData, homepage: e.target.value})}
                placeholder="https://example.com"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">代码仓库</label>
              <Input
                value={formData.repository}
                onChange={(e) => setFormData({...formData, repository: e.target.value})}
                placeholder="https://github.com/user/repo"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">许可证</label>
            <select
              value={formData.license}
              onChange={(e) => setFormData({...formData, license: e.target.value})}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="MIT">MIT License</option>
              <option value="Apache-2.0">Apache License 2.0</option>
              <option value="GPL-3.0">GNU General Public License v3.0</option>
              <option value="BSD-3-Clause">BSD 3-Clause License</option>
              <option value="ISC">ISC License</option>
              <option value="Unlicense">The Unlicense</option>
              <option value="Custom">自定义许可证</option>
            </select>
          </div>

          {/* 操作按钮 */}
          <div className="flex justify-between pt-6 border-t border-gray-200 dark:border-gray-700">
            <button
              onClick={() => setStep(1)}
              className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600 rounded-md transition-colors"
            >
              <ChevronLeft className="w-4 h-4" />
              上一步
            </button>
            <button
              onClick={() => setStep(3)}
              className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md transition-colors"
            >
              下一步
              <ChevronRight className="w-4 h-4" />
            </button>
          </div>
        </div>
      )}

      {step === 3 && (
        <div className="space-y-6">
          <div>
            <h4 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">参数定义</h4>
            <p className="text-gray-600 dark:text-gray-400 mb-6">
              定义工作流的输入和输出参数，这些参数将在插件被使用时显示给用户。
            </p>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* 输入参数 */}
            <div>
              <div className="flex items-center justify-between mb-4">
                <h5 className="font-medium text-gray-900 dark:text-white">输入参数</h5>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    setInputParameters([
                      ...inputParameters,
                      { name: '', type: 'string', required: false, description: '' }
                    ])
                  }}
                >
                  + 添加
                </Button>
              </div>
              <div className="space-y-3 max-h-96 overflow-y-auto">
                {inputParameters.map((param, index) => (
                  <div key={index} className="p-3 border border-gray-200 dark:border-gray-700 rounded-lg">
                    <div className="grid grid-cols-2 gap-2 mb-2">
                      <Input
                        size="sm"
                        placeholder="参数名"
                        value={param.name}
                        onChange={(e) => {
                          const newParams = [...inputParameters]
                          newParams[index] = { ...param, name: e.target.value }
                          setInputParameters(newParams)
                        }}
                      />
                      <select
                        value={param.type}
                        onChange={(e) => {
                          const newParams = [...inputParameters]
                          newParams[index] = { ...param, type: e.target.value }
                          setInputParameters(newParams)
                        }}
                        className="px-2 py-1 border border-gray-300 dark:border-gray-600 rounded text-xs bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                      >
                        <option value="string">字符串</option>
                        <option value="number">数字</option>
                        <option value="boolean">布尔值</option>
                        <option value="object">对象</option>
                        <option value="array">数组</option>
                      </select>
                    </div>
                    <Input
                      size="sm"
                      placeholder="描述"
                      value={param.description}
                      onChange={(e) => {
                        const newParams = [...inputParameters]
                        newParams[index] = { ...param, description: e.target.value }
                        setInputParameters(newParams)
                      }}
                    />
                    <div className="flex items-center justify-between mt-2">
                      <label className="flex items-center text-xs gap-2">
                        <input
                          type="checkbox"
                          checked={param.required}
                          onChange={(e) => {
                            const newParams = [...inputParameters]
                            newParams[index] = { ...param, required: e.target.checked }
                            setInputParameters(newParams)
                          }}
                        />
                        <span>必需</span>
                      </label>
                      <Button
                        variant="outline"
                        size="xs"
                        onClick={() => {
                          setInputParameters(inputParameters.filter((_, i) => i !== index))
                        }}
                      >
                        删除
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* 输出参数 */}
            <div>
              <div className="flex items-center justify-between mb-4">
                <h5 className="font-medium text-gray-900 dark:text-white">输出参数</h5>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    setOutputParameters([
                      ...outputParameters,
                      { name: '', type: 'string', required: false, description: '' }
                    ])
                  }}
                >
                  + 添加
                </Button>
              </div>
              <div className="space-y-3 max-h-96 overflow-y-auto">
                {outputParameters.map((param, index) => (
                  <div key={index} className="p-3 border border-gray-200 dark:border-gray-700 rounded-lg">
                    <div className="grid grid-cols-2 gap-2 mb-2">
                      <Input
                        size="sm"
                        placeholder="参数名"
                        value={param.name}
                        onChange={(e) => {
                          const newParams = [...outputParameters]
                          newParams[index] = { ...param, name: e.target.value }
                          setOutputParameters(newParams)
                        }}
                      />
                      <select
                        value={param.type}
                        onChange={(e) => {
                          const newParams = [...outputParameters]
                          newParams[index] = { ...param, type: e.target.value }
                          setOutputParameters(newParams)
                        }}
                        className="px-2 py-1 border border-gray-300 dark:border-gray-600 rounded text-xs bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                      >
                        <option value="string">字符串</option>
                        <option value="number">数字</option>
                        <option value="boolean">布尔值</option>
                        <option value="object">对象</option>
                        <option value="array">数组</option>
                      </select>
                    </div>
                    <Input
                      size="sm"
                      placeholder="描述"
                      value={param.description}
                      onChange={(e) => {
                        const newParams = [...outputParameters]
                        newParams[index] = { ...param, description: e.target.value }
                        setOutputParameters(newParams)
                      }}
                    />
                    <div className="flex items-center justify-between mt-2">
                      <label className="flex items-center text-xs gap-2">
                        <input
                          type="checkbox"
                          checked={param.required}
                          onChange={(e) => {
                            const newParams = [...outputParameters]
                            newParams[index] = { ...param, required: e.target.checked }
                            setOutputParameters(newParams)
                          }}
                        />
                        <span>必需</span>
                      </label>
                      <Button
                        variant="outline"
                        size="xs"
                        onClick={() => {
                          setOutputParameters(outputParameters.filter((_, i) => i !== index))
                        }}
                      >
                        删除
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* 操作按钮 */}
          <div className="flex justify-between pt-6 border-t border-gray-200 dark:border-gray-700">
            <button
              onClick={() => setStep(2)}
              className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600 rounded-md transition-colors"
            >
              <ChevronLeft className="w-4 h-4" />
              上一步
            </button>
            <button
              onClick={handleSubmit}
              disabled={loading}
              className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-green-600 hover:bg-green-700 rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Package className="w-4 h-4" />
              {loading ? '发布中...' : '发布插件'}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

export default WorkflowEditor