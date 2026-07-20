import { useCallback, useEffect, useState } from 'react'
import { Card, Drawer, Form, Modal, Popconfirm, Space, Table, Tag, Typography } from '@arco-design/web-react'
import { Button, Input, TableEmpty } from '@/components/ui'
import { useTranslation } from '@/i18n'
import { showAlert } from '@/utils/notification'
import {
  countUnansweredQuestions,
  createKnowledgeChunk,
  deleteKnowledgeChunk,
  deleteUnansweredQuestion,
  downloadKnowledgeChunksExport,
  getHFQuestionStats,
  getHFDailySummary,
  getQuoteRateReport,
  getWorkerStats,
  listHFQuestionAnswers,
  listHFQuestions,
  listKnowledgeChunks,
  listUnansweredQuestions,
  resolveUnansweredQuestion,
  draftUnansweredAnswer,
  updateKnowledgeChunk,
  type HFDailyStat,
  type HFAnsweredRecord,
  type KnowledgeChunk,
  type KnowledgeTypicalQuestion,
  type KnowledgeUnansweredQuestion,
  type KnowledgeWorkerSnapshot,
  type QuoteRateOverview,
} from '@/api/knowledgeOps'
import KnowledgeWorkerQueueBar, { formatDocQueueHint, workerJobKindLabel } from '@/pages/knowledge/WorkerQueueBar'

const FormItem = Form.Item
const TextArea = Input.TextArea

type Props = { nsId: string }

export default function KnowledgeOpsTabs({ nsId }: Props) {
  const { t } = useTranslation()
  const [chunks, setChunks] = useState<KnowledgeChunk[]>([])
  const [unanswered, setUnanswered] = useState<KnowledgeUnansweredQuestion[]>([])
  const [hfQuestions, setHFQuestions] = useState<KnowledgeTypicalQuestion[]>([])
  const [quoteOverview, setQuoteOverview] = useState<QuoteRateOverview | null>(null)
  const [unansweredCounts, setUnansweredCounts] = useState({ open: 0, resolved: 0 })
  const [workerSnapshot, setWorkerSnapshot] = useState<KnowledgeWorkerSnapshot | null>(null)
  const [editingChunkId, setEditingChunkId] = useState<string | null>(null)
  const [editDraft, setEditDraft] = useState({ title: '', content: '' })
  const [hfDrill, setHfDrill] = useState<KnowledgeTypicalQuestion | null>(null)
  const [hfDailyStats, setHfDailyStats] = useState<HFDailyStat[]>([])
  const [hfSummary, setHfSummary] = useState<HFDailyStat[]>([])
  const [hfSelectedDay, setHfSelectedDay] = useState('')
  const [hfAnswers, setHfAnswers] = useState<HFAnsweredRecord[]>([])
  const [chunkModal, setChunkModal] = useState(false)
  const [resolveModal, setResolveModal] = useState<KnowledgeUnansweredQuestion | null>(null)
  const [resolveDrafting, setResolveDrafting] = useState(false)
  const [chunkForm] = Form.useForm()
  const [resolveForm] = Form.useForm()

  const openResolveModal = async (row: KnowledgeUnansweredQuestion) => {
    setResolveModal(row)
    resolveForm.setFieldsValue({ title: row.question, content: '' })
    setResolveDrafting(true)
    try {
      const res = await draftUnansweredAnswer(nsId, row.id)
      if (res.code === 200 && res.data) {
        resolveForm.setFieldsValue({
          title: res.data.title || row.question,
          content: res.data.content || '',
        })
      }
    } catch {
      // keep empty content; user can still fill manually
    } finally {
      setResolveDrafting(false)
    }
  }

  const reload = useCallback(async () => {
    if (!nsId) return
    const [c, u, h, counts, stats] = await Promise.all([
      listKnowledgeChunks(nsId),
      listUnansweredQuestions(nsId),
      listHFQuestions(nsId),
      countUnansweredQuestions(nsId),
      getWorkerStats(nsId),
    ])
    if (c.code === 200 && c.data) setChunks(c.data)
    if (u.code === 200 && u.data) setUnanswered(u.data.items || [])
    if (h.code === 200 && h.data) setHFQuestions(h.data.items || [])
    if (counts.code === 200 && counts.data) setUnansweredCounts(counts.data)
    if (stats.code === 200 && stats.data) setWorkerSnapshot(stats.data)
  }, [nsId])

  useEffect(() => { reload() }, [reload])

  useEffect(() => {
    if ((workerSnapshot?.unfinished ?? 0) <= 0) return
    const timer = window.setInterval(() => { void reload() }, 3000)
    return () => window.clearInterval(timer)
  }, [workerSnapshot?.unfinished, reload])

  useEffect(() => {
    if (!nsId) return
    getHFDailySummary(nsId).then((res) => {
      if (res.code === 200 && res.data?.items) setHfSummary(res.data.items)
    })
  }, [nsId])

  const openHfDrill = async (row: KnowledgeTypicalQuestion) => {
    setHfDrill(row)
    setHfSelectedDay('')
    setHfAnswers([])
    const res = await getHFQuestionStats(nsId, row.id)
    if (res.code === 200 && res.data?.items) setHfDailyStats(res.data.items)
  }

  const loadHfAnswers = async (day: string) => {
    if (!hfDrill) return
    setHfSelectedDay(day)
    const res = await listHFQuestionAnswers(nsId, hfDrill.id, day)
    if (res.code === 200 && res.data) setHfAnswers(res.data.items || [])
  }

  const startEditChunk = (row: KnowledgeChunk) => {
    setEditingChunkId(row.id)
    setEditDraft({ title: row.title || '', content: row.content || '' })
  }

  const saveEditChunk = async () => {
    if (!editingChunkId) return
    const res = await updateKnowledgeChunk(nsId, editingChunkId, editDraft)
    if (res.code === 200) {
      showAlert(t('common.success'), 'success')
      setEditingChunkId(null)
      reload()
    }
  }

  const loadQuoteRate = async () => {
    const res = await getQuoteRateReport(nsId, {})
    if (res.code === 200 && res.data) setQuoteOverview(res.data.overview)
  }

  const handleCreateChunk = async () => {
    const values = await chunkForm.validate()
    const res = await createKnowledgeChunk(nsId, { title: values.title, content: values.content })
    if (res.code === 200) {
      showAlert(t('common.success'), 'success')
      setChunkModal(false)
      chunkForm.resetFields()
      reload()
    }
  }

  const handleResolve = async () => {
    if (!resolveModal) return
    const values = await resolveForm.validate()
    const res = await resolveUnansweredQuestion(nsId, resolveModal.id, values)
    if (res.code === 200) {
      showAlert(t('knowledgeOps.resolveQueued'), 'success')
      setResolveModal(null)
      resolveForm.resetFields()
      reload()
    }
  }

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card title={t('knowledgeOps.slicesTitle')} bordered={false} style={{ borderRadius: 12 }}>
        <Space direction="vertical" size={12} style={{ width: '100%', marginBottom: 12 }}>
          <Space wrap>
            <Button onClick={() => setChunkModal(true)}>{t('knowledgeOps.addSlice')}</Button>
            <Button onClick={() => void downloadKnowledgeChunksExport(nsId).catch(() => showAlert(t('request.failed'), 'error'))}>
              {t('knowledgeOps.exportExcel')}
            </Button>
          </Space>
          <KnowledgeWorkerQueueBar snapshot={workerSnapshot} />
          {(workerSnapshot?.jobs?.length ?? 0) > 0 ? (
            <Table
              size="small"
              pagination={false}
              rowKey="taskId"
              data={workerSnapshot?.jobs ?? []}
              columns={[
                {
                  title: t('knowledgeOps.workerColKind'),
                  dataIndex: 'kind',
                  width: 88,
                  render: (kind: 'ingest' | 'purge' | 'sync') => (
                    <Tag size="small">{workerJobKindLabel(t, kind)}</Tag>
                  ),
                },
                {
                  title: t('knowledgeOps.workerColDoc'),
                  dataIndex: 'docId',
                  width: 100,
                  render: (docId?: number) => (docId ? `#${docId}` : '—'),
                },
                {
                  title: t('knowledgeOps.workerColStatus'),
                  dataIndex: 'taskStatus',
                  width: 96,
                  render: (status: string) => <Tag size="small">{status}</Tag>,
                },
                {
                  title: t('knowledgeOps.workerColQueueAhead'),
                  dataIndex: 'queueAhead',
                  width: 120,
                  render: (_: unknown, row) => formatDocQueueHint(t, row) || '—',
                },
              ]}
            />
          ) : null}
        </Space>
        <Table
          data={chunks}
          rowKey="id"
          pagination={false}
          noDataElement={<TableEmpty description={t('common.noData')} />}
          columns={[
            { title: '#', dataIndex: 'chunkIndex', width: 60 },
            {
              title: t('knowledgeOps.evalRecordIdCol'),
              dataIndex: 'recordId',
              width: 140,
              ellipsis: true,
              render: (v: string) => (
                <Typography.Text copyable={!!v} ellipsis style={{ maxWidth: 120 }}>
                  {v || '—'}
                </Typography.Text>
              ),
            },
            {
              title: t('knowledgeBase.doc.colTitle'),
              dataIndex: 'title',
              ellipsis: true,
              render: (_: unknown, row: KnowledgeChunk) =>
                editingChunkId === row.id ? (
                  <Input value={editDraft.title} onChange={(v) => setEditDraft({ ...editDraft, title: v })} />
                ) : (
                  row.title
                ),
            },
            {
              title: t('knowledgeOps.content'),
              dataIndex: 'content',
              ellipsis: true,
              render: (_: unknown, row: KnowledgeChunk) =>
                editingChunkId === row.id ? (
                  <TextArea
                    autoSize={{ minRows: 2, maxRows: 6 }}
                    value={editDraft.content}
                    onChange={(v) => setEditDraft({ ...editDraft, content: v })}
                  />
                ) : (
                  <Typography.Text ellipsis style={{ maxWidth: 320 }}>{row.content}</Typography.Text>
                ),
            },
            { title: t('knowledgeOps.sourceType'), dataIndex: 'sourceType', width: 100 },
            {
              title: t('knowledgeBase.colActions'),
              width: 160,
              render: (_: unknown, row: KnowledgeChunk) =>
                editingChunkId === row.id ? (
                  <Space>
                    <Button size="small" type="primary" onClick={saveEditChunk}>{t('common.save')}</Button>
                    <Button size="small" onClick={() => setEditingChunkId(null)}>{t('common.cancel')}</Button>
                  </Space>
                ) : (
                  <Space>
                    <Button size="small" onClick={() => startEditChunk(row)}>{t('common.edit')}</Button>
                    <Popconfirm title={t('common.confirmDelete')} onOk={async () => { await deleteKnowledgeChunk(nsId, row.id); reload() }}>
                      <Button size="small" status="danger">{t('common.delete')}</Button>
                    </Popconfirm>
                  </Space>
                ),
            },
          ]}
        />
      </Card>

      <Card title={t('knowledgeOps.unansweredTitle')} bordered={false} style={{ borderRadius: 12 }}>
        <Space style={{ marginBottom: 12 }}>
          <Tag color="orangered">{t('knowledgeOps.openCount')}: {unansweredCounts.open}</Tag>
          <Tag color="green">{t('knowledgeOps.resolvedCount')}: {unansweredCounts.resolved}</Tag>
        </Space>
        <Table
          data={unanswered}
          rowKey="id"
          pagination={false}
          noDataElement={<TableEmpty description={t('knowledgeOps.noUnanswered')} />}
          columns={[
            { title: t('knowledgeOps.question'), dataIndex: 'question', ellipsis: true },
            { title: t('knowledgeOps.occurrence'), dataIndex: 'occurrenceCount', width: 90 },
            {
              title: t('knowledgeBase.colActions'),
              width: 180,
              render: (_: unknown, row: KnowledgeUnansweredQuestion) => (
                <Space>
                  <Button size="small" onClick={() => void openResolveModal(row)}>
                    {t('knowledgeOps.resolve')}
                  </Button>
                  <Popconfirm title={t('knowledgeOps.ignoreConfirm')} onOk={async () => { await deleteUnansweredQuestion(nsId, row.id); reload() }}>
                    <Button size="small">{t('knowledgeOps.ignore')}</Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </Card>

      <Card title={t('knowledgeOps.hfTitle')} bordered={false} style={{ borderRadius: 12 }}>
        {hfSummary.length > 0 ? (
          <Space wrap style={{ marginBottom: 12 }}>
            {hfSummary.slice(-7).map((d) => (
              <Tag key={d.statDate} color="arcoblue">
                {d.statDate}: {d.count}
                {d.quotedCount > 0 ? ` (${t('knowledgeOps.quotedCount')} ${d.quotedCount})` : ''}
              </Tag>
            ))}
          </Space>
        ) : null}
        <Table
          data={hfQuestions}
          rowKey="id"
          pagination={false}
          noDataElement={<TableEmpty description={t('common.noData')} />}
          columns={[
            { title: t('knowledgeOps.question'), dataIndex: 'question', ellipsis: true },
            { title: t('knowledgeOps.totalCount'), dataIndex: 'totalCount', width: 90 },
            { title: t('knowledgeOps.quotedCount'), dataIndex: 'quotedCount', width: 90 },
            {
              title: t('knowledgeBase.colActions'),
              width: 100,
              render: (_: unknown, row: KnowledgeTypicalQuestion) => (
                <Button size="small" onClick={() => openHfDrill(row)}>{t('knowledgeOps.drillDown')}</Button>
              ),
            },
          ]}
        />
      </Card>

      <Card title={t('knowledgeOps.analyticsTitle')} bordered={false} style={{ borderRadius: 12 }}>
        <Button onClick={loadQuoteRate} style={{ marginBottom: 12 }}>{t('knowledgeOps.loadQuoteRate')}</Button>
        {quoteOverview ? (
          <Space wrap>
            <Tag color="arcoblue">{t('knowledgeOps.quoteRate')}: {(quoteOverview.quoteRate * 100).toFixed(1)}%</Tag>
            <Tag color="green">{t('knowledgeOps.hitRate')}: {(quoteOverview.hitRate * 100).toFixed(1)}%</Tag>
            <Tag>{t('knowledgeOps.kbAttachRate')}: {(quoteOverview.kbAttachRate * 100).toFixed(1)}%</Tag>
            <Tag>{t('knowledgeOps.totalRetrievals')}: {quoteOverview.totalRetrievals}</Tag>
          </Space>
        ) : (
          <Typography.Text type="secondary">{t('knowledgeOps.analyticsHint')}</Typography.Text>
        )}
      </Card>

      <Modal visible={chunkModal} title={t('knowledgeOps.addSlice')} onOk={handleCreateChunk} onCancel={() => setChunkModal(false)} unmountOnExit>
        <Form form={chunkForm} layout="vertical">
          <FormItem label={t('knowledgeBase.doc.colTitle')} field="title"><Input /></FormItem>
          <FormItem label={t('knowledgeOps.content')} field="content" rules={[{ required: true }]}><TextArea rows={6} /></FormItem>
        </Form>
      </Modal>

      <Modal
        visible={!!resolveModal}
        title={t('knowledgeOps.resolve')}
        onOk={handleResolve}
        onCancel={() => setResolveModal(null)}
        okButtonProps={{ disabled: resolveDrafting }}
        unmountOnExit
      >
        <Form form={resolveForm} layout="vertical">
          <FormItem label={t('knowledgeBase.doc.colTitle')} field="title"><Input /></FormItem>
          <FormItem
            label={t('knowledgeOps.answerContent')}
            field="content"
            rules={[{ required: true }]}
            extra={resolveDrafting ? t('knowledgeOps.draftingHint') : t('knowledgeOps.draftReadyHint')}
          >
            <TextArea rows={8} placeholder={resolveDrafting ? t('knowledgeOps.draftingPlaceholder') : undefined} />
          </FormItem>
        </Form>
      </Modal>

      <Drawer
        width={560}
        title={hfDrill?.question || t('knowledgeOps.hfTitle')}
        visible={!!hfDrill}
        onCancel={() => setHfDrill(null)}
        footer={null}
      >
        <Typography.Text bold>{t('knowledgeOps.dailyStats')}</Typography.Text>
        <Table
          style={{ marginTop: 8, marginBottom: 16 }}
          data={hfDailyStats}
          rowKey="statDate"
          pagination={false}
          columns={[
            { title: t('knowledgeOps.statDate'), dataIndex: 'statDate', width: 120 },
            { title: t('knowledgeOps.totalCount'), dataIndex: 'count', width: 80 },
            { title: t('knowledgeOps.quotedCount'), dataIndex: 'quotedCount', width: 80 },
            {
              title: t('knowledgeBase.colActions'),
              width: 80,
              render: (_: unknown, row: HFDailyStat) => (
                <Button size="mini" onClick={() => loadHfAnswers(row.statDate)}>{t('knowledgeOps.viewDay')}</Button>
              ),
            },
          ]}
        />
        {hfSelectedDay ? (
          <>
            <Typography.Text bold>{t('knowledgeOps.dayDetail', { day: hfSelectedDay })}</Typography.Text>
            <Table
              style={{ marginTop: 8 }}
              data={hfAnswers}
              rowKey="id"
              pagination={false}
              columns={[
                { title: t('knowledgeOps.question'), dataIndex: 'question', ellipsis: true },
                {
                  title: t('knowledgeOps.quotedCount'),
                  width: 80,
                  render: (_: unknown, row: HFAnsweredRecord) => (row.knowledgeQuoted ? t('common.yes') : t('common.no')),
                },
                { title: 'Hits', dataIndex: 'retrievalHitCount', width: 60 },
                { title: 'Call', dataIndex: 'callId', width: 100, ellipsis: true },
              ]}
            />
          </>
        ) : null}
      </Drawer>
    </Space>
  )
}
