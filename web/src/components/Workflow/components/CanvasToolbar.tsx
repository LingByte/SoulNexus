import React from 'react'
import { Plus, AlertCircle, Trash2, Square, Save, FileText } from 'lucide-react'
import { Button } from '@/components/ui'

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
  t,
}) => {
  return (
    <div className="relative z-20 flex flex-nowrap items-center justify-between gap-2 overflow-x-auto border-b border-gray-200 bg-white p-2 md:p-3 dark:border-gray-700 dark:bg-gray-800">
      <div className="flex shrink-0 items-center gap-2">
        {!validation.valid && (
          <div className="flex items-center text-sm text-red-600">
            <AlertCircle className="h-4 w-4" />
            <span className="ml-1">{validation.message}</span>
          </div>
        )}
        <Button type="primary" size="sm" icon={<Plus className="h-4 w-4" />} onClick={onAddNode}>
          {t('workflow.editor.addNode')}
        </Button>
        <Button type="text" size="sm" icon={<FileText className="h-4 w-4" />} onClick={onHelp} title={t('workflow.editor.help')} />
      </div>

      <div className="flex shrink-0 items-center gap-1 md:gap-2">
        <div className="flex items-center gap-1 rounded-lg bg-gray-100 p-1 dark:bg-gray-700">
          <Button type="text" size="mini" onClick={onZoomOut}>
            −
          </Button>
          <span className="min-w-[2.5rem] px-1 text-center text-xs text-gray-600 dark:text-gray-400">
            {Math.round(canvasScale * 100)}%
          </span>
          <Button type="text" size="mini" onClick={onZoomIn}>
            +
          </Button>
        </div>
        <Button type="text" size="mini" onClick={onResetView}>
          {t('workflow.editor.resetView')}
        </Button>
        <Button type="text" size="mini" onClick={onCenterNodes}>
          {t('workflow.editor.center')}
        </Button>
        <div className="mx-1 h-6 w-px bg-gray-300 dark:bg-gray-600" />
        {selectedConnection ? (
          <Button type="primary" status="danger" size="sm" icon={<Trash2 className="h-4 w-4" />} onClick={onDeleteConnection}>
            {t('workflow.editor.deleteConnection')}
          </Button>
        ) : null}
        {isRunning ? (
          <Button type="primary" status="danger" size="sm" icon={<Square className="h-4 w-4" />} onClick={onStop}>
            {t('common.stop')}
          </Button>
        ) : (
          <Button type="primary" status="success" size="sm" onClick={onRun}>
            {t('workflow.editor.run')}
          </Button>
        )}
        <Button type="outline" size="sm" icon={<Save className="h-4 w-4" />} onClick={onSave}>
          {t('workflow.editor.save')}
        </Button>
      </div>
    </div>
  )
}
