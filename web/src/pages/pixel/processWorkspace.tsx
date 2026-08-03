import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Slider } from '@arco-design/web-react'
import { Button, Empty, Input, Select } from '@/components/ui'
import { Download, Paintbrush, Upload } from 'lucide-react'
import { cropImageBlob } from '@/lib/pixel/cropImage'
import { applyInnerStroke } from '@/lib/pixel/innerStroke'
import { applyChromaKey, CHROMA_PRESETS } from '@/lib/pixel/matteOps'
import { resizeNearest, resizeSmooth } from '@/lib/pixel/resizeNearest'
import { setFineHandoffDataUrl } from '@/lib/pixel/fineHandoff'
import { blobToImage } from '@/lib/pixel/canvasUtils'
import { triggerDownload } from '@/lib/pixel/imageExport'
import { showAlert } from '@/utils/notification'

export default function ProcessWorkspace() {
  const navigate = useNavigate()
  const [source, setSource] = useState<Blob | null>(null)
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const [cropL, setCropL] = useState(0)
  const [cropT, setCropT] = useState(0)
  const [cropR, setCropR] = useState(0)
  const [cropB, setCropB] = useState(0)

  const [matteOn, setMatteOn] = useState(false)
  const [mattePreset, setMattePreset] = useState<'green' | 'blue' | 'custom'>('green')
  const [customRgb, setCustomRgb] = useState({ r: 0, g: 177, b: 64 })
  const [tolerance, setTolerance] = useState(40)
  const [feather, setFeather] = useState(8)

  const [targetW, setTargetW] = useState(0)
  const [targetH, setTargetH] = useState(0)
  const [keepAspect, setKeepAspect] = useState(true)
  const [nearest, setNearest] = useState(true)

  const [strokeW, setStrokeW] = useState(0)
  const [strokeColor, setStrokeColor] = useState('#000000')

  useEffect(
    () => () => {
      if (previewUrl) URL.revokeObjectURL(previewUrl)
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  )

  const onUpload = async (file: File) => {
    setSource(file)
    if (previewUrl) URL.revokeObjectURL(previewUrl)
    setPreviewUrl(URL.createObjectURL(file))
    try {
      const img = await blobToImage(file)
      setTargetW(img.naturalWidth)
      setTargetH(img.naturalHeight)
    } catch {
      /* ignore */
    }
  }

  const runPipeline = async () => {
    if (!source) return
    setLoading(true)
    try {
      let blob: Blob = source
      if (cropL || cropT || cropR || cropB) {
        blob = await cropImageBlob(blob, { left: cropL, top: cropT, right: cropR, bottom: cropB })
      }
      if (matteOn) {
        const bg = mattePreset === 'custom' ? customRgb : CHROMA_PRESETS[mattePreset]
        blob = await applyChromaKey(blob, bg, tolerance, feather)
      }
      if (targetW > 0 && targetH > 0) {
        blob = nearest
          ? await resizeNearest(blob, targetW, targetH, keepAspect)
          : await resizeSmooth(blob, targetW, targetH, keepAspect)
      }
      if (strokeW > 0) {
        blob = await applyInnerStroke(blob, strokeW, strokeColor)
      }
      if (previewUrl) URL.revokeObjectURL(previewUrl)
      setPreviewUrl(URL.createObjectURL(blob))
      setSource(blob)
      showAlert('处理完成', 'success')
    } catch (e) {
      showAlert((e as Error).message || '处理失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  const download = () => {
    if (!source) return
    triggerDownload(source, 'processed.png')
  }

  const sendToFine = async () => {
    if (!source) return
    const dataUrl = await new Promise<string>((resolve, reject) => {
      const reader = new FileReader()
      reader.onload = () => resolve(String(reader.result))
      reader.onerror = () => reject(new Error('读取图片失败'))
      reader.readAsDataURL(source)
    })
    setFineHandoffDataUrl(dataUrl)
    navigate('/pixel/fine')
  }

  return (
    <div className="grid gap-4 xl:grid-cols-[340px_1fr_280px]">
      <div className="space-y-3 rounded-xl border border-border bg-card p-4 max-h-[80vh] overflow-auto">
        <label className="flex cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-border px-4 py-8 text-sm text-muted-foreground hover:bg-muted/40">
          <Upload size={18} />
          上传图片
          <input hidden type="file" accept="image/*" onChange={(e) => e.target.files?.[0] && void onUpload(e.target.files[0])} />
        </label>

        <div className="text-sm font-medium">裁切（边缘像素）</div>
        <div className="grid grid-cols-2 gap-2">
          <Input addBefore="左" type="number" value={String(cropL)} onChange={(v) => setCropL(Math.max(0, Number(v) || 0))} />
          <Input addBefore="上" type="number" value={String(cropT)} onChange={(v) => setCropT(Math.max(0, Number(v) || 0))} />
          <Input addBefore="右" type="number" value={String(cropR)} onChange={(v) => setCropR(Math.max(0, Number(v) || 0))} />
          <Input addBefore="下" type="number" value={String(cropB)} onChange={(v) => setCropB(Math.max(0, Number(v) || 0))} />
        </div>

        <div className="flex items-center justify-between text-sm font-medium">
          <span>绿 / 蓝幕抠图</span>
          <Button size="mini" variant={matteOn ? 'default' : 'outline'} onClick={() => setMatteOn((v) => !v)}>
            {matteOn ? '已开启' : '关闭'}
          </Button>
        </div>
        {matteOn ? (
          <div className="space-y-2">
            <Select
              value={mattePreset}
              options={[
                { value: 'green', label: '绿色幕' },
                { value: 'blue', label: '蓝色幕' },
                { value: 'custom', label: '自定义 RGB' },
              ]}
              onChange={(v) => setMattePreset(v as 'green' | 'blue' | 'custom')}
            />
            {mattePreset === 'custom' ? (
              <div className="grid grid-cols-3 gap-1">
                <Input addBefore="R" type="number" value={String(customRgb.r)} onChange={(v) => setCustomRgb((c) => ({ ...c, r: Number(v) || 0 }))} />
                <Input addBefore="G" type="number" value={String(customRgb.g)} onChange={(v) => setCustomRgb((c) => ({ ...c, g: Number(v) || 0 }))} />
                <Input addBefore="B" type="number" value={String(customRgb.b)} onChange={(v) => setCustomRgb((c) => ({ ...c, b: Number(v) || 0 }))} />
              </div>
            ) : null}
            <div>
              <div className="mb-1 flex justify-between text-xs text-muted-foreground">
                <span>容差</span>
                <span>{tolerance}</span>
              </div>
              <Slider min={0} max={180} value={tolerance} onChange={(v) => setTolerance(Number(v))} />
            </div>
            <div>
              <div className="mb-1 flex justify-between text-xs text-muted-foreground">
                <span>羽化</span>
                <span>{feather}</span>
              </div>
              <Slider min={0} max={80} value={feather} onChange={(v) => setFeather(Number(v))} />
            </div>
          </div>
        ) : null}

        <div className="text-sm font-medium">缩放</div>
        <div className="grid grid-cols-2 gap-2">
          <Input addBefore="宽" type="number" value={String(targetW)} onChange={(v) => setTargetW(Math.max(0, Number(v) || 0))} />
          <Input addBefore="高" type="number" value={String(targetH)} onChange={(v) => setTargetH(Math.max(0, Number(v) || 0))} />
        </div>
        <div className="flex gap-2">
          <Button size="sm" variant={keepAspect ? 'default' : 'outline'} onClick={() => setKeepAspect(true)}>保持比例</Button>
          <Button size="sm" variant={!keepAspect ? 'default' : 'outline'} onClick={() => setKeepAspect(false)}>拉伸</Button>
          <Button size="sm" variant={nearest ? 'default' : 'outline'} onClick={() => setNearest(true)}>硬缩放</Button>
          <Button size="sm" variant={!nearest ? 'default' : 'outline'} onClick={() => setNearest(false)}>平滑</Button>
        </div>

        <div className="text-sm font-medium">内描边</div>
        <div className="grid grid-cols-[1fr_100px] gap-2">
          <Input addBefore="宽度" type="number" value={String(strokeW)} onChange={(v) => setStrokeW(Math.max(0, Number(v) || 0))} />
          <input
            type="color"
            value={strokeColor}
            onChange={(e) => setStrokeColor(e.target.value)}
            className="h-9 w-full cursor-pointer rounded border border-border bg-transparent"
          />
        </div>

        <Button type="primary" block loading={loading} disabled={!source} onClick={() => void runPipeline()}>
          应用处理
        </Button>
        <Button block variant="outline" disabled={!source} icon={<Download size={14} />} onClick={download}>
          下载 PNG
        </Button>
        <Button block variant="outline" disabled={!source} icon={<Paintbrush size={14} />} onClick={() => void sendToFine()}>
          送到精细编辑
        </Button>
      </div>

      <div className="rounded-xl border border-border bg-card p-4">
        {previewUrl ? (
          <div className="grid min-h-[360px] place-items-center bg-[repeating-conic-gradient(#e5e5e5_0%_25%,#fff_0%_50%)] bg-[length:16px_16px] rounded-lg">
            <img src={previewUrl} alt="preview" className="max-h-[560px] max-w-full object-contain" />
          </div>
        ) : (
          <Empty description="上传图片后预览" />
        )}
      </div>

      <div className="rounded-xl border border-border bg-card p-4 text-sm text-muted-foreground space-y-2">
        <p>处理顺序：裁切 → 抠图 → 缩放 → 内描边。</p>
        <p>硬缩放使用最近邻采样，适合像素风素材。</p>
        <p>可一键送到精细编辑继续画笔修图。</p>
      </div>
    </div>
  )
}
