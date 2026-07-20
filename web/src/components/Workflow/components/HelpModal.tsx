import React from 'react'
import { Drawer } from '@arco-design/web-react'
import { useTranslation } from '@/i18n'

interface HelpDrawerProps {
  isOpen: boolean
  onClose: () => void
}

/** 工作流编辑器操作说明（右侧抽屉） */
export const HelpDrawer: React.FC<HelpDrawerProps> = ({ isOpen, onClose }) => {
  const { t } = useTranslation()
  const h = (key: string) => t(`workflow.help.${key}`)

  return (
    <Drawer
      width={420}
      title={t('workflow.editor.help')}
      visible={isOpen}
      onCancel={onClose}
      footer={null}
      unmountOnExit
    >
      <div className="space-y-6 pb-4">
        <div>
          <h4 className="mb-3 text-sm font-semibold text-gray-900 dark:text-white">{h('connectionsTitle')}</h4>
          <div className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
            <div className="flex items-center gap-2">
              <div className="h-3 w-3 rounded-full bg-blue-500" />
              <span>{h('outputPoint')}</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="h-3 w-3 rounded-full bg-green-500" />
              <span>{h('inputPoint')}</span>
            </div>
            <ul className="ml-5 list-inside list-disc space-y-1">
              <li>{h('connClickBlue')}</li>
              <li>{h('connDragGreen')}</li>
              <li>{h('connClickLine')}</li>
              <li>{h('connRightClick')}</li>
              <li>{h('connDoubleClick')}</li>
              <li>{h('connDeleteKey')}</li>
            </ul>
          </div>
        </div>

        <div>
          <h4 className="mb-3 text-sm font-semibold text-gray-900 dark:text-white">{h('canvasTitle')}</h4>
          <ul className="ml-5 list-inside list-disc space-y-1 text-sm text-gray-600 dark:text-gray-400">
            <li>{h('canvasPan')}</li>
            <li>{h('canvasDragNode')}</li>
            <li>{h('canvasZoomBtn')}</li>
            <li>{h('canvasWheel')}</li>
            <li>{h('canvasReset')}</li>
            <li>{h('canvasCenter')}</li>
            <li>{h('canvasSelect')}</li>
            <li>{h('canvasConfig')}</li>
          </ul>
        </div>

        <div>
          <h4 className="mb-3 text-sm font-semibold text-gray-900 dark:text-white">{h('shortcutsTitle')}</h4>
          <ul className="ml-5 list-inside list-disc space-y-1 text-sm text-gray-600 dark:text-gray-400">
            <li>
              <kbd className="rounded bg-gray-200 px-2 py-1 text-xs dark:bg-gray-700">Delete</kbd> — {h('shortcutDelete')}
            </li>
            <li>
              <kbd className="rounded bg-gray-200 px-2 py-1 text-xs dark:bg-gray-700">Esc</kbd> — {h('shortcutEsc')}
            </li>
            <li>
              <kbd className="rounded bg-gray-200 px-2 py-1 text-xs dark:bg-gray-700">Ctrl+S</kbd> — {h('shortcutSave')}
            </li>
          </ul>
        </div>

        <div>
          <h4 className="mb-3 text-sm font-semibold text-gray-900 dark:text-white">{h('nodeTypesTitle')}</h4>
          <div className="grid grid-cols-2 gap-3 text-sm text-gray-600 dark:text-gray-400">
            {(
              [
                ['nodeStart', 'nodeStartDesc'],
                ['nodeEnd', 'nodeEndDesc'],
                ['nodeTask', 'nodeTaskDesc'],
                ['nodeGateway', 'nodeGatewayDesc'],
                ['nodeEvent', 'nodeEventDesc'],
                ['nodeScript', 'nodeScriptDesc'],
              ] as const
            ).map(([titleKey, descKey]) => (
              <div key={titleKey}>
                <strong className="text-gray-900 dark:text-white">{h(titleKey)}</strong>
                <p className="text-xs">{h(descKey)}</p>
              </div>
            ))}
          </div>
        </div>
      </div>
    </Drawer>
  )
}

/** @deprecated Use HelpDrawer */
export const HelpModal = HelpDrawer
