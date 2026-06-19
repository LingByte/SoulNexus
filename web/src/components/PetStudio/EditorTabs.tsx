import { Suspense, lazy, useCallback, useRef } from 'react'
import { ImageIcon, X } from 'lucide-react'
import { languageForFile } from '@/pages/pet-market/projectUtils'
import {
  formatBinarySize,
  isBinaryProjectFile,
  toDataUrl,
} from '@/pages/pet-market/projectAssetUtils'

const MonacoEditor = lazy(() => import('@monaco-editor/react'))

interface EditorTabsProps {
  openFiles: string[]
  activeFile: string
  dirtyFiles: Set<string>
  fileContents: Record<string, string>
  onSelect: (path: string) => void
  onClose: (path: string) => void
  onChange: (path: string, value: string) => void
  onSave?: () => void
}

function BinaryFileView({ path, content }: { path: string; content: string }) {
  const dataUrl = toDataUrl(path, content)
  const isImage = /\.(png|jpe?g|webp|gif)$/i.test(path)

  if (isImage && dataUrl) {
    return (
      <div className="h-full flex flex-col items-center justify-center gap-3 p-6 bg-[#1e1e1e]">
        <img src={dataUrl} alt={path} className="max-w-full max-h-[70vh] object-contain rounded border border-[#3c3c3c]" />
        <p className="text-xs text-[#858585]">{path} · {formatBinarySize(content)}</p>
      </div>
    )
  }

  return (
    <div className="h-full flex flex-col items-center justify-center gap-3 p-8 bg-[#1e1e1e] text-center">
      <ImageIcon className="w-10 h-10 text-[#858585]" />
      <div>
        <p className="text-sm text-[#cccccc]">{path}</p>
        <p className="text-xs text-[#858585] mt-1">二进制资源 · {formatBinarySize(content)}</p>
        <p className="text-[11px] text-[#666] mt-2 max-w-sm">此文件由模板导入，保存后会一并上传到对象存储。可在资源管理器中替换对应路径的文件内容。</p>
      </div>
    </div>
  )
}

export default function EditorTabs({
  openFiles,
  activeFile,
  dirtyFiles,
  fileContents,
  onSelect,
  onClose,
  onChange,
  onSave,
}: EditorTabsProps) {
  const onSaveRef = useRef(onSave)
  onSaveRef.current = onSave
  const activeContent = fileContents[activeFile] ?? ''
  const binaryView = isBinaryProjectFile(activeFile, activeContent)

  const handleEditorMount = useCallback((editor: import('monaco-editor').editor.IStandaloneCodeEditor, monaco: typeof import('monaco-editor')) => {
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
      onSaveRef.current?.()
    })
  }, [])

  return (
    <div className="flex flex-col flex-1 min-w-0 min-h-0">
      <div className="flex items-stretch h-9 bg-[#252526] border-b border-[#2b2b2b] overflow-x-auto shrink-0">
        {openFiles.map((file) => {
          const active = file === activeFile
          const dirty = dirtyFiles.has(file)
          return (
            <div
              key={file}
              className={`group flex items-center gap-1 pl-3 pr-1 border-r border-[#2b2b2b] cursor-pointer shrink-0 max-w-[180px] ${
                active ? 'bg-[#1e1e1e] text-[#ffffff]' : 'bg-[#2d2d2d] text-[#969696] hover:bg-[#1e1e1e]'
              }`}
            >
              <button type="button" onClick={() => onSelect(file)} className="flex items-center gap-1 min-w-0 py-2 text-[12px]">
                <span className="truncate">{file}</span>
                {dirty && <span className="text-[#cccccc]">•</span>}
              </button>
              <button
                type="button"
                onClick={(e) => { e.stopPropagation(); onClose(file) }}
                className="p-0.5 rounded opacity-0 group-hover:opacity-100 hover:bg-[#3c3c3c]"
              >
                <X className="w-3 h-3" />
              </button>
            </div>
          )
        })}
      </div>
      <div className="flex-1 min-h-0">
        {binaryView ? (
          <BinaryFileView path={activeFile} content={activeContent} />
        ) : (
          <Suspense fallback={<div className="h-full flex items-center justify-center text-[#858585] text-sm">加载编辑器...</div>}>
            <MonacoEditor
              key={activeFile}
              height="100%"
              language={languageForFile(activeFile)}
              value={activeContent}
              onChange={(v) => onChange(activeFile, v ?? '')}
              onMount={handleEditorMount}
              theme="vs-dark"
              options={{
                minimap: { enabled: true },
                fontSize: 13,
                lineNumbers: 'on',
                wordWrap: 'on',
                automaticLayout: true,
                tabSize: 2,
                scrollBeyondLastLine: false,
                padding: { top: 8 },
              }}
            />
          </Suspense>
        )}
      </div>
    </div>
  )
}
