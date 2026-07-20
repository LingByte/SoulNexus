import { useCallback, useEffect, useRef, useState } from 'react'
import { Drawer, Form, Input, Upload } from '@arco-design/web-react'
import { Loader2, Plus, Mic, Waves } from 'lucide-react'
import BaseLayout from '@/components/Layout/BaseLayout'
import VoiceCloneListTable from '@/components/voice/VoiceCloneListTable'
import { Button } from '@/components/ui'
import { useSiteConfig } from '@/contexts/siteConfig'
import { cn } from '@/utils/cn'
import {
  createVoiceClone,
  listVoiceClones,
  submitVoiceCloneAudio,
  type VoiceCloneProfile,
} from '@/api/voiceClone'
import { showAlert } from '@/utils/notification'
import VoiceSynthesisTab from '@/components/voice/VoiceSynthesisTab'

export default function VoiceCloneManagerPage() {
  const { ready } = useSiteConfig()
  const [tab, setTab] = useState<'training' | 'synthesis'>('training')
  const [rows, setRows] = useState<VoiceCloneProfile[]>([])
  const [loading, setLoading] = useState(true)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [creating, setCreating] = useState(false)
  const [audioFile, setAudioFile] = useState<File | null>(null)
  const [form] = Form.useForm()
  const mountedRef = useRef(true)

  const reload = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listVoiceClones()
      if (mountedRef.current && res.code === 200 && res.data) setRows(res.data)
    } finally {
      if (mountedRef.current) setLoading(false)
    }
  }, [])

  useEffect(() => {
    mountedRef.current = true
    void reload()
    return () => { mountedRef.current = false }
  }, [reload])

  const openDrawer = () => {
    setDrawerOpen(true)
    setAudioFile(null)
    form.resetFields()
  }

  const handleCreate = async () => {
    try {
      const values = await form.validate()
      if (!audioFile) { showAlert('请上传训练音频', 'error'); return }
      setCreating(true)
      const res = await createVoiceClone({
        name: String(values.name || '').trim(),
        speakerId: String(values.speakerId || '').trim(),
        language: 'zh',
      })
      if (res.code !== 200 || !res.data?.id) { showAlert(res.msg || '创建失败', 'error'); return }
      const uploadRes = await submitVoiceCloneAudio(res.data.id, audioFile)
      if (uploadRes.code !== 200) { showAlert(uploadRes.msg || '音频上传失败', 'error'); return }
      showAlert('已提交训练，请稍后刷新查看进度', 'success')
      setDrawerOpen(false)
      form.resetFields()
      setAudioFile(null)
      void reload()
    } catch { /* validation */ }
    finally { setCreating(false) }
  }

  if (!ready) {
    return (
      <BaseLayout title="音色克隆" description="管理已创建的音色克隆模型">
        <div className="flex items-center justify-center py-24"><Loader2 size={24} className="animate-spin text-neutral-400" /></div>
      </BaseLayout>
    )
  }

  return (
    <BaseLayout title="音色克隆" description="管理已创建的音色克隆模型">
      <>
        <div className="rounded-xl border border-border bg-card">
          <div className="flex items-center justify-between border-b border-neutral-100 px-4 py-2">
            <div className="flex gap-1">
              <button
                type="button"
                className={cn('flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-medium transition', tab === 'training' ? 'bg-violet-50 text-violet-700' : 'text-neutral-500 hover:bg-neutral-50')}
                onClick={() => setTab('training')}
              >
                <Mic size={14} className="text-violet-500" />
                训练管理
              </button>
              <button
                type="button"
                className={cn('flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-medium transition', tab === 'synthesis' ? 'bg-blue-50 text-blue-700' : 'text-neutral-500 hover:bg-neutral-50')}
                onClick={() => setTab('synthesis')}
              >
                <Waves size={14} className="text-blue-500" />
                合成历史
              </button>
            </div>
            <Button type="primary" size="sm" icon={<Plus size={14} />} onClick={openDrawer}>新建克隆</Button>
          </div>
          <div className="p-4">
            {tab === 'training' ? (
              <VoiceCloneListTable rows={rows} loading={loading} onReload={() => void reload()} />
            ) : (
              <VoiceSynthesisTab profiles={rows} onReload={() => void reload()} />
            )}
          </div>
        </div>

      <Drawer
        title="新建音色克隆"
        width={480}
        visible={drawerOpen}
        onCancel={() => setDrawerOpen(false)}
        footer={
          <div className="flex justify-end gap-2">
            <Button type="outline" onClick={() => setDrawerOpen(false)}>取消</Button>
            <Button type="primary" loading={creating} onClick={() => void handleCreate()}>提交训练</Button>
          </div>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item label="名称" field="name" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="例如：客服小美" />
          </Form.Item>
          <Form.Item label="资源 ID（Speaker ID）" field="speakerId" rules={[{ required: true, message: '请输入资源 ID' }]}>
            <Input placeholder="S_xxxxxxxx" />
          </Form.Item>
          <Form.Item label="训练音频" required>
            <Upload
              accept="audio/*,.wav"
              limit={1}
              fileList={audioFile ? [{ uid: '1', name: audioFile.name, originFile: audioFile } as never] : []}
              onChange={(_, file) => { const f = file.originFile instanceof File ? file.originFile : null; setAudioFile(f) }}
              onRemove={() => setAudioFile(null)}
            />
            <p className="mt-1 text-xs text-neutral-400">推荐 16kHz 单声道 WAV</p>
          </Form.Item>
        </Form>
      </Drawer>
      </>
    </BaseLayout>
  )
}
