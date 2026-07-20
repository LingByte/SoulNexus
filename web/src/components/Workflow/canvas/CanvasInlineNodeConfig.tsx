import React from 'react'
import type { WorkflowNode } from '../types/workflow'
import { CONFIG_NODE_TYPES } from '../canvas/canvasStyles'
import { AIChatNodeConfigBody, KnowledgeBaseNodeConfigBody } from '../node-configs/InlineNodeConfigs'

export function CanvasInlineNodeConfig({
  node,
  onConfigChange,
}: {
  node: WorkflowNode
  onConfigChange: (config: Record<string, unknown>) => void
}) {
  if (!CONFIG_NODE_TYPES.has(node.type)) return null

  if (node.type === 'knowledge_base') {
    return <KnowledgeBaseNodeConfigBody node={node} onConfigChange={onConfigChange} compact />
  }
  if (node.type === 'ai_chat') {
    return <AIChatNodeConfigBody node={node} onConfigChange={onConfigChange} compact />
  }

  return null
}
