import React, {useEffect, useState} from 'react';
import { useNavigate } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { Key, Settings, AppWindow, ChevronDown, RefreshCw, ArrowRight, Bot, MessageCircle, Users, Zap, Circle, ExternalLink, Mic } from 'lucide-react';
import { cn } from '@/utils/cn';
import {getKnowledgeBaseByUser} from "@/api/knowledge.ts";
import { jsTemplateService, JSTemplate } from '@/api/jsTemplate';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/UI/Select.tsx';
import Button from '@/components/UI/Button';
import { Switch } from '@/components/UI/Switch';
import Card from '@/components/UI/Card';
import { getVoiceOptions, VoiceOption, getLanguageOptions, LanguageOption } from '@/api/assistant';
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
    language: string
    selectedSpeaker: string
    systemPrompt: string
    temperature: number
    maxTokens: number
    llmModel: string // LLM模型名称

    // 设置更新函数
    onLanguageChange: (value: string) => void
    onSpeakerChange: (value: string) => void
    onSystemPromptChange: (value: string) => void
    onTemperatureChange: (value: number) => void
    onMaxTokensChange: (value: number) => void
    onLlmModelChange: (value: string) => void

    // 助手设置
    assistantName: string
    assistantDescription: string
    assistantIcon: string
    enableGraphMemory?: boolean
    onAssistantNameChange: (value: string) => void
    onAssistantDescriptionChange: (value: string) => void
    onAssistantIconChange: (value: string) => void
    onEnableGraphMemoryChange?: (value: boolean) => void
    // VAD 配置
    enableVAD?: boolean
    vadThreshold?: number
    vadConsecutiveFrames?: number
    onEnableVADChange?: (value: boolean) => void
    onVADThresholdChange?: (value: number) => void
    onVADConsecutiveFramesChange?: (value: number) => void
    onSaveSettings: () => void
    onDeleteAssistant: () => void
    // JS模板配置
    selectedJSTemplate: string | null
    onJSTemplateChange: (value: string) => void
    // 知识库配置
    knowledgeBases: Array<{id: string, name: string}>
    selectedKnowledgeBase: string | null
    onKnowledgeBaseChange: (value: string) => void
    onRefreshKnowledgeBases: () => void
    onManageKnowledgeBases: () => void
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
// 默认语言列表（当无法从API获取时使用）
const DEFAULT_LANGUAGES = [
    { value: 'zh-CN', label: '中文（简体）' },
    { value: 'en-US', label: '英语（美国）' },
    { value: 'ja-JP', label: '日语' },
    { value: 'ko-KR', label: '韩语' }
]


const ICON_MAP = {
    Bot: <Bot className="w-5 h-5" />,
    MessageCircle: <MessageCircle className="w-5 h-5" />,
    Users: <Users className="w-5 h-5" />,
    Zap: <Zap className="w-5 h-5" />,
    Circle: <Circle className="w-5 h-5" />
}

const ControlPanel: React.FC<ControlPanelProps> = ({
                                                       apiKey,
                                                       apiSecret,
                                                       onApiKeyChange,
                                                       onApiSecretChange,
                                                       ttsProvider,
                                                       language,
                                                       selectedSpeaker,
                                                       systemPrompt,
                                                       temperature,
                                                       maxTokens,
                                                       llmModel,
                                                       onLanguageChange,
                                                       onSpeakerChange,
                                                       onSystemPromptChange,
                                                       onTemperatureChange,
                                                       onMaxTokensChange,
                                                       onLlmModelChange,
                                                       assistantName,
                                                       assistantDescription,
                                                       assistantIcon,
                                                       enableGraphMemory = false,
                                                       onAssistantNameChange,
                                                       onAssistantDescriptionChange,
                                                       onAssistantIconChange,
                                                       onEnableGraphMemoryChange,
                                                       enableVAD = true,
                                                       vadThreshold = 500,
                                                       vadConsecutiveFrames = 2,
                                                       onEnableVADChange,
                                                       onVADThresholdChange,
                                                       onVADConsecutiveFramesChange,
                                                       onSaveSettings,
                                                       onDeleteAssistant,
                                                       selectedJSTemplate,
                                                       onJSTemplateChange,
                                                       onMethodClick,
                                                       selectedKnowledgeBase,
                                                       onKnowledgeBaseChange,
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
    const [localKnowledgeBases, setLocalKnowledgeBases] = useState<Array<{id: string, name: string}>>([]);
    const [jsTemplates, setJsTemplates] = useState<JSTemplate[]>([]);
    const [voiceOptions, setVoiceOptions] = useState<VoiceOption[]>([]);
    const [loadingVoices, setLoadingVoices] = useState(false);
    const [languageOptions, setLanguageOptions] = useState<LanguageOption[]>([]);
    const [loadingLanguages, setLoadingLanguages] = useState(false);

    const fetchKnowledgeBases = async () => {
        try {
            const response = await getKnowledgeBaseByUser();
            if (response.code === 200 && Array.isArray(response.data)) {
                // 修改数据转换逻辑，适应后端返回的完整格式
                // 后端返回格式: { id, knowledge_key, knowledge_name, provider, config, ... }
                const transformedData = response.data
                    .filter((item: any) => item && (item.knowledge_key || item.key || item.id)) // 过滤无效数据
                    .map((item: any, index: number) => ({
                        id: item.knowledge_key || item.key || `kb-${item.id || index}`, // 确保唯一ID
                        name: item.knowledge_name || item.name || '未命名知识库'
                    }));
                setLocalKnowledgeBases(transformedData);
            } else {
                setLocalKnowledgeBases([]);
            }
        } catch (error) {
            console.error('获取知识库列表失败:', error);
            setLocalKnowledgeBases([]);
        }
    };

    const fetchJSTemplates = async () => {
        try {
            const response = await jsTemplateService.getTemplates({ page: 1, limit: 100 });
            if (response.code === 200) {
                setJsTemplates(response.data.data);
            }
        } catch (error) {
            console.error('获取JS模板失败:', error);
        }
    };
    const handleRefreshKnowledgeBases = () => {
        fetchKnowledgeBases();
    };

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

    // 根据TTS Provider加载语言列表
    const fetchLanguageOptions = async (provider: string, currentLanguage?: string) => {
        if (!provider) {
            // 如果没有provider，使用默认语言列表
            setLanguageOptions(DEFAULT_LANGUAGES.map(lang => ({
                code: lang.value,
                name: lang.label,
                nativeName: lang.label,
                configKey: 'language',
                description: lang.label
            })));
            return;
        }

        setLoadingLanguages(true);
        try {
            const response = await getLanguageOptions(provider);
            if (response.code === 200 && response.data?.languages) {
                setLanguageOptions(response.data.languages);
                // 如果当前选中的语言不在新列表中，重置为第一个语言
                if (currentLanguage && !response.data.languages.find(l => l.code === currentLanguage)) {
                    if (response.data.languages.length > 0) {
                        onLanguageChange(response.data.languages[0].code);
                    }
                } else if (!currentLanguage && response.data.languages.length > 0) {
                    onLanguageChange(response.data.languages[0].code);
                }
            } else {
                // 如果API返回失败，使用默认语言列表
                setLanguageOptions(DEFAULT_LANGUAGES.map(lang => ({
                    code: lang.value,
                    name: lang.label,
                    nativeName: lang.label,
                    configKey: 'language',
                    description: lang.label
                })));
            }
        } catch (error) {
            console.error('获取语言列表失败:', error);
            // 使用默认语言列表
            setLanguageOptions(DEFAULT_LANGUAGES.map(lang => ({
                code: lang.value,
                name: lang.label,
                nativeName: lang.label,
                configKey: 'language',
                description: lang.label
            })));
        } finally {
            setLoadingLanguages(false);
        }
    };

    useEffect(() => {
        fetchKnowledgeBases();
        fetchJSTemplates();
    }, []);

    // 当TTS Provider变化时，重新加载音色列表和语言列表
    useEffect(() => {
        const provider = ttsProvider || 'tencent'; // 如果没有provider，使用默认的腾讯云音色列表（向后兼容）
        fetchVoiceOptions(provider, selectedSpeaker);
        fetchLanguageOptions(provider, language);
    }, [ttsProvider]); // 只依赖ttsProvider，selectedSpeaker和language的变化不影响重新加载
    const safeKnowledgeBases = localKnowledgeBases;
    const navigate = useNavigate();
    const [expandedSections, setExpandedSections] = useState({
        api: true,
        call: true,
        assistant: true,
        integration: true,
        knowledge: true,
        voiceClone: true,
        vad: true,
    })
    const toggleSection = (section: keyof typeof expandedSections) => {
        setExpandedSections(prev => ({
            ...prev,
            [section]: !prev[section]
        }))
    }

    const SectionHeader: React.FC<{
        title: string
        icon: React.ReactNode
        section: keyof typeof expandedSections
        children?: React.ReactNode
    }> = ({ title, icon, section, children }) => (
        <motion.div
            className="flex justify-between items-center cursor-pointer group"
            onClick={() => toggleSection(section)}
            whileHover={{ scale: 1.02 }}
            whileTap={{ scale: 0.98 }}
        >
            <div className="flex items-center">
                <h3 className="text-lg font-semibold flex items-center">
                    {icon}
                    <span className="ml-2">{title}</span>
                </h3>
                <motion.div
                    animate={{ rotate: expandedSections[section] ? 0 : -90 }}
                    transition={{ duration: 0.2 }}
                    className="ml-2"
                >
                    <ChevronDown className="w-4 h-4 text-gray-500 group-hover:text-purple-600 transition-colors" />
                </motion.div>
            </div>
            {children}
        </motion.div>
    )

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
                    <SectionHeader
                        title={t('controlPanel.api.title')}
                        icon={<Key className="w-5 h-5" />}
                        section="api"
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
                    <SectionHeader
                        title={t('controlPanel.call.title')}
                        icon={<Settings className="w-5 h-5" />}
                        section="call"
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
                                    {/* 语言选择 */}
                                    <div className="space-y-2">
                                        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                                            {t('controlPanel.call.language')}
                                            {ttsProvider && (
                                                <span className="ml-2 text-xs text-gray-500 dark:text-gray-400">
                          ({ttsProvider})
                        </span>
                                            )}
                                        </label>
                                        {loadingLanguages ? (
                                            <div className="w-full p-3 text-sm text-gray-500 dark:text-gray-400 text-center border border-gray-200 dark:border-gray-700 rounded-lg bg-gray-50 dark:bg-gray-800">
                                                {t('controlPanel.call.loadingLanguages')}
                                            </div>
                                        ) : languageOptions.length > 0 ? (
                                            <Select
                                                value={language}
                                                onValueChange={onLanguageChange}
                                                className="w-full"
                                            >
                                                <SelectTrigger>
                                                    <SelectValue placeholder={t('controlPanel.call.languagePlaceholder')}>
                                                        {languageOptions.find(l => l.code === language)
                                                            ? `${languageOptions.find(l => l.code === language)?.name} (${languageOptions.find(l => l.code === language)?.nativeName})`
                                                            : t('controlPanel.call.languagePlaceholder')}
                                                    </SelectValue>
                                                </SelectTrigger>
                                                <SelectContent>
                                                    {languageOptions.map(lang => (
                                                        <SelectItem key={lang.code} value={lang.code}>
                                                            <div className="flex flex-col">
                                                                <span className="font-medium">{lang.name}</span>
                                                                <span className="text-xs text-gray-500 dark:text-gray-400">
                                  {lang.nativeName} · {lang.description}
                                </span>
                                                            </div>
                                                        </SelectItem>
                                                    ))}
                                                </SelectContent>
                                            </Select>
                                        ) : (
                                            <select
                                                value={language}
                                                onChange={(e) => onLanguageChange(e.target.value)}
                                                className="w-full p-2 text-sm border rounded-lg focus:ring-2 focus:ring-purple-500 dark:bg-neutral-700 dark:border-neutral-600"
                                            >
                                                {DEFAULT_LANGUAGES.map(lang => (
                                                    <option key={lang.value} value={lang.value}>
                                                        {lang.label}
                                                    </option>
                                                ))}
                                            </select>
                                        )}
                                    </div>

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
                                                <SelectContent>
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
                    <SectionHeader
                        title={t('controlPanel.assistant.title')}
                        icon={<Settings className="w-5 h-5" />}
                        section="assistant"
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

                                        {/* 图数据库长期记忆开关 */}
                                        <div className="space-y-2">
                                            <div className="flex items-center justify-between">
                                                <div className="flex-1">
                                                    <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
                          {t('controlPanel.assistant.graphMemoryTitle') || 'Neo4j 长期记忆'}
                                            </label>
                                                    <p className="text-xs text-gray-500 dark:text-gray-400">
                                                        {t('controlPanel.assistant.graphMemoryDesc') || '开启后，将把该助手的对话写入 Neo4j，用于个性化记忆和知识图谱。'}
                                                    </p>
                                                </div>
                                                <div className="ml-4 flex-shrink-0">
                                                    <Switch
                                                        checked={enableGraphMemory || false}
                                                        onCheckedChange={(checked) => {
                                                    if (onEnableGraphMemoryChange) {
                                                                onEnableGraphMemoryChange(checked);
                                                    }
                                                }}
                                                        size="md"
                                                        className="flex-shrink-0"
                        />
                                                </div>
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
                                            <label className="text-xs text-gray-500 dark:text-gray-400">{t('controlPanel.assistant.icon')}</label>
                                            <div className="grid grid-cols-5 gap-2">
                                                {Object.keys(ICON_MAP).map(iconName => (
                                                    <button
                                                        key={iconName}
                                                        onClick={() => onAssistantIconChange(iconName)}
                                                        className={cn(
                                                            'p-2 rounded-lg transition-colors border-2',
                                                            assistantIcon === iconName
                                                                ? 'bg-purple-100 dark:bg-purple-900/30 border-purple-500'
                                                                : 'hover:bg-gray-100 dark:hover:bg-neutral-600 border-transparent'
                                                        )}
                                                    >
                                                        {ICON_MAP[iconName as keyof typeof ICON_MAP]}
                                                    </button>
                                                ))}
                                            </div>
                                        </div>
                                    </div>

                                    {/* JS模板选择 */}
                                    <div>
                                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                                            {t('controlPanel.assistant.jsTemplate')}
                                        </label>
                                        <Select value={selectedJSTemplate || ""} onValueChange={onJSTemplateChange}>
                                            <SelectTrigger
                                                className="w-full h-10 px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-800 hover:border-gray-400 dark:hover:border-gray-500 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                                                selectedValue={
                                                    selectedJSTemplate ?
                                                        jsTemplates.find(t => t.jsSourceId === selectedJSTemplate)?.name || '未知模板'
                                                        : t('controlPanel.assistant.jsTemplateDefault')
                                                }
                                            >
                                                <SelectValue placeholder={t('controlPanel.assistant.jsTemplatePlaceholder')} />
                                            </SelectTrigger>
                                            <SelectContent className="z-50 max-h-60 overflow-auto">
                                                <SelectItem value="" className="px-3 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer">
                                                    <div className="flex items-center gap-2">
                            <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200">
                              默认
                            </span>
                                                        {t('controlPanel.assistant.jsTemplateDefault')}
                                                    </div>
                                                </SelectItem>
                                                {jsTemplates.map((template) => (
                                                    <SelectItem key={template.id} value={template.jsSourceId} className="px-3 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer">
                                                        <div className="flex items-center gap-2">
                              <span className={cn(
                                  'inline-flex items-center px-2 py-0.5 rounded text-xs font-medium',
                                  template.type === 'default'
                                      ? 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-100'
                                      : 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-100'
                              )}>
                                {template.type === 'default' ? '默认' : '自定义'}
                              </span>
                                                            <span className="truncate">{template.name}</span>
                                                        </div>
                                                    </SelectItem>
                                                ))}
                                            </SelectContent>
                                        </Select>
                                        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                                            {t('controlPanel.assistant.jsTemplateHint')}
                                        </p>
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
                                            leftIcon={<Settings className="w-4 h-4" />}
                                            className="flex-1"
                                        >
                                            {t('controlPanel.assistant.save')}
                                        </Button>
                                    </div>
                                </div>
                            </motion.div>
                        )}
                    </AnimatePresence>
                </motion.div>
                {/* 知识库配置 */}
                <motion.div
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.15 }}
                    className="space-y-4"
                >
                    <SectionHeader
                        title={t('controlPanel.knowledge.title')}
                        icon={<AppWindow className="w-5 h-5" />}
                        section="knowledge"
                    />

                    <AnimatePresence>
                        {expandedSections.knowledge && (
                            <motion.div
                                initial={{ height: 0, opacity: 0 }}
                                animate={{ height: 'auto', opacity: 1 }}
                                exit={{ height: 0, opacity: 0 }}
                                transition={{ duration: 0.3, ease: 'easeInOut' }}
                                className="overflow-hidden"
                            >
                                <div className="space-y-4 pt-4">
                                    <div className="space-y-2">
                                        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('controlPanel.knowledge.select')}</label>
                                        <select
                                            value={selectedKnowledgeBase || ''}
                                            onChange={(e) => onKnowledgeBaseChange(e.target.value)}
                                            className="w-full p-2 text-sm border rounded-lg focus:ring-2 focus:ring-purple-500 dark:bg-neutral-700 dark:border-neutral-600"
                                        >
                                            <option value="">{t('controlPanel.knowledge.none')}</option>
                                            {Array.isArray(safeKnowledgeBases) && safeKnowledgeBases.length > 0 ? (
                                                safeKnowledgeBases.map((kb) => (
                                                    <option key={kb.id} value={kb.id}>
                                                        {kb.name}
                                                    </option>
                                                ))
                                            ) : null}
                                        </select>
                                        {selectedKnowledgeBase && localKnowledgeBases && (
                                            <div className="mt-3 p-3 bg-purple-50 dark:bg-purple-900/20 border border-purple-200 dark:border-purple-800 rounded-lg">
                                                <p className="text-sm text-purple-700 dark:text-purple-300">
                                                    {t('controlPanel.knowledge.current')} <span className="font-medium">{localKnowledgeBases.find((kb) => kb.id === selectedKnowledgeBase)?.name}</span>
                                                </p>
                                            </div>
                                        )}
                                    </div>

                                    <div className="flex space-x-2">
                                        <Button
                                            onClick={handleRefreshKnowledgeBases}
                                            variant="outline"
                                            size="sm"
                                            leftIcon={<RefreshCw className="w-4 h-4" />}
                                            className="flex-1"
                                        >
                                            {t('controlPanel.knowledge.refresh')}
                                        </Button>
                                        <Button
                                            onClick={() => navigate('/knowledge')}
                                            variant="primary"
                                            size="sm"
                                            leftIcon={<ExternalLink className="w-4 h-4" />}
                                            className="flex-1"
                                        >
                                            {t('controlPanel.knowledge.manage')}
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
                    <SectionHeader
                        title={t('controlPanel.voiceClone.title')}
                        icon={<Settings className="w-5 h-5" />}
                        section="voiceClone"
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
                                                <SelectContent>
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
                    <SectionHeader
                        title={t('controlPanel.vad.title')}
                        icon={<Mic className="w-5 h-5" />}
                        section="vad"
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
                    <SectionHeader
                        title={t('controlPanel.integration.title')}
                        icon={<AppWindow className="w-5 h-5" />}
                        section="integration"
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

                                    <Card
                                        variant="filled"
                                        padding="sm"
                                        className="mt-4"
                                    >
                                        <p className="text-xs text-gray-600 dark:text-gray-400 text-center">
                                            {t('controlPanel.integration.hint')}
                                        </p>
                                    </Card>
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
