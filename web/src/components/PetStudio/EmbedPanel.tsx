import { useMemo, useState } from 'react'
import { Check, Code2, Copy, X } from 'lucide-react'
import { getApiBaseURL } from '@/config/apiConfig'

interface EmbedPanelProps {
  jsSourceId: string
  templateName: string
  onClose: () => void
}

export default function EmbedPanel({ jsSourceId, templateName, onClose }: EmbedPanelProps) {
  const [copied, setCopied] = useState<string | null>(null)
  const apiBase = getApiBaseURL().replace(/\/$/, '')
  const staticBase = apiBase.replace(/\/api$/, '') + '/api/static/pet'

  const snippets = useMemo(() => ({
    script: `<!-- ${templateName} — JS 注入（需 HTTP 页面） -->
<div class="soul-pet-widget" style="position:fixed;right:24px;bottom:24px;width:360px;height:480px;z-index:9999;pointer-events:auto;overflow:hidden">
  <div id="app" style="width:100%;height:100%"></div>
</div>
<script>
  window.SERVER_BASE = '${apiBase}';
  window.__AIPetConfig = {
    agentId: YOUR_AGENT_ID,
    apiKey: 'yourApiKey',
    apiSecret: 'yourApiSecret',
    cmdVoiceBase: 'http://127.0.0.1:7080'
  };
</script>
<script src="${apiBase}/js-templates/embed/${jsSourceId}/loader.js"></script>`,
    iframe: `<!-- ${templateName} — iframe 桌宠 SDK（需 HTTP 页面，勿用 file://） -->
<script>
  window.__AIPetConfig = {
    jsSourceId: '${jsSourceId}',
    position: 'bottom-right',
    width: 360,
    height: 480
  };
</script>
<script src="${staticBase}/loader.js"></script>`,
    agent: `<!-- 绑定到语音智能体后，用智能体 loader（含桌宠 + 聊天按钮） -->
<script>
  window.__AIPetConfig = {
    agentId: YOUR_AGENT_ID,
    apiKey: 'yourApiKey',
    apiSecret: 'yourApiSecret',
    cmdVoiceBase: 'http://127.0.0.1:7080'
  };
</script>
<script src="${apiBase}/agents/lingecho/client/YOUR_AGENT_JS_SOURCE_ID/loader.js"></script>`,
  }), [apiBase, staticBase, jsSourceId, templateName])

  const copy = async (key: keyof typeof snippets) => {
    await navigator.clipboard.writeText(snippets[key])
    setCopied(key)
    window.setTimeout(() => setCopied(null), 2000)
  }

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/60 p-4" onClick={onClose}>
      <div
        className="w-full max-w-2xl max-h-[90vh] overflow-auto rounded-xl bg-[#252526] border border-[#3c3c3c] shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between px-4 py-3 border-b border-[#3c3c3c]">
          <div className="flex items-center gap-2 text-white text-sm font-medium">
            <Code2 className="w-4 h-4 text-[#007fd4]" />
            嵌入到第三方项目
          </div>
          <button type="button" onClick={onClose} className="p-1 rounded hover:bg-[#3c3c3c] text-[#858585]">
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="p-4 space-y-4 text-[#cccccc] text-sm">
          <p className="text-amber-400/90 text-xs leading-relaxed rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2">
            请勿用 <code className="text-amber-200">file://</code> 双击本地 HTML 测试。请通过 HTTP 访问演示页：
            <a className="text-sky-400 underline ml-1" href={`${apiBase.replace(/\/api$/, '')}/api/static/pet/host-demo.html?jsSourceId=${jsSourceId}`} target="_blank" rel="noreferrer">
              打开 host-demo
            </a>
          </p>
          <p className="text-[#858585] text-xs leading-relaxed">
            桌宠代码保存在对象存储；第三方页面通过 <code className="text-[#9cdcfe]">loader.js</code> 注入即可运行。
            推荐先在本 Studio 保存成功，再复制下方代码。帧动画桌宠右下角有<strong className="text-[#ccc]">聊天按钮</strong>，点击即可开始对话（需配置 apiKey / apiSecret）。
          </p>

          {(
            [
              ['script', '方式一：JS 注入（推荐）', '直接执行你编辑的 pet.js + 帧动画'],
              ['iframe', '方式二：iframe SDK', '透明浮窗桌宠，适合不改动宿主 DOM'],
              ['agent', '方式三：语音智能体', '需在智能体里绑定此桌宠模板'],
            ] as const
          ).map(([key, title, desc]) => (
            <section key={key}>
              <div className="flex items-center justify-between mb-1.5">
                <div>
                  <h3 className="text-white text-[13px] font-medium">{title}</h3>
                  <p className="text-[11px] text-[#666]">{desc}</p>
                </div>
                <button
                  type="button"
                  onClick={() => void copy(key)}
                  className="flex items-center gap-1 px-2 py-1 rounded text-[11px] bg-[#3c3c3c] hover:bg-[#4a4a4a] text-white shrink-0"
                >
                  {copied === key ? <Check className="w-3 h-3" /> : <Copy className="w-3 h-3" />}
                  {copied === key ? '已复制' : '复制'}
                </button>
              </div>
              <pre className="p-3 rounded-lg bg-[#1e1e1e] border border-[#3c3c3c] text-[11px] leading-relaxed overflow-x-auto text-[#d4d4d4] whitespace-pre-wrap">
                {snippets[key]}
              </pre>
            </section>
          ))}
        </div>
      </div>
    </div>
  )
}
