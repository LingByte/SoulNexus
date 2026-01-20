import React, { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { showAlert } from '@/utils/notification'
import { useI18nStore } from '@/stores/i18nStore'
import Card, { CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from '@/components/UI/Card'
import Input from '@/components/UI/Input'
import Button from '@/components/UI/Button'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/UI/Select'
import FileUpload from '@/components/UI/FileUpload'
import FormField from '@/components/Forms/FormField'
import { Upload, RefreshCw, History, Play, Pause, Volume2, Trash2, Edit3, Zap, Settings, Save } from 'lucide-react'
import { get, post } from '@/utils/request'
import { getSystemInit, saveVoiceCloneConfig } from '@/api/system'
import { getApiBaseURL } from '@/config/apiConfig'

interface VoiceClone {
    id: number
    voiceName: string
    voiceDescription: string
    isActive: boolean
    createdAt: string
    audioUrl?: string
    provider?: string
    assetId?: string // 音色ID（speaker_id）
}

interface SynthesisRecord {
    id: number
    voiceCloneId: number
    text: string
    audioUrl: string
    createdAt: string
}

const VoiceTrainingVolcengine: React.FC = () => {
    const { t } = useI18nStore()
    const navigate = useNavigate()

    const [speakerId, setSpeakerId] = useState('')
    const [uploading, setUploading] = useState(false)
    const [querying, setQuerying] = useState(false)
    const [taskStatus, setTaskStatus] = useState<any>(null)

    // 音色管理相关状态
    const [voiceClones, setVoiceClones] = useState<VoiceClone[]>([])
    const [loadingClones, setLoadingClones] = useState(false)
    const [synthesisHistory, setSynthesisHistory] = useState<SynthesisRecord[]>([])
    const [loadingHistory, setLoadingHistory] = useState(false)
    const [activeTab, setActiveTab] = useState<'training' | 'clones' | 'history'>('training')

    // 音频播放相关状态
    const [playingAudio, setPlayingAudio] = useState<string | null>(null)
    const [audioRef, setAudioRef] = useState<HTMLAudioElement | null>(null)

    // 音色管理相关状态
    const [editingClone, setEditingClone] = useState<VoiceClone | null>(null)
    const [editName, setEditName] = useState('')
    const [editDescription, setEditDescription] = useState('')
    const [synthesisText, setSynthesisText] = useState('')
    const [synthesizing, setSynthesizing] = useState(false)
    const [selectedCloneForSynthesis, setSelectedCloneForSynthesis] = useState<number | null>(null)
    const [configChecked, setConfigChecked] = useState(false)
    const [configConfigured, setConfigConfigured] = useState(false)
    const [configData, setConfigData] = useState<any>(null)
    const [configForm, setConfigForm] = useState({
        app_id: '',
        token: '',
        cluster: 'volcano_icl',
        voice_type: '',
        encoding: 'pcm',
        frame_duration: '20ms',
        sample_rate: 8000,
        bit_depth: 16,
        channels: 1,
        speed_ratio: 1.0,
        training_times: 1
    })
    const [savingConfig, setSavingConfig] = useState(false)

    useEffect(() => {
        checkConfig()
    }, [])

    useEffect(() => {
        if (configChecked && configConfigured) {
            refreshVoiceClones()
            refreshSynthesisHistory()
        }
    }, [configChecked, configConfigured])

    const checkConfig = async () => {
        try {
            const response = await getSystemInit()
            if (response.code === 200 && response.data) {
                const volcengineConfig = response.data.voiceClone?.volcengine
                const configured = volcengineConfig?.configured || false
                setConfigConfigured(configured)
                setConfigData(volcengineConfig)
                if (volcengineConfig?.config) {
                    setConfigForm({
                        app_id: volcengineConfig.config.app_id || '',
                        token: volcengineConfig.config.token || '',
                        cluster: volcengineConfig.config.cluster || 'volcano_icl',
                        voice_type: volcengineConfig.config.voice_type || '',
                        encoding: volcengineConfig.config.encoding || 'pcm',
                        frame_duration: volcengineConfig.config.frame_duration || '20ms',
                        sample_rate: volcengineConfig.config.sample_rate || 8000,
                        bit_depth: volcengineConfig.config.bit_depth || 16,
                        channels: volcengineConfig.config.channels || 1,
                        speed_ratio: volcengineConfig.config.speed_ratio || 1.0,
                        training_times: volcengineConfig.config.training_times || 1
                    })
                }
                setConfigChecked(true)
            } else {
                setConfigChecked(true)
            }
        } catch (err: any) {
            console.error('检查配置失败:', err)
            setConfigChecked(true)
        }
    }

    const handleSaveConfig = async () => {
        try {
            setSavingConfig(true)
            const response = await saveVoiceCloneConfig({
                provider: 'volcengine',
                config: configForm
            })
            if (response.code === 200) {
                showAlert('配置保存成功', 'success')
                setConfigConfigured(true)
                await checkConfig()
            } else {
                throw new Error(response.msg || '保存配置失败')
            }
        } catch (err: any) {
            console.error('保存配置失败:', err)
            showAlert(err?.message || '保存配置失败', 'error')
        } finally {
            setSavingConfig(false)
        }
    }

    const refreshVoiceClones = async () => {
        try {
            setLoadingClones(true)
            const response = await get('/voice/clones?provider=volcengine')
            const list = response.data || []
            setVoiceClones(list.map((x: any) => ({
                id: x.id ?? x.ID,
                voiceName: x.voiceName || x.voice_name || '',
                voiceDescription: x.voiceDescription || x.voice_description || '',
                isActive: x.IsActive ?? x.is_active ?? false,
                createdAt: x.createdAt || x.created_at || '',
                provider: x.provider || 'volcengine',
                assetId: x.assetId || x.asset_id || '' // 音色ID（speaker_id）
            })))
        } catch (err: any) {
            console.error('获取音色列表失败:', err)
            showAlert(err?.message || t('voiceTraining.messages.fetchClonesFailed'), 'error')
        } finally {
            setLoadingClones(false)
        }
    }

    const refreshSynthesisHistory = async () => {
        try {
            setLoadingHistory(true)
            // 直接按 provider 过滤，不依赖 voiceClones 数组
            const response = await get('/voice/synthesis/history?provider=volcengine')
            const list = response.data || []
            setSynthesisHistory(list.map((x: any) => ({
                id: x.id ?? x.ID,
                voiceCloneId: x.voiceCloneId ?? x.voice_clone_id,
                text: x.text || '',
                audioUrl: x.audioUrl || x.audio_url || '',
                createdAt: x.createdAt || x.created_at || ''
            })))
        } catch (err: any) {
            console.error('获取合成历史失败:', err)
            showAlert(err?.message || t('voiceTraining.messages.fetchHistoryFailed'), 'error')
        } finally {
            setLoadingHistory(false)
        }
    }

    // 音频播放功能
    const playAudio = (audioUrl: string) => {
        // 停止当前播放的音频
        if (audioRef) {
            audioRef.pause()
            audioRef.currentTime = 0
        }

        // 处理音频URL - 如果是相对路径，添加服务器基础URL
        // 直接使用数据库中的 URL，不做格式转换（因为旧数据可能是 .pcm，新数据是 .wav）
        let fullAudioUrl = audioUrl
        if (audioUrl.startsWith('/media/') || audioUrl.startsWith('/uploads/')) {
            // 从 API base URL 提取基础 URL（去掉 /api 后缀）
            const apiBaseURL = getApiBaseURL()
            const baseURL = apiBaseURL.replace('/api', '')
            fullAudioUrl = `${baseURL}${audioUrl}`
        } else if (audioUrl.startsWith('/') && !audioUrl.startsWith('http://') && !audioUrl.startsWith('https://')) {
            // 如果是其他相对路径，也添加基础 URL
            const apiBaseURL = getApiBaseURL()
            const baseURL = apiBaseURL.replace('/api', '')
            fullAudioUrl = `${baseURL}${audioUrl}`
        }

        // 创建新的音频元素
        const audio = new Audio(fullAudioUrl)
        setAudioRef(audio)
        setPlayingAudio(audioUrl)

        audio.onended = () => {
            setPlayingAudio(null)
            setAudioRef(null)
        }

        audio.onerror = () => {
            showAlert(t('voiceTraining.messages.audioPlayFailed'), 'error')
            setPlayingAudio(null)
            setAudioRef(null)
        }

        audio.play().catch(err => {
            console.error('音频播放失败:', err)
            showAlert(t('voiceTraining.messages.audioPlayFailed'), 'error')
            setPlayingAudio(null)
            setAudioRef(null)
        })
    }

    const stopAudio = () => {
        if (audioRef) {
            audioRef.pause()
            audioRef.currentTime = 0
        }
        setPlayingAudio(null)
        setAudioRef(null)
    }

    // 清理音频资源
    useEffect(() => {
        return () => {
            if (audioRef) {
                audioRef.pause()
                audioRef.currentTime = 0
            }
        }
    }, [audioRef])

    // 提交音频训练
    const handleSubmitAudio = async (file: File) => {
        if (!speakerId.trim()) {
            showAlert(t('voiceTraining.volcengine.messages.enterSpeakerId'), 'warning')
            return
        }

        try {
            setUploading(true)
            const formData = new FormData()
            formData.append('audio', file)
            formData.append('speakerId', speakerId)
            formData.append('language', 'zh-CN')

            await post('/volcengine/task/submit-audio', formData, {
                headers: {
                    'Content-Type': 'multipart/form-data'
                }
            })

            showAlert(t('voiceTraining.volcengine.messages.audioSubmitSuccess'), 'success')
        } catch (err: any) {
            console.error('提交音频失败:', err)
            showAlert(err?.message || '提交音频失败', 'error')
        } finally {
            setUploading(false)
        }
    }

    // 查询训练状态
    const handleQueryStatus = async () => {
        if (!speakerId.trim()) {
            showAlert(t('voiceTraining.volcengine.messages.enterSpeakerIdRequired'), 'warning')
            return
        }

        try {
            setQuerying(true)
            const response = await post('/volcengine/task/query', {
                speakerId: speakerId
            })
            setTaskStatus(response.data)
            showAlert('查询成功', 'success')
        } catch (err: any) {
            console.error('查询状态失败:', err)
            showAlert(err?.message || '查询状态失败', 'error')
        } finally {
            setQuerying(false)
        }
    }

    // 合成语音功能
    const synthesizeVoice = async (clone: VoiceClone) => {
        if (!synthesisText.trim()) {
            showAlert(t('voiceTraining.messages.enterSynthesisText'), 'warning')
            return
        }

        try {
            setSynthesizing(true)
            // 使用 assetId（speaker_id），如果没有则使用 voiceName 作为 fallback
            const assetId = clone.assetId || clone.voiceName
            if (!assetId) {
                showAlert(t('voiceTraining.volcengine.messages.assetIdNotFound'), 'error')
                return
            }
            await post('/volcengine/synthesize', {
                assetId: assetId, // 使用 speaker_id（asset_id）
                text: synthesisText,
                language: 'zh-CN'
            })

            showAlert(t('voiceTraining.messages.synthesisSuccess'), 'success')
            setSynthesisText('')
            refreshSynthesisHistory()
        } catch (err: any) {
            console.error('语音合成失败:', err)
            showAlert(err?.message || t('voiceTraining.messages.synthesisFailed'), 'error')
        } finally {
            setSynthesizing(false)
        }
    }

    const getStatusText = (status: number) => {
        switch (status) {
            case 0: return t('voiceTraining.volcengine.status.notFound')
            case 1: return t('voiceTraining.volcengine.status.training')
            case 2: return t('voiceTraining.volcengine.status.success')
            case 3: return t('voiceTraining.volcengine.status.failed')
            case 4: return t('voiceTraining.volcengine.status.available')
            default: return t('voiceTraining.volcengine.status.unknown')
        }
    }

    // 如果配置未检查完成，显示加载中
    if (!configChecked) {
        return (
            <div className="flex items-center justify-center min-h-screen">
                <div className="text-center">
                    <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
                    <p className="text-gray-600 dark:text-gray-400">检查配置中...</p>
                </div>
            </div>
        )
    }

    // 如果配置未配置，显示配置表单
    if (!configConfigured) {
        return (
            <div className="container mx-auto px-4 py-8">
                <Card>
                    <CardHeader>
                        <CardTitle className="flex items-center gap-2">
                            <Settings className="w-5 h-5" />
                            配置火山引擎音色克隆服务
                        </CardTitle>
                        <CardDescription>
                            请填写以下配置信息以使用音色克隆功能
                        </CardDescription>
                    </CardHeader>
                    <CardContent className="pt-6">
                        <div className="space-y-4">
                            <FormField label="App ID" required>
                                <Input
                                    value={configForm.app_id}
                                    onChange={(e) => setConfigForm({ ...configForm, app_id: e.target.value })}
                                    placeholder="请输入 App ID"
                                />
                            </FormField>
                            <FormField label="Token" required>
                                <Input
                                    type="password"
                                    value={configForm.token}
                                    onChange={(e) => setConfigForm({ ...configForm, token: e.target.value })}
                                    placeholder="请输入 Token"
                                />
                            </FormField>
                            <FormField label="Cluster">
                                <Input
                                    value={configForm.cluster}
                                    onChange={(e) => setConfigForm({ ...configForm, cluster: e.target.value })}
                                    placeholder="volcano_icl"
                                />
                            </FormField>
                            <FormField label="Voice Type">
                                <Input
                                    value={configForm.voice_type}
                                    onChange={(e) => setConfigForm({ ...configForm, voice_type: e.target.value })}
                                    placeholder="请输入音色类型"
                                />
                            </FormField>
                            <FormField label="Encoding">
                                <Input
                                    value={configForm.encoding}
                                    onChange={(e) => setConfigForm({ ...configForm, encoding: e.target.value })}
                                    placeholder="pcm"
                                />
                            </FormField>
                            <FormField label="Frame Duration">
                                <Input
                                    value={configForm.frame_duration}
                                    onChange={(e) => setConfigForm({ ...configForm, frame_duration: e.target.value })}
                                    placeholder="20ms"
                                />
                            </FormField>
                        </div>
                    </CardContent>
                    <CardFooter>
                        <Button
                            onClick={handleSaveConfig}
                            disabled={savingConfig || !configForm.app_id || !configForm.token}
                            leftIcon={<Save className="w-4 h-4" />}
                        >
                            {savingConfig ? '保存中...' : '保存配置'}
                        </Button>
                    </CardFooter>
                </Card>
            </div>
        )
    }

    return (
        <div className="min-h-screen bg-gradient-to-br from-slate-50 to-purple-50/20 dark:from-neutral-900 dark:via-neutral-800 dark:to-neutral-900">
            <div className="relative max-w-6xl mx-auto px-4 py-8">
                {/* 返回按钮 */}
                <Button
                    variant="outline"
                    size="sm"
                    onClick={() => navigate('/voice-training')}
                    className="mb-4"
                >
                    ← {t('voiceTraining.back')}
                </Button>

                {/* 页面头部 */}
                <div className="flex items-center justify-between mb-8">
                    <div className="space-y-2">
                        <h1 className="text-2xl font-semibold text-gray-900 dark:text-white flex items-center gap-2">
                            <Zap className="w-6 h-6 text-orange-500" />
                            {t('voiceTraining.volcengine.title')}
                        </h1>
                        <p className="text-gray-600 dark:text-gray-400">
                            {t('voiceTraining.volcengine.subtitle')}
                        </p>
                    </div>
                </div>

                {/* 标签页 */}
                <div className="flex gap-2 mb-6">
                    <Button
                        variant={activeTab === 'training' ? 'primary' : 'outline'}
                        size="sm"
                        onClick={() => setActiveTab('training')}
                    >
                        {t('voiceTraining.volcengine.tab.training')}
                    </Button>
                    <Button
                        variant={activeTab === 'clones' ? 'primary' : 'outline'}
                        size="sm"
                        onClick={() => setActiveTab('clones')}
                    >
                        {t('voiceTraining.volcengine.tab.clones')}
                    </Button>
                    <Button
                        variant={activeTab === 'history' ? 'primary' : 'outline'}
                        size="sm"
                        onClick={() => setActiveTab('history')}
                    >
                        {t('voiceTraining.volcengine.tab.history')}
                    </Button>
                </div>

                {/* 训练任务标签页 */}
                {activeTab === 'training' && (
                    <div className="space-y-6">
                        <Card variant="elevated" padding="lg">
                            <CardHeader>
                                <CardTitle>{t('voiceTraining.volcengine.submitAudio.title')}</CardTitle>
                                <CardDescription>
                                    {t('voiceTraining.volcengine.submitAudio.desc')}
                                </CardDescription>
                            </CardHeader>
                            <CardContent className="space-y-4">
                                <FormField label={t('voiceTraining.volcengine.speakerId')} required>
                                    <Input
                                        value={speakerId}
                                        onValueChange={setSpeakerId}
                                        placeholder={t('voiceTraining.volcengine.speakerIdPlaceholder')}
                                        size="md"
                                    />
                                </FormField>
                                <FormField label={t('voiceTraining.volcengine.uploadAudio')} required>
                                    <FileUpload
                                        accept="audio/*"
                                        onFileSelect={handleSubmitAudio}
                                        disabled={uploading || !speakerId.trim()}
                                    />
                                </FormField>
                                <Button
                                    onClick={handleQueryStatus}
                                    loading={querying}
                                    variant="primary"
                                    size="md"
                                    disabled={!speakerId.trim()}
                                >
                                    {t('voiceTraining.volcengine.queryStatus')}
                                </Button>
                            </CardContent>
                        </Card>

                        {taskStatus && (
                            <Card variant="elevated" padding="lg">
                                <CardHeader>
                                    <CardTitle>{t('voiceTraining.volcengine.taskStatus')}</CardTitle>
                                </CardHeader>
                                <CardContent>
                                    <div className="space-y-2">
                                        <p><strong>{t('voiceTraining.volcengine.speakerId')}:</strong> {taskStatus.speakerId}</p>
                                        <p><strong>{t('voiceTraining.status')}:</strong> {getStatusText(taskStatus.status)}</p>
                                        {taskStatus.failedDesc && (
                                            <p><strong>{t('voiceTraining.volcengine.failedReason')}:</strong> {taskStatus.failedDesc}</p>
                                        )}
                                    </div>
                                </CardContent>
                            </Card>
                        )}
                    </div>
                )}

                {/* 音色管理标签页 */}
                {activeTab === 'clones' && (
                    <div className="space-y-6">
                        {/* 合成语音功能 */}
                        {voiceClones.length > 0 && (
                            <Card variant="elevated" padding="lg">
                                <CardHeader>
                                    <CardTitle>{t('voiceTraining.synthesize.title')}</CardTitle>
                                </CardHeader>
                                <CardContent className="space-y-4">
                                    <FormField label={t('voiceTraining.selectVoice')} required>
                                        <Select
                                            value={selectedCloneForSynthesis?.toString() ?? ''}
                                            onValueChange={(value) => setSelectedCloneForSynthesis(value === '' ? null : Number(value))}
                                        >
                                            <SelectTrigger>
                                                <SelectValue placeholder={t('voiceTraining.selectVoicePlaceholder')}>
                                                    {selectedCloneForSynthesis === null
                                                        ? t('voiceTraining.selectVoicePlaceholder')
                                                        : voiceClones.find(vc => vc.id === selectedCloneForSynthesis)?.voiceName || t('voiceTraining.unknownVoice')
                                                    }
                                                </SelectValue>
                                            </SelectTrigger>
                                            <SelectContent>
                                                {voiceClones.map(clone => (
                                                    <SelectItem key={clone.id} value={clone.id.toString()}>
                                                        {clone.voiceName}
                                                    </SelectItem>
                                                ))}
                                            </SelectContent>
                                        </Select>
                                    </FormField>
                                    <FormField label={t('voiceTraining.synthesizeText')} required>
                                        <Input
                                            value={synthesisText}
                                            onValueChange={setSynthesisText}
                                            placeholder={t('voiceTraining.synthesizeTextPlaceholder')}
                                            size="md"
                                        />
                                    </FormField>
                                </CardContent>
                                <CardFooter>
                                    <Button
                                        onClick={() => {
                                            if (!selectedCloneForSynthesis) {
                                                showAlert(t('voiceTraining.messages.selectVoiceFirst'), 'warning')
                                                return
                                            }
                                            const selectedClone = voiceClones.find(vc => vc.id === selectedCloneForSynthesis)
                                            if (!selectedClone) {
                                                showAlert(t('voiceTraining.messages.voiceNotFound'), 'error')
                                                return
                                            }
                                            synthesizeVoice(selectedClone)
                                        }}
                                        loading={synthesizing}
                                        variant="primary"
                                        size="lg"
                                        fullWidth
                                        disabled={!selectedCloneForSynthesis || !synthesisText.trim()}
                                    >
                                        {synthesizing ? t('voiceTraining.synthesizing') : t('voiceTraining.startSynthesize')}
                                    </Button>
                                </CardFooter>
                            </Card>
                        )}

                        {/* 音色列表 */}
                        <Card variant="elevated" padding="lg">
                            <CardHeader>
                                <div className="flex items-center justify-between">
                                    <div>
                                        <CardTitle>{t('voiceTraining.volcengine.myVoices.title')}</CardTitle>
                                        <CardDescription>{t('voiceTraining.volcengine.myVoices.desc')}</CardDescription>
                                    </div>
                                    <Button
                                        onClick={refreshVoiceClones}
                                        variant="outline"
                                        size="sm"
                                        loading={loadingClones}
                                        leftIcon={<RefreshCw className="w-4 h-4" />}
                                    >
                                        {t('voiceTraining.refresh')}
                                    </Button>
                                </div>
                            </CardHeader>
                            <CardContent>
                                {loadingClones ? (
                                    <div className="text-center py-8 text-gray-500">{t('voiceTraining.loadingClones')}</div>
                                ) : voiceClones.length === 0 ? (
                                    <div className="text-center py-8 text-gray-500">{t('voiceTraining.noClones')}</div>
                                ) : (
                                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                        {voiceClones.map((clone) => (
                                            <Card key={clone.id} variant="outlined" padding="md">
                                                <CardContent>
                                                    <h3 className="font-semibold mb-2">{clone.voiceName}</h3>
                                                    <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                                                        {clone.voiceDescription || t('voiceTraining.noVoiceDescription')}
                                                    </p>
                                                </CardContent>
                                            </Card>
                                        ))}
                                    </div>
                                )}
                            </CardContent>
                        </Card>
                    </div>
                )}

                {/* 合成历史标签页 */}
                {activeTab === 'history' && (
                    <Card variant="elevated" padding="lg">
                        <CardHeader>
                            <CardTitle>{t('voiceTraining.synthesisHistory.title')}</CardTitle>
                        </CardHeader>
                        <CardContent>
                            {loadingHistory ? (
                                <div className="text-center py-8 text-gray-500">{t('voiceTraining.loadingHistory')}</div>
                            ) : synthesisHistory.length === 0 ? (
                                <div className="text-center py-8 text-gray-500">{t('voiceTraining.noHistory')}</div>
                            ) : (
                                <div className="space-y-4">
                                    {synthesisHistory.map((record) => (
                                        <Card key={record.id} variant="outlined" padding="md">
                                            <CardContent>
                                                <div className="flex items-start justify-between gap-4">
                                                    <div className="flex-1">
                                                        <p className="mb-2 text-gray-900 dark:text-white">{record.text}</p>
                                                        <p className="text-xs text-gray-500">{new Date(record.createdAt).toLocaleString()}</p>
                                                    </div>
                                                    {record.audioUrl && (
                                                        <div className="flex items-center gap-2">
                                                            {playingAudio === record.audioUrl ? (
                                                                <Button
                                                                    variant="outline"
                                                                    size="sm"
                                                                    onClick={stopAudio}
                                                                    leftIcon={<Pause className="w-4 h-4" />}
                                                                >
                                                                    {t('voiceTraining.stop')}
                                                                </Button>
                                                            ) : (
                                                                <Button
                                                                    variant="outline"
                                                                    size="sm"
                                                                    onClick={() => playAudio(record.audioUrl)}
                                                                    leftIcon={<Play className="w-4 h-4" />}
                                                                >
                                                                    {t('voiceTraining.play')}
                                                                </Button>
                                                            )}
                                                        </div>
                                                    )}
                                                </div>
                                            </CardContent>
                                        </Card>
                                    ))}
                                </div>
                            )}
                        </CardContent>
                    </Card>
                )}
            </div>
        </div>
    )
}

export default VoiceTrainingVolcengine

