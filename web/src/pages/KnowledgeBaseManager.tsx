import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { ChevronDown, ChevronLeft } from 'lucide-react'
import Card from '@/components/UI/Card'
import Input from '@/components/UI/Input'
import Button from '@/components/UI/Button'
import Modal from '@/components/UI/Modal'
import ConfirmDialog from '@/components/UI/ConfirmDialog'
import FileUpload from '@/components/UI/FileUpload'
import Progress from '@/components/UI/Progress'
import CollapsibleSectionHeader from '@/components/UI/CollapsibleSectionHeader'
import { showAlert } from '@/utils/notification'
import { cn } from '@/utils/cn'
import {
  createKnowledgeBase,
  deleteKnowledgeBase,
  deleteKnowledgeDocument,
  fetchKnowledgeBaseById,
  listKnowledgeBases,
  listKnowledgeDocuments,
  listSupportedDocumentTypes,
  recallTestKnowledgeBase,
  type KnowledgeBase,
  type KnowledgeDocumentItem,
  type SupportedDocumentTypesResponse,
  uploadKnowledgeDocumentWithProgress,
} from '@/api/knowledgeBase'

function maskSecret(value: string): string {
  const s = value.trim()
  if (!s) return '未配置'
  if (s.length <= 8) return '••••••••'
  return `${s.slice(0, 4)}…${s.slice(-4)}`
}

function ReadOnlyRow({
  label,
  value,
  mono,
  subdued,
}: {
  label: string
  value: string
  mono?: boolean
  subdued?: boolean
}) {
  return (
    <div className="grid grid-cols-1 gap-0.5 sm:grid-cols-[minmax(0,140px)_1fr] sm:gap-3 text-sm">
      <div className="text-muted-foreground shrink-0">{label}</div>
      <div
        className={cn(
          'break-all text-foreground/90',
          mono && 'font-mono text-xs',
          subdued && 'text-muted-foreground',
        )}
      >
        {value || '—'}
      </div>
    </div>
  )
}

function ConfigFoldSection({
  title,
  defaultOpen,
  children,
}: {
  title: string
  defaultOpen?: boolean
  children: ReactNode
}) {
  return (
    <details
      open={defaultOpen}
      className="group rounded-lg border border-border bg-card/30 first:mt-0 mt-2"
    >
      <summary
        className="flex cursor-pointer list-none items-center gap-2 px-3 py-2.5 text-sm font-medium text-foreground hover:bg-muted/40 [&::-webkit-details-marker]:hidden"
      >
        <ChevronDown className="h-4 w-4 shrink-0 text-muted-foreground transition-transform group-open:rotate-180" />
        {title}
      </summary>
      <div className="space-y-2.5 border-t border-border px-3 py-3">{children}</div>
    </details>
  )
}

const emptyForm = {
  name: '',
  description: '',
  provider: 'qdrant',
  endpointUrl: 'http://localhost:6333',
  apiKey: '',
  apiSecret: '',
  indexName: '',
  namespace: 'ling_kb',
  embeddingUrl: 'https://integrate.api.nvidia.com/v1/embeddings',
  embeddingKey: '',
  embeddingModel: 'nvidia/nv-embed-v1',
}

type FormState = typeof emptyForm

const KnowledgeBaseManager = () => {
  const [view, setView] = useState<'list' | 'detail'>('list')
  const [detailId, setDetailId] = useState<number | null>(null)

  const [list, setList] = useState<KnowledgeBase[]>([])
  const [loading, setLoading] = useState(false)
  const [supported, setSupported] = useState<SupportedDocumentTypesResponse | null>(null)

  const [createOpen, setCreateOpen] = useState(false)
  const [createForm, setCreateForm] = useState<FormState>(emptyForm)

  const [detailForm, setDetailForm] = useState<FormState>(emptyForm)
  const [detailLoading, setDetailLoading] = useState(false)

  const [query, setQuery] = useState('')
  const [topK, setTopK] = useState(5)
  const [recallResult, setRecallResult] = useState<any[]>([])

  const [docList, setDocList] = useState<KnowledgeDocumentItem[]>([])
  const [docsLoading, setDocsLoading] = useState(false)
  const [docsMeta, setDocsMeta] = useState({ totalDocs: 0, totalChunks: 0 })

  const [uploading, setUploading] = useState(false)
  const [uploadPercent, setUploadPercent] = useState(0)
  const [uploadMessage, setUploadMessage] = useState('')

  const [kbDeletePending, setKbDeletePending] = useState<{ id: number; name: string } | null>(null)
  const [docDeletePending, setDocDeletePending] = useState<{ docId: string; label: string } | null>(null)

  const acceptExtensions = useMemo(() => {
    if (!supported?.formats?.length) {
      return '.txt,.md,.pdf,.docx,.pptx,.xlsx,.csv,.html,.json,.yaml,.yml,.eml,.rtf,.png,.jpg,.jpeg'
    }
    return supported.formats.map(f => f.extension).join(',')
  }, [supported])

  const loadList = useCallback(async () => {
    try {
      setLoading(true)
      const res = await listKnowledgeBases()
      if (res.code === 200) {
        setList(res.data || [])
      }
    } catch (e: any) {
      showAlert(e?.msg || '加载知识库失败', 'error')
    } finally {
      setLoading(false)
    }
  }, [])

  const loadSupported = useCallback(async () => {
    try {
      const res = await listSupportedDocumentTypes()
      if (res.code === 200 && res.data) {
        setSupported(res.data)
      }
    } catch {
      /* 非致命 */
    }
  }, [])

  useEffect(() => {
    loadList()
    loadSupported()
  }, [loadList, loadSupported])

  const kbToForm = (kb: KnowledgeBase): FormState => ({
    name: kb.name || '',
    description: kb.description || '',
    provider: kb.provider || 'qdrant',
    endpointUrl: kb.endpointUrl || '',
    apiKey: kb.apiKey || '',
    apiSecret: kb.apiSecret || '',
    indexName: kb.indexName || '',
    namespace: kb.namespace || '',
    embeddingUrl: kb.extraConfig?.embeddingUrl || '',
    embeddingKey: kb.extraConfig?.embeddingKey || '',
    embeddingModel: kb.extraConfig?.embeddingModel || '',
  })

  useEffect(() => {
    if (view !== 'detail' || !detailId) return
    let cancelled = false
    ;(async () => {
      try {
        setDetailLoading(true)
        const res = await fetchKnowledgeBaseById(detailId)
        if (cancelled) return
        if (res.code === 200 && res.data) {
          setDetailForm(kbToForm(res.data))
        }
      } catch (e: any) {
        if (!cancelled) showAlert(e?.msg || '加载详情失败', 'error')
      } finally {
        if (!cancelled) setDetailLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [view, detailId])

  const loadDocuments = useCallback(async () => {
    if (!detailId) return
    try {
      setDocsLoading(true)
      const res = await listKnowledgeDocuments(detailId)
      if (res.code === 200 && res.data) {
        setDocList(res.data.items || [])
        setDocsMeta({
          totalDocs: res.data.totalDocs ?? (res.data.items?.length || 0),
          totalChunks: res.data.totalChunks ?? 0,
        })
      }
    } catch (e: any) {
      showAlert(e?.msg || '加载文档列表失败', 'error')
    } finally {
      setDocsLoading(false)
    }
  }, [detailId])

  useEffect(() => {
    if (view !== 'detail' || !detailId || detailLoading) return
    void loadDocuments()
  }, [view, detailId, detailLoading, loadDocuments])

  const openCreateModal = () => {
    setCreateForm({ ...emptyForm })
    setCreateOpen(true)
  }

  const onCreateSubmit = async () => {
    if (!createForm.name.trim()) return showAlert('请填写知识库名称', 'error')
    try {
      const res = await createKnowledgeBase(createForm)
      if (res.code === 200 && res.data?.id) {
        showAlert('创建成功', 'success')
        setCreateOpen(false)
        await loadList()
        setDetailId(res.data.id)
        setView('detail')
      }
    } catch (e: any) {
      showAlert(e?.msg || '创建失败', 'error')
    }
  }

  const confirmDeleteKB = async () => {
    if (!kbDeletePending) return
    const { id } = kbDeletePending
    try {
      const res = await deleteKnowledgeBase(id)
      if (res.code === 200) {
        showAlert('删除成功', 'success')
        if (detailId === id) {
          setView('list')
          setDetailId(null)
        }
        await loadList()
        return
      }
      showAlert(res.msg || '删除失败', 'error')
      throw new Error('kb-delete')
    } catch (e: any) {
      if (e?.message !== 'kb-delete') {
        showAlert(e?.msg || '删除失败', 'error')
      }
      throw e
    }
  }

  const confirmDeleteDoc = async () => {
    if (!detailId || !docDeletePending) return
    try {
      const res = await deleteKnowledgeDocument(detailId, { docId: docDeletePending.docId })
      if (res.code === 200) {
        showAlert(`已删除 ${res.data?.deleted ?? 0} 条切块`, 'success')
        await loadDocuments()
        return
      }
      showAlert(res.msg || '删除文档失败', 'error')
      throw new Error('doc-delete')
    } catch (e: any) {
      if (e?.message !== 'doc-delete') {
        showAlert(e?.msg || '删除文档失败', 'error')
      }
      throw e
    }
  }

  const onUploadFiles = async (files: File[]) => {
    const file = files[0]
    if (!file || !detailId) return
    setUploading(true)
    setUploadPercent(0)
    setUploadMessage('准备上传…')
    try {
      const data = await uploadKnowledgeDocumentWithProgress(detailId, file, {
        onOverallProgress: (pct, msg) => {
          setUploadPercent(pct)
          setUploadMessage(msg)
        },
      })
      showAlert(`上传成功，切块 ${data.chunks} 条`, 'success')
      await loadDocuments()
    } catch (e: any) {
      showAlert(e?.msg || '上传失败', 'error')
    } finally {
      setUploading(false)
      setUploadPercent(0)
      setUploadMessage('')
    }
  }

  const onRecallTest = async () => {
    if (!detailId || !query.trim()) return showAlert('请输入召回问题', 'error')
    try {
      const res = await recallTestKnowledgeBase(detailId, query.trim(), topK)
      if (res.code === 200) {
        setRecallResult(res.data.items || [])
      }
    } catch (e: any) {
      showAlert(e?.msg || '召回实验失败', 'error')
    }
  }

  const detailTitle = detailForm.name || '知识库详情'

  return (
    <>
      {view === 'detail' && detailId ? (
      <div className="p-6 space-y-6">
        <div className="flex items-center gap-3">
          <Button variant="outline" className="gap-1" onClick={() => { setView('list'); setDetailId(null) }}>
            <ChevronLeft className="w-4 h-4" />
            返回列表
          </Button>
          <h1 className="text-xl font-semibold truncate">{detailTitle}</h1>
        </div>

        {detailLoading ? (
          <div className="text-sm text-gray-500">加载详情中…</div>
        ) : (
          <>
            <Card className="p-4 space-y-2">
              <div className="text-sm font-medium">基础配置</div>
              <p className="text-xs text-muted-foreground">
                只读展示；密钥类字段已脱敏。如需调整连接或密钥，请删除本知识库后在列表中新建。
              </p>
              <ConfigFoldSection title="基本信息" defaultOpen>
                <ReadOnlyRow label="名称" value={detailForm.name} />
                <ReadOnlyRow label="描述" value={detailForm.description || '—'} subdued={!detailForm.description} />
                <ReadOnlyRow label="Provider" value={detailForm.provider} mono />
              </ConfigFoldSection>
              <ConfigFoldSection title="连接与索引">
                <ReadOnlyRow label="Endpoint URL" value={detailForm.endpointUrl} mono />
                <ReadOnlyRow label="Index Name" value={detailForm.indexName} mono />
                <ReadOnlyRow label="Namespace" value={detailForm.namespace} mono />
              </ConfigFoldSection>
              <ConfigFoldSection title="密钥与嵌入（脱敏）">
                <ReadOnlyRow label="API Key" value={maskSecret(detailForm.apiKey)} mono subdued />
                <ReadOnlyRow label="API Secret" value={maskSecret(detailForm.apiSecret)} mono subdued />
                <ReadOnlyRow label="Embedding URL" value={detailForm.embeddingUrl} mono />
                <ReadOnlyRow label="Embedding Key" value={maskSecret(detailForm.embeddingKey)} mono subdued />
                <ReadOnlyRow label="Embedding Model" value={detailForm.embeddingModel} mono />
              </ConfigFoldSection>
              <div className="pt-2">
                <Button variant="destructive" onClick={() => setKbDeletePending({ id: detailId, name: detailForm.name })}>
                  删除知识库
                </Button>
              </div>
            </Card>

            <Card className="p-4 space-y-3">
              <div className="text-sm font-medium">文档上传</div>
              <div className="text-xs text-muted-foreground flex flex-wrap gap-1.5 items-center">
                <span className="shrink-0 font-medium text-foreground/80">支持类型</span>
                {supported?.formats?.length
                  ? supported.formats.map(f => (
                    <span key={f.extension} className="rounded-md bg-muted px-2 py-0.5 text-[11px] text-foreground/90">
                      <span className="font-mono">{f.extension}</span>
                      <span className="text-muted-foreground"> · {f.description}</span>
                    </span>
                  ))
                  : acceptExtensions.split(',').map(ext => (
                    <span key={ext} className="rounded-md bg-muted px-2 py-0.5 text-[11px] font-mono">{ext}</span>
                  ))}
              </div>
              {supported?.notes?.length ? (
                <ul className="text-xs text-amber-700 dark:text-amber-400/90 list-disc pl-4 space-y-0.5">
                  {supported.notes.map((n, i) => (
                    <li key={i}>{n}</li>
                  ))}
                </ul>
              ) : null}
              <FileUpload
                label="选择或拖拽文件"
                accept={acceptExtensions}
                multiple={false}
                maxFiles={1}
                maxSize={50}
                disabled={uploading}
                onFileSelect={onUploadFiles}
              />
              {(uploading || uploadPercent > 0) && (
                <div className="space-y-1">
                  <div className="flex justify-between text-xs text-muted-foreground">
                    <span>{uploadMessage || (uploading ? '处理中…' : '')}</span>
                    <span>{uploadPercent}%</span>
                  </div>
                  <Progress value={uploadPercent} max={100} />
                </div>
              )}
            </Card>

            <Card className="p-0 overflow-hidden">
              <div className="flex flex-wrap items-center justify-between gap-2 px-4 py-3 border-b border-border bg-muted/30">
                <div className="text-sm font-medium">已上传文档</div>
                <div className="flex items-center gap-3 text-xs text-muted-foreground">
                  {docsMeta.totalDocs > 0 && (
                    <span>{docsMeta.totalDocs} 个文档 · {docsMeta.totalChunks} 条切块</span>
                  )}
                  <Button size="sm" variant="outline" disabled={docsLoading} onClick={() => void loadDocuments()}>
                    {docsLoading ? '刷新中…' : '刷新列表'}
                  </Button>
                </div>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border text-left bg-muted/20">
                      <th className="px-4 py-2 font-medium">文件名</th>
                      <th className="px-4 py-2 font-medium">Doc ID</th>
                      <th className="px-4 py-2 font-medium w-[100px]">切块数</th>
                      <th className="px-4 py-2 font-medium w-[120px] text-right">操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    {docsLoading && docList.length === 0 && (
                      <tr>
                        <td colSpan={4} className="px-4 py-8 text-center text-muted-foreground">加载中…</td>
                      </tr>
                    )}
                    {!docsLoading && docList.length === 0 && (
                      <tr>
                        <td colSpan={4} className="px-4 py-8 text-center text-muted-foreground">
                          暂无文档，上传后将在向量库中按 Doc ID 聚合展示
                        </td>
                      </tr>
                    )}
                    {docList.map(row => (
                      <tr key={row.docId} className="border-b border-border/60 hover:bg-muted/15">
                        <td className="px-4 py-2.5 max-w-[240px] truncate" title={row.filename || '—'}>{row.filename || '—'}</td>
                        <td className="px-4 py-2.5 font-mono text-xs text-muted-foreground">{row.docId}</td>
                        <td className="px-4 py-2.5 tabular-nums">{row.chunks}</td>
                        <td className="px-4 py-2.5 text-right">
                          <Button
                            size="sm"
                            variant="destructive"
                            onClick={() => setDocDeletePending({ docId: row.docId, label: row.filename || row.docId })}
                          >
                            删除
                          </Button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </Card>

            <Card className="p-4 space-y-3">
              <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
                <div className="md:col-span-3">
                  <Input label="查询问题" value={query} onChange={e => setQuery(e.target.value)} />
                </div>
                <Input label="TopK" type="number" value={String(topK)} onChange={e => setTopK(Number(e.target.value || 5))} />
              </div>
              <Button onClick={onRecallTest}>开始召回</Button>
              <div className="space-y-2">
                {recallResult.map((it, idx) => (
                  <div key={idx} className="rounded border border-gray-200 dark:border-gray-700 p-3">
                    <div className="text-xs text-gray-500">score: {Number(it?.Score || it?.score || 0).toFixed(4)}</div>
                    <div className="text-sm font-medium">{it?.Record?.Title || it?.record?.title || '-'}</div>
                    <div className="text-sm whitespace-pre-wrap">{it?.Record?.Content || it?.record?.content || '-'}</div>
                  </div>
                ))}
              </div>
            </Card>
          </>
        )}
      </div>
      ) : (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-6">
      <CollapsibleSectionHeader
        title="知识库管理"
        icon={<ChevronDown className="w-4 h-4 text-primary" />}
        expanded
        onToggle={() => {}}
        showChevron={false}
        clickable={false}
        compact
        titleSize="lg"
        withDivider
        className="mb-6"
        rightContent={(
          <Button variant="primary" size="sm" onClick={openCreateModal}>新建知识库</Button>
        )}
      />

      <Card className="p-0 overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/40 text-left">
                <th className="px-4 py-3 font-medium">名称</th>
                <th className="px-4 py-3 font-medium">Provider</th>
                <th className="px-4 py-3 font-medium">索引 / 集合</th>
                <th className="px-4 py-3 font-medium w-[200px] text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              {loading && (
                <tr>
                  <td colSpan={4} className="px-4 py-8 text-center text-muted-foreground">加载中…</td>
                </tr>
              )}
              {!loading && list.length === 0 && (
                <tr>
                  <td colSpan={4} className="px-4 py-8 text-center text-muted-foreground">
                    暂无知识库，点击右上角「新建知识库」创建
                  </td>
                </tr>
              )}
              {!loading && list.map(item => (
                <tr key={item.id} className="border-b border-border/60 hover:bg-muted/20">
                  <td className="px-4 py-3 font-medium">{item.name}</td>
                  <td className="px-4 py-3 text-muted-foreground">{item.provider}</td>
                  <td className="px-4 py-3 text-muted-foreground">{item.indexName || '—'}</td>
                  <td className="px-4 py-3 text-right space-x-2">
                    <Button size="sm" variant="outline" onClick={() => { setDetailId(item.id); setView('detail') }}>
                      详情
                    </Button>
                    <Button size="sm" variant="destructive" onClick={() => setKbDeletePending({ id: item.id, name: item.name })}>
                      删除
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>

      <p className="text-xs text-muted-foreground">
        文档支持格式可参见详情页「文档上传」；亦可通过接口 GET /knowledge-base/supported-document-types 获取。
      </p>

      <Modal isOpen={createOpen} onClose={() => setCreateOpen(false)} title="新建知识库" size="lg">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <Input label="名称" value={createForm.name} onChange={e => setCreateForm(v => ({ ...v, name: e.target.value }))} />
          <Input label="Provider" value={createForm.provider} onChange={e => setCreateForm(v => ({ ...v, provider: e.target.value }))} />
          <Input label="Endpoint URL" value={createForm.endpointUrl} onChange={e => setCreateForm(v => ({ ...v, endpointUrl: e.target.value }))} />
          <Input label="Index Name" value={createForm.indexName} onChange={e => setCreateForm(v => ({ ...v, indexName: e.target.value }))} />
          <Input label="Namespace" value={createForm.namespace} onChange={e => setCreateForm(v => ({ ...v, namespace: e.target.value }))} />
          <Input label="Description" value={createForm.description} onChange={e => setCreateForm(v => ({ ...v, description: e.target.value }))} />
          <Input label="API Key" value={createForm.apiKey} onChange={e => setCreateForm(v => ({ ...v, apiKey: e.target.value }))} />
          <Input label="API Secret" value={createForm.apiSecret} onChange={e => setCreateForm(v => ({ ...v, apiSecret: e.target.value }))} />
          <Input label="Embedding URL" value={createForm.embeddingUrl} onChange={e => setCreateForm(v => ({ ...v, embeddingUrl: e.target.value }))} />
          <Input label="Embedding Key" value={createForm.embeddingKey} onChange={e => setCreateForm(v => ({ ...v, embeddingKey: e.target.value }))} />
          <Input label="Embedding Model" value={createForm.embeddingModel} onChange={e => setCreateForm(v => ({ ...v, embeddingModel: e.target.value }))} />
        </div>
        <div className="flex justify-end gap-2 mt-6">
          <Button variant="outline" onClick={() => setCreateOpen(false)}>取消</Button>
          <Button variant="primary" onClick={onCreateSubmit}>创建</Button>
        </div>
      </Modal>
    </div>
      )}

      <ConfirmDialog
        isOpen={!!kbDeletePending}
        onClose={() => setKbDeletePending(null)}
        onConfirm={confirmDeleteKB}
        title="删除知识库"
        message={kbDeletePending ? `确定删除知识库「${kbDeletePending.name}」？向量数据与配置将一并移除，且不可恢复。` : ''}
        type="danger"
        confirmText="删除"
        cancelText="取消"
      />
      <ConfirmDialog
        isOpen={!!docDeletePending}
        onClose={() => setDocDeletePending(null)}
        onConfirm={confirmDeleteDoc}
        title="删除文档"
        message={docDeletePending ? `确定删除文档「${docDeletePending.label}」及其全部向量切块？` : ''}
        type="danger"
        confirmText="删除"
        cancelText="取消"
      />
    </>
  )
}

export default KnowledgeBaseManager
