import { useState, type ComponentType } from 'react'
import { useNavigate } from 'react-router-dom'
import { Typography } from '@arco-design/web-react'
import {
  IconBook,
  IconFile,
  IconMessage,
  IconMoon,
  IconPlus,
  IconSearch,
} from '@arco-design/web-react/icon'
import {
  ASSISTANT_TEMPLATES,
  type AssistantTemplate,
  type AssistantTemplateId,
} from '@/constants/assistantTemplates'
import { useTranslation } from '@/i18n'

const TEMPLATE_ICONS: Record<AssistantTemplateId, ComponentType<{ className?: string }>> = {
  blank: IconPlus,
  customer_cleaning: IconSearch,
  waking_up: IconMoon,
  inbound_knowledge: IconBook,
  outbound_notify: IconMessage,
  outbound_collect: IconFile,
}

function TemplateCard({
  template,
  selected,
  onSelect,
}: {
  template: AssistantTemplate
  selected: boolean
  onSelect: () => void
}) {
  const Icon = TEMPLATE_ICONS[template.id]
  return (
    <button
      type="button"
      onClick={onSelect}
      className={[
        'w-full rounded-2xl border bg-card p-4 text-left transition-all',
        'hover:shadow-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40',
        selected ? 'border-foreground shadow-md' : 'border-border',
      ].join(' ')}
    >
      <div className="flex flex-col gap-3.5">
        <div className="flex items-center gap-2">
          <div
            className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-full shadow-sm ${template.iconBg}`}
          >
            <Icon className={`h-5 w-5 ${template.iconColor}`} />
          </div>
          <Typography.Text bold style={{ fontSize: 14 }}>
            {template.label}
          </Typography.Text>
        </div>
        <Typography.Paragraph type="secondary" style={{ margin: 0, fontSize: 12, lineHeight: '16px' }}>
          {template.description}
        </Typography.Paragraph>
      </div>
    </button>
  )
}

export type AssistantTemplatePickerProps = {
  createPath?: string
  scopeTenantId?: string
}

export default function AssistantTemplatePicker({
  createPath = '/assistant-manager/create',
  scopeTenantId,
}: AssistantTemplatePickerProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [selected, setSelected] = useState<AssistantTemplateId | null>(null)

  const goCreate = (templateId: AssistantTemplateId) => {
    const q = new URLSearchParams({ template: templateId })
    if (scopeTenantId && scopeTenantId !== '0') {
      q.set('tenantId', scopeTenantId)
    }
    navigate(`${createPath}?${q.toString()}`)
  }

  return (
    <div className="flex w-full flex-col gap-4">
      <div className="w-full rounded-2xl border border-border bg-card px-6 py-8">
        <Typography.Title heading={5} style={{ margin: 0, textAlign: 'center', fontWeight: 500 }}>
          {t('assistant.templatePickerTitle')}
        </Typography.Title>

        <div className="mt-6 grid w-full grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-3">
          {ASSISTANT_TEMPLATES.map((template) => (
            <TemplateCard
              key={template.id}
              template={template}
              selected={selected === template.id}
              onSelect={() => {
                setSelected(template.id)
                goCreate(template.id)
              }}
            />
          ))}
        </div>
      </div>
    </div>
  )
}
