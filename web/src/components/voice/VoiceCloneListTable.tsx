import { Modal, Upload } from '@arco-design/web-react'
import { Trash2, Play, RefreshCw, Upload as UploadIcon } from 'lucide-react'
import { Button, DataList } from '@/components/ui'
import type { DataListColumn } from '@/components/ui'
import {
  deleteVoiceClone,
  previewVoiceClone,
  submitVoiceCloneAudio,
  syncVoiceCloneStatus,
  type VoiceCloneProfile,
} from '@/api/voiceClone'
import { playVoicePreviewPcm, playVoicePreviewUrl } from '@/utils/voicePreview'
import { showAlert } from '@/utils/notification'

const statusColor: Record<string, string> = { pending: 'gray', training: 'arcoblue', success: 'green', failed: 'red' }
const statusLabel: Record<string, string> = { pending: '待训练', training: '训练中', success: '已完成', failed: '失败' }

export type VoiceCloneListTableProps = {
  rows: VoiceCloneProfile[]
  loading?: boolean
  onReload: () => void
}

export default function VoiceCloneListTable({ rows, loading, onReload }: VoiceCloneListTableProps) {
  const handleUpload = async (row: VoiceCloneProfile, file: File) => {
    const res = await submitVoiceCloneAudio(row.id, file)
    if (res.code !== 200) { showAlert(res.msg || '上传失败', 'error'); return }
    showAlert('音频已提交，正在训练', 'success'); onReload()
  }

  const handleSync = async (row: VoiceCloneProfile) => {
    const res = await syncVoiceCloneStatus(row.id)
    if (res.code !== 200) { showAlert(res.msg || '查询失败', 'error'); return }
    onReload()
    if (res.data?.status === 'success') showAlert('训练完成', 'success')
    else if (res.data?.status === 'failed') showAlert(res.data.failedReason || '训练失败', 'error')
  }

  const handlePreview = async (row: VoiceCloneProfile) => {
    const res = await previewVoiceClone(row.id)
    if (res.code !== 200 || !res.data) {
      showAlert(res.msg || '试听失败', 'error')
      return
    }
    try {
      if (res.data.audioUrl?.trim()) {
        await playVoicePreviewUrl(res.data.audioUrl.trim())
        return
      }
      if (res.data.pcmBase64) {
        await playVoicePreviewPcm(res.data.pcmBase64, res.data.sampleRate ?? 16000)
        return
      }
      showAlert('试听失败：无音频数据', 'error')
    } catch (e: unknown) {
      showAlert((e as Error)?.message || '试听播放失败', 'error')
    }
  }

  const handleDelete = (row: VoiceCloneProfile) => {
    Modal.confirm({
      title: '删除克隆音色',
      content: `确定删除「${row.name}」？`,
      onOk: async () => {
        const res = await deleteVoiceClone(row.id)
        if (res.code !== 200) { showAlert(res.msg || '删除失败', 'error'); return }
        onReload()
      },
    })
  }

  const columns: DataListColumn<Record<string, unknown>>[] = [
    {
      key: 'icon', width: 40,
      render: () => (
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-violet-50 text-violet-600">
          <UserCircle size={18} strokeWidth={1.75} />
        </div>
      ),
    },
    {
      key: 'info', title: '名称',
      render: (_, r) => (
        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-medium text-neutral-900">{String(r.name || '—')}</div>
          <div className="mt-0.5 text-xs text-neutral-500">{String(r.assetId || r.speakerId || '—')}</div>
        </div>
      ),
    },
    {
      key: 'status', title: '状态', width: 100,
      render: (_, r) => <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${statusColor[String(r.status)] === 'green' ? 'bg-emerald-50 text-emerald-700' : statusColor[String(r.status)] === 'red' ? 'bg-red-50 text-red-700' : statusColor[String(r.status)] === 'arcoblue' ? 'bg-blue-50 text-blue-700' : 'bg-neutral-100 text-neutral-600'}`}>{statusLabel[String(r.status)] || String(r.status || '—')}</span>,
    },
    {
      key: 'reason', title: '失败原因', width: 140,
      render: (_, r) => <span className="truncate text-xs text-neutral-500">{String(r.failedReason || '—')}</span>,
    },
    {
      key: 'actions', width: 200, align: 'right',
      render: (_, r) => {
        const row = r as unknown as VoiceCloneProfile
        return (
          <div className="flex items-center justify-end gap-1">
            {row.status !== 'success' ? (
              <>
                <Upload accept="audio/*,.wav" showUploadList={false} customRequest={({ file }) => { const f = file instanceof File ? file : (file as { originFile?: File }).originFile; if (f) void handleUpload(row, f) }}>
                  <Button size="mini" icon={<UploadIcon size={12} />}>上传音频</Button>
                </Upload>
                <Button size="mini" icon={<RefreshCw size={12} />} onClick={() => void handleSync(row)}>查进度</Button>
              </>
            ) : (
              <Button size="mini" icon={<Play size={12} />} onClick={() => void handlePreview(row)}>试听</Button>
            )}
            <Button size="mini" status="danger" icon={<Trash2 size={12} />} onClick={() => handleDelete(row)} />
          </div>
        )
      },
    },
  ]

  return (
    <DataList
      data={rows as unknown as (VoiceCloneProfile & Record<string, unknown>)[]}
      columns={columns}
      loading={loading}
      rowKey="id"
      emptyText="暂无克隆音色"
    />
  )
}

function UserCircle({ size, strokeWidth }: { size: number; strokeWidth?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={strokeWidth || 2} strokeLinecap="round" strokeLinejoin="round">
      <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
      <circle cx="12" cy="7" r="4" />
    </svg>
  )
}
