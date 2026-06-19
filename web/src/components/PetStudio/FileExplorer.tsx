import {
  ChevronDown,
  ChevronRight,
  FileCode,
  FileJson,
  FilePlus,
  FileText,
  Folder,
  FolderPlus,
  Image,
  Palette,
  Trash2,
  Upload,
} from 'lucide-react'
import { useMemo, useRef, useState } from 'react'
import { PROJECT_FILES } from '@/pages/pet-market/types'
import {
  buildFileTree,
  collectFolderPaths,
  dirname,
  isProtectedProjectFile,
  normalizeProjectPath,
  validateProjectPath,
  type FileTreeNode,
} from '@/pages/pet-market/fileTreeUtils'
import {
  collectFilesFromDataTransfer,
  encodeDroppedFiles,
} from '@/pages/pet-market/projectAssetUtils'
import { showAlert } from '@/utils/notification'

const INTERNAL_DRAG_MIME = 'application/x-soul-pet-path'

interface FileExplorerProps {
  files: string[]
  activeFile: string
  onSelect: (path: string) => void
  onCreateFile: (path: string) => void
  onCreateFolder: (folderPath: string) => void
  onImportFiles?: (files: Record<string, string>) => void
  onDeleteFile?: (path: string) => void
  onMoveFile?: (from: string, toFolder: string) => void
}

function iconForFile(name: string) {
  if (name.endsWith('.json')) return FileJson
  if (/\.(png|jpe?g|webp|gif)$/i.test(name)) return Image
  if (name.endsWith('.css')) return Palette
  if (name.endsWith('.md')) return FileText
  return FileCode
}

function TreeNode({
  node,
  depth,
  activeFile,
  expanded,
  dropTarget,
  onToggle,
  onSelect,
  onDeleteFile,
  onMoveFile,
  onFolderDragOver,
  onFolderDragLeave,
  onFolderDrop,
}: {
  node: FileTreeNode
  depth: number
  activeFile: string
  expanded: Set<string>
  dropTarget: string | null
  onToggle: (path: string) => void
  onSelect: (path: string) => void
  onDeleteFile?: (path: string) => void
  onMoveFile?: (from: string, toFolder: string) => void
  onFolderDragOver: (folder: string, e: React.DragEvent) => void
  onFolderDragLeave: (folder: string, e: React.DragEvent) => void
  onFolderDrop: (folder: string, e: React.DragEvent) => void
}) {
  const pad = 12 + depth * 12

  if (node.kind === 'folder') {
    const open = expanded.has(node.path)
    const isDrop = dropTarget === node.path
    return (
      <li>
        <button
          type="button"
          onClick={() => onToggle(node.path)}
          onDragOver={(e) => onFolderDragOver(node.path, e)}
          onDragLeave={(e) => onFolderDragLeave(node.path, e)}
          onDrop={(e) => onFolderDrop(node.path, e)}
          className={`w-full flex items-center gap-1 pr-3 py-1 text-[13px] text-left text-[#cccccc] hover:bg-[#2a2d2e] ${
            isDrop ? 'bg-[#094771]/40 ring-1 ring-inset ring-[#007fd4]' : ''
          }`}
          style={{ paddingLeft: pad }}
        >
          {open ? <ChevronDown className="w-3 h-3 shrink-0 opacity-70" /> : <ChevronRight className="w-3 h-3 shrink-0 opacity-70" />}
          <Folder className="w-3.5 h-3.5 shrink-0 text-[#dcb67a]" />
          <span className="truncate">{node.name}</span>
        </button>
        {open && node.children.length > 0 && (
          <ul>
            {node.children.map((child) => (
              <TreeNode
                key={child.kind === 'folder' ? `d:${child.path}` : `f:${child.path}`}
                node={child}
                depth={depth + 1}
                activeFile={activeFile}
                expanded={expanded}
                dropTarget={dropTarget}
                onToggle={onToggle}
                onSelect={onSelect}
                onDeleteFile={onDeleteFile}
                onMoveFile={onMoveFile}
                onFolderDragOver={onFolderDragOver}
                onFolderDragLeave={onFolderDragLeave}
                onFolderDrop={onFolderDrop}
              />
            ))}
          </ul>
        )}
      </li>
    )
  }

  const Icon = iconForFile(node.name)
  const active = node.path === activeFile
  const protectedFile = isProtectedProjectFile(node.path)
  const canDrag = !protectedFile && !!onMoveFile

  return (
    <li
      draggable={canDrag}
      onDragStart={(e) => {
        if (!canDrag) return
        e.dataTransfer.setData(INTERNAL_DRAG_MIME, node.path)
        e.dataTransfer.effectAllowed = 'move'
      }}
      className="group/file"
    >
      <div
        className={`flex items-center pr-1 ${
          active ? 'bg-[#37373d] text-[#ffffff]' : 'text-[#cccccc] hover:bg-[#2a2d2e]'
        }`}
      >
        <button
          type="button"
          onClick={() => onSelect(node.path)}
          className="flex flex-1 min-w-0 items-center gap-2 pr-3 py-1 text-[13px] text-left"
          style={{ paddingLeft: pad + 16 }}
        >
          <Icon className="w-3.5 h-3.5 shrink-0 opacity-70" />
          <span className="truncate">{node.name}</span>
        </button>
        {!protectedFile && onDeleteFile && (
          <button
            type="button"
            title="删除文件"
            onClick={(e) => {
              e.stopPropagation()
              onDeleteFile(node.path)
            }}
            className="p-1 rounded opacity-0 group-hover/file:opacity-100 hover:bg-[#3c3c3c] text-[#858585] hover:text-red-400 shrink-0"
          >
            <Trash2 className="w-3 h-3" />
          </button>
        )}
      </div>
    </li>
  )
}

export default function FileExplorer({
  files,
  activeFile,
  onSelect,
  onCreateFile,
  onCreateFolder,
  onImportFiles,
  onDeleteFile,
  onMoveFile,
}: FileExplorerProps) {
  const [rootOpen, setRootOpen] = useState(true)
  const [expanded, setExpanded] = useState<Set<string>>(() => {
    const folders = new Set(collectFolderPaths(files))
    folders.add('assets')
    folders.add('assets/sprites')
    return folders
  })
  const [prompt, setPrompt] = useState<'file' | 'folder' | null>(null)
  const [inputValue, setInputValue] = useState('')
  const [dragOver, setDragOver] = useState(false)
  const [dropTarget, setDropTarget] = useState<string | null>(null)
  const [importing, setImporting] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const tree = useMemo(() => buildFileTree(files), [files])

  const isInternalDrag = (e: React.DragEvent) =>
    Array.from(e.dataTransfer.types).includes(INTERNAL_DRAG_MIME)

  const processImport = async (fileMap: Map<string, File>) => {
    if (!onImportFiles || fileMap.size === 0) return
    setImporting(true)
    try {
      const encoded = await encodeDroppedFiles(fileMap)
      const valid: Record<string, string> = {}
      for (const [path, content] of Object.entries(encoded)) {
        const normalized = normalizeProjectPath(path)
        if (validateProjectPath(normalized, false)) continue
        valid[normalized] = content
      }
      if (Object.keys(valid).length === 0) {
        showAlert('没有可导入的文件（路径不合法或为空）', 'warning')
        return
      }
      onImportFiles(valid)
      setExpanded((prev) => {
        const next = new Set(prev)
        next.add('assets')
        next.add('assets/sprites')
        for (const p of Object.keys(valid)) {
          for (const folder of collectFolderPaths([p])) next.add(folder)
        }
        return next
      })
      showAlert(`已导入 ${Object.keys(valid).length} 个文件到项目`, 'success')
    } catch {
      showAlert('文件导入失败', 'error')
    } finally {
      setImporting(false)
      setDragOver(false)
    }
  }

  const handleExternalDrop = async (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (isInternalDrag(e) || !onImportFiles || importing) return
    const fileMap = await collectFilesFromDataTransfer(e.dataTransfer, 'assets/sprites/')
    await processImport(fileMap)
  }

  const handleFolderDragOver = (folder: string, e: React.DragEvent) => {
    if (!isInternalDrag(e) || !onMoveFile) return
    e.preventDefault()
    e.stopPropagation()
    e.dataTransfer.dropEffect = 'move'
    setDropTarget(folder)
    setDragOver(false)
  }

  const handleFolderDragLeave = (folder: string, e: React.DragEvent) => {
    e.stopPropagation()
    if (dropTarget === folder) setDropTarget(null)
  }

  const handleFolderDrop = (folder: string, e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setDropTarget(null)
    const from = e.dataTransfer.getData(INTERNAL_DRAG_MIME)
    if (!from || !onMoveFile) return
    if (dirname(from) === folder) return
    onMoveFile(from, folder)
  }

  const handleFileInputChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const list = e.target.files
    if (!list?.length) return
    const fileMap = new Map<string, File>()
    for (let i = 0; i < list.length; i++) {
      const file = list[i]
      const rel = (file as File & { webkitRelativePath?: string }).webkitRelativePath || file.name
      fileMap.set(`assets/sprites/${rel.replace(/^\/+/, '')}`, file)
    }
    e.target.value = ''
    await processImport(fileMap)
  }

  const toggleFolder = (path: string) => {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(path)) next.delete(path)
      else next.add(path)
      return next
    })
  }

  const openPrompt = (type: 'file' | 'folder') => {
    setPrompt(type)
    setInputValue(type === 'file' ? 'assets/' : 'assets')
  }

  const submitPrompt = () => {
    const raw = normalizeProjectPath(inputValue)
    if (!raw) return
    if (prompt === 'file') onCreateFile(raw)
    else if (prompt === 'folder') onCreateFolder(raw)
    setPrompt(null)
    setInputValue('')
    if (prompt === 'folder') {
      setExpanded((prev) => new Set(prev).add(raw))
    }
  }

  return (
    <div
      className="py-1 relative min-h-[120px]"
      onDragEnter={(e) => {
        if (isInternalDrag(e)) return
        e.preventDefault()
        setDragOver(true)
      }}
      onDragOver={(e) => {
        if (isInternalDrag(e)) return
        e.preventDefault()
        setDragOver(true)
      }}
      onDragLeave={(e) => {
        if (isInternalDrag(e)) return
        if (e.currentTarget.contains(e.relatedTarget as Node)) return
        setDragOver(false)
      }}
      onDrop={handleExternalDrop}
    >
      <input
        ref={fileInputRef}
        type="file"
        multiple
        className="hidden"
        onChange={handleFileInputChange}
      />
      {dragOver && onImportFiles && (
        <div className="absolute inset-1 z-10 rounded-lg border-2 border-dashed border-[#007fd4] bg-[#007fd4]/10 flex items-center justify-center pointer-events-none">
          <p className="text-[11px] text-[#9cdcfe] px-3 text-center">
            松开以导入到 assets/sprites/
          </p>
        </div>
      )}
      <div className="flex items-center justify-between px-2 mb-1">
        <button
          type="button"
          onClick={() => setRootOpen(!rootOpen)}
          className="flex items-center gap-1 flex-1 min-w-0 px-1 py-1 text-[11px] font-semibold uppercase tracking-wide text-[#bbbbbb] hover:bg-[#2a2d2e] rounded"
        >
          {rootOpen ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" /> }
          <span className="truncate">桌宠项目</span>
        </button>
        <div className="flex items-center shrink-0">
          <button
            type="button"
            title="上传文件到 assets/sprites/"
            onClick={() => fileInputRef.current?.click()}
            disabled={importing || !onImportFiles}
            className="p-1 rounded hover:bg-[#2a2d2e] text-[#858585] hover:text-[#cccccc] disabled:opacity-40"
          >
            <Upload className="w-3.5 h-3.5" />
          </button>
          <button
            type="button"
            title="新建文件"
            onClick={() => openPrompt('file')}
            className="p-1 rounded hover:bg-[#2a2d2e] text-[#858585] hover:text-[#cccccc]"
          >
            <FilePlus className="w-3.5 h-3.5" />
          </button>
          <button
            type="button"
            title="新建文件夹"
            onClick={() => openPrompt('folder')}
            className="p-1 rounded hover:bg-[#2a2d2e] text-[#858585] hover:text-[#cccccc]"
          >
            <FolderPlus className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      {prompt && (
        <div className="mx-2 mb-2 p-2 rounded border border-[#3c3c3c] bg-[#1e1e1e]">
          <div className="text-[10px] text-[#858585] mb-1">
            {prompt === 'file' ? '新建文件（如 assets/config.json）' : '新建文件夹（如 assets）'}
          </div>
          <input
            autoFocus
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') submitPrompt()
              if (e.key === 'Escape') setPrompt(null)
            }}
            className="w-full px-2 py-1 text-[12px] rounded bg-[#3c3c3c] border border-[#4a4a4a] text-white outline-none focus:border-[#007fd4]"
            placeholder={prompt === 'file' ? 'path/to/file.js' : 'folder-name'}
          />
          <div className="flex justify-end gap-1 mt-2">
            <button type="button" onClick={() => setPrompt(null)} className="px-2 py-0.5 text-[11px] text-[#858585] hover:text-white">
              取消
            </button>
            <button type="button" onClick={submitPrompt} className="px-2 py-0.5 text-[11px] bg-[#007fd4] text-white rounded">
              创建
            </button>
          </div>
        </div>
      )}

      {rootOpen && (
        <ul className="mt-0.5">
          {tree.map((node) => (
            <TreeNode
              key={node.kind === 'folder' ? `d:${node.path}` : `f:${node.path}`}
              node={node}
              depth={0}
              activeFile={activeFile}
              expanded={expanded}
              dropTarget={dropTarget}
              onToggle={toggleFolder}
              onSelect={onSelect}
              onDeleteFile={onDeleteFile}
              onMoveFile={onMoveFile}
              onFolderDragOver={handleFolderDragOver}
              onFolderDragLeave={handleFolderDragLeave}
              onFolderDrop={handleFolderDrop}
            />
          ))}
        </ul>
      )}

      {rootOpen && (
        <p className="px-4 py-2 text-[10px] text-[#555] leading-relaxed">
          拖入外部文件 → 导入；拖项目内文件到文件夹 → 移动；悬停显示删除按钮
        </p>
      )}
    </div>
  )
}

/** @deprecated flat list helper */
export function sortProjectFiles(files: string[]): string[] {
  const order: string[] = [PROJECT_FILES.manifest, PROJECT_FILES.entry, PROJECT_FILES.style, PROJECT_FILES.readme]
  return [...files].sort((a, b) => {
    const ai = order.indexOf(a)
    const bi = order.indexOf(b)
    return (ai === -1 ? 99 : ai) - (bi === -1 ? 99 : bi)
  })
}
