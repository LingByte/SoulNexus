import { useCallback, useEffect, useMemo, useState } from 'react'
import { Card, Form, InputNumber, Modal, Popconfirm, Select, Space, Table, Tag, Typography } from '@arco-design/web-react'
import { Button, Input } from '@/components/ui'
import { useTranslation } from '@/i18n'
import { showAlert } from '@/utils/notification'
import {
  compareKnowledgeEval,
  createEvalDataset,
  deleteEvalDataset,
  getKnowledgeEvalJob,
  listEvalDatasets,
  listKnowledgeChunks,
  runKnowledgeEval,
  type EvalJobStatus,
  type KnowledgeChunk,
  type KnowledgeEvalDataset,
} from '@/api/knowledgeOps'

const FormItem = Form.Item

type Props = { nsId: string }

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function parseEvalResult(job: EvalJobStatus): Record<string, unknown> | null {
  if (!job.result) return null
  if (typeof job.result === 'string') {
    try {
      return JSON.parse(job.result) as Record<string, unknown>
    } catch {
      return null
    }
  }
  return job.result as Record<string, unknown>
}

export default function KnowledgeEvalTab({ nsId }: Props) {
  const { t } = useTranslation()
  const [chunks, setChunks] = useState<KnowledgeChunk[]>([])
  const [evalDatasets, setEvalDatasets] = useState<KnowledgeEvalDataset[]>([])
  const [evalResult, setEvalResult] = useState<Record<string, unknown> | null>(null)
  const [selectedDatasetId, setSelectedDatasetId] = useState('')
  const [evalStrategy, setEvalStrategy] = useState('hybrid')
  const [evalTopK, setEvalTopK] = useState(5)
  const [datasetModal, setDatasetModal] = useState(false)
  const [evalSampleRows, setEvalSampleRows] = useState([{ query: '', relevantIds: [] as string[] }])
  const [evalRunning, setEvalRunning] = useState(false)
  const [datasetForm] = Form.useForm()

  const chunkSelectOptions = useMemo(
    () =>
      chunks
        .filter((c) => c.recordId)
        .map((c) => {
          const title = c.title?.trim() || `#${c.chunkIndex}`
          return { value: c.recordId, label: `${title} · ${c.recordId}` }
        }),
    [chunks],
  )

  const reload = useCallback(async () => {
    if (!nsId) return
    const [c, datasets] = await Promise.all([listKnowledgeChunks(nsId), listEvalDatasets(nsId)])
    if (c.code === 200 && c.data) setChunks(c.data)
    if (datasets.code === 200 && datasets.data) setEvalDatasets(datasets.data)
  }, [nsId])

  useEffect(() => { void reload() }, [reload])

  const formatMetricMap = (m?: Record<string, number>) =>
    Object.entries(m ?? {})
      .sort(([a], [b]) => Number(a) - Number(b))
      .map(([k, v]) => `Recall@${k}: ${(v * 100).toFixed(1)}%`)
      .join(' · ')

  const pollEvalJob = async (jobId: string) => {
    for (let i = 0; i < 180; i++) {
      await sleep(2000)
      const res = await getKnowledgeEvalJob(nsId, jobId)
      if (res.code !== 200 || !res.data) continue
      const job = res.data
      if (job.status === 'done') {
        const parsed = parseEvalResult(job)
        if (parsed) setEvalResult(parsed)
        else showAlert(t('knowledgeOps.evalRunFailed'), 'error')
        return
      }
      if (job.status === 'failed') {
        showAlert(job.error || t('knowledgeOps.evalRunFailed'), 'error')
        return
      }
    }
    showAlert(t('request.timeout'), 'error')
  }

  const handleCreateDataset = async () => {
    const values = await datasetForm.validate()
    const items = evalSampleRows
      .map((r) => ({ query: r.query.trim(), relevantIds: r.relevantIds.filter(Boolean) }))
      .filter((r) => r.query)
    if (items.length === 0) {
      showAlert(t('knowledgeOps.evalSampleRequired'), 'warning')
      return
    }
    if (items.some((r) => r.relevantIds.length === 0)) {
      showAlert(t('knowledgeOps.evalRecordRequired'), 'warning')
      return
    }
    const res = await createEvalDataset(nsId, { name: values.name, items })
    if (res.code === 200) {
      showAlert(t('common.success'), 'success')
      setDatasetModal(false)
      datasetForm.resetFields()
      setEvalSampleRows([{ query: '', relevantIds: [] }])
      void reload()
    }
  }

  const handleRunEval = async (datasetId?: string) => {
    const id = (datasetId ?? selectedDatasetId).trim()
    if (!id) {
      showAlert(t('knowledgeOps.evalNoDatasetSelected'), 'warning')
      return
    }
    setSelectedDatasetId(id)
    setEvalRunning(true)
    setEvalResult(null)
    try {
      const res = await runKnowledgeEval(nsId, { datasetId: id, strategy: evalStrategy, topK: evalTopK })
      if (res.code === 200 && res.data?.jobId) {
        await pollEvalJob(res.data.jobId)
      } else {
        showAlert(res.msg || t('knowledgeOps.evalRunFailed'), 'error')
      }
    } catch (e: unknown) {
      const err = e as { msg?: string }
      showAlert(err?.msg || t('request.failed'), 'error')
    } finally {
      setEvalRunning(false)
    }
  }

  const handleCompareEval = async (datasetId?: string) => {
    const id = (datasetId ?? selectedDatasetId).trim()
    if (!id) {
      showAlert(t('knowledgeOps.evalNoDatasetSelected'), 'warning')
      return
    }
    setSelectedDatasetId(id)
    setEvalRunning(true)
    setEvalResult(null)
    try {
      const res = await compareKnowledgeEval(nsId, { datasetId: id, topK: evalTopK })
      if (res.code === 200 && res.data?.jobId) {
        await pollEvalJob(res.data.jobId)
      } else {
        showAlert(res.msg || t('knowledgeOps.evalRunFailed'), 'error')
      }
    } catch (e: unknown) {
      const err = e as { msg?: string }
      showAlert(err?.msg || t('request.failed'), 'error')
    } finally {
      setEvalRunning(false)
    }
  }

  return (
    <Card title={t('knowledgeOps.evalTitle')} bordered={false} style={{ borderRadius: 12 }}>
      <Typography.Paragraph type="secondary" style={{ marginTop: 0, marginBottom: 12 }}>
        {t('knowledgeOps.evalDesc')}
      </Typography.Paragraph>
      <Space style={{ marginBottom: 12 }} wrap>
        <Button onClick={() => setDatasetModal(true)}>{t('knowledgeOps.addDataset')}</Button>
        <Select
          style={{ width: 280 }}
          placeholder={t('knowledgeOps.evalDatasetSelect')}
          value={selectedDatasetId || undefined}
          onChange={(v) => setSelectedDatasetId(v ?? '')}
          allowClear
          showSearch
        >
          {evalDatasets.map((d) => (
            <Select.Option key={d.id} value={d.id}>
              {d.name} (N={d.sampleCount})
            </Select.Option>
          ))}
        </Select>
        <Input style={{ width: 120 }} placeholder={t('knowledgeOps.evalStrategy')} value={evalStrategy} onChange={setEvalStrategy} />
        <InputNumber min={1} max={20} value={evalTopK} onChange={(v) => setEvalTopK(Number(v) || 5)} />
        <Button loading={evalRunning} onClick={() => void handleRunEval()}>{t('knowledgeOps.runEval')}</Button>
        <Button loading={evalRunning} onClick={() => void handleCompareEval()}>{t('knowledgeOps.compareStrategies')}</Button>
        {evalRunning ? <Typography.Text type="secondary">{t('knowledgeOps.evalRunning')}</Typography.Text> : null}
      </Space>
      <Table
        data={evalDatasets}
        rowKey="id"
        pagination={false}
        style={{ marginBottom: 12 }}
        noDataElement={<Typography.Text type="secondary">{t('knowledgeOps.noDatasets')}</Typography.Text>}
        columns={[
          { title: t('knowledgeOps.datasetName'), dataIndex: 'name' },
          { title: 'ID', dataIndex: 'id', width: 180, ellipsis: true },
          { title: 'N', dataIndex: 'sampleCount', width: 80 },
          {
            title: t('knowledgeBase.colActions'),
            width: 120,
            render: (_: unknown, row: KnowledgeEvalDataset) => (
              <Space>
                <Button size="small" loading={evalRunning} onClick={() => void handleRunEval(row.id)}>{t('knowledgeOps.runEval')}</Button>
                <Popconfirm title={t('common.confirmDelete')} onOk={async () => { await deleteEvalDataset(nsId, row.id); void reload() }}>
                  <Button size="small" status="danger">{t('common.delete')}</Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />
      {evalResult ? (
        <Typography.Paragraph>
          <Typography.Text bold>{t('knowledgeOps.evalResult')}</Typography.Text>
          {'strategies' in evalResult && Array.isArray(evalResult.strategies) ? (
            <Space direction="vertical" style={{ width: '100%', marginTop: 8 }}>
              {(evalResult.strategies as Record<string, unknown>[]).map((r) => (
                <div key={String(r.strategy ?? r.namespace)}>
                  <Tag color="arcoblue">{String(r.strategy || '?')}</Tag>
                  <Typography.Text>{formatMetricMap(r.recall_at_k as Record<string, number> | undefined)}</Typography.Text>
                  {typeof r.mrr === 'number' ? <Tag>MRR {(r.mrr as number).toFixed(3)}</Tag> : null}
                </div>
              ))}
            </Space>
          ) : (
            <Space wrap style={{ marginTop: 8, marginBottom: 8 }}>
              {evalResult.strategy ? <Tag color="arcoblue">{String(evalResult.strategy)}</Tag> : null}
              {formatMetricMap(evalResult.recall_at_k as Record<string, number> | undefined) ? (
                <Typography.Text>{formatMetricMap(evalResult.recall_at_k as Record<string, number>)}</Typography.Text>
              ) : null}
              {typeof evalResult.mrr === 'number' ? <Tag>MRR {(evalResult.mrr as number).toFixed(3)}</Tag> : null}
              {typeof evalResult.map === 'number' ? <Tag>MAP {(evalResult.map as number).toFixed(3)}</Tag> : null}
              {typeof evalResult.queries === 'number' ? <Tag>N={String(evalResult.queries)}</Tag> : null}
            </Space>
          )}
          <pre style={{ marginTop: 8, maxHeight: 240, overflow: 'auto', fontSize: 12 }}>{JSON.stringify(evalResult, null, 2)}</pre>
        </Typography.Paragraph>
      ) : null}

      <Modal visible={datasetModal} title={t('knowledgeOps.addDataset')} onOk={() => void handleCreateDataset()} onCancel={() => setDatasetModal(false)} unmountOnExit style={{ width: 720 }}>
        <Form form={datasetForm} layout="vertical">
          <FormItem label={t('knowledgeOps.datasetName')} field="name" rules={[{ required: true }]}><Input /></FormItem>
          <Typography.Text type="secondary" style={{ display: 'block', marginBottom: 8 }}>{t('knowledgeOps.evalSampleHint')}</Typography.Text>
          {chunkSelectOptions.length === 0 ? (
            <Typography.Text type="warning" style={{ display: 'block', marginBottom: 8 }}>{t('knowledgeOps.evalNoChunks')}</Typography.Text>
          ) : null}
          {evalSampleRows.map((row, idx) => (
            <Space key={idx} align="start" style={{ display: 'flex', marginBottom: 8 }}>
              <Input
                style={{ width: 240 }}
                placeholder={t('knowledgeOps.evalQueryPlaceholder')}
                value={row.query}
                onChange={(v) => {
                  const next = [...evalSampleRows]
                  next[idx] = { ...next[idx], query: v }
                  setEvalSampleRows(next)
                }}
              />
              <Select
                mode="multiple"
                style={{ width: 360 }}
                placeholder={t('knowledgeOps.evalChunkSelect')}
                value={row.relevantIds}
                options={chunkSelectOptions}
                onChange={(v) => {
                  const next = [...evalSampleRows]
                  next[idx] = { ...next[idx], relevantIds: v }
                  setEvalSampleRows(next)
                }}
                allowClear
                showSearch
                maxTagCount={2}
              />
              {evalSampleRows.length > 1 ? (
                <Button size="small" status="danger" onClick={() => setEvalSampleRows(evalSampleRows.filter((_, i) => i !== idx))}>
                  {t('common.delete')}
                </Button>
              ) : null}
            </Space>
          ))}
          <Button size="small" onClick={() => setEvalSampleRows([...evalSampleRows, { query: '', relevantIds: [] }])}>
            {t('knowledgeOps.addEvalSample')}
          </Button>
        </Form>
      </Modal>
    </Card>
  )
}
