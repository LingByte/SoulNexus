import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  Drawer,
  Space,
  Table,
  Tag,
  Typography,
} from '@arco-design/web-react'
import { Button, Empty, Card } from '@/components/ui'
import { Loading } from '@/components/ui/loading'
import { IconArrowLeft } from '@arco-design/web-react/icon'
import BaseLayout from '@/components/Layout/BaseLayout'
import { useTranslation } from '@/i18n'
import {
  getKnowledgeDocument,
  getKnowledgeDocumentChunk,
  getKnowledgeNamespace,
  listKnowledgeDocumentChunks,
  type KnowledgeDocument,
  type KnowledgeDocumentChunk,
  type KnowledgeDocumentChunkDetail,
  type KnowledgeNamespace,
} from '@/api/knowledgeNamespaces'
import { chunkStrategyLabel } from '@/pages/knowledge/chunkStrategy'
import { showAlert } from '@/utils/notification'

export default function KnowledgeDocumentChunks() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { nsId = '', docId = '' } = useParams()

  const [namespace, setNamespace] = useState<KnowledgeNamespace | null>(null)
  const [document, setDocument] = useState<KnowledgeDocument | null>(null)
  const [chunks, setChunks] = useState<KnowledgeDocumentChunk[]>([])
  const [chunkStrategy, setChunkStrategy] = useState('')
  const [loading, setLoading] = useState(true)
  const [detailVisible, setDetailVisible] = useState(false)
  const [detailLoading, setDetailLoading] = useState(false)
  const [detail, setDetail] = useState<KnowledgeDocumentChunkDetail | null>(null)

  const loadAll = useCallback(async () => {
    if (!nsId || !docId) return
    setLoading(true)
    try {
      const [nsRes, docRes, chunkRes] = await Promise.all([
        getKnowledgeNamespace(nsId),
        getKnowledgeDocument(nsId, docId),
        listKnowledgeDocumentChunks(nsId, docId),
      ])
      if (nsRes.code === 200 && nsRes.data) setNamespace(nsRes.data)
      if (docRes.code === 200 && docRes.data) setDocument(docRes.data)
      if (chunkRes.code === 200 && chunkRes.data) {
        setChunks(chunkRes.data.chunks || [])
        setChunkStrategy(chunkRes.data.chunkStrategy || docRes.data?.chunkStrategy || '')
      }
    } catch {
      showAlert(t('common.failed'), 'error')
    }
    setLoading(false)
  }, [docId, nsId, t])

  useEffect(() => { void loadAll() }, [loadAll])

  const openDetail = async (index: number) => {
    if (!nsId || !docId) return
    setDetailVisible(true)
    setDetailLoading(true)
    setDetail(null)
    try {
      const res = await getKnowledgeDocumentChunk(nsId, docId, index)
      if (res.code === 200 && res.data) {
        setDetail(res.data)
      }
    } catch (err: any) {
      showAlert(err?.msg || t('common.failed'), 'error')
      setDetailVisible(false)
    }
    setDetailLoading(false)
  }

  const columns = useMemo(() => [
    { title: t('knowledgeBase.doc.chunkColIndex'), dataIndex: 'index', width: 56 },
    {
      title: t('knowledgeBase.doc.chunkColTitle'),
      dataIndex: 'title',
      width: 160,
      render: (v: string) => v || '-',
    },
    {
      title: t('knowledgeBase.doc.chunkColPreview'),
      dataIndex: 'preview',
      render: (v: string) => (
        <Typography.Paragraph style={{ marginBottom: 0 }} ellipsis={{ rows: 2, showTooltip: true }}>
          {v}
        </Typography.Paragraph>
      ),
    },
    {
      title: t('knowledgeBase.doc.chunkColChars'),
      dataIndex: 'charCount',
      width: 80,
    },
    {
      title: t('knowledgeBase.colActions'),
      width: 100,
      render: (_: unknown, record: KnowledgeDocumentChunk) => (
        <Button type="text" size="small" onClick={() => void openDetail(record.index)}>
          {t('knowledgeBase.doc.chunkViewDetail')}
        </Button>
      ),
    },
  ], [t])

  if (!nsId || !docId) {
    return (
      <BaseLayout title={t('knowledgeBase.doc.chunkPageTitle')}>
        <Empty preset="no-permission" description={t('knowledgeBase.invalidNamespace')} />
      </BaseLayout>
    )
  }

  return (
    <BaseLayout
      title={t('knowledgeBase.doc.chunkPageTitle')}
      description={document?.title || namespace?.name}
      actions={
        <Button
          type="outline"
          size="small"
          icon={<IconArrowLeft />}
          onClick={() => navigate(`/knowledge-base/${nsId}`)}
        >
          {t('knowledgeBase.doc.backToDocuments')}
        </Button>
      }
    >
      {loading ? (
        <Loading block />
      ) : (
        <Card bordered={false} style={{ borderRadius: 12 }}>
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Space wrap>
              <Tag color="blue">{chunkStrategyLabel(t, chunkStrategy)}</Tag>
              <Typography.Text type="secondary">
                {t('knowledgeBase.doc.colChunks')}: {chunks.length}
              </Typography.Text>
            </Space>
            {chunks.length === 0 ? (
              <Empty preset="no-data" description={t('knowledgeBase.doc.chunkEmpty')} />
            ) : (
              <Table
                rowKey="index"
                columns={columns}
                data={chunks}
                pagination={chunks.length > 15 ? { pageSize: 15 } : false}
              />
            )}
          </Space>
        </Card>
      )}

      <Drawer
        width={640}
        title={
          detail
            ? `${t('knowledgeBase.doc.chunkDetailTitle')} #${detail.index}${detail.title ? ` — ${detail.title}` : ''}`
            : t('knowledgeBase.doc.chunkDetailTitle')
        }
        visible={detailVisible}
        onCancel={() => { setDetailVisible(false); setDetail(null) }}
        footer={null}
        unmountOnExit
      >
        {detailLoading ? (
          <Loading block />
        ) : detail ? (
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            <Space wrap>
              <Tag color="arcoblue">{t('knowledgeBase.doc.chunkColChars')}: {detail.charCount}</Tag>
              {detail.chunkStrategy ? (
                <Tag color="blue">{chunkStrategyLabel(t, detail.chunkStrategy)}</Tag>
              ) : null}
            </Space>
            <Typography.Paragraph style={{ marginBottom: 0, whiteSpace: 'pre-wrap' }}>
              {detail.content}
            </Typography.Paragraph>
          </Space>
        ) : (
          <Empty preset="500" description={t('common.failed')} />
        )}
      </Drawer>
    </BaseLayout>
  )
}
