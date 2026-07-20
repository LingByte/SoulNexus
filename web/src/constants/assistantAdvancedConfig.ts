/** Advanced assistant config — form-friendly shapes (stored as JSON on backend). */

export interface VadConfigDraft {
  vadMode?: number
  energyThreshold?: number
  positiveSpeechThreshold?: number
  negativeSpeechThreshold?: number
  minSpeechFrames?: number
  redemptionFrames?: number
  paddingDuration?: number
  ratio?: number
  speechStartMs?: number
  speechEndMs?: number
}


export interface KnowledgeConfigDraft {
  topK?: number
  sliceMinimumScore?: number
  threshold?: number
  useMemoEnhanceQuery?: boolean
  usePreviousRoundsSlice?: number
  autoEnrich?: boolean
}

export interface AgentConfigDraft {
  startCanBreak?: boolean
  threshold?: number
  topK?: number
  maxDialogueGapTimes?: number
  maxSilentAskTimes?: number
  sliceTime?: number
  enableHangupTool?: boolean
  useFiller?: boolean
  fillerWords?: string[]
  knowledgeConfig?: KnowledgeConfigDraft
  fileUpload?: FileUploadConfigDraft
  /** Catalog tool IDs bound to this assistant (HTTP/custom tools). Empty = none. */
  customToolIds?: string[]
  /** Tenant dialog skill codes bound to this assistant. Empty = none. */
  dialogSkills?: string[]
}

export interface FileUploadConfigDraft {
  enabled?: boolean
  documentEnabled?: boolean
  imageEnabled?: boolean
  maxFiles?: number
  pdfEnhanced?: boolean
}

export interface AudioProcessConfigDraft {
  vadType?: string
  noiseSuppressionEnabled?: boolean
  noiseSuppressionType?: string
}

export interface QueryRewriterDraft {
  useRewriter?: boolean
  rewritePrompt?: string
}

export interface McpServerDraft {
  name?: string
  command?: string
  args?: string[]
  envs?: Record<string, string>
}

export interface HotWordDraft {
  word: string
  weight?: number
  replacedWords?: string[]
  enableFuzzyMatch?: boolean
}

export interface InterruptionConfigDraft {
  method?: string
  unInterruptableAfterPlayStart?: number
  talkOverThreshold?: number
  resumePlay?: boolean
}

export interface EffectAudioTrackDraft {
  filename?: string
  volume?: number
  isActive?: boolean
  mode?: string
}

export interface AudioTrackConfigDraft {
  masterVolume?: number
  mainTrackVolume?: number
  effectAudioTracks?: EffectAudioTrackDraft[]
}

export const defaultVadConfig = (): VadConfigDraft => ({
  vadMode: 3,
  energyThreshold: 5500,
  positiveSpeechThreshold: 0.5,
  negativeSpeechThreshold: 0.4,
  minSpeechFrames: 12,
  redemptionFrames: 10,
})


export const defaultAgentConfig = (): AgentConfigDraft => ({
  startCanBreak: true,
  threshold: 0.4,
  topK: 3,
  maxDialogueGapTimes: 3,
  maxSilentAskTimes: 3,
  sliceTime: 5000,
  enableHangupTool: true,
  customToolIds: [],
  dialogSkills: [],
	knowledgeConfig: {
		topK: 3,
		sliceMinimumScore: 0.4,
		threshold: 0.4,
		useMemoEnhanceQuery: true,
		usePreviousRoundsSlice: 0,
		autoEnrich: true,
	},
})

function num(v: unknown, fallback?: number): number | undefined {
  if (v === undefined || v === null || v === '') return fallback
  const n = Number(v)
  return Number.isFinite(n) ? n : fallback
}

function bool(v: unknown, fallback = false): boolean {
  if (typeof v === 'boolean') return v
  return fallback
}

export function vadConfigFromJSON(raw: unknown): VadConfigDraft {
  const d = defaultVadConfig()
  if (!raw || typeof raw !== 'object') return d
  const m = raw as Record<string, unknown>
  return {
    vadMode: num(m.vadMode, d.vadMode),
    energyThreshold: num(m.energyThreshold, d.energyThreshold),
    positiveSpeechThreshold: num(m.positiveSpeechThreshold, d.positiveSpeechThreshold),
    negativeSpeechThreshold: num(m.negativeSpeechThreshold, d.negativeSpeechThreshold),
    minSpeechFrames: num(m.minSpeechFrames, d.minSpeechFrames),
    redemptionFrames: num(m.redemptionFrames, d.redemptionFrames),
    paddingDuration: num(m.paddingDuration, 300),
    ratio: num(m.ratio, 1),
    speechStartMs: num(m.speechStartMs, 0),
    speechEndMs: num(m.speechEndMs, 0),
  }
}


export function agentConfigFromJSON(raw: unknown): AgentConfigDraft {
  const d = defaultAgentConfig()
  if (!raw || typeof raw !== 'object') return d
  const m = raw as Record<string, unknown>
  const funcs = Array.isArray(m.functions) ? m.functions : []
  const names = new Set(
    funcs.map((f) => (f && typeof f === 'object' ? String((f as { name?: string }).name || '') : '')),
  )
  const rawToolIds = m.customToolIds ?? m.custom_tool_ids
  const customToolIds = Array.isArray(rawToolIds)
    ? rawToolIds.map((x) => String(x ?? '').trim()).filter(Boolean)
    : []
  const rawSkills = m.dialogSkills ?? m.dialog_skills
  const dialogSkills = Array.isArray(rawSkills)
    ? rawSkills.map((x) => String(x ?? '').trim()).filter(Boolean)
    : []
  return {
    startCanBreak: bool(m.startCanBreak, d.startCanBreak),
    threshold: num(m.threshold, d.threshold),
    topK: num(m.topK, d.topK),
    maxDialogueGapTimes: num(m.maxDialogueGapTimes, d.maxDialogueGapTimes),
    maxSilentAskTimes: num(m.maxSilentAskTimes, d.maxSilentAskTimes),
    sliceTime: num(m.sliceTime, d.sliceTime),
    enableHangupTool: names.has('hangup'),
    useFiller: bool(m.useFiller, d.useFiller ?? false),
    fillerWords: Array.isArray(m.fillerWords)
      ? m.fillerWords.map((w) => String(w || '').trim()).filter(Boolean)
      : [],
    knowledgeConfig: knowledgeConfigFromJSON(m.knowledgeConfig, d.knowledgeConfig),
    fileUpload: fileUploadConfigFromJSON(m.fileUpload),
    customToolIds,
    dialogSkills,
  }
}

function fileUploadConfigFromJSON(raw: unknown): FileUploadConfigDraft {
  const d: FileUploadConfigDraft = {
    enabled: false,
    documentEnabled: true,
    imageEnabled: false,
    maxFiles: 8,
    pdfEnhanced: false,
  }
  if (!raw || typeof raw !== 'object') return d
  const m = raw as Record<string, unknown>
  return {
    enabled: bool(m.enabled, d.enabled ?? false),
    documentEnabled: bool(m.documentEnabled, d.documentEnabled ?? true),
    imageEnabled: bool(m.imageEnabled, d.imageEnabled ?? false),
    maxFiles: num(m.maxFiles, d.maxFiles) ?? 8,
    pdfEnhanced: bool(m.pdfEnhanced, d.pdfEnhanced ?? false),
  }
}

function knowledgeConfigFromJSON(raw: unknown, fallback?: KnowledgeConfigDraft): KnowledgeConfigDraft {
  const d = fallback ?? defaultAgentConfig().knowledgeConfig!
  if (!raw || typeof raw !== 'object') return { ...d }
  const m = raw as Record<string, unknown>
  return {
    topK: num(m.topK, d.topK),
    sliceMinimumScore: num(m.sliceMinimumScore, d.sliceMinimumScore),
    threshold: num(m.threshold, d.threshold),
    useMemoEnhanceQuery: bool(m.useMemoEnhanceQuery, d.useMemoEnhanceQuery),
    usePreviousRoundsSlice: num(m.usePreviousRoundsSlice, d.usePreviousRoundsSlice),
    autoEnrich: bool(m.autoEnrich, d.autoEnrich),
  }
}

export function vadConfigToJSON(d: VadConfigDraft): string {
  return JSON.stringify({
    vadMode: d.vadMode ?? 3,
    energyThreshold: d.energyThreshold ?? 5500,
    positiveSpeechThreshold: d.positiveSpeechThreshold ?? 0.5,
    negativeSpeechThreshold: d.negativeSpeechThreshold ?? 0.4,
    minSpeechFrames: d.minSpeechFrames ?? 10,
    redemptionFrames: d.redemptionFrames ?? 10,
    paddingDuration: d.paddingDuration ?? 300,
    ratio: d.ratio ?? 1,
    speechStartMs: d.speechStartMs ?? 0,
    speechEndMs: d.speechEndMs ?? 0,
  })
}


export function agentConfigToJSON(d: AgentConfigDraft): string {
  const functions: { name: string; description?: string }[] = []
  if (d.enableHangupTool) {
    functions.push({ name: 'hangup', description: '结束会话' })
  }
  return JSON.stringify({
    startCanBreak: d.startCanBreak ?? true,
    threshold: d.threshold ?? 0.4,
    topK: d.topK ?? 3,
    maxDialogueGapTimes: d.maxDialogueGapTimes ?? 3,
    maxSilentAskTimes: d.maxSilentAskTimes ?? 3,
    sliceTime: d.sliceTime ?? 5000,
    useFiller: !!d.useFiller,
    fillerWords: d.fillerWords ?? [],
    functions,
    knowledgeConfig: {
      topK: d.knowledgeConfig?.topK ?? d.topK ?? 3,
      sliceMinimumScore: d.knowledgeConfig?.sliceMinimumScore ?? d.threshold ?? 0.4,
      threshold: d.knowledgeConfig?.threshold ?? d.threshold ?? 0.4,
      useMemoEnhanceQuery: d.knowledgeConfig?.useMemoEnhanceQuery ?? true,
      usePreviousRoundsSlice: d.knowledgeConfig?.usePreviousRoundsSlice ?? 0,
      autoEnrich: d.knowledgeConfig?.autoEnrich ?? true,
    },
    fileUpload: {
      enabled: !!d.fileUpload?.enabled,
      documentEnabled: d.fileUpload?.documentEnabled ?? true,
      imageEnabled: !!d.fileUpload?.imageEnabled,
      maxFiles: d.fileUpload?.maxFiles ?? 8,
      pdfEnhanced: !!d.fileUpload?.pdfEnhanced,
    },
    customToolIds: Array.isArray(d.customToolIds)
      ? d.customToolIds.map((x) => String(x || '').trim()).filter(Boolean)
      : [],
    dialogSkills: Array.isArray(d.dialogSkills)
      ? d.dialogSkills.map((x) => String(x || '').trim()).filter(Boolean)
      : [],
  })
}

export const defaultInterruptionConfig = (): InterruptionConfigDraft => ({
  method: 'vad+transcribing',
  unInterruptableAfterPlayStart: 1200,
  talkOverThreshold: 0,
  resumePlay: false,
})

export const defaultAudioTrackConfig = (): AudioTrackConfigDraft => ({
  masterVolume: 1,
  mainTrackVolume: 0.8,
  effectAudioTracks: [],
})

export const defaultAudioProcessConfig = (): AudioProcessConfigDraft => ({
  vadType: 'energy',
  noiseSuppressionEnabled: true,
  noiseSuppressionType: 'ledenoise',
})

export const defaultQueryRewriter = (): QueryRewriterDraft => ({
  useRewriter: false,
  rewritePrompt: '',
})

export const defaultMcpServers = (): McpServerDraft[] => []

export function hotWordsFromJSON(raw: unknown): HotWordDraft[] {
  if (!Array.isArray(raw)) return []
  return raw
    .map((row) => {
      if (!row || typeof row !== 'object') return null
      const m = row as Record<string, unknown>
      const word = String(m.word || '').trim()
      if (!word) return null
      return {
        word,
        weight: num(m.weight, 10),
        replacedWords: Array.isArray(m.replacedWords) ? m.replacedWords.map(String) : [],
        enableFuzzyMatch: bool(m.enableFuzzyMatch, false),
      }
    })
    .filter(Boolean) as HotWordDraft[]
}

export function hotWordsToJSON(rows: HotWordDraft[]): string {
  return JSON.stringify(
    rows
      .filter((r) => r.word.trim())
      .map((r) => ({
        word: r.word.trim(),
        weight: r.weight ?? 10,
        replacedWords: r.replacedWords?.filter(Boolean) ?? [],
        enableFuzzyMatch: !!r.enableFuzzyMatch,
      })),
  )
}

export function interruptionConfigFromJSON(raw: unknown): InterruptionConfigDraft {
  const d = defaultInterruptionConfig()
  if (!raw || typeof raw !== 'object') return d
  const m = raw as Record<string, unknown>
  return {
    method: typeof m.method === 'string' ? m.method : d.method,
    unInterruptableAfterPlayStart: num(m.unInterruptableAfterPlayStart, d.unInterruptableAfterPlayStart),
    talkOverThreshold: num(m.talkOverThreshold, d.talkOverThreshold),
    resumePlay: bool(m.resumePlay, d.resumePlay),
  }
}

export function interruptionConfigToJSON(d: InterruptionConfigDraft): string {
  const base = defaultInterruptionConfig()
  return JSON.stringify({
    method: d.method ?? base.method,
    unInterruptableAfterPlayStart: d.unInterruptableAfterPlayStart ?? base.unInterruptableAfterPlayStart,
    talkOverThreshold: d.talkOverThreshold ?? base.talkOverThreshold,
    resumePlay: !!d.resumePlay,
  })
}

export function audioTrackConfigFromJSON(raw: unknown): AudioTrackConfigDraft {
  const d = defaultAudioTrackConfig()
  if (!raw || typeof raw !== 'object') return d
  const m = raw as Record<string, unknown>
  const tracks = Array.isArray(m.effectAudioTracks)
    ? m.effectAudioTracks.map((t) => {
        const row = t as Record<string, unknown>
        return {
          filename: String(row.filename || ''),
          volume: num(row.volume, 0.8),
          isActive: bool(row.isActive, true),
          mode: String(row.mode || ''),
        }
      })
    : []
  return {
    masterVolume: num(m.masterVolume, d.masterVolume),
    mainTrackVolume: num(m.mainTrackVolume, d.mainTrackVolume),
    effectAudioTracks: tracks,
  }
}

export function audioTrackConfigToJSON(d: AudioTrackConfigDraft): string {
  const base = defaultAudioTrackConfig()
  return JSON.stringify({
    masterVolume: d.masterVolume ?? base.masterVolume,
    mainTrackVolume: d.mainTrackVolume ?? base.mainTrackVolume,
    effectAudioTracks: d.effectAudioTracks ?? [],
  })
}

export function audioProcessConfigFromJSON(raw: unknown): AudioProcessConfigDraft {
  const d = defaultAudioProcessConfig()
  if (!raw || typeof raw !== 'object') return d
  const m = raw as Record<string, unknown>
  return {
    vadType: typeof m.vadType === 'string' ? m.vadType : d.vadType,
    noiseSuppressionEnabled: bool(m.noiseSuppressionEnabled, d.noiseSuppressionEnabled ?? true),
    noiseSuppressionType: typeof m.noiseSuppressionType === 'string' ? m.noiseSuppressionType : d.noiseSuppressionType,
  }
}

export function audioProcessConfigToJSON(d: AudioProcessConfigDraft): string {
  const base = defaultAudioProcessConfig()
  return JSON.stringify({
    vadType: d.vadType ?? base.vadType,
    noiseSuppressionEnabled: d.noiseSuppressionEnabled ?? base.noiseSuppressionEnabled,
    noiseSuppressionType: d.noiseSuppressionType ?? base.noiseSuppressionType,
  })
}

export function queryRewriterFromJSON(raw: unknown): QueryRewriterDraft {
  const d = defaultQueryRewriter()
  if (!raw || typeof raw !== 'object') return d
  const m = raw as Record<string, unknown>
  return {
    useRewriter: bool(m.useRewriter, d.useRewriter ?? false),
    rewritePrompt: typeof m.rewritePrompt === 'string' ? m.rewritePrompt : '',
  }
}

export function queryRewriterToJSON(d: QueryRewriterDraft): string {
  return JSON.stringify({
    useRewriter: !!d.useRewriter,
    rewritePrompt: d.rewritePrompt ?? '',
  })
}

export function mcpServersFromJSON(raw: unknown): McpServerDraft[] {
  if (!Array.isArray(raw)) return []
  return raw.map((row) => {
    const m = (row && typeof row === 'object' ? row : {}) as Record<string, unknown>
    const envs: Record<string, string> = {}
    if (m.envs && typeof m.envs === 'object') {
      for (const [k, v] of Object.entries(m.envs as Record<string, unknown>)) {
        envs[k] = String(v ?? '')
      }
    }
    return {
      name: String(m.name || ''),
      command: String(m.command || ''),
      args: Array.isArray(m.args) ? m.args.map((a) => String(a || '')).filter(Boolean) : [],
      envs,
    }
  })
}

export function mcpServersToJSON(rows: McpServerDraft[]): string {
  return JSON.stringify(
    rows.map((r) => ({
      name: r.name ?? '',
      command: r.command ?? '',
      args: r.args ?? [],
      envs: r.envs ?? {},
    })),
  )
}
