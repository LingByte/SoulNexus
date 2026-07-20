import React, { useState } from 'react'
import { Input, InputTag, Select, Typography } from '@arco-design/web-react'
import { Button } from '@/components/ui'
import { useTranslation } from '@/i18n'
import { showAlert } from '@/utils/notification'
import type { WorkflowDefinition } from '@/api/workflow'

type WorkflowStatus = 'draft' | 'active' | 'archived'

// 工作流表单组件
export interface WorkflowFormProps {
  workflow?: WorkflowDefinition | null
  onSave: (data: Partial<WorkflowDefinition>) => Promise<void>
  onCancel: () => void
  saving?: boolean
}

export const WorkflowForm: React.FC<WorkflowFormProps> = ({ workflow, onSave, onCancel, saving = false }) => {
  const { t } = useTranslation()
  const [formData, setFormData] = useState({
    name: workflow?.name || '',
    description: workflow?.description || '',
    status: workflow?.status || ('draft' as WorkflowStatus),
  })
  const [tagList, setTagList] = useState<string[]>(workflow?.tags || [])

  const submit = async () => {
    if (!formData.name.trim()) {
      showAlert(t('workflow.messages.nameRequired'), 'error')
      return
    }
    await onSave({
      name: formData.name.trim(),
      description: formData.description.trim(),
      status: formData.status,
      tags: tagList,
      definition: workflow?.definition || { nodes: [], edges: [] },
    })
  }

  return (
    <div className="space-y-3">
      <div>
        <Typography.Text className="!text-xs">{t('workflow.name')}</Typography.Text>
        <Input value={formData.name} onChange={(v) => setFormData({ ...formData, name: v })} placeholder={t('workflow.name')} />
      </div>
      <div>
        <Typography.Text className="!text-xs">{t('common.description')}</Typography.Text>
        <Input.TextArea
          value={formData.description}
          onChange={(v) => setFormData({ ...formData, description: v })}
          rows={3}
        />
      </div>
      <div>
        <Typography.Text className="!text-xs">{t('common.status')}</Typography.Text>
        <Select
          value={formData.status}
          onChange={(v) => setFormData({ ...formData, status: v as WorkflowStatus })}
          options={[
            { value: 'draft', label: t('workflow.status.draft') },
            { value: 'active', label: t('workflow.status.active') },
            { value: 'archived', label: t('workflow.status.archived') },
          ]}
          style={{ width: '100%' }}
        />
      </div>
      <div>
        <Typography.Text className="!text-xs">{t('workflow.tags')}</Typography.Text>
        <InputTag
          value={tagList}
          onChange={setTagList}
          placeholder={t('workflow.tagsPlaceholder')}
          allowClear
          style={{ width: '100%' }}
        />
      </div>
      <div className="flex justify-end gap-2 border-t border-[var(--color-border-2)] pt-4">
        <Button type="outline" onClick={onCancel} disabled={saving}>
          {t('common.cancel')}
        </Button>
        <Button type="primary" loading={saving} disabled={saving} onClick={() => void submit()}>
          {t('common.save')}
        </Button>
      </div>
    </div>
  )
}
