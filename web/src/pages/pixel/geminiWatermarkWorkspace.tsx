import { useEffect, useState } from 'react'
import { Button, Empty, Select } from '@/components/ui'
import { Download, Upload } from 'lucide-react'
import {
  getWatermarkParams,
  getWatermarkSize,
  removeGeminiWatermarkFromBlob,
  type WatermarkSize,
} from '@/lib/pixel/geminiWatermark'
import { blobToImage } from '@/lib/pixel/canvasUtils'
import { triggerDownload } from '@/lib/pixel/imageExport'
import { showAlert } from '@/utils/notification'

export default function GeminiWatermarkWorkspace() {
  const [file, setFile] = useState<File | null>(null)
  const [preview, setPreview] = useState<string | null>(null)
  const [result, setResult] = useState<string | null>(null)
  const [resultBlob, setResultBlob] = useState<Blob | null>(null)
  const [sizeMode, setSizeMode] = useState<'auto' | WatermarkSize>('auto')
  const [meta, setMeta] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(
    () => () => {
      if (preview) URL.revokeObjectURL(preview)
      if (result) URL.revokeObjectURL(result)
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  )

  const onUpload = async (f: File) => {
    if (preview) URL.revokeObjectURL(preview)
    if (result) URL.revokeObjectURL(result)
    setFile(f)
    setPreview(URL.createObjectURL(f))
    setResult(null)
    setResultBlob(null)
    try {
      const img = await blobToImage(f)
      const auto = getWatermarkSize(img.naturalWidth, img.naturalHeight)
      const p = getWatermarkParams(img.naturalWidth, img.naturalHeight, auto)
      setMeta(`${img.naturalWidth}×${img.naturalHeight} · 建议 ${auto}px @ (${p.x},${p.y})`)
    } catch {
      setMeta('')
    }
  }

  const run = async () => {
    if (!file) return
    setLoading(true)
    try {
      let blob: Blob
      if (sizeMode === 'auto') {
        blob = await removeGeminiWatermarkFromBlob(file)
      } else {
        // Force size by temporarily wrapping — reuse auto path after override via blob pipeline
        const img = await blobToImage(file)
        const { removeWatermarkReverseAlpha, getEmbeddedAlphaMask } = await import('@/lib/pixel/geminiWatermark')
        const params = getWatermarkParams(img.naturalWidth, img.naturalHeight, sizeMode)
        const mask = await getEmbeddedAlphaMask(sizeMode)
        const canvas = document.createElement('canvas')
        canvas.width = img.naturalWidth
        canvas.height = img.naturalHeight
        const ctx = canvas.getContext('2d')!
        ctx.drawImage(img, 0, 0)
        const id = ctx.getImageData(0, 0, canvas.width, canvas.height)
        removeWatermarkReverseAlpha(id, mask.alpha, mask.width, mask.height, params.x, params.y)
        ctx.putImageData(id, 0, 0)
        blob = await new Promise<Blob>((res, rej) => canvas.toBlob((b) => (b ? res(b) : rej(new Error('导出失败'))), 'image/png'))
      }
      if (result) URL.revokeObjectURL(result)
      setResultBlob(blob)
      setResult(URL.createObjectURL(blob))
      showAlert('水印已去除', 'success')
    } catch (e) {
      showAlert((e as Error).message || '失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="grid gap-4 xl:grid-cols-[300px_1fr_260px]">
      <div className="space-y-3 rounded-xl border border-border bg-card p-4">
        <label className="flex cursor-pointer flex-col items-center gap-2 rounded-lg border border-dashed border-border px-4 py-8 text-sm text-muted-foreground hover:bg-muted/40">
          <Upload size={18} />
          上传带 Gemini 水印的图
          <input hidden type="file" accept="image/*" onChange={(e) => e.target.files?.[0] && void onUpload(e.target.files[0])} />
        </label>
        {meta ? <p className="text-xs text-muted-foreground">{meta}</p> : null}
        <Select
          value={String(sizeMode)}
          options={[
            { value: 'auto', label: '自动 48/96' },
            { value: '48', label: '强制 48' },
            { value: '96', label: '强制 96' },
          ]}
          onChange={(v) => setSizeMode(v === 'auto' ? 'auto' : (Number(v) as WatermarkSize))}
        />
        <Button type="primary" block loading={loading} disabled={!file} onClick={() => void run()}>
          去除水印
        </Button>
        <Button block variant="outline" disabled={!resultBlob} icon={<Download size={14} />} onClick={() => resultBlob && triggerDownload(resultBlob, 'no_watermark.png')}>
          下载 PNG
        </Button>
      </div>
      <div className="rounded-xl border border-border bg-card p-4">
        {result || preview ? (
          <div className="grid min-h-[360px] place-items-center rounded-lg bg-[repeating-conic-gradient(#e5e5e5_0%_25%,#fff_0%_50%)] bg-[length:16px_16px]">
            <img src={result || preview!} alt="" className="max-h-[520px] max-w-full object-contain" />
          </div>
        ) : (
          <Empty description="上传后预览" />
        )}
      </div>
      <div className="space-y-2 rounded-xl border border-border bg-card p-4 text-sm text-muted-foreground">
        <p>基于 GeminiWatermarkTool 的 Reverse Alpha 算法，纯本地处理，无需 API Key。</p>
        <p>水印默认在右下角；大图用 96px logo，小图用 48px。</p>
      </div>
    </div>
  )
}
