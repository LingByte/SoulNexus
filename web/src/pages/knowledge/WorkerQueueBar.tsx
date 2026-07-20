import { Space, Tag, Typography } from '@arco-design/web-react'
import { useTranslation } from '@/i18n'
import type { KnowledgeWorkerJob, KnowledgeWorkerSnapshot } from '@/api/knowledgeOps'

type Props = {
  snapshot: KnowledgeWorkerSnapshot | null
  compact?: boolean
}

export function workerJobKindLabel(t: (key: string) => string, kind: KnowledgeWorkerJob['kind']) {
  switch (kind) {
    case 'purge':
      return t('knowledgeOps.workerJobKindPurge')
    case 'sync':
      return t('knowledgeOps.workerJobKindSync')
    default:
      return t('knowledgeOps.workerJobKindIngest')
  }
}

export function formatDocQueueHint(
  t: (key: string, params?: Record<string, string>) => string,
  job?: KnowledgeWorkerJob,
) {
  if (!job) return null
  if (job.taskStatus === 'running') return t('knowledgeBase.doc.queueRunning')
  if (job.queueAhead <= 0) return t('knowledgeBase.doc.queueNext')
  return t('knowledgeBase.doc.queueAhead', { n: String(job.queueAhead) })
}

export default function KnowledgeWorkerQueueBar({ snapshot, compact }: Props) {
  const { t } = useTranslation()
  if (!snapshot || snapshot.unfinished <= 0) return null

  if (compact) {
    return (
      <Space size={8} wrap>
        <Tag color="gray">{t('knowledgeOps.workerQueued')}: {snapshot.queued}</Tag>
        <Tag color="blue">{t('knowledgeOps.workerRunning')}: {snapshot.running}</Tag>
      </Space>
    )
  }

  return (
    <Space direction="vertical" size={8} style={{ width: '100%' }}>
      <Space size={8} wrap>
        <Tag color="gray">{t('knowledgeOps.workerQueued')}: {snapshot.queued}</Tag>
        <Tag color="blue">{t('knowledgeOps.workerRunning')}: {snapshot.running}</Tag>
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          {t('knowledgeBase.doc.workerQueueSummary', {
            queued: String(snapshot.queued),
            running: String(snapshot.running),
          })}
        </Typography.Text>
      </Space>
    </Space>
  )
}
