import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Switch, Tabs, Tag, Typography, Alert } from '@arco-design/web-react'
import { Input, Link, Select } from '@/components/ui'
import { Loading } from '@/components/ui/loading'
import BaseLayout from '@/components/Layout/BaseLayout'
import AssistantPageHeader from '@/pages/assistants/AssistantPageHeader'
import {
  ASSISTANT_SCENES,
  createAssistant,
  diffAssistantVersions,
  getAssistant,
  listAssistantVersions,
  publishAssistant,
  rollbackAssistant,
  updateAssistant,
  type AssistantVersionRow,
} from '@/api/assistants'
import { listKnowledgeNamespaces } from '@/api/knowledgeNamespaces'
import { listNluModels } from '@/api/nluModels'
import { bindVoiceprintAssistant, listVoiceprints, snowflakeStr } from '@/api/voiceprint'
import { useSiteConfig } from '@/contexts/siteConfig'
import {
  agentConfigFromJSON,
  agentConfigToJSON,
  audioProcessConfigFromJSON,
  audioProcessConfigToJSON,
  audioTrackConfigFromJSON,
  audioTrackConfigToJSON,
  defaultAgentConfig,
  defaultAudioProcessConfig,
  defaultAudioTrackConfig,
  defaultInterruptionConfig,
  defaultMcpServers,
  defaultQueryRewriter,
  defaultVadConfig,
  hotWordsFromJSON,
  hotWordsToJSON,
  interruptionConfigFromJSON,
  interruptionConfigToJSON,
  mcpServersFromJSON,
  mcpServersToJSON,
  queryRewriterFromJSON,
  queryRewriterToJSON,
  type AgentConfigDraft,
  type AudioProcessConfigDraft,
  type AudioTrackConfigDraft,
  type HotWordDraft,
  type InterruptionConfigDraft,
  type McpServerDraft,
  type QueryRewriterDraft,
  type VadConfigDraft,
  vadConfigFromJSON,
  vadConfigToJSON,
} from '@/constants/assistantAdvancedConfig'
import { getTenantVoiceProviders } from '@/api/voices'
import { listVoiceClones, type VoiceCloneProfile } from '@/api/voiceClone'
import { listCredentials, type CredentialRow } from '@/api/credentials'
import { useAuthStore } from '@/stores/authStore'
import { showAlert } from '@/utils/notification'
import { useUnsavedLeaveGuard } from '@/hooks/useUnsavedLeaveGuard'
import { useTranslation } from '@/i18n'
import { applyAssistantTemplate, getAssistantTemplate } from '@/constants/assistantTemplates'
import AssistantVersionPanel from '@/components/assistant/AssistantVersionPanel'
import AssistantExternalAPIPanel from '@/components/assistant/AssistantExternalAPIPanel'
import AssistantEmbedPanel from '@/components/assistant/AssistantEmbedPanel'
import AssistantSettingsModal from '@/components/assistant/AssistantSettingsModal'
import AssistantFileUploadModal, { AssistantFileUploadRow } from '@/components/assistant/AssistantFileUploadModal'
import { AssistantDebugPanel } from '@/pages/assistants/AssistantDebugPage'
import { cn } from '@/utils/cn'
import { defaultVoiceProvider } from '@/constants/tenantAiConfigRules'
import {
  AssistantBehaviorFields,
  AssistantDialogFields,
  AssistantHotWordsFields,
  AssistantInterruptionFields,
  AssistantAudioTrackFields,
  AssistantAudioProcessFields,
  AssistantQueryRewriterFields,
  AssistantCatalogToolBindFields,
  AssistantDialogSkillBindFields,
  AssistantVoiceprintBindFields,
  AssistantVoiceFields,
  AssistantVadFields,
} from '@/pages/assistants/assistantFormFields'

const TabPane = Tabs.TabPane

export type AssistantManagerFormPageProps = {
  mode: 'create' | 'edit'
  assistantId?: string
  scopeTenantId?: string
  templateId?: string
  listPath: string
}

export default function AssistantManagerFormPage({
  mode,
  assistantId,
  scopeTenantId,
  templateId,
  listPath,
}: AssistantManagerFormPageProps) {
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const { t } = useTranslation()
  const authUser = useAuthStore((s) => s.user)
  const { config: siteConfig } = useSiteConfig()
  const nluEnabled = Boolean(siteConfig.nluEnabled)
  const voiceprintEnabled = Boolean(siteConfig.VOICEPRINT_PROVIDER?.trim())
  const isEdit = mode === 'edit'

  const [loading, setLoading] = useState(isEdit)
  const [name, setName] = useState('')
  const [scene, setScene] = useState<string>('general')
  const [description, setDescription] = useState('')
  const [enabled, setEnabled] = useState(true)
  const [professionalMode, setProfessionalMode] = useState(false)
  const [tenantVoiceMode, setTenantVoiceMode] = useState<'pipeline' | 'realtime'>('pipeline')
  const [welcome, setWelcome] = useState('')
  const [prompt, setPrompt] = useState('')
  const [knowledgeNamespace, setKnowledgeNamespace] = useState('')
  const [knowledgeOptions, setKnowledgeOptions] = useState<{ value: string; label: string }[]>([])
  const [nluModelId, setNluModelId] = useState('')
  const [nluOptions, setNluOptions] = useState<{ value: string; label: string }[]>([])
  const [voiceprintIds, setVoiceprintIds] = useState<string[]>([])
  const [agent, setAgent] = useState<AgentConfigDraft>(() => defaultAgentConfig())
  const [vad, setVad] = useState<VadConfigDraft>(() => defaultVadConfig())
  const [hotWords, setHotWords] = useState<HotWordDraft[]>([])
  const [interruption, setInterruption] = useState<InterruptionConfigDraft>(() => defaultInterruptionConfig())
  const [audioTrack, setAudioTrack] = useState<AudioTrackConfigDraft>(() => defaultAudioTrackConfig())
  const [audioProcess, setAudioProcess] = useState<AudioProcessConfigDraft>(() => defaultAudioProcessConfig())
  const [queryRewriter, setQueryRewriter] = useState<QueryRewriterDraft>(() => defaultQueryRewriter())
  const [mcpServers, setMcpServers] = useState<McpServerDraft[]>(() => defaultMcpServers())
  const [ttsVoice, setTtsVoice] = useState('')
  const [realtimeVoice, setRealtimeVoice] = useState('')
  const [selectedCloneProfile, setSelectedCloneProfile] = useState<VoiceCloneProfile | null>(null)
  const [cloneVoiceIds, setCloneVoiceIds] = useState<Set<string>>(() => new Set())
  const [tenantTtsProvider, setTenantTtsProvider] = useState('')
  const [tenantRealtimeProvider, setTenantRealtimeProvider] = useState('')
  const [poolTtsVoiceIds, setPoolTtsVoiceIds] = useState<string[]>([])
  const [poolRealtimeVoiceIds, setPoolRealtimeVoiceIds] = useState<string[]>([])
  const [credentialId, setCredentialId] = useState('')
  const [boundCredentialId, setBoundCredentialId] = useState('')
  const [credentials, setCredentials] = useState<CredentialRow[]>([])
  const [credsLoaded, setCredsLoaded] = useState(false)
  const [voiceDialogWsUrl, setVoiceDialogWsUrl] = useState('')
  const [boundJsTemplateSourceId, setBoundJsTemplateSourceId] = useState('')
  const [debugOpen, setDebugOpen] = useState(() => searchParams.get('debug') === '1')
  const [debugRendered, setDebugRendered] = useState(() => searchParams.get('debug') === '1')
  const [debugVisible, setDebugVisible] = useState(() => searchParams.get('debug') === '1')
  const [tenantId, setTenantId] = useState('')
  const [saving, setSaving] = useState(false)
  const [publishing, setPublishing] = useState(false)
  const [versions, setVersions] = useState<AssistantVersionRow[]>([])
  const [publishedVersionId, setPublishedVersionId] = useState('')
  const [draftDiffKeys, setDraftDiffKeys] = useState<string[]>([])
  const [baselineSig, setBaselineSig] = useState('')
  const [activeTab, setActiveTab] = useState('dialog')
  const [avatarUrl, setAvatarUrl] = useState('')
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [fileUploadOpen, setFileUploadOpen] = useState(false)

  const isPublished = !!publishedVersionId.trim()

  const effectiveTenantId = (scopeTenantId?.trim() || tenantId.trim() || String(authUser?.tenantId ?? '')) || ''

  const voicePickerValue = useMemo(() => {
    const tv = ttsVoice.trim()
    if (tv && (cloneVoiceIds.has(tv) || selectedCloneProfile)) return tv
    return tenantVoiceMode === 'realtime' ? realtimeVoice.trim() : tv
  }, [ttsVoice, realtimeVoice, tenantVoiceMode, selectedCloneProfile, cloneVoiceIds])

  const selectedCredential = useMemo(
    () => credentials.find((c) => String(c.id) === String(credentialId)) || null,
    [credentials, credentialId],
  )

  const isPlatformCredential = Boolean(
    selectedCredential &&
      (selectedCredential.kind === 'platform_bundle' || selectedCredential.usesTenantAi),
  )

  // 音色目录跟「当前 API Key 对应的厂商」走：
  // - 平台托管 Key → 号池厂商 / 号池绑定音色
  // - 用户自定义 Key → 该 Key 自己的 TTS/Realtime provider（厂商不同则音色 ID 不可通用）
  const voiceProvider = useMemo(() => {
    if (selectedCredential && !isPlatformCredential) {
      const cfg =
        tenantVoiceMode === 'realtime'
          ? selectedCredential.realtimeConfig
          : selectedCredential.ttsConfig
      const fromKey = String((cfg as { provider?: string } | null | undefined)?.provider || '')
        .trim()
        .toLowerCase()
      if (fromKey) return fromKey
    }
    const fromTenant =
      tenantVoiceMode === 'realtime' ? tenantRealtimeProvider : tenantTtsProvider
    return (
      fromTenant.trim() ||
      defaultVoiceProvider(tenantVoiceMode === 'realtime' ? 'realtime' : 'tts')
    )
  }, [
    selectedCredential,
    isPlatformCredential,
    tenantVoiceMode,
    tenantTtsProvider,
    tenantRealtimeProvider,
  ])

  const allowedVoiceIds = useMemo(() => {
    // 用户 Key：展示该厂商完整目录（不做号池 allow-list）
    if (selectedCredential && !isPlatformCredential) return undefined
    const ids = tenantVoiceMode === 'realtime' ? poolRealtimeVoiceIds : poolTtsVoiceIds
    return ids.length > 0 ? ids : undefined
  }, [
    selectedCredential,
    isPlatformCredential,
    tenantVoiceMode,
    poolTtsVoiceIds,
    poolRealtimeVoiceIds,
  ])

  const providerOfCredential = useCallback(
    (cred: CredentialRow | null | undefined) => {
      if (!cred) {
        const fromTenant =
          tenantVoiceMode === 'realtime' ? tenantRealtimeProvider : tenantTtsProvider
        return (
          fromTenant.trim() ||
          defaultVoiceProvider(tenantVoiceMode === 'realtime' ? 'realtime' : 'tts')
        )
      }
      const platform = cred.kind === 'platform_bundle' || cred.usesTenantAi
      if (platform) {
        const fromTenant =
          tenantVoiceMode === 'realtime' ? tenantRealtimeProvider : tenantTtsProvider
        return (
          fromTenant.trim() ||
          defaultVoiceProvider(tenantVoiceMode === 'realtime' ? 'realtime' : 'tts')
        )
      }
      const cfg =
        tenantVoiceMode === 'realtime' ? cred.realtimeConfig : cred.ttsConfig
      const fromKey = String((cfg as { provider?: string } | null | undefined)?.provider || '')
        .trim()
        .toLowerCase()
      return (
        fromKey ||
        defaultVoiceProvider(tenantVoiceMode === 'realtime' ? 'realtime' : 'tts')
      )
    },
    [tenantVoiceMode, tenantTtsProvider, tenantRealtimeProvider],
  )

  const onCredentialChange = (nextId: string) => {
    const next = String(nextId || '')
    const prevProv = providerOfCredential(selectedCredential)
    const nextCred = credentials.find((c) => String(c.id) === next) || null
    const nextProv = providerOfCredential(nextCred)
    setCredentialId(next)
    if (prevProv && nextProv && prevProv !== nextProv) {
      setTtsVoice('')
      setRealtimeVoice('')
      setSelectedCloneProfile(null)
      showAlert(t('assistant.voiceClearedOnKeyChange'), 'warning')
    }
  }

  useEffect(() => {
    const fromAuth = authUser?.tenantVoiceMode
    if (!scopeTenantId && (fromAuth === 'realtime' || fromAuth === 'pipeline')) {
      setTenantVoiceMode(fromAuth)
    }
  }, [authUser?.tenantVoiceMode, scopeTenantId])

  useEffect(() => {
    let cancelled = false
    setCredsLoaded(false)
    void listCredentials({ page: 1, size: 100, status: 'active' })
      .then((res) => {
        if (cancelled) return
        const list = res.code === 200 && res.data ? res.data.list || [] : []
        setCredentials(list)
        setCredsLoaded(true)
      })
      .catch(() => {
        if (!cancelled) {
          setCredentials([])
          setCredsLoaded(true)
        }
      })
    return () => {
      cancelled = true
    }
  }, [effectiveTenantId])

  // Prefer assistant-bound key; otherwise auto-pick the first active key.
  useEffect(() => {
    if (!credsLoaded) return
    if (isEdit && loading) return
    setCredentialId((prev) => {
      const prevId = String(prev || '').trim()
      if (prevId && credentials.some((c) => String(c.id) === prevId)) return prevId
      const bound = String(boundCredentialId || '').trim()
      if (bound && bound !== '0' && credentials.some((c) => String(c.id) === bound)) {
        return bound
      }
      return String(credentials[0]?.id || '')
    })
  }, [credsLoaded, credentials, boundCredentialId, isEdit, loading])

  useEffect(() => {
    if (!effectiveTenantId) return
    let cancelled = false
    const providerTenantId =
      scopeTenantId?.trim() ||
      (authUser?.principal === 'platform' ? effectiveTenantId : undefined)
    void (async () => {
      try {
        const res = await getTenantVoiceProviders(providerTenantId || undefined)
        if (cancelled || res.code !== 200 || !res.data) return
        const vm = res.data.voiceMode === 'realtime' ? 'realtime' : 'pipeline'
        setTenantVoiceMode(vm)
        setTenantTtsProvider(String(res.data.ttsProvider || defaultVoiceProvider('tts')))
        setTenantRealtimeProvider(String(res.data.realtimeProvider || defaultVoiceProvider('realtime')))
        setPoolTtsVoiceIds(Array.isArray(res.data.ttsVoiceIds) ? res.data.ttsVoiceIds : [])
        setPoolRealtimeVoiceIds(
          Array.isArray(res.data.realtimeVoiceIds) ? res.data.realtimeVoiceIds : [],
        )
      } catch {
        /* optional */
      }
    })()
    return () => { cancelled = true }
  }, [effectiveTenantId, scopeTenantId, authUser?.principal])

  useEffect(() => {
    let cancelled = false
    void listVoiceClones('success')
      .then((res) => {
        if (cancelled || res.code !== 200 || !res.data) return
        const ids = new Set<string>()
        for (const c of res.data) {
          const id = (c.assetId || c.speakerId || String(c.id)).trim()
          if (id) ids.add(id)
        }
        setCloneVoiceIds(ids)
        const tv = ttsVoice.trim()
        if (!tv) {
          setSelectedCloneProfile(null)
          return
        }
        if (!ids.has(tv)) {
          setSelectedCloneProfile(null)
          return
        }
        const matched = res.data.find(
          (c) => (c.assetId || c.speakerId || String(c.id)).trim() === tv,
        )
        setSelectedCloneProfile(matched ?? null)
      })
      .catch(() => {
        if (!cancelled) {
          setCloneVoiceIds(new Set())
          setSelectedCloneProfile(null)
        }
      })
    return () => {
      cancelled = true
    }
  }, [ttsVoice, effectiveTenantId])

  const visibleTabs = useMemo(() => {
    const base = [
      { key: 'dialog', label: t('assistant.tabDialog') },
      { key: 'voice', label: t('assistant.tabVoice') },
      { key: 'tools', label: t('assistant.tabTools') },
      ...(isEdit ? [{ key: 'access', label: t('assistant.tabAccess') }] : []),
      ...(isEdit ? [{ key: 'publish', label: t('assistant.tabPublish') }] : []),
    ]
    if (professionalMode) {
      return [...base, { key: 'developer', label: t('assistant.tabDeveloper') }]
    }
    return base
  }, [professionalMode, isEdit, t])

  useEffect(() => {
    if (!visibleTabs.some((t) => t.key === activeTab)) {
      setActiveTab(visibleTabs[0]?.key || 'dialog')
    }
  }, [visibleTabs, activeTab])

  useEffect(() => {
    void (async () => {
      try {
        const res = await listKnowledgeNamespaces()
        if (res.code === 200 && res.data) {
          setKnowledgeOptions(
            res.data
              .map((n) => ({
                value: n.namespace || '',
                label: n.name ? `${n.name} (${n.namespace})` : String(n.namespace),
              }))
              .filter((o) => o.value),
          )
        }
      } catch { /* optional */ }
    })()
  }, [])

  useEffect(() => {
    if (!nluEnabled) {
      setNluOptions([])
      return
    }
    void (async () => {
      try {
        const rows = await listNluModels()
        setNluOptions(
          rows
            .filter((m) => String(m.status || '').toLowerCase() === 'ready')
            .map((m) => ({
              value: String(m.id),
              label: `${m.name || '未命名'}${m.minConfidence != null ? ` · conf≥${m.minConfidence}` : ''}`,
            })),
        )
      } catch {
        setNluOptions([])
      }
    })()
  }, [nluEnabled])

  useEffect(() => {
    if (isEdit || !templateId) return
    const template = getAssistantTemplate(templateId)
    if (!template) return
    const preset = applyAssistantTemplate(template)
    setName(preset.name)
    setScene(preset.scene)
    setDescription(preset.description)
    setWelcome(preset.welcome)
    setPrompt(preset.prompt)
    setKnowledgeNamespace(preset.knowledgeNamespace)
    setAgent(preset.agent)
  }, [isEdit, templateId])

  useEffect(() => {
    if (!isEdit || !assistantId) return
    let cancelled = false
    setLoading(true)
    void (async () => {
      try {
        const res = await getAssistant(assistantId)
        if (cancelled) return
        if (res.code !== 200 || !res.data) {
          showAlert(res.msg || t('assistant.loadFailed'), 'error')
          navigate(listPath)
          return
        }
        const row = res.data
        // 从 llmConfig / realtimeConfig 中提取 instructions / welcome，作为 prompt / welcome 的 fallback
        const llmCfg = (row.llmConfig as Record<string, unknown> | null) ?? null
        const rtCfg = (row.realtimeConfig as Record<string, unknown> | null) ?? null
        const fallbackPrompt =
          (typeof llmCfg?.instructions === 'string' && llmCfg.instructions) ||
          ''
        const fallbackWelcome =
          (typeof llmCfg?.welcome === 'string' && llmCfg.welcome) ||
          (typeof rtCfg?.welcome === 'string' && rtCfg.welcome) ||
          ''

        if (row.tenantId) setTenantId(String(row.tenantId))
        setPublishedVersionId(String(row.publishedVersionId || ''))
        setName(row.name || '')
        setAvatarUrl(row.avatarUrl || row.avatar || '')
        setScene(row.scene || 'general')
        setDescription(row.description || '')
        setEnabled(!!row.enabled)
        setWelcome(row.welcome || fallbackWelcome)
        setPrompt(row.prompt || fallbackPrompt)
        setKnowledgeNamespace(row.knowledgeNamespace || '')
        setNluModelId(row.nluModelId && row.nluModelId !== '0' ? String(row.nluModelId) : '')
        setAgent(agentConfigFromJSON(row.agentConfig))
        setVad(vadConfigFromJSON(row.vadConfig))
        setHotWords(hotWordsFromJSON(row.hotWords))
        setInterruption(interruptionConfigFromJSON(row.interruptionConfig))
        setAudioTrack(audioTrackConfigFromJSON(row.audioTrackConfig))
        setAudioProcess(audioProcessConfigFromJSON(row.audioProcessConfig))
        setQueryRewriter(queryRewriterFromJSON(row.queryRewriter))
        setMcpServers(mcpServersFromJSON(row.mcpServers))
        setTtsVoice(String(row.ttsVoice || ''))
        setRealtimeVoice(String(row.realtimeVoice || ''))
        {
          const bound = String(row.credentialId || '').trim()
          setBoundCredentialId(bound && bound !== '0' ? bound : '')
        }
        setVoiceDialogWsUrl(String(row.voiceDialogWsUrl || ''))
        setBoundJsTemplateSourceId(String(row.boundJsTemplateSourceId || ''))
        try {
          if (voiceprintEnabled) {
            const vpRes = await listVoiceprints()
            if (!cancelled && vpRes.code === 200 && Array.isArray(vpRes.data)) {
              const bound = vpRes.data
                .filter((p) => snowflakeStr(p.assistantId) === snowflakeStr(assistantId))
                .map((p) => snowflakeStr(p.id))
              setVoiceprintIds(bound)
            }
          }
        } catch {
          /* voiceprint optional */
        }
        const verRes = await listAssistantVersions(assistantId)
        if (!cancelled && verRes.code === 200 && verRes.data) {
          setVersions(verRes.data)
        }
      } catch (e: unknown) {
        if (!cancelled) {
          showAlert((e as { msg?: string })?.msg || t('assistant.loadFailed'), 'error')
          navigate(listPath)
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => { cancelled = true }
  }, [assistantId, isEdit, listPath, navigate, voiceprintEnabled])

  const buildBody = () => ({
    name: name.trim(),
    scene,
    description: description.trim(),
    enabled,
    welcome: welcome.trim(),
    prompt: prompt.trim(),
    knowledgeNamespace: knowledgeNamespace.trim(),
    nluModelId: nluModelId.trim() || '0',
    agentConfig: agentConfigToJSON(agent),
    vadConfig: vadConfigToJSON(vad),
    hotWords: hotWordsToJSON(hotWords),
    interruptionConfig: interruptionConfigToJSON(interruption),
    audioTrackConfig: audioTrackConfigToJSON(audioTrack),
    audioProcessConfig: audioProcessConfigToJSON(audioProcess),
    queryRewriter: queryRewriterToJSON(queryRewriter),
    mcpServers: mcpServersToJSON(mcpServers),
    ttsVoice: ttsVoice.trim(),
    realtimeVoice: realtimeVoice.trim(),
    credentialId: credentialId.trim() || '0',
    voiceDialogWsUrl: voiceDialogWsUrl.trim(),
    boundJsTemplateSourceId: boundJsTemplateSourceId.trim(),
    ...(scopeTenantId && !isEdit ? { tenantId: scopeTenantId } : {}),
  })

  const formSig = useMemo(() => JSON.stringify({ body: buildBody(), voiceprintIds }), [
    name, scene, description, enabled, welcome, prompt, knowledgeNamespace, nluModelId,
    agent, vad, hotWords, interruption, audioTrack, audioProcess, queryRewriter, mcpServers,
    ttsVoice, realtimeVoice, credentialId, voiceDialogWsUrl, boundJsTemplateSourceId, scopeTenantId, isEdit,
    voiceprintIds,
  ])
  const formDirty = !!baselineSig && formSig !== baselineSig
  const leaveDirty = formDirty || draftDiffKeys.length > 0
  const { confirmLeave } = useUnsavedLeaveGuard({
    dirty: leaveDirty,
    message: t('assistant.leaveUnpublishedConfirm'),
  })

  const refreshDraftDiff = async (id: string) => {
    try {
      const res = await diffAssistantVersions(id)
      if (res.code === 200 && Array.isArray(res.data?.changedKeys) && res.data.changedKeys.length > 0) {
        setDraftDiffKeys(res.data.changedKeys)
      } else {
        setDraftDiffKeys([])
      }
    } catch {
      setDraftDiffKeys([])
    }
  }

  useEffect(() => {
    if (!isEdit || loading || baselineSig) return
    setBaselineSig(formSig)
  }, [isEdit, loading, formSig, baselineSig])

  useEffect(() => {
    if (!isEdit || !assistantId || loading) return
    void refreshDraftDiff(assistantId)
  }, [isEdit, assistantId, loading])

  const syncVoiceprintBindings = async (aid: string, selected: string[]) => {
    const want = new Set(selected.map(String))
    const listRes = await listVoiceprints()
    if (listRes.code !== 200 || !Array.isArray(listRes.data)) return
    const aidStr = snowflakeStr(aid)
    for (const row of listRes.data) {
      const id = snowflakeStr(row.id)
      const currentlyBound = snowflakeStr(row.assistantId) === aidStr
      const shouldBind = want.has(id)
      if (shouldBind && !currentlyBound) {
        const res = await bindVoiceprintAssistant(row.id, aidStr)
        if (res.code !== 200) throw new Error(res.msg || `绑定声纹失败: ${row.name}`)
      } else if (!shouldBind && currentlyBound) {
        const res = await bindVoiceprintAssistant(row.id, null)
        if (res.code !== 200) throw new Error(res.msg || `解绑声纹失败: ${row.name}`)
      }
    }
  }

  const save = async () => {
    if (!isEdit && scopeTenantId && (!scopeTenantId.trim() || scopeTenantId === '0')) {
      return showAlert(t('common.selectTenant'), 'error')
    }
    if (!name.trim()) return showAlert(t('common.nameRequired'), 'error')
    if (!prompt.trim()) return showAlert(t('common.promptRequired'), 'error')
    if (!credentialId.trim() || credentialId === '0') {
      return showAlert(t('assistant.credentialRequired'), 'error')
    }
    const voiceOk =
      tenantVoiceMode === 'realtime' ? Boolean(realtimeVoice.trim()) : Boolean(ttsVoice.trim())
    if (!voiceOk && !selectedCloneProfile) {
      return showAlert(t('assistant.voiceRequired'), 'error')
    }

    setSaving(true)
    try {
      const body = buildBody()
      const res =
        isEdit && assistantId ? await updateAssistant(assistantId, body) : await createAssistant(body)
      if (res.code !== 200) {
        showAlert(res.msg || t('assistant.saveFailed'), 'error')
        return
      }
      const savedId = isEdit && assistantId ? assistantId : String(res.data?.id || '')
      if (savedId && voiceprintEnabled && voiceprintIds.length >= 0) {
        try {
          await syncVoiceprintBindings(savedId, voiceprintIds)
        } catch (e: unknown) {
          showAlert((e as Error)?.message || '智能体已保存，但声纹绑定失败', 'warning')
        }
      }
      showAlert(t('assistant.saveSuccess'), 'success')
      setBaselineSig(JSON.stringify({ body, voiceprintIds }))
      if (isEdit && assistantId) await refreshDraftDiff(assistantId)
      if (!isEdit && res.data?.id) {
        navigate(`${listPath}/${res.data.id}/edit`)
      }
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || t('assistant.saveFailed'), 'error')
    } finally {
      setSaving(false)
    }
  }

  const publish = async () => {
    if (!assistantId) return
    setPublishing(true)
    try {
      // Persist draft first so publish captures latest form edits.
      if (formDirty) {
        const body = buildBody()
        const saveRes = await updateAssistant(assistantId, body)
        if (saveRes.code !== 200) {
          showAlert(saveRes.msg || t('assistant.saveFailed'), 'error')
          return
        }
        setBaselineSig(JSON.stringify(body))
      }
      const res = await publishAssistant(assistantId)
      if (res.code !== 200) {
        showAlert(res.msg || t('common.publishFailed'), 'error')
        return
      }
      showAlert(t('common.publishSuccess'), 'success')
      setDraftDiffKeys([])
      const verRes = await listAssistantVersions(assistantId)
      if (verRes.code === 200 && verRes.data) {
        setVersions(verRes.data)
        if (verRes.data[0]?.id) setPublishedVersionId(String(verRes.data[0].id))
      }
      const refreshed = await getAssistant(assistantId)
      if (refreshed.code === 200 && refreshed.data?.publishedVersionId) {
        setPublishedVersionId(String(refreshed.data.publishedVersionId))
      }
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || t('common.publishFailed'), 'error')
    } finally {
      setPublishing(false)
    }
  }

  const rollback = async (versionId: string) => {
    if (!assistantId) return
    try {
      const res = await rollbackAssistant(assistantId, versionId)
      if (res.code !== 200) {
        showAlert(res.msg || t('common.rollbackFailed'), 'error')
        return
      }
      showAlert(t('common.rollbackSuccess'), 'success')
      window.location.reload()
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || t('common.rollbackFailed'), 'error')
    }
  }

  const toggleDebug = () => {
    setDebugOpen((open) => {
      const next = !open
      const params = new URLSearchParams(searchParams)
      if (next) params.set('debug', '1')
      else params.delete('debug')
      setSearchParams(params, { replace: true })
      return next
    })
  }

  useEffect(() => {
    if (debugOpen) {
      setDebugRendered(true)
      const frame = requestAnimationFrame(() => setDebugVisible(true))
      return () => cancelAnimationFrame(frame)
    }
    setDebugVisible(false)
    const timer = window.setTimeout(() => setDebugRendered(false), 320)
    return () => window.clearTimeout(timer)
  }, [debugOpen])

  const sceneLabel = ASSISTANT_SCENES.find((s) => s.value === scene)?.label ?? scene
  const createBackPath =
    templateId != null
      ? `/assistant-manager/new${scopeTenantId ? `?tenantId=${encodeURIComponent(scopeTenantId)}` : ''}`
      : listPath

  return (
    <BaseLayout hideHeader>
      <AssistantPageHeader
        listPath={listPath}
        currentLabel={loading ? t('common.loading') : name.trim() || (isEdit ? t('assistant.formEdit') : t('assistant.formCreate'))}
        onCancel={() => {
          if (!confirmLeave()) return
          navigate(isEdit ? listPath : createBackPath)
        }}
        primaryLabel={isEdit ? t('assistant.save') : t('assistant.create')}
        primaryLoading={saving}
        onPrimary={() => void save()}
        showPrimary
        secondaryLabel={t('common.publishVersion')}
        secondaryLoading={publishing}
        onSecondary={() => void publish()}
        showSecondary={isEdit && !!assistantId}
        onDebugClick={isEdit && assistantId ? toggleDebug : undefined}
        debugActive={debugOpen}
        onSettingsClick={isEdit && assistantId ? () => setSettingsOpen(true) : undefined}
      />

      {isEdit && assistantId && (
        <AssistantSettingsModal
          visible={settingsOpen}
          assistantId={assistantId}
          name={name}
          description={description}
          avatarUrl={avatarUrl}
          onClose={() => setSettingsOpen(false)}
          onSaved={(patch) => {
            setName(patch.name)
            setDescription(patch.description)
            if (patch.avatarUrl) setAvatarUrl(patch.avatarUrl)
          }}
        />
      )}

      <AssistantFileUploadModal
        visible={fileUploadOpen}
        value={agent.fileUpload || {}}
        onChange={(next) => setAgent((prev) => ({ ...prev, fileUpload: next }))}
        onClose={() => setFileUploadOpen(false)}
        onConfirm={() => setFileUploadOpen(false)}
      />

      <div
        className={cn(
          'flex min-h-0 w-full',
          debugRendered && isEdit && assistantId ? 'max-lg:flex-col lg:flex-row' : 'flex-col',
        )}
      >
        <div
          className={cn(
            'min-w-0 overflow-auto px-6 py-6 transition-[flex,width,opacity] duration-300 ease-out',
            debugRendered && isEdit && assistantId ? 'w-full lg:w-2/3 lg:flex-1' : 'w-full',
            debugRendered && isEdit && assistantId ? 'max-lg:hidden' : '',
          )}
        >
        {loading ? (
          <Loading block tip={t('common.loading')} />
        ) : (
          <div className="w-full space-y-4">
            <div className="flex flex-wrap items-center gap-3">
              <Input
                placeholder={t('common.namePlaceholder')}
                value={name}
                onChange={setName}
                style={{ fontSize: 18, fontWeight: 600, flex: 1, minWidth: 240, maxWidth: 560 }}
              />
              {scene !== 'general' && <Tag color="arcoblue">{sceneLabel}</Tag>}
                {isEdit && (
                <Tag color={isPublished ? 'green' : 'orange'}>{isPublished ? t('common.published') : t('common.unpublish')}</Tag>
              )}
              <div className="ml-auto flex flex-wrap items-center gap-4">
                <label className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Switch size="small" checked={enabled} onChange={setEnabled} />
                  {t('common.enable')}
                </label>
                <label className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Switch size="small" checked={professionalMode} onChange={setProfessionalMode} />
                  {t('common.professionalMode')}
                </label>
              </div>
            </div>

            {isEdit && assistantId && (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                ID: {assistantId}
                {versions[0]?.version ? ` · ${t('assistant.latestVersion', { version: versions[0].version })}` : ''}
              </Typography.Text>
            )}

            {isEdit && !isPublished && (
              <Alert
                type="warning"
                title={t('assistant.unpublishAlert')}
                content={t('assistant.unpublishContent')}
                closable={false}
              />
            )}
            {isEdit && draftDiffKeys.length > 0 && (
              <Alert
                type="warning"
                title={t('assistant.draftUnpublishedAlert')}
                content={t('assistant.draftUnpublishedKeys', { keys: draftDiffKeys.join(', ') })}
                closable={false}
              />
            )}

            <Tabs activeTab={activeTab} onChange={setActiveTab} type="rounded">
              <TabPane key="dialog" title={t('assistant.tabDialog')}>
                <div className="pt-4 space-y-4">
                  <AssistantFileUploadRow
                    config={agent.fileUpload || {}}
                    onConfigure={() => setFileUploadOpen(true)}
                  />
                  <AssistantDialogFields
                    welcome={welcome}
                    setWelcome={setWelcome}
                    description={description}
                    setDescription={setDescription}
                    prompt={prompt}
                    setPrompt={setPrompt}
                    knowledgeNamespace={knowledgeNamespace}
                    setKnowledgeNamespace={setKnowledgeNamespace}
                    knowledgeOptions={knowledgeOptions}
                    nluModelId={nluModelId}
                    setNluModelId={setNluModelId}
                    nluOptions={nluOptions}
                    nluEnabled={nluEnabled}
                  />
                  {voiceprintEnabled ? (
                    <div className="rounded-xl border border-border bg-card p-5 space-y-4">
                      <Typography.Text bold>声纹说话人</Typography.Text>
                      <AssistantVoiceprintBindFields
                        assistantId={isEdit ? assistantId : undefined}
                        selectedIds={voiceprintIds}
                        onChange={setVoiceprintIds}
                      />
                    </div>
                  ) : null}
                </div>
              </TabPane>

              <TabPane key="voice" title={t('assistant.tabVoice')}>
                <div className="pt-4">
                  <div className="rounded-xl border border-border bg-card p-5 space-y-3">
                    <Typography.Text bold>{t('assistant.voiceModeLabel')}</Typography.Text>
                    <Typography.Text style={{ display: 'block' }}>
                      {t('assistant.voiceModeDesc')}
                      <Tag
                        color={tenantVoiceMode === 'realtime' ? 'green' : 'arcoblue'}
                        style={{ marginLeft: 8, maxWidth: 140, verticalAlign: 'middle' }}
                        title={
                          tenantVoiceMode === 'realtime'
                            ? t('assistant.voiceRealtimeLabel')
                            : t('assistant.voicePipelineLabel')
                        }
                      >
                        <span className="block max-w-full truncate">
                          {tenantVoiceMode === 'realtime'
                            ? t('assistant.voiceRealtimeLabel')
                            : t('assistant.voicePipelineLabel')}
                        </span>
                      </Tag>
                    </Typography.Text>
                    <Typography.Paragraph type="secondary" style={{ marginBottom: 0, fontSize: 13 }}>
                      {selectedCloneProfile ? (
                        <>
                          已选择克隆音色「{selectedCloneProfile.name}」，会话时将优先使用{' '}
                          <strong>Pipeline</strong> 及{' '}
                          <strong>
                            {selectedCloneProfile.provider === 'volcengine'
                              ? '火山声音复刻 TTS'
                              : '讯飞克隆 TTS'}
                          </strong>
                          ；下方仍可切换当前密钥可用的预置音色。
                        </>
                      ) : (
                        t('assistant.voiceModeDetailSimple')
                      )}
                    </Typography.Paragraph>
                  </div>
                  <div className="rounded-xl border border-border bg-card p-5 space-y-3">
                    <Typography.Text bold>{t('assistant.defaultCredentialLabel')}</Typography.Text>
                    <Typography.Paragraph type="secondary" style={{ marginBottom: 0, fontSize: 13 }}>
                      {t('assistant.defaultCredentialHint')}
                    </Typography.Paragraph>
                    <Select
                      style={{ width: '100%', maxWidth: 480 }}
                      placeholder={
                        !credsLoaded
                          ? '加载密钥…'
                          : credentials.length === 0
                            ? '暂无可用密钥'
                            : t('assistant.defaultCredentialPlaceholder')
                      }
                      value={credentialId || undefined}
                      allowClear={false}
                      disabled={!credsLoaded || credentials.length === 0}
                      options={credentials.map((c) => ({
                        value: String(c.id),
                        label: `${c.name || '—'} · ${c.apiKeyPrefix || c.accessKey || ''} · ${
                          c.kind === 'platform_bundle' || c.usesTenantAi
                            ? t('credentialAi.kindPlatform')
                            : t('credentialAi.kindUser')
                        }`,
                      }))}
                      onChange={(v) => onCredentialChange(String(v || ''))}
                    />
                    {credsLoaded && credentials.length === 0 ? (
                      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                        {t('assistant.defaultCredentialEmpty')}{' '}
                        <Link to="/access-keys">{t('assistant.defaultCredentialManage')}</Link>
                      </Typography.Text>
                    ) : selectedCredential ? (
                      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                        {t('assistant.defaultCredentialCurrent')}：
                        {selectedCredential.name || '—'} · 厂商 {voiceProvider}
                      </Typography.Text>
                    ) : null}
                  </div>
                  <div className="rounded-xl border border-border bg-card p-5 space-y-3">
                    <Typography.Text bold>{t('assistant.voiceSelectLabel')}</Typography.Text>
                    <Typography.Paragraph type="secondary" style={{ marginBottom: 0, fontSize: 13 }}>
                      {t('assistant.voiceSelectHint')}
                    </Typography.Paragraph>
                    <AssistantVoiceFields
                      voiceMode={tenantVoiceMode}
                      provider={voiceProvider}
                      value={voicePickerValue}
                      setValue={(v) => {
                        if (cloneVoiceIds.has(v)) return
                        setSelectedCloneProfile(null)
                        if (tenantVoiceMode === 'realtime') {
                          setRealtimeVoice(v)
                          setTtsVoice('')
                        } else {
                          setTtsVoice(v)
                        }
                      }}
                      tenantId={effectiveTenantId || undefined}
                      credentialId={credentialId || undefined}
                      allowedVoiceIds={allowedVoiceIds}
                      includeCloneVoices
                      onCloneVoiceSelect={(profile, voiceId) => {
                        setSelectedCloneProfile(profile)
                        if (profile) {
                          setTtsVoice(voiceId)
                          return
                        }
                        if (cloneVoiceIds.has(ttsVoice.trim())) {
                          setTtsVoice('')
                        }
                      }}
                    />
                  </div>
                  <div className="rounded-xl border border-border bg-card p-5 space-y-3">
                    <Typography.Text bold>{t('assistant.wsLabel')}</Typography.Text>
                    <Typography.Paragraph type="secondary" style={{ marginBottom: 8, fontSize: 13 }}>
                      {t('assistant.wsHint')}
                    </Typography.Paragraph>
                    <Input.TextArea
                      placeholder={t('assistant.wsPlaceholder')}
                      autoSize={{ minRows: 2, maxRows: 4 }}
                      value={voiceDialogWsUrl}
                      onChange={setVoiceDialogWsUrl}
                      style={{ fontFamily: 'monospace', fontSize: 12 }}
                    />
                  </div>
                </div>
              </TabPane>

              <TabPane key="tools" title={t('assistant.tabTools')}>
                <div className="pt-4 space-y-4">
                  <div className="rounded-xl border border-border bg-card p-5 space-y-4">
                    <Typography.Text bold>{t('assistant.catalogToolsTitle')}</Typography.Text>
                    <AssistantCatalogToolBindFields
                      selectedIds={agent.customToolIds || []}
                      onChange={(ids) => setAgent({ ...agent, customToolIds: ids })}
                    />
                  </div>
                  <div className="rounded-xl border border-border bg-card p-5 space-y-4">
                    <Typography.Text bold>{t('assistant.dialogSkillsTitle')}</Typography.Text>
                    <AssistantDialogSkillBindFields
                      selectedCodes={agent.dialogSkills || []}
                      onChange={(codes) => setAgent({ ...agent, dialogSkills: codes })}
                    />
                  </div>
                </div>
              </TabPane>

              {professionalMode && (
                <TabPane key="developer" title={t('assistant.tabDeveloper')}>
                  <div className="pt-4 space-y-4">
                    <div className="rounded-xl border border-border bg-card p-5 space-y-4">
                      <Typography.Text bold>{t('assistant.dialogBehavior')}</Typography.Text>
                      <AssistantBehaviorFields agent={agent} setAgent={setAgent} />
                    </div>
                    <div className="rounded-xl border border-border bg-card p-5 space-y-4">
                      <Typography.Text bold>{t('assistant.vadConfig')}</Typography.Text>
                      <AssistantVadFields vad={vad} setVad={setVad} />
                    </div>
                    <div className="rounded-xl border border-border bg-card p-5 space-y-4">
                      <Typography.Text bold>{t('assistant.interruptionConfig')}</Typography.Text>
                      <AssistantInterruptionFields cfg={interruption} setCfg={setInterruption} />
                    </div>
                    <div className="rounded-xl border border-border bg-card p-5 space-y-4">
                      <Typography.Text bold>{t('assistant.hotWordsConfig')}</Typography.Text>
                      <AssistantHotWordsFields rows={hotWords} setRows={setHotWords} />
                    </div>
                    <div className="rounded-xl border border-border bg-card p-5 space-y-4">
                      <Typography.Text bold>{t('assistant.audioProcess')}</Typography.Text>
                      <AssistantAudioProcessFields cfg={audioProcess} setCfg={setAudioProcess} />
                    </div>
                    <div className="rounded-xl border border-border bg-card p-5 space-y-4">
                      <Typography.Text bold>{t('assistant.queryRewriter')}</Typography.Text>
                      <AssistantQueryRewriterFields cfg={queryRewriter} setCfg={setQueryRewriter} />
                    </div>
                    <div className="rounded-xl border border-border bg-card p-5 space-y-4">
                      <Typography.Text bold>{t('assistant.audioTrack')}</Typography.Text>
                      <AssistantAudioTrackFields cfg={audioTrack} setCfg={setAudioTrack} />
                    </div>
                  </div>
                </TabPane>
              )}

              {isEdit && assistantId ? (
                <TabPane key="access" title={t('assistant.tabAccess')}>
                  <div className="pt-4 space-y-4">
                    <div className="rounded-xl border border-border bg-card p-5">
                      <AssistantEmbedPanel
                        assistantId={assistantId}
                        boundJsTemplateSourceId={boundJsTemplateSourceId}
                        onBoundJsTemplateSourceIdChange={setBoundJsTemplateSourceId}
                      />
                    </div>
                    <div className="rounded-xl border border-border bg-card p-5">
                      <AssistantExternalAPIPanel assistantId={assistantId} />
                    </div>
                  </div>
                </TabPane>
              ) : null}

              {isEdit && (
                <TabPane key="publish" title={t('assistant.tabPublish')}>
                  <div className="pt-4">
                    <AssistantVersionPanel
                      assistantId={assistantId}
                      versions={versions}
                      publishedVersionId={publishedVersionId}
                      publishing={publishing}
                      onPublish={() => void publish()}
                      onRollback={rollback}
                    />
                  </div>
                </TabPane>
              )}
            </Tabs>
          </div>
        )}
        </div>

        {debugRendered && isEdit && assistantId ? (
          <aside
            className={cn(
              'shrink-0 border-border bg-card transition-all duration-300 ease-out',
              'max-lg:fixed max-lg:inset-0 max-lg:z-50 max-lg:w-full max-lg:border-0',
              'lg:sticky lg:top-0 lg:h-[calc(100vh-52px)] lg:min-w-[320px] lg:max-w-[480px] lg:border-l',
              debugVisible
                ? 'max-lg:translate-y-0 max-lg:opacity-100 lg:w-1/3 lg:translate-x-0 lg:opacity-100'
                : 'max-lg:translate-y-full max-lg:opacity-0 lg:w-0 lg:translate-x-8 lg:opacity-0 lg:overflow-hidden lg:border-l-0',
            )}
          >
            <AssistantDebugPanel
              assistantId={assistantId}
              embedded
              assistantName={name.trim() || undefined}
              assistantAvatar={avatarUrl || undefined}
              fileUploadEnabled={
                !!agent.fileUpload?.enabled &&
                (!!agent.fileUpload?.documentEnabled || !!agent.fileUpload?.imageEnabled)
              }
              maxFiles={agent.fileUpload?.maxFiles ?? 8}
              onClose={toggleDebug}
            />
          </aside>
        ) : null}
      </div>
    </BaseLayout>
  )
}
