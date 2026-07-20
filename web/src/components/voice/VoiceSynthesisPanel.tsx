import { useCallback, useEffect, useRef, useState } from 'react'
import { Button, Form, Input, Popconfirm, Select, Space, Table, Tag, Typography } from '@arco-design/web-react'
import { IconPlayArrow, IconRefresh } from '@arco-design/web-react/icon'
import {
  deleteVoiceSynthesisHistory,
  listVoiceSynthesisHistory,
  synthesizeVoiceClone,
  type VoiceCloneProfile,
  type VoiceSynthesisHistory,
} from '@/api/voiceClone'
import { playVoicePreviewPcm, playVoicePreviewUrl } from '@/utils/voicePreview'
import { showAlert } from '@/utils/notification'

type VoiceSynthesisPanelProps = {
  profiles: VoiceCloneProfile[]
  onReload?: () => void
}

export default function VoiceSynthesisPanel({ profiles, onReload }: VoiceSynthesisPanelProps) {
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
    return () => {
      mountedRef.current = false
    }
  }, [reloadHistory])

  useEffect(() => {
    if (!profileId && successProfiles.length > 0) {
      setProfileId(successProfiles[0].id)
    }
  }, [profileId, successProfiles])

  const handleSynthesize = async () => {
    if (!profileId) {
      showAlert('请选择已训练完成的音色', 'error')
      return
    }
    const sample = text.trim()
    if (!sample) {
      showAlert('请输入合成文本', 'error')
      return
    }
    setSynthesizing(true)
    try {
      const res = await synthesizeVoiceClone(profileId, sample)
      if (res.code !== 200 || !res.data) {
        showAlert(res.msg || '合成失败', 'error')
        return
      }
      try {
        if (res.data.audioUrl?.trim()) {
          await playVoicePreviewUrl(res.data.audioUrl.trim())
        } else if (res.data.pcmBase64) {
          await playVoicePreviewPcm(res.data.pcmBase64, res.data.sampleRate ?? 16000)
        } else {
          showAlert('合成失败：无音频数据', 'error')
          return
        }
      } catch (e: unknown) {
        showAlert((e as Error)?.message || '播放失败', 'error')
        return
      }
      showAlert('合成成功', 'success')
      void reloadHistory()
      onReload?.()
    } finally {
      setSynthesizing(false)
    }
  }

  const playHistory = async (row: VoiceSynthesisHistory) => {
    if (row.audioUrl) {
      const audio = new Audio(row.audioUrl)
      await audio.play()
      return
    }
    showAlert('暂无音频文件', 'error')
  }

  const handleDeleteHistory = async (id: string) => {
    const res = await deleteVoiceSynthesisHistory(id)
    if (res.code !== 200) {
      showAlert(res.msg || '删除失败', 'error')
      return
    }
    showAlert('已删除', 'success')
    void reloadHistory()
  }

  return (
    <div className="rounded-xl border border-border bg-card p-5 space-y-4">
      <div>
        <Typography.Text bold>音色合成测试</Typography.Text>
        <Typography.Paragraph type="secondary" style={{ marginBottom: 0, fontSize: 13 }}>
          选择已训练完成的音色，输入文本进行合成试听，记录将保存在合成历史中。
        </Typography.Paragraph>
      </div>

      {successProfiles.length === 0 ? (
        <Typography.Text type="secondary">暂无可用音色，请先完成至少一条克隆训练。</Typography.Text>
      ) : (
        <Space direction="vertical" size="medium" className="w-full">
          <Select
            placeholder="选择音色"
            value={profileId}
            onChange={(v) => setProfileId(Number(v))}
            options={successProfiles.map((p) => ({
              value: p.id,
              label: `${p.name} (${p.assetId || p.speakerId || p.id})`,
            }))}
          />
          <Input.TextArea
            value={text}
            onChange={setText}
            autoSize={{ minRows: 2, maxRows: 5 }}
            placeholder="输入要合成的文本"
          />
          <Button type="primary" icon={<IconPlayArrow />} loading={synthesizing} onClick={() => void handleSynthesize()}>
            合成试听
          </Button>
        </Space>
      )}

      <div className="flex items-center justify-between">
        <Typography.Text bold>合成记录</Typography.Text>
        <Button size="mini" icon={<IconRefresh />} onClick={() => void reloadHistory()}>
          刷新
        </Button>
      </div>
      <Table
        rowKey="id"
        loading={historyLoading}
        data={history}
        pagination={false}
        size="small"
        columns={[
          { title: '音色', dataIndex: 'voiceName', width: 120, ellipsis: true },
          {
            title: '文本',
            dataIndex: 'text',
            ellipsis: true,
            render: (v: string) => v || '—',
          },
          {
            title: '状态',
            dataIndex: 'status',
            width: 80,
            render: (s: string) => <Tag color={s === 'success' ? 'green' : 'red'}>{s}</Tag>,
          },
          {
            title: '时间',
            dataIndex: 'createdAt',
            width: 160,
            render: (v: string) => (v ? new Date(v).toLocaleString() : '—'),
          },
          {
            title: '操作',
            width: 120,
            render: (_: unknown, row: VoiceSynthesisHistory) => (
              <Space size="mini">
                {row.status === 'success' ? (
                  <Button size="mini" type="text" onClick={() => void playHistory(row)}>
                    播放
                  </Button>
                ) : null}
                <Popconfirm title="确定删除这条合成记录？" onOk={() => void handleDeleteHistory(String(row.id))}>
                  <Button size="mini" type="text" status="danger">
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />
    </div>
  )
}
