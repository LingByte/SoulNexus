// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useState } from 'react'
import { Input as ArcoInput, Select as ArcoSelect, Drawer, Form as ArcoForm } from '@arco-design/web-react'
import Button from '@/components/UI/Button'
import { useI18nStore } from '@/stores/i18nStore'
import { showAlert } from '@/utils/alert'
import { getGroupList, type Group } from '@/api/group'
import { createKnowledgeNamespace, type KnowledgeNamespaceRow } from '@/api/knowledge'

type Props = {
  open: boolean
  onClose: () => void
  onCreated: (row: KnowledgeNamespaceRow) => void
}

const FormItem = ArcoForm.Item

const KnowledgeCreateDrawer: React.FC<Props> = ({ open, onClose, onCreated }) => {
  const { t } = useI18nStore()
  const [busy, setBusy] = useState(false)
  const [groups, setGroups] = useState<Group[]>([])
  const [form] = ArcoForm.useForm()

  useEffect(() => {
    if (!open) return
    void getGroupList().then((res) => {
      if (res.code === 200 && Array.isArray(res.data)) setGroups(res.data)
      else setGroups([])
    })
  }, [open])

  const handleClose = () => {
    if (busy) return
    form.resetFields()
    onClose()
  }

  const submit = async () => {
    const values = await form.validate()
    if (!values) return

    setBusy(true)
    try {
      const body: Parameters<typeof createKnowledgeNamespace>[0] = {
        namespace: values.namespace.trim(),
        name: values.name.trim(),
        description: values.description?.trim() || undefined,
      }
      if (values.groupId) {
        const gid = parseInt(values.groupId, 10)
        if (!Number.isNaN(gid)) body.groupId = gid
      }
      const res = await createKnowledgeNamespace(body)
      if (res.code !== 200 || !res.data) {
        showAlert(res.msg || 'failed', 'error', t('knowledge.create'))
        return
      }
      showAlert(res.msg || 'ok', 'success', t('knowledge.create'))
      onCreated(res.data)
      form.resetFields()
      onClose()
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || String(e), 'error', t('knowledge.create'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Drawer
      width={440}
      title={
        <div className="flex items-center gap-2">
          <span className="text-base font-semibold">{t('knowledge.newBase')}</span>
        </div>
      }
      visible={open}
      onCancel={handleClose}
      footer={
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={handleClose} disabled={busy}>
            {t('knowledge.cancel')}
          </Button>
          <Button variant="primary" onClick={() => void submit()} loading={busy} disabled={busy}>
            {t('knowledge.create')}
          </Button>
        </div>
      }
      maskClosable={!busy}
      closable={!busy}
    >
      <ArcoForm
        form={form}
        layout="vertical"
        className="space-y-1"
        initialValues={{ namespace: '', name: '', description: '', groupId: undefined }}
      >
        <FormItem label={t('knowledge.fieldGroup')} field="groupId">
          <ArcoSelect
            size="large"
            placeholder={t('knowledge.fieldGroupHint')}
            allowClear
            className="w-full"
            options={[
              ...groups.map((g) => ({ label: `${g.name} (#${g.id})`, value: String(g.id) })),
            ]}
          />
        </FormItem>

        <FormItem
          label={t('knowledge.fieldNamespace')}
          field="namespace"
          rules={[
            { required: true, message: 'Required' },
            {
              validator: (_value: string | undefined, callback: (error?: React.ReactNode) => void) => {
                if (_value && !/^[a-zA-Z0-9_-]+$/.test(_value)) {
                  callback('ASCII letters, numbers, hyphens, underscores only')
                } else {
                  callback()
                }
              },
            },
          ]}
        >
          <ArcoInput
            size="large"
            placeholder="e.g. my-knowledge-base"
          />
        </FormItem>

        <FormItem
          label={t('knowledge.fieldName')}
          field="name"
          rules={[{ required: true, message: 'Required' }]}
        >
          <ArcoInput
            size="large"
            placeholder={t('knowledge.fieldName')}
          />
        </FormItem>

        <FormItem label={t('knowledge.fieldDesc')} field="description">
          <ArcoInput.TextArea
            rows={3}
            placeholder={t('knowledge.fieldDesc')}
          />
        </FormItem>
      </ArcoForm>
    </Drawer>
  )
}

export default KnowledgeCreateDrawer
