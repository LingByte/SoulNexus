import { useCallback, useEffect, useMemo, useState } from 'react'
import { Button, Card, Empty, Input, Select, Tooltip } from '@/components/UI'
import { Slider } from '@arco-design/web-react'
import {
  ArrowDownToLine,
  AppWindow,
  Delete,
  Download,
  FolderOpen,
  Image as ImageIcon,
  Plus,
  Save,
  Search,
  Settings2,
  Swap,
  Upload,
  X,
} from 'lucide-react'
import { addAssetFromFile, assetToBlob, deleteAsset, getImageDimensions, listAssets, updateAsset, type LocalAsset } from '@/lib/pixelcraft/localAssetStore'
import { loadProjectSnapshots, saveProjectSnapshot } from '@/lib/pixelcraft/assetProject'
import { convertImageBlob, FORMAT_OPTIONS, zipBlobs } from '@/lib/pixelcraft/imageExport'
import { useAssetThumbnails } from '@/hooks/useAssetThumbnails'

const functionCategories = ['全部', '角色类', '道具物品类', '场景环境类', 'UI交互类', '特效动作类', '地图瓦片类']
const materialCategories = ['全部', '硬质材质', '软质材质', '自然材质', '魔幻特殊材质', '复古写实材质', '卡通极简材质']

export default function AssetWorkspace() {
  const [assets, setAssets] = useState<LocalAsset[]>([])
  const [funcFilter, setFuncFilter] = useState('全部')
  const [matFilter, setMatFilter] = useState('全部')
  const [folderFilter, setFolderFilter] = useState('全部')
  const [searchText, setSearchText] = useState('')
  const [compressQuality, setCompressQuality] = useState(80)
  const [maxEdge, setMaxEdge] = useState(512)
  const [convertFormat, setConvertFormat] = useState('image/png')
  const [selectedIds, setSelectedIds] = useState<number[]>([])
  const [previewAsset, setPreviewAsset] = useState<LocalAsset | null>(null)
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)
  const [editMeta, setEditMeta] = useState({ funcType: '', matType: '', folder: '', name: '' })
  const [snapshots, setSnapshots] = useState(() => loadProjectSnapshots())
  const [loading, setLoading] = useState(false)
  const thumbMap = useAssetThumbnails(assets)

  const refresh = useCallback(async () => {
    const list = await listAssets()
    setAssets(list)
  }, [])

  useEffect(() => {
    void refresh()
  }, [refresh])

  useEffect(() => {
    if (!previewAsset) {
      setPreviewUrl(null)
      return undefined
    }
    setEditMeta({
      name: previewAsset.name,
      funcType: previewAsset.funcType,
      matType: previewAsset.matType,
      folder: previewAsset.folder || '默认',
    })
    const url = URL.createObjectURL(assetToBlob(previewAsset))
    setPreviewUrl(url)
    return () => URL.revokeObjectURL(url)
  }, [previewAsset])

  const folders = useMemo(() => ['全部', ...new Set(assets.map((a) => a.folder || '默认'))], [assets])
  const filtered = useMemo(() => assets.filter((a) => {
    if (funcFilter !== '全部' && a.funcType !== funcFilter) return false
    if (matFilter !== '全部' && a.matType !== matFilter) return false
    if (folderFilter !== '全部' && (a.folder || '默认') !== folderFilter) return false
    if (searchText && !a.name.includes(searchText)) return false
    return true
  }), [assets, funcFilter, matFilter, folderFilter, searchText])

  const selectedAssets = filtered.filter((a) => a.id != null && selectedIds.includes(a.id))

  const toggleSelect = (id: number) => {
    setSelectedIds((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]))
  }

  const handleImport = async (files: FileList | File[]) => {
    const list = Array.from(files ?? []).filter(Boolean)
    if (!list.length) return
    setLoading(true)
    try {
      for (const file of list) {
        let width: number | null = null
        let height: number | null = null
        if (file.type.startsWith('image/')) {
          const dim = await getImageDimensions(file)
          width = dim.width
          height = dim.height
        }
        await addAssetFromFile(file, {
          funcType: funcFilter !== '全部' ? funcFilter : '道具物品类',
          matType: matFilter !== '全部' ? matFilter : '卡通极简材质',
          folder: folderFilter !== '全部' ? folderFilter : '默认',
          width,
          height,
        })
      }
      await refresh()
    } finally {
      setLoading(false)
    }
  }

  const handleBatchDownload = async () => {
    const targets = selectedAssets.length ? selectedAssets : filtered
    if (!targets.length) return
    setLoading(true)
    try {
      await zipBlobs(targets.map((a) => ({ name: a.name, blob: assetToBlob(a) })), 'assets_batch.zip')
    } finally {
      setLoading(false)
    }
  }

  const handleBatchConvert = async () => {
    const targets = selectedAssets.length ? selectedAssets : filtered
    if (!targets.length) return
    setLoading(true)
    try {
      const fmt = FORMAT_OPTIONS.find((f) => f.value === convertFormat) ?? FORMAT_OPTIONS[0]
      const entries: Array<{ name: string; blob: Blob }> = []
      for (const asset of targets) {
        if (!asset.mimeType?.startsWith('image/')) continue
        const blob = await convertImageBlob(assetToBlob(asset), {
          format: fmt.value,
          quality: compressQuality / 100,
          maxEdge,
        })
        const base = asset.name.replace(/\.[^.]+$/, '')
        entries.push({ name: `${base}.${fmt.ext}`, blob })
      }
      if (entries.length) await zipBlobs(entries, `converted_${fmt.ext}.zip`)
    } finally {
      setLoading(false)
    }
  }

  const handleUiPack = async () => {
    const uiAssets = (selectedAssets.length ? selectedAssets : filtered).filter((a) => a.funcType === 'UI交互类' || selectedAssets.length)
    if (!uiAssets.length) return
    setLoading(true)
    try {
      await zipBlobs(uiAssets.map((a) => ({ name: `ui/${a.name}`, blob: assetToBlob(a) })), 'ui_pack.zip')
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (id: number) => {
    await deleteAsset(id)
    setSelectedIds((prev) => prev.filter((x) => x !== id))
    await refresh()
  }

  const handleSavePreviewMeta = async () => {
    if (!previewAsset?.id) return
    await updateAsset(previewAsset.id, {
      name: editMeta.name,
      funcType: editMeta.funcType,
      matType: editMeta.matType,
      folder: editMeta.folder,
    })
    await refresh()
    setPreviewAsset((prev) => (prev ? { ...prev, ...editMeta } : prev))
  }

  return (
    <div className="space-y-5">
      <header className="atelier-hero atelier-enter rounded-2xl border border-border/60 bg-card/80 p-4 shadow-sm">
        <div className="atelier-title-row flex items-center gap-2">
          <AppWindow size={18} />
          <h1 className="text-xl font-semibold text-foreground">素材仓库</h1>
        </div>
        <p className="atelier-subtitle mt-1 text-sm text-muted-foreground">本地 IndexedDB 存储，支持导入、分类、格式转换、压缩与工程快照。</p>
      </header>

      <div className="grid gap-4 xl:grid-cols-[320px_minmax(0,1fr)_320px]">
        <aside className="space-y-4 rounded-2xl border border-border/60 bg-card/80 p-4 shadow-sm">
          <div>
            <h3 className="text-sm font-semibold text-foreground">筛选与导入</h3>
          </div>
          <label className="grid cursor-pointer gap-2 rounded-2xl border border-dashed border-border/70 bg-background/70 p-4 text-center">
            <Upload className="mx-auto size-6 text-primary" />
            <span className="text-sm font-medium text-foreground">拖拽或点击导入素材库</span>
            <span className="text-xs text-muted-foreground">支持图片、视频、音频、GIF、ZIP</span>
            <input hidden multiple type="file" onChange={(e) => e.target.files && void handleImport(e.target.files)} accept="image/*,video/*,audio/*,.gif,.zip,.json" />
          </label>
          <Input.Search placeholder="搜索素材名称..." value={searchText} onChange={(e) => setSearchText(e.target.value)} allowClear />
          <Select value={folderFilter} onValueChange={setFolderFilter} options={folders.map((f) => ({ label: f, value: f }))} />
          <div className="space-y-2">
            <div className="text-xs text-muted-foreground">功能分类</div>
            <div className="flex flex-wrap gap-2">
              {functionCategories.map((c) => (
                <button key={c} type="button" className={`rounded-full border px-3 py-1 text-xs ${funcFilter === c ? 'border-primary bg-primary/10 text-primary' : 'border-border/60 bg-background/70 text-foreground'}`} onClick={() => setFuncFilter(c)}>{c}</button>
              ))}
            </div>
          </div>
          <div className="space-y-2">
            <div className="text-xs text-muted-foreground">材质分类</div>
            <div className="flex flex-wrap gap-2">
              {materialCategories.map((c) => (
                <button key={c} type="button" className={`rounded-full border px-3 py-1 text-xs ${matFilter === c ? 'border-primary bg-primary/10 text-primary' : 'border-border/60 bg-background/70 text-foreground'}`} onClick={() => setMatFilter(c)}>{c}</button>
              ))}
            </div>
          </div>
        </aside>

        <main className="space-y-4 rounded-2xl border border-border/60 bg-card/80 p-4 shadow-sm">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <span className="text-sm text-muted-foreground">共 {filtered.length} 个素材 · 已选 {selectedIds.length}</span>
            <div className="flex flex-wrap gap-2">
              <Button size="sm" variant="outline" loading={loading} onClick={() => void handleBatchDownload()}>批量下载</Button>
              <Select value={convertFormat} onValueChange={setConvertFormat} options={FORMAT_OPTIONS.map((f) => ({ label: f.label, value: f.value }))} />
              <Button size="sm" variant="outline" loading={loading} onClick={() => void handleBatchConvert()}>格式转换 ZIP</Button>
              <Button size="sm" variant="outline" loading={loading} onClick={() => void handleUiPack()}>UI 打包</Button>
              <Button size="sm" variant="outline" onClick={() => { saveProjectSnapshot({ name: `工程_${Date.now()}`, assets: filtered.map(({ data, ...rest }) => rest), note: '元数据快照' }); setSnapshots(loadProjectSnapshots()) }}>工程备份</Button>
            </div>
          </div>

          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {filtered.length === 0 ? (
              <div className="sm:col-span-2 xl:col-span-3 rounded-2xl border border-dashed border-border/70 bg-background/70 p-8 text-center">
                <Empty title="仓库为空" description="从左侧导入 PNG / GIF / 音频等文件。" />
              </div>
            ) : (
              filtered.map((asset) => (
                <button key={asset.id} type="button" className={`rounded-2xl border p-3 text-left ${asset.id != null && selectedIds.includes(asset.id) ? 'border-primary/40 bg-primary/10' : 'border-border/60 bg-background/70'}`} onClick={() => { setPreviewAsset(asset); asset.id != null && toggleSelect(asset.id) }}>
                  <div className="flex aspect-square items-center justify-center overflow-hidden rounded-xl border border-border/50 bg-background">
                    {asset.id != null && thumbMap[asset.id] ? <img src={thumbMap[asset.id]} alt="" className="h-full w-full object-cover" /> : <ImageIcon className="size-8 text-muted-foreground" />}
                  </div>
                  <div className="mt-3 space-y-1">
                    <p className="text-sm font-medium text-foreground line-clamp-1">{asset.name}</p>
                    <p className="text-xs text-muted-foreground">{asset.funcType} · {asset.folder}</p>
                    <p className="text-xs text-muted-foreground">{asset.width && asset.height ? `${asset.width}×${asset.height}` : `${(asset.sizeBytes / 1024).toFixed(1)} KB`}</p>
                  </div>
                </button>
              ))
            )}
          </div>

          {snapshots.length > 0 && (
            <div className="flex flex-wrap gap-2 rounded-2xl border border-border/60 bg-background/70 p-3 text-xs text-muted-foreground">
              <span>最近工程备份：</span>
              {snapshots.slice(0, 5).map((s) => <span key={s.id} className="rounded-full border border-border/60 px-2 py-1 text-foreground">{s.name} · {s.assetCount} 项</span>)}
            </div>
          )}
        </main>

        <aside className="space-y-4 rounded-2xl border border-border/60 bg-background/60 p-4">
          <Card className="border border-border/60 bg-card/80 p-4 shadow-sm">
            <h4 className="text-sm font-semibold text-foreground">批量参数</h4>
            <div className="mt-3 space-y-3">
              <div>
                <div className="mb-2 flex items-center justify-between text-xs text-muted-foreground"><span>压缩质量</span><span>{compressQuality}%</span></div>
                <Slider min={40} max={100} value={compressQuality} onChange={setCompressQuality} />
              </div>
              <div>
                <div className="mb-2 flex items-center justify-between text-xs text-muted-foreground"><span>最长边</span><span>{maxEdge}px</span></div>
                <Slider min={64} max={2048} step={64} value={maxEdge} onChange={setMaxEdge} />
              </div>
            </div>
          </Card>

          <Card className="border border-border/60 bg-card/80 p-4 shadow-sm">
            <h4 className="text-sm font-semibold text-foreground">预览信息</h4>
            <p className="mt-2 text-sm text-muted-foreground">点击素材查看预览并编辑元数据。</p>
          </Card>
        </aside>
      </div>

      {previewAsset && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="w-full max-w-2xl rounded-2xl border border-border/60 bg-background p-4 shadow-xl">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-semibold text-foreground">{previewAsset.name}</h3>
              <button type="button" className="rounded-full p-2 hover:bg-muted" onClick={() => setPreviewAsset(null)}>
                <X size={18} />
              </button>
            </div>
            <div className="mt-4 grid gap-4 md:grid-cols-[minmax(0,1fr)_280px]">
              <div className="rounded-2xl border border-border/60 bg-card/80 p-3">
                {previewUrl && previewAsset.mimeType?.startsWith('image/') ? <img src={previewUrl} alt="" className="max-h-[420px] w-full rounded-xl object-contain" /> : <Empty title="预览" description="当前素材暂无图像预览。" />}
              </div>
              <div className="space-y-3 rounded-2xl border border-border/60 bg-card/80 p-3">
                <div>
                  <div className="mb-1 text-xs text-muted-foreground">文件名</div>
                  <Input value={editMeta.name} onChange={(e) => setEditMeta((m) => ({ ...m, name: e.target.value }))} />
                </div>
                <div>
                  <div className="mb-1 text-xs text-muted-foreground">功能分类</div>
                  <Select value={editMeta.funcType} onValueChange={(v) => setEditMeta((m) => ({ ...m, funcType: v }))} options={functionCategories.filter((c) => c !== '全部').map((c) => ({ label: c, value: c }))} />
                </div>
                <div>
                  <div className="mb-1 text-xs text-muted-foreground">材质分类</div>
                  <Select value={editMeta.matType} onValueChange={(v) => setEditMeta((m) => ({ ...m, matType: v }))} options={materialCategories.filter((c) => c !== '全部').map((c) => ({ label: c, value: c }))} />
                </div>
                <div>
                  <div className="mb-1 text-xs text-muted-foreground">文件夹</div>
                  <Input value={editMeta.folder} onChange={(e) => setEditMeta((m) => ({ ...m, folder: e.target.value }))} />
                </div>
                <div className="flex flex-wrap gap-2 pt-2">
                  <Button variant="outline" onClick={() => setPreviewAsset(null)}>关闭</Button>
                  <Button onClick={() => void handleSavePreviewMeta()}>保存元数据</Button>
                  <Button variant="outline" onClick={() => { if (!previewAsset) return; void handleDelete(previewAsset.id) }}>删除</Button>
                  <Button variant="outline" onClick={() => { if (!previewAsset) return; const blob = assetToBlob(previewAsset); const url = URL.createObjectURL(blob); const a = document.createElement('a'); a.href = url; a.download = previewAsset.name; a.click(); URL.revokeObjectURL(url) }}>下载原文件</Button>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
