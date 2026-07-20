import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  Form,
  InputNumber,
  Modal,
  Popconfirm,
  Space,
  Table,
  Tabs,
  Tag,
  Typography,
  Upload,
} from '@arco-design/web-react'
import { Button, Input, Empty, Tooltip, Card, TableEmpty } from '@/components/ui'
import { Loading } from '@/components/ui/loading'
import { IconArrowLeft, IconDelete, IconEdit, IconEye, IconPlus, IconSearch, IconUpload } from '@arco-design/web-react/icon'
import BaseLayout from '@/components/Layout/BaseLayout'
import { useTranslation } from '@/i18n'
import {
  deleteKnowledgeDocument,
  confirmKnowledgeDocumentIndex,
  getKnowledgeDocumentPreview,
  getKnowledgeNamespace,
  listKnowledgeDocuments,
  recallKnowledgeDocuments,
  uploadKnowledgeDocument,
  type DocumentPreviewResult,
  type KnowledgeDocument,
  type KnowledgeNamespace,
  type RecallHit,
  type RecallPipeline,
} from '@/api/knowledgeNamespaces'
import { getWorkerStats, type KnowledgeWorkerJob, type KnowledgeWorkerSnapshot } from '@/api/knowledgeOps'
import { chunkStrategyLabel, recallStrategyLabel } from '@/pages/knowledge/chunkStrategy'
import KnowledgeWorkerQueueBar, { formatDocQueueHint } from '@/pages/knowledge/WorkerQueueBar'
import KnowledgeOpsTabs from '@/pages/KnowledgeOpsTabs'
import KnowledgeSyncTab from '@/pages/KnowledgeSyncTab'
import KnowledgeEvalTab from '@/pages/KnowledgeEvalTab'
import { showAlert } from '@/utils/notification'

const FormItem = Form.Item
const TextArea = Input.TextArea

function isDocInProgress(status: string) {
  return ['queued', 'parsing', 'preview', 'indexing', 'processing'].includes(status)
}

function formatRecallScore(value?: number) {
  if (value == null || Number.isNaN(value)) return ''
  return value.toFixed(4)
}

export default function KnowledgeBaseDetail() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { nsId = '' } = useParams()

  const [namespace, setNamespace] = useState<KnowledgeNamespace | null>(null)
  const [documents, setDocuments] = useState<KnowledgeDocument[]>([])
  const [loading, setLoading] = useState(true)
  const [uploadVisible, setUploadVisible] = useState(false)
  const [previewVisible, setPreviewVisible] = useState(false)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [previewData, setPreviewData] = useState<DocumentPreviewResult | null>(null)
  const [confirmLoading, setConfirmLoading] = useState(false)
  const [recallLoading, setRecallLoading] = useState(false)
  const [recallResults, setRecallResults] = useState<RecallHit[]>([])
  const [recallStrategy, setRecallStrategy] = useState('')
  const [recallElapsed, setRecallElapsed] = useState('')
  const [recallHitCount, setRecallHitCount] = useState(0)
  const [recallPipeline, setRecallPipeline] = useState<RecallPipeline | null>(null)
  const [workerSnapshot, setWorkerSnapshot] = useState<KnowledgeWorkerSnapshot | null>(null)
  const [uploadForm] = Form.useForm()
  const [recallForm] = Form.useForm()

  const loadAll = useCallback(async () => {
    if (!nsId) return
    setLoading(true)
    try {
      const [nsRes, docRes, workerRes] = await Promise.all([
        getKnowledgeNamespace(nsId),
        listKnowledgeDocuments(nsId),
        getWorkerStats(nsId),
      ])
      if (nsRes.code === 200 && nsRes.data) setNamespace(nsRes.data)
      if (docRes.code === 200 && docRes.data) setDocuments(docRes.data)
      if (workerRes.code === 200 && workerRes.data) setWorkerSnapshot(workerRes.data)
    } catch {
      showAlert(t('common.failed'), 'error')
    }
    setLoading(false)
  }, [nsId, t])

  useEffect(() => { loadAll() }, [loadAll])

  useEffect(() => {
    const needsPoll = documents.some((d) => isDocInProgress(d.status)) || (workerSnapshot?.unfinished ?? 0) > 0
    if (!needsPoll) return
    const timer = window.setInterval(() => { loadAll() }, 3000)
    return () => window.clearInterval(timer)
  }, [documents, workerSnapshot?.unfinished, loadAll])

  const progressByDocId = useMemo(() => {
    const map = new Map<string, KnowledgeWorkerJob>()
    for (const job of workerSnapshot?.jobs ?? []) {
      if (job.docId) map.set(String(job.docId), job)
    }
    return map
  }, [workerSnapshot])

  const docStatusTag = useCallback((status: string, indexError?: string, queueJob?: KnowledgeWorkerJob) => {
    const map: Record<string, { color: string; label: string }> = {
      queued: { color: 'gray', label: t('knowledgeBase.doc.statusQueued') },
      parsing: { color: 'orangered', label: t('knowledgeBase.doc.statusParsing') },
      preview: { color: 'gold', label: t('knowledgeBase.doc.statusPreview') },
      indexing: { color: 'blue', label: t('knowledgeBase.doc.statusIndexing') },
      processing: { color: 'orangered', label: t('knowledgeBase.doc.statusProcessing') },
      failed: { color: 'red', label: t('knowledgeBase.doc.statusFailed') },
      active: { color: 'green', label: t('knowledgeBase.doc.statusActive') },
    }
    const item = map[status] || map.processing
    const queueHint = isDocInProgress(status) ? formatDocQueueHint(t, queueJob) : null
    if (status === 'failed') {
      return (
        <Space direction="vertical" size={2} style={{ maxWidth: 160 }}>
          <Tag color={item.color}>{item.label}</Tag>
          {indexError ? (
            <Tooltip content={indexError}>
              <Typography.Text type="error" style={{ fontSize: 12, cursor: 'help' }} ellipsis>
                {indexError}
              </Typography.Text>
            </Tooltip>
          ) : null}
        </Space>
      )
    }
    return (
      <Space direction="vertical" size={2} style={{ maxWidth: 160 }}>
        <Tag color={item.color}>{item.label}</Tag>
        {queueHint ? (
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {queueHint}
          </Typography.Text>
        ) : null}
      </Space>
    )
  }, [t])

  const handleUpload = async () => {
    const values = await uploadForm.validate()
    const fileList = values.file as { originFile?: File }[] | undefined
    const file = fileList?.[0]?.originFile
    const content = String(values.content || '').trim()
    if (!file && !content) {
      showAlert(t('knowledgeBase.doc.uploadRequired'), 'warning')
      return
    }
    try {
      if (file) {
        await uploadKnowledgeDocument(nsId, {
          title: values.title || file.name,
          file,
          docType: values.docType,
          tags: values.tags ? String(values.tags).split(',').map((s: string) => s.trim()).filter(Boolean) : undefined,
          campaignId: values.campaignId,
          productLine: values.productLine,
          indexMode: values.indexMode || 'parent_child',
        })
      } else {
        await uploadKnowledgeDocument(nsId, {
          title: values.title,
          content: values.content,
          docType: values.docType,
          tags: values.tags ? String(values.tags).split(',').map((s: string) => s.trim()).filter(Boolean) : undefined,
          campaignId: values.campaignId,
          productLine: values.productLine,
          indexMode: values.indexMode || 'parent_child',
        })
      }
      showAlert(t('knowledgeBase.doc.uploadQueued'), 'success')
      setUploadVisible(false)
      uploadForm.resetFields()
      loadAll()
    } catch (err: any) {
      showAlert(err?.msg || err?.message || t('common.failed'), 'error')
    }
  }

  const handleDelete = async (docId: string) => {
    try {
      await deleteKnowledgeDocument(nsId, docId)
      showAlert(t('common.success'), 'success')
      loadAll()
    } catch (err: any) {
      showAlert(err?.msg || t('common.failed'), 'error')
    }
  }

  const openPreview = async (docId: string) => {
    setPreviewVisible(true)
    setPreviewLoading(true)
    setPreviewData(null)
    try {
      const res = await getKnowledgeDocumentPreview(nsId, docId)
      if (res.code === 200 && res.data) setPreviewData(res.data)
    } catch (err: any) {
      showAlert(err?.msg || t('common.failed'), 'error')
      setPreviewVisible(false)
    }
    setPreviewLoading(false)
  }

  const handleConfirmIndex = async (docId: string) => {
    setConfirmLoading(true)
    try {
      await confirmKnowledgeDocumentIndex(nsId, docId)
      showAlert('已提交索引', 'success')
      setPreviewVisible(false)
      loadAll()
    } catch (err: any) {
      showAlert(err?.msg || t('common.failed'), 'error')
    }
    setConfirmLoading(false)
  }

  const handleRecall = async () => {
    const values = await recallForm.validate()
    setRecallLoading(true)
    try {
      const tags = values.tags
        ? String(values.tags).split(',').map((s: string) => s.trim()).filter(Boolean)
        : undefined
      const res = await recallKnowledgeDocuments(nsId, {
        query: values.query,
        topK: values.topK || 5,
        minScore: values.minScore || 0,
        docTypes: values.docType ? [values.docType] : undefined,
        tags,
        campaignId: values.campaignId || undefined,
        productLine: values.productLine || undefined,
      })
      if (res.code === 200 && res.data) {
        setRecallResults(res.data.results || [])
        setRecallStrategy(res.data.strategy || '')
        setRecallElapsed(res.data.elapsed || '')
        setRecallHitCount(res.data.hitCount ?? 0)
        setRecallPipeline(res.data.pipeline ?? null)
      }
    } catch (err: any) {
      showAlert(err?.msg || t('common.failed'), 'error')
    }
    setRecallLoading(false)
  }

  const docColumns = useMemo(() => [
    { title: t('knowledgeBase.doc.colTitle'), dataIndex: 'title', ellipsis: true },
    {
      title: t('knowledgeBase.doc.colStatus'),
      dataIndex: 'status',
      width: 200,
      render: (status: string, record: KnowledgeDocument) =>
        docStatusTag(status, record.indexError, progressByDocId.get(record.id)),
    },
    {
      title: t('knowledgeBase.doc.colStrategy'),
      dataIndex: 'chunkStrategy',
      width: 120,
      render: (strategy: string | undefined, record: KnowledgeDocument) => (
        isDocInProgress(record.status)
          ? <Tag>-</Tag>
          : <Tag color="blue">{chunkStrategyLabel(t, strategy)}</Tag>
      ),
    },
    {
      title: t('knowledgeBase.doc.colChunks'),
      dataIndex: 'chunkCount',
      width: 90,
      render: (n: number, record: KnowledgeDocument) => (
        isDocInProgress(record.status) && record.status !== 'preview' ? <Tag>-</Tag> : <Tag color="arcoblue">{n ?? 0}</Tag>
      ),
    },
    {
      title: t('knowledgeBase.doc.colUpdated'),
      dataIndex: 'updatedAt',
      width: 180,
      render: (v: string) => (v ? new Date(v).toLocaleString() : '-'),
    },
    {
      title: t('knowledgeBase.colActions'),
      width: 240,
      render: (_: unknown, record: KnowledgeDocument) => (
        <Space>
          {record.status === 'preview' ? (
            <Button size="small" type="primary" onClick={() => openPreview(record.id)}>
              预览确认
            </Button>
          ) : null}
          <Button
            icon={<IconEye />}
            disabled={record.status !== 'active'}
            title={t('knowledgeBase.doc.viewChunks')}
            onClick={() => navigate(`/knowledge-base/${nsId}/documents/${record.id}/chunks`)}
          />
          <Button
            icon={<IconEdit />}
            disabled={isDocInProgress(record.status)}
            onClick={() => navigate(`/knowledge-base/${nsId}/documents/${record.id}/edit`)}
          />
          <Popconfirm title={t('common.confirm')} onOk={() => handleDelete(record.id)}>
            <Button status="danger" icon={<IconDelete />} />
          </Popconfirm>
        </Space>
      ),
    },
  ], [t, docStatusTag, navigate, nsId, progressByDocId])

  if (!nsId) {
    return (
      <BaseLayout title={t('knowledgeBase.title')}>
        <Empty preset="no-permission" description={t('knowledgeBase.invalidNamespace')} />
      </BaseLayout>
    )
  }

  return (
    <BaseLayout
      title={namespace?.name || t('knowledgeBase.title')}
      description={namespace?.namespace}
    >
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <Button icon={<IconArrowLeft />} onClick={() => navigate('/knowledge-base')}>
          {t('knowledgeBase.backToList')}
        </Button>

        {loading && !namespace ? (
          <Loading block />
        ) : (
          <Tabs defaultActiveTab="documents">
            <Tabs.TabPane key="documents" title={t('knowledgeBase.doc.tabDocuments')}>
              <Card bordered={false} style={{ borderRadius: 12 }}>
                <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 12, flexWrap: 'wrap' }}>
                  <Space direction="vertical" size={8}>
                    <Typography.Text type="secondary">
                      {t('knowledgeBase.doc.total')}: {documents.length}
                    </Typography.Text>
                    <KnowledgeWorkerQueueBar snapshot={workerSnapshot} compact />
                  </Space>
                  <Button icon={<IconPlus />} onClick={() => { uploadForm.resetFields(); setUploadVisible(true) }}>
                    {t('knowledgeBase.doc.upload')}
                  </Button>
                </div>
                <Table loading={loading} columns={docColumns} data={documents} rowKey="id" pagination={false} noDataElement={<TableEmpty description={t('common.noData')} />} />
              </Card>
            </Tabs.TabPane>

            <Tabs.TabPane key="recall" title={t('knowledgeBase.recall.tab')}>
              <Card bordered={false} style={{ borderRadius: 12 }}>
                {recallStrategy ? (
                  <Tag color="arcoblue" style={{ marginBottom: 12 }}>
                    {recallStrategyLabel(t, recallStrategy)}
                  </Tag>
                ) : null}
                <Typography.Text type="secondary" style={{ display: 'block', marginBottom: 12 }}>
                  {t('knowledgeBase.recall.evalHint')}
                </Typography.Text>
                <Form form={recallForm} layout="vertical" initialValues={{ topK: 5, minScore: 0 }}>
                  <FormItem label={t('knowledgeBase.recall.query')} field="query" rules={[{ required: true }]}>
                    <Input.TextArea rows={3} placeholder={t('knowledgeBase.recall.queryPlaceholder')} />
                  </FormItem>
                  <Space wrap>
                    <FormItem label="文档类型" field="docType">
                      <Input placeholder="manual / faq" style={{ width: 160 }} />
                    </FormItem>
                    <FormItem label="标签" field="tags">
                      <Input placeholder="tag1,tag2" style={{ width: 160 }} />
                    </FormItem>
                    <FormItem label="业务标签" field="campaignId">
                      <Input style={{ width: 160 }} />
                    </FormItem>
                    <FormItem label="产品线" field="productLine">
                      <Input style={{ width: 160 }} />
                    </FormItem>
                  </Space>
                  <Space>
                    <FormItem label={t('knowledgeBase.recall.topK')} field="topK">
                      <InputNumber min={1} max={20} style={{ width: 120 }} />
                    </FormItem>
                    <FormItem label={t('knowledgeBase.recall.minScore')} field="minScore">
                      <InputNumber min={0} max={1} step={0.05} style={{ width: 120 }} />
                    </FormItem>
                  </Space>
                  <Button icon={<IconSearch />} loading={recallLoading} onClick={handleRecall}>
                    {t('knowledgeBase.recall.run')}
                  </Button>
                </Form>

                <div style={{ marginTop: 24 }}>
                  {recallResults.length === 0 ? (
                    <Empty preset="no-data" description={t('knowledgeBase.recall.empty')} />
                  ) : (
                    <>
                      <div style={{ marginBottom: 12, display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
                        <Tag color="arcoblue">{t('knowledgeBase.recall.hitCount')}: {recallHitCount}</Tag>
                        {recallElapsed && (
                          <Tag color="green">{t('knowledgeBase.recall.elapsed')}: {recallElapsed}</Tag>
                        )}
                        {recallPipeline?.rerankEnabled ? (
                          <Tag color="purple">{t('knowledgeBase.recall.pipelineRerank')}{recallPipeline.rerankModel ? `: ${recallPipeline.rerankModel}` : ''}</Tag>
                        ) : null}
                        {recallPipeline?.compositeRerank ? (
                          <Tag color="orangered">{t('knowledgeBase.recall.pipelineComposite')}</Tag>
                        ) : null}
                        {recallPipeline?.enableMMR ? (
                          <Tag>{t('knowledgeBase.recall.pipelineMMR')}</Tag>
                        ) : null}
                        {recallPipeline?.enableDedup ? (
                          <Tag>{t('knowledgeBase.recall.pipelineDedup')}</Tag>
                        ) : null}
                        {recallPipeline?.rrfK ? (
                          <Tag color="cyan">{t('knowledgeBase.recall.pipelineRRF')}: {recallPipeline.rrfK}</Tag>
                        ) : null}
                        {recallStrategy === 'hybrid' && recallPipeline?.rrfVectorWeight != null && recallPipeline?.rrfKeywordWeight != null ? (
                          <Tag color="cyan">
                            RRF {recallPipeline.rrfVectorWeight}/{recallPipeline.rrfKeywordWeight}
                          </Tag>
                        ) : null}
                      </div>
                      <Space direction="vertical" size={12} style={{ width: '100%' }}>
                        {recallResults.map((hit) => (
                          <Card key={hit.id} size="small" style={{ borderRadius: 8 }}>
                            <Space direction="vertical" size={4} style={{ width: '100%' }}>
                              <Space wrap>
                                <Tag>{t('knowledgeBase.recall.recordId')}: {hit.id}</Tag>
                                <Tag color="green">{t('knowledgeBase.recall.scoreFinal')}: {hit.score.toFixed(4)}</Tag>
                                {hit.scores?.rrf != null && (
                                  <Tag color="cyan">{t('knowledgeBase.recall.scoreRrf')}: {formatRecallScore(hit.scores.rrf)}</Tag>
                                )}
                                {hit.scores?.vectorRrf != null && (
                                  <Tag color="blue">{t('knowledgeBase.recall.scoreVectorRrf')}: {formatRecallScore(hit.scores.vectorRrf)}</Tag>
                                )}
                                {hit.scores?.keywordRrf != null && (
                                  <Tag color="blue">{t('knowledgeBase.recall.scoreKeywordRrf')}: {formatRecallScore(hit.scores.keywordRrf)}</Tag>
                                )}
                                {hit.scores?.vectorRank != null && (
                                  <Tag>{t('knowledgeBase.recall.scoreVectorRank')}: #{hit.scores.vectorRank}</Tag>
                                )}
                                {hit.scores?.keywordRank != null && (
                                  <Tag>{t('knowledgeBase.recall.scoreKeywordRank')}: #{hit.scores.keywordRank}</Tag>
                                )}
                                {hit.scores?.base != null && (
                                  <Tag color="arcoblue">{t('knowledgeBase.recall.scoreBase')}: {formatRecallScore(hit.scores.base)}</Tag>
                                )}
                                {hit.scores?.model != null && (
                                  <Tag color="purple">{t('knowledgeBase.recall.scoreModel')}: {formatRecallScore(hit.scores.model)}</Tag>
                                )}
                                {hit.scores?.composite != null && (
                                  <Tag color="orangered">{t('knowledgeBase.recall.scoreComposite')}: {formatRecallScore(hit.scores.composite)}</Tag>
                                )}
                                <Typography.Text bold>{hit.title || hit.id}</Typography.Text>
                              </Space>
                              <Typography.Paragraph style={{ marginBottom: 0, whiteSpace: 'pre-wrap' }}>
                                {hit.content}
                              </Typography.Paragraph>
                            </Space>
                          </Card>
                        ))}
                      </Space>
                    </>
                  )}
                </div>
              </Card>
            </Tabs.TabPane>

            <Tabs.TabPane key="ops" title={t('knowledgeOps.tab')}>
              <KnowledgeOpsTabs nsId={nsId} />
            </Tabs.TabPane>

            <Tabs.TabPane key="sync" title={t('knowledgeOps.syncTab')}>
              <KnowledgeSyncTab nsId={nsId} />
            </Tabs.TabPane>

            <Tabs.TabPane key="eval" title={t('knowledgeOps.evalTab')}>
              <KnowledgeEvalTab nsId={nsId} />
            </Tabs.TabPane>
          </Tabs>
        )}
      </Space>

      <Modal
        title={t('knowledgeBase.doc.upload')}
        visible={uploadVisible}
        onOk={handleUpload}
        onCancel={() => setUploadVisible(false)}
        unmountOnExit
        style={{ width: 560 }}
      >
        <Form form={uploadForm} layout="vertical" initialValues={{ indexMode: 'parent_child' }}>
          <FormItem label={t('knowledgeBase.doc.colTitle')} field="title">
            <Input placeholder={t('knowledgeBase.doc.titlePlaceholder')} />
          </FormItem>
          <Space wrap>
            <FormItem label="文档类型" field="docType">
              <Input placeholder="manual" style={{ width: 140 }} />
            </FormItem>
            <FormItem label="标签" field="tags">
              <Input placeholder="a,b" style={{ width: 140 }} />
            </FormItem>
            <FormItem label="业务标签" field="campaignId">
              <Input style={{ width: 140 }} />
            </FormItem>
            <FormItem label="产品线" field="productLine">
              <Input style={{ width: 140 }} />
            </FormItem>
          </Space>
          <FormItem label={t('knowledgeBase.doc.file')} field="file">
            <Upload
              accept=".txt,.md,.markdown,.mdx,.csv,.html,.htm,.json,.yaml,.yml,.eml,.rtf,.pdf,.docx,.pptx,.xlsx,.png,.jpg,.jpeg"
              limit={1}
              autoUpload={false}
            >
              <Button icon={<IconUpload />}>{t('knowledgeBase.doc.chooseFile')}</Button>
            </Upload>
          </FormItem>
          <Typography.Text type="secondary" style={{ display: 'block', marginBottom: 8 }}>
            {t('knowledgeBase.doc.orPaste')}
          </Typography.Text>
          <FormItem field="content">
            <TextArea rows={8} placeholder={t('knowledgeBase.doc.contentPlaceholder')} />
          </FormItem>
        </Form>
      </Modal>

      <Modal
        title="分块预览 · 确认索引"
        visible={previewVisible}
        onCancel={() => setPreviewVisible(false)}
        footer={null}
        style={{ width: 720 }}
      >
        {previewLoading ? (
          <Loading block />
        ) : previewData?.preview ? (
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            <Space wrap>
              <Tag color="blue">{previewData.preview.strategy}</Tag>
              <Tag>Parent: {previewData.parentCount ?? previewData.preview.parentCount}</Tag>
              <Tag>Child: {previewData.childCount ?? previewData.preview.childCount}</Tag>
              <Tag>{previewData.preview.charCount} chars</Tag>
            </Space>
            {previewData.preview.summary ? (
              <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                摘要：{previewData.preview.summary}
              </Typography.Paragraph>
            ) : null}
            {previewData.preview.parse?.preview ? (
              <Typography.Paragraph style={{ whiteSpace: 'pre-wrap', maxHeight: 120, overflow: 'auto' }}>
                解析预览：{previewData.preview.parse.preview}
              </Typography.Paragraph>
            ) : null}
            <Table
              size="small"
              pagination={{ pageSize: 8 }}
              rowKey="index"
              data={previewData.preview.children}
              columns={[
                { title: '#', dataIndex: 'index', width: 60 },
                { title: 'Level', dataIndex: 'level', width: 80 },
                { title: 'Parent', dataIndex: 'parentIndex', width: 80, render: (v: number) => (v >= 0 ? v : '-') },
                { title: 'Preview', dataIndex: 'preview', ellipsis: true },
                { title: 'Chars', dataIndex: 'charCount', width: 80 },
              ]}
            />
            {previewData.status === 'preview' ? (
              <Button type="primary" loading={confirmLoading} onClick={() => handleConfirmIndex(previewData.docId)}>
                确认并开始索引
              </Button>
            ) : null}
          </Space>
        ) : (
          <Empty preset="no-data" description="暂无预览数据" />
        )}
      </Modal>
    </BaseLayout>
  )
}
