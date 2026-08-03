import { useMemo, useState } from 'react'
import { Modal, Input, Select, Button, Upload, message } from 'antd'
import { UploadOutlined } from '@ant-design/icons'
import type { UploadFile } from 'antd'
import {
  generateImageToImage,
  getMediaJob,
  parseImageSize,
  resolveMediaUrl,
  friendlyMediaError,
} from '@/api/mediaGenerate'
import { sizeFromResRatio } from '@/pages/Generate/seaartLayout'

const { TextArea } = Input

/** 扩图一致性基础提示词（俯瞰像素地图边缘延续） */
export const MAP_STITCH_BASE_PROMPT = `あなたは俯瞰・拡張マップを専門とするプロの背景アーティストです。入力画像の透明ピクセル領域を、既存の端の描画（重なりエッジ）に基づいて埋め、元のスタイルと完全に一致させ、鮮明・高解像度・鳥瞰図で遠近歪みのないマップにしてください。透明部以外の既存ピクセルはスタイル参照として保持し、継ぎ目が目立たないように自然に拡張してください。`

const RATIOS = ['auto', '1:1', '4:3', '3:4', '16:9', '9:16'] as const
const RESOLUTIONS = ['1K', '2K', '4K'] as const

export type MapStitchApiGenerateResult = {
  file: File
  previewUrl: string
}

type Props = {
  open: boolean
  tileKey: string
  /** Overlap edge template PNG (transparent center + edge context) */
  templateFile: File | null
  templatePreviewUrl: string | null
  onCancel: () => void
  onGenerated: (result: MapStitchApiGenerateResult) => void
}

async function pollUntilDone(jobId: string, onProgress?: (p: number, status: string) => void): Promise<string> {
  const max = 120
  for (let i = 0; i < max; i++) {
    const res = await getMediaJob(jobId)
    const job = res.data
    if (!job) throw new Error(res.msg || '任务查询失败')
    onProgress?.(job.progress ?? Math.min(95, i * 2), job.status)
    if (job.status === 'succeeded') {
      const url = resolveMediaUrl(job.url || job.remoteUrl)
      if (!url) throw new Error('生成成功但未返回图片地址')
      return url
    }
    if (job.status === 'failed' || job.status === 'cancelled' || job.status === 'expired') {
      throw new Error(friendlyMediaError(job.errorMessage, '地图扩图生成失败'))
    }
    await new Promise((r) => setTimeout(r, 2500))
  }
  throw new Error('生成超时，请稍后在「图片生成」历史中查看')
}

export default function MapStitchApiModal({
  open,
  tileKey,
  templateFile,
  templatePreviewUrl,
  onCancel,
  onGenerated,
}: Props) {
  const [extra, setExtra] = useState('')
  const [ratio, setRatio] = useState<(typeof RATIOS)[number]>('auto')
  const [resolution, setResolution] = useState<(typeof RESOLUTIONS)[number]>('1K')
  const [styleRef, setStyleRef] = useState<File | null>(null)
  const [styleList, setStyleList] = useState<UploadFile[]>([])
  const [loading, setLoading] = useState(false)
  const [progress, setProgress] = useState('')

  const sizeLabel = useMemo(() => {
    if (ratio === 'auto') return resolution === '1K' ? '1024x1024' : sizeFromResRatio(resolution, '1:1')
    return sizeFromResRatio(resolution, ratio)
  }, [ratio, resolution])

  const run = async () => {
    if (!templateFile) {
      message.error('请先为该格准备边缘模板（可先点「下载」或直接 API 生成会自动拼模板）')
      return
    }
    setLoading(true)
    setProgress('提交任务…')
    try {
      const prompt = extra.trim()
        ? `${MAP_STITCH_BASE_PROMPT}\n\n追加需求：${extra.trim()}`
        : MAP_STITCH_BASE_PROMPT
      const { width, height } = parseImageSize(sizeLabel)
      const image = templateFile
      const create = await generateImageToImage({
        image,
        prompt: styleRef
          ? `${prompt}\n\n（用户另附风格参考图文件名：${styleRef.name}，请尽量贴近该参考的画风与调色。）`
          : prompt,
        width,
        height,
        style: 'pixel',
        category: 'MAP',
        negative: '文字,水印,UI,透视变形,角色立绘,近景特写',
      })
      if (create.code !== 200 || !create.data?.jobId) {
        throw new Error(create.msg || '创建生成任务失败')
      }
      setProgress('排队生成中…')
      const url = await pollUntilDone(create.data.jobId, (p, status) => {
        setProgress(`${status} · ${p}%`)
      })
      const blob = await fetch(url).then((r) => {
        if (!r.ok) throw new Error('下载生成结果失败')
        return r.blob()
      })
      const file = new File([blob], `map_stitch_${tileKey.replace(',', '_')}_gen.png`, {
        type: blob.type || 'image/png',
      })
      const previewUrl = URL.createObjectURL(file)
      message.success('API 生成完成，已填入该格')
      onGenerated({ file, previewUrl })
    } catch (e) {
      message.error(friendlyMediaError((e as Error).message, String(e)))
    } finally {
      setLoading(false)
      setProgress('')
    }
  }

  return (
    <Modal
      title="本次 API 生成提示词"
      open={open}
      onCancel={onCancel}
      width={720}
      destroyOnClose
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
            <Button onClick={onCancel} disabled={loading}>
              取消
            </Button>
            <Button type="primary" loading={loading} onClick={() => void run()} style={{ background: '#c45c26', borderColor: '#c45c26' }}>
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
        {progress ? <div style={{ fontSize: 12, color: '#c45c26' }}>{progress}</div> : null}
      </div>
    </Modal>
  )
}
