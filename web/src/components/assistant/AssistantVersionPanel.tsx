import { useMemo, useState } from 'react'
import { Collapse, Descriptions, Drawer, Modal, Select, Tag, Typography } from '@arco-design/web-react'
import { Button } from '@/components/ui'
import type { AssistantVersionRow } from '@/api/assistants'
import { diffAssistantVersions } from '@/api/assistants'
import StructuredFieldDiff from '@/components/diff/StructuredFieldDiff'
import { useTranslation } from '@/i18n'
import { showAlert } from '@/utils/notification'

const CollapseItem = Collapse.Item

function formatJsonBlock(raw: unknown): string {
  if (raw == null || raw === '') return '—'
  if (typeof raw === 'string') {
    try {
      return JSON.stringify(JSON.parse(raw), null, 2)
    } catch {
      return raw
    }
  }
  try {
    return JSON.stringify(raw, null, 2)
  } catch {
    return String(raw)
  }
}

function truncateText(text: string, max = 280) {
  const s = text.trim()
  if (s.length <= max) return s
  return `${s.slice(0, max)}…`
}

function customToolIdsFromAgentConfig(raw: unknown): string[] {
  if (raw == null) return []
  let obj: Record<string, unknown>
  if (typeof raw === 'string') {
    try {
      obj = JSON.parse(raw) as Record<string, unknown>
    } catch {
      return []
    }
  } else if (typeof raw === 'object' && !Array.isArray(raw)) {
    obj = raw as Record<string, unknown>
  } else {
    return []
  }
  const ids = obj.customToolIds ?? obj.custom_tool_ids
  if (!Array.isArray(ids)) return []
  return ids.map((x) => String(x).trim()).filter(Boolean)
}

function versionSnapshot(v: AssistantVersionRow): Record<string, unknown> {
  return {
    version: v.version,
    versionNo: v.versionNo,
    scene: v.scene,
    description: v.description,
    welcome: v.welcome,
    prompt: v.prompt,
    knowledgeNamespace: v.knowledgeNamespace,
    agentConfig: v.agentConfig,
    vadConfig: v.vadConfig,
    hotWords: v.hotWords,
    interruptionConfig: v.interruptionConfig,
    audioTrackConfig: v.audioTrackConfig,
    audioProcessConfig: v.audioProcessConfig,
    queryRewriter: v.queryRewriter,
    mcpServers: v.mcpServers,
    ttsVoice: v.ttsVoice,
    realtimeVoice: v.realtimeVoice,
    voiceDialogWsUrl: v.voiceDialogWsUrl,
  }
}

export type AssistantVersionPanelProps = {
  assistantId?: string
  versions: AssistantVersionRow[]
  publishedVersionId?: string
  onRollback: (versionId: string) => Promise<void>
  publishing?: boolean
  onPublish?: () => void
}

export default function AssistantVersionPanel({
  assistantId,
  versions,
  publishedVersionId,
  onRollback,
  publishing,
  onPublish,
}: AssistantVersionPanelProps) {
  const { t } = useTranslation()
  const [detailOpen, setDetailOpen] = useState(false)
  const [selected, setSelected] = useState<AssistantVersionRow | null>(null)
  const [rollingBack, setRollingBack] = useState(false)
  const [diffOpen, setDiffOpen] = useState(false)
  const [diffLoading, setDiffLoading] = useState(false)
  const [compareToId, setCompareToId] = useState<string>('')
  const [changedKeys, setChangedKeys] = useState<string[]>([])
  const [leftSnap, setLeftSnap] = useState<unknown>(null)
  const [rightSnap, setRightSnap] = useState<unknown>(null)
  const [leftTitle, setLeftTitle] = useState('')
  const [rightTitle, setRightTitle] = useState('')

  const selectedLabel = useMemo(() => {
    if (!selected) return ''
    return selected.version || `#${selected.versionNo ?? selected.id}`
  }, [selected])

  const selectedBoundCatalogIds = useMemo(
    () => (selected ? customToolIdsFromAgentConfig(selected.agentConfig) : []),
    [selected],
  )

  const selectedHasMcpServers = useMemo(() => {
    if (!selected) return false
    const raw = (selected as Record<string, unknown>).mcpServers
    if (raw == null || raw === '') return false
    if (typeof raw === 'string') {
      const t = raw.trim()
      if (!t || t === '[]' || t === 'null') return false
      try {
        const parsed = JSON.parse(t) as unknown
        return Array.isArray(parsed) && parsed.length > 0
      } catch {
        return true
      }
    }
    return Array.isArray(raw) && raw.length > 0
  }, [selected])

  const openDetail = (v: AssistantVersionRow) => {
    setSelected(v)
    setDetailOpen(true)
  }

  const confirmRollback = (v: AssistantVersionRow) => {
    const label = v.version || `#${v.versionNo ?? v.id}`
    Modal.confirm({
      title: t('assistant.versionRollbackConfirmTitle'),
      content: t('assistant.versionRollbackConfirmContent', { version: label }),
      okText: t('common.rollback'),
      cancelText: t('common.cancel'),
      okButtonProps: { status: 'warning' },
      onOk: async () => {
        setRollingBack(true)
        try {
          await onRollback(v.id)
          setDetailOpen(false)
        } finally {
          setRollingBack(false)
        }
      },
    })
  }

  const openDraftDiff = async () => {
    if (!assistantId) return
    setDiffLoading(true)
    setDiffOpen(true)
    setLeftTitle(t('assistant.diffLeft'))
    setRightTitle(t('assistant.diffRight'))
    try {
      const res = await diffAssistantVersions(assistantId)
      if (res.code !== 200 || !res.data) {
        showAlert(res.msg || 'diff failed', 'error')
        return
      }
      setChangedKeys(res.data.changedKeys || [])
      setLeftSnap(res.data.fromSnapshot ?? null)
      setRightSnap(res.data.toSnapshot ?? null)
    } catch {
      showAlert('diff failed', 'error')
    } finally {
      setDiffLoading(false)
    }
  }

  const openVersionDiff = async (fromId: string, toId: string) => {
    if (!assistantId || !fromId || !toId) return
    setDiffLoading(true)
    setDiffOpen(true)
    const fromV = versions.find((v) => String(v.id) === String(fromId))
    const toV = versions.find((v) => String(v.id) === String(toId))
    setLeftTitle(fromV?.version || `#${fromId}`)
    setRightTitle(toV?.version || `#${toId}`)
    try {
      const res = await diffAssistantVersions(assistantId, { from: fromId, to: toId })
      if (res.code !== 200 || !res.data) {
        // Fallback: local snapshot compare when API omits blobs
        if (fromV && toV) {
          setChangedKeys([])
          setLeftSnap(versionSnapshot(fromV))
          setRightSnap(versionSnapshot(toV))
          return
        }
        showAlert(res.msg || 'diff failed', 'error')
        return
      }
      setChangedKeys(res.data.changedKeys || [])
      setLeftSnap(res.data.fromSnapshot ?? (fromV ? versionSnapshot(fromV) : null))
      setRightSnap(res.data.toSnapshot ?? (toV ? versionSnapshot(toV) : null))
    } catch {
      if (fromV && toV) {
        setLeftSnap(versionSnapshot(fromV))
        setRightSnap(versionSnapshot(toV))
      } else {
        showAlert('diff failed', 'error')
      }
    } finally {
      setDiffLoading(false)
    }
  }

  return (
    <>
      <div className="rounded-xl border border-border bg-card p-5 space-y-3">
        <Typography.Paragraph type="secondary" style={{ marginBottom: 8, fontSize: 13 }}>
          {t('assistant.publishDesc')}
        </Typography.Paragraph>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <Typography.Text bold>{t('assistant.publishRecord')}</Typography.Text>
          <div className="flex flex-wrap items-center gap-2">
            {assistantId ? (
              <Button size="small" type="outline" loading={diffLoading} onClick={() => void openDraftDiff()}>
                {t('assistant.compareWithPublished')}
              </Button>
            ) : null}
            {onPublish ? (
              <Button size="small" type="primary" loading={publishing} onClick={onPublish}>
                {t('common.publishVersion')}
              </Button>
            ) : null}
          </div>
        </div>

        {versions.length >= 2 ? (
          <div className="flex flex-wrap items-center gap-2 rounded-lg border border-dashed border-border px-3 py-2">
            <span className="text-xs text-muted-foreground">{t('assistant.comparePick')}</span>
            <Select
              size="small"
              style={{ width: 160 }}
              placeholder="from"
              value={compareToId || undefined}
              onChange={(v) => setCompareToId(String(v || ''))}
              options={versions.map((v) => ({
                value: String(v.id),
                label: v.version || `#${v.versionNo ?? v.id}`,
              }))}
            />
            <Button
              size="mini"
              type="outline"
              disabled={!compareToId || !publishedVersionId}
              onClick={() => void openVersionDiff(compareToId, String(publishedVersionId))}
            >
              {t('assistant.compareVersions')}
            </Button>
          </div>
        ) : null}

        {versions.length === 0 ? (
          <Typography.Text type="secondary">{t('assistant.noPublishRecord')}</Typography.Text>
        ) : (
          <div className="divide-y divide-border rounded-lg border border-border">
            {versions.map((v) => {
              const label = v.version || `#${v.versionNo ?? v.id}`
              const isCurrent = publishedVersionId && String(v.id) === String(publishedVersionId)
              return (
                <div key={v.id} className="flex flex-wrap items-center justify-between gap-2 px-3 py-2.5">
                  <div className="min-w-0 space-y-0.5">
                    <div className="flex flex-wrap items-center gap-2 text-sm">
                      <span className="font-medium">{label}</span>
                      {isCurrent ? <Tag size="small" color="green">{t('common.published')}</Tag> : null}
                    </div>
                    <div className="text-xs text-muted-foreground">
                      {v.publishedAt ? new Date(v.publishedAt).toLocaleString() : ''}
                      {v.publishedBy ? ` · ${v.publishedBy}` : ''}
                    </div>
                  </div>
                  <div className="flex items-center gap-1.5">
                    <Button size="mini" type="outline" onClick={() => openDetail(v)}>
                      {t('assistant.viewVersionConfig')}
                    </Button>
                    <Button
                      size="mini"
                      type="outline"
                      status="warning"
                      disabled={isCurrent}
                      onClick={() => confirmRollback(v)}
                    >
                      {t('common.rollback')}
                    </Button>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>

      <Drawer
        width={960}
        title={t('assistant.compareVersions')}
        visible={diffOpen}
        onCancel={() => setDiffOpen(false)}
        footer={null}
      >
        {diffLoading ? (
          <Typography.Text type="secondary">{t('common.loading')}</Typography.Text>
        ) : (
          <div className="space-y-3">
            {changedKeys.length > 0 ? (
              <div className="text-xs text-muted-foreground">
                {t('assistant.diffChangedKeys')}: {changedKeys.join(', ')}
              </div>
            ) : null}
            <StructuredFieldDiff
              left={leftSnap}
              right={rightSnap}
              leftTitle={leftTitle}
              rightTitle={rightTitle}
              changedKeys={changedKeys}
              onlyChanged
            />
          </div>
        )}
      </Drawer>

      <Drawer
        width={560}
        title={t('assistant.versionConfigTitle', { version: selectedLabel })}
        visible={detailOpen}
        onCancel={() => setDetailOpen(false)}
        footer={
          selected ? (
            <div className="flex justify-end gap-2">
              <Button onClick={() => setDetailOpen(false)}>{t('common.cancel')}</Button>
              <Button
                type="primary"
                status="warning"
                loading={rollingBack}
                disabled={!!publishedVersionId && String(selected.id) === String(publishedVersionId)}
                onClick={() => confirmRollback(selected)}
              >
                {t('assistant.rollbackToVersion')}
              </Button>
            </div>
          ) : null
        }
      >
        {selected ? (
          <div className="space-y-4">
            <Descriptions
              column={1}
              size="small"
              border
              data={[
                { label: t('common.version'), value: selected.version || `#${selected.versionNo ?? selected.id}` },
                {
                  label: t('assistant.versionPublishedAt'),
                  value: selected.publishedAt ? new Date(selected.publishedAt).toLocaleString() : '—',
                },
                { label: t('assistant.versionPublishedBy'), value: selected.publishedBy || '—' },
                { label: t('assistant.versionScene'), value: selected.scene || '—' },
                { label: t('common.description'), value: selected.description || '—' },
              ]}
            />

            <div className="space-y-2">
              <Typography.Text bold>{t('assistant.tabDialog')}</Typography.Text>
              <div className="rounded-lg border border-border p-3 space-y-2 text-sm">
                <div>
                  <div className="text-xs text-muted-foreground mb-1">开场白</div>
                  <div className="whitespace-pre-wrap">{selected.welcome?.trim() || '—'}</div>
                </div>
                <div>
                  <div className="text-xs text-muted-foreground mb-1">提示词</div>
                  <div className="whitespace-pre-wrap">{truncateText(selected.prompt || '', 600) || '—'}</div>
                </div>
                <div>
                  <div className="text-xs text-muted-foreground mb-1">知识库</div>
                  <div>{selected.knowledgeNamespace?.trim() || '—'}</div>
                </div>
              </div>
            </div>

            <div className="space-y-2">
              <Typography.Text bold>{t('assistant.tabVoice')}</Typography.Text>
              <Descriptions
                column={1}
                size="small"
                border
                data={[
                  { label: 'TTS', value: selected.ttsVoice || '—' },
                  { label: 'Realtime', value: selected.realtimeVoice || '—' },
                  { label: 'WS', value: selected.voiceDialogWsUrl || '—' },
                ]}
              />
            </div>

            <div className="space-y-2">
              <Typography.Text bold>{t('assistant.versionCatalogTools')}</Typography.Text>
              <div className="rounded-lg border border-border p-3 text-sm">
                {selectedBoundCatalogIds.length === 0 ? (
                  <span className="text-muted-foreground text-xs">{t('assistant.versionCatalogToolsEmpty')}</span>
                ) : (
                  <div className="flex flex-wrap gap-1.5">
                    {selectedBoundCatalogIds.map((id) => (
                      <Tag key={id} size="small" color="arcoblue">{id}</Tag>
                    ))}
                  </div>
                )}
                <Typography.Paragraph type="secondary" style={{ marginTop: 8, marginBottom: 0, fontSize: 11 }}>
                  {t('assistant.versionCatalogToolsHint')}
                </Typography.Paragraph>
              </div>
            </div>

            {!selectedHasMcpServers && selectedBoundCatalogIds.length > 0 ? (
              <div className="rounded-lg border border-dashed border-border bg-muted/20 px-3 py-2 text-xs text-muted-foreground">
                {t('assistant.versionMcpServersSkipped')}
              </div>
            ) : null}

            <Collapse>
              {(
                [
                  ['agentConfig', t('assistant.versionAgentConfig')],
                  ['vadConfig', 'VAD'],
                  ['hotWords', 'HotWords'],
                  ['interruptionConfig', 'Interrupt'],
                  ['audioTrackConfig', 'AudioTrack'],
                  ['audioProcessConfig', 'AudioProcess'],
                  ['queryRewriter', 'QueryRewriter'],
                  ...(selectedHasMcpServers
                    ? ([['mcpServers', t('assistant.mcpToolsAdvanced')]] as const)
                    : []),
                ] as const
              ).map(([key, title]) => (
                <CollapseItem key={key} header={title} name={key}>
                  <pre className="max-h-48 overflow-auto whitespace-pre-wrap break-all text-[11px]">
                    {formatJsonBlock((selected as Record<string, unknown>)[key])}
                  </pre>
                </CollapseItem>
              ))}
            </Collapse>
          </div>
        ) : null}
      </Drawer>
    </>
  )
}
