// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useState } from 'react'
import { Input as ArcoInput, Select as ArcoSelect, Drawer } from '@arco-design/web-react'
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

const KnowledgeCreateDrawer: React.FC<Props> = ({ open, onClose, onCreated }) => {
  const { t } = useI18nStore()
  const [busy, setBusy] = useState(false)
  const [groups, setGroups] = useState<Group[]>([])
  const [formNs, setFormNs] = useState('')
  const [formName, setFormName] = useState('')
  const [formDesc, setFormDesc] = useState('')
  const [formGroupId, setFormGroupId] = useState<string | undefined>(undefined)

  useEffect(() => {
    if (!open) return
    void getGroupList().then((res) => {
      if (res.code === 200 && Array.isArray(res.data)) setGroups(res.data)
      else setGroups([])
    })
  }, [open])

  const resetForm = () => {
    setFormNs('')
    setFormName('')
    setFormDesc('')
    setFormGroupId(undefined)
  }

  const handleClose = () => {
    if (busy) return
    resetForm()
    onClose()
  }

  const submit = async () => {
    const ns = formNs.trim()
    const name = formName.trim()
    if (!ns || !name) {
      showAlert('namespace / name required', 'error', t('knowledge.create'))
      return
    }
    setBusy(true)
    try {
      const body: Parameters<typeof createKnowledgeNamespace>[0] = {
        namespace: ns,
        name,
        description: formDesc.trim() || undefined,
      }
      if (formGroupId) {
        const gid = parseInt(formGroupId, 10)
        if (!Number.isNaN(gid)) body.groupId = gid
      }
      const res = await createKnowledgeNamespace(body)
      if (res.code !== 200 || !res.data) {
        showAlert(res.msg || 'failed', 'error', t('knowledge.create'))
        return
      }
      showAlert(res.msg || 'ok', 'success', t('knowledge.create'))
      onCreated(res.data)
      resetForm()
      onClose()
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || String(e), 'error', t('knowledge.create'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Drawer
      width={420}
      title={t('knowledge.newBase')}
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
      <div className="space-y-4">
        <div>
          <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t('knowledge.fieldGroup')}
          </label>
          <ArcoSelect
            size="large"
            value={formGroupId}
            onChange={(val) => setFormGroupId(val as string | undefined)}
            placeholder={t('knowledge.fieldGroupHint')}
            allowClear
            className="w-full"
            options={[
              { label: t('knowledge.fieldGroupHint'), value: '' },
              ...groups.map((g) => ({ label: `${g.name} (#${g.id})`, value: String(g.id) })),
            ]}
          />
        </div>
        <div>
          <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t('knowledge.fieldNamespace')} <span className="text-red-500">*</span>
          </label>
          <ArcoInput
            size="large"
            value={formNs}
            onChange={(val) => setFormNs(val)}
            placeholder="e.g. my-knowledge-base"
          />
        </div>
        <div>
          <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t('knowledge.fieldName')} <span className="text-red-500">*</span>
          </label>
          <ArcoInput
            size="large"
            value={formName}
            onChange={(val) => setFormName(val)}
            placeholder={t('knowledge.fieldName')}
          />
        </div>
        <div>
          <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t('knowledge.fieldDesc')}
          </label>
          <ArcoInput.TextArea
            value={formDesc}
            onChange={(val: string) => setFormDesc(val)}
            rows={3}
            placeholder={t('knowledge.fieldDesc')}
          />
        </div>
      </div>
    </Drawer>
  )
}

export default KnowledgeCreateDrawer
