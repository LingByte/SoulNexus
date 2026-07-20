import { useCallback } from 'react'
import type { WorkflowNode, WorkflowConnection } from '../types/workflow'
import { getDefaultNodeConfig } from '../utils/defaultNodeConfig'

interface UseNodeOperationsProps {
  nodes: WorkflowNode[]
  connections: WorkflowConnection[]
  setNodes: (nodes: WorkflowNode[] | ((prev: WorkflowNode[]) => WorkflowNode[])) => void
  setConnections: (connections: WorkflowConnection[] | ((prev: WorkflowConnection[]) => WorkflowConnection[])) => void
  selectedNode: string | null
  setSelectedNode: (nodeId: string | null) => void
  selectedConnection: string | null
  setSelectedConnection: (connectionId: string | null) => void
  NODE_TYPES: Record<string, any>
}

/**
 * 节点操作 hooks
 * 包含添加、删除、更新节点等操作
 */
export const useNodeOperations = ({
  nodes,
  connections,
  setNodes,
  setConnections,
  selectedNode,
  setSelectedNode,
  selectedConnection,
  setSelectedConnection,
  NODE_TYPES
}: UseNodeOperationsProps) => {
  // 添加节点
  const addNode = useCallback((type: WorkflowNode['type'], position: { x: number; y: number }) => {
    const defaultConfig = getDefaultNodeConfig(type)
    const nodeId = `node-${Date.now()}`
    let inputs = Array(NODE_TYPES[type].inputs).fill('').map((_, i) => `input-${i}`)
    let outputs = Array(NODE_TYPES[type].outputs).fill('').map((_, i) => `output-${i}`)
    if (type === 'knowledge_base') {
      inputs = [String(defaultConfig.inputVariable || 'query')]
      outputs = [String(defaultConfig.outputVariable || 'kb_result')]
    } else if (type === 'ai_chat') {
      inputs = [String(defaultConfig.inputVariable || 'user_input')]
      outputs = [String(defaultConfig.outputVariable || 'ai_response')]
    }
    const newNode: WorkflowNode = {
      id: nodeId,
      type,
      position,
      data: {
        label: NODE_TYPES[type].label,
        config: defaultConfig
      },
      inputs,
      outputs,
    }
    setNodes(prev => [...prev, newNode])
    setSelectedNode(nodeId)
    return nodeId
  }, [NODE_TYPES, setNodes, setSelectedNode])

  // 添加插件节点
  const addPluginNode = useCallback((plugin: any, position: { x: number; y: number }) => {
    const nodeId = `plugin-${Date.now()}`
    const newNode: WorkflowNode = {
      id: nodeId,
      type: 'workflow_plugin',
      position,
      data: {
        label: plugin.plugin?.displayName || plugin.plugin?.name || '插件节点',
        config: {
          pluginId: plugin.pluginId,
          parameters: {}
        },
        pluginId: plugin.pluginId
      },
      inputs: plugin.plugin?.inputSchema?.parameters?.map((p: any, i: number) => p.name || `input-${i}`) || ['input-0'],
      outputs: plugin.plugin?.outputSchema?.parameters?.map((p: any, i: number) => p.name || `output-${i}`) || ['output-0']
    }
    setNodes(prev => [...prev, newNode])
    setSelectedNode(nodeId)
    return nodeId
  }, [setNodes, setSelectedNode])

  // 删除节点
  const deleteNode = useCallback((nodeId: string) => {
    setNodes(prev => prev.filter(node => node.id !== nodeId))
    setConnections(prev => prev.filter(conn => 
      conn.source !== nodeId && conn.target !== nodeId
    ))
    if (selectedNode === nodeId) {
      setSelectedNode(null)
    }
  }, [selectedNode, setNodes, setConnections, setSelectedNode])

  const copyNode = useCallback((nodeId: string) => {
    const source = nodes.find((n) => n.id === nodeId)
    if (!source) return null
    const newId = `${source.type.replace(/_/g, '-')}-${Date.now()}`
    const copy: WorkflowNode = {
      ...source,
      id: newId,
      position: { x: source.position.x + 48, y: source.position.y + 48 },
      data: {
        ...source.data,
        label: `${source.data.label} (副本)`,
      },
    }
    setNodes((prev) => [...prev, copy])
    setSelectedNode(newId)
    return newId
  }, [nodes, setNodes, setSelectedNode])

  // 更新节点位置
  const updateNodePosition = useCallback((nodeId: string, position: { x: number; y: number }) => {
    setNodes(prev => prev.map(node => 
      node.id === nodeId ? { ...node, position } : node
    ))
  }, [setNodes])

  // 更新节点配置
  const updateNodeConfig = useCallback((nodeId: string, config: any) => {
    setNodes(prev => prev.map(node => {
      if (node.id === nodeId) {
        const updatedNode = { ...node, data: { ...node.data, config } }
        
        // 对于 AI 对话节点，根据 inputVariable 和 outputVariable 更新 inputs 和 outputs
        if (node.type === 'ai_chat') {
          const inputVar = config?.inputVariable
          const outputVar = config?.outputVariable
          updatedNode.inputs = inputVar ? [inputVar] : ['input-0']
          updatedNode.outputs = outputVar ? [outputVar] : ['output-0']
        }

        if (node.type === 'knowledge_base') {
          const inputVar = config?.inputVariable
          const outputVar = config?.outputVariable
          updatedNode.inputs = inputVar ? [inputVar] : ['query']
          updatedNode.outputs = outputVar ? [outputVar] : ['kb_result']
        }
        
        return updatedNode
      }
      return node
    }))
  }, [setNodes])

  // 删除连接
  const deleteConnection = useCallback((connectionId: string) => {
    setConnections(prev => prev.filter(conn => conn.id !== connectionId))
    if (selectedConnection === connectionId) {
      setSelectedConnection(null)
    }
  }, [selectedConnection, setConnections, setSelectedConnection])

  // 检查输出点是否已经有连接
  const isOutputConnected = useCallback((nodeId: string, outputHandle: string) => {
    return connections.some(conn => 
      conn.source === nodeId && conn.sourceHandle === outputHandle
    )
  }, [connections])

  return {
    addNode,
    addPluginNode,
    deleteNode,
    copyNode,
    updateNodePosition,
    updateNodeConfig,
    deleteConnection,
    isOutputConnected
  }
}
