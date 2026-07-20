import React from 'react'
import { motion } from 'framer-motion'
import { Copy, Play, Settings, Trash2 } from 'lucide-react'
import { cn } from '@/utils/utils'
import { Button } from '@/components/ui'
import type { WorkflowNode } from '../types/workflow'
import type { NodeTypeConfig } from '../utils/nodeTypeConfig'
import { getNodeCanvasWidth } from '../canvas/canvasStyles'
import { getPortOffsets } from '../canvas/portOffsets'

export interface WorkflowCanvasNodeProps {
  node: WorkflowNode
  nodeConfig: NodeTypeConfig
  selected: boolean
  dragged: boolean
  isConnecting: boolean
  connectionStartNodeId?: string | null
  getIconComponent: (iconName: string) => React.ReactNode
  inlineConfig?: React.ReactNode
  onMouseDown: (e: React.MouseEvent, nodeId: string) => void
  onCopy: (nodeId: string) => void
  onDebug: (nodeId: string) => void
  onConfigure: (nodeId: string) => void
  onDelete: (nodeId: string) => void
  onCompleteConnection: (nodeId: string, handle: string) => void
  onStartConnection: (nodeId: string, handle: string) => void
  isOutputConnected: (nodeId: string, handle: string) => boolean
}

export const WorkflowCanvasNode: React.FC<WorkflowCanvasNodeProps> = ({
  node,
  nodeConfig,
  selected,
  dragged,
  isConnecting,
  connectionStartNodeId,
  getIconComponent,
  inlineConfig,
  onMouseDown,
  onCopy,
  onDebug,
  onConfigure,
  onDelete,
  onCompleteConnection,
  onStartConnection,
  isOutputConnected,
}) => {
  const width = getNodeCanvasWidth(node.type)
  const canDebug = node.type !== 'start' && node.type !== 'end'

  return (
    <motion.div
      className={cn('absolute cursor-move select-none group', dragged ? 'z-50' : 'z-10')}
      style={{ left: node.position.x, top: node.position.y, width }}
      onMouseDown={(e) => onMouseDown(e, node.id)}
      initial={{ opacity: 0, scale: 0.98 }}
      animate={{ opacity: 1, scale: dragged ? 1.01 : 1 }}
      transition={{ duration: 0.12 }}
    >
      <div
        className={cn(
          'relative overflow-hidden rounded-xl border bg-white shadow-sm transition-all dark:bg-[var(--color-bg-2)]',
          selected
            ? 'border-[rgb(var(--primary-6))] shadow-md ring-2 ring-[rgb(var(--primary-6))]/15'
            : 'border-[var(--color-border-2)] hover:border-[rgb(var(--primary-3))] hover:shadow',
          dragged && 'shadow-lg',
        )}
      >
        {/* header */}
        <div className="border-b border-[var(--color-border-1)] px-3.5 py-3">
          <div className="flex items-start gap-3">
            <div
              className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-white shadow-sm"
              style={{ background: `linear-gradient(135deg, ${nodeConfig.color}, ${nodeConfig.color}cc)` }}
            >
              <div className="scale-90">{getIconComponent(nodeConfig.iconName)}</div>
            </div>
            <div className="min-w-0 flex-1">
              <div className="truncate text-sm font-semibold text-[var(--color-text-1)]">
                {node.data.label}
              </div>
              <div className="mt-0.5 truncate text-xs text-[var(--color-text-3)]">
                {nodeConfig.label}
              </div>
            </div>
            <div
              className={cn(
                'flex shrink-0 items-center gap-0.5 transition-opacity',
                selected ? 'opacity-100' : 'opacity-0 group-hover:opacity-100',
              )}
            >
              <Button
                type="text"
                size="mini"
                icon={<Copy className="h-3.5 w-3.5" />}
                title="复制节点"
                onClick={(e) => {
                  e.stopPropagation()
                  onCopy(node.id)
                }}
              />
              {canDebug ? (
                <Button
                  type="text"
                  size="mini"
                  icon={<Play className="h-3.5 w-3.5" />}
                  title="单节点调试"
                  onClick={(e) => {
                    e.stopPropagation()
                    onDebug(node.id)
                  }}
                />
              ) : null}
              <Button
                type="text"
                size="mini"
                icon={<Settings className="h-3.5 w-3.5" />}
                title="配置"
                onClick={(e) => {
                  e.stopPropagation()
                  onConfigure(node.id)
                }}
              />
              <Button
                type="text"
                size="mini"
                status="danger"
                icon={<Trash2 className="h-3.5 w-3.5" />}
                title="删除"
                onClick={(e) => {
                  e.stopPropagation()
                  onDelete(node.id)
                }}
              />
            </div>
          </div>
        </div>

        {/* body */}
        <div className="space-y-2.5 px-3.5 py-3">
          {inlineConfig}

          {!inlineConfig && node.type === 'start' && node.inputs.length > 0 ? (
            <ParamChipSection title="输入" items={node.inputs} tone="green" />
          ) : null}

          {!inlineConfig && node.type === 'end' && node.outputs.length > 0 ? (
            <ParamChipSection title="输出" items={node.outputs} tone="blue" />
          ) : null}

          {!inlineConfig && node.type !== 'start' && node.type !== 'end' && (node.inputs.length > 0 || node.outputs.length > 0) ? (
            <div className="space-y-2">
              {node.inputs.length > 0 ? (
                <ParamChipSection title="输入" items={node.inputs} tone="green" />
              ) : null}
              {node.outputs.length > 0 && node.type !== 'gateway' ? (
                <ParamChipSection title="输出" items={node.outputs} tone="blue" />
              ) : null}
            </div>
          ) : null}
        </div>
      </div>

      {/* input handles */}
      {node.inputs.map((input, index) => {
        const offsets = getPortOffsets(node.inputs.length, 44)
        const top = offsets[index] ?? 44
        const active = isConnecting && connectionStartNodeId !== node.id
        return (
          <div
            key={input}
            className="absolute z-30"
            style={{ left: -5, top }}
            onMouseDown={(e) => {
              e.stopPropagation()
              if (isConnecting) onCompleteConnection(node.id, input)
            }}
          >
            <div
              className={cn(
                'h-2.5 w-2.5 cursor-pointer rounded-full border-2 border-white bg-[#93c5fd] shadow-sm transition-transform dark:border-[var(--color-bg-2)]',
                active && 'scale-125 ring-2 ring-[#93c5fd]/40',
              )}
              title={`输入: ${input}`}
            />
          </div>
        )
      })}

      {/* output handles */}
      {node.outputs.map((output, index) => {
        const offsets = getPortOffsets(node.outputs.length, 44)
        const top = offsets[index] ?? 44
        const connected = isOutputConnected(node.id, output)
        const canStart = !connected && !isConnecting
        return (
          <div key={output} className="absolute z-30" style={{ right: -5, top }}>
            <div
              className={cn(
                'h-2.5 w-2.5 rounded-full border-2 border-white shadow-sm transition-transform dark:border-[var(--color-bg-2)]',
                connected ? 'cursor-not-allowed bg-[var(--color-fill-3)]' : 'cursor-pointer bg-[#93c5fd] hover:scale-110',
                isConnecting && connectionStartNodeId === node.id && 'ring-2 ring-[#93c5fd]/40',
              )}
              onMouseDown={(e) => {
                e.stopPropagation()
                if (canStart) onStartConnection(node.id, output)
              }}
              title={connected ? `已连接: ${output}` : `输出: ${output}`}
            />
          </div>
        )
      })}
    </motion.div>
  )
}

const ParamChipSection: React.FC<{
  title: string
  items: string[]
  tone: 'green' | 'blue'
}> = ({ title, items, tone }) => (
  <div className="rounded-lg bg-[var(--color-fill-1)] px-2.5 py-2">
    <div className="mb-1.5 flex items-center gap-1.5">
      <span className={cn('h-2.5 w-0.5 rounded-full', tone === 'green' ? 'bg-[rgb(var(--success-6))]' : 'bg-[rgb(var(--primary-6))]')} />
      <span className="text-[11px] font-medium text-[var(--color-text-2)]">{title}</span>
    </div>
    <div className="flex flex-wrap gap-1">
      {items.slice(0, 4).map((item) => (
        <span
          key={item}
          className="max-w-[120px] truncate rounded bg-white px-1.5 py-0.5 text-[10px] text-[var(--color-text-2)] dark:bg-[var(--color-bg-3)]"
          title={item}
        >
          {item}
        </span>
      ))}
      {items.length > 4 ? (
        <span className="text-[10px] text-[var(--color-text-3)]">+{items.length - 4}</span>
      ) : null}
    </div>
  </div>
)
