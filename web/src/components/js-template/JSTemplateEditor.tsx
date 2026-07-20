import React, { Suspense, useCallback, useEffect, useMemo, useState } from 'react'
import type { editor } from 'monaco-editor'
import { Alert, Drawer, Spin, Tabs, Typography } from '@arco-design/web-react'
import { IconDelete, IconPlus, IconRefresh, IconSettings } from '@arco-design/web-react/icon'
import { Button, Input } from '@/components/ui'
import { useTranslation } from '@/i18n'

const MonacoEditor = React.lazy(() => import('@monaco-editor/react'))

export interface JSTemplateCodeEditorProps {
  value: string
  onChange: (value: string) => void
  height?: string
  readOnly?: boolean
}

export interface EditorDiagnostic {
  line: number
  column: number
  message: string
  severity: 'error' | 'warning' | 'info'
}

export const JSTemplateCodeEditor: React.FC<JSTemplateCodeEditorProps> = ({
  value,
  onChange,
  height = '480px',
  readOnly = false,
}) => {
  const { t } = useTranslation()
  const [diagnostics, setDiagnostics] = useState<EditorDiagnostic[]>([])

  const handleValidate = useCallback((markers: editor.IMarker[]) => {
    setDiagnostics(
      markers.map((m) => ({
        line: m.startLineNumber,
        column: m.startColumn,
        message: m.message,
        severity: m.severity === 8 ? 'error' : m.severity === 4 ? 'warning' : 'info',
      })),
    )
  }, [])

  const errorCount = useMemo(
    () => diagnostics.filter((d) => d.severity === 'error').length,
    [diagnostics],
  )

  return (
    <div className="space-y-2">
      <div className="overflow-hidden rounded-lg border border-[var(--color-border-2)]">
        <Suspense
          fallback={
            <div className="flex h-[480px] items-center justify-center bg-[#1e1e1e]">
              <Spin tip={t('jsTemplate.editorLoading')} />
            </div>
          }
        >
          <MonacoEditor
            height={height}
            language="javascript"
            value={value}
            onChange={(v) => onChange(v || '')}
            theme="vs-dark"
            onValidate={handleValidate}
            options={{
              readOnly,
              minimap: { enabled: false },
              scrollBeyondLastLine: false,
              fontSize: 13,
              lineNumbers: 'on',
              wordWrap: 'on',
              automaticLayout: true,
              tabSize: 2,
              formatOnPaste: true,
              formatOnType: true,
              suggestOnTriggerCharacters: true,
              quickSuggestions: true,
            }}
          />
        </Suspense>
      </div>

      <div className="rounded-lg border border-[var(--color-border-2)] bg-[var(--color-fill-1)] p-3">
        <div className="mb-2 flex items-center justify-between">
          <Typography.Text bold className="!text-xs">
            {t('jsTemplate.syntaxCheck')}
          </Typography.Text>
          <Typography.Text
            type={errorCount > 0 ? 'error' : 'success'}
            className="!text-xs"
          >
            {errorCount > 0
              ? t('jsTemplate.syntaxErrorCount').replace('{count}', String(errorCount))
              : t('jsTemplate.noSyntaxErrors')}
          </Typography.Text>
        </div>
        {diagnostics.length === 0 ? (
          <Typography.Text type="secondary" className="!text-xs">
            {t('jsTemplate.syntaxHint')}
          </Typography.Text>
        ) : (
          <ul className="max-h-32 space-y-1 overflow-y-auto font-mono text-xs">
            {diagnostics.map((d, i) => (
              <li
                key={`${d.line}-${d.column}-${i}`}
                className={
                  d.severity === 'error'
                    ? 'text-[rgb(var(--danger-6))]'
                    : d.severity === 'warning'
                      ? 'text-[rgb(var(--warning-6))]'
                      : 'text-[var(--color-text-3)]'
                }
              >
                L{d.line}:C{d.column} — {d.message}
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}

export type EmbedPreviewField = {
  id: string
  key: string
  value: string
}

export type EmbedPreviewConfig = {
  fields: EmbedPreviewField[]
}

let previewFieldSeq = 0

function nextPreviewFieldId() {
  previewFieldSeq += 1
  return `pf-${previewFieldSeq}`
}

export function createDefaultEmbedPreviewConfig(apiBase: string): EmbedPreviewConfig {
  return {
    fields: [
      { id: nextPreviewFieldId(), key: 'apiBase', value: apiBase },
      { id: nextPreviewFieldId(), key: 'apiKey', value: '' },
      { id: nextPreviewFieldId(), key: 'assistantId', value: '' },
      { id: nextPreviewFieldId(), key: 'transport', value: 'text' }, // dialog/v1 text chat
    ],
  }
}

const EMBED_PREVIEW_CONFIG_STORAGE_KEY = 'lingecho.embed.preview.config'

export function loadEmbedPreviewConfig(apiBase: string): EmbedPreviewConfig {
  try {
    const raw = localStorage.getItem(EMBED_PREVIEW_CONFIG_STORAGE_KEY)
    if (!raw) return createDefaultEmbedPreviewConfig(apiBase)
    const parsed = JSON.parse(raw) as { fields?: Array<{ key?: string; value?: string }> }
    if (!Array.isArray(parsed.fields) || parsed.fields.length === 0) {
      return createDefaultEmbedPreviewConfig(apiBase)
    }
    return {
      fields: parsed.fields.map((f) => ({
        id: nextPreviewFieldId(),
        key: String(f.key ?? ''),
        value: String(f.value ?? ''),
      })),
    }
  } catch {
    return createDefaultEmbedPreviewConfig(apiBase)
  }
}

export function saveEmbedPreviewConfig(config: EmbedPreviewConfig): void {
  try {
    localStorage.setItem(
      EMBED_PREVIEW_CONFIG_STORAGE_KEY,
      JSON.stringify({
        fields: config.fields.map(({ key, value }) => ({ key, value })),
      }),
    )
  } catch {
    /* quota / private mode */
  }
}

const PRESET_FIELD_KEYS = ['token', 'title', 'autoMount'] as const

export function fieldsToEmbedConfig(fields: EmbedPreviewField[]): Record<string, unknown> {
  const out: Record<string, unknown> = {}
  for (const field of fields) {
    const key = field.key.trim()
    if (!key) continue
    const raw = field.value
    const trimmed = raw.trim()
    if (trimmed === '') continue
    if (trimmed === 'true') {
      out[key] = true
    } else if (trimmed === 'false') {
      out[key] = false
    } else if (/^-?\d+(\.\d+)?$/.test(trimmed) && key !== 'assistantId' && key !== 'apiKey') {
      out[key] = Number(trimmed)
    } else {
      out[key] = raw
    }
  }
  if (out.autoMount === undefined) {
    out.autoMount = true
  }
  return out
}

function clonePreviewConfig(config: EmbedPreviewConfig): EmbedPreviewConfig {
  return {
    fields: config.fields.map((f) => ({ ...f })),
  }
}

function buildPreviewDoc(content: string, config: EmbedPreviewConfig) {
  const escaped = content.replace(/<\/script/gi, '<\\/script')
  const configJson = JSON.stringify(fieldsToEmbedConfig(config.fields))

  return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <style>
    body { margin: 0; font-family: system-ui, sans-serif; background: #f5f5f5; min-height: 100vh; }
    #preview-root { min-height: 100vh; }
    .preview-error { color: #c0392b; padding: 16px; font-family: monospace; white-space: pre-wrap; }
  </style>
</head>
<body>
  <div id="preview-root"></div>
  <script>
    (function () {
      window.__LingEchoConfig = ${configJson};

      var nativeFetch = window.fetch.bind(window);
      window.fetch = function (input, init) {
        return nativeFetch(input, init).then(function (res) {
          if (!res || typeof res.clone !== 'function') return res;
          return res.clone().text().then(function (text) {
            if (text && text.trim()) return res;
            return {
              ok: res.ok,
              status: res.status,
              statusText: res.statusText,
              json: function () {
                return Promise.resolve({
                  code: res.status || 500,
                  msg: 'Empty API response — check API Base, API Key, and Assistant ID',
                });
              },
              text: function () { return Promise.resolve(''); },
            };
          });
        });
      };
    })();
  </script>
  <script>
    try {
      ${escaped}
    } catch (e) {
      document.getElementById('preview-root').innerHTML =
        '<pre class="preview-error">Preview error:\\n' + (e && e.message ? e.message : String(e)) + '</pre>';
    }
  </script>
</body>
</html>`
}

function EmbedPreviewConfigDrawer({
  visible,
  draft,
  onDraftChange,
  onClose,
  onApply,
}: {
  visible: boolean
  draft: EmbedPreviewConfig
  onDraftChange: (next: EmbedPreviewConfig) => void
  onClose: () => void
  onApply: () => void
}) {
  const { t } = useTranslation()
  const configPreview = useMemo(
    () => JSON.stringify(fieldsToEmbedConfig(draft.fields), null, 2),
    [draft.fields],
  )

  const existingKeys = useMemo(
    () => new Set(draft.fields.map((f) => f.key.trim()).filter(Boolean)),
    [draft.fields],
  )

  const addField = (key = '', value = '') => {
    onDraftChange({
      fields: [...draft.fields, { id: nextPreviewFieldId(), key, value }],
    })
  }

  const updateField = (id: string, patch: Partial<Pick<EmbedPreviewField, 'key' | 'value'>>) => {
    onDraftChange({
      fields: draft.fields.map((f) => (f.id === id ? { ...f, ...patch } : f)),
    })
  }

  const removeField = (id: string) => {
    onDraftChange({ fields: draft.fields.filter((f) => f.id !== id) })
  }

  return (
    <Drawer
      width={420}
      title={t('jsTemplate.preview.configTitle')}
      visible={visible}
      onCancel={onClose}
      footer={
        <div className="flex justify-end gap-2">
          <Button onClick={onClose}>{t('common.cancel')}</Button>
          <Button type="primary" onClick={onApply}>
            {t('jsTemplate.preview.applyConfig')}
          </Button>
        </div>
      }
    >
      <div className="space-y-4">
        <Typography.Text type="secondary" className="!text-xs">
          {t('jsTemplate.preview.configDesc')}
        </Typography.Text>

        <div className="space-y-2">
          {draft.fields.map((field) => (
            <div key={field.id} className="flex items-start gap-2">
              <Input
                value={field.key}
                onChange={(v) => updateField(field.id, { key: v })}
                placeholder={t('jsTemplate.preview.fieldKey')}
                className="!w-[38%]"
              />
              <Input
                value={field.value}
                onChange={(v) => updateField(field.id, { value: v })}
                placeholder={t('jsTemplate.preview.fieldValue')}
                className="flex-1"
              />
              <Button
                type="text"
                icon={<IconDelete />}
                onClick={() => removeField(field.id)}
                aria-label={t('common.delete')}
              />
            </div>
          ))}
        </div>

        <div className="flex flex-wrap gap-2">
          <Button type="outline" size="sm" icon={<IconPlus />} onClick={() => addField()}>
            {t('jsTemplate.preview.addField')}
          </Button>
          {PRESET_FIELD_KEYS.filter((k) => !existingKeys.has(k)).map((key) => (
            <Button
              key={key}
              type="text"
              size="sm"
              onClick={() => addField(key, key === 'autoMount' ? 'true' : '')}
            >
              + {key}
            </Button>
          ))}
        </div>

        <div>
          <Typography.Text bold className="!mb-2 !block !text-xs">
            {t('jsTemplate.preview.configJson')}
          </Typography.Text>
          <pre className="max-h-48 overflow-auto rounded-lg border border-[var(--color-border-2)] bg-[var(--color-fill-1)] p-3 font-mono text-xs">
            {configPreview}
          </pre>
        </div>
      </div>
    </Drawer>
  )
}

export interface JSTemplatePreviewProps {
  content: string
  previewConfig: EmbedPreviewConfig
  onPreviewConfigChange: (config: EmbedPreviewConfig) => void
}

export const JSTemplatePreview: React.FC<JSTemplatePreviewProps> = ({
  content,
  previewConfig,
  onPreviewConfigChange,
}) => {
  const { t } = useTranslation()
  const [previewKey, setPreviewKey] = useState(0)
  const [configOpen, setConfigOpen] = useState(false)
  const [draftConfig, setDraftConfig] = useState<EmbedPreviewConfig>(() => clonePreviewConfig(previewConfig))

  useEffect(() => {
    if (configOpen) {
      setDraftConfig(clonePreviewConfig(previewConfig))
    }
  }, [configOpen, previewConfig])

  const previewDoc = useMemo(
    () => buildPreviewDoc(content, previewConfig),
    [content, previewConfig],
  )

  const applyConfig = () => {
    const next = clonePreviewConfig(draftConfig)
    onPreviewConfigChange(next)
    saveEmbedPreviewConfig(next)
    setConfigOpen(false)
    setPreviewKey((k) => k + 1)
  }

  return (
    <div className="flex h-full min-h-0 flex-col gap-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <Alert type="info" content={t('jsTemplate.preview.hint')} className="!flex-1 !text-xs" />
        <div className="flex shrink-0 gap-2">
          <Button
            type="outline"
            size="sm"
            icon={<IconRefresh />}
            onClick={() => setPreviewKey((k) => k + 1)}
          >
            {t('jsTemplate.preview.reloadPreview')}
          </Button>
          <Button
            type="outline"
            size="sm"
            icon={<IconSettings />}
            onClick={() => setConfigOpen(true)}
          >
            {t('jsTemplate.preview.openConfig')}
          </Button>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-hidden rounded-lg border border-[var(--color-border-2)] bg-white">
        <iframe
          key={previewKey}
          title="JS template preview"
          srcDoc={previewDoc}
          sandbox="allow-scripts allow-same-origin"
          className="h-full min-h-[520px] w-full border-0"
        />
      </div>

      <EmbedPreviewConfigDrawer
        visible={configOpen}
        draft={draftConfig}
        onDraftChange={setDraftConfig}
        onClose={() => setConfigOpen(false)}
        onApply={applyConfig}
      />
    </div>
  )
}

export interface JSTemplateEditorSplitProps {
  content: string
  onChange: (value: string) => void
  editorHeight?: string
  previewConfig: EmbedPreviewConfig
  onPreviewConfigChange: (config: EmbedPreviewConfig) => void
  contentLoading?: boolean
  deferPreview?: boolean
}

export const JSTemplateEditorSplit: React.FC<JSTemplateEditorSplitProps> = ({
  content,
  onChange,
  editorHeight = 'calc(100vh - 120px)',
  previewConfig,
  onPreviewConfigChange,
  contentLoading = false,
  deferPreview = false,
}) => {
  const { t } = useTranslation()
  const [previewReady, setPreviewReady] = useState(!deferPreview)

  useEffect(() => {
    if (!deferPreview) {
      setPreviewReady(true)
      return
    }
    const timer = window.setTimeout(() => setPreviewReady(true), 400)
    return () => window.clearTimeout(timer)
  }, [deferPreview])

  return (
    <div className="grid h-full min-h-0 grid-cols-1 gap-0 lg:grid-cols-2">
      <div className="min-h-0 border-b border-[var(--color-border-2)] p-4 lg:border-b-0 lg:border-r">
        <Typography.Text bold className="!mb-3 !block !text-sm">
          {t('jsTemplate.codeEditor')}
          {contentLoading ? (
            <Typography.Text type="secondary" className="!ml-2 !text-xs">
              ({t('common.loading')})
            </Typography.Text>
          ) : null}
        </Typography.Text>
        <JSTemplateCodeEditor value={content} onChange={onChange} height={editorHeight} />
      </div>
      <div className="flex min-h-0 flex-col p-4">
        <Typography.Text bold className="!mb-3 !block !text-sm">
          {t('jsTemplate.livePreview')}
        </Typography.Text>
        {previewReady ? (
          <JSTemplatePreview
            content={content}
            previewConfig={previewConfig}
            onPreviewConfigChange={onPreviewConfigChange}
          />
        ) : (
          <div className="flex min-h-[520px] flex-1 items-center justify-center rounded-lg border border-[var(--color-border-2)] bg-[var(--color-fill-1)]">
            <Spin tip={t('jsTemplate.previewLoading')} />
          </div>
        )}
      </div>
    </div>
  )
}

export interface JSTemplateEditorTabsProps {
  content: string
  onChange: (value: string) => void
  previewConfig: EmbedPreviewConfig
  onPreviewConfigChange: (config: EmbedPreviewConfig) => void
}

export const JSTemplateEditorTabs: React.FC<JSTemplateEditorTabsProps> = ({
  content,
  onChange,
  previewConfig,
  onPreviewConfigChange,
}) => {
  const { t } = useTranslation()

  return (
    <Tabs defaultActiveTab="code">
      <Tabs.TabPane key="code" title={t('jsTemplate.codeEditor')}>
        <JSTemplateCodeEditor value={content} onChange={onChange} />
      </Tabs.TabPane>
      <Tabs.TabPane key="preview" title={t('jsTemplate.livePreview')}>
        <JSTemplatePreview
          content={content}
          previewConfig={previewConfig}
          onPreviewConfigChange={onPreviewConfigChange}
        />
      </Tabs.TabPane>
    </Tabs>
  )
}
