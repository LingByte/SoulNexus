import React, { useState, useEffect, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { 
  Search,
  FileText,
  GitBranch,
  X,
  ChevronRight,
  AlertCircle,
  Lightbulb,
  Zap,
  Globe,
  Calendar,
  Webhook,
  Bot,
  Edit,
  Trash2
} from 'lucide-react'
import Button from '@/components/UI/Button'
import Card from '@/components/UI/Card'
import Badge from '@/components/UI/Badge'
import Input from '@/components/UI/Input'
import Modal from '@/components/UI/Modal'
import EmptyState from '@/components/UI/EmptyState'
import WorkflowEditor, { Workflow, WorkflowConnection } from '@/components/Voice/WorkflowEditor'
import Terminal, { TerminalLog } from '@/components/Workflow/Terminal'
import { showAlert } from '@/utils/notification'
import workflowService, { 
  WorkflowDefinition, 
  WorkflowGraph, 
  WorkflowNodeType, 
  WorkflowEdgeType,
  CreateWorkflowDefinitionRequest,
  UpdateWorkflowDefinitionRequest,
  WorkflowInstance,
  ExecutionLog,
  WorkflowTriggerConfig,
  WorkflowVersion,
  WorkflowVersionCompareResponse
} from '@/api/workflow'
import { buildWebSocketURL } from '@/config/apiConfig'

// 根据后端模型定义的类型（从 API 导入）
type WorkflowStatus = 'draft' | 'active' | 'archived'


const WorkflowManager: React.FC = () => {
  const [workflows, setWorkflows] = useState<WorkflowDefinition[]>([])
  const [filteredWorkflows, setFilteredWorkflows] = useState<WorkflowDefinition[]>([])
  const [selectedStatus, setSelectedStatus] = useState<string>('all')
  const [searchTerm, setSearchTerm] = useState('')
  const [selectedWorkflow, setSelectedWorkflow] = useState<WorkflowDefinition | null>(null)
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
  const [isEditModalOpen, setIsEditModalOpen] = useState(false)
  const [editingWorkflow, setEditingWorkflow] = useState<WorkflowDefinition | null>(null)
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid')
  const [showProperties, setShowProperties] = useState(true)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [terminalLogs, setTerminalLogs] = useState<TerminalLog[]>([])
  const [isTerminalVisible, setIsTerminalVisible] = useState(false)
  const [currentInstanceId, setCurrentInstanceId] = useState<number | null>(null)
  const [isRunning, setIsRunning] = useState(false)
  const [showTriggerConfig, setShowTriggerConfig] = useState(false)
  const [triggerConfig, setTriggerConfig] = useState<WorkflowDefinition['triggers']>({})
  const wsRef = useRef<WebSocket | null>(null)
  const [showVersionHistory, setShowVersionHistory] = useState(false)
  const [versions, setVersions] = useState<WorkflowVersion[]>([])
  const [loadingVersions, setLoadingVersions] = useState(false)
  const [showVersionCompare, setShowVersionCompare] = useState(false)
  const [compareData, setCompareData] = useState<WorkflowVersionCompareResponse | null>(null)
  const [selectedVersion1, setSelectedVersion1] = useState<number | null>(null)
  const [selectedVersion2, setSelectedVersion2] = useState<number | null>(null)
  const [changeNote, setChangeNote] = useState('')
  const [showParametersPanel, setShowParametersPanel] = useState(false)
  const [inputParameters, setInputParameters] = useState<any[]>([])
  const [outputParameters, setOutputParameters] = useState<any[]>([])

  // 加载工作流列表
  useEffect(() => {
    loadWorkflows()
  }, [selectedStatus, searchTerm])

  // 当选中工作流时，初始化触发器配置和参数
  useEffect(() => {
    if (selectedWorkflow) {
      setTriggerConfig(selectedWorkflow.triggers || {})
      setInputParameters(selectedWorkflow.inputParameters || [])
      setOutputParameters(selectedWorkflow.outputParameters || [])
    }
  }, [selectedWorkflow])

  const loadWorkflows = async () => {
    setLoading(true)
    setError(null)
    try {
      const params: { status?: WorkflowStatus; keyword?: string } = {}
      if (selectedStatus !== 'all') {
        params.status = selectedStatus as WorkflowStatus
      }
      if (searchTerm) {
        params.keyword = searchTerm
      }
      
      const response = await workflowService.listDefinitions(params)
      if (response.code === 200) {
        setWorkflows(response.data)
      } else {
        setError(response.msg || '加载工作流列表失败')
      }
    } catch (err: any) {
      setError(err.msg || err.message || '加载工作流列表失败')
      console.error('Failed to load workflows:', err)
    } finally {
      setLoading(false)
    }
  }

  // 过滤和搜索（前端过滤，如果后端支持搜索则不需要）
  useEffect(() => {
    let filtered = workflows

    if (selectedStatus === 'all' && !searchTerm) {
      // 如果使用后端搜索，直接使用返回的数据
      setFilteredWorkflows(filtered)
      return
    }

    // 前端过滤（作为备用）
    if (selectedStatus !== 'all') {
      filtered = filtered.filter(w => w.status === selectedStatus)
    }

    if (searchTerm) {
      const term = searchTerm.toLowerCase()
      filtered = filtered.filter(w => 
        w.name.toLowerCase().includes(term) ||
        w.slug.toLowerCase().includes(term) ||
        w.description.toLowerCase().includes(term) ||
        w.tags?.some(tag => tag.toLowerCase().includes(term))
      )
    }

    setFilteredWorkflows(filtered)
  }, [workflows, selectedStatus, searchTerm])

  const handleCreate = () => {
    setEditingWorkflow(null)
    setIsCreateModalOpen(true)
  }

  const handleEdit = (workflow: WorkflowDefinition) => {
    setEditingWorkflow(workflow)
    setIsEditModalOpen(true)
  }

  const handleSelectWorkflow = async (workflow: WorkflowDefinition) => {
    setLoading(true)
    setError(null)
    try {
      // 重新获取最新版本的工作流
      const response = await workflowService.getDefinition(workflow.id)
      if (response.code === 200) {
        setSelectedWorkflow(response.data)
      } else {
        setError(response.msg || '加载工作流失败')
      }
    } catch (err: any) {
      setError(err.msg || err.message || '加载工作流失败')
      console.error('Failed to load workflow:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (id: number) => {
    if (!window.confirm('确定要删除这个工作流定义吗？')) {
      return
    }

    setLoading(true)
    setError(null)
    try {
      const response = await workflowService.deleteDefinition(id)
      if (response.code === 200) {
        setWorkflows(prev => prev.filter(w => w.id !== id))
        if (selectedWorkflow?.id === id) {
          setSelectedWorkflow(null)
        }
      } else {
        setError(response.msg || '删除工作流失败')
      }
    } catch (err: any) {
      setError(err.msg || err.message || '删除工作流失败')
      console.error('Failed to delete workflow:', err)
    } finally {
      setLoading(false)
    }
  }

  // 加载版本历史
  const loadVersions = async (workflowId: number) => {
    setLoadingVersions(true)
    setVersions([]) // 清空之前的版本列表
    try {
      const response = await workflowService.listVersions(workflowId)
      console.log('版本历史响应:', response) // 调试日志
      if (response.code === 200) {
        setVersions(response.data || [])
        console.log('版本列表:', response.data) // 调试日志
        if (!response.data || response.data.length === 0) {
          showAlert('该工作流暂无版本历史记录。创建或更新工作流时会自动保存版本。', 'info', '提示')
        }
      } else {
        setError(response.msg || '加载版本历史失败')
        showAlert(response.msg || '加载版本历史失败', 'error', '加载失败')
      }
    } catch (err: any) {
      console.error('加载版本历史错误:', err) // 调试日志
      setError(err.msg || err.message || '加载版本历史失败')
      showAlert(err.msg || err.message || '加载版本历史失败', 'error', '加载失败')
    } finally {
      setLoadingVersions(false)
    }
  }

  // 对比版本
  const handleCompareVersions = async (workflowId: number, v1Id: number, v2Id: number) => {
    try {
      const response = await workflowService.compareVersions(workflowId, v1Id, v2Id)
      if (response.code === 200) {
        setCompareData(response.data)
        setShowVersionCompare(true)
      } else {
        setError(response.msg || '对比版本失败')
      }
    } catch (err: any) {
      setError(err.msg || err.message || '对比版本失败')
    }
  }

  // 回滚版本
  const handleRollback = async (workflowId: number, versionId: number) => {
    if (!confirm('确定要回滚到此版本吗？当前版本将被保存为历史版本。')) {
      return
    }
    setLoading(true)
    try {
      const response = await workflowService.rollbackVersion(workflowId, versionId)
      if (response.code === 200) {
        await loadWorkflows()
        if (selectedWorkflow?.id === workflowId) {
          const reloadResponse = await workflowService.getDefinition(workflowId)
          if (reloadResponse.code === 200) {
            setSelectedWorkflow(reloadResponse.data)
          }
        }
        setShowVersionHistory(false)
        showAlert('工作流已成功回滚到指定版本', 'success', '回滚成功')
      } else {
        setError(response.msg || '回滚失败')
      }
    } catch (err: any) {
      setError(err.msg || err.message || '回滚失败')
    } finally {
      setLoading(false)
    }
  }

  const handleSave = async (workflowData: Partial<WorkflowDefinition> & { changeNote?: string }) => {
    setSaving(true)
    setError(null)
    try {
      if (editingWorkflow) {
        // 更新工作流
        const updateData: UpdateWorkflowDefinitionRequest = {
          name: workflowData.name,
          description: workflowData.description,
          status: workflowData.status,
          definition: workflowData.definition,
          settings: workflowData.settings,
          tags: workflowData.tags,
          version: editingWorkflow.version, // 必须提供当前版本号
          changeNote: workflowData.changeNote || ''
        }
        
        const response = await workflowService.updateDefinition(editingWorkflow.id, updateData)
        if (response.code === 200) {
          setWorkflows(prev => prev.map(w => 
            w.id === editingWorkflow.id ? response.data : w
          ))
          if (selectedWorkflow?.id === editingWorkflow.id) {
            setSelectedWorkflow(response.data)
          }
          setIsEditModalOpen(false)
        } else {
          setError(response.msg || '更新工作流失败')
          if (response.msg?.includes('version conflict')) {
            // 版本冲突，重新加载数据
            await loadWorkflows()
          }
        }
      } else {
        // 创建工作流
        if (!workflowData.name || !workflowData.slug) {
          setError('名称和 Slug 是必填项')
          return
        }
        
        // 创建工作流时，至少需要一个开始节点和一个结束节点
        // 检查 definition 是否为空或没有节点
        let finalDefinition: WorkflowGraph
        if (!workflowData.definition || !workflowData.definition.nodes || workflowData.definition.nodes.length === 0) {
          // 生成唯一的 ID
          const timestamp = Date.now()
          const startId = `start-${timestamp}`
          const endId = `end-${timestamp}`
          const edgeId = `e-${timestamp}`
          
          finalDefinition = {
            nodes: [
              { id: startId, name: '开始', type: 'start', position: { x: 100, y: 100 } },
              { id: endId, name: '结束', type: 'end', position: { x: 300, y: 100 } }
            ],
            edges: [
              { id: edgeId, source: startId, target: endId, type: 'default' }
            ]
          }
        } else {
          finalDefinition = workflowData.definition
        }
        
        const createData: CreateWorkflowDefinitionRequest = {
          name: workflowData.name,
          slug: workflowData.slug,
          description: workflowData.description,
          status: workflowData.status || 'draft',
          definition: finalDefinition,
          settings: workflowData.settings,
          tags: workflowData.tags
        }
        
        const response = await workflowService.createDefinition(createData)
        if (response.code === 200) {
          setWorkflows(prev => [response.data, ...prev])
          setIsCreateModalOpen(false)
        } else {
          setError(response.msg || '创建工作流失败')
        }
      }
    } catch (err: any) {
      setError(err.msg || err.message || '保存工作流失败')
      console.error('Failed to save workflow:', err)
    } finally {
      setSaving(false)
    }
    setEditingWorkflow(null)
  }

  const getStatusBadge = (status: WorkflowStatus) => {
    const variants = {
      draft: { variant: 'muted' as const, label: '草稿' },
      active: { variant: 'success' as const, label: '激活' },
      archived: { variant: 'outline' as const, label: '归档' }
    }
    const config = variants[status]
    return <Badge variant={config.variant}>{config.label}</Badge>
  }

  const formatDate = (dateString: string) => {
    const date = new Date(dateString)
    return date.toLocaleDateString('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' })
  }

  // 如果选择了工作流，显示编辑器（Coze 风格的分屏布局）
  if (selectedWorkflow) {
    return (
      <div className="h-screen flex flex-col bg-gray-50 dark:bg-gray-900">
        {/* 顶部工具栏 */}
        <div className="h-14 border-b border-gray-200 dark:border-gray-800 dark:bg-gray-800 flex items-center justify-between px-4">
          <div className="flex items-center gap-4">
            <Button 
              variant="ghost"
              size="sm"
              onClick={() => setSelectedWorkflow(null)}
            >
              返回
            </Button>
            <div className="h-6 w-px bg-gray-300 dark:bg-gray-600" />
            <div>
              <h1 className="text-sm font-semibold text-gray-900 dark:text-white">
                {selectedWorkflow.name}
              </h1>
              <p className="text-xs text-gray-500 dark:text-gray-400">
                {selectedWorkflow.description}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="sm"
              onClick={async () => {
                console.log('点击版本历史按钮，工作流ID:', selectedWorkflow.id)
                await loadVersions(selectedWorkflow.id)
                console.log('设置showVersionHistory为true，当前versions:', versions)
                setShowVersionHistory(true)
                // 使用setTimeout确保状态更新
                setTimeout(() => {
                  console.log('状态更新后，showVersionHistory应该是true')
                }, 100)
              }}
            >
              版本历史
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowTriggerConfig(true)}
            >
              触发器配置
            </Button>
            {getStatusBadge(selectedWorkflow.status)}
            <Badge variant="outline" size="sm">v{selectedWorkflow.version}</Badge>
          </div>
        </div>

        {/* 主内容区域 - 编辑器全屏，使用内部的节点库 */}
        <div className="flex-1 flex overflow-hidden">
          {/* 编辑器 - 全屏显示，使用 WorkflowEditor 内部的节点库 */}
          <div className="flex-1 relative overflow-hidden">
            <WorkflowEditor
              workflowId={selectedWorkflow.id}
              workflow={{
                id: selectedWorkflow.id.toString(),
                name: selectedWorkflow.name,
                description: selectedWorkflow.description,
                nodes: selectedWorkflow.definition.nodes.map(n => {
                  // 节点类型直接使用后端定义的类型，不需要映射
                  // WorkflowEditor 现在支持所有后端节点类型
                  const nodeType = n.type as 'start' | 'end' | 'task' | 'gateway' | 'event' | 'subflow' | 'parallel' | 'wait' | 'timer' | 'script' | 'workflow_plugin' | 'ai_chat'
                  
                  // 根据节点类型生成输入输出
                  // 使用默认配置：start 0输入1输出，end 1输入0输出，其他1输入1输出，condition/gateway/parallel 1输入2输出
                  const getInputsOutputs = (type: WorkflowNodeType) => {
                    switch (type) {
                      case 'start':
                        return { inputs: 0, outputs: 1 }
                      case 'end':
                        return { inputs: 1, outputs: 0 }
                      case 'gateway':
                      case 'parallel':
                        return { inputs: 1, outputs: 2 }
                      case 'event':
                        return { inputs: 0, outputs: 1 }
                      default:
                        return { inputs: 1, outputs: 1 }
                    }
                  }
                  
                  const { inputs: inputCount, outputs: outputCount } = getInputsOutputs(n.type)
                  
                  // 优先使用 inputMap/outputMap 中定义的键，如果没有则生成默认的
                  const inputKeys = Object.keys(n.inputMap || {})
                  const outputKeys = Object.keys(n.outputMap || {})
                  
                  // 提取 pluginId 到顶层 data 对象（用于渲染时查找插件）
                  const pluginIdStr = n.properties?.pluginId || n.properties?.['pluginId']
                  const pluginId = pluginIdStr ? (typeof pluginIdStr === 'string' ? parseInt(pluginIdStr, 10) : pluginIdStr) : undefined
                  
                  return {
                    id: n.id,
                    type: nodeType,
                    position: n.position || { x: 0, y: 0 },
                    data: { 
                      label: n.name, 
                      config: n.properties || {},
                      // 保存原始的 inputMap 和 outputMap，以便在保存时恢复
                      _inputMap: n.inputMap,
                      _outputMap: n.outputMap,
                      // 为 workflow_plugin 节点提取 pluginId
                      ...(n.type === 'workflow_plugin' && pluginId ? { pluginId } : {})
                    },
                    inputs: inputKeys.length > 0 
                      ? inputKeys
                      : Array(inputCount).fill('').map((_, i) => `input-${i}`),
                    outputs: outputKeys.length > 0 
                      ? outputKeys
                      : Array(outputCount).fill('').map((_, i) => `output-${i}`)
                  }
                }),
                connections: selectedWorkflow.definition.edges.map(e => {
                  // 根据边的类型和条件，确定 sourceHandle 和 targetHandle
                  // 对于 condition/gateway 节点，true 分支使用 output-0，false 分支使用 output-1
                  // 对于 parallel 节点，branch 类型使用不同的 output
                  let sourceHandle = 'output-0'
                  let targetHandle = 'input-0'
                  
                  const sourceNode = selectedWorkflow.definition.nodes.find(n => n.id === e.source)
                  if (sourceNode) {
                    if (sourceNode.type === 'condition' || sourceNode.type === 'gateway') {
                      if (e.type === 'true') {
                        sourceHandle = 'output-0'
                      } else if (e.type === 'false') {
                        sourceHandle = 'output-1'
                      }
                    } else if (sourceNode.type === 'parallel' && e.type === 'branch') {
                      // 对于并行节点，可能需要根据边的索引确定 output
                      const branchIndex = selectedWorkflow.definition.edges
                        .filter(edge => edge.source === e.source && edge.type === 'branch')
                        .findIndex(edge => edge.id === e.id)
                      sourceHandle = `output-${branchIndex}`
                    }
                  }
                  
                  return {
                    id: e.id,
                    source: e.source,
                    target: e.target,
                    sourceHandle,
                    targetHandle,
                    type: e.type || 'default',
                    condition: e.condition
                  }
                }),
                createdAt: selectedWorkflow.createdAt,
                updatedAt: selectedWorkflow.updatedAt
              }}
              onSave={async (workflow: Workflow) => {
                if (!selectedWorkflow) {
                  showAlert('没有选中的工作流', 'error', '保存失败')
                  return
                }
                
                setSaving(true)
                setError(null)
                
                try {
                  // 将 WorkflowEditor 的格式转换回后端格式
                  const updatedDefinition: WorkflowGraph = {
                    nodes: workflow.nodes.map(node => {
                      // 根据节点类型和当前的 inputs/outputs 生成 inputMap/outputMap
                      // 开始节点：只有输入参数
                      // 结束节点：只有输出参数
                      // 其他节点：有输入和输出参数
                      const inputMap: Record<string, string> = {}
                      const outputMap: Record<string, string> = {}
                      
                      if (node.type === 'start') {
                        // 开始节点：只处理输入参数
                        if (node.inputs && node.inputs.length > 0) {
                          node.inputs.forEach((input) => {
                            if (input && input.trim()) {
                              // 对于开始节点，输入参数名就是参数名本身
                              // 运行时，这些值会从 context.Parameters 中获取
                              inputMap[input] = input
                            }
                          })
                        }
                      } else if (node.type === 'end') {
                        // 结束节点：只处理输出参数
                        if (node.outputs && node.outputs.length > 0) {
                          node.outputs.forEach((output) => {
                            if (output && output.trim()) {
                              // 对于结束节点，尝试从上游节点获取对应的输出
                              // 首先查找连接到结束节点的边
                              const incomingEdge = workflow.connections.find(conn => conn.target === node.id)
                              if (incomingEdge) {
                                const sourceNode = workflow.nodes.find(n => n.id === incomingEdge.source)
                                if (sourceNode) {
                                  if (sourceNode.type === 'ai_chat') {
                                    // 如果上游是 AI 对话节点，使用其 outputVariable
                                    const outputVar = sourceNode.data.config?.outputVariable
                                    if (outputVar) {
                                      outputMap[output] = `${sourceNode.id}.${outputVar}`
                                    } else {
                                      outputMap[output] = output
                                    }
                                  } else if (sourceNode.type === 'workflow_plugin' || sourceNode.type === 'script') {
                                    // 如果上游是工作流插件或脚本节点，使用 nodeId.outputName 格式
                                    outputMap[output] = `${sourceNode.id}.${output}`
                                  } else {
                                    // 其他节点类型，使用默认逻辑
                                    outputMap[output] = output
                                  }
                                } else {
                                  // 没有找到上游节点，使用默认逻辑
                                  outputMap[output] = output
                                }
                              } else {
                                // 没有找到上游节点，使用默认逻辑
                                outputMap[output] = output
                              }
                            }
                          })
                        }
                      } else if (node.type === 'ai_chat') {
                        // AI对话节点：使用 inputVariable 和 outputVariable
                        // inputVariable 是从上游节点获取数据的源
                        // outputVariable 是保存 AI 响应的目标变量名
                        const inputVar = node.data.config?.inputVariable
                        const outputVar = node.data.config?.outputVariable
                        
                        // inputMap: 将输入参数别名映射到源
                        // 对于 AI 对话节点，我们使用 inputVariable 作为别名
                        if (inputVar) {
                          inputMap[inputVar] = inputVar
                        }
                        
                        // outputMap: 将输出参数别名映射到目标
                        // 对于 AI 对话节点，我们使用 outputVariable 作为别名
                        if (outputVar) {
                          outputMap[outputVar] = `${node.id}.${outputVar}`
                        }
                      } else {
                        // 其他节点：处理输入和输出参数
                        if (node.inputs && node.inputs.length > 0) {
                          node.inputs.forEach((input) => {
                            if (input && input.trim()) {
                              // 查找连接到此节点的上游节点
                              const incomingEdge = workflow.connections.find(conn => conn.target === node.id)
                              if (incomingEdge) {
                                const sourceNode = workflow.nodes.find(n => n.id === incomingEdge.source)
                                if (sourceNode) {
                                  // 根据源节点类型确定输出参数名
                                  let sourceOutput = input // 默认使用相同的参数名
                                  
                                  if (sourceNode.type === 'workflow_plugin') {
                                    // 工作流插件节点：输出参数来自 outputMap
                                    // 查找匹配的输出参数
                                    const sourceOutputMap = sourceNode.data._outputMap || sourceNode.data.config?.outputMap || {}
                                    // 尝试找到匹配的输出参数
                                    for (const [key, value] of Object.entries(sourceOutputMap)) {
                                      if (key === input || value === input) {
                                        sourceOutput = key
                                        break
                                      }
                                    }
                                    // 如果没找到，使用第一个输出参数
                                    if (!sourceOutput && sourceNode.outputs && sourceNode.outputs.length > 0) {
                                      sourceOutput = sourceNode.outputs[0]
                                    }
                                    inputMap[input] = `${sourceNode.id}.${sourceOutput}`
                                  } else if (sourceNode.type === 'ai_chat') {
                                    // AI对话节点：使用 outputVariable
                                    const outputVar = sourceNode.data.config?.outputVariable || 'ai_response'
                                    inputMap[input] = `${sourceNode.id}.${outputVar}`
                                  } else {
                                    // 其他节点类型：使用 nodeId.outputName 格式
                                    inputMap[input] = `${sourceNode.id}.${sourceOutput}`
                                  }
                                } else {
                                  // 找不到源节点，使用默认值
                                  inputMap[input] = input
                                }
                              } else {
                                // 没有上游连接，使用默认值
                                inputMap[input] = input
                              }
                            }
                          })
                        }
                        
                        if (node.outputs && node.outputs.length > 0) {
                          node.outputs.forEach((output) => {
                            if (output && output.trim()) {
                              // 默认情况下，target 使用节点ID作为前缀，避免冲突
                              // 格式: nodeId.outputName
                              outputMap[output] = `${node.id}.${output}`
                            }
                          })
                        }
                      }
                      
                      // Convert config to properties (ensure all values are strings)
                      const properties: Record<string, string> = {}
                      if (node.data.config) {
                        for (const [key, value] of Object.entries(node.data.config)) {
                          // Convert all values to strings for properties
                          if (value !== null && value !== undefined) {
                            // Special handling for parameters object in workflow plugin nodes
                            if (key === 'parameters' && typeof value === 'object') {
                              properties[key] = JSON.stringify(value)
                            } else {
                              properties[key] = String(value)
                            }
                          }
                        }
                      }
                      
                      return {
                        id: node.id,
                        name: node.data.label,
                        type: node.type as WorkflowNodeType,
                        description: undefined,
                        position: node.position,
                        properties,
                        inputMap,
                        outputMap,
                        lanes: undefined
                      }
                    }),
                    edges: workflow.connections.map((conn: WorkflowConnection) => {
                      // 根据 sourceHandle 和节点类型确定边的类型
                      // conn.type 和 conn.condition 都是可选的，需要处理 undefined 情况
                      let edgeType: WorkflowEdgeType = (conn.type as WorkflowEdgeType | undefined) || 'default'
                      
                      const sourceNode = workflow.nodes.find(n => n.id === conn.source)
                      if (sourceNode) {
                        if (sourceNode.type === 'gateway') {
                          // 对于 gateway 节点，根据 sourceHandle 确定类型
                          const outputIndex = sourceNode.outputs.findIndex(o => o === conn.sourceHandle)
                          if (outputIndex === 0) {
                            edgeType = 'true'
                          } else if (outputIndex === 1) {
                            edgeType = 'false'
                          }
                        } else if (sourceNode.type === 'parallel') {
                          // 对于 parallel 节点，使用 branch 类型
                          edgeType = 'branch'
                        }
                      }
                      
                      return {
                        id: conn.id,
                        source: conn.source,
                        target: conn.target,
                        type: edgeType,
                        condition: conn.condition || undefined,
                        description: undefined,
                        metadata: undefined
                      }
                    })
                  }
                  
                  // 直接调用 API 更新工作流
                  const updateData: UpdateWorkflowDefinitionRequest = {
                    name: workflow.name,
                    description: workflow.description,
                    definition: updatedDefinition,
                    triggers: triggerConfig,
                    version: selectedWorkflow.version, // 必须提供当前版本号
                    changeNote: changeNote || '',
                    inputParameters: inputParameters,
                    outputParameters: outputParameters
                  }
                  setChangeNote('') // 清空变更说明
                  
                  const response = await workflowService.updateDefinition(selectedWorkflow.id, updateData)
                  
                  if (response.code === 200) {
                    // 更新本地状态
                    setWorkflows(prev => prev.map(w => 
                      w.id === selectedWorkflow.id ? response.data : w
                    ))
                    setSelectedWorkflow(response.data)
                    
                    // 显示成功提示
                    showAlert('工作流已成功保存', 'success', '保存成功')
                  } else {
                    setError(response.msg || '更新工作流失败')
                    showAlert(response.msg || '更新工作流失败', 'error', '保存失败')
                    
                    if (response.msg?.includes('version conflict')) {
                      // 版本冲突，重新加载数据
                      await loadWorkflows()
                      // 重新加载选中的工作流
                      const reloadResponse = await workflowService.getDefinition(selectedWorkflow.id)
                      if (reloadResponse.code === 200) {
                        setSelectedWorkflow(reloadResponse.data)
                      }
                    }
                  }
                } catch (error: any) {
                  setError(error.msg || error.message || '保存工作流失败')
                  showAlert(error.msg || error.message || '保存工作流时发生错误', 'error', '保存失败')
                  console.error('Failed to save workflow:', error)
                } finally {
                  setSaving(false)
                }
              }}
              onStop={async (instanceId: number) => {
                try {
                  const response = await workflowService.stopInstance(instanceId)

                  if (response.code === 200) {
                    setIsRunning(false)
                    setCurrentInstanceId(null)
                    const now = new Date()
                    const timestamp = `${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}:${now.getSeconds().toString().padStart(2, '0')}.${now.getMilliseconds().toString().padStart(3, '0')}`
                    setTerminalLogs(prev => [...prev, {
                      timestamp,
                      level: 'warning',
                      message: '工作流已被用户停止'
                    }])
                    showAlert('工作流已停止', 'success', '停止成功')
                  } else {
                    console.error('Failed to stop workflow:', response.msg)
                    showAlert(response.msg || '停止工作流失败', 'error', '停止失败')
                  }
                } catch (error: any) {
                  console.error('Error stopping workflow:', error)
                  showAlert(error.msg || error.message || '停止工作流时发生错误', 'error', '停止失败')
                }
              }}
              isRunning={isRunning}
              currentInstanceId={currentInstanceId}
              onRun={async (workflow, parameters = {}) => {
                if (!selectedWorkflow) {
                  showAlert('没有选中的工作流', 'error', '运行失败')
                  return
                }
                
                // 清空之前的日志并显示终端
                setTerminalLogs([])
                setIsTerminalVisible(true)
                
                // 关闭之前的 WebSocket 连接
                if (wsRef.current) {
                  wsRef.current.close()
                  wsRef.current = null
                }
                
                // 建立 WebSocket 连接以接收实时日志
                try {
                  const token = localStorage.getItem('auth_token')
                  const wsUrl = token 
                    ? `${buildWebSocketURL('/api/ws')}?token=${encodeURIComponent(token)}`
                    : buildWebSocketURL('/api/ws')
                  const ws = new WebSocket(wsUrl)
                  wsRef.current = ws
                  
                  ws.onopen = () => {
                    console.log('WebSocket connected for workflow logs')
                  }
                  
                  ws.onmessage = (event) => {
                    try {
                      // 处理可能的多行 JSON（虽然后端已修复，但为了健壮性保留）
                      const dataStr = event.data.toString().trim()
                      if (!dataStr) return
                      
                      // 尝试解析单条消息
                      const message = JSON.parse(dataStr)
                      if (message.type === 'workflow_log' && message.data) {
                        const log = message.data as ExecutionLog
                        const convertedLog: TerminalLog = {
                          timestamp: log.timestamp,
                          level: log.level as TerminalLog['level'],
                          message: log.message,
                          nodeId: log.nodeId,
                          nodeName: log.nodeName
                        }
                        setTerminalLogs(prev => [...prev, convertedLog])
                      }
                    } catch (err) {
                      // 如果解析失败，尝试处理多行 JSON（向后兼容）
                      try {
                        const lines = event.data.toString().split('\n').filter((line: string) => line.trim())
                        for (const line of lines) {
                          const message = JSON.parse(line.trim())
                          if (message.type === 'workflow_log' && message.data) {
                            const log = message.data as ExecutionLog
                            const convertedLog: TerminalLog = {
                              timestamp: log.timestamp,
                              level: log.level as TerminalLog['level'],
                              message: log.message,
                              nodeId: log.nodeId,
                              nodeName: log.nodeName
                            }
                            setTerminalLogs(prev => [...prev, convertedLog])
                          }
                        }
                      } catch (parseErr) {
                        console.error('Failed to parse WebSocket message:', err, 'Data:', event.data)
                      }
                    }
                  }
                  
                  ws.onerror = (error) => {
                    console.error('WebSocket error:', error)
                  }
                  
                  ws.onclose = () => {
                    console.log('WebSocket closed')
                    wsRef.current = null
                  }
                } catch (wsError) {
                  console.error('Failed to establish WebSocket connection:', wsError)
                }
                
                try {
                  // 添加开始日志
                  const now = new Date()
                  const timestamp = `${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}:${now.getSeconds().toString().padStart(2, '0')}.${now.getMilliseconds().toString().padStart(3, '0')}`
                  setTerminalLogs(prev => [...prev, {
                    timestamp,
                    level: 'info',
                    message: `开始运行工作流: ${workflow.name}`
                  }])
                  
                  const response = await workflowService.runDefinition(selectedWorkflow.id, parameters)
                  
                  if (response.code === 200) {
                    // 后端返回的数据可能是 { instance: WorkflowInstance, logs?: ExecutionLog[] } 格式
                    // 或者直接是 WorkflowInstance（向后兼容）
                    let instance: WorkflowInstance | null = null
                    let logs: ExecutionLog[] | undefined = undefined
                    
                    if (response.data && typeof response.data === 'object') {
                      // 检查是否是包装格式 { instance: ..., logs: ... }
                      if ('instance' in response.data) {
                        const runResponse = response.data as { instance: WorkflowInstance; logs?: ExecutionLog[] }
                        instance = runResponse.instance
                        logs = runResponse.logs
                      } else if ('id' in response.data && 'status' in response.data) {
                        // 直接是 WorkflowInstance 格式
                        instance = response.data as WorkflowInstance
                        logs = undefined
                      }
                    }
                    
                    if (!instance) {
                      throw new Error('无法解析工作流执行结果')
                    }
                    
                    // 设置当前运行的实例ID和运行状态
                    setCurrentInstanceId(instance.id)
                    setIsRunning(true)
                    
                    // 添加后端返回的日志
                    if (logs && logs.length > 0) {
                      const convertedLogs: TerminalLog[] = logs.map(log => ({
                        timestamp: log.timestamp,
                        level: log.level as TerminalLog['level'],
                        message: log.message,
                        nodeId: log.nodeId,
                        nodeName: log.nodeName
                      }))
                      setTerminalLogs(prev => [...prev, ...convertedLogs])
                    }
                    
                    if (instance.status === 'completed') {
                      const duration = instance.completedAt && instance.startedAt 
                        ? `${Math.round((new Date(instance.completedAt).getTime() - new Date(instance.startedAt!).getTime()) / 1000)}秒`
                        : '未知'
                      
                      const completedTime = new Date()
                      const completedTimestamp = `${completedTime.getHours().toString().padStart(2, '0')}:${completedTime.getMinutes().toString().padStart(2, '0')}:${completedTime.getSeconds().toString().padStart(2, '0')}.${completedTime.getMilliseconds().toString().padStart(3, '0')}`
                      setTerminalLogs(prev => [...prev, {
                        timestamp: completedTimestamp,
                        level: 'success',
                        message: `工作流执行完成，耗时: ${duration}`
                      }])
                      
                      setIsRunning(false)
                      setCurrentInstanceId(null)
                      showAlert(`工作流执行完成，耗时: ${duration}`, 'success', '运行成功')
                    } else if (instance && instance.status === 'failed') {
                      const failedTime = new Date()
                      const failedTimestamp = `${failedTime.getHours().toString().padStart(2, '0')}:${failedTime.getMinutes().toString().padStart(2, '0')}:${failedTime.getSeconds().toString().padStart(2, '0')}.${failedTime.getMilliseconds().toString().padStart(3, '0')}`
                      setTerminalLogs(prev => [...prev, {
                        timestamp: failedTimestamp,
                        level: 'error',
                        message: instance.resultData?.error || '工作流执行失败'
                      }])
                      
                      setIsRunning(false)
                      setCurrentInstanceId(null)
                      showAlert(instance.resultData?.error || '工作流执行失败', 'error', '运行失败')
                    }
                    
                    // 返回实例信息供前端使用
                    return { instance }
                  } else {
                    const errorTime = new Date()
                    const errorTimestamp = `${errorTime.getHours().toString().padStart(2, '0')}:${errorTime.getMinutes().toString().padStart(2, '0')}:${errorTime.getSeconds().toString().padStart(2, '0')}.${errorTime.getMilliseconds().toString().padStart(3, '0')}`
                    setTerminalLogs(prev => [...prev, {
                      timestamp: errorTimestamp,
                      level: 'error',
                      message: response.msg || '运行工作流时发生错误'
                    }])
                    
                    showAlert(response.msg || '运行工作流时发生错误', 'error', '运行失败')
                  }
                } catch (error: any) {
                  const catchErrorTime = new Date()
                  const catchErrorTimestamp = `${catchErrorTime.getHours().toString().padStart(2, '0')}:${catchErrorTime.getMinutes().toString().padStart(2, '0')}:${catchErrorTime.getSeconds().toString().padStart(2, '0')}.${catchErrorTime.getMilliseconds().toString().padStart(3, '0')}`
                  setTerminalLogs(prev => [...prev, {
                    timestamp: catchErrorTimestamp,
                    level: 'error',
                    message: error.msg || error.message || '运行工作流时发生错误'
                  }])
                  
                  showAlert(error.msg || error.message || '运行工作流时发生错误', 'error', '运行失败')
                  console.error('Failed to run workflow:', error)
                } finally {
                  // 关闭 WebSocket 连接
                  if (wsRef.current) {
                    wsRef.current.close()
                    wsRef.current = null
                  }
                }
              }}
            />
          </div>

          {/* 右侧：属性面板 */}
          <AnimatePresence>
            {showProperties && (
              <motion.div
                initial={{ width: 0, opacity: 0 }}
                animate={{ width: 320, opacity: 1 }}
                exit={{ width: 0, opacity: 0 }}
                transition={{ duration: 0.2 }}
                className="border-l border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-800 overflow-hidden flex flex-col"
              >
                <div className="p-4 border-b border-gray-200 dark:border-gray-700">
                  <div className="flex items-center justify-between">
                    <h2 className="text-sm font-semibold text-gray-900 dark:text-white">属性</h2>
                    <Button
                      variant="ghost"
                      size="xs"
                      onClick={() => setShowProperties(false)}
                    >
                      <X className="w-4 h-4" />
                    </Button>
            </div>
            {/* 选项卡 */}
            <div className="flex gap-2 mt-3">
              <Button
                variant={!showParametersPanel ? "primary" : "outline"}
                size="xs"
                onClick={() => setShowParametersPanel(false)}
              >
                基本信息
              </Button>
              <Button
                variant={showParametersPanel ? "primary" : "outline"}
                size="xs"
                onClick={() => setShowParametersPanel(true)}
              >
                参数定义
              </Button>
            </div>
          </div>
                <div className="flex-1 overflow-y-auto p-4 space-y-4">
                  {!showParametersPanel ? (
                    <>
                      {/* 基本信息选项卡 */}
                      {/* 变更说明输入框 */}
                      <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                          变更说明（可选）
                        </label>
                        <textarea
                          className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm"
                          rows={3}
                          placeholder="描述本次更新的内容..."
                          value={changeNote}
                          onChange={(e) => setChangeNote(e.target.value)}
                        />
                        <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                          保存工作流时会自动创建版本历史记录
                        </p>
                      </div>
                      <div className="space-y-4">
                        <div>
                          <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
                            工作流名称
                          </label>
                          <Input
                            size="sm"
                            value={selectedWorkflow.name}
                            readOnly
                          />
                        </div>
                        <div>
                          <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
                            描述
                          </label>
                          <textarea
                            className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
                            rows={3}
                            value={selectedWorkflow.description}
                            readOnly
                          />
                            </div>
                        <div>
                          <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
                            标签
                          </label>
                          <div className="flex flex-wrap gap-2">
                            {selectedWorkflow.tags?.map((tag, idx) => (
                              <Badge key={idx} variant="outline" size="xs">{tag}</Badge>
                            ))}
                          </div>
                      </div>
                        <div className="pt-4 border-t border-gray-200 dark:border-gray-700">
                          <div className="text-xs text-gray-500 dark:text-gray-400 space-y-1">
                            <div className="flex justify-between">
                              <span>创建时间</span>
                              <span>{formatDate(selectedWorkflow.createdAt)}</span>
                </div>
                            <div className="flex justify-between">
                              <span>更新时间</span>
                              <span>{formatDate(selectedWorkflow.updatedAt)}</span>
            </div>
                            <div className="flex justify-between">
                              <span>创建者</span>
                              <span>{selectedWorkflow.createdBy}</span>
                </div>
                        </div>
                      </div>
                    </div>
                    </>
                  ) : (
                    <>
                      {/* 参数定义选项卡 */}
                      <div className="space-y-4">
                        {/* 输入参数 */}
                        <div>
                          <div className="flex items-center justify-between mb-3">
                            <h3 className="font-medium text-gray-900 dark:text-white">输入参数</h3>
                            <Button
                              variant="outline"
                              size="xs"
                              onClick={() => {
                                setInputParameters([
                                  ...inputParameters,
                                  { name: '', type: 'string', required: false, description: '' }
                                ])
                              }}
                            >
                              添加
                            </Button>
                          </div>
                          <div className="space-y-2 max-h-48 overflow-y-auto">
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
                        <div className="pt-4 border-t border-gray-200 dark:border-gray-700">
                          <div className="flex items-center justify-between mb-3">
                            <h3 className="font-medium text-gray-900 dark:text-white">输出参数</h3>
                            <Button
                              variant="outline"
                              size="xs"
                              onClick={() => {
                                setOutputParameters([
                                  ...outputParameters,
                                  { name: '', type: 'string', required: false, description: '' }
                                ])
                              }}
                            >
                              添加
                            </Button>
                          </div>
                          <div className="space-y-2 max-h-48 overflow-y-auto">
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
                    </>
                  )}
                </div>
              </motion.div>
            )}
          </AnimatePresence>

          {!showProperties && (
            <Button
              variant="ghost"
              size="sm"
              className="absolute right-2 top-2 z-10"
              onClick={() => setShowProperties(true)}
            >
              <ChevronRight className="w-4 h-4 rotate-180" />
            </Button>
          )}
        </div>
        
        {/* 终端组件 */}
        <Terminal
          logs={terminalLogs}
          isVisible={isTerminalVisible}
          onClose={() => setIsTerminalVisible(false)}
          onClear={() => setTerminalLogs([])}
        />

        {/* 触发器配置模态框 */}
        <Modal
          isOpen={showTriggerConfig}
          onClose={() => setShowTriggerConfig(false)}
          title="触发器配置"
          size="xl"
        >
          <TriggerConfigPanel
            workflow={selectedWorkflow}
            triggerConfig={triggerConfig}
            onUpdate={(config) => setTriggerConfig(config)}
            onSave={async () => {
              if (!selectedWorkflow) return
              
              setSaving(true)
              try {
                const updateData: UpdateWorkflowDefinitionRequest = {
                  triggers: triggerConfig,
                  version: selectedWorkflow.version,
                  changeNote: changeNote || ''
                }
                setChangeNote('')
                
                const response = await workflowService.updateDefinition(selectedWorkflow.id, updateData)
                
                if (response.code === 200) {
                  setWorkflows(prev => prev.map(w => 
                    w.id === selectedWorkflow.id ? response.data : w
                  ))
                  setSelectedWorkflow(response.data)
                  setShowTriggerConfig(false)
                  
                  showAlert('触发器配置已保存', 'success', '保存成功')
                } else {
                  showAlert(response.msg || '保存触发器配置失败', 'error', '保存失败')
                }
              } catch (error: any) {
                showAlert(error.msg || error.message || '保存触发器配置时发生错误', 'error', '保存失败')
              } finally {
                setSaving(false)
              }
            }}
            saving={saving}
          />
        </Modal>

        {/* 版本历史模态框 */}
        <Modal
          isOpen={showVersionHistory}
          onClose={() => {
            setShowVersionHistory(false)
            setVersions([])
          }}
          title="版本历史"
          size="xl"
        >
          <div className="space-y-4">
            {loadingVersions ? (
              <div className="text-center py-8 text-gray-500 dark:text-gray-400">加载中...</div>
            ) : versions.length === 0 ? (
              <div className="text-center py-8 text-gray-500 dark:text-gray-400">暂无版本历史</div>
            ) : (
              <div className="space-y-2 max-h-96 overflow-y-auto">
                {versions.map((version) => (
                  <div
                    key={version.id}
                    className="p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
                  >
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-2">
                          <Badge variant="outline">v{version.version}</Badge>
                          {version.changeNote && (
                            <span className="text-sm text-gray-600 dark:text-gray-400">
                              {version.changeNote}
                            </span>
                          )}
                        </div>
                        <div className="text-xs text-gray-500 dark:text-gray-400 space-y-1">
                          <div>更新者: {version.updatedBy}</div>
                          <div>更新时间: {new Date(version.createdAt).toLocaleString('zh-CN')}</div>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <Button
                          variant="ghost"
                          size="xs"
                          onClick={() => {
                            if (selectedVersion1 === null) {
                              setSelectedVersion1(version.id)
                            } else if (selectedVersion2 === null && selectedVersion1 !== version.id) {
                              setSelectedVersion2(version.id)
                              if (selectedWorkflow) {
                                handleCompareVersions(selectedWorkflow.id, selectedVersion1, version.id)
                              }
                            } else {
                              setSelectedVersion1(version.id)
                              setSelectedVersion2(null)
                            }
                          }}
                        >
                          对比
                        </Button>
                        <Button
                          variant="ghost"
                          size="xs"
                          onClick={() => {
                            if (selectedWorkflow) {
                              handleRollback(selectedWorkflow.id, version.id)
                            }
                          }}
                        >
                          回滚
                        </Button>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </Modal>

        {/* 版本对比模态框 */}
        <Modal
          isOpen={showVersionCompare}
          onClose={() => {
            setShowVersionCompare(false)
            setCompareData(null)
            setSelectedVersion1(null)
            setSelectedVersion2(null)
          }}
          title="版本对比"
          size="xl"
        >
          {compareData && (
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4 mb-4">
                <div className="p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                  <div className="text-sm font-semibold text-blue-900 dark:text-blue-300 mb-1">
                    版本 {compareData.version1.version}
                  </div>
                  <div className="text-xs text-blue-700 dark:text-blue-400">
                    {new Date(compareData.version1.createdAt).toLocaleString('zh-CN')}
                  </div>
                </div>
                <div className="p-3 bg-green-50 dark:bg-green-900/20 rounded-lg">
                  <div className="text-sm font-semibold text-green-900 dark:text-green-300 mb-1">
                    版本 {compareData.version2.version}
                  </div>
                  <div className="text-xs text-green-700 dark:text-green-400">
                    {new Date(compareData.version2.createdAt).toLocaleString('zh-CN')}
                  </div>
                </div>
              </div>

              <div className="space-y-4 max-h-96 overflow-y-auto">
                {compareData.diff.name && (
                  <div className="p-3 border border-gray-200 dark:border-gray-700 rounded-lg">
                    <div className="text-sm font-semibold mb-2">名称变更</div>
                    <div className="text-sm">
                      <div className="text-red-600 dark:text-red-400">- {compareData.diff.name.old}</div>
                      <div className="text-green-600 dark:text-green-400">+ {compareData.diff.name.new}</div>
                    </div>
                  </div>
                )}

                {compareData.diff.description && (
                  <div className="p-3 border border-gray-200 dark:border-gray-700 rounded-lg">
                    <div className="text-sm font-semibold mb-2">描述变更</div>
                    <div className="text-sm">
                      <div className="text-red-600 dark:text-red-400">- {compareData.diff.description.old}</div>
                      <div className="text-green-600 dark:text-green-400">+ {compareData.diff.description.new}</div>
                    </div>
                  </div>
                )}

                {compareData.diff.nodes && (
                  <div className="p-3 border border-gray-200 dark:border-gray-700 rounded-lg">
                    <div className="text-sm font-semibold mb-2">节点变更</div>
                    {compareData.diff.nodes.added && compareData.diff.nodes.added.length > 0 && (
                      <div className="mb-2">
                        <div className="text-xs font-medium text-green-600 dark:text-green-400 mb-1">新增节点:</div>
                        {compareData.diff.nodes.added.map((node) => (
                          <div key={node.id} className="text-sm text-green-600 dark:text-green-400 ml-2">
                            + {node.name} ({node.type})
                          </div>
                        ))}
                      </div>
                    )}
                    {compareData.diff.nodes.removed && compareData.diff.nodes.removed.length > 0 && (
                      <div className="mb-2">
                        <div className="text-xs font-medium text-red-600 dark:text-red-400 mb-1">删除节点:</div>
                        {compareData.diff.nodes.removed.map((node) => (
                          <div key={node.id} className="text-sm text-red-600 dark:text-red-400 ml-2">
                            - {node.name} ({node.type})
                          </div>
                        ))}
                      </div>
                    )}
                    {compareData.diff.nodes.modified && compareData.diff.nodes.modified.length > 0 && (
                      <div>
                        <div className="text-xs font-medium text-yellow-600 dark:text-yellow-400 mb-1">修改节点:</div>
                        {compareData.diff.nodes.modified.map((item) => (
                          <div key={item.id} className="text-sm ml-2">
                            <div className="text-yellow-600 dark:text-yellow-400">
                              ~ {item.old.name} → {item.new.name}
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}

                {compareData.diff.edges && (
                  <div className="p-3 border border-gray-200 dark:border-gray-700 rounded-lg">
                    <div className="text-sm font-semibold mb-2">边变更</div>
                    {compareData.diff.edges.added && compareData.diff.edges.added.length > 0 && (
                      <div className="mb-2">
                        <div className="text-xs font-medium text-green-600 dark:text-green-400 mb-1">新增边:</div>
                        {compareData.diff.edges.added.map((edge) => (
                          <div key={edge.id} className="text-sm text-green-600 dark:text-green-400 ml-2">
                            + {edge.source} → {edge.target}
                          </div>
                        ))}
                      </div>
                    )}
                    {compareData.diff.edges.removed && compareData.diff.edges.removed.length > 0 && (
                      <div className="mb-2">
                        <div className="text-xs font-medium text-red-600 dark:text-red-400 mb-1">删除边:</div>
                        {compareData.diff.edges.removed.map((edge) => (
                          <div key={edge.id} className="text-sm text-red-600 dark:text-red-400 ml-2">
                            - {edge.source} → {edge.target}
                          </div>
                        ))}
                      </div>
                    )}
                    {compareData.diff.edges.modified && compareData.diff.edges.modified.length > 0 && (
                      <div>
                        <div className="text-xs font-medium text-yellow-600 dark:text-yellow-400 mb-1">修改边:</div>
                        {compareData.diff.edges.modified.map((item) => (
                          <div key={item.id} className="text-sm ml-2 text-yellow-600 dark:text-yellow-400">
                            ~ {item.old.source} → {item.old.target} → {item.new.source} → {item.new.target}
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </div>
            </div>
          )}
        </Modal>
      </div>
  )
  }

  // 列表视图
  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header */}
        <div className="mb-8">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-4">
            <div className="flex-1 min-w-0 relative pl-4">
              <motion.div
                layoutId="pageTitleIndicator"
                className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-8 bg-primary rounded-r-full"
                transition={{ type: 'spring', bounce: 0.2, duration: 0.3 }}
              />
              <h1 className="text-2xl sm:text-3xl font-bold text-gray-900 dark:text-white mb-2">
                工作流管理
              </h1>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                创建和管理自动化工作流程
              </p>
            </div>
            {error && (
              <div className="flex items-center gap-2 px-4 py-2 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-red-700 dark:text-red-400">
                <AlertCircle className="w-4 h-4 flex-shrink-0" />
                <span className="text-sm break-words">{error}</span>
                <button
                  onClick={() => setError(null)}
                  className="ml-2 text-red-500 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300 flex-shrink-0"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
            )}
            <div className="flex flex-wrap items-center gap-2">
              <Button
                variant={viewMode === 'grid' ? 'primary' : 'outline'}
                size="sm"
                onClick={(e) => {
                  e.preventDefault()
                  e.stopPropagation()
                  setViewMode('grid')
                }}
              >
                网格
              </Button>
              <Button
                variant={viewMode === 'list' ? 'primary' : 'outline'}
                size="sm"
                onClick={(e) => {
                  e.preventDefault()
                  e.stopPropagation()
                  setViewMode('list')
                }}
              >
                列表
              </Button>
              <Button
                variant="primary"
                onClick={handleCreate}
              >
                创建工作流
              </Button>
            </div>
          </div>
        </div>

        {/* Filters */}
        <Card className="mb-6" padding="md">
          <div className="flex flex-col sm:flex-row gap-4">
            <div className="flex-1">
              <Input
                placeholder="搜索工作流..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                leftIcon={<Search className="w-4 h-4" />}
                clearable
              />
            </div>
            <div className="flex gap-2">
              <Button
                variant={selectedStatus === 'all' ? 'primary' : 'outline'}
                size="sm"
                onClick={() => setSelectedStatus('all')}
              >
                全部 ({workflows.length})
              </Button>
              <Button
                variant={selectedStatus === 'draft' ? 'primary' : 'outline'}
                size="sm"
                onClick={() => setSelectedStatus('draft')}
              >
                草稿 ({workflows.filter(w => w.status === 'draft').length})
              </Button>
              <Button
                variant={selectedStatus === 'active' ? 'primary' : 'outline'}
                size="sm"
                onClick={() => setSelectedStatus('active')}
              >
                激活 ({workflows.filter(w => w.status === 'active').length})
              </Button>
              <Button
                variant={selectedStatus === 'archived' ? 'primary' : 'outline'}
                size="sm"
                onClick={() => setSelectedStatus('archived')}
              >
                归档 ({workflows.filter(w => w.status === 'archived').length})
              </Button>
          </div>
        </div>
        </Card>

        {/* Workflow List */}
        {loading && workflows.length === 0 ? (
          <Card className="p-12">
            <div className="flex flex-col items-center justify-center">
              <div className="animate-spin rounded-full h-12 w-12 border-4 border-gray-300 border-t-blue-500 mb-4"></div>
              <p className="text-gray-500 dark:text-gray-400">加载中...</p>
            </div>
          </Card>
        ) : filteredWorkflows.length === 0 ? (
          <EmptyState
            icon={FileText}
            title="暂无工作流"
            description={searchTerm || selectedStatus !== 'all' 
              ? "没有找到匹配的工作流定义" 
              : "创建你的第一个工作流定义"}
            action={!searchTerm && selectedStatus === 'all' ? {
              label: '创建工作流',
              onClick: handleCreate
            } : undefined}
          />
        ) : viewMode === 'grid' ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {filteredWorkflows.map((workflow) => (
            <motion.div
              key={workflow.id}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
                whileHover={{ y: -4 }}
                transition={{ duration: 0.2 }}
              >
                <Card
                  hover
                  onClick={() => handleSelectWorkflow(workflow)}
                  className="cursor-pointer h-full"
            >
              <div className="flex items-start justify-between mb-4">
                    <div className="flex-1 min-w-0">
                      <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-1 truncate">
                        {workflow.name}
                      </h3>
                      <p className="text-sm text-gray-500 dark:text-gray-400 line-clamp-2">
                        {workflow.description || '无描述'}
                      </p>
                </div>
                    <div className="ml-4 flex-shrink-0">
                      {getStatusBadge(workflow.status)}
                </div>
              </div>
              
                  <div className="flex items-center gap-2 mb-4 flex-wrap">
                    {workflow.tags?.slice(0, 3).map((tag, idx) => (
                      <Badge key={idx} variant="outline" size="xs">
                        {tag}
                      </Badge>
                    ))}
                    {workflow.tags && workflow.tags.length > 3 && (
                      <Badge variant="muted" size="xs">+{workflow.tags.length - 3}</Badge>
                    )}
                    <Badge variant="muted" size="xs">v{workflow.version}</Badge>
                  </div>

                  <div className="flex items-center justify-between pt-4 border-t border-gray-200 dark:border-gray-700">
                    <div className="flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400">
                      <div className="flex items-center gap-1">
                        <GitBranch className="w-3 h-3" />
                        <span>{workflow.definition.nodes.length}</span>
                      </div>
                      <div className="flex items-center gap-1">
                        <FileText className="w-3 h-3" />
                        <span>{workflow.definition.edges.length}</span>
                      </div>
                    </div>
                    <div className="flex items-center gap-1">
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          handleEdit(workflow)
                        }}
                        className="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors"
                      >
                        <Edit className="w-3 h-3" />
                        编辑
                      </button>
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          handleDelete(workflow.id)
                        }}
                        className="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-colors"
                      >
                        <Trash2 className="w-3 h-3" />
                        删除
                      </button>
              </div>
                  </div>
                </Card>
            </motion.div>
          ))}
        </div>
        ) : (
          <Card padding="none">
            <div className="divide-y divide-gray-200 dark:divide-gray-700">
              {filteredWorkflows.map((workflow) => (
          <motion.div
                  key={workflow.id}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
                  whileHover={{ backgroundColor: 'rgba(0,0,0,0.02)' }}
                  className="p-4 cursor-pointer transition-colors"
                  onClick={() => handleSelectWorkflow(workflow)}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-4 flex-1 min-w-0">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <h3 className="text-base font-semibold text-gray-900 dark:text-white">
                            {workflow.name}
            </h3>
                          {getStatusBadge(workflow.status)}
                          <Badge variant="muted" size="xs">v{workflow.version}</Badge>
                        </div>
                        <p className="text-sm text-gray-500 dark:text-gray-400 truncate">
                          {workflow.description || '无描述'}
                        </p>
                        <div className="flex items-center gap-4 mt-2 text-xs text-gray-400">
                          <span>{formatDate(workflow.updatedAt)}</span>
                          <span>{workflow.definition.nodes.length} 节点</span>
                          <span>{workflow.definition.edges.length} 连接</span>
                        </div>
                      </div>
                      <div className="flex items-center gap-2 flex-shrink-0">
                        {workflow.tags?.slice(0, 2).map((tag, idx) => (
                          <Badge key={idx} variant="outline" size="xs">{tag}</Badge>
                        ))}
                      </div>
                    </div>
                    <div className="flex items-center gap-1 ml-4">
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          handleEdit(workflow)
                        }}
                        className="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors"
                      >
                        <Edit className="w-3 h-3" />
                        编辑
                      </button>
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          handleDelete(workflow.id)
                        }}
                        className="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-colors"
                      >
                        <Trash2 className="w-3 h-3" />
                        删除
                      </button>
                    </div>
                  </div>
          </motion.div>
              ))}
            </div>
          </Card>
        )}

        {/* Create/Edit Modal */}
        <Modal
          isOpen={isCreateModalOpen || isEditModalOpen}
          onClose={() => {
            setIsCreateModalOpen(false)
            setIsEditModalOpen(false)
            setEditingWorkflow(null)
          }}
          title={editingWorkflow ? '编辑工作流' : '创建工作流'}
          size="lg"
        >
          <WorkflowForm
            workflow={editingWorkflow}
            onSave={handleSave}
            saving={saving}
            onCancel={() => {
              setIsCreateModalOpen(false)
              setIsEditModalOpen(false)
              setEditingWorkflow(null)
              setError(null)
            }}
          />
        </Modal>
      </div>
      
      {/* 终端组件 */}
      <Terminal
        logs={terminalLogs}
        isVisible={isTerminalVisible}
        onClose={() => setIsTerminalVisible(false)}
        onClear={() => setTerminalLogs([])}
      />
    </div>
  )
}

// 触发器配置面板组件
interface TriggerConfigPanelProps {
  workflow: WorkflowDefinition | null
  triggerConfig: WorkflowTriggerConfig | undefined
  onUpdate: (config: WorkflowTriggerConfig) => void
  onSave: () => Promise<void>
  saving: boolean
}

const TriggerConfigPanel: React.FC<TriggerConfigPanelProps> = ({
  workflow,
  triggerConfig,
  onUpdate,
  onSave,
  saving
}) => {
  const [copiedKey, setCopiedKey] = useState(false)
  const [apiKeyVisible, setApiKeyVisible] = useState(false)

  // 确保 triggerConfig 不为 undefined
  const safeTriggerConfig: WorkflowTriggerConfig = triggerConfig || {}

  const generateAPIKey = () => {
    const key = Array.from(crypto.getRandomValues(new Uint8Array(32)))
      .map(b => b.toString(16).padStart(2, '0'))
      .join('')
    
    onUpdate({
      ...safeTriggerConfig,
      api: {
        ...safeTriggerConfig.api,
        enabled: true,
        apiKey: key,
        public: safeTriggerConfig.api?.public ?? false
      }
    })
  }

  const copyAPIKey = () => {
    if (safeTriggerConfig.api?.apiKey) {
      navigator.clipboard.writeText(safeTriggerConfig.api.apiKey)
      setCopiedKey(true)
      setTimeout(() => setCopiedKey(false), 2000)
    }
  }

  const getWebhookURL = () => {
    if (!workflow) return ''
    const baseURL = window.location.origin
    return `${baseURL}/api/public/workflows/webhook/${workflow.slug}`
  }

  return (
    <div className="space-y-6 max-h-[70vh] overflow-y-auto">
      {/* API 触发 */}
      <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Globe className="w-5 h-5 text-blue-500" />
            <h3 className="text-base font-semibold text-gray-900 dark:text-white">API 触发</h3>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={safeTriggerConfig.api?.enabled || false}
              onChange={(e) => onUpdate({
                ...safeTriggerConfig,
                api: {
                  ...safeTriggerConfig.api,
                  enabled: e.target.checked,
                  public: safeTriggerConfig.api?.public ?? false,
                  apiKey: safeTriggerConfig.api?.apiKey
                }
              })}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 dark:peer-focus:ring-blue-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-blue-600"></div>
          </label>
        </div>
        
        {safeTriggerConfig.api?.enabled && (
          <div className="space-y-4 mt-4">
            <div>
              <label className="flex items-center gap-2 mb-2">
                <input
                  type="checkbox"
                  checked={safeTriggerConfig.api?.public || false}
                  onChange={(e) => onUpdate({
                    ...safeTriggerConfig,
                    api: {
                      ...safeTriggerConfig.api,
                      enabled: true,
                      public: e.target.checked,
                      apiKey: safeTriggerConfig.api?.apiKey
                    }
                  })}
                  className="rounded border-gray-300"
                />
                <span className="text-sm text-gray-700 dark:text-gray-300">公开 API（不需要认证）</span>
              </label>
            </div>
            
            {safeTriggerConfig.api?.public && (
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <Input
                    type={apiKeyVisible ? 'text' : 'password'}
                    value={safeTriggerConfig.api?.apiKey || ''}
                    onChange={(e) => onUpdate({
                      ...safeTriggerConfig,
                      api: {
                        ...safeTriggerConfig.api,
                        enabled: true,
                        public: true,
                        apiKey: e.target.value
                      }
                    })}
                    placeholder="API 密钥"
                    className="flex-1"
                  />
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setApiKeyVisible(!apiKeyVisible)}
                    title={apiKeyVisible ? '隐藏密钥' : '显示密钥'}
                  >
                    {apiKeyVisible ? '隐藏' : '显示'}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={generateAPIKey}
                  >
                    生成
                  </Button>
                  {safeTriggerConfig.api?.apiKey && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={copyAPIKey}
                    >
                      {copiedKey ? '已复制' : '复制'}
                    </Button>
                  )}
                </div>
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  API 地址: <code className="bg-gray-100 dark:bg-gray-800 px-1 rounded">POST /api/public/workflows/{workflow?.slug}/execute</code>
                </p>
              </div>
            )}
          </div>
        )}
      </div>

      {/* 事件触发 */}
      <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Zap className="w-5 h-5 text-yellow-500" />
            <h3 className="text-base font-semibold text-gray-900 dark:text-white">事件触发</h3>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={safeTriggerConfig.event?.enabled || false}
              onChange={(e) => onUpdate({
                ...safeTriggerConfig,
                event: {
                  ...safeTriggerConfig.event,
                  enabled: e.target.checked,
                  events: safeTriggerConfig.event?.events || []
                }
              })}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 dark:peer-focus:ring-blue-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-blue-600"></div>
          </label>
        </div>
        
        {safeTriggerConfig.event?.enabled && (
          <div className="space-y-4 mt-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                监听的事件类型（每行一个，支持通配符 *）
              </label>
              <textarea
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm"
                rows={4}
                value={safeTriggerConfig.event?.events?.join('\n') || ''}
                onChange={(e) => onUpdate({
                  ...safeTriggerConfig,
                  event: {
                    ...safeTriggerConfig.event,
                    enabled: true,
                    events: e.target.value.split('\n').filter(s => s.trim())
                  }
                })}
                placeholder="user.created&#10;order.paid&#10;*"
              />
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                示例: user.created, order.paid, * (监听所有事件)
              </p>
            </div>
          </div>
        )}
      </div>

      {/* 定时触发 */}
      <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Calendar className="w-5 h-5 text-green-500" />
            <h3 className="text-base font-semibold text-gray-900 dark:text-white">定时触发</h3>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={safeTriggerConfig.schedule?.enabled || false}
              onChange={(e) => onUpdate({
                ...safeTriggerConfig,
                schedule: {
                  ...safeTriggerConfig.schedule,
                  enabled: e.target.checked,
                  cronExpr: safeTriggerConfig.schedule?.cronExpr || '0 0 * * *'
                }
              })}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 dark:peer-focus:ring-blue-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-blue-600"></div>
          </label>
        </div>
        
        {safeTriggerConfig.schedule?.enabled && (
          <div className="space-y-4 mt-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Cron 表达式
              </label>
              <Input
                value={safeTriggerConfig.schedule?.cronExpr || ''}
                onChange={(e) => onUpdate({
                  ...safeTriggerConfig,
                  schedule: {
                    ...safeTriggerConfig.schedule,
                    enabled: true,
                    cronExpr: e.target.value
                  }
                })}
                placeholder="0 0 * * *"
              />
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                格式: 秒 分 时 日 月 星期。示例: 0 0 * * * (每天0点), 0 */30 * * * * (每30分钟)
              </p>
            </div>
          </div>
        )}
      </div>

      {/* Webhook 触发 */}
      <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Webhook className="w-5 h-5 text-purple-500" />
            <h3 className="text-base font-semibold text-gray-900 dark:text-white">Webhook 触发</h3>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={safeTriggerConfig.webhook?.enabled || false}
              onChange={(e) => onUpdate({
                ...safeTriggerConfig,
                webhook: {
                  ...safeTriggerConfig.webhook,
                  enabled: e.target.checked,
                  secret: safeTriggerConfig.webhook?.secret
                }
              })}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 dark:peer-focus:ring-blue-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-blue-600"></div>
          </label>
        </div>
        
        {safeTriggerConfig.webhook?.enabled && (
          <div className="space-y-4 mt-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Webhook URL
              </label>
              <div className="flex items-center gap-2">
                <Input
                  value={getWebhookURL()}
                  readOnly
                  className="flex-1"
                />
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    navigator.clipboard.writeText(getWebhookURL())
                    showAlert('Webhook URL 已复制到剪贴板', 'success', '已复制')
                  }}
                >
                  复制
                </Button>
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Webhook 密钥（可选，用于验证）
              </label>
              <Input
                type="password"
                value={safeTriggerConfig.webhook?.secret || ''}
                onChange={(e) => onUpdate({
                  ...safeTriggerConfig,
                  webhook: {
                    ...safeTriggerConfig.webhook,
                    enabled: true,
                    secret: e.target.value
                  }
                })}
                placeholder="留空则不验证"
              />
            </div>
          </div>
        )}
      </div>

      {/* 智能体触发 */}
      <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Bot className="w-5 h-5 text-indigo-500" />
            <h3 className="text-base font-semibold text-gray-900 dark:text-white">智能体触发</h3>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={safeTriggerConfig.assistant?.enabled || false}
              onChange={(e) => onUpdate({
                ...safeTriggerConfig,
                assistant: {
                  ...safeTriggerConfig.assistant,
                  enabled: e.target.checked,
                  assistantIds: safeTriggerConfig.assistant?.assistantIds || [],
                  intents: safeTriggerConfig.assistant?.intents || []
                }
              })}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 dark:peer-focus:ring-blue-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-blue-600"></div>
          </label>
        </div>
        
        {safeTriggerConfig.assistant?.enabled && (
          <div className="space-y-4 mt-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                描述（用于智能体识别）
              </label>
              <textarea
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm"
                rows={4}
                value={safeTriggerConfig.assistant?.description || ''}
                onChange={(e) => onUpdate({
                  ...safeTriggerConfig,
                  assistant: {
                    ...safeTriggerConfig.assistant,
                    enabled: true,
                    description: e.target.value
                  }
                })}
                placeholder="描述此工作流的用途、适用场景和调用时机，帮助智能体决定何时调用此工作流。&#10;&#10;示例：&#10;• 处理用户订单：当用户需要下单、查询订单状态或取消订单时调用&#10;• 发送邮件通知：当需要向用户发送邮件通知时调用&#10;• 数据分析和报告：当用户需要生成数据报告或进行数据分析时调用"
              />
              <div className="mt-2 p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-md">
                <p className="text-xs font-medium text-blue-900 dark:text-blue-300 mb-1 flex items-center gap-1">
                  <Lightbulb className="w-3 h-3" />
                  工作原理：
                </p>
                <ul className="text-xs text-blue-800 dark:text-blue-400 space-y-1 list-disc list-inside">
                  <li>启用后，此工作流会被注册为智能体的工具（类似天气查询、计算器等）</li>
                  <li>智能体在对话中会根据用户意图和此描述，自动决定是否调用此工作流</li>
                  <li>描述越清晰，智能体越能准确判断何时调用</li>
                  <li>建议包含：工作流功能、适用场景、调用时机</li>
                </ul>
              </div>
            </div>
            <div className="p-3 bg-gray-50 dark:bg-gray-800 rounded-md">
              <p className="text-xs font-medium text-gray-700 dark:text-gray-300 mb-2">📝 填写示例：</p>
              <div className="space-y-2 text-xs text-gray-600 dark:text-gray-400">
                <div>
                  <strong className="text-gray-900 dark:text-white">订单处理工作流：</strong>
                  <p className="mt-1">处理用户订单相关操作。当用户需要创建订单、查询订单状态、取消订单或处理退款时调用。需要提供订单ID、用户ID等参数。</p>
                </div>
                <div>
                  <strong className="text-gray-900 dark:text-white">邮件发送工作流：</strong>
                  <p className="mt-1">向指定邮箱发送邮件。当用户需要发送通知邮件、确认邮件或营销邮件时调用。需要提供收件人邮箱、邮件主题和内容。</p>
                </div>
                <div>
                  <strong className="text-gray-900 dark:text-white">数据分析工作流：</strong>
                  <p className="mt-1">生成数据报告和图表。当用户需要查看销售数据、用户统计或生成业务报告时调用。可以指定时间范围和数据类型。</p>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* 保存按钮 */}
      <div className="flex justify-end gap-2 pt-4 border-t border-gray-200 dark:border-gray-700">
        <Button
          variant="outline"
          onClick={() => {/* 关闭模态框由父组件处理 */}}
          disabled={saving}
        >
          取消
        </Button>
        <Button
          variant="primary"
          onClick={onSave}
          loading={saving}
          disabled={saving}
        >
          保存配置
        </Button>
      </div>
    </div>
  )
}

// 工作流表单组件
interface WorkflowFormProps {
  workflow?: WorkflowDefinition | null
  onSave: (data: Partial<WorkflowDefinition>) => Promise<void>
  onCancel: () => void
  saving?: boolean
}

const WorkflowForm: React.FC<WorkflowFormProps> = ({ workflow, onSave, onCancel, saving = false }) => {
  const [formData, setFormData] = useState({
    name: workflow?.name || '',
    slug: workflow?.slug || '',
    description: workflow?.description || '',
    status: workflow?.status || 'draft' as WorkflowStatus,
    tags: workflow?.tags?.join(', ') || ''
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    await onSave({
      ...formData,
      tags: formData.tags.split(',').map(t => t.trim()).filter(Boolean),
      definition: workflow?.definition || { nodes: [], edges: [] }
    })
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <Input
        label="名称"
        value={formData.name}
        onChange={(e) => setFormData({ ...formData, name: e.target.value })}
        required
      />
      <Input
        label="Slug"
        value={formData.slug}
        onChange={(e) => setFormData({ ...formData, slug: e.target.value })}
        required
        helperText="唯一标识符，用于URL"
      />
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          描述
        </label>
        <textarea
          className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
          rows={3}
          value={formData.description}
          onChange={(e) => setFormData({ ...formData, description: e.target.value })}
        />
      </div>
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          状态
        </label>
        <select
          className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
          value={formData.status}
          onChange={(e) => setFormData({ ...formData, status: e.target.value as WorkflowStatus })}
        >
          <option value="draft">草稿</option>
          <option value="active">激活</option>
          <option value="archived">归档</option>
        </select>
      </div>
      <Input
        label="标签 (逗号分隔)"
        value={formData.tags}
        onChange={(e) => setFormData({ ...formData, tags: e.target.value })}
        helperText="使用逗号分隔多个标签"
      />
      <div className="flex justify-end gap-2 pt-4 border-t border-gray-200 dark:border-gray-700">
        <Button variant="outline" onClick={onCancel} disabled={saving}>
          取消
        </Button>
        <Button variant="primary" type="submit" loading={saving} disabled={saving}>
          保存
        </Button>
      </div>
    </form>
  )
}

export default WorkflowManager

