import { useCallback } from 'react'
import type { WorkflowNode, WorkflowConnection } from '../types/workflow'

interface UseConnectionOperationsProps {
  nodes: WorkflowNode[]
  connections: WorkflowConnection[]
  setConnections: (connections: WorkflowConnection[] | ((prev: WorkflowConnection[]) => WorkflowConnection[])) => void
  isConnecting: boolean
  connectionStart: { nodeId: string; handle: string } | null
  setIsConnecting: (isConnecting: boolean) => void
  setConnectionStart: (connectionStart: { nodeId: string; handle: string } | null) => void
}

/**
 * 连接操作 hooks
 * 包含创建、删除、验证连接等操作
 */
export const useConnectionOperations = ({
  nodes,
  connections,
  setConnections,
  isConnecting,
  connectionStart,
  setIsConnecting,
  setConnectionStart
}: UseConnectionOperationsProps) => {
  // 开始连接
  const startConnection = useCallback((nodeId: string, handle: string) => {
    setIsConnecting(true)
    setConnectionStart({ nodeId, handle })
  }, [setIsConnecting, setConnectionStart])

  // 完成连接
  const completeConnection = useCallback((nodeId: string, handle: string) => {
    if (isConnecting && connectionStart && connectionStart.nodeId !== nodeId) {
      // 检查源输出点是否已经连接
      const sourceAlreadyConnected = connections.some(conn => 
        conn.source === connectionStart.nodeId && conn.sourceHandle === connectionStart.handle
      )
      
      if (sourceAlreadyConnected) {
        console.log('输出点已经连接，无法再次连接')
        setIsConnecting(false)
        setConnectionStart(null)
        return
      }
      
      // 检查是否已存在完全相同的连接
      const existingConnection = connections.find(conn => 
        conn.source === connectionStart.nodeId && 
        conn.target === nodeId &&
        conn.sourceHandle === connectionStart.handle &&
        conn.targetHandle === handle
      )
      
      if (!existingConnection) {
        // 根据源节点类型和 sourceHandle 确定边的类型
        const sourceNode = nodes.find(n => n.id === connectionStart.nodeId)
        let edgeType: 'default' | 'true' | 'false' | 'error' | 'branch' | undefined = 'default'
        
        if (sourceNode) {
          if (sourceNode.type === 'gateway') {
            const outputIndex = sourceNode.outputs.findIndex(o => o === connectionStart.handle)
            if (outputIndex === 0) {
              edgeType = 'true'
            } else if (outputIndex === 1) {
              edgeType = 'false'
            }
          } else if (sourceNode.type === 'parallel') {
            edgeType = 'branch'
          }
        }
        
        const newConnection: WorkflowConnection = {
          id: `conn-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
          source: connectionStart.nodeId,
          target: nodeId,
          sourceHandle: connectionStart.handle,
          targetHandle: handle,
          type: edgeType
        }
        setConnections(prev => [...prev, newConnection])
      } else {
        console.log('连接已存在，跳过创建')
      }
    }
    setIsConnecting(false)
    setConnectionStart(null)
  }, [isConnecting, connectionStart, connections, nodes, setConnections, setIsConnecting, setConnectionStart])

  // 取消连接
  const cancelConnection = useCallback(() => {
    setIsConnecting(false)
    setConnectionStart(null)
  }, [setIsConnecting, setConnectionStart])

  return {
    startConnection,
    completeConnection,
    cancelConnection
  }
}
