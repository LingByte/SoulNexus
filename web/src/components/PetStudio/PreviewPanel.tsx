import { useCallback, useEffect, useRef, useState } from 'react'
import { AlertCircle, Eye, RefreshCw } from 'lucide-react'
import { PROJECT_FILES, type PetProjectV1 } from '@/pages/pet-market/types'
import { getApiBaseURL } from '@/config/apiConfig'
import { petProjectService } from '@/api/petProject'
import { hasSpriteAssets } from '@/pages/pet-market/spriteProjectUtils'
import { guardLegacyPetJs, previewApiBase, previewRewriteApiUrl } from '@/lib/voice/guardLegacyPetJs'

const PREVIEW_CHANNEL = 'soul-pet-preview'
const PREVIEW_URL = '/pet-studio-preview.html'

interface PreviewPanelProps {
  project: PetProjectV1
  visible: boolean
  templateId?: string | null
  /** Public embed id — used when model files are already saved to object storage */
  jsSourceId?: string | null
  /** True when assets/sprites/ is missing from the in-memory project */
  missingModelAssets?: boolean
  onToggle: () => void
}

function projectToPayload(project: PetProjectV1, projectBase: string) {
  return {
    entry: guardLegacyPetJs(project.files[project.entry] || project.files[PROJECT_FILES.entry] || ''),
    css: project.files[PROJECT_FILES.style] || '',
    manifest: project.files[PROJECT_FILES.manifest] || '{}',
    apiBase: previewApiBase(),
    projectBase,
  }
}

/** Public embed base — no auth, works from Vite preview iframe (cross-origin). */
function embedProjectBase(jsSourceId: string): string {
  return previewRewriteApiUrl(`${getApiBaseURL()}/js-templates/embed/${encodeURIComponent(jsSourceId)}/file/`)
}

export default function PreviewPanel({
  project,
  visible,
  templateId,
  jsSourceId,
  missingModelAssets = false,
  onToggle,
}: PreviewPanelProps) {
  const iframeRef = useRef<HTMLIFrameElement>(null)
  const [iframeReady, setIframeReady] = useState(false)
  const [previewError, setPreviewError] = useState<string | null>(null)
  const [registering, setRegistering] = useState(false)
  const projectRef = useRef(project)
  const projectBaseRef = useRef('')
  projectRef.current = project

  const postRender = useCallback(() => {
    const win = iframeRef.current?.contentWindow
    if (!win) return
    setPreviewError(null)
    win.postMessage(
      {
        channel: PREVIEW_CHANNEL,
        type: 'render',
        payload: projectToPayload(projectRef.current, projectBaseRef.current),
      },
      '*',
    )
  }, [])

  useEffect(() => {
    const onMessage = (e: MessageEvent) => {
      if (e.data?.channel !== PREVIEW_CHANNEL) return
      if (e.data?.type === 'ready') {
        setIframeReady(true)
      }
      if (e.data?.type === 'error' && typeof e.data.message === 'string') {
        setPreviewError(e.data.message)
      }
    }
    window.addEventListener('message', onMessage)
    return () => window.removeEventListener('message', onMessage)
  }, [postRender])

  useEffect(() => {
    if (!visible || !iframeReady) return
    let cancelled = false
    const run = async () => {
      setRegistering(true)
      try {
        const hasSprites = hasSpriteAssets(projectRef.current)

        if (hasSprites || !jsSourceId?.trim()) {
          const res = await petProjectService.registerPreviewSession(projectRef.current.files)
          if (cancelled) return
          if (res.code !== 200 || !res.data?.baseUrl) {
            setPreviewError(
              missingModelAssets
                ? '缺少 assets/sprites/ 帧图，请上传 PNG 后重试'
                : '预览资源注册失败，请先保存项目或确认已登录',
            )
            return
          }
          projectBaseRef.current = previewRewriteApiUrl(res.data.baseUrl)
        } else {
          projectBaseRef.current = embedProjectBase(jsSourceId!.trim())
        }
        postRender()
      } catch {
        if (!cancelled) setPreviewError('预览资源加载失败')
      } finally {
        if (!cancelled) setRegistering(false)
      }
    }
    const t = window.setTimeout(() => { void run() }, 400)
    return () => {
      cancelled = true
      window.clearTimeout(t)
    }
  }, [project, visible, iframeReady, jsSourceId, missingModelAssets, templateId, postRender])

  const handleRefresh = () => {
    setIframeReady(false)
    setPreviewError(null)
    projectBaseRef.current = ''
    if (iframeRef.current) {
      iframeRef.current.src = `${PREVIEW_URL}?t=${Date.now()}`
    }
  }

  if (!visible) return null

  return (
    <div className="flex flex-col h-full w-full border-l border-[#2b2b2b] bg-[#1e1e1e] min-w-[300px]">
      <div className="h-9 flex items-center justify-between px-3 border-b border-[#2b2b2b] shrink-0">
        <span className="text-[11px] uppercase tracking-wider text-[#cccccc] flex items-center gap-1.5">
          <Eye className="w-3.5 h-3.5" /> 预览
          {(!iframeReady || registering) && (
            <span className="text-[10px] text-[#858585] normal-case">加载中…</span>
          )}
        </span>
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={handleRefresh}
            className="p-1 rounded hover:bg-[#2a2d2e] text-[#858585] hover:text-[#cccccc]"
            title="刷新预览"
          >
            <RefreshCw className="w-3.5 h-3.5" />
          </button>
          <button type="button" onClick={onToggle} className="text-[10px] text-[#858585] hover:text-[#cccccc] px-1">
            隐藏
          </button>
        </div>
      </div>
      {previewError && (
        <div className="px-2 py-1.5 bg-red-950/50 border-b border-red-900/50 flex gap-1.5 text-[10px] text-red-300">
          <AlertCircle className="w-3 h-3 shrink-0 mt-0.5" />
          <span className="line-clamp-4 whitespace-pre-wrap">{previewError}</span>
        </div>
      )}
      <div className="flex-1 p-2 min-h-0 bg-[#0f172a]">
        <iframe
          ref={iframeRef}
          src={PREVIEW_URL}
          title="Pet preview"
          className="w-full h-full rounded border border-[#3c3c3c]"
          sandbox="allow-scripts allow-same-origin allow-popups"
        />
      </div>
      <p className="px-2 py-1 text-[10px] text-[#666] border-t border-[#2b2b2c] shrink-0">
        精灵帧图从 assets/sprites/ 加载；未保存时走临时预览会话
      </p>
    </div>
  )
}
