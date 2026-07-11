import { useEffect, useMemo, useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { ArrowLeft, Code2, Files, PanelRight, Play, Save, Settings2, Sparkles } from 'lucide-react'
import Button from '@/components/UI/Button'
import FileExplorer from './FileExplorer'
import EditorTabs from './EditorTabs'
import PreviewPanel from './PreviewPanel'
import EmbedPanel from './EmbedPanel'
import ManifestAiPanel from './ManifestAiPanel'
import { PROJECT_FILES, type PetProjectV1 } from '@/pages/pet-market/types'
import { petNameFromProject } from '@/pages/pet-market/projectUtils'
import {
  defaultFileContent,
  deleteProjectFile,
  folderMarkerPath,
  isProtectedProjectFile,
  moveProjectFile,
  normalizeProjectPath,
  validateProjectPath,
} from '@/pages/pet-market/fileTreeUtils'
import { showAlert } from '@/utils/notification'

interface PetStudioLayoutProps {
  project: PetProjectV1
  templateName: string
  isNew: boolean
  saving: boolean
  savedRevision?: number
  storageHint?: string | null
  starterTemplateLabel?: string
  missingModelAssets?: boolean
  importingModelAssets?: boolean
  onImportModelAssets?: () => void
  jsSourceId?: string | null
  templateId?: string | null
  onProjectChange: (project: PetProjectV1) => void
  onTemplateNameChange: (name: string) => void
  onSave: () => void
  onQuickSave?: () => void
  onBack: () => void
}

export default function PetStudioLayout({
  project,
  templateName,
  isNew,
  saving,
  savedRevision = 0,
  storageHint = null,
  starterTemplateLabel,
  missingModelAssets = false,
  importingModelAssets = false,
  onImportModelAssets,
  jsSourceId = null,
  templateId = null,
  onProjectChange,
  onTemplateNameChange,
  onSave,
  onQuickSave,
  onBack,
}: PetStudioLayoutProps) {
  const fileList = useMemo(() => Object.keys(project.files), [project.files])
  const [activeFile, setActiveFile] = useState(project.entry || PROJECT_FILES.entry)
  const [openFiles, setOpenFiles] = useState<string[]>([activeFile])
  const [dirtyFiles, setDirtyFiles] = useState<Set<string>>(new Set())
  const [showPreview, setShowPreview] = useState(true)
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [showEmbed, setShowEmbed] = useState(false)
  const [showAiPanel, setShowAiPanel] = useState(false)

  const assetFiles = useMemo(
    () =>
      Object.keys(project.files).filter(
        (p) => /\.(png|jpe?g|webp|gif|moc3|model3\.json)$/i.test(p) && !p.endsWith('.gitkeep'),
      ),
    [project.files],
  )

  const manifestKind = useMemo(() => {
    try {
      const m = JSON.parse(project.files[PROJECT_FILES.manifest] || '{}') as { kind?: string }
      return m.kind
    } catch {
      return undefined
    }
  }, [project.files])

  useEffect(() => {
    if (savedRevision > 0) setDirtyFiles(new Set())
  }, [savedRevision])

  const displayName = petNameFromProject(project, templateName) || (isNew ? '未命名桌宠' : templateName)

  const updateFile = (path: string, value: string) => {
    const next: PetProjectV1 = {
      ...project,
      files: { ...project.files, [path]: value },
    }
    onProjectChange(next)
    setDirtyFiles((prev) => new Set(prev).add(path))
    if (path === PROJECT_FILES.manifest) {
      try {
        const m = JSON.parse(value) as { name?: string }
        if (m.name?.trim()) onTemplateNameChange(m.name.trim())
      } catch { /* ignore invalid json while typing */ }
    }
  }

  const selectFile = (path: string) => {
    setActiveFile(path)
    if (!openFiles.includes(path)) setOpenFiles((f) => [...f, path])
  }

  const closeFile = (path: string) => {
    const next = openFiles.filter((f) => f !== path)
    setOpenFiles(next.length ? next : [activeFile])
    if (path === activeFile && next.length) setActiveFile(next[next.length - 1])
  }

  const createFile = (rawPath: string) => {
    const path = normalizeProjectPath(rawPath)
    const err = validateProjectPath(path, false)
    if (err) {
      showAlert(err, 'warning')
      return
    }
    if (project.files[path]) {
      showAlert('文件已存在', 'warning')
      selectFile(path)
      return
    }
    updateFile(path, defaultFileContent(path))
    selectFile(path)
  }

  const createFolder = (rawPath: string) => {
    const folder = normalizeProjectPath(rawPath).replace(/\/$/, '')
    const err = validateProjectPath(folder, true)
    if (err) {
      showAlert(err, 'warning')
      return
    }
    const marker = folderMarkerPath(folder)
    if (project.files[marker]) {
      showAlert('文件夹已存在', 'warning')
      return
    }
    updateFile(marker, '')
  }

  const importFiles = (incoming: Record<string, string>) => {
    const nextFiles = { ...project.files, ...incoming }
    onProjectChange({ ...project, files: nextFiles })
    for (const path of Object.keys(incoming)) {
      setDirtyFiles((prev) => new Set(prev).add(path))
      if (!openFiles.includes(path)) setOpenFiles((f) => [...f, path])
    }
    const first = Object.keys(incoming).find((p) => p.endsWith('.model3.json')) ?? Object.keys(incoming)[0]
    if (first) selectFile(first)
  }

  const deleteFile = (path: string) => {
    if (isProtectedProjectFile(path)) {
      showAlert('manifest.json / pet.js / style.css / README.md 不可删除', 'warning')
      return
    }
    const nextFiles = deleteProjectFile(project.files, path)
    if (!nextFiles) {
      showAlert('无法删除该文件', 'warning')
      return
    }
    onProjectChange({ ...project, files: nextFiles })
    closeFile(path)
    setDirtyFiles((prev) => {
      const next = new Set(prev)
      next.delete(path)
      return next
    })
    if (activeFile === path) {
      const fallback = Object.keys(nextFiles).find((p) => !p.endsWith('.gitkeep')) ?? PROJECT_FILES.entry
      selectFile(fallback)
    }
  }

  const moveFile = (from: string, toFolder: string) => {
    const result = moveProjectFile(project.files, from, toFolder)
    if (!result) {
      showAlert('无法移动到该目录（核心文件不可移动，或目标已存在）', 'warning')
      return
    }
    onProjectChange({ ...project, files: result.files })
    setDirtyFiles((prev) => {
      const next = new Set(prev)
      next.delete(from)
      next.add(result.newPath)
      return next
    })
    setOpenFiles((files) => files.map((f) => (f === from ? result.newPath : f)))
    if (activeFile === from) setActiveFile(result.newPath)
  }

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 's') {
        e.preventDefault()
        if (!saving) (onQuickSave ?? onSave)()
        return
      }
      const tag = (e.target as HTMLElement | null)?.tagName
      if (tag === 'INPUT' || tag === 'TEXTAREA' || (e.target as HTMLElement)?.isContentEditable) return
      if (e.key === 'Delete' || (e.key === 'Backspace' && (e.metaKey || e.ctrlKey))) {
        if (!isProtectedProjectFile(activeFile)) {
          e.preventDefault()
          deleteFile(activeFile)
        }
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [onSave, onQuickSave, saving, activeFile, project.files])

  return (
    <div className="h-screen w-full flex flex-col bg-[#1e1e1e] text-[#cccccc] overflow-hidden">
      {/* Title bar */}
      <header className="h-11 flex items-center justify-between px-3 bg-[#323233] border-b border-[#2b2b2b] shrink-0">
        <div className="flex items-center gap-3 min-w-0">
          <button type="button" onClick={onBack} className="p-1.5 rounded hover:bg-[#3c3c3c] text-[#cccccc]" title="返回市场">
            <ArrowLeft className="w-4 h-4" />
          </button>
          <div className="min-w-0">
            <div className="text-sm font-medium text-white truncate">{displayName}</div>
            <div className="text-[10px] text-[#858585]">
              Soul Pet Studio · {isNew ? '新建项目' : '编辑项目'}
              {starterTemplateLabel ? ` · ${starterTemplateLabel}` : ''}
            </div>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => setShowPreview((v) => !v)}
            className={`p-1.5 rounded ${showPreview ? 'bg-[#094771] text-white' : 'hover:bg-[#3c3c3c]'}`}
            title="切换预览"
          >
            <PanelRight className="w-4 h-4" />
          </button>
          <Button size="sm" variant="ghost" leftIcon={<Play className="w-3.5 h-3.5" />} onClick={() => setShowPreview(true)}>
            预览
          </Button>
          {jsSourceId && templateId && (
            <Button size="sm" variant="ghost" leftIcon={<Code2 className="w-3.5 h-3.5" />} onClick={() => setShowEmbed(true)} title="嵌入代码">
              嵌入
            </Button>
          )}
          {activeFile === PROJECT_FILES.manifest && (
            <Button
              size="sm"
              variant="ghost"
              leftIcon={<Sparkles className="w-3.5 h-3.5 text-amber-400" />}
              onClick={() => setShowAiPanel((v) => !v)}
              title="AI 辅助编辑 manifest"
            >
              AI
            </Button>
          )}
          <Button size="sm" variant="primary" leftIcon={<Save className="w-3.5 h-3.5" />} onClick={onSave} disabled={saving} title="保存到对象存储 (Ctrl+S 快速保存)">
            {saving ? '保存中…' : '保存'}
          </Button>
        </div>
      </header>

      {missingModelAssets && (
        <div className="flex items-center justify-between gap-3 px-4 py-2 bg-amber-950/80 border-b border-amber-800/60 text-amber-100 text-[12px] shrink-0">
          <span>
            缺少 <code className="text-amber-200">assets/sprites/</code> 帧图（idle.png、talk.png 等 PNG 精灵图）。
            可拖拽 PNG 到资源管理器导入，或先使用占位绘制预览。
          </span>
          {onImportModelAssets && (
            <Button size="sm" variant="outline" onClick={onImportModelAssets} disabled={importingModelAssets}>
              {importingModelAssets ? '导入中…' : '查看说明'}
            </Button>
          )}
        </div>
      )}

      <div className="flex flex-1 min-h-0">
        {/* Activity bar */}
        <aside className="w-12 bg-[#333333] flex flex-col items-center py-2 gap-1 shrink-0 border-r border-[#2b2b2b]">
          <button
            type="button"
            onClick={() => setSidebarOpen(true)}
            className={`p-2 rounded ${sidebarOpen ? 'text-white border-l-2 border-[#007fd4]' : 'text-[#858585] hover:text-white'}`}
            title="资源管理器"
          >
            <Files className="w-5 h-5" />
          </button>
          <button type="button" className="p-2 rounded text-[#858585] hover:text-white opacity-50 cursor-not-allowed" title="设置（即将推出）">
            <Settings2 className="w-5 h-5" />
          </button>
        </aside>

        {/* Sidebar */}
        {sidebarOpen && (
          <aside className="w-56 bg-[#252526] border-r border-[#2b2b2b] shrink-0 overflow-y-auto">
            <div className="px-3 py-2 text-[11px] font-semibold uppercase tracking-wide text-[#bbbbbb] border-b border-[#2b2b2b]">
              资源管理器
            </div>
            <FileExplorer
              files={fileList}
              activeFile={activeFile}
              onSelect={selectFile}
              onCreateFile={createFile}
              onCreateFolder={createFolder}
              onImportFiles={importFiles}
              onDeleteFile={deleteFile}
              onMoveFile={moveFile}
            />
          </aside>
        )}

        {/* Editor */}
        <main className="relative flex flex-1 min-w-0 min-h-0 overflow-hidden">
          <EditorTabs
            openFiles={openFiles}
            activeFile={activeFile}
            dirtyFiles={dirtyFiles}
            fileContents={project.files}
            onSelect={selectFile}
            onClose={closeFile}
            onChange={updateFile}
            onSave={onQuickSave ?? onSave}
          />
          {showAiPanel && activeFile === PROJECT_FILES.manifest && (
            <ManifestAiPanel
              manifest={project.files[PROJECT_FILES.manifest] || '{}'}
              kind={manifestKind}
              assetFiles={assetFiles}
              onApply={(manifest) => {
                updateFile(PROJECT_FILES.manifest, manifest)
              }}
              onClose={() => setShowAiPanel(false)}
            />
          )}
          <AnimatePresence initial={false}>
            {showPreview && (
              <motion.div
                key="pet-preview"
                initial={{ width: 0, opacity: 0 }}
                animate={{ width: 'min(420px, 40vw)', opacity: 1 }}
                exit={{ width: 0, opacity: 0 }}
                transition={{ duration: 0.22, ease: [0.4, 0, 0.2, 1] }}
                className="overflow-hidden shrink-0 flex flex-col h-full min-w-0"
              >
                <PreviewPanel
                  project={project}
                  templateId={templateId}
                  jsSourceId={jsSourceId}
                  missingModelAssets={missingModelAssets}
                  visible
                  onToggle={() => setShowPreview(false)}
                />
              </motion.div>
            )}
          </AnimatePresence>
        </main>
      </div>

      {/* Status bar */}
      <footer className="h-6 flex items-center justify-between px-3 bg-[#007acc] text-white text-[11px] shrink-0">
        <span>{activeFile}</span>
        <span className="truncate ml-4 text-right">
          {dirtyFiles.size
            ? `${dirtyFiles.size} 个未保存更改 · 编辑在浏览器内存，Ctrl+S 保存到对象存储`
            : storageHint ?? 'Ctrl+S 保存 · 代码持久化在对象存储，数据库仅存元数据指针'}
        </span>
      </footer>

      {showEmbed && jsSourceId && (
        <EmbedPanel
          jsSourceId={jsSourceId}
          templateName={displayName}
          onClose={() => setShowEmbed(false)}
        />
      )}
    </div>
  )
}
