import React, {useEffect, useState} from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Key, Settings, AppWindow, RefreshCw, ArrowRight, Mic } from 'lucide-react';
import { cn } from '@/utils/cn';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/UI/Select';
import Button from '@/components/UI/Button';
import { Switch } from '@/components/UI/Switch';
import Card from '@/components/UI/Card';
import CollapsibleSectionHeader from '@/components/UI/CollapsibleSectionHeader';
import { getVoiceOptions, VoiceOption } from '@/api/assistant';
import { jsTemplateService, type JSTemplate } from '@/api/jsTemplate';
import { highlightContent } from '@/utils/highlight';
import { useI18nStore } from '@/stores/i18nStore';

interface ControlPanelProps {
    // API 配置
    apiKey: string
    apiSecret: string
    onApiKeyChange: (value: string) => void
    onApiSecretChange: (value: string) => void

    // TTS Provider配置
    ttsProvider?: string  // TTS平台提供商，如 "tencent", "qiniu", "baidu" 等

    // 通话设置
    selectedSpeaker: string
    systemPrompt: string
    temperature: number
    maxTokens: number
    llmModel: string // LLM模型名称

    // 设置更新函数
    onSpeakerChange: (value: string) => void
    onSystemPromptChange: (value: string) => void
    onTemperatureChange: (value: number) => void
    onMaxTokensChange: (value: number) => void
    onLlmModelChange: (value: string) => void
    jsSourceId?: string
    onJsSourceIdChange?: (value: string) => void

    // 助手设置
    assistantName: string
    assistantDescription: string
    onAssistantNameChange: (value: string) => void
    onAssistantDescriptionChange: (value: string) => void
    // VAD 配置
    enableVAD?: boolean
    vadThreshold?: number
    vadConsecutiveFrames?: number
    onEnableVADChange?: (value: boolean) => void
    onVADThresholdChange?: (value: number) => void
    onVADConsecutiveFramesChange?: (value: number) => void
    // JSON 输出配置
    enableJSONOutput?: boolean
    onEnableJSONOutputChange?: (value: boolean) => void
    onSaveSettings: () => void
    isSavingSettings?: boolean // 保存状态
    onDeleteAssistant: () => void
    // 训练音色配置
    selectedVoiceCloneId: number | null
    onVoiceCloneChange: (value: number | null) => void
    voiceClones: Array<{id: number, voice_name: string}>
    onRefreshVoiceClones: () => void
    onNavigateToVoiceTraining: () => void
    // 应用接入
    onMethodClick: (method: string) => void

    // 搜索高亮（可选）
    searchKeyword?: string
    highlightFragments?: Record<string, string[]> | null
    highlightResultId?: string

    className?: string
}
const ControlPanel: React.FC<ControlPanelProps> = ({
                                                       apiKey,
                                                       apiSecret,
                                                       onApiKeyChange,
                                                       onApiSecretChange,
                                                       ttsProvider,
                                                       selectedSpeaker,
                                                       systemPrompt,
                                                       temperature,
                                                       maxTokens,
                                                       llmModel,
                                                       onSpeakerChange,
                                                       onSystemPromptChange,
                                                       onTemperatureChange,
                                                       onMaxTokensChange,
                                                       onLlmModelChange,
                                                       jsSourceId = '',
                                                       onJsSourceIdChange,
                                                       assistantName,
                                                       assistantDescription,
                                                       onAssistantNameChange,
                                                       onAssistantDescriptionChange,
                                                       enableVAD = true,
                                                       vadThreshold = 500,
                                                       vadConsecutiveFrames = 2,
                                                       onEnableVADChange,
                                                       onVADThresholdChange,
                                                       onVADConsecutiveFramesChange,
                                                       enableJSONOutput = false,
                                                       onEnableJSONOutputChange,
                                                       onSaveSettings,
                                                       isSavingSettings = false,
                                                       onDeleteAssistant,
                                                       onMethodClick,
                                                       selectedVoiceCloneId,
                                                       onVoiceCloneChange,
                                                       voiceClones,
                                                       onRefreshVoiceClones,
                                                       onNavigateToVoiceTraining,
                                                       searchKeyword,
                                                       highlightFragments,
                                                       highlightResultId,
                                                       className = ''
                                                   }) => {
    const { t } = useI18nStore()
    const [voiceOptions, setVoiceOptions] = useState<VoiceOption[]>([]);
    const [loadingVoices, setLoadingVoices] = useState(false);
    const [jsTemplates, setJsTemplates] = useState<JSTemplate[]>([])


    // 根据TTS Provider加载音色列表
    const fetchVoiceOptions = async (provider: string, currentSpeaker?: string) => {
        if (!provider) {
            setVoiceOptions([]);
            return;
        }

        setLoadingVoices(true);
        try {
            const response = await getVoiceOptions(provider);
            if (response.code === 200 && response.data?.voices) {
                setVoiceOptions(response.data.voices);
                // 如果当前选中的音色不在新列表中，重置为第一个音色
                if (currentSpeaker && !response.data.voices.find(v => v.id === currentSpeaker)) {
                    if (response.data.voices.length > 0) {
                        onSpeakerChange(response.data.voices[0].id);
                    }
                } else if (!currentSpeaker && response.data.voices.length > 0) {
                    onSpeakerChange(response.data.voices[0].id);
                }
            }
        } catch (error) {
            console.error('获取音色列表失败:', error);
            setVoiceOptions([]);
        } finally {
            setLoadingVoices(false);
        }
    };

    // 当TTS Provider变化时，重新加载音色列表
    useEffect(() => {
        const provider = ttsProvider || 'tencent'; // 如果没有provider，使用默认的腾讯云音色列表（向后兼容）
        fetchVoiceOptions(provider, selectedSpeaker);
    }, [ttsProvider]); // 只依赖ttsProvider，selectedSpeaker和language的变化不影响重新加载

    useEffect(() => {
        const fetchJSTemplates = async () => {
            try {
                const response = await jsTemplateService.getTemplates({ page: 1, limit: 200 })
                const templates = response?.data?.data || []
                setJsTemplates(templates)
            } catch (error) {
                console.error('获取JS模板失败:', error)
                setJsTemplates([])
            }
        }
        fetchJSTemplates()
    }, [])
    const [expandedSections, setExpandedSections] = useState({
        api: true,
        call: true,
        assistant: true,
        integration: true,
        voiceClone: true,
        vad: true,
    })
    const toggleSection = (section: keyof typeof expandedSections) => {
        setExpandedSections(prev => ({
            ...prev,
            [section]: !prev[section]
        }))
    }

    // @ts-ignore
    return (
        <div className={cn('flex-1 p-6 overflow-y-auto space-y-4 custom-scrollbar', className)}>
            <div className="space-y-6 min-h-0 max-h-[85vh]">
                {/* API 密钥配置 */}
                <motion.div
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    className="space-y-4"
                >
                    <CollapsibleSectionHeader
                        title={t('controlPanel.api.title')}
                        icon={<Key className="w-5 h-5" />}
                        expanded={expandedSections.api}
                        onToggle={() => toggleSection('api')}
                    />

                    <AnimatePresence>
                        {expandedSections.api && (
                            <motion.div
                                initial={{ height: 0, opacity: 0 }}
                                animate={{ height: 'auto', opacity: 1 }}
                                exit={{ height: 0, opacity: 0 }}
                                transition={{ duration: 0.3, ease: 'easeInOut' }}
                                className="overflow-hidden"
                            >
                                <div className="space-y-4 pt-4">
                                    <div className="space-y-2">
                                        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('controlPanel.api.apiKey')}</label>
                                        <input
                                            type="text"
                                            value={apiKey}
                                            onChange={(e) => onApiKeyChange(e.target.value)}
                                            className="w-full p-2 text-sm border rounded-lg focus:ring-2 focus:ring-purple-500 dark:bg-neutral-700 dark:border-neutral-600"
                                            placeholder={t('controlPanel.api.apiKeyPlaceholder')}
                                        />
                                    </div>

                                    <div className="space-y-2">
                                        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('controlPanel.api.apiSecret')}</label>
                                        <input
                                            type="password"
                                            value={apiSecret}
                                            onChange={(e) => onApiSecretChange(e.target.value)}
                                            className="w-full p-2 text-sm border rounded-lg focus:ring-2 focus:ring-purple-500 dark:bg-neutral-700 dark:border-neutral-600"
                                            placeholder={t('controlPanel.api.apiSecretPlaceholder')}
                                        />
                                    </div>
                                </div>
                            </motion.div>
                        )}
                    </AnimatePresence>
                </motion.div>

                {/* 通话设置 */}
                <motion.div
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.1 }}
                    className="space-y-4"
                >
                    <CollapsibleSectionHeader
                        title={t('controlPanel.call.title')}
                        icon={<Settings className="w-5 h-5" />}
                        expanded={expandedSections.call}
                        onToggle={() => toggleSection('call')}
                    />

                    <AnimatePresence>
                        {expandedSections.call && (
                            <motion.div
                                initial={{ height: 0, opacity: 0 }}
                                animate={{ height: 'auto', opacity: 1 }}
                                exit={{ height: 0, opacity: 0 }}
                                transition={{ duration: 0.3, ease: 'easeInOut' }}
                                className="overflow-hidden"
                            >
                                <div className="space-y-4 pt-4">
                                    {/* 发音人选择 */}
                                    <div className="space-y-2">
                                        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                                            {t('controlPanel.call.speaker')}
                                            {ttsProvider && (
                                                <span className="ml-2 text-xs text-gray-500 dark:text-gray-400">
                          ({ttsProvider})
                        </span>
                                            )}
                                        </label>
                                        {loadingVoices ? (
                                            <div className="w-full p-3 text-sm text-gray-500 dark:text-gray-400 text-center border border-gray-200 dark:border-gray-700 rounded-lg bg-gray-50 dark:bg-gray-800">
                                                {t('controlPanel.call.loadingVoices')}
                                            </div>
                                        ) : voiceOptions.length > 0 ? (
                                            <Select
                                                value={selectedSpeaker}
                                                onValueChange={onSpeakerChange}
                                                className="w-full"
                                            >
                                                <SelectTrigger>
                                                    <SelectValue placeholder={t('controlPanel.call.speakerPlaceholder')}>
                                                        {voiceOptions.find(v => v.id === selectedSpeaker)
                                                            ? `${voiceOptions.find(v => v.id === selectedSpeaker)?.name} - ${voiceOptions.find(v => v.id === selectedSpeaker)?.description}`
                                                            : t('controlPanel.call.speakerPlaceholder')}
                                                    </SelectValue>
                                                </SelectTrigger>
                                                <SelectContent searchable searchPlaceholder="搜索音色">
                                                    {voiceOptions.map(voice => (
                                                        <SelectItem key={voice.id} value={voice.id}>
                                                            <div className="flex flex-col">
                                                                <span className="font-medium">{voice.name}</span>
                                                                <span className="text-xs text-gray-500 dark:text-gray-400">
                                  {voice.description} · {voice.type}
                                </span>
                                                            </div>
                                                        </SelectItem>
                                                    ))}
                                                </SelectContent>
                                            </Select>
                                        ) : (
                                            <div className="w-full p-3 text-sm text-gray-500 dark:text-gray-400 text-center border border-gray-200 dark:border-gray-700 rounded-lg bg-gray-50 dark:bg-gray-800">
                                                {ttsProvider ? t('controlPanel.call.noVoices', { provider: ttsProvider }) : t('controlPanel.call.noProvider')}
                                            </div>
                                        )}
                                    </div>

                                    {/* 系统提示词 */}
                                    <div className="space-y-2">
                                        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('controlPanel.call.systemPrompt')}</label>
                                        <div className="space-y-1">
                      <textarea
                          value={systemPrompt}
                          onChange={(e) => onSystemPromptChange(e.target.value)}
                          className="w-full p-2 text-sm border rounded-lg focus:ring-2 focus:ring-purple-500 dark:bg-neutral-700 dark:border-neutral-600"
                          placeholder={t('controlPanel.call.systemPromptPlaceholder')}
                          rows={3}
                      />
                                            {searchKeyword && systemPrompt && (
                                                <div
                                                    className="text-xs text-gray-400 p-2 bg-gray-50 dark:bg-neutral-800 rounded border"
                                                    dangerouslySetInnerHTML={{
                                                        __html: highlightContent(systemPrompt, searchKeyword, highlightFragments ?? undefined)
                                                    }}
                                                />
                                            )}
                                        </div>
                                    </div>

                                    {/* Temperature 控制 */}
                                    <div className="space-y-2">
                                        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('controlPanel.call.temperature')}</label>
                                        <div className="flex justify-between text-sm">
                                            <span className="text-gray-500">{t('controlPanel.call.temperatureLabel')}</span>
                                            <span className="font-medium text-purple-600">{temperature.toFixed(1)}</span>
                                        </div>
                                        <input
                                            type="range"
                                            min="0"
                                            max="1.5"
                                            step="0.1"
                                            value={temperature}
                                            onChange={(e) => onTemperatureChange(parseFloat(e.target.value))}
                                            className="w-full"
                                        />
                                    </div>

                                    {/* Max Tokens 控制 */}
                                    <div className="space-y-2">
                                        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('controlPanel.call.maxTokens')}</label>
                                        <input
                                            type="number"
                                            min="10"
                                            max="2048"
                                            step="10"
                                            value={maxTokens}
                                            onChange={(e) => onMaxTokensChange(parseInt(e.target.value))}
                                            className="w-full p-2 text-sm border rounded-lg focus:ring-2 focus:ring-purple-500 dark:bg-neutral-700 dark:border-neutral-600"
                                            placeholder={t('controlPanel.call.maxTokensPlaceholder')}
                                        />
                                    </div>

                                    {/* LLM 模型设置 */}
                                    <div className="space-y-2">
                                        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('controlPanel.call.llmModel')}</label>
                                        <input
                                            type="text"
                                            value={llmModel}
                                            onChange={(e) => onLlmModelChange(e.target.value)}
                                            className="w-full p-2 text-sm border rounded-lg focus:ring-2 focus:ring-purple-500 dark:bg-neutral-700 dark:border-neutral-600"
                                            placeholder={t('controlPanel.call.llmModelPlaceholder')}
                                        />
                                        <p className="text-xs text-gray-500 dark:text-gray-400">
                                            {t('controlPanel.call.llmModelHint')}
                                        </p>
                                    </div>

                                </div>
                            </motion.div>
                        )}
                    </AnimatePresence>
                </motion.div>

                {/* 助手设置 */}
                <motion.div
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.2 }}
                    className="space-y-4"
                >
                    <CollapsibleSectionHeader
                        title={t('controlPanel.assistant.title')}
                        icon={<Settings className="w-5 h-5" />}
                        expanded={expandedSections.assistant}
                        onToggle={() => toggleSection('assistant')}
                    />

                    <AnimatePresence>
                        {expandedSections.assistant && (
                            <motion.div
                                initial={{ height: 0, opacity: 0 }}
                                animate={{ height: 'auto', opacity: 1 }}
                                exit={{ height: 0, opacity: 0 }}
                                transition={{ duration: 0.3, ease: 'easeInOut' }}
                                className="overflow-hidden"
                            >
                                <div className="pt-4 border-t dark:border-neutral-700 mb-6 space-y-6">
                                    {/* 助手基本信息 */}
                                    <div className="space-y-4">
                                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                                            {t('controlPanel.assistant.basicInfo')}
                                        </label>

                                        <div className="space-y-2">
                                            <label className="text-xs text-gray-500 dark:text-gray-400">{t('controlPanel.assistant.name')}</label>
                                            <div
                                                className={`w-full p-2 text-sm border rounded-lg focus-within:ring-2 focus-within:ring-purple-500 dark:bg-neutral-700 dark:border-neutral-600 dark:text-gray-100 ${highlightResultId?.startsWith('assistant_') ? 'ring-2 ring-yellow-400' : ''}`}
                                            >
                                                <input
                                                    type="text"
                                                    value={assistantName}
                                                    onChange={(e) => onAssistantNameChange(e.target.value)}
                                                    className="w-full bg-transparent border-none outline-none"
                                                    placeholder={t('controlPanel.assistant.namePlaceholder')}
                                                />
                                                {searchKeyword && (
                                                    <div
                                                        className="text-xs text-gray-400 mt-1"
                                                        dangerouslySetInnerHTML={{
                                                            __html: highlightContent(assistantName, searchKeyword, highlightFragments ?? undefined)
                                                        }}
                                                    />
                                                )}
                                            </div>
                                        </div>

                                        <div className="space-y-2">
                                            <label className="text-xs text-gray-500 dark:text-gray-400">{t('controlPanel.assistant.description')}</label>
                                            <div className="space-y-1">
                      <textarea
                          value={assistantDescription}
                          onChange={(e) => onAssistantDescriptionChange(e.target.value)}
                          className="w-full p-2 text-sm border rounded-lg focus:ring-2 focus:ring-purple-500 dark:bg-neutral-700 dark:border-neutral-600 dark:text-gray-100"
                          rows={2}
                          placeholder={t('controlPanel.assistant.descriptionPlaceholder')}
                      />
                                                {searchKeyword && assistantDescription && (
                                                    <div
                                                        className="text-xs text-gray-400 p-2 bg-gray-50 dark:bg-neutral-800 rounded border"
                                                        dangerouslySetInnerHTML={{
                                                            __html: highlightContent(assistantDescription, searchKeyword, highlightFragments ?? undefined)
                                                        }}
                                                    />
                                                )}
                                            </div>
                                        </div>

                                        <div className="space-y-2">
                                            <label className="text-xs text-gray-500 dark:text-gray-400">
                                                {t('controlPanel.assistant.jsTemplate')}
                                            </label>
                                            <Select
                                                value={jsSourceId || ''}
                                                onValueChange={(value) => onJsSourceIdChange?.(value)}
                                                className="w-full"
                                            >
                                                <SelectTrigger>
                                                    <SelectValue placeholder={t('controlPanel.assistant.jsTemplatePlaceholder')}>
                                                        {jsSourceId
                                                            ? jsTemplates.find(tpl => tpl.jsSourceId === jsSourceId)?.name || jsSourceId
                                                            : t('controlPanel.assistant.jsTemplateDefault')}
                                                    </SelectValue>
                                                </SelectTrigger>
                                                <SelectContent searchable searchPlaceholder="搜索模板">
                                                    <SelectItem value="">{t('controlPanel.assistant.jsTemplateDefault')}</SelectItem>
                                                    {jsTemplates.map((tpl) => (
                                                        <SelectItem key={tpl.id} value={tpl.jsSourceId}>
                                                            {tpl.name}
                                                        </SelectItem>
                                                    ))}
                                                </SelectContent>
                                            </Select>
                                            <p className="text-xs text-gray-500 dark:text-gray-400">
                                                {t('controlPanel.assistant.jsTemplateHint')}
                                            </p>
                                        </div>
                                    </div>

                                    <div className="flex justify-between pt-4 border-t dark:border-neutral-700 gap-3">
                                        <Button
                                            onClick={onDeleteAssistant}
                                            variant="destructive"
                                            size="md"
                                            className="flex-1"
                                        >
                                            {t('controlPanel.assistant.delete')}
                                        </Button>
                                        <Button
                                            onClick={onSaveSettings}
                                            variant="success"
                                            size="md"
                                            loading={isSavingSettings}
                                            disabled={isSavingSettings}
                                            leftIcon={<Settings className="w-4 h-4" />}
                                            className="flex-1"
                                        >
                                            {isSavingSettings ? t('controlPanel.assistant.saving') : t('controlPanel.assistant.save')}
                                        </Button>
                                    </div>
                                </div>
                            </motion.div>
                        )}
                    </AnimatePresence>
                </motion.div>
                {/* 训练音色配置 */}
                <motion.div
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.2 }}
                    className="space-y-4"
                >
                    <CollapsibleSectionHeader
                        title={t('controlPanel.voiceClone.title')}
                        icon={<Settings className="w-5 h-5" />}
                        expanded={expandedSections.voiceClone}
                        onToggle={() => toggleSection('voiceClone')}
                    />

                    <AnimatePresence>
                        {expandedSections.voiceClone && (
                            <motion.div
                                initial={{ height: 0, opacity: 0 }}
                                animate={{ height: 'auto', opacity: 1 }}
                                exit={{ height: 0, opacity: 0 }}
                                transition={{ duration: 0.3, ease: 'easeInOut' }}
                                className="overflow-hidden"
                            >
                                <div className="space-y-4 pt-4 mb-24">
                                    <div className="space-y-2 mb-6">
                                        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('controlPanel.voiceClone.select')}</label>
                                        <div className="flex items-center gap-2 mb-10">
                                            <Select
                                                className="flex-1"
                                                value={selectedVoiceCloneId?.toString() ?? ''}
                                                onValueChange={(value) => onVoiceCloneChange(value === '' ? null : Number(value) || null)}
                                            >
                                                <SelectTrigger className="flex-1 shadow-sm">
                                                    <SelectValue placeholder={t('controlPanel.voiceClone.select')}>
                                                        {selectedVoiceCloneId === null
                                                            ? t('controlPanel.voiceClone.none')
                                                            : selectedVoiceCloneId ?
                                                                voiceClones.find(vc => vc.id === selectedVoiceCloneId)?.voice_name || t('controlPanel.voiceClone.unknown')
                                                                : t('controlPanel.voiceClone.select')
                                                        }
                                                    </SelectValue>
                                                </SelectTrigger>
                                                <SelectContent searchable searchPlaceholder="搜索训练音色">
                                                    <SelectItem key="none" value="">
                                                        {t('controlPanel.voiceClone.none')}
                                                    </SelectItem>
                                                    {voiceClones.map(vc => (
                                                        <SelectItem key={vc.id} value={vc.id.toString()}>
                                                            {vc.voice_name}
                                                        </SelectItem>
                                                    ))}
                                                </SelectContent>
                                            </Select>
                                        </div>
                                        <div className="flex space-x-2 mt-6 mb-6">
                                            <Button
                                                variant="outline"
                                                size="sm"
                                                onClick={onRefreshVoiceClones}
                                                leftIcon={<RefreshCw className="w-3 h-3" />}
                                                className="shadow-sm hover:shadow-md"
                                            >
                                                {t('controlPanel.voiceClone.refresh')}
                                            </Button>
                                            <Button
                                                variant="primary"
                                                size="sm"
                                                onClick={onNavigateToVoiceTraining}
                                                leftIcon={<ArrowRight className="w-3 h-3" />}
                                                className="shadow-sm hover:shadow-md"
                                            >
                                                {t('controlPanel.voiceClone.training')}
                                            </Button>
                                        </div>
                                        <p className="text-xs text-gray-500 dark:text-gray-400">
                                            {t('controlPanel.voiceClone.hint')}
                                        </p>
                                    </div>
                                </div>
                            </motion.div>
                        )}
                    </AnimatePresence>
                </motion.div>

                {/* VAD 监测配置 */}
                <motion.div
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.25 }}
                    className="space-y-4"
                >
                    <CollapsibleSectionHeader
                        title={t('controlPanel.vad.title')}
                        icon={<Mic className="w-5 h-5" />}
                        expanded={expandedSections.vad}
                        onToggle={() => toggleSection('vad')}
                    />

                    <AnimatePresence>
                        {expandedSections.vad && (
                            <motion.div
                                initial={{ height: 0, opacity: 0 }}
                                animate={{ height: 'auto', opacity: 1 }}
                                exit={{ height: 0, opacity: 0 }}
                                transition={{ duration: 0.3, ease: 'easeInOut' }}
                                className="overflow-hidden"
                            >
                                <div className="space-y-4 pt-4">
                                    {/* 启用 VAD 开关 */}
                                    <div className="space-y-2">
                                        <div className="flex items-center justify-between">
                                            <div className="flex-1">
                                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                                    {t('controlPanel.vad.enable')}
                                                </label>
                                                <p className="text-xs text-gray-500 dark:text-gray-400">
                                                    {t('controlPanel.vad.enableDesc')}
                                                </p>
                                            </div>
                                            <div className="ml-4 flex-shrink-0">
                                                <Switch
                                                    checked={enableVAD}
                                                    onCheckedChange={(checked) => {
                                                        if (onEnableVADChange) {
                                                            onEnableVADChange(checked)
                                                        }
                                                    }}
                                                    size="md"
                                                    className="flex-shrink-0"
                                                />
                                            </div>
                                        </div>
                                    </div>

                                    {/* VAD 阈值 */}
                                    {enableVAD && (
                                        <>
                                            <div className="space-y-2">
                                                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                                                    {t('controlPanel.vad.threshold')}
                                                </label>
                                                <div className="flex justify-between text-sm">
                                                    <span className="text-gray-500">{t('controlPanel.vad.thresholdLabel')}</span>
                                                    <span className="font-medium text-purple-600">{vadThreshold}</span>
                                                </div>
                                                <input
                                                    type="range"
                                                    min="100"
                                                    max="5000"
                                                    step="50"
                                                    value={vadThreshold}
                                                    onChange={(e) => {
                                                        if (onVADThresholdChange) {
                                                            onVADThresholdChange(parseFloat(e.target.value))
                                                        }
                                                    }}
                                                    className="w-full"
                                                    disabled={!enableVAD}
                                                />
                                                <p className="text-xs text-gray-500 dark:text-gray-400">
                                                    {t('controlPanel.vad.thresholdHint')}
                                                </p>
                                            </div>

                                            {/* 连续帧数 */}
                                            <div className="space-y-2">
                                                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                                                    {t('controlPanel.vad.consecutiveFrames')}
                                                </label>
                                                <input
                                                    type="number"
                                                    min="1"
                                                    max="10"
                                                    step="1"
                                                    value={vadConsecutiveFrames}
                                                    onChange={(e) => {
                                                        if (onVADConsecutiveFramesChange) {
                                                            onVADConsecutiveFramesChange(parseInt(e.target.value) || 2)
                                                        }
                                                    }}
                                                    className="w-full p-2 text-sm border rounded-lg focus:ring-2 focus:ring-purple-500 dark:bg-neutral-700 dark:border-neutral-600"
                                                    placeholder="2"
                                                    disabled={!enableVAD}
                                                />
                                                <p className="text-xs text-gray-500 dark:text-gray-400">
                                                    {t('controlPanel.vad.consecutiveFramesHint')}
                                                </p>
                                            </div>
                                        </>
                                    )}

                                    {/* JSON 输出配置 */}
                                    <div className="space-y-2 pt-4 border-t border-gray-200 dark:border-neutral-700">
                                        <div className="flex items-center justify-between">
                                            <div className="flex-1">
                                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                                    JSON 格式化输出
                                                </label>
                                                <p className="text-xs text-gray-500 dark:text-gray-400">
                                                    启用后，LLM 将返回结构化的 JSON 格式响应
                                                </p>
                                            </div>
                                            <div className="ml-4 flex-shrink-0">
                                                <Switch
                                                    checked={enableJSONOutput}
                                                    onCheckedChange={(checked) => {
                                                        if (onEnableJSONOutputChange) {
                                                            onEnableJSONOutputChange(checked)
                                                        }
                                                    }}
                                                    size="md"
                                                    className="flex-shrink-0"
                                                />
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </motion.div>
                        )}
                    </AnimatePresence>
                </motion.div>

                {/* 应用接入 */}
                <motion.div
                    initial={{opacity: 0, y: 20}}
                    animate={{opacity: 1, y: 0 }}
                    transition={{ delay: 0.3 }}
                    className="space-y-4"
                >
                    <CollapsibleSectionHeader
                        title={t('controlPanel.integration.title')}
                        icon={<AppWindow className="w-5 h-5" />}
                        expanded={expandedSections.integration}
                        onToggle={() => toggleSection('integration')}
                    />

                    <AnimatePresence>
                        {expandedSections.integration && (
                            <motion.div
                                initial={{ height: 0, opacity: 0 }}
                                animate={{ height: 'auto', opacity: 1 }}
                                exit={{ height: 0, opacity: 0 }}
                                transition={{ duration: 0.3, ease: 'easeInOut' }}
                                className="overflow-hidden"
                            >
                                <div className="pt-4">
                                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                        {/* Web应用嵌入 */}
                                        <Card
                                            variant="outlined"
                                            padding="md"
                                            hover={true}
                                            onClick={() => onMethodClick('web')}
                                            className="cursor-pointer border-purple-200 dark:border-purple-800 hover:border-purple-400 dark:hover:border-purple-600 transition-all duration-200"
                                        >
                                            <div className="text-center">
                                                <div className="w-12 h-12 mx-auto mb-3 rounded-lg bg-purple-100 dark:bg-purple-900/30 flex items-center justify-center transition-colors">
                                                    <svg className="w-6 h-6 text-purple-600 dark:text-purple-400" viewBox="0 0 1024 1024" version="1.1" xmlns="http://www.w3.org/2000/svg">
                                                        <path
                                                            d="M853.333333 170.666667H170.666667c-46.933333 0-85.333333 38.4-85.333334 85.333333v512c0 46.933333 38.4 85.333333 85.333334 85.333333h682.666666c46.933333 0 85.333333-38.4 85.333334-85.333333V256c0-46.933333-38.4-85.333333-85.333334-85.333333z m-213.333333 597.333333H170.666667v-170.666667h469.333333v170.666667z m0-213.333333H170.666667V384h469.333333v170.666667z m213.333333 213.333333h-170.666666V384h170.666666v384z"
                                                            fill="currentColor"></path>
                                                    </svg>
                                                </div>
                                                <h4 className="text-sm font-semibold text-gray-800 dark:text-gray-200 mb-1">
                                                    {t('controlPanel.integration.web')}
                                                </h4>
                                                <p className="text-xs text-gray-500 dark:text-gray-400">
                                                    {t('controlPanel.integration.webDesc')}
                                                </p>
                                            </div>
                                        </Card>

                                        {/* Flutter应用集成 */}
                                        <Card
                                            variant="outlined"
                                            padding="md"
                                            hover={true}
                                            onClick={() => onMethodClick('flutter')}
                                            className="cursor-pointer border-green-200 dark:border-green-800 hover:border-green-400 dark:hover:border-green-600 transition-all duration-200"
                                        >
                                            <div className="text-center">
                                                <div className="w-12 h-12 mx-auto mb-3 rounded-lg bg-green-100 dark:bg-green-900/30 flex items-center justify-center transition-colors">
                                                    <svg className="w-6 h-6 text-green-600 dark:text-green-400" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                                                        <path d="M14.5 12C14.5 13.3807 13.3807 14.5 12 14.5C10.6193 14.5 9.5 13.3807 9.5 12C9.5 10.6193 10.6193 9.5 12 9.5C13.3807 9.5 14.5 10.6193 14.5 12Z" fill="currentColor"/>
                                                        <path d="M12 2C13.1 2 14 2.9 14 4V8C14 9.1 13.1 10 12 10C10.9 10 10 9.1 10 8V4C10 2.9 10.9 2 12 2ZM19 8C19 12.4 15.4 16 11 16H10V18H14V20H10V18H6V16H5C0.6 16 -3 12.4 -3 8H1C1 11.3 3.7 14 7 14H17C20.3 14 23 11.3 23 8H19Z" fill="currentColor"/>
                                                    </svg>
                                                </div>
                                                <h4 className="text-sm font-semibold text-gray-800 dark:text-gray-200 mb-1">
                                                    {t('controlPanel.integration.flutter')}
                                                </h4>
                                                <p className="text-xs text-gray-500 dark:text-gray-400">
                                                    {t('controlPanel.integration.flutterDesc')}
                                                </p>
                                            </div>
                                        </Card>
                                    </div>
                                </div>
                            </motion.div>
                        )}
                    </AnimatePresence>
                </motion.div>

            </div>
        </div>
    )
}


export default ControlPanel
