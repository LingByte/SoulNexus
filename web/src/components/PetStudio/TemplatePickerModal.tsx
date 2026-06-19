import { useMemo } from 'react'
import { X } from 'lucide-react'
import { useI18nStore } from '@/stores/i18nStore'
import {
  listStarterTemplates,
  type PetStarterTemplateId,
} from '@/pages/pet-market/templates/registry'

interface TemplatePickerModalProps {
  open: boolean
  onClose: () => void
  onSelect: (templateId: PetStarterTemplateId) => void
}

export default function TemplatePickerModal({ open, onClose, onSelect }: TemplatePickerModalProps) {
  const { t } = useI18nStore()
  const templates = useMemo(() => listStarterTemplates(), [])

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-[200] flex items-center justify-center bg-black/55 p-4"
      onClick={onClose}
      role="presentation"
    >
      <div
        className="w-full max-w-3xl max-h-[90vh] overflow-auto rounded-2xl bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 shadow-2xl"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="pet-template-picker-title"
      >
        <div className="flex items-center justify-between px-5 py-4 border-b border-gray-100 dark:border-gray-800">
          <div>
            <h2 id="pet-template-picker-title" className="text-base font-semibold text-gray-900 dark:text-white">
              {t('petMarket.templatePicker.title')}
            </h2>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
              {t('petMarket.templatePicker.subtitle')}
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="p-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-500"
            aria-label="close"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="p-5 grid grid-cols-1 sm:grid-cols-2 gap-3">
          {templates.map((tpl) => (
            <button
              key={tpl.id}
              type="button"
              onClick={() => onSelect(tpl.id)}
              className="text-left rounded-xl border border-gray-200 dark:border-gray-700 p-4 hover:border-indigo-400 dark:hover:border-indigo-500 hover:shadow-md hover:shadow-indigo-500/10 transition-all bg-gray-50/50 dark:bg-gray-800/40"
            >
              <div className="flex items-start gap-3">
                <span
                  className="shrink-0 min-w-[3rem] px-2 py-1 rounded-md text-[11px] font-semibold uppercase tracking-wide text-indigo-700 dark:text-indigo-300 bg-indigo-100 dark:bg-indigo-950/60 border border-indigo-200/70 dark:border-indigo-800/60 text-center"
                  aria-hidden
                >
                  {tpl.badge}
                </span>
                <div className="min-w-0">
                  <div className="font-medium text-gray-900 dark:text-white text-sm">
                    {t(tpl.nameKey) !== tpl.nameKey ? t(tpl.nameKey) : tpl.name}
                  </div>
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1 leading-relaxed">
                    {t(tpl.descKey) !== tpl.descKey ? t(tpl.descKey) : tpl.description}
                  </p>
                  <span className="inline-block mt-2 text-[10px] uppercase tracking-wide text-indigo-600 dark:text-indigo-400">
                    {tpl.previewType === 'sprite' ? 'Sprite' : 'Canvas'}
                  </span>
                </div>
              </div>
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}
