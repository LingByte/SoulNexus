import { useCallback, useEffect, useRef, useState } from 'react'
import { Modal } from '@arco-design/web-react'
import { Play, Trash2, Volume2, RefreshCw } from 'lucide-react'
import { Button, DataList, Input, Select } from '@/components/ui'
import type { DataListColumn } from '@/components/ui'
import {
  deleteVoiceSynthesisHistory,
  listVoiceSynthesisHistory,
  synthesizeVoiceClone,
  type VoiceCloneProfile,
  type VoiceSynthesisHistory,
} from '@/api/voiceClone'
import { playVoicePreviewPcm } from '@/utils/voicePreview'
import { showAlert } from '@/utils/notification'

type Props = {
  profiles: VoiceCloneProfile[]
  onReload?: () => void
}

export default function VoiceSynthesisTab({ profiles, onReload }: Props) {
  const successProfiles = profiles.filter((p) => p.status === 'success')
  const [profileId, setProfileId] = useState<number | undefined>()
  const [text, setText] = useState('您好，这是音色合成测试。')
  const [synthesizing, setSynthesizing] = useState(false)
  const [history, setHistory] = useState<VoiceSynthesisHistory[]>([])
  const [historyLoading, setHistoryLoading] = useState(false)
  const mountedRef = useRef(true)

  const reloadHistory = useCallback(async () => {
    setHistoryLoading(true)
    try {
      const res = await listVoiceSynthesisHistory(profileId)
      if (mountedRef.current && res.code === 200 && res.data) setHistory(res.data)
    } finally {
      if (mountedRef.current) setHistoryLoading(false)
    }
  }, [profileId])

  useEffect(() => {
    mountedRef.current = true
    void reloadHistory()
    return () => { mountedRef.current = false }
  }, [reloadHistory])

  useEffect(() => {
    if (!profileId && successProfiles.length > 0) setProfileId(successProfiles[0].id)
  }, [profileId, successProfiles])

  const handleSynthesize = async () => {
    if (!profileId) { showAlert('请选择已训练完成的音色', 'error'); return }
    const sample = text.trim()
    if (!sample) { showAlert('请输入合成文本', 'error'); return }
    setSynthesizing(true)
    try {
      const res = await synthesizeVoiceClone(profileId, sample)
      if (res.code !== 200 || !res.data?.pcmBase64) { showAlert(res.msg || '合成失败', 'error'); return }
      await playVoicePreviewPcm(res.data.pcmBase64, res.data.sampleRate ?? 16000)
      showAlert('合成成功', 'success'); void reloadHistory(); onReload?.()
    } finally { setSynthesizing(false) }
  }

  const playHistory = async (row: VoiceSynthesisHistory) => {
    if (row.audioUrl) { const audio = new Audio(row.audioUrl); await audio.play(); return }
    showAlert('暂无音频文件', 'error')
  }

  const handleDeleteHistory = async (id: string) => {
    const res = await deleteVoiceSynthesisHistory(id)
    if (res.code !== 200) { showAlert(res.msg || '删除失败', 'error'); return }
    showAlert('已删除', 'success'); void reloadHistory()
  }

  const columns: DataListColumn<Record<string, unknown>>[] = [
    {
      key: 'icon', width: 40,
      render: () => (
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-blue-50 text-blue-600">
          <Volume2 size={18} strokeWidth={1.75} />
        </div>
      ),
    },
    {
      key: 'info', title: '音色 / 文本',
      render: (_, r) => (
        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-medium text-neutral-900">{String(r.voiceName || '—')}</div>
          <div className="mt-0.5 truncate text-xs text-neutral-500">{String(r.text || '—')}</div>
        </div>
      ),
    },
    {
      key: 'status', title: '状态', width: 80,
      render: (_, r) => <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${String(r.status) === 'success' ? 'bg-emerald-50 text-emerald-700' : 'bg-red-50 text-red-700'}`}>{String(r.status || '—')}</span>,
    },
    {
      key: 'time', title: '时间', width: 170,
      render: (_, r) => <span className="text-sm text-neutral-700">{r.createdAt ? new Date(String(r.createdAt)).toLocaleString() : '—'}</span>,
    },
    {
      key: 'actions', width: 120, align: 'right',
      render: (_, r) => {
        const row = r as unknown as VoiceSynthesisHistory
        return (
          <div className="flex items-center justify-end gap-1">
            {row.status === 'success' ? (
              <Button size="mini" icon={<Play size={12} />} onClick={() => void playHistory(row)}>播放</Button>
            ) : null}
            <Button size="mini" status="danger" icon={<Trash2 size={12} />} onClick={() => {
              Modal.confirm({ title: '删除合成记录', content: '确定删除这条合成记录？', onOk: () => void handleDeleteHistory(String(row.id)) })
            }}>删除</Button>
          </div>
        )
      },
    },
  ]

  return (
    <div className="space-y-4">
      <div className="rounded-xl border border-neutral-100 bg-white p-4">
        <div className="mb-3 text-sm font-medium text-neutral-900">合成测试</div>
        {successProfiles.length === 0 ? (
          <p className="text-sm text-neutral-400">暂无可用音色，请先完成至少一条克隆训练。</p>
        ) : (
          <div className="space-y-3">
            <Select
              placeholder="选择音色"
              value={profileId}
              onChange={(v) => setProfileId(Number(v))}
              options={successProfiles.map((p) => ({ value: p.id, label: `${p.name} (${p.assetId || p.speakerId || p.id})` }))}
            />
            <Input.TextArea
              value={text}
              onChange={setText}
              autoSize={{ minRows: 2, maxRows: 5 }}
              placeholder="输入要合成的文本"
            />
            <Button type="primary" icon={<Play size={14} />} loading={synthesizing} onClick={() => void handleSynthesize()}>
              合成试听
            </Button>
          </div>
        )}
      </div>

      <DataList
        data={history as unknown as (VoiceSynthesisHistory & Record<string, unknown>)[]}
        columns={columns}
        loading={historyLoading}
        rowKey="id"
        emptyText="暂无合成记录"
        header={
          <div className="flex items-center justify-between gap-2">
            <span className="text-sm font-medium text-neutral-900">合成记录</span>
            <Button type="outline" icon={<RefreshCw size={14} />} onClick={() => void reloadHistory()}>刷新</Button>
          </div>
        }
      />
    </div>
  )
}
