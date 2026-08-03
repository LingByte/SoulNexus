import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Tabs } from '@arco-design/web-react'
import { Button, Empty, Input, Select } from '@/components/ui'
import { Upload, Paintbrush, Scissors } from 'lucide-react'
import { resizeNearest, resizeSmooth } from '@/lib/pixel/resizeNearest'
import { applyChromaKey, CHROMA_PRESETS } from '@/lib/pixel/matteOps'
import { detectAutoSplitFromBlob, unifySizeSheet, type UnifyPadH, type UnifyPadV } from '@/lib/pixel/unifySize'
import { splitSpritesheet, mergeSpritesheet, revokeSheetTiles } from '@/lib/pixel/spritesheet'
import { triggerDownload, zipBlobs } from '@/lib/pixel/imageExport'
import { showAlert } from '@/utils/notification'

const TabPane = Tabs.TabPane

export default function ProWorkspace() {
  const navigate = useNavigate()
  const [tab, setTab] = useState('scale')

  // scale
  const [scaleFile, setScaleFile] = useState<File | null>(null)
  const [scalePreview, setScalePreview] = useState<string | null>(null)
  const [tw, setTw] = useState(256)
  const [th, setTh] = useState(256)
  const [keepAspect, setKeepAspect] = useState(true)
  const [nearest, setNearest] = useState(true)
  const [matteOn, setMatteOn] = useState(false)

  // slice
  const [sliceFile, setSliceFile] = useState<File | null>(null)
  const [rows, setRows] = useState(4)
  const [cols, setCols] = useState(4)
  const [slicePreview, setSlicePreview] = useState<string | null>(null)

  // unify
  const [unifyFiles, setUnifyFiles] = useState<File[]>([])
  const [unifyCols, setUnifyCols] = useState(4)
  const [padH, setPadH] = useState<UnifyPadH>('center')
  const [padV, setPadV] = useState<UnifyPadV>('bottom')
  const [unifyResult, setUnifyResult] = useState<string | null>(null)

  const [loading, setLoading] = useState(false)

  useEffect(
    () => () => {
      if (scalePreview) URL.revokeObjectURL(scalePreview)
      if (slicePreview) URL.revokeObjectURL(slicePreview)
      if (unifyResult) URL.revokeObjectURL(unifyResult)
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  )

  const runScale = async () => {
    if (!scaleFile) return
    setLoading(true)
    try {
      let blob: Blob = scaleFile
      if (matteOn) blob = await applyChromaKey(blob, CHROMA_PRESETS.green, 40, 8)
      blob = nearest
        ? await resizeNearest(blob, tw, th, keepAspect)
        : await resizeSmooth(blob, tw, th, keepAspect)
      if (scalePreview) URL.revokeObjectURL(scalePreview)
      setScalePreview(URL.createObjectURL(blob))
      triggerDownload(blob, 'pro_scale.png')
      showAlert('缩放完成', 'success')
    } catch (e) {
      showAlert((e as Error).message || '失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  const runSlice = async () => {
    if (!sliceFile) return
    setLoading(true)
    try {
      const { tiles } = await splitSpritesheet(sliceFile, { rows, cols, padding: 0, prefix: 'slice' })
      await zipBlobs(
        tiles.map((t) => ({ name: t.name, blob: t.blob })),
        'pro_slices.zip',
      )
      const merged = await mergeSpritesheet(
        tiles.map((t) => t.blob),
        { rows, cols, padding: 2 },
      )
      if (slicePreview) URL.revokeObjectURL(slicePreview)
      setSlicePreview(merged.dataUrl)
      revokeSheetTiles(tiles)
      showAlert(`已切出 ${tiles.length} 帧`, 'success')
    } catch (e) {
      showAlert((e as Error).message || '失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  const autoDetect = async () => {
    if (!sliceFile) return
    try {
      const d = await detectAutoSplitFromBlob(sliceFile)
      setCols(d.cols)
      setRows(d.rows)
      showAlert(`检测到约 ${d.cols}×${d.rows}`, 'success')
    } catch (e) {
      showAlert((e as Error).message || '检测失败', 'error')
    }
  }

  const runUnify = async () => {
    if (!unifyFiles.length) return
    setLoading(true)
    try {
      const { blob } = await unifySizeSheet(unifyFiles, unifyCols, padH, padV)
      if (unifyResult) URL.revokeObjectURL(unifyResult)
      const url = URL.createObjectURL(blob)
      setUnifyResult(url)
      triggerDownload(blob, 'pro_unify.png')
      showAlert('统一尺寸完成', 'success')
    } catch (e) {
      showAlert((e as Error).message || '失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-4">
      <Tabs activeTab={tab} onChange={setTab}>
        <TabPane key="scale" title="自定义缩放" />
        <TabPane key="slice" title="自定义切片" />
        <TabPane key="unify" title="统一尺寸" />
        <TabPane key="single" title="单图调整 Pro" />
      </Tabs>

      <div className="grid gap-4 xl:grid-cols-[320px_1fr]">
        <div className="space-y-3 rounded-xl border border-border bg-card p-4">
          {tab === 'scale' ? (
            <>
              <label className="flex cursor-pointer flex-col items-center gap-2 rounded-lg border border-dashed border-border px-4 py-6 text-sm text-muted-foreground hover:bg-muted/40">
                <Upload size={18} /> 上传
                <input hidden type="file" accept="image/*" onChange={(e) => e.target.files?.[0] && setScaleFile(e.target.files[0])} />
              </label>
              <div className="grid grid-cols-2 gap-2">
                <Input addBefore="宽" type="number" value={String(tw)} onChange={(v) => setTw(Math.max(1, Number(v) || 1))} />
                <Input addBefore="高" type="number" value={String(th)} onChange={(v) => setTh(Math.max(1, Number(v) || 1))} />
              </div>
              <div className="flex flex-wrap gap-2">
                <Button size="sm" variant={keepAspect ? 'default' : 'outline'} onClick={() => setKeepAspect(true)}>比例</Button>
                <Button size="sm" variant={!keepAspect ? 'default' : 'outline'} onClick={() => setKeepAspect(false)}>拉伸</Button>
                <Button size="sm" variant={nearest ? 'default' : 'outline'} onClick={() => setNearest(true)}>硬缩放</Button>
                <Button size="sm" variant={!nearest ? 'default' : 'outline'} onClick={() => setNearest(false)}>平滑</Button>
                <Button size="sm" variant={matteOn ? 'default' : 'outline'} onClick={() => setMatteOn((v) => !v)}>绿幕</Button>
              </div>
              <Button type="primary" block loading={loading} disabled={!scaleFile} onClick={() => void runScale()}>缩放并下载</Button>
            </>
          ) : null}

          {tab === 'slice' ? (
            <>
              <label className="flex cursor-pointer flex-col items-center gap-2 rounded-lg border border-dashed border-border px-4 py-6 text-sm text-muted-foreground hover:bg-muted/40">
                <Upload size={18} /> 上传 Sheet
                <input hidden type="file" accept="image/*" onChange={(e) => e.target.files?.[0] && setSliceFile(e.target.files[0])} />
              </label>
              <div className="grid grid-cols-2 gap-2">
                <Input addBefore="行" type="number" value={String(rows)} onChange={(v) => setRows(Math.max(1, Number(v) || 1))} />
                <Input addBefore="列" type="number" value={String(cols)} onChange={(v) => setCols(Math.max(1, Number(v) || 1))} />
              </div>
              <Button variant="outline" block icon={<Scissors size={14} />} disabled={!sliceFile} onClick={() => void autoDetect()}>
                透明缝自动检测
              </Button>
              <Button type="primary" block loading={loading} disabled={!sliceFile} onClick={() => void runSlice()}>
                切片并导出 ZIP
              </Button>
            </>
          ) : null}

          {tab === 'unify' ? (
            <>
              <label className="flex cursor-pointer flex-col items-center gap-2 rounded-lg border border-dashed border-border px-4 py-6 text-sm text-muted-foreground hover:bg-muted/40">
                <Upload size={18} /> 多图统一尺寸
                <input
                  hidden
                  multiple
                  type="file"
                  accept="image/*"
                  onChange={(e) => setUnifyFiles(Array.from(e.target.files || []))}
                />
              </label>
              <p className="text-xs text-muted-foreground">已选 {unifyFiles.length} 张</p>
              <Input addBefore="列数" type="number" value={String(unifyCols)} onChange={(v) => setUnifyCols(Math.max(1, Number(v) || 1))} />
              <Select
                value={padH}
                options={[
                  { value: 'left', label: '水平靠左' },
                  { value: 'center', label: '水平居中' },
                  { value: 'right', label: '水平靠右' },
                ]}
                onChange={(v) => setPadH(v as UnifyPadH)}
              />
              <Select
                value={padV}
                options={[
                  { value: 'top', label: '垂直靠上' },
                  { value: 'center', label: '垂直居中' },
                  { value: 'bottom', label: '垂直靠下' },
                ]}
                onChange={(v) => setPadV(v as UnifyPadV)}
              />
              <Button type="primary" block loading={loading} disabled={!unifyFiles.length} onClick={() => void runUnify()}>
                统一并导出
              </Button>
            </>
          ) : null}

          {tab === 'single' ? (
            <div className="space-y-3 text-sm text-muted-foreground">
              <p>单图 Pro 调整复用现有像素管线：裁切 / 抠图 / 缩放 / 描边 / 精细画笔。</p>
              <Button block type="primary" onClick={() => navigate('/pixel/process')}>打开像素处理</Button>
              <Button block variant="outline" icon={<Paintbrush size={14} />} onClick={() => navigate('/pixel/fine')}>
                打开精细编辑
              </Button>
              <Button block variant="outline" onClick={() => navigate('/pixel/gemini-watermark')}>
                Gemini 去水印
              </Button>
            </div>
          ) : null}
        </div>

        <div className="rounded-xl border border-border bg-card p-4">
          {tab === 'scale' && scalePreview ? (
            <img src={scalePreview} alt="" className="mx-auto max-h-[520px] object-contain" style={{ imageRendering: nearest ? 'pixelated' : 'auto' }} />
          ) : tab === 'slice' && slicePreview ? (
            <img src={slicePreview} alt="" className="mx-auto max-h-[520px] object-contain" style={{ imageRendering: 'pixelated' }} />
          ) : tab === 'unify' && unifyResult ? (
            <img src={unifyResult} alt="" className="mx-auto max-h-[520px] object-contain" style={{ imageRendering: 'pixelated' }} />
          ) : (
            <Empty description="参数在左侧，结果在此预览" />
          )}
        </div>
      </div>
    </div>
  )
}
