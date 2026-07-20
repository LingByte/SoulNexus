import { useCallback, useEffect, useState } from 'react'
import { Card, Form, InputNumber, Modal, Popconfirm, Select, Space, Table, Tag, Typography } from '@arco-design/web-react'
import { IconPlus, IconRefresh } from '@arco-design/web-react/icon'
import { Button, Input, TableEmpty } from '@/components/ui'
import { useTranslation } from '@/i18n'
import { showAlert } from '@/utils/notification'
import {
  createSyncSource,
  deleteSyncSource,
  listSyncSources,
  triggerSyncSource,
  updateSyncSource,
  type KnowledgeSyncSource,
} from '@/api/knowledgeOps'

const FormItem = Form.Item

type Props = { nsId: string }

type SyncFormValues = {
  name: string
  sourceUrl: string
  intervalMinutes: number
  indexColumns?: string
  titleColumn?: string
  keyColumn?: string
  tableFormat?: string
}

function formatInterval(minutes: number, t: (k: string) => string) {
  if (minutes >= 1440 && minutes % 1440 === 0) {
    const days = minutes / 1440
    return days === 1 ? t('knowledgeOps.syncIntervalDay') : t('knowledgeOps.syncIntervalDays', { n: String(days) })
  }
  if (minutes >= 60 && minutes % 60 === 0) {
    return t('knowledgeOps.syncIntervalHours', { n: String(minutes / 60) })
  }
  return t('knowledgeOps.syncIntervalMinutes', { n: String(minutes) })
}

export default function KnowledgeSyncTab({ nsId }: Props) {
  const { t } = useTranslation()
  const [syncSources, setSyncSources] = useState<KnowledgeSyncSource[]>([])
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<KnowledgeSyncSource | null>(null)
  const [syncSourceType, setSyncSourceType] = useState('url')
  const [saving, setSaving] = useState(false)
  const [syncForm] = Form.useForm<SyncFormValues>()

  const reload = useCallback(async () => {
    if (!nsId) return
    const res = await listSyncSources(nsId)
    if (res.code === 200 && res.data) setSyncSources(res.data)
  }, [nsId])

  useEffect(() => { void reload() }, [reload])

  const openCreate = () => {
    setEditing(null)
    setSyncSourceType('url')
    syncForm.resetFields()
    syncForm.setFieldsValue({ intervalMinutes: 1440, tableFormat: 'csv' })
    setModalOpen(true)
  }

  const openEdit = (row: KnowledgeSyncSource) => {
    setEditing(row)
    const type = row.sourceType || 'url'
    setSyncSourceType(type)
    syncForm.setFieldsValue({
      name: row.name,
      sourceUrl: row.sourceUrl,
      intervalMinutes: row.intervalMinutes || 1440,
    })
    setModalOpen(true)
  }

  const buildBody = (values: SyncFormValues) => {
    const body: Record<string, unknown> = {
      name: values.name,
      sourceType: syncSourceType,
      sourceUrl: values.sourceUrl,
      intervalMinutes: values.intervalMinutes || 1440,
    }
    if (syncSourceType === 'table') {
      const cols = String(values.indexColumns || '')
        .split(/[,，]/)
        .map((s) => s.trim())
        .filter(Boolean)
      body.chunkConfig = {
        format: values.tableFormat || 'csv',
        indexColumns: cols,
        titleColumn: values.titleColumn || '',
        keyColumn: values.keyColumn || '',
      }
    }
    return body as Parameters<typeof createSyncSource>[1]
  }

  const handleSave = async () => {
    const values = await syncForm.validate()
    setSaving(true)
    try {
      const body = buildBody(values)
      const res = editing
        ? await updateSyncSource(nsId, editing.id, body)
        : await createSyncSource(nsId, body)
      if (res.code === 200) {
        showAlert(t('common.success'), 'success')
        setModalOpen(false)
        void reload()
      } else {
        showAlert(res.msg || t('request.failed'), 'error')
      }
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card bordered={false} style={{ borderRadius: 12 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 16, gap: 12 }}>
        <div>
          <Typography.Title heading={6} style={{ margin: 0 }}>{t('knowledgeOps.syncTitle')}</Typography.Title>
          <Typography.Paragraph type="secondary" style={{ margin: '4px 0 0', maxWidth: 560 }}>
            {t('knowledgeOps.syncHint')}
          </Typography.Paragraph>
        </div>
        <Space>
          <Button icon={<IconRefresh />} onClick={() => void reload()}>{t('knowledgeOps.syncRefresh')}</Button>
          <Button type="primary" icon={<IconPlus />} onClick={openCreate}>{t('knowledgeOps.addSync')}</Button>
        </Space>
      </div>

      <Table
        data={syncSources}
        rowKey="id"
        pagination={false}
        border={{ cell: false }}
        noDataElement={<TableEmpty description={t('knowledgeOps.syncEmpty')} />}
        columns={[
          {
            title: t('knowledgeOps.syncName'),
            dataIndex: 'name',
            width: 120,
            render: (v: string) => <Typography.Text bold>{v}</Typography.Text>,
          },
          {
            title: t('knowledgeOps.syncType'),
            dataIndex: 'sourceType',
            width: 88,
            render: (v: string) => (
              <Tag size="small" color={v === 'table' ? 'purple' : 'arcoblue'}>{v || 'url'}</Tag>
            ),
          },
          {
            title: 'URL',
            dataIndex: 'sourceUrl',
            ellipsis: true,
            render: (v: string) => (
              <Typography.Text copyable={!!v} ellipsis style={{ maxWidth: 280 }}>{v}</Typography.Text>
            ),
          },
          {
            title: t('knowledgeOps.syncInterval'),
            dataIndex: 'intervalMinutes',
            width: 100,
            render: (v: number) => formatInterval(v || 1440, t),
          },
          {
            title: t('knowledgeOps.syncLastRun'),
            width: 160,
            render: (_: unknown, row: KnowledgeSyncSource) => (
              <Space direction="vertical" size={0}>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  {row.lastSyncAt ? new Date(row.lastSyncAt).toLocaleString() : '—'}
                </Typography.Text>
                {row.lastSyncError ? (
                  <Typography.Text type="error" ellipsis style={{ fontSize: 11, maxWidth: 140 }}>
                    {row.lastSyncError}
                  </Typography.Text>
                ) : null}
              </Space>
            ),
          },
          {
            title: t('knowledgeBase.colActions'),
            width: 200,
            fixed: 'right' as const,
            render: (_: unknown, row: KnowledgeSyncSource) => (
              <Space size={4}>
                <Button
                  size="mini"
                  type="primary"
                  onClick={async () => {
                    await triggerSyncSource(nsId, row.id)
                    showAlert(t('knowledgeOps.syncQueued'), 'success')
                  }}
                >
                  {t('knowledgeOps.syncNow')}
                </Button>
                <Button size="mini" onClick={() => openEdit(row)}>{t('common.edit')}</Button>
                <Popconfirm
                  title={t('common.confirmDelete')}
                  onOk={async () => {
                    await deleteSyncSource(nsId, row.id)
                    void reload()
                  }}
                >
                  <Button size="mini" status="danger">{t('common.delete')}</Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />

      <Modal
        visible={modalOpen}
        title={editing ? t('knowledgeOps.editSync') : t('knowledgeOps.addSync')}
        onOk={() => void handleSave()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={saving}
        unmountOnExit
        style={{ width: 560 }}
      >
        <Form form={syncForm} layout="vertical">
          <FormItem label={t('knowledgeOps.syncName')} field="name" rules={[{ required: true }]}>
            <Input placeholder={t('knowledgeOps.syncNamePlaceholder')} />
          </FormItem>
          <FormItem label={t('knowledgeOps.syncType')}>
            <Select value={syncSourceType} onChange={setSyncSourceType}>
              <Select.Option value="url">URL</Select.Option>
              <Select.Option value="table">{t('knowledgeOps.syncTypeTable')}</Select.Option>
            </Select>
          </FormItem>
          <FormItem label="URL" field="sourceUrl" rules={[{ required: true }]}>
            <Input placeholder={syncSourceType === 'table' ? 'https://.../data.csv' : 'https://...'} />
          </FormItem>
          <FormItem label={t('knowledgeOps.syncInterval')} field="intervalMinutes" initialValue={1440}>
            <InputNumber min={30} step={30} style={{ width: '100%' }} suffix={t('knowledgeOps.syncMinutesSuffix')} />
          </FormItem>
          {syncSourceType === 'table' ? (
            <>
              <FormItem label={t('knowledgeOps.tableIndexColumns')} field="indexColumns" rules={[{ required: true }]}>
                <Input placeholder={t('knowledgeOps.tableIndexColumns')} />
              </FormItem>
              <Space style={{ width: '100%' }} size={12}>
                <FormItem label={t('knowledgeOps.tableTitleColumn')} field="titleColumn" style={{ flex: 1 }}>
                  <Input />
                </FormItem>
                <FormItem label={t('knowledgeOps.tableKeyColumn')} field="keyColumn" style={{ flex: 1 }}>
                  <Input />
                </FormItem>
              </Space>
              <FormItem label={t('knowledgeOps.syncFormat')} field="tableFormat" initialValue="csv">
                <Select>
                  <Select.Option value="csv">CSV</Select.Option>
                  <Select.Option value="tsv">TSV</Select.Option>
                  <Select.Option value="json">JSON</Select.Option>
                </Select>
              </FormItem>
            </>
          ) : null}
        </Form>
      </Modal>
    </Card>
  )
}
