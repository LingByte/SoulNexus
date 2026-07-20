import { useCallback, useEffect, useRef, useState } from 'react'
import { Alert, Button, Drawer, Form, Input, Space, Typography, Upload } from '@arco-design/web-react'
import { IconRefresh } from '@arco-design/web-react/icon'
import BaseLayout from '@/components/Layout/BaseLayout'
import VoiceCloneListTable from '@/components/voice/VoiceCloneListTable'
import VoiceSynthesisPanel from '@/components/voice/VoiceSynthesisPanel'
import {
  createVoiceClone,
  listVoiceClones,
  submitVoiceCloneAudio,
  type VoiceCloneProfile,
} from '@/api/voiceClone'
import { useSiteConfig } from '@/contexts/siteConfig'
import { showAlert } from '@/utils/notification'

export default function VoiceCloneVolcenginePage() {
  const { config } = useSiteConfig()
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
    return () => {
      mountedRef.current = false
    }
  }, [reload])

  const openDrawer = () => {
    setDrawerOpen(true)
    setAudioFile(null)
    form.resetFields()
  }

  const handleCreate = async () => {
    try {
      const values = await form.validate()
      if (!audioFile) {
        showAlert('请上传训练音频', 'error')
        return
      }
      setCreating(true)
      const res = await createVoiceClone({
        name: String(values.name || '').trim(),
        speakerId: String(values.speakerId || '').trim(),
        language: 'zh',
      })
      if (res.code !== 200 || !res.data?.id) {
        showAlert(res.msg || '创建失败', 'error')
        return
      }
      const uploadRes = await submitVoiceCloneAudio(res.data.id, audioFile)
      if (uploadRes.code !== 200) {
        showAlert(uploadRes.msg || '音频上传失败', 'error')
        return
      }
      showAlert('已提交训练，请稍后刷新查看进度', 'success')
      setDrawerOpen(false)
      form.resetFields()
      setAudioFile(null)
      void reload()
    } catch {
      /* validation */
    } finally {
      setCreating(false)
    }
  }

  const label = config.VOICE_CLONE_LABEL || '火山引擎'

  return (
    <BaseLayout>
      <div className="mx-auto max-w-5xl space-y-4 p-6">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <Typography.Title heading={5} style={{ margin: 0 }}>
              音色克隆 · 火山
            </Typography.Title>
            <Typography.Text type="secondary" style={{ fontSize: 13 }}>
              填写资源 ID 并上传训练音频，完成一句话音色克隆
            </Typography.Text>
          </div>
          <Space>
            <Button icon={<IconRefresh />} onClick={() => void reload()}>
              刷新
            </Button>
            <Button type="primary" onClick={openDrawer}>
              新建克隆
            </Button>
          </Space>
        </div>

        <Alert
          type="info"
          content={
            <span>
              当前引擎：<strong>{label}</strong> · 资源 ID 需由系统管理员分配，如有需要请联系管理员获取
            </span>
          }
        />

        <VoiceCloneListTable rows={rows} loading={loading} onReload={() => void reload()} />
        <VoiceSynthesisPanel profiles={rows} onReload={() => void reload()} />
      </div>

      <Drawer
        title="新建火山音色克隆"
        width={480}
        visible={drawerOpen}
        onCancel={() => setDrawerOpen(false)}
        footer={
          <Space>
            <Button onClick={() => setDrawerOpen(false)}>取消</Button>
            <Button type="primary" loading={creating} onClick={() => void handleCreate()}>
              提交训练
            </Button>
          </Space>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item label="名称" field="name" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="例如：客服小美" />
          </Form.Item>
          <Form.Item
            label="资源 ID（Speaker ID）"
            field="speakerId"
            rules={[{ required: true, message: '请输入资源 ID' }]}
            extra="由管理员分配的 S_ 开头音色资源 ID"
          >
            <Input placeholder="S_xxxxxxxx" />
          </Form.Item>
          <Form.Item label="训练音频" required>
            <Upload
              accept="audio/*,.wav"
              limit={1}
              fileList={
                audioFile
                  ? [{ uid: '1', name: audioFile.name, originFile: audioFile } as never]
                  : []
              }
              onChange={(_, file) => {
                const f = file.originFile instanceof File ? file.originFile : null
                setAudioFile(f)
              }}
              onRemove={() => setAudioFile(null)}
            />
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              推荐 16kHz 单声道 WAV
            </Typography.Text>
          </Form.Item>
        </Form>
      </Drawer>
    </BaseLayout>
  )
}
