import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Form, Space, Tag } from '@arco-design/web-react'
import { Card, Input, Button, Empty } from '@/components/ui'
import { Loading } from '@/components/ui/loading'
import { IconArrowLeft } from '@arco-design/web-react/icon'
import BaseLayout from '@/components/Layout/BaseLayout'
import { useTranslation } from '@/i18n'
import {
  getKnowledgeDocument,
  getKnowledgeDocumentContent,
  getKnowledgeNamespace,
  updateKnowledgeDocument,
  type KnowledgeDocument,
  type KnowledgeNamespace,
} from '@/api/knowledgeNamespaces'
import { showAlert } from '@/utils/notification'

const FormItem = Form.Item
const TextArea = Input.TextArea

export default function KnowledgeDocumentEdit() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { nsId = '', docId = '' } = useParams()
  const [form] = Form.useForm()
  const [namespace, setNamespace] = useState<KnowledgeNamespace | null>(null)
  const [document, setDocument] = useState<KnowledgeDocument | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  const loadAll = useCallback(async () => {
    if (!nsId || !docId) return
    setLoading(true)
    try {
      const [nsRes, docRes, contentRes] = await Promise.all([
        getKnowledgeNamespace(nsId),
        getKnowledgeDocument(nsId, docId),
        getKnowledgeDocumentContent(nsId, docId),
      ])
      if (nsRes.code === 200 && nsRes.data) setNamespace(nsRes.data)
      if (docRes.code === 200 && docRes.data) setDocument(docRes.data)
      if (contentRes.code === 200 && contentRes.data) {
        form.setFieldsValue({
          title: contentRes.data.title,
          content: contentRes.data.content,
        })
      }
    } catch {
      showAlert(t('common.failed'), 'error')
    }
    setLoading(false)
  }, [docId, form, nsId, t])

  useEffect(() => { void loadAll() }, [loadAll])

  const handleSave = async () => {
    if (!nsId || !docId) return
    const values = await form.validate()
    setSaving(true)
    try {
      await updateKnowledgeDocument(nsId, docId, {
        title: values.title,
        content: values.content,
      })
      showAlert(t('knowledgeBase.doc.uploadQueued'), 'success')
      navigate(`/knowledge-base/${nsId}`)
    } catch (err: any) {
      showAlert(err?.msg || t('common.failed'), 'error')
    }
    setSaving(false)
  }

  if (!nsId || !docId) {
    return (
      <BaseLayout title={t('knowledgeBase.doc.edit')}>
        <Empty preset="no-permission" description={t('knowledgeBase.invalidNamespace')} />
      </BaseLayout>
    )
  }

  return (
    <BaseLayout
      title={t('knowledgeBase.doc.editPageTitle')}
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
        <Card bordered={false} style={{ borderRadius: 12, maxWidth: 960 }}>
          {document?.status === 'processing' ? (
            <Space direction="vertical" size={12}>
              <Tag color="orangered">{t('knowledgeBase.doc.statusProcessing')}</Tag>
              <Empty preset="no-data" description={t('knowledgeBase.doc.editWhileProcessing')} />
            </Space>
          ) : (
            <Form form={form} layout="vertical">
              <FormItem label={t('knowledgeBase.doc.colTitle')} field="title" rules={[{ required: true }]}>
                <Input placeholder={t('knowledgeBase.doc.titlePlaceholder')} />
              </FormItem>
              <FormItem label={t('knowledgeBase.doc.content')} field="content" rules={[{ required: true }]}>
                <TextArea rows={24} style={{ fontFamily: 'monospace' }} />
              </FormItem>
              <Space>
                <Button type="primary" loading={saving} onClick={() => void handleSave()}>
                  {t('knowledgeBase.doc.saveReindex')}
                </Button>
                <Button onClick={() => navigate(`/knowledge-base/${nsId}`)}>
                  {t('common.cancel')}
                </Button>
              </Space>
            </Form>
          )}
        </Card>
      )}
    </BaseLayout>
  )
}
