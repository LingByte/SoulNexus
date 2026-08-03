import { useEffect, useState } from 'react'
import { Tabs, Slider } from '@arco-design/web-react'
import { Button, Empty, Select } from '@/components/ui'
import { Download, Upload } from 'lucide-react'
import { mergeNearbyColors, pixelateBlock, reduceTo16Colors } from '@/lib/pixel/pixelate'
import { triggerDownload } from '@/lib/pixel/imageExport'
import { showAlert } from '@/utils/notification'

const TabPane = Tabs.TabPane

export default function PixelateWorkspace() {
  const [tab, setTab] = useState('block')
  const [file, setFile] = useState<File | null>(null)
  const [preview, setPreview] = useState<string | null>(null)
  const [result, setResult] = useState<string | null>(null)
  const [resultBlob, setResultBlob] = useState<Blob | null>(null)
  const [loading, setLoading] = useState(false)
  const [pixelSize, setPixelSize] = useState(8)
  const [mergeStrength, setMergeStrength] = useState(40)
  const [colorMethod, setColorMethod] = useState<'rgb' | 'lab'>('lab')
  const [dither, setDither] = useState(true)

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
      let blob: Blob
      if (tab === 'block') blob = await pixelateBlock(file, pixelSize)
      else if (tab === 'merge') blob = await mergeNearbyColors(file, mergeStrength)
      else blob = await reduceTo16Colors(file, { method: colorMethod, dither })
      if (result) URL.revokeObjectURL(result)
      setResultBlob(blob)
      setResult(URL.createObjectURL(blob))
      showAlert('处理完成', 'success')
    } catch (e) {
      showAlert((e as Error).message || '处理失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-4">
      <Tabs activeTab={tab} onChange={setTab}>
        <TabPane key="block" title="块像素化" />
        <TabPane key="merge" title="相近色合并" />
        <TabPane key="color16" title="16 色量化" />
      </Tabs>
      <div className="grid gap-4 xl:grid-cols-[300px_1fr_260px]">
        <div className="space-y-3 rounded-xl border border-border bg-card p-4">
          <label className="flex cursor-pointer flex-col items-center gap-2 rounded-lg border border-dashed border-border px-4 py-8 text-sm text-muted-foreground hover:bg-muted/40">
            <Upload size={18} />
            上传图片
            <input hidden type="file" accept="image/*" onChange={(e) => e.target.files?.[0] && onUpload(e.target.files[0])} />
          </label>
          {tab === 'block' ? (
            <div>
              <div className="mb-1 flex justify-between text-xs text-muted-foreground"><span>像素块</span><span>{pixelSize}</span></div>
              <Slider min={2} max={64} value={pixelSize} onChange={(v) => setPixelSize(Number(v))} />
            </div>
          ) : null}
          {tab === 'merge' ? (
            <div>
              <div className="mb-1 flex justify-between text-xs text-muted-foreground"><span>合并强度</span><span>{mergeStrength}</span></div>
              <Slider min={0} max={100} value={mergeStrength} onChange={(v) => setMergeStrength(Number(v))} />
            </div>
          ) : null}
          {tab === 'color16' ? (
            <>
              <Select
                value={colorMethod}
                options={[
                  { value: 'lab', label: 'LAB 感知' },
                  { value: 'rgb', label: 'RGB' },
                ]}
                onChange={(v) => setColorMethod(v as 'rgb' | 'lab')}
              />
              <Button size="sm" variant={dither ? 'default' : 'outline'} onClick={() => setDither((d) => !d)}>
                {dither ? '抖动：开' : '抖动：关'}
              </Button>
            </>
          ) : null}
          <Button type="primary" block loading={loading} disabled={!file} onClick={() => void run()}>
            开始处理
          </Button>
          <Button
            block
            variant="outline"
            disabled={!resultBlob}
            icon={<Download size={14} />}
            onClick={() => resultBlob && triggerDownload(resultBlob, 'pixelated.png')}
          >
            下载 PNG
          </Button>
        </div>
        <div className="rounded-xl border border-border bg-card p-4">
          {result || preview ? (
            <div className="grid min-h-[360px] place-items-center bg-[repeating-conic-gradient(#e5e5e5_0%_25%,#fff_0%_50%)] bg-[length:16px_16px] rounded-lg">
              <img src={result || preview!} alt="preview" className="max-h-[520px] max-w-full object-contain" style={{ imageRendering: 'pixelated' }} />
            </div>
          ) : (
            <Empty description="上传后预览" />
          )}
        </div>
        <div className="space-y-2 rounded-xl border border-border bg-card p-4 text-sm text-muted-foreground">
          <p>块像素化：降采样后再硬放大。</p>
          <p>相近色合并：LAB 空间网格量化。</p>
          <p>16 色：中位切分调色板 + 可选 Floyd–Steinberg 抖动。</p>
          <p>高级 OpenCV 网格像素化可后续再接。</p>
        </div>
      </div>
    </div>
  )
}
