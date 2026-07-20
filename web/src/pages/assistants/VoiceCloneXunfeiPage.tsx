import { useCallback, useEffect, useRef, useState } from 'react'
import { Alert, Button, Drawer, Form, Input, Radio, Select, Space, Spin, Typography } from '@arco-design/web-react'
import { IconRefresh } from '@arco-design/web-react/icon'
import BaseLayout from '@/components/Layout/BaseLayout'
import VoiceCloneListTable from '@/components/voice/VoiceCloneListTable'
import VoiceSynthesisPanel from '@/components/voice/VoiceSynthesisPanel'
import {
  createVoiceClone,
  getVoiceCloneTrainingTexts,
  listVoiceClones,
  type TrainingTextResponse,
  type VoiceCloneProfile,
} from '@/api/voiceClone'
import { useSiteConfig } from '@/contexts/siteConfig'
import { showAlert } from '@/utils/notification'

const XUNFEI_DEFAULT_TEXT_ID = 5001

function parseSegId(v: number | string): number {
  if (typeof v === 'number') return v
  const n = Number(v)
  return Number.isFinite(n) ? n : 1
}

export default function VoiceCloneXunfeiPage() {
  const { config } = useSiteConfig()
  const [rows, setRows] = useState<VoiceCloneProfile[]>([])
  const [loading, setLoading] = useState(true)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [creating, setCreating] = useState(false)
  const [textLoading, setTextLoading] = useState(false)
  const [trainingText, setTrainingText] = useState<TrainingTextResponse | null>(null)
  const [selectedSeg, setSelectedSeg] = useState(0)
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

  const loadTrainingTexts = useCallback(async () => {
    setTextLoading(true)
    try {
      const res = await getVoiceCloneTrainingTexts(XUNFEI_DEFAULT_TEXT_ID)
      if (res.code === 200 && res.data?.segments?.length) {
        setTrainingText(res.data)
        setSelectedSeg(0)
      } else {
        setTrainingText(null)
        showAlert(res.msg || '获取训练文本失败，请稍后重试', 'error')
      }
    } finally {
      setTextLoading(false)
    }
  }, [])

  const openDrawer = () => {
    setDrawerOpen(true)
    form.resetFields()
    form.setFieldsValue({ sex: 2 })
    void loadTrainingTexts()
  }

  const handleCreate = async () => {
    try {
      const values = await form.validate()
      const segments = trainingText?.segments ?? []
      const seg = segments[selectedSeg]
      if (!seg) {
        showAlert('请先选择训练文本段落', 'error')
        return
      }
      setCreating(true)
      const res = await createVoiceClone({
        name: String(values.name || '').trim(),
        sex: values.sex,
        language: 'zh',
        textId: trainingText?.text_id ?? XUNFEI_DEFAULT_TEXT_ID,
        textSegId: parseSegId(seg.seg_id),
        trainText: seg.seg_text,
      })
      if (res.code !== 200) {
        showAlert(res.msg || '创建失败', 'error')
        return
      }
      showAlert('任务已创建，请按所选文本录制并上传 WAV 音频', 'success')
      setDrawerOpen(false)
      void reload()
    } catch {
      /* validation */
    } finally {
      setCreating(false)
    }
  }

  const label = config.VOICE_CLONE_LABEL || '讯飞星火'

  return (
    <BaseLayout>
      <div className="mx-auto max-w-5xl space-y-4 p-6">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <Typography.Title heading={5} style={{ margin: 0 }}>
              音色克隆 · 讯飞
            </Typography.Title>
            <Typography.Text type="secondary" style={{ fontSize: 13 }}>
              从讯飞获取训练文本 → 创建任务 → 按文本录制并上传音频
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

        <Alert type="info" content={<span>当前引擎：<strong>{label}</strong></span>} />

        <VoiceCloneListTable rows={rows} loading={loading} onReload={() => void reload()} />
        <VoiceSynthesisPanel profiles={rows} onReload={() => void reload()} />
      </div>

      <Drawer
        title="新建讯飞音色克隆"
        width={520}
        visible={drawerOpen}
        onCancel={() => setDrawerOpen(false)}
        footer={
          <Space>
            <Button onClick={() => setDrawerOpen(false)}>取消</Button>
            <Button type="primary" loading={creating} onClick={() => void handleCreate()}>
              创建任务
            </Button>
          </Space>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item label="名称" field="name" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="例如：客服小美" />
          </Form.Item>
          <Form.Item label="性别" field="sex" initialValue={2}>
            <Select
              options={[
                { value: 1, label: '男' },
                { value: 2, label: '女' },
              ]}
            />
          </Form.Item>

          <Typography.Text bold style={{ display: 'block', marginBottom: 8 }}>
            训练文本（来自讯飞）
          </Typography.Text>
          {textLoading ? (
            <div className="flex justify-center py-6">
              <Spin />
            </div>
          ) : (trainingText?.segments?.length ?? 0) > 0 ? (
            <Radio.Group
              value={selectedSeg}
              onChange={setSelectedSeg}
              direction="vertical"
              className="w-full"
            >
              {trainingText!.segments.map((seg, idx) => (
                <Radio key={idx} value={idx} style={{ alignItems: 'flex-start', marginBottom: 8 }}>
                  <span className="text-sm leading-relaxed">{seg.seg_text}</span>
                </Radio>
              ))}
            </Radio.Group>
          ) : (
            <Typography.Text type="secondary">
              未能加载训练文本，
              <Button type="text" size="mini" onClick={() => void loadTrainingTexts()}>
                点击重试
              </Button>
            </Typography.Text>
          )}
          <Typography.Text type="secondary" style={{ display: 'block', marginTop: 12, fontSize: 12 }}>
            创建成功后，请按所选段落录制 16kHz 单声道 WAV 并上传。
          </Typography.Text>
        </Form>
      </Drawer>
    </BaseLayout>
  )
}
