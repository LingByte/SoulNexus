import { useState } from 'react'
import { Sparkles, Loader2, X } from 'lucide-react'
import Button from '@/components/UI/Button'
import Textarea from '@/components/UI/Textarea'
import { petPackageService } from '@/api/petPackage'
import { notifyApiError, notifyApiResult } from '@/utils/apiFeedback'
import { showAlert } from '@/utils/notification'
import { PROJECT_FILES } from '@/pages/pet-market/types'

interface ManifestAiPanelProps {
  manifest: string
  kind?: string
  assetFiles: string[]
  onApply: (manifest: string) => void
  onClose: () => void
}

export default function ManifestAiPanel({ manifest, kind, assetFiles, onApply, onClose }: ManifestAiPanelProps) {
  const [instruction, setInstruction] = useState('')
  const [loading, setLoading] = useState(false)
  const [explanation, setExplanation] = useState<string | null>(null)

  const handleRun = async () => {
    const text = instruction.trim()
    if (!text) {
      showAlert('请输入修改指令', 'warning')
      return
    }
    setLoading(true)
    setExplanation(null)
    try {
      const res = await petPackageService.aiAssistManifest({
        instruction: text,
        manifest,
        kind,
        assetFiles,
      })
      if (!notifyApiResult(res, { silentSuccess: true })) return
      const data = res.data
      if (!data?.manifest) {
        showAlert('AI 未返回有效 manifest', 'error')
        return
      }
      setExplanation(data.explanation || null)
      onApply(data.manifest)
      showAlert('已应用 AI 生成的 manifest', 'success')
    } catch (e) {
      notifyApiError(e, 'AI 辅助失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="absolute right-0 top-0 bottom-0 w-80 z-20 bg-[#252526] border-l border-[#2b2b2b] flex flex-col shadow-xl">
      <div className="flex items-center justify-between px-3 py-2 border-b border-[#2b2b2b]">
        <div className="flex items-center gap-2 text-sm text-white">
          <Sparkles className="w-4 h-4 text-amber-400" />
          AI Manifest
        </div>
        <button type="button" onClick={onClose} className="p-1 rounded hover:bg-[#3c3c3c] text-[#858585]">
          <X className="w-4 h-4" />
        </button>
      </div>
      <div className="p-3 flex-1 flex flex-col gap-3 overflow-y-auto text-[12px]">
        <p className="text-[#858585] leading-relaxed">
          描述你想对 <code className="text-amber-200/90">{PROJECT_FILES.manifest}</code> 做的修改，例如添加动画、根据 PNG 文件名整理 frames。
        </p>
        <Textarea
          value={instruction}
          onChange={(e) => setInstruction(e.target.value)}
          placeholder="例如：添加 tap 动画，使用 ghost_daze_1 到 ghost_daze_8，fps 12，loop false"
          rows={6}
          className="text-[12px] bg-[#1e1e1e] border-[#3c3c3c] text-[#cccccc]"
        />
        {assetFiles.length > 0 && (
          <div className="text-[10px] text-[#858585]">
            已传入 {assetFiles.length} 个资源文件名供 AI 参考
          </div>
        )}
        {explanation && (
          <div className="rounded border border-[#3c3c3c] bg-[#1e1e1e] p-2 text-[#cccccc]">{explanation}</div>
        )}
        <Button
          size="sm"
          variant="primary"
          className="w-full"
          leftIcon={loading ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Sparkles className="w-3.5 h-3.5" />}
          onClick={() => void handleRun()}
          disabled={loading}
        >
          {loading ? '生成中…' : '生成并应用'}
        </Button>
      </div>
    </div>
  )
}
