import { useCallback, useEffect, useMemo, useState } from 'react'
import { Form, Input, InputNumber, Modal, Select, Switch, Tabs, Tag, Drawer, Upload } from '@arco-design/web-react'
import { Link as RouterLink, useNavigate, useSearchParams } from 'react-router-dom'
import { Pencil, Trash2, Radar, Store, Upload as UploadIcon, Eye, ChevronDown, Search, PackageMinus, RefreshCw, Plus } from 'lucide-react'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Button, DataList } from '@/components/ui'
import type { DataListColumn } from '@/components/ui'
import ToolSchemaDetail from '@/components/mcp/ToolSchemaDetail'
import { httpToolSchema, mcpToolSchema, schemaParamRows } from '@/components/mcp/toolSchema'
import {
  createAssistantTool, deleteAssistantTool, discoverAssistantTool, listAssistantTools, updateAssistantTool,
  type AssistantToolRow, type AssistantToolWriteBody, type DiscoveredMCPTool,
} from '@/api/assistantTools'
import { delistCustomMcpFromMarket, publishCustomMcpToMarket, uploadMcpMarketLogo } from '@/api/mcpMarket'
import { getUploadsBaseURL } from '@/config/apiConfig'
import { useTranslation } from '@/i18n'
import { showAlert } from '@/utils/notification'
import { cn } from '@/utils/cn'
import { extractApiErrorMessage } from '@/utils/apiError'

const FormItem = Form.Item
const Option = Select.Option
const TextArea = Input.TextArea
const TabPane = Tabs.TabPane

type MineTab = 'activated' | 'custom'
const METHOD_OPTIONS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE']

function defaultParametersJSON(): string {
  return JSON.stringify({ type: 'object', properties: { orderId: { type: 'string', description: '订单号' } }, required: ['orderId'] }, null, 2)
}

function safeParseJSONObject(raw: string, label: string): Record<string, unknown> | undefined {
  const s = raw.trim(); if (!s) return undefined
  try { const v = JSON.parse(s); if (!v || typeof v !== 'object' || Array.isArray(v)) throw new Error(`${label} must be a JSON object`); return v as Record<string, unknown> }
  catch (e) { throw new Error(e instanceof Error ? e.message : String(e)) }
}

function headersToLines(headers?: Record<string, string> | null): string {
  if (!headers) return ''; return Object.entries(headers).map(([k, v]) => `${k}: ${v}`).join('\n')
}

function linesToHeaders(raw: string): Record<string, string> | undefined {
  const out: Record<string, string> = {}
  for (const line of raw.split('\n')) { const t = line.trim(); if (!t) continue; const i = t.indexOf(':'); if (i <= 0) continue; const k = t.slice(0, i).trim(); const v = t.slice(i + 1).trim(); if (k) out[k] = v }
  return Object.keys(out).length ? out : undefined
}

export default function AssistantToolsPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = (searchParams.get('tab') === 'custom' ? 'custom' : 'activated') as MineTab
  const [loading, setLoading] = useState(true)
  const [rows, setRows] = useState<AssistantToolRow[]>([])
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<AssistantToolRow | null>(null)
  const [saving, setSaving] = useState(false)
  const [discoveringId, setDiscoveringId] = useState<string | null>(null)
  const [publishingId, setPublishingId] = useState<string | null>(null)
  const [delistingId, setDelistingId] = useState<string | null>(null)
  const [publishRow, setPublishRow] = useState<AssistantToolRow | null>(null)
  const [publishLogoUrl, setPublishLogoUrl] = useState('')
  const [publishLogoUploading, setPublishLogoUploading] = useState(false)
  const [toolDetailSearch, setToolDetailSearch] = useState('')
  const [detailRow, setDetailRow] = useState<AssistantToolRow | null>(null)
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set())
  const [form] = Form.useForm()
  const [publishForm] = Form.useForm()
  const kindWatch = Form.useWatch('kind', form) as string | undefined

  const setTab = (next: string) => setSearchParams(next === 'custom' ? { tab: 'custom' } : { tab: 'activated' })

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const source = tab === 'activated' ? 'market' : 'custom'
      const res = await listAssistantTools({ source })
      if (res.code !== 200) { showAlert(res.msg || t('assistantTools.loadFailed'), 'error'); return }
      setRows(Array.isArray(res.data) ? res.data : [])
    } catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('assistantTools.loadFailed')), 'error') }
    finally { setLoading(false) }
  }, [t, tab])

  useEffect(() => { void load() }, [load])

  const openCreate = (kind: 'http' | 'mcp_sse' = 'mcp_sse') => {
    setEditing(null); form.resetFields()
    if (kind === 'mcp_sse') { form.setFieldsValue({ kind: 'mcp_sse', name: 'lingmcp', displayName: t('assistantTools.defaultMcpDisplayName'), description: t('assistantTools.defaultMcpDescription'), mcpSseUrl: 'http://127.0.0.1:3920/sse', timeoutMs: 8000, enabled: true, headersText: '' }) }
    else { form.setFieldsValue({ kind: 'http', name: 'order_lookup', displayName: t('assistantTools.defaultDisplayName'), description: t('assistantTools.defaultDescription'), method: 'POST', url: 'https://crm.example.com/api/orders/lookup', bodyTemplate: '{"orderId":"{{orderId}}"}', timeoutMs: 10000, enabled: true, headersText: '', parametersText: defaultParametersJSON() }) }
    setModalOpen(true)
  }

  const openEdit = (row: AssistantToolRow) => {
    setEditing(row); form.setFieldsValue({ kind: row.kind || 'http', name: row.name, displayName: row.displayName || '', description: row.description || '', method: (row.method || 'POST').toUpperCase(), url: row.url || '', mcpSseUrl: row.mcpSseUrl || '', bodyTemplate: row.bodyTemplate || '', timeoutMs: row.timeoutMs || 8000, enabled: row.enabled !== false, headersText: headersToLines(row.headers), parametersText: row.parameters ? JSON.stringify(row.parameters, null, 2) : '' }); setModalOpen(true)
  }

  const handleSave = async () => {
    try {
      const values = await form.validate(); const kind = String(values.kind || 'http')
      let parameters: Record<string, unknown> | undefined
      if (kind === 'http') { try { parameters = safeParseJSONObject(String(values.parametersText || ''), 'parameters') } catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('assistantTools.parametersInvalid')), 'error'); return } }
      const body: AssistantToolWriteBody = { name: String(values.name || '').trim(), displayName: String(values.displayName || '').trim(), description: String(values.description || '').trim(), kind, enabled: !!values.enabled, timeoutMs: Number(values.timeoutMs) || 8000, headers: linesToHeaders(String(values.headersText || '')) }
      if (kind === 'mcp_sse') { body.mcpSseUrl = String(values.mcpSseUrl || '').trim() } else { body.method = String(values.method || 'POST').toUpperCase(); body.url = String(values.url || '').trim(); body.bodyTemplate = String(values.bodyTemplate || ''); body.parameters = parameters }
      setSaving(true)
      const res = editing ? await updateAssistantTool(editing.id, body) : await createAssistantTool(body)
      if (res.code !== 200) { showAlert(res.msg || t('assistantTools.saveFailed'), 'error'); return }
      showAlert(editing ? t('assistantTools.updateOk') : t('assistantTools.createOk'), 'success'); setModalOpen(false); void load()
    } catch (e: unknown) { if (e && typeof e === 'object' && 'errorFields' in e) return; showAlert(extractApiErrorMessage(e, t('assistantTools.saveFailed')), 'error') }
    finally { setSaving(false) }
  }

  const handleDelete = (row: AssistantToolRow) => {
    Modal.confirm({ title: t('assistantTools.deleteConfirmTitle'), content: t('assistantTools.deleteConfirmBody', { name: row.displayName || row.name }), onOk: async () => { try { const res = await deleteAssistantTool(row.id); if (res.code !== 200) { showAlert(res.msg || t('assistantTools.deleteFailed'), 'error'); return } showAlert(t('assistantTools.deleteOk'), 'success'); void load() } catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('assistantTools.deleteFailed')), 'error') } } })
  }

  const handleDiscover = async (row: AssistantToolRow) => {
    setDiscoveringId(row.id)
    try { const res = await discoverAssistantTool(row.id); if (res.code !== 200) { showAlert(res.msg || t('assistantTools.discoverFailed'), 'error'); return } showAlert(t('assistantTools.discoverOk', { count: res.data?.discoveredTools?.length ?? 0 }), 'success'); if (res.data?.tool) { setDetailRow(res.data.tool); setExpandedIds((ids) => new Set([...ids, row.id])) } void load() }
    catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('assistantTools.discoverFailed')), 'error') }
    finally { setDiscoveringId(null) }
  }

  const handlePublish = (row: AssistantToolRow) => { setPublishRow(row); setPublishLogoUrl(''); publishForm.setFieldsValue({ slug: row.name, displayName: row.displayName || row.name, description: row.description || '', category: 'custom', version: '1.0.0', tags: '' }) }

  const handlePublishSubmit = async () => {
    if (!publishRow) return
    try { const values = await publishForm.validate(); setPublishingId(publishRow.id)
      const res = await publishCustomMcpToMarket({ toolId: publishRow.id, slug: String(values.slug || '').trim(), displayName: String(values.displayName || '').trim(), description: String(values.description || '').trim(), category: String(values.category || 'custom'), version: String(values.version || '').trim(), logoUrl: publishLogoUrl || undefined, tags: String(values.tags || '').trim(), publish: true })
      if (res.code !== 200) { showAlert(res.msg || t('mcpMarket.publishFailed'), 'error'); return }
      showAlert(t('mcpMarket.publishOk'), 'success'); setPublishRow(null); void load()
    } catch (e: unknown) { if (e && typeof e === 'object' && 'errorFields' in e) return; showAlert(extractApiErrorMessage(e, t('mcpMarket.publishFailed')), 'error') }
    finally { setPublishingId(null) }
  }

  const handleDelist = (row: AssistantToolRow) => {
    Modal.confirm({ title: t('mcpMarket.delistTitle'), content: t('mcpMarket.delistBody', { name: row.displayName || row.name }), onOk: async () => { setDelistingId(row.id); try { const res = await delistCustomMcpFromMarket(row.id); if (res.code !== 200) { showAlert(res.msg || t('mcpMarket.delistFailed'), 'error'); return } showAlert(t('mcpMarket.delistOk'), 'success'); void load() } catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('mcpMarket.delistFailed')), 'error') } finally { setDelistingId(null) } } })
  }

  const handlePublishLogo = async (file: File) => {
    if (!['image/jpeg', 'image/png'].includes(file.type)) { showAlert(t('mcpMarket.logoFormatInvalid'), 'error'); return false }
    if (file.size > 5 * 1024 * 1024) { showAlert(t('mcpMarket.logoTooLarge'), 'error'); return false }
    setPublishLogoUploading(true)
    try { const res = await uploadMcpMarketLogo(file); if (res.code !== 200 || !res.data?.logoUrl) { showAlert(res.msg || t('mcpMarket.logoUploadFailed'), 'error'); return false } setPublishLogoUrl(res.data.logoUrl); return false }
    catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('mcpMarket.logoUploadFailed')), 'error'); return false }
    finally { setPublishLogoUploading(false) }
  }

  const resolveLogoUrl = (url?: string) => { if (!url) return ''; const u = url.trim(); if (/^https?:\/\//i.test(u)) return u; const base = getUploadsBaseURL().replace(/\/$/, ''); return u.startsWith('/') ? `${base}${u}` : `${base}/${u}` }

  const filteredDiscoveredTools = useMemo(() => { const tools = detailRow?.discoveredTools || []; const q = toolDetailSearch.trim().toLowerCase(); if (!q) return tools; return tools.filter((dt) => dt.name.toLowerCase().includes(q) || (dt.description || '').toLowerCase().includes(q)) }, [detailRow, toolDetailSearch])

  const toggleEnabled = async (row: AssistantToolRow, enabled: boolean) => {
    try { const res = await updateAssistantTool(row.id, { enabled }); if (res.code !== 200) { showAlert(res.msg || t('assistantTools.saveFailed'), 'error'); return } void load() }
    catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('assistantTools.saveFailed')), 'error') }
  }

  const toggleExpand = (id: string) => setExpandedIds((ids) => { const next = new Set(ids); if (next.has(id)) next.delete(id); else next.add(id); return next })

  const renderDiscoveredToolsTable = (tools: DiscoveredMCPTool[]) => (
    <div className="space-y-3">
      {tools.map((dt) => {
        const schema = mcpToolSchema(dt); const params = schemaParamRows(schema)
        return (
          <div key={dt.name} className="rounded-xl border border-neutral-100 bg-white overflow-hidden">
            <div className="px-4 py-3 border-b border-neutral-100 bg-neutral-50/80">
              <Tag size="small" color="arcoblue" className="!font-mono !text-xs !rounded-full">{dt.name}</Tag>
              <p className="mt-2 text-sm text-neutral-700 leading-relaxed whitespace-pre-wrap">{dt.description?.trim() || '—'}</p>
            </div>
            {params.length > 0 ? (
              <div className="px-4 py-3"><div className="mb-2 text-xs font-medium text-neutral-500">{t('assistantTools.schemaParams')}</div><ToolSchemaDetail schema={schema} compact /></div>
            ) : null}
          </div>
        )
      })}
    </div>
  )

  const expandedRowRender = (r: AssistantToolRow) => {
    if ((r.kind || '') === 'mcp_sse') { const tools = r.discoveredTools || []; if (!tools.length) return <p className="text-xs text-neutral-400">{t('assistantTools.discoverFirst')}</p>; return renderDiscoveredToolsTable(tools) }
    if ((r.kind || '') === 'http' && r.parameters) return <div className="py-1"><div className="mb-1 text-xs font-medium text-neutral-500">{t('assistantTools.schemaParams')}</div><ToolSchemaDetail schema={httpToolSchema(r)} /></div>
    return null
  }

  const isMcp = (kindWatch || editing?.kind || 'http') === 'mcp_sse'

  const columns: DataListColumn<Record<string, unknown>>[] = [
    {
      key: 'info', title: t('assistantTools.colName'),
      render: (_, r) => (
        <div>
          <div className="font-medium text-sm text-neutral-900">{String(r.displayName || r.name)}</div>
          <div className="text-xs text-neutral-400 font-mono">{String(r.name)}</div>
        </div>
      ),
    },
    {
      key: 'kind', title: t('assistantTools.colKind'), width: 100,
      render: (_, r) => <Tag size="small" color={String(r.kind) === 'mcp_sse' ? 'purple' : 'arcoblue'} className="!rounded-full">{String(r.kind) === 'mcp_sse' ? 'MCP' : 'HTTP'}</Tag>,
    },
    {
      key: 'endpoint', title: t('assistantTools.colEndpoint'),
      render: (_, r) => {
        if (String(r.kind || 'http') === 'mcp_sse') {
          const n = (r.discoveredTools as unknown[])?.length ?? 0
          return (
            <span className="font-mono text-xs text-neutral-700">
              {String(r.mcpSseUrl || '—')}
              {n > 0 ? <Tag size="small" color="green" className="!ml-1.5 !rounded-full">{t('assistantTools.toolsCached', { count: n })}</Tag> : null}
            </span>
          )
        }
        return (
          <span className="font-mono text-xs text-neutral-700">
            <Tag size="small" color="arcoblue" className="!mr-1.5 !rounded-full">{String(r.method || 'POST').toUpperCase()}</Tag>
            {String(r.url || '—')}
          </span>
        )
      },
    },
    {
      key: 'enabled', title: t('assistantTools.colEnabled'), width: 80,
      render: (_, r) => <Switch size="small" checked={r.enabled !== false} onChange={(on) => void toggleEnabled(r as unknown as AssistantToolRow, on)} />,
    },
    {
      key: 'actions', title: t('assistantTools.colActions'), width: 260, align: 'right',
      render: (_, r) => {
        const row = r as unknown as AssistantToolRow
        return (
          <div className="flex items-center justify-end gap-1">
            {(row.kind || '') === 'mcp_sse' && (row.discoveredTools?.length ?? 0) > 0 ? (
              <Button size="mini" icon={<Eye size={12} />} onClick={(e) => { e.stopPropagation(); setDetailRow(row) }}>{t('assistantTools.viewTools')}</Button>
            ) : null}
            {(row.kind || '') === 'mcp_sse' ? (
              <Button size="mini" icon={<Radar size={12} />} loading={discoveringId === row.id} onClick={(e) => { e.stopPropagation(); void handleDiscover(row) }}>{t('assistantTools.discover')}</Button>
            ) : null}
            {tab === 'custom' && (row.kind || '') === 'mcp_sse' ? (
              row.marketPublished ? (
                <Button size="mini" icon={<PackageMinus size={12} />} loading={delistingId === row.id} onClick={(e) => { e.stopPropagation(); handleDelist(row) }}>{t('mcpMarket.delist')}</Button>
              ) : (
                <Button size="mini" icon={<UploadIcon size={12} />} loading={publishingId === row.id} onClick={(e) => { e.stopPropagation(); handlePublish(row) }}>{t('mcpMarket.publish')}</Button>
              )
            ) : null}
            <Button size="mini" icon={<Pencil size={12} />} onClick={(e) => { e.stopPropagation(); openEdit(row) }}>{t('assistantTools.edit')}</Button>
            <Button size="mini" status="danger" icon={<Trash2 size={12} />} onClick={(e) => { e.stopPropagation(); handleDelete(row) }}>{t('assistantTools.delete')}</Button>
          </div>
        )
      },
    },
  ]

  return (
    <BaseLayout title={t('pages.myMcp.title')} description={t('pages.myMcp.description')}>
      <p className="mb-4 text-xs text-neutral-500">
        {t('myMcp.marketHint')}{' '}
        <RouterLink className="text-primary underline" to="/mcp-market">{t('nav.mcpMarket')}</RouterLink>
        {' · '}
        {t('assistantTools.bindHint')}{' '}
        <RouterLink className="text-primary underline" to="/assistant-manager">{t('nav.assistantManager')}</RouterLink>
      </p>

      <Tabs activeTab={tab} onChange={setTab} type="rounded" className="mb-4">
        <TabPane key="activated" title={t('myMcp.tabActivated')} />
        <TabPane key="custom" title={t('myMcp.tabCustom')} />
      </Tabs>

      <DataList
        data={rows as unknown as (AssistantToolRow & Record<string, unknown>)[]}
        columns={columns}
        loading={loading}
        rowKey="id"
        emptyText={tab === 'activated' ? t('myMcp.emptyActivated') : t('assistantTools.empty')}
        header={
          <div className="flex items-center justify-end gap-2">
            {tab === 'custom' ? <Button type="primary" icon={<Plus size={14} />} onClick={() => openCreate('mcp_sse')}>{t('assistantTools.create')}</Button> : null}
            <Button type="outline" icon={<Store size={14} />} onClick={() => navigate('/mcp-market')}>{t('nav.mcpMarket')}</Button>
            <Button type="outline" icon={<RefreshCw size={14} />} loading={loading} onClick={() => void load()}>{t('common.refresh')}</Button>
          </div>
        }
        renderRow={(row) => {
          const r = row as unknown as AssistantToolRow
          const isExpanded = expandedIds.has(r.id)
          const hasExpandable = ((r.kind || '') === 'mcp_sse' && (r.discoveredTools?.length ?? 0) > 0) || ((r.kind || '') === 'http' && !!r.parameters)
          return (
            <div>
              <div
                className={cn('flex items-center gap-4 px-4 py-3 transition-colors', hasExpandable ? 'cursor-pointer hover:bg-neutral-50' : '')}
                onClick={hasExpandable ? () => toggleExpand(r.id) : undefined}
              >
                {hasExpandable ? (
                  <ChevronDown size={14} className={cn('shrink-0 text-neutral-400 transition-transform', isExpanded && 'rotate-180')} />
                ) : (
                  <div className="w-3.5 shrink-0" />
                )}
                {columns.map((col) => (
                  <div key={col.key} className={cn('min-w-0 shrink-0', col.align === 'center' && 'text-center', col.align === 'right' && 'text-right', col.className)} style={{ width: col.width, minWidth: col.minWidth, flex: col.width ? undefined : 1 }}>
                    {col.render ? col.render(row[col.key], row, 0) : <span className="truncate text-sm text-neutral-900">{String(row[col.key] ?? '—')}</span>}
                  </div>
                ))}
              </div>
              <div
                className={cn('overflow-hidden transition-all duration-200 ease-in-out', isExpanded ? 'max-h-[600px] opacity-100' : 'max-h-0 opacity-0')}
              >
                <div className="border-t border-neutral-100 bg-neutral-50/50 px-4 py-2">
                  {expandedRowRender(r)}
                </div>
              </div>
            </div>
          )
        }}
      />

      <Drawer title={editing ? t('assistantTools.editTitle') : t('assistantTools.createTitle')} visible={modalOpen} placement="right" width={640} onCancel={() => setModalOpen(false)} footer={<div className="flex justify-end gap-2"><Button onClick={() => setModalOpen(false)} disabled={saving}>{t('common.cancel')}</Button><Button type="primary" loading={saving} onClick={() => void handleSave()}>{t('common.save')}</Button></div>}>
        <p className="mb-3 text-xs text-neutral-500">{isMcp ? t('assistantTools.formHintMcp') : t('assistantTools.formHint')}</p>
        <Form form={form} layout="vertical" requiredSymbol initialValues={{ kind: 'mcp_sse' }}>
          <FormItem label={t('assistantTools.fieldKind')} field="kind" rules={[{ required: true }]}><Select disabled={!!editing}><Option value="mcp_sse">MCP (SSE)</Option><Option value="http">HTTP</Option></Select></FormItem>
          <FormItem label={t('assistantTools.fieldName')} field="name" rules={[{ required: true }]}><Input placeholder={isMcp ? 'lingmcp' : 'order_lookup'} disabled={!!editing} /></FormItem>
          <FormItem label={t('assistantTools.fieldDisplayName')} field="displayName"><Input /></FormItem>
          <FormItem label={t('assistantTools.fieldDescription')} field="description"><TextArea autoSize={{ minRows: 2, maxRows: 4 }} /></FormItem>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <FormItem label={t('assistantTools.fieldTimeout')} field="timeoutMs" extra={t('assistantTools.fieldTimeoutHint')}><InputNumber min={2000} max={10000} step={1000} style={{ width: '100%' }} /></FormItem>
            <FormItem label={t('assistantTools.fieldEnabled')} field="enabled" triggerPropName="checked"><Switch /></FormItem>
          </div>
          {isMcp ? (
            <FormItem label={t('assistantTools.fieldMcpSseUrl')} field="mcpSseUrl" rules={[{ required: true }]}><Input placeholder="http://127.0.0.1:3920/sse" /></FormItem>
          ) : (
            <>
              <FormItem label={t('assistantTools.fieldMethod')} field="method" rules={[{ required: true }]}><Select>{METHOD_OPTIONS.map((m) => <Option key={m} value={m}>{m}</Option>)}</Select></FormItem>
              <FormItem label={t('assistantTools.fieldUrl')} field="url" rules={[{ required: true }]}><Input placeholder="https://api.example.com/orders/lookup" /></FormItem>
              <FormItem label={t('assistantTools.fieldBody')} field="bodyTemplate"><TextArea placeholder={'{"orderId":"{{orderId}}"}'} autoSize={{ minRows: 2, maxRows: 6 }} style={{ fontFamily: 'monospace', fontSize: 12 }} /></FormItem>
              <FormItem label={t('assistantTools.fieldParameters')} field="parametersText"><TextArea autoSize={{ minRows: 5, maxRows: 12 }} style={{ fontFamily: 'monospace', fontSize: 12 }} /></FormItem>
            </>
          )}
          <FormItem label={t('assistantTools.fieldHeaders')} field="headersText"><TextArea placeholder={'Authorization: Bearer xxx'} autoSize={{ minRows: 2, maxRows: 5 }} style={{ fontFamily: 'monospace', fontSize: 12 }} /></FormItem>
        </Form>
      </Drawer>

      <Drawer title={t('assistantTools.toolDetails')} visible={!!detailRow} placement="right" width={720} onCancel={() => { setDetailRow(null); setToolDetailSearch('') }} footer={null}>
        {detailRow && (detailRow.kind || '') === 'mcp_sse' ? (
          <div className="space-y-4">
            <div className="rounded-xl border border-neutral-100 bg-neutral-50 px-4 py-3">
              <div className="text-xs text-neutral-400 mb-1">SSE</div>
              <div className="text-sm font-mono break-all">{detailRow.mcpSseUrl}</div>
              {(detailRow.discoveredTools?.length ?? 0) > 0 ? <Tag size="small" className="mt-2 !rounded-full" color="green">{t('assistant.mcpToolCount', { count: detailRow.discoveredTools?.length ?? 0 })}</Tag> : null}
            </div>
            {(detailRow.discoveredTools?.length ?? 0) > 0 ? (
              <>
                <Input allowClear prefix={<Search size={14} className="text-neutral-400" />} placeholder={t('assistantTools.toolSearchPlaceholder')} value={toolDetailSearch} onChange={setToolDetailSearch} />
                {filteredDiscoveredTools.length > 0 ? renderDiscoveredToolsTable(filteredDiscoveredTools) : <p className="text-sm text-neutral-400">{t('assistantTools.toolSearchEmpty')}</p>}
              </>
            ) : <p className="text-sm text-neutral-400">{t('assistantTools.discoverFirst')}</p>}
          </div>
        ) : detailRow && (detailRow.kind || '') === 'http' ? (
          <div className="space-y-3">
            <p className="text-xs text-neutral-500 font-mono">{(detailRow.method || 'POST').toUpperCase()} {detailRow.url}</p>
            <ToolSchemaDetail schema={httpToolSchema(detailRow)} />
          </div>
        ) : null}
      </Drawer>

      <Modal title={t('mcpMarket.publishTitle')} visible={!!publishRow} onCancel={() => setPublishRow(null)} onOk={() => void handlePublishSubmit()} confirmLoading={publishingId === publishRow?.id} unmountOnExit>
        <Form form={publishForm} layout="vertical" requiredSymbol>
          <FormItem label={t('assistantTools.fieldDisplayName')} field="displayName" rules={[{ required: true }]}><Input /></FormItem>
          <FormItem label={t('mcpMarket.fieldSlug')} field="slug" rules={[{ required: true }]}><Input placeholder="lingmcp" /></FormItem>
          <FormItem label={t('assistantTools.fieldDescription')} field="description"><TextArea autoSize={{ minRows: 2, maxRows: 4 }} /></FormItem>
          <FormItem label={t('mcpMarket.fieldCategory')} field="category"><Select>{['custom', 'order', 'crm', 'utility'].map((c) => <Option key={c} value={c}>{t(`mcpMarket.category.${c}`)}</Option>)}</Select></FormItem>
          <FormItem label={t('mcpMarket.fieldTags')} field="tags" extra={t('mcpMarket.fieldTagsHint')}><Input placeholder={t('mcpMarket.fieldTagsPlaceholder')} /></FormItem>
          <FormItem label={t('mcpMarket.fieldLogo')} extra={t('mcpMarket.fieldLogoHint')}>
            <div className="flex items-center gap-3">
              {publishLogoUrl ? <img src={resolveLogoUrl(publishLogoUrl)} alt="logo" className="h-12 w-12 rounded-lg border border-neutral-100 object-cover" /> : null}
              <Upload accept="image/jpeg,image/png" showUploadList={false} beforeUpload={(file) => { void handlePublishLogo(file); return false }}>
                <Button type="outline" loading={publishLogoUploading}>{t('mcpMarket.uploadLogo')}</Button>
              </Upload>
            </div>
          </FormItem>
        </Form>
      </Modal>
    </BaseLayout>
  )
}
