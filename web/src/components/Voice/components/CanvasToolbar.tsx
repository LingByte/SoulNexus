import React from 'react'
import { motion } from 'framer-motion'
import { Plus, AlertCircle, Trash2, Square, Save, FileText } from 'lucide-react'
import Button from '@/components/UI/Button'

interface CanvasToolbarProps {
  validation: { valid: boolean; message: string }
  onAddNode: () => void
  onHelp: () => void
  canvasScale: number
  onZoomIn: () => void
  onZoomOut: () => void
  onResetView: () => void
  onCenterNodes: () => void
  selectedConnection: string | null
  onDeleteConnection: () => void
  isRunning: boolean
  onRun: () => void
  onStop: () => void
  onSave: () => void
  t: (key: string) => string
}

/**
 * 画布工具栏组件
 * 包含缩放、重置、居中、运行等操作
 */
export const CanvasToolbar: React.FC<CanvasToolbarProps> = ({
  validation,
  onAddNode,
  onHelp,
  canvasScale,
  onZoomIn,
  onZoomOut,
  onResetView,
  onCenterNodes,
  selectedConnection,
  onDeleteConnection,
  isRunning,
  onRun,
  onStop,
  onSave,
  t
}) => {
  return (
    <div className="flex items-center justify-between p-4 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
      {/* 左侧：验证状态和操作按钮 */}
      <div className="flex items-center space-x-4">
        {!validation.valid && (
          <div className="flex items-center text-red-600 text-sm">
            <AlertCircle className="w-4 h-4" />
            <span className="ml-1">{validation.message}</span>
          </div>
        )}
        
        <div className="flex items-center gap-2">
          <button
            onClick={onAddNode}
            className="inline-flex items-center gap-2 px-3 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md transition-colors"
          >
            <Plus className="w-4 h-4" />
            <span>{t('workflow.editor.addNode')}</span>
          </button>
          
          <Button
            variant="ghost"
            size="icon"
            onClick={onHelp}
            title={t('workflow.editor.help')}
          >
            <FileText className="w-5 h-5" />
          </Button>
        </div>
      </div>
      
      {/* 右侧：画布控制和主要操作 */}
      <div className="flex items-center space-x-3">
        {/* 缩放控制 */}
        <ZoomControls
          scale={canvasScale}
          onZoomIn={onZoomIn}
          onZoomOut={onZoomOut}
        />

        {/* 视图控制 */}
        <ViewControls
          onResetView={onResetView}
          onCenterNodes={onCenterNodes}
        />

        {/* 分隔线 */}
        <div className="h-6 w-px bg-gray-300 dark:bg-gray-600"></div>

        {/* 连接删除按钮 */}
        {selectedConnection && (
          <Button
            variant="destructive"
            size="sm"
            onClick={onDeleteConnection}
          >
            <Trash2 className="w-4 h-4" />
            删除连接
          </Button>
        )}
        
        {/* 主要操作按钮 */}
        <ActionButtons
          isRunning={isRunning}
          onRun={onRun}
          onStop={onStop}
          onSave={onSave}
        />
      </div>
    </div>
  )
}

/**
 * 缩放控制组件
 */
interface ZoomControlsProps {
  scale: number
  onZoomIn: () => void
  onZoomOut: () => void
}

const ZoomControls: React.FC<ZoomControlsProps> = ({ scale, onZoomIn, onZoomOut }) => {
  return (
    <div className="flex items-center space-x-1 bg-gray-100 dark:bg-gray-700 rounded-lg p-1">
      <motion.button
        onClick={onZoomOut}
        className="p-1.5 text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-white dark:hover:bg-gray-600 rounded transition-colors"
        whileHover={{ scale: 1.05 }}
        whileTap={{ scale: 0.95 }}
        title="缩小"
      >
        <span className="text-sm font-bold">-</span>
      </motion.button>
      
      <span className="text-xs text-gray-600 dark:text-gray-400 min-w-[2.5rem] text-center px-1">
        {Math.round(scale * 100)}%
      </span>
      
      <motion.button
        onClick={onZoomIn}
        className="p-1.5 text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-white dark:hover:bg-gray-600 rounded transition-colors"
        whileHover={{ scale: 1.05 }}
        whileTap={{ scale: 0.95 }}
        title="放大"
      >
        <span className="text-sm font-bold">+</span>
      </motion.button>
    </div>
  )
}

/**
 * 视图控制组件
 */
interface ViewControlsProps {
  onResetView: () => void
  onCenterNodes: () => void
}

const ViewControls: React.FC<ViewControlsProps> = ({ onResetView, onCenterNodes }) => {
  return (
    <div className="flex items-center space-x-1">
      <motion.button
        onClick={onResetView}
        className="px-2 py-1.5 text-xs text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors"
        whileHover={{ scale: 1.02 }}
        whileTap={{ scale: 0.98 }}
        title="重置视图"
      >
        重置
      </motion.button>
      
      <motion.button
        onClick={onCenterNodes}
        className="px-2 py-1.5 text-xs text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors"
        whileHover={{ scale: 1.02 }}
        whileTap={{ scale: 0.98 }}
        title="居中显示所有节点"
      >
        居中
      </motion.button>
    </div>
  )
}

/**
 * 主要操作按钮组件
 */
interface ActionButtonsProps {
  isRunning: boolean
  onRun: () => void
  onStop: () => void
  onSave: () => void
}

const ActionButtons: React.FC<ActionButtonsProps> = ({ isRunning, onRun, onStop, onSave }) => {
  return (
    <div className="flex items-center space-x-2">
      {isRunning ? (
        <button
          onClick={onStop}
          className="inline-flex items-center gap-2 px-3 py-2 text-sm font-medium text-white bg-red-600 hover:bg-red-700 rounded-md transition-colors"
        >
          <Square className="w-4 h-4" />
          停止
        </button>
      ) : (
        <button
          onClick={onRun}
          className="inline-flex items-center gap-2 px-3 py-2 text-sm font-medium text-white bg-green-600 hover:bg-green-700 rounded-md transition-colors"
        >
          <span>▶</span>
          运行
        </button>
      )}
      
      <button
        onClick={onSave}
        className="inline-flex items-center gap-2 px-3 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md transition-colors"
      >
        <Save className="w-4 h-4" />
        保存
      </button>
    </div>
  )
}
