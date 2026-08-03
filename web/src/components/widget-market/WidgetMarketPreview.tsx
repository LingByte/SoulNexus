import { useEffect, useMemo, useState } from 'react'
import { Typography } from '@arco-design/web-react'
import { Loading } from '@/components/ui'
import { useTranslation } from '@/i18n'

type WidgetMarketPreviewProps = {
  embedPath: string
  displayName?: string
  jsSourceId?: string
}

function buildPreviewSrcDoc(opts: {
  apiBase: string
  scriptUrl: string
  title: string
}): string {
  const { apiBase, scriptUrl, title } = opts
  const cfg = {
    apiBase,
    autoMount: true,
    position: 'right',
      size: 88,
    title,
    name: title,
    persist: false,
    autoWander: true,
    autoChat: false,
    watchCoding: false,
  }
  return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8" />
<meta name="viewport" content="width=device-width, initial-scale=1.0" />
<title>${title.replace(/</g, '')} preview</title>
<style>
  html, body { margin: 0; height: 100%; background:
    radial-gradient(ellipse at 30% 20%, #f8fafc 0%, transparent 50%),
    radial-gradient(ellipse at 80% 80%, #eef2ff 0%, transparent 45%),
    linear-gradient(160deg, #f4f4f5 0%, #e4e4e7 100%);
    font: 13px system-ui, sans-serif; overflow: hidden; }
  #boot { padding: 14px 16px; color: #71717a; }
  #boot.err { color: #b91c1c; white-space: pre-wrap; }
  #hint { position: absolute; left: 12px; bottom: 10px; z-index: 0;
    font-size: 11px; color: #a1a1aa; pointer-events: none; }
</style>
</head>
<body>
<div id="boot">正在加载预览…</div>
<div id="hint">可拖拽角色 · 预览不连对话</div>
<script>
(function () {
  var boot = document.getElementById('boot');
  function fail(msg) {
    boot.textContent = msg;
    boot.classList.add('err');
  }
  window.__LINGECHO_EMBED_MODE__ = 'preview';
  var cfg = ${JSON.stringify(cfg)};
  window.__LingEchoConfig = Object.assign({}, window.__LingEchoConfig || {}, cfg);
  window.__LanlanConfig = Object.assign({}, window.__LanlanConfig || {}, cfg);
  window.__KongkongConfig = Object.assign({}, window.__KongkongConfig || {}, cfg);
  var scriptUrl = ${JSON.stringify(scriptUrl)};
  var s = document.createElement('script');
  s.src = scriptUrl + (scriptUrl.indexOf('?') >= 0 ? '&' : '?') + '_=' + Date.now();
  s.onload = function () {
    var watch = setInterval(function () {
      var root = document.getElementById('lingecho-embed-root')
        || document.getElementById('lanlan-pet-root')
        || document.getElementById('kongkong-pet-root');
      if (root) {
        clearInterval(watch);
        boot.style.display = 'none';
      }
    }, 200);
    setTimeout(function () {
      clearInterval(watch);
      if (boot.style.display !== 'none') fail('预览超时，请检查挂件脚本是否可访问');
    }, 20000);
  };
  s.onerror = function () { fail('加载失败:\\n' + scriptUrl); };
  document.body.appendChild(s);
})();
</script>
</body>
</html>`
}

export default function WidgetMarketPreview({
  embedPath,
  displayName,
  jsSourceId,
}: WidgetMarketPreviewProps) {
  const { t } = useTranslation()
  const [ready, setReady] = useState(false)

  const { apiBase, scriptUrl, title } = useMemo(() => {
    const origin = typeof window !== 'undefined' ? window.location.origin : ''
    const base = `${origin}/api`
    const path = embedPath.startsWith('/') ? embedPath : `/${embedPath}`
    return {
      apiBase: base,
      scriptUrl: `${base}${path}`,
      title: displayName || jsSourceId || 'Widget',
    }
  }, [displayName, embedPath, jsSourceId])

  const srcDoc = useMemo(
    () => buildPreviewSrcDoc({ apiBase, scriptUrl, title }),
    [apiBase, scriptUrl, title],
  )

  useEffect(() => {
    setReady(false)
    const timer = window.setTimeout(() => setReady(true), 40)
    return () => window.clearTimeout(timer)
  }, [srcDoc])

  if (!embedPath) {
    return (
      <Typography.Text type="secondary">{t('widgetMarket.previewUnavailable')}</Typography.Text>
    )
  }

  return (
    <div className="relative overflow-hidden rounded-xl border border-border bg-muted/40">
      <div className="flex items-center justify-between border-b border-border/70 px-3 py-2">
        <Typography.Text className="!text-xs !font-medium text-neutral-700">
          {t('widgetMarket.livePreview')}
        </Typography.Text>
        <Typography.Text type="secondary" className="!text-[11px] font-mono truncate max-w-[55%]">
          {jsSourceId}
        </Typography.Text>
      </div>
      <div className="relative h-[360px] w-full bg-transparent">
        {!ready ? (
          <div className="absolute inset-0 flex items-center justify-center">
            <Loading tip={t('widgetMarket.previewLoading')} />
          </div>
        ) : (
          <iframe
            title={t('widgetMarket.livePreview')}
            srcDoc={srcDoc}
            className="h-full w-full border-0"
            sandbox="allow-scripts allow-same-origin"
            referrerPolicy="no-referrer"
          />
        )}
      </div>
    </div>
  )
}
