import { useEffect, useState } from 'react'
import { Button, Empty, Input } from '@/components/ui'
import { Download, Upload } from 'lucide-react'
import { expandShrinkImage } from '@/lib/pixel/expandShrink'
import { triggerDownload } from '@/lib/pixel/imageExport'
import { showAlert } from '@/utils/notification'

export default function ExpandShrinkWorkspace() {
  const [file, setFile] = useState<File | null>(null)
  const [preview, setPreview] = useState<string | null>(null)
  const [result, setResult] = useState<string | null>(null)
  const [resultBlob, setResultBlob] = useState<Blob | null>(null)
  const [cols, setCols] = useState(4)
  const [rows, setRows] = useState(4)
  const [cellW, setCellW] = useState(32)
  const [cellH, setCellH] = useState(32)
  const [loading, setLoading] = useState(false)

  useEffect(
    () => () => {
      if (preview) URL.revokeObjectURL(preview)
      if (result) URL.revokeObjectURL(result)
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  )

  const onUpload = (f: File) => {
    if (preview) URL.revokeObjectURL(preview)
    if (result) URL.revokeObjectURL(result)
    setFile(f)
    setPreview(URL.createObjectURL(f))
    setResult(null)
    setResultBlob(null)
  }

  const run = async () => {
    if (!file) return
    setLoading(true)
    try {
      const blob = await expandShrinkImage(file, cols, rows, cellW, cellH)
      if (result) URL.revokeObjectURL(result)
      setResultBlob(blob)
      setResult(URL.createObjectURL(blob))
      showAlert('处理完成', 'success')
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
          上传图片
          <input hidden type="file" accept="image/*" onChange={(e) => e.target.files?.[0] && onUpload(e.target.files[0])} />
        </label>
        <div className="grid grid-cols-2 gap-2">
          <Input addBefore="列 N" type="number" value={String(cols)} onChange={(v) => setCols(Math.max(1, Number(v) || 1))} />
          <Input addBefore="行 M" type="number" value={String(rows)} onChange={(v) => setRows(Math.max(1, Number(v) || 1))} />
          <Input addBefore="格宽" type="number" value={String(cellW)} onChange={(v) => setCellW(Math.max(1, Number(v) || 1))} />
          <Input addBefore="格高" type="number" value={String(cellH)} onChange={(v) => setCellH(Math.max(1, Number(v) || 1))} />
        </div>
        <p className="text-xs text-muted-foreground">每格从中心裁出 cellW×cellH（不缩放）；小于目标时透明填充。</p>
        <Button type="primary" block loading={loading} disabled={!file} onClick={() => void run()}>
          扩/缩处理
        </Button>
        <Button block variant="outline" disabled={!resultBlob} icon={<Download size={14} />} onClick={() => resultBlob && triggerDownload(resultBlob, 'expand_shrink.png')}>
          下载 PNG
        </Button>
      </div>
      <div className="rounded-xl border border-border bg-card p-4">
        {result || preview ? (
          <div className="grid min-h-[360px] place-items-center bg-[repeating-conic-gradient(#e5e5e5_0%_25%,#fff_0%_50%)] bg-[length:16px_16px] rounded-lg">
            <img src={result || preview!} alt="" className="max-h-[520px] max-w-full object-contain" />
          </div>
        ) : (
          <Empty description="上传后预览" />
        )}
      </div>
      <div className="rounded-xl border border-border bg-card p-4 text-sm text-muted-foreground">
        <p>适合把精灵表单元格统一成固定尺寸（扩图补透明 / 缩图中心裁）。</p>
      </div>
    </div>
  )
}
