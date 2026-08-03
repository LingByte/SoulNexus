import { useEffect, useMemo, useState } from 'react'
import { Tabs } from '@arco-design/web-react'
import { Button, Empty, Input } from '@/components/ui'
import { Download, Film, Upload } from 'lucide-react'
import FrameAnimationPreview from '@/components/pixel/FrameAnimationPreview'
import { decodeGif, encodeGif, revokeGifFrames, type GifFrame } from '@/lib/pixel/gifCodec'
import { mergeSpritesheet } from '@/lib/pixel/spritesheet'
import { splitSpritesheet } from '@/lib/pixel/spritesheet'
import { triggerDownload, zipBlobs } from '@/lib/pixel/imageExport'
import { showAlert } from '@/utils/notification'

const TabPane = Tabs.TabPane

export default function GifWorkspace() {
  const [tab, setTab] = useState('gif2frames')
  const [gifFrames, setGifFrames] = useState<GifFrame[]>([])
  const [fps, setFps] = useState(10)
  const [rows, setRows] = useState(4)
  const [cols, setCols] = useState(4)
  const [padding, setPadding] = useState(0)
  const [loading, setLoading] = useState(false)
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)
  const [seqFiles, setSeqFiles] = useState<File[]>([])
  const [composeFiles, setComposeFiles] = useState<File[]>([])

  useEffect(
    () => () => {
      revokeGifFrames(gifFrames)
      if (previewUrl) URL.revokeObjectURL(previewUrl)
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  )

  const frameUrls = useMemo(() => gifFrames.map((f) => f.dataUrl), [gifFrames])

  const onGifUpload = async (file: File) => {
    setLoading(true)
    try {
      revokeGifFrames(gifFrames)
      const frames = await decodeGif(file)
      setGifFrames(frames)
      showAlert(`已拆出 ${frames.length} 帧`, 'success')
    } catch (e) {
      showAlert((e as Error).message || 'GIF 解析失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  const exportGifZip = async () => {
    if (!gifFrames.length) return
    await zipBlobs(
      gifFrames.map((f, i) => ({ name: `frame_${String(i + 1).padStart(3, '0')}.png`, blob: f.blob })),
      'gif_frames.zip',
    )
  }

  const onSeqToGif = async () => {
    if (!seqFiles.length) {
      showAlert('请先选择序列帧图片', 'warning')
      return
    }
    setLoading(true)
    try {
      if (previewUrl) URL.revokeObjectURL(previewUrl)
      const blob = await encodeGif(
        seqFiles.map((f) => ({ blob: f })),
        { fps, loop: 0 },
      )
      const url = URL.createObjectURL(blob)
      setPreviewUrl(url)
      triggerDownload(blob, 'animation.gif')
    } catch (e) {
      showAlert((e as Error).message || '合成 GIF 失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  const onComposeSheet = async () => {
    if (!composeFiles.length) {
      showAlert('请先选择图片', 'warning')
      return
    }
    setLoading(true)
    try {
      if (previewUrl) URL.revokeObjectURL(previewUrl)
      // 若只选一张图：按行列拆成多格再合成（方便「拆分单图」）
      let blobs: Blob[] = composeFiles
      if (composeFiles.length === 1) {
        const { tiles } = await splitSpritesheet(composeFiles[0], { rows, cols, padding })
        blobs = tiles.map((t) => t.blob)
      }
      const { blob, dataUrl } = await mergeSpritesheet(blobs, { rows, cols, padding })
      setPreviewUrl(dataUrl)
      triggerDownload(blob, 'spritesheet.png')
    } catch (e) {
      showAlert((e as Error).message || '合成失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-4">
      <Tabs activeTab={tab} onChange={setTab}>
        <TabPane key="gif2frames" title="GIF → 序列帧" />
        <TabPane key="frames2gif" title="序列帧 → GIF" />
        <TabPane key="images2single" title="多图合成单图" />
      </Tabs>

      <div className="grid gap-4 xl:grid-cols-[320px_1fr_300px]">
        <div className="space-y-3 rounded-xl border border-border bg-card p-4">
          {tab === 'gif2frames' ? (
            <>
              <label className="flex cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-border px-4 py-8 text-sm text-muted-foreground hover:bg-muted/40">
                <Upload size={18} />
                上传 GIF
                <input
                  hidden
                  type="file"
                  accept="image/gif"
                  onChange={(e) => e.target.files?.[0] && void onGifUpload(e.target.files[0])}
                />
              </label>
              <Button type="primary" disabled={!gifFrames.length || loading} icon={<Download size={14} />} onClick={() => void exportGifZip()}>
                导出 ZIP
              </Button>
            </>
          ) : null}

          {tab === 'frames2gif' ? (
            <>
              <label className="flex cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-border px-4 py-8 text-sm text-muted-foreground hover:bg-muted/40">
                <Upload size={18} />
                选择序列帧（多选）
                <input
                  hidden
                  multiple
                  type="file"
                  accept="image/*"
                  onChange={(e) => setSeqFiles(Array.from(e.target.files || []))}
                />
              </label>
              <p className="text-xs text-muted-foreground">已选 {seqFiles.length} 张</p>
              <div>
                <div className="mb-1 text-xs text-muted-foreground">FPS</div>
                <Input type="number" value={String(fps)} onChange={(v) => setFps(Math.max(1, Number(v) || 1))} />
              </div>
              <Button type="primary" loading={loading} icon={<Film size={14} />} onClick={() => void onSeqToGif()}>
                合成 GIF
              </Button>
            </>
          ) : null}

          {tab === 'images2single' ? (
            <>
              <label className="flex cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-border px-4 py-8 text-sm text-muted-foreground hover:bg-muted/40">
                <Upload size={18} />
                多图，或单图按行列拆分
                <input
                  hidden
                  multiple
                  type="file"
                  accept="image/*"
                  onChange={(e) => setComposeFiles(Array.from(e.target.files || []))}
                />
              </label>
              <p className="text-xs text-muted-foreground">已选 {composeFiles.length} 张</p>
              <div className="grid grid-cols-3 gap-2">
                <div>
                  <div className="mb-1 text-xs text-muted-foreground">行</div>
                  <Input type="number" value={String(rows)} onChange={(v) => setRows(Math.max(1, Number(v) || 1))} />
                </div>
                <div>
                  <div className="mb-1 text-xs text-muted-foreground">列</div>
                  <Input type="number" value={String(cols)} onChange={(v) => setCols(Math.max(1, Number(v) || 1))} />
                </div>
                <div>
                  <div className="mb-1 text-xs text-muted-foreground">间隙</div>
                  <Input type="number" value={String(padding)} onChange={(v) => setPadding(Math.max(0, Number(v) || 0))} />
                </div>
              </div>
              <Button type="primary" loading={loading} onClick={() => void onComposeSheet()}>
                合成 Sprite Sheet
              </Button>
            </>
          ) : null}
        </div>

        <div className="rounded-xl border border-border bg-card p-4">
          {tab === 'gif2frames' && gifFrames.length ? (
            <FrameAnimationPreview frames={frameUrls} defaultFps={Math.round(1000 / (gifFrames[0]?.delayMs || 100))} />
          ) : previewUrl ? (
            <div className="grid min-h-[280px] place-items-center">
              <img src={previewUrl} alt="preview" className="max-h-[480px] max-w-full object-contain" />
            </div>
          ) : (
            <Empty description="上传后在此预览" />
          )}
        </div>

        <div className="rounded-xl border border-border bg-card p-4 text-sm text-muted-foreground space-y-2">
          <p>全部处理在浏览器本地完成，不会上传服务器。</p>
          {tab === 'gif2frames' ? <p>支持处置方式（disposal）的 GIF 合成拆帧。</p> : null}
          {tab === 'images2single' ? <p>单图时按行×列切格后再拼回一张表；多图则按顺序铺满网格。</p> : null}
        </div>
      </div>
    </div>
  )
}
