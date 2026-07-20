import React from 'react'
import type { WorkflowNode } from '../types/workflow'
import type { NodeTypeConfig } from '../utils/nodeTypeConfig'
import { WorkflowCanvasNode } from './WorkflowCanvasNode'
import { CanvasInlineNodeConfig } from './CanvasInlineNodeConfig'
import { CONFIG_NODE_TYPES } from './canvasStyles'

export interface WorkflowCanvasNodesProps {
  nodes: WorkflowNode[]
  nodeTypes: Record<string, NodeTypeConfig>
  selectedNode: string | null
  draggedNode: string | null
  isConnecting: boolean
  connectionStart: { nodeId: string; handle: string } | null
  getIconComponent: (iconName: string) => React.ReactNode
  onNodeMouseDown: (e: React.MouseEvent, nodeId: string) => void
  onCopy: (nodeId: string) => void
  onDebug: (nodeId: string) => void
  onConfigure: (nodeId: string) => void
  onDelete: (nodeId: string) => void
  onInlineConfigChange: (nodeId: string, config: Record<string, unknown>) => void
  onCompleteConnection: (nodeId: string, handle: string) => void
  onStartConnection: (nodeId: string, handle: string) => void
  isOutputConnected: (nodeId: string, handle: string) => boolean
}

export const WorkflowCanvasNodes: React.FC<WorkflowCanvasNodesProps> = ({
  nodes,
  nodeTypes,
  selectedNode,
  draggedNode,
  isConnecting,
  connectionStart,
  getIconComponent,
  onNodeMouseDown,
  onCopy,
  onDebug,
  onConfigure,
  onDelete,
  onInlineConfigChange,
  onCompleteConnection,
  onStartConnection,
  isOutputConnected,
}) => (
  <>
    {nodes.map((node) => {
      const nodeConfig = nodeTypes[node.type]
      if (!nodeConfig) {
        console.warn(`Unknown node type: ${node.type}`)
        return null
      }

      const showInline = CONFIG_NODE_TYPES.has(node.type)

      return (
        <WorkflowCanvasNode
          key={node.id}
          node={node}
          nodeConfig={nodeConfig}
          selected={selectedNode === node.id}
          dragged={draggedNode === node.id}
          isConnecting={isConnecting}
          connectionStartNodeId={connectionStart?.nodeId}
          getIconComponent={getIconComponent}
          onMouseDown={onNodeMouseDown}
          onCopy={onCopy}
          onDebug={onDebug}
          onConfigure={onConfigure}
          onDelete={onDelete}
          onCompleteConnection={onCompleteConnection}
          onStartConnection={onStartConnection}
          isOutputConnected={isOutputConnected}
          inlineConfig={
            showInline ? (
              <CanvasInlineNodeConfig
                node={node}
                onConfigChange={(config) => onInlineConfigChange(node.id, config)}
              />
            ) : undefined
          }
        />
      )
    })}
  </>
)
