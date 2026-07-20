import { useCallback, useEffect, useState } from 'react'
import { Drawer, InputNumber, Modal } from '@arco-design/web-react'
import { Plus, Pencil, Trash2, RefreshCw } from 'lucide-react'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Button, DataList, Input, Select } from '@/components/ui'
import type { DataListColumn } from '@/components/ui'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useTranslation } from '@/i18n'
import {
  createBillingPlan, deleteBillingPlan, listBillingPlans, updateBillingPlan, type BillingPlanRow,
} from '@/api/platformCost'

type FormState = { name: string; description: string; currency: string; callRatePerMinute: number; llmRatePer1kTokens: number; status: string }
const defaultForm = (): FormState => ({ name: '', description: '', currency: 'CNY', callRatePerMinute: 0, llmRatePer1kTokens: 0, status: 'active' })

export default function BillingPlan() {
  const { t } = useTranslation()
  const [rows, setRows] = useState<BillingPlanRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [nameQ, setNameQ] = useState('')
  const [loading, setLoading] = useState(false)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)
  const [form, setForm] = useState<FormState>(defaultForm)
  const [saving, setSaving] = useState(false)
  const [delTarget, setDelTarget] = useState<BillingPlanRow | null>(null)
  const [delLoading, setDelLoading] = useState(false)
  const pageSize = 20

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listBillingPlans(page, pageSize, nameQ.trim() || undefined)
      if (res.code === 200 && res.data) { setRows(res.data.list || []); setTotal(res.data.total || 0) }
      else showAlert(res.msg || t('common.loadFailed'), 'error')
    } catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('common.loadFailed')), 'error') }
    finally { setLoading(false) }
  }, [page, nameQ, t])

  useEffect(() => { void load() }, [load])

  const openCreate = () => { setEditingId(null); setForm(defaultForm()); setDrawerOpen(true) }
  const openEdit = (r: BillingPlanRow) => {
    setEditingId(r.id)
    setForm({ name: r.name, description: r.description || '', currency: r.currency || 'CNY', callRatePerMinute: r.callRatePerMinute ?? 0, llmRatePer1kTokens: r.llmRatePer1kTokens ?? 0, status: r.status || 'active' })
    setDrawerOpen(true)
  }

  const save = async () => {
    const name = form.name.trim()
    if (!name) { showAlert('方案名称不能为空', 'error'); return }
    setSaving(true)
    try {
      const body = { name, description: form.description.trim() || undefined, currency: form.currency.trim() || 'CNY', callRatePerMinute: form.callRatePerMinute, llmRatePer1kTokens: form.llmRatePer1kTokens, status: form.status }
      const res = editingId == null ? await createBillingPlan(body) : await updateBillingPlan(editingId, body)
      if (res.code === 200) { showAlert(t('common.saveSuccess'), 'success'); setDrawerOpen(false); void load() }
      else showAlert(res.msg || t('common.saveFailed'), 'error')
    } catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('common.saveFailed')), 'error') }
    finally { setSaving(false) }
  }

  const confirmDelete = async () => {
    if (!delTarget) return
    setDelLoading(true)
    try {
      const res = await deleteBillingPlan(delTarget.id)
      if (res.code === 200) { showAlert(t('common.deleteSuccess'), 'success'); setDelTarget(null); void load() }
      else showAlert(res.msg || t('common.deleteFailed'), 'error')
    } catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('common.deleteFailed')), 'error') }
    finally { setDelLoading(false) }
  }

  const columns: DataListColumn<Record<string, unknown>>[] = [
    { key: 'name', title: '名称', render: (_, r) => <span className="truncate text-sm font-medium text-neutral-900">{String(r.name || '—')}</span> },
    { key: 'currency', title: '币种', width: 80, render: (_, r) => <span className="text-sm text-neutral-700">{String(r.currency || '—')}</span> },
    { key: 'callRate', title: '会话/分钟', width: 110, render: (_, r) => <span className="font-mono text-sm text-neutral-900">¥ {Number(r.callRatePerMinute || 0).toFixed(4)}</span> },
    { key: 'llmRate', title: 'LLM/千token', width: 120, render: (_, r) => <span className="font-mono text-sm text-neutral-900">¥ {Number(r.llmRatePer1kTokens || 0).toFixed(4)}</span> },
    { key: 'status', title: '状态', width: 90, render: (_, r) => <span className="text-sm text-neutral-700">{String(r.status || '—')}</span> },
    {
      key: 'actions', title: '操作', width: 120, align: 'right',
      render: (_, r) => (
        <div className="flex items-center justify-end gap-1">
          <Button size="mini" icon={<Pencil size={12} />} onClick={() => openEdit(r as unknown as BillingPlanRow)}>编辑</Button>
          <Button size="mini" status="danger" icon={<Trash2 size={12} />} onClick={() => setDelTarget(r as unknown as BillingPlanRow)}>删除</Button>
        </div>
      ),
    },
  ]

  return (
    <BaseLayout title={t('nav.billingPlan')} description="维护默认用量与 LLM 单价方案">
      <DataList
        data={rows as unknown as (BillingPlanRow & Record<string, unknown>)[]}
        columns={columns}
        loading={loading}
        rowKey="id"
        emptyText="暂无数据"
        pagination={{ current: page, pageSize, total, onChange: (p) => setPage(p) }}
        header={
          <div className="flex items-center justify-between gap-3">
            <Input allowClear placeholder="方案名称" style={{ width: 200 }} value={nameQ} onChange={setNameQ} />
            <div className="flex items-center gap-2">
              <Button type="outline" icon={<RefreshCw size={14} />} onClick={() => { setPage(1); void load() }}>搜索</Button>
              <Button type="primary" icon={<Plus size={14} />} onClick={openCreate}>新建方案</Button>
            </div>
          </div>
        }
      />

      <Drawer
        title={editingId == null ? '新建成本方案' : '编辑成本方案'}
        visible={drawerOpen} width={480}
        onCancel={() => { if (!saving) setDrawerOpen(false) }}
        footer={<div className="flex justify-end gap-2"><Button type="outline" onClick={() => setDrawerOpen(false)} disabled={saving}>取消</Button><Button type="primary" loading={saving} onClick={() => void save()}>保存</Button></div>}
      >
        <div className="space-y-4">
          <div><label className="mb-1 block text-sm text-neutral-500">名称 *</label><Input value={form.name} onChange={(v) => setForm((f) => ({ ...f, name: v }))} /></div>
          <div><label className="mb-1 block text-sm text-neutral-500">描述</label><Input.TextArea value={form.description} onChange={(v) => setForm((f) => ({ ...f, description: v }))} /></div>
          <div><label className="mb-1 block text-sm text-neutral-500">币种</label><Input value={form.currency} onChange={(v) => setForm((f) => ({ ...f, currency: v }))} /></div>
          <div><label className="mb-1 block text-sm text-neutral-500">会话单价（元/分钟）</label><InputNumber min={0} step={0.01} style={{ width: '100%' }} value={form.callRatePerMinute} onChange={(v) => setForm((f) => ({ ...f, callRatePerMinute: Number(v) || 0 }))} /></div>
          <div><label className="mb-1 block text-sm text-neutral-500">LLM 单价（元/千 token）</label><InputNumber min={0} step={0.001} style={{ width: '100%' }} value={form.llmRatePer1kTokens} onChange={(v) => setForm((f) => ({ ...f, llmRatePer1kTokens: Number(v) || 0 }))} /></div>
          <div><label className="mb-1 block text-sm text-neutral-500">状态</label><Select value={form.status} onChange={(v) => setForm((f) => ({ ...f, status: String(v) }))} options={[{ label: '启用', value: 'active' }, { label: '停用', value: 'disabled' }]} /></div>
        </div>
      </Drawer>

      <Modal title="确认删除" visible={!!delTarget} onCancel={() => { if (!delLoading) setDelTarget(null) }}
        footer={<div className="flex justify-end gap-2"><Button type="outline" onClick={() => setDelTarget(null)} disabled={delLoading}>取消</Button><Button status="danger" loading={delLoading} onClick={() => void confirmDelete()}>删除</Button></div>}
      >
        确定删除方案「{delTarget?.name}」？已绑定线路需另行调整。
      </Modal>
    </BaseLayout>
  )
}
