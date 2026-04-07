import { useCallback, useEffect, useState } from 'react'
import Button from '@/components/UI/Button'
import LoadingAnimation from '@/components/Animations/LoadingAnimation'
import { useI18nStore } from '@/stores/i18nStore'
import { showAlert } from '@/utils/notification'
import {
  createSIPScriptTemplate,
  deleteSIPScriptTemplate,
  listSIPScriptTemplates,
  updateSIPScriptTemplate,
  type SIPScriptTemplateRow,
} from '@/api/sipContactCenter'
import ConfirmDialog from '@/components/UI/ConfirmDialog'

type FormState = {
  name: string
  description: string
  enabled: boolean
  scriptSpec: string
}

const defaultScriptSpec = `{
  "id": "followup-v1",
  "version": "2026-04-06",
  "start_id": "begin",
  "steps": [
    { "id": "begin", "type": "say", "prompt": "你好，这里是云联络中心回访。", "next_id": "end" },
    { "id": "end", "type": "end" }
  ]
}`

const defaultForm = (): FormState => ({
  name: '',
  description: '',
  enabled: true,
  scriptSpec: defaultScriptSpec,
})

export default function ScriptManagerTab() {
  const { t } = useI18nStore()
  const [rows, setRows] = useState<SIPScriptTemplateRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<SIPScriptTemplateRow | null>(null)
  const [form, setForm] = useState<FormState>(defaultForm)
  const [scriptDeleteOpen, setScriptDeleteOpen] = useState(false)
  const [scriptDeleteId, setScriptDeleteId] = useState<number | null>(null)
  const pageSize = 20

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listSIPScriptTemplates(page, pageSize)
      if (res.code === 200 && res.data) {
        setRows(res.data.list || [])
        setTotal(res.data.total || 0)
      }
    } catch (e: unknown) {
      const err = e as { msg?: string }
      showAlert(err?.msg || t('common.failed'), 'error')
    } finally {
      setLoading(false)
    }
  }, [page, t])

  useEffect(() => {
    void load()
  }, [load])

  const openCreate = () => {
    setEditing(null)
    setForm(defaultForm())
    setModalOpen(true)
  }

  const openEdit = (row: SIPScriptTemplateRow) => {
    setEditing(row)
    setForm({
      name: row.name || '',
      description: row.description || '',
      enabled: !!row.enabled,
      scriptSpec:
        typeof row.scriptSpec === 'string'
          ? row.scriptSpec
          : JSON.stringify(row.scriptSpec || {}, null, 2),
    })
    setModalOpen(true)
  }

  const save = async () => {
    if (!form.name.trim()) {
      showAlert(t('contactCenter.script.required'), 'error')
      return
    }
    setSaving(true)
    try {
      const body = {
        name: form.name.trim(),
        description: form.description.trim(),
        enabled: form.enabled,
        scriptSpec: form.scriptSpec.trim(),
      }
      const res = editing
        ? await updateSIPScriptTemplate(editing.id, body)
        : await createSIPScriptTemplate(body)
      if (res.code === 200) {
        showAlert(t('contactCenter.toast.saveScriptOk'), 'success')
        setModalOpen(false)
        setEditing(null)
        await load()
      } else {
        showAlert(res.msg || t('common.failed'), 'error')
      }
    } catch (e: unknown) {
      const err = e as { msg?: string }
      showAlert(err?.msg || t('common.failed'), 'error')
    } finally {
      setSaving(false)
    }
  }

  const openScriptDelete = (id: number) => {
    setScriptDeleteId(id)
    setScriptDeleteOpen(true)
  }

  const confirmScriptDelete = async () => {
    if (scriptDeleteId == null) return
    const id = scriptDeleteId
    try {
      const res = await deleteSIPScriptTemplate(id)
      if (res.code === 200) {
        showAlert(t('contactCenter.toast.deleteScriptOk'), 'success')
        await load()
        return
      }
      showAlert(res.msg || t('common.failed'), 'error')
      throw new Error('script-delete-failed')
    } catch (e: unknown) {
      if (e instanceof Error && e.message === 'script-delete-failed') throw e
      const err = e as { msg?: string }
      showAlert(err?.msg || t('common.failed'), 'error')
      throw e
    }
  }

  return (
    <div className="mt-4 space-y-3">
      <p className="text-xs text-muted-foreground leading-relaxed rounded-lg border border-border bg-muted/30 px-3 py-2.5">
        {t('contactCenter.script.hint')}
      </p>
      <div className="flex gap-2">
        <Button size="sm" variant="outline" onClick={openCreate}>
          {t('contactCenter.script.create')}
        </Button>
        <Button size="sm" onClick={() => void load()}>
          {t('common.refresh')}
        </Button>
      </div>

      {loading ? (
        <LoadingAnimation />
      ) : (
        <div className="overflow-x-auto rounded-lg border border-border bg-card">
          <table className="min-w-[920px] w-full text-sm">
            <thead className="bg-muted/50">
              <tr>
                <th className="text-left p-3">ID</th>
                <th className="text-left p-3">{t('contactCenter.script.name')}</th>
                <th className="text-left p-3">{t('contactCenter.script.scriptId')}</th>
                <th className="text-left p-3">{t('contactCenter.script.enabled')}</th>
                <th className="text-left p-3">{t('contactCenter.script.updatedAt')}</th>
                <th className="text-right p-3">{t('contactCenter.ai.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {rows.length === 0 ? (
                <tr>
                  <td colSpan={6} className="p-6 text-center text-muted-foreground">
                    {t('common.noData')}
                  </td>
                </tr>
              ) : (
                rows.map((r) => (
                  <tr key={r.id} className="border-t border-border">
                    <td className="p-3">{r.id}</td>
                    <td className="p-3">{r.name}</td>
                    <td className="p-3 font-mono text-xs">{r.scriptId}</td>
                    <td className="p-3">{r.enabled ? t('common.status') + ': on' : t('common.status') + ': off'}</td>
                    <td className="p-3 text-xs">{r.updatedAt ? new Date(r.updatedAt).toLocaleString() : '—'}</td>
                    <td className="p-3 text-right space-x-2">
                      <Button variant="outline" size="sm" onClick={() => openEdit(r)}>
                        {t('common.edit')}
                      </Button>
                      <Button variant="outline" size="sm" onClick={() => openScriptDelete(r.id)}>
                        {t('common.delete')}
                      </Button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
          <div className="flex items-center justify-between p-3 border-t border-border text-sm">
            <span className="text-muted-foreground">
              {t('common.total')}: {total}
            </span>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>
                {t('common.prevPage')}
              </Button>
              <Button variant="outline" size="sm" disabled={page * pageSize >= total} onClick={() => setPage((p) => p + 1)}>
                {t('common.nextPage')}
              </Button>
            </div>
          </div>
        </div>
      )}

      {modalOpen && (
        <div className="fixed inset-0 z-[120] flex items-center justify-center p-4">
          <button type="button" className="absolute inset-0 bg-black/50" aria-label={t('common.close')} onClick={() => setModalOpen(false)} />
          <div className="relative z-[121] w-full max-w-2xl rounded-lg border border-border bg-card p-5 shadow-xl space-y-3">
            <h3 className="text-lg font-semibold">
              {editing ? t('contactCenter.script.edit') : t('contactCenter.script.create')}
            </h3>
            <div className="grid grid-cols-1 gap-3">
              <input className="border border-border rounded-md px-3 py-2 bg-background" placeholder={t('contactCenter.script.name')} value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} />
            </div>
            <textarea className="border border-border rounded-md px-3 py-2 bg-background w-full h-20" placeholder={t('contactCenter.script.description')} value={form.description} onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))} />
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={form.enabled} onChange={(e) => setForm((f) => ({ ...f, enabled: e.target.checked }))} />
              {t('contactCenter.script.enabled')}
            </label>
            <textarea className="border border-border rounded-md px-3 py-2 bg-background w-full h-64 font-mono text-xs" value={form.scriptSpec} onChange={(e) => setForm((f) => ({ ...f, scriptSpec: e.target.value }))} />
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setModalOpen(false)} disabled={saving}>
                {t('common.cancel')}
              </Button>
              <Button onClick={() => void save()} disabled={saving}>
                {saving ? t('common.loading') : t('common.save')}
              </Button>
            </div>
          </div>
        </div>
      )}

      <ConfirmDialog
        isOpen={scriptDeleteOpen}
        onClose={() => {
          setScriptDeleteOpen(false)
          setScriptDeleteId(null)
        }}
        onConfirm={confirmScriptDelete}
        title={t('contactCenter.confirm.deleteScriptTitle')}
        message={t('contactCenter.confirm.deleteScriptMessage')}
        confirmText={t('contactCenter.confirm.confirmDelete')}
        cancelText={t('common.cancel')}
        type="danger"
      />
    </div>
  )
}

