// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect, useState } from 'react'
import { X } from 'lucide-react'
import Button from '@/components/UI/Button'
import { Input as ArcoInput } from '@arco-design/web-react'
import { Select as ArcoSelect } from '@arco-design/web-react'
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
  const [enter, setEnter] = useState(false)
  const [busy, setBusy] = useState(false)
  const [groups, setGroups] = useState<Group[]>([])
  const [formNs, setFormNs] = useState('')
  const [formName, setFormName] = useState('')
  const [formDesc, setFormDesc] = useState('')
  const [formGroupId, setFormGroupId] = useState('')

  useEffect(() => {
    if (!open) {
      setEnter(false)
      return
    }
    const t0 = requestAnimationFrame(() => setEnter(true))
    void getGroupList().then((res) => {
      if (res.code === 200 && Array.isArray(res.data)) setGroups(res.data)
      else setGroups([])
    })
    return () => cancelAnimationFrame(t0)
  }, [open])

  const close = () => {
    setEnter(false)
    setTimeout(onClose, 200)
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
      setFormNs('')
      setFormName('')
      setFormDesc('')
      setFormGroupId('')
      close()
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || String(e), 'error', t('knowledge.create'))
    } finally {
      setBusy(false)
    }
  }

  if (!open) return null

  return (
    <div className="fixed inset-0 z-[100]">
      <button type="button" className={`absolute inset-0 bg-black/40 transition-opacity ${enter ? 'opacity-100' : 'opacity-0'}`} aria-label="Close" onClick={() => !busy && close()} />
      <div
        className={`absolute right-0 top-0 flex h-full w-full max-w-md flex-col border-l border-border bg-background shadow-2xl transition-transform duration-200 ease-out ${
          enter ? 'translate-x-0' : 'translate-x-full'
        }`}
      >
        <div className="flex items-center justify-between border-b border-border px-4 py-3">
          <span className="font-semibold">{t('knowledge.newBase')}</span>
          <button type="button" className="rounded p-1 hover:bg-muted" onClick={() => !busy && close()}>
            <X className="h-5 w-5" />
          </button>
        </div>
        <div className="flex-1 space-y-4 overflow-y-auto p-4">
          <div>
            <label className="mb-1 block text-xs font-medium text-muted-foreground">{t('knowledge.fieldGroup')}</label>
            <Select value={formGroupId} onValueChange={setFormGroupId}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder={t('knowledge.fieldGroupHint')} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">{t('knowledge.fieldGroupHint')}</SelectItem>
                {groups.map((g) => (
                  <SelectItem key={g.id} value={String(g.id)}>
                    {g.name} (#{g.id})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-muted-foreground">{t('knowledge.fieldNamespace')}</label>
            <ArcoInput size="large" className="!h-10 !text-base" value={formNs} onChange={(e) => setFormNs(e.target.value)} className="font-mono text-sm" />
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-muted-foreground">{t('knowledge.fieldName')}</label>
            <ArcoInput size="large" className="!h-10 !text-base" value={formName} onChange={(e) => setFormName(e.target.value)} />
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-muted-foreground">{t('knowledge.fieldDesc')}</label>
            <textarea value={formDesc} onChange={(e) => setFormDesc(e.target.value)} rows={2} className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm" />
          </div>
        </div>
        <div className="flex justify-end gap-2 border-t border-border p-4">
          <Button variant="ghost" onClick={() => !busy && close()} disabled={busy}>
            {t('knowledge.cancel')}
          </Button>
          <Button variant="primary" onClick={() => void submit()} loading={busy} disabled={busy}>
            {t('knowledge.create')}
          </Button>
        </div>
      </div>
    </div>
  )
}

export default KnowledgeCreateDrawer
