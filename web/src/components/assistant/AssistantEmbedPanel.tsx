import { useMemo, useState, useEffect } from 'react'
import { Select, Typography } from '@arco-design/web-react'
import { IconCopy, IconLaunch } from '@arco-design/web-react/icon'
import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui'
import { getApiBaseURL } from '@/config/apiConfig'
import { listJSTemplates, type JSTemplateRow } from '@/api/jsTemplates'
import { useTranslation } from '@/i18n'
import { showAlert } from '@/utils/notification'

type Props = {
  assistantId: string
  boundJsTemplateSourceId?: string
  onBoundJsTemplateSourceIdChange?: (jsSourceId: string) => void
}

export default function AssistantEmbedPanel({
  assistantId,
  boundJsTemplateSourceId = '',
  onBoundJsTemplateSourceIdChange,
}: Props) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [copied, setCopied] = useState(false)
  const [templates, setTemplates] = useState<JSTemplateRow[]>([])

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const res = await listJSTemplates(1, 100)
        if (!cancelled && res.code === 200 && res.data?.list) {
          setTemplates(res.data.list)
        }
      } catch {
        /* optional */
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  const snippet = useMemo(() => {
    const apiBase = getApiBaseURL().replace(/\/$/, '')
    const jsId = boundJsTemplateSourceId.trim()
    const scriptSrc = jsId
      ? `${apiBase}/lingecho/embed/v1/t/${encodeURIComponent(jsId)}/embed.js`
      : `${apiBase}/lingecho/embed/v1/embed.js`
    return `<script>
  window.__LingEchoConfig = {
    apiBase: '${apiBase}',
    apiKey: 'YOUR_API_KEY',
    assistantId: '${assistantId}',
    transport: 'text', // dialog/v1; or 'websocket' / 'webrtc' for voice
  };
</script>
<script src="${scriptSrc}" async></script>`
  }, [assistantId, boundJsTemplateSourceId])

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(snippet)
      setCopied(true)
      showAlert(t('common.copySuccess'), 'success')
      window.setTimeout(() => setCopied(false), 1500)
    } catch {
      showAlert(t('common.copyFailed'), 'error')
    }
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <Typography.Text bold>{t('assistant.embedTitle')}</Typography.Text>
          <p className="mt-1 text-sm text-muted-foreground">{t('assistant.embedDesc')}</p>
        </div>
        <Button
          type="outline"
          size="mini"
          icon={<IconLaunch />}
          onClick={() => navigate('/js-templates')}
        >
          {t('assistant.manageWidgets')}
        </Button>
      </div>
      <div className="space-y-1.5">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <Typography.Text className="!text-xs text-muted-foreground">{t('assistant.embedTemplate')}</Typography.Text>
          <button
            type="button"
            className="text-xs text-primary hover:underline"
            onClick={() => navigate('/js-templates/new')}
          >
            {t('assistant.createWidget')}
          </button>
        </div>
        <Select
          allowClear
          placeholder={t('assistant.embedTemplateDefault')}
          value={boundJsTemplateSourceId || undefined}
          onChange={(v) => onBoundJsTemplateSourceIdChange?.(String(v || ''))}
          options={[
            ...templates.map((tpl) => ({
              value: tpl.jsSourceId,
              label: `${tpl.name} (${tpl.jsSourceId})`,
            })),
          ]}
          style={{ width: '100%' }}
        />
      </div>
      <div className="flex items-center justify-between gap-2">
        <Typography.Text className="!text-xs text-muted-foreground">{t('assistant.embedHint')}</Typography.Text>
        <Button type="outline" size="mini" icon={<IconCopy />} onClick={() => void copy()}>
          {copied ? t('common.copySuccess') : t('common.copy')}
        </Button>
      </div>
      <pre className="overflow-x-auto rounded-lg border border-border bg-muted/40 p-3 font-mono text-[11px] leading-relaxed text-muted-foreground whitespace-pre">
        {snippet}
      </pre>
    </div>
  )
}
