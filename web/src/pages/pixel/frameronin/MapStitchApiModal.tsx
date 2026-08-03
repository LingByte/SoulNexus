import { useMemo, useState } from 'react'
import { Drawer, Input, Select, Button, Upload, message } from 'antd'
import { UploadOutlined } from '@ant-design/icons'
import type { UploadFile } from 'antd'
import { parseImageSize } from '@/api/mediaGenerate'
import { sizeFromResRatio } from '@/pages/Generate/seaartLayout'

const { TextArea } = Input

/** 扩图一致性基础提示词（俯瞰像素地图边缘延续） */
export const MAP_STITCH_BASE_PROMPT =
  '你是一位专精俯瞰扩展地图的专业背景画师。请根据输入图像中已有的边缘描画（重叠边缘），填补透明像素区域，使新内容与原风格完全一致，画面清晰、高分辨率，采用鸟瞰视角且无透视变形。透明区域以外的已有像素请作为风格参考予以保留，并自然扩展，使接缝不明显。'

const RATIOS = ['auto', '1:1', '4:3', '3:4', '16:9', '9:16'] as const
const RESOLUTIONS = ['1K', '2K', '4K'] as const

export type MapStitchApiSubmitParams = {
  prompt: string
  width: number
  height: number
  styleRef: File | null
  sizeLabel: string
}

type Props = {
  open: boolean
  tileKey: string
  /** Overlap edge template PNG (transparent center + edge context) */
  templateFile: File | null
  templatePreviewUrl: string | null
  onClose: () => void
  /** 提交后抽屉关闭，由父组件异步生成并填格展示 */
  onSubmit: (params: MapStitchApiSubmitParams) => void
}

export default function MapStitchApiModal({
  open,
  tileKey,
  templateFile,
  templatePreviewUrl,
  onClose,
  onSubmit,
}: Props) {
  const [extra, setExtra] = useState('')
  const [ratio, setRatio] = useState<(typeof RATIOS)[number]>('auto')
  const [resolution, setResolution] = useState<(typeof RESOLUTIONS)[number]>('1K')
  const [styleRef, setStyleRef] = useState<File | null>(null)
  const [styleList, setStyleList] = useState<UploadFile[]>([])

  const sizeLabel = useMemo(() => {
    if (ratio === 'auto') return resolution === '1K' ? '1024x1024' : sizeFromResRatio(resolution, '1:1')
    return sizeFromResRatio(resolution, ratio)
  }, [ratio, resolution])

  const handleStart = () => {
    if (!templateFile) {
      message.error('请先为该格准备边缘模板（可先点「下载」或直接 API 生成会自动拼模板）')
      return
    }
    const prompt = extra.trim()
      ? `${MAP_STITCH_BASE_PROMPT}\n\n追加需求：${extra.trim()}`
      : MAP_STITCH_BASE_PROMPT
    const { width, height } = parseImageSize(sizeLabel)
    onSubmit({
      prompt: styleRef
        ? `${prompt}\n\n（用户另附风格参考图文件名：${styleRef.name}，请尽量贴近该参考的画风与调色。）`
        : prompt,
      width,
      height,
      styleRef,
      sizeLabel,
    })
  }

  return (
    <Drawer
      title="本次 API 生成提示词"
      open={open}
      onClose={onClose}
      width={Math.min(520, typeof window !== 'undefined' ? window.innerWidth - 24 : 520)}
      destroyOnClose
      placement="right"
      footer={
        <div style={{ display: 'flex', justifyContent: 'space-between', gap: 8, flexWrap: 'wrap' }}>
          <Upload
            accept="image/*"
            maxCount={1}
            fileList={styleList}
            beforeUpload={(f) => {
              setStyleRef(f)
              setStyleList([{ uid: '1', name: f.name, status: 'done' }])
              return false
            }}
            onRemove={() => {
              setStyleRef(null)
              setStyleList([])
            }}
            showUploadList={false}
          >
            <Button icon={<UploadOutlined />}>风格参考</Button>
          </Upload>
          <div style={{ display: 'flex', gap: 8 }}>
            <Button onClick={onClose}>取消</Button>
            <Button type="primary" onClick={handleStart} style={{ background: '#c45c26', borderColor: '#c45c26' }}>
              开始生成
            </Button>
          </div>
        </div>
      }
    >
      <div style={{ display: 'grid', gap: 14 }}>
        <div>
          <div style={{ marginBottom: 6, fontWeight: 600 }}>基础提示词</div>
          <TextArea value={MAP_STITCH_BASE_PROMPT} readOnly autoSize={{ minRows: 5, maxRows: 10 }} />
        </div>
        <div>
          <div style={{ marginBottom: 6, fontWeight: 600 }}>本次追加需求（可选）</div>
          <TextArea
            value={extra}
            onChange={(e) => setExtra(e.target.value)}
            placeholder="例如：白色空区域生成一片废墟广场，中央有破损石像，保持像素地图风格。留空则只使用基础提示词。"
            autoSize={{ minRows: 3, maxRows: 6 }}
          />
        </div>
        <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap' }}>
          <div>
            <div style={{ marginBottom: 6, fontWeight: 600 }}>图片比例</div>
            <Select
              style={{ width: 140 }}
              value={ratio}
              onChange={setRatio}
              options={RATIOS.map((v) => ({ value: v, label: v }))}
            />
          </div>
          <div>
            <div style={{ marginBottom: 6, fontWeight: 600 }}>分辨率</div>
            <Select
              style={{ width: 140 }}
              value={resolution}
              onChange={setResolution}
              options={RESOLUTIONS.map((v) => ({ value: v, label: v.toLowerCase() }))}
            />
          </div>
          <div style={{ alignSelf: 'flex-end', color: '#888', fontSize: 12 }}>输出约 {sizeLabel}</div>
        </div>
        {styleRef ? (
          <div style={{ fontSize: 12, color: '#666' }}>
            风格参考：{styleRef.name}（会写入提示词；图生图输入仍使用边缘模板以保障接缝一致）
          </div>
        ) : null}
        {templatePreviewUrl ? (
          <div>
            <div style={{ marginBottom: 6, fontWeight: 600 }}>边缘模板预览（格 {tileKey}）</div>
            <div
              style={{
                maxHeight: 180,
                overflow: 'auto',
                border: '1px solid #ddd',
                borderRadius: 8,
                background:
                  'repeating-conic-gradient(#d9d0c2 0% 25%, #eee7dc 0% 50%) 50% / 16px 16px',
                padding: 8,
              }}
            >
              <img src={templatePreviewUrl} alt="template" style={{ maxWidth: '100%', imageRendering: 'pixelated' }} />
            </div>
          </div>
        ) : null}
      </div>
    </Drawer>
  )
}
