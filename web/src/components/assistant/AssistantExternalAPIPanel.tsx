import { useMemo, useState } from 'react'
import { Typography } from '@arco-design/web-react'
import { IconCopy } from '@arco-design/web-react/icon'
import { Button, Link } from '@/components/ui'
import { getApiBaseURL, getApiMountPath, buildWebSocketURL } from '@/config/apiConfig'
import { useTranslation } from '@/i18n'
import { showAlert } from '@/utils/notification'

type Props = {
  assistantId: string
}

function joinAPI(path: string): string {
  const base = getApiBaseURL().replace(/\/$/, '')
  const p = path.startsWith('/') ? path : `/${path}`
  if (!base.includes('://')) {
    return `${base}${p.startsWith(base) ? p.slice(base.length) : p}`
  }
  try {
    const u = new URL(base)
    const origin = u.origin
    const basePath = u.pathname.replace(/\/$/, '')
    if (!basePath || basePath === '/') return `${origin}${p}`
    if (p.startsWith(basePath)) return `${origin}${p}`
    return `${origin}${basePath}${p}`
  } catch {
    return `${base}${p}`
  }
}

export default function AssistantExternalAPIPanel({ assistantId }: Props) {
  const { t } = useTranslation()
  const [copied, setCopied] = useState<string | null>(null)

  const absolute = useMemo(() => {
    const mount = getApiMountPath() || '/api'
    const voiceCreate = `${mount}/lingecho/voice-session/v1/sessions`
    const wsPath = `${mount}/lingecho/voice-session/v1/ws`
    const webrtcPath = `${mount}/lingecho/voice-session/v1/webrtc/offer`
    const voiceEnd = `${mount}/lingecho/voice-session/v1/sessions/{sessionId}`
    const dialogCreate = `${mount}/lingecho/dialog/v1/conversations`
    const dialogMessages = `${mount}/lingecho/dialog/v1/conversations/{conversationId}/messages`
    const dialogEnd = `${mount}/lingecho/dialog/v1/conversations/{conversationId}/end`
    const ws = new URL(buildWebSocketURL(wsPath))
    ws.searchParams.set('session_id', '{sessionId}')
    return {
      voiceCreate: joinAPI(voiceCreate),
      webrtc: joinAPI(webrtcPath),
      voiceEnd: joinAPI(voiceEnd),
      dialogCreate: joinAPI(dialogCreate),
      dialogMessages: joinAPI(dialogMessages),
      dialogEnd: joinAPI(dialogEnd),
      ws: ws.toString(),
    }
  }, [])

  const copy = async (label: string, value: string) => {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(label)
      showAlert(t('common.copySuccess'), 'success')
      window.setTimeout(() => setCopied(null), 1500)
    } catch {
      showAlert(t('common.copyFailed'), 'error')
    }
  }

  const createBody = JSON.stringify({ transport: 'websocket', assistantId, sampleRateHz: 16000 })
  const curlCreate = `curl -X POST '${absolute.voiceCreate}' \\
  -H 'X-API-Key: <API_KEY>' \\
  -H 'Content-Type: application/json' \\
  -d '${createBody}'`

  const dialogCreateBody = JSON.stringify({ assistantId, channel: 'api' })
  const curlDialog = `curl -X POST '${absolute.dialogCreate}' \\
  -H 'X-API-Key: <API_KEY>' \\
  -H 'Content-Type: application/json' \\
  -d '${dialogCreateBody}'`

  const EndpointRow = ({
    label,
    method,
    value,
  }: {
    label: string
    method?: string
    value: string
  }) => (
    <div className="flex flex-col gap-1.5 rounded-lg border border-border/70 bg-muted/30 px-3 py-2.5">
      <div className="flex items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2">
          {method ? (
            <span className="shrink-0 rounded bg-neutral-900 px-1.5 py-0.5 font-mono text-[10px] font-semibold text-white dark:bg-neutral-100 dark:text-neutral-900">
              {method}
            </span>
          ) : null}
          <Typography.Text bold className="!text-xs">
            {label}
          </Typography.Text>
        </div>
        <Button type="text" size="mini" icon={<IconCopy />} onClick={() => void copy(label, value)}>
          {copied === label ? t('common.copySuccess') : t('common.copy')}
        </Button>
      </div>
      <code className="break-all font-mono text-[11px] leading-relaxed text-muted-foreground">{value}</code>
    </div>
  )

  return (
    <div className="space-y-4">
      <div>
        <Typography.Text bold>{t('assistant.externalApiTitle')}</Typography.Text>
        <p className="mt-1 text-sm text-muted-foreground">{t('assistant.externalApiDesc')}</p>
      </div>

      <div className="rounded-lg border border-amber-200/80 bg-amber-50/60 px-3 py-2.5 text-xs leading-relaxed text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-100">
        {t('assistant.externalApiAuthHint')}{' '}
        <Link to="/profile/access-keys" className="!text-xs font-medium underline">
          {t('nav.accessKeys')}
        </Link>
        {t('assistant.externalApiAuthHintSuffix')}
      </div>

      <div className="space-y-2">
        <EndpointRow label={t('assistant.externalApiCreate')} method="POST" value={absolute.voiceCreate} />
        <EndpointRow label={t('assistant.externalApiWs')} method="GET" value={absolute.ws} />
        <EndpointRow label={t('assistant.externalApiWebrtc')} method="POST" value={absolute.webrtc} />
        <EndpointRow label={t('assistant.externalApiEnd')} method="DELETE" value={absolute.voiceEnd} />
        <EndpointRow label={t('assistant.externalApiDialogCreate')} method="POST" value={absolute.dialogCreate} />
        <EndpointRow label={t('assistant.externalApiMessages')} method="POST" value={absolute.dialogMessages} />
        <EndpointRow label={t('assistant.externalApiDialogEnd')} method="POST" value={absolute.dialogEnd} />
      </div>

      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Typography.Text bold className="!text-xs">
            {t('assistant.externalApiCurl')}
          </Typography.Text>
          <Button type="text" size="mini" icon={<IconCopy />} onClick={() => void copy('curl', curlCreate)}>
            {t('common.copy')}
          </Button>
        </div>
        <pre className="overflow-x-auto rounded-lg border border-border bg-muted/40 p-3 font-mono text-[11px] leading-relaxed text-muted-foreground">
          {curlCreate}
        </pre>
      </div>

      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Typography.Text bold className="!text-xs">
            {t('assistant.externalApiDialogCurl')}
          </Typography.Text>
          <Button type="text" size="mini" icon={<IconCopy />} onClick={() => void copy('curl-dialog', curlDialog)}>
            {t('common.copy')}
          </Button>
        </div>
        <pre className="overflow-x-auto rounded-lg border border-border bg-muted/40 p-3 font-mono text-[11px] leading-relaxed text-muted-foreground">
          {curlDialog}
        </pre>
      </div>

      <div className="space-y-1 text-xs text-muted-foreground">
        <p>{t('assistant.externalApiFlow1')}</p>
        <p>{t('assistant.externalApiFlow2')}</p>
        <p>{t('assistant.externalApiFlow3')}</p>
      </div>
    </div>
  )
}
