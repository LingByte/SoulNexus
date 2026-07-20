import { useCallback, useEffect, useState } from 'react'
import { Modal as ArcoModal, Tag, Upload } from '@arco-design/web-react'
import { RefreshCw, Pencil, Trash2, Power, PowerOff, Package } from 'lucide-react'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Button, DataList, Input, Select } from '@/components/ui'
import type { DataListColumn } from '@/components/ui'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  createPlatformMcpMarketItem, deletePlatformMcpMarketItem,
  listPlatformMcpMarket, updatePlatformMcpMarketItem,
  uploadPlatformMcpMarketLogo, type McpMarketItem,
} from '@/api/mcpMarket'
import { getUploadsBaseURL } from '@/config/apiConfig'
import { useTranslation } from '@/i18n'

const emptyForm = () => ({ slug: '', name: '', displayName: '', description: '', category: 'utility', mcpSseUrl: '', version: '1.0.0', status: 'draft', author: 'platform', tags: '', logoUrl: '', timeoutMs: 15000 })

export default function PlatformMcpMarketPage() {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [list, setList] = useState<McpMarketItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [status, setStatus] = useState('')
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<McpMarketItem | null>(null)
  const [form, setForm] = useState(emptyForm())
  const [saving, setSaving] = useState(false)
  const [logoUploading, setLogoUploading] = useState(false)

  const resolveLogoUrl = (url?: string) => {
    if (!url) return ''
    const u = url.trim()
    if (/^https?:\/\//i.test(u)) return u
    const base = getUploadsBaseURL().replace(/\/$/, '')
    return u.startsWith('/') ? `${base}${u}` : `${base}/${u}`
  }

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listPlatformMcpMarket({ page, size: 20, status: status || undefined })
      if (res.code === 200 && res.data) { setList(res.data.list || []); setTotal(res.data.total || 0) }
      else showAlert(res.msg || t('common.loadFailed'), 'error')
    } catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('common.loadFailed')), 'error') }
    finally { setLoading(false) }
  }, [page, status, t])

  useEffect(() => { void load() }, [load])

  const openCreate = () => { setEditing(null); setForm(emptyForm()); setModalOpen(true) }
  const openEdit = (row: McpMarketItem) => {
    setEditing(row)
    setForm({ slug: row.slug, name: row.name, displayName: row.displayName || '', description: row.description || '', category: row.category || 'utility', mcpSseUrl: row.mcpSseUrl || '', version: row.version || '1.0.0', status: row.status || 'draft', author: row.author || 'platform', tags: row.tags || '', logoUrl: row.logoUrl || '', timeoutMs: row.timeoutMs || 15000 })
    setModalOpen(true)
  }

  const handleSave = async () => {
    if (!form.slug.trim() || !form.mcpSseUrl.trim()) { showAlert(t('mcpMarketAdmin.requiredFields'), 'error'); return }
    setSaving(true)
    try {
      const body = { slug: form.slug.trim(), name: form.name.trim() || form.slug.trim(), displayName: form.displayName.trim(), description: form.description.trim(), category: form.category, mcpSseUrl: form.mcpSseUrl.trim(), version: form.version.trim(), status: form.status as 'draft' | 'published' | 'archived', author: form.author.trim() || 'platform', tags: form.tags.trim(), logoUrl: form.logoUrl.trim(), timeoutMs: form.timeoutMs }
      const res = editing ? await updatePlatformMcpMarketItem(editing.id, body as Parameters<typeof updatePlatformMcpMarketItem>[1]) : await createPlatformMcpMarketItem(body as Parameters<typeof createPlatformMcpMarketItem>[0])
      if (res.code !== 200) { showAlert(res.msg || t('common.saveFailed'), 'error'); return }
      showAlert(t('common.saveSuccess'), 'success'); setModalOpen(false); void load()
    } catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('common.saveFailed')), 'error') }
    finally { setSaving(false) }
  }

  const remove = (row: McpMarketItem) => {
    ArcoModal.confirm({
      title: t('common.confirmDelete'), content: row.displayName || row.name,
      onOk: async () => {
        const res = await deletePlatformMcpMarketItem(row.id)
        if (res.code !== 200) { showAlert(res.msg || t('common.deleteFailed'), 'error'); return }
        showAlert(t('common.deleteSuccess'), 'success'); void load()
      },
    })
  }

  const setItemStatus = async (row: McpMarketItem, next: string) => {
    const res = await updatePlatformMcpMarketItem(row.id, { status: next as 'draft' | 'published' | 'archived' })
    if (res.code !== 200) { showAlert(res.msg || t('common.saveFailed'), 'error'); return }
    showAlert(t('common.saveSuccess'), 'success'); void load()
  }

  const handleLogoUpload = async (file: File) => {
    if (!['image/jpeg', 'image/png'].includes(file.type)) { showAlert(t('mcpMarket.logoFormatInvalid'), 'error'); return false }
    if (file.size > 5 * 1024 * 1024) { showAlert(t('mcpMarket.logoTooLarge'), 'error'); return false }
    setLogoUploading(true)
    try {
      const res = await uploadPlatformMcpMarketLogo(file)
      if (res.code !== 200 || !res.data?.logoUrl) { showAlert(res.msg || t('mcpMarket.logoUploadFailed'), 'error'); return false }
      setForm((prev) => ({ ...prev, logoUrl: res.data!.logoUrl })); return false
    } catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('mcpMarket.logoUploadFailed')), 'error'); return false }
    finally { setLogoUploading(false) }
  }

  const columns: DataListColumn<Record<string, unknown>>[] = [
    {
      key: 'info', title: t('mcpMarketAdmin.colName'),
      render: (_, r) => (
        <div className="flex items-center gap-3">
          {r.logoUrl ? (
            <img src={resolveLogoUrl(String(r.logoUrl))} alt="" className="h-8 w-8 shrink-0 rounded object-cover" />
          ) : (
            <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded bg-neutral-100 text-neutral-400">
              <Package size={16} />
            </div>
          )}
          <div className="min-w-0">
            <div className="truncate text-sm font-medium text-neutral-900">{String(r.displayName || r.name || '—')}</div>
            <div className="font-mono text-xs text-neutral-400">{String(r.slug || '')}</div>
          </div>
        </div>
      ),
    },
    { key: 'status', title: t('common.status'), width: 110, render: (_, r) => <Tag className="!rounded-full">{String(r.status || '—')}</Tag> },
    { key: 'installs', title: t('mcpMarketAdmin.colInstalls'), width: 90, render: (_, r) => <span className="text-sm text-neutral-700">{String(r.installCount ?? 0)}</span> },
    {
      key: 'actions', title: t('common.actions'), width: 220, align: 'right',
      render: (_, r) => (
        <div className="flex items-center justify-end gap-1">
          <Button size="mini" icon={<Pencil size={12} />} onClick={() => openEdit(r as unknown as McpMarketItem)}>{t('common.edit')}</Button>
          {String(r.status) !== 'published' ? (
            <Button size="mini" icon={<Power size={12} />} onClick={() => void setItemStatus(r as unknown as McpMarketItem, 'published')}>{t('mcpMarketAdmin.publish')}</Button>
          ) : (
            <Button size="mini" icon={<PowerOff size={12} />} onClick={() => void setItemStatus(r as unknown as McpMarketItem, 'archived')}>{t('mcpMarketAdmin.archive')}</Button>
          )}
          <Button size="mini" status="danger" icon={<Trash2 size={12} />} onClick={() => remove(r as unknown as McpMarketItem)}>{t('common.delete')}</Button>
        </div>
      ),
    },
  ]

  return (
    <BaseLayout title={t('pages.platformMcpMarket.title')} description={t('pages.platformMcpMarket.description')}>
      <DataList
        data={list as unknown as (McpMarketItem & Record<string, unknown>)[]}
        columns={columns}
        loading={loading}
        rowKey="id"
        emptyText={t('common.noData')}
        pagination={{ current: page, pageSize: 20, total, onChange: (p) => setPage(p) }}
        header={
          <div className="flex items-center justify-between gap-3">
            <div>
              <div className="mb-1 text-xs text-neutral-400">{t('common.status')}</div>
              <Select allowClear value={status || undefined} onChange={(v) => { setStatus(String(v || '')); setPage(1) }} style={{ width: 140 }} options={[{ value: 'published', label: 'published' }, { value: 'draft', label: 'draft' }, { value: 'archived', label: 'archived' }]} />
            </div>
            <div className="flex items-center gap-2">
              <Button type="outline" icon={<RefreshCw size={14} />} loading={loading} onClick={() => void load()}>{t('common.refresh')}</Button>
              <Button type="primary" onClick={openCreate}>{t('common.create')}</Button>
            </div>
          </div>
        }
      />

      <ArcoModal
        title={editing ? t('mcpMarketAdmin.editTitle') : t('mcpMarketAdmin.createTitle')}
        visible={modalOpen} onCancel={() => setModalOpen(false)} onOk={() => void handleSave()} confirmLoading={saving} style={{ width: 640 }}
      >
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          <div><div className="mb-1 text-xs text-neutral-400">{t('mcpMarket.fieldSlug')}</div><Input value={form.slug} onChange={(v) => setForm((p) => ({ ...p, slug: v }))} disabled={!!editing} /></div>
          <div><div className="mb-1 text-xs text-neutral-400">{t('assistantTools.fieldName')}</div><Input value={form.name} onChange={(v) => setForm((p) => ({ ...p, name: v }))} /></div>
          <div className="md:col-span-2"><div className="mb-1 text-xs text-neutral-400">{t('assistantTools.fieldDisplayName')}</div><Input value={form.displayName} onChange={(v) => setForm((p) => ({ ...p, displayName: v }))} /></div>
          <div className="md:col-span-2"><div className="mb-1 text-xs text-neutral-400">{t('assistantTools.fieldDescription')}</div><Input.TextArea value={form.description} onChange={(v) => setForm((p) => ({ ...p, description: v }))} /></div>
          <div><div className="mb-1 text-xs text-neutral-400">{t('mcpMarket.fieldCategory')}</div><Select value={form.category} onChange={(v) => setForm((p) => ({ ...p, category: String(v) }))} options={['utility', 'order', 'crm', 'custom'].map((c) => ({ value: c, label: t(`mcpMarket.category.${c}`) }))} /></div>
          <div><div className="mb-1 text-xs text-neutral-400">{t('common.status')}</div><Select value={form.status} onChange={(v) => setForm((p) => ({ ...p, status: String(v) }))} options={[{ value: 'draft', label: 'draft' }, { value: 'published', label: 'published' }, { value: 'archived', label: 'archived' }]} /></div>
          <div className="md:col-span-2"><div className="mb-1 text-xs text-neutral-400">{t('assistantTools.fieldMcpSseUrl')}</div><Input value={form.mcpSseUrl} onChange={(v) => setForm((p) => ({ ...p, mcpSseUrl: v }))} /></div>
          <div className="md:col-span-2"><div className="mb-1 text-xs text-neutral-400">{t('mcpMarket.fieldTags')}</div><Input value={form.tags} onChange={(v) => setForm((p) => ({ ...p, tags: v }))} placeholder={t('mcpMarket.fieldTagsPlaceholder')} /></div>
          <div className="md:col-span-2">
            <div className="mb-1 text-xs text-neutral-400">{t('mcpMarket.fieldLogo')}</div>
            <div className="flex items-center gap-3">
              {form.logoUrl ? <img src={resolveLogoUrl(form.logoUrl)} alt="" className="h-12 w-12 rounded border object-cover" /> : null}
              <Upload accept="image/jpeg,image/png" showUploadList={false} beforeUpload={(file) => { void handleLogoUpload(file); return false }}>
                <Button type="outline" loading={logoUploading}>{t('mcpMarket.uploadLogo')}</Button>
              </Upload>
            </div>
          </div>
        </div>
      </ArcoModal>
    </BaseLayout>
  )
}
