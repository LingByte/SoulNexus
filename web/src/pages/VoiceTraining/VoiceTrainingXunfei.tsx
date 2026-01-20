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
import { Upload, RefreshCw, Clock, Mic, History, Play, Pause, Volume2, Trash2, Edit3, Sparkles, Settings, Save } from 'lucide-react'
import { get, post } from '@/utils/request'
import { getSystemInit, saveVoiceCloneConfig } from '@/api/system'
import { getApiBaseURL } from '@/config/apiConfig'

interface TrainingTextSegment {
    id: number
    text_id: number
    seg_id: string
    seg_text: string
    created_at: string
    updated_at: string
}

interface TrainingText {
    id: number
    text_id: number
    text_name: string
    language: string
    is_active: boolean
    created_at: string
    updated_at: string
    text_segments: TrainingTextSegment[]
}

interface TaskInfo {
    taskId: string
    status: number // 后端返回的数字状态：-1=训练中, 0=失败, 1=成功, 2=排队中
    progress?: number
    message?: string
}

interface VoiceClone {
    id: number
    voiceName: string
    voiceDescription: string
    isActive: boolean
    createdAt: string
    audioUrl?: string
}

interface SynthesisRecord {
    id: number
    voiceCloneId: number
    text: string
    audioUrl: string
    createdAt: string
}

const VoiceTrainingXunfei: React.FC = () => {
    const { t } = useI18nStore()
    const navigate = useNavigate()
    
    // 状态转换函数
    const getStatusInfo = (status: number) => {
        switch (status) {
            case -1: return { text: t('voiceTraining.status.training'), color: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400' }
            case 0: return { text: t('voiceTraining.status.failed'), color: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' }
            case 1: return { text: t('voiceTraining.status.success'), color: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' }
            case 2: return { text: t('voiceTraining.status.queued'), color: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400' }
            default: return { text: t('voiceTraining.status.unknown'), color: 'bg-gray-100 text-gray-700 dark:bg-gray-900/30 dark:text-gray-400' }
        }
    }

    const [taskName, setTaskName] = useState('')
    const [sex, setSex] = useState<number>(1) // 1: female, 2: male
    const [ageGroup, setAgeGroup] = useState<number>(2) // 1: child, 2: youth, 3: middle, 4: old
    const [language, setLanguage] = useState('zh-CN')

    const [creating, setCreating] = useState(false)
    const [currentTask, setCurrentTask] = useState<TaskInfo | null>(null)
    const [uploading, setUploading] = useState(false)
    const [polling, setPolling] = useState(false)
    const [selectedTextSegment, setSelectedTextSegment] = useState<TrainingTextSegment | null>(null)

    const [trainingTexts, setTrainingTexts] = useState<TrainingText[]>([])
    const [loadingTexts, setLoadingTexts] = useState(false)

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
    const [selectedCloneForSynthesis, setSelectedCloneForSynthesis] = useState<number | null>(null) // 用于合成的音色ID
    const [configChecked, setConfigChecked] = useState(false)
    const [configConfigured, setConfigConfigured] = useState(false)
    const [configData, setConfigData] = useState<any>(null)
    const [configForm, setConfigForm] = useState({
        app_id: '',
        api_key: '',
        base_url: 'http://opentrain.xfyousheng.com',
        ws_app_id: '',
        ws_api_key: '',
        ws_api_secret: ''
    })
    const [savingConfig, setSavingConfig] = useState(false)

    useEffect(() => {
        checkConfig()
    }, [])

    useEffect(() => {
        if (configChecked && configConfigured) {
            refreshTrainingTexts()
            refreshVoiceClones()
            refreshSynthesisHistory()
        }
    }, [configChecked, configConfigured])

    const checkConfig = async () => {
        try {
            const response = await getSystemInit()
            if (response.code === 200 && response.data) {
                const xunfeiConfig = response.data.voiceClone?.xunfei
                const configured = xunfeiConfig?.configured || false
                setConfigConfigured(configured)
                setConfigData(xunfeiConfig)
                if (xunfeiConfig?.config) {
                    setConfigForm({
                        app_id: xunfeiConfig.config.app_id || '',
                        api_key: xunfeiConfig.config.api_key || '',
                        base_url: xunfeiConfig.config.base_url || 'http://opentrain.xfyousheng.com',
                        ws_app_id: xunfeiConfig.config.ws_app_id || '',
                        ws_api_key: xunfeiConfig.config.ws_api_key || '',
                        ws_api_secret: xunfeiConfig.config.ws_api_secret || ''
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
                provider: 'xunfei',
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

    const refreshTrainingTexts = async () => {
        try {
            setLoadingTexts(true)
            const response = await get('/voice/training-texts')
            console.log('训练文本API响应:', response)

            // 处理不同的响应结构
            let list = []
            if (Array.isArray(response.data)) {
                list = response.data
            } else if (response.data && Array.isArray(response.data.data)) {
                list = response.data.data
            } else if (response.data && response.data.list) {
                list = response.data.list
            } else if (response.data && response.data.text_segments) {
                // 如果返回的是单个训练文本对象，包装成数组
                list = [response.data]
            }

            console.log('处理后的训练文本列表:', list)
            setTrainingTexts(list)
        } catch (err: any) {
            console.error('获取训练文本失败:', err)
            showAlert(err?.message || t('voiceTraining.messages.fetchTextsFailed'), 'error')
        } finally {
            setLoadingTexts(false)
        }
    }


    const refreshVoiceClones = async () => {
        try {
            setLoadingClones(true)
            const response = await get('/voice/clones?provider=xunfei')
            console.log('音色列表API响应:', response)

            // 确保data是数组
            let list = []
            if (Array.isArray(response.data)) {
                list = response.data
            } else if (response.data && Array.isArray(response.data.data)) {
                list = response.data.data
            } else if (response.data && response.data.list) {
                list = response.data.list
            }

            setVoiceClones(list.map((x: any) => ({
                id: x.id ?? x.ID,
                voiceName: x.voiceName || x.voice_name || '',
                voiceDescription: x.voiceDescription || x.voice_description || '',
                isActive: x.isActive ?? x.is_active ?? false,
                createdAt: x.createdAt || x.created_at || ''
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
            const response = await get('/voice/synthesis/history?provider=xunfei')
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

    const deleteSynthesisRecord = async (recordId: number) => {
        try {
            await post('/voice/synthesis/delete', { id: recordId })
            showAlert(t('voiceTraining.messages.deleteRecordSuccess'), 'success')
            refreshSynthesisHistory() // 刷新列表
        } catch (err: any) {
            console.error('删除合成记录失败:', err)
            showAlert(err?.message || t('voiceTraining.messages.deleteRecordFailed'), 'error')
        }
    }

    const createTask = async () => {
        try {
            setCreating(true)
            const response = await post('/voice/training/create', { taskName, sex, ageGroup, language })
            console.log('创建任务响应:', response)

            const taskId = response.data?.task_id || response.data?.taskId
            if (!taskId) {
                console.error('响应数据:', response.data)
                throw new Error(response.msg || '返回缺少taskId')
            }
            setCurrentTask({ taskId, status: 2 }) // 2 = 排队中
            showAlert(t('voiceTraining.messages.createSuccess'), 'success')
        } catch (err: any) {
            console.error('创建任务失败:', err)
            
            // 检查是否是API错误响应，优先显示服务器返回的错误信息
            if (err.response && err.response.data) {
                const errorData = err.response.data
                if (errorData.msg) {
                    showAlert(errorData.msg, 'error')
                } else if (errorData.data) {
                    showAlert(errorData.data, 'error')
                } else {
                    showAlert(err?.message || t('voiceTraining.messages.createFailed'), 'error')
                }
            } else {
                showAlert(err?.message || t('voiceTraining.messages.createFailed'), 'error')
            }
        } finally {
            setCreating(false)
        }
    }

    const submitAudio = async (file: File) => {
        if (!currentTask?.taskId) {
            showAlert(t('voiceTraining.messages.pleaseCreateTask'), 'warning')
            return
        }
        if (!selectedTextSegment) {
            showAlert(t('voiceTraining.messages.selectTextSegment'), 'warning')
            return
        }
        try {
            setUploading(true)
            const form = new FormData()
            form.append('taskId', currentTask.taskId)
            form.append('textSegId', selectedTextSegment.id.toString())
            form.append('audio', file)
            await post('/voice/training/submit-audio', form)
            showAlert(t('voiceTraining.messages.uploadSuccess'), 'success')
        } catch (err: any) {
            console.error('上传音频失败:', err)
            showAlert(err?.message || t('voiceTraining.messages.uploadFailed'), 'error')
        } finally {
            setUploading(false)
        }
    }

    const queryTask = async () => {
        if (!currentTask?.taskId) {
            showAlert(t('voiceTraining.messages.pleaseCreateTask'), 'warning')
            return
        }
        try {
            const response = await post('/voice/training/query', { taskId: currentTask.taskId })
            const status = response.data?.status || 2 // 默认为排队中
            const progress = response.data?.progress
            const message = response.data?.message
            const failedReason = response.data?.failed_reason

            setCurrentTask(prev => prev ? {
                ...prev,
                status,
                progress,
                message: message || failedReason
            } : { taskId: '', status })

            // 如果训练失败，显示失败原因
            if (status === 0 && failedReason) {
                showAlert(`${t('voiceTraining.messages.trainingFailed')}: ${failedReason}`, 'error')
            }

            // 如果训练成功，显示成功信息
            if (status === 1) {
                showAlert(t('voiceTraining.messages.trainingSuccess'), 'success')
            }
        } catch (err: any) {
            console.error('查询任务失败:', err)
            showAlert(err?.message || t('voiceTraining.messages.queryFailed'), 'error')
        }
    }

    const startPolling = () => {
        if (polling) return
        setPolling(true)
        const iv = setInterval(() => {
            queryTask()
        }, 3000)
        const stop = () => {
            clearInterval(iv)
            setPolling(false)
        }
        // 自动在组件卸载或状态变化时停止
        window.addEventListener('beforeunload', stop)
        setTimeout(() => window.removeEventListener('beforeunload', stop), 0)
    }

    // 音频播放功能
    const playAudio = (audioUrl: string) => {
        // 停止当前播放的音频
        if (audioRef) {
            audioRef.pause()
            audioRef.currentTime = 0
        }

        // 处理音频URL - 如果是相对路径，添加服务器基础URL
        let fullAudioUrl = audioUrl
        if (audioUrl.startsWith('/media/') || audioUrl.startsWith('/uploads/')) {
            // 从 API base URL 提取基础 URL（去掉 /api 后缀）
            const apiBaseURL = getApiBaseURL()
            const baseURL = apiBaseURL.replace('/api', '')
            fullAudioUrl = `${baseURL}${audioUrl}`
        } else if (!audioUrl.startsWith('http://') && !audioUrl.startsWith('https://')) {
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

    // 音色试听功能
    const auditionVoice = async (clone: VoiceClone) => {
        if (!clone.audioUrl) {
            showAlert(t('voiceTraining.messages.noAuditionAudio'), 'warning')
            return
        }
        
        playAudio(clone.audioUrl)
    }

    // 编辑音色功能
    const editVoice = (clone: VoiceClone) => {
        setEditingClone(clone)
        setEditName(clone.voiceName)
        setEditDescription(clone.voiceDescription)
    }

    // 保存编辑
    const saveEdit = async () => {
        if (!editingClone) return
        
        try {
            await post('/voice/clones/update', {
                id: editingClone.id,
                voiceName: editName,
                voiceDescription: editDescription
            })
            
            showAlert(t('voiceTraining.messages.updateSuccess'), 'success')
            setEditingClone(null)
            setEditName('')
            setEditDescription('')
            refreshVoiceClones()
        } catch (err: any) {
            console.error('更新音色失败:', err)
            showAlert(err?.message || t('voiceTraining.messages.updateFailed'), 'error')
        }
    }

    // 删除音色功能
    const deleteVoice = async (clone: VoiceClone) => {
        if (!confirm(t('voiceTraining.messages.deleteConfirm').replace('{name}', clone.voiceName))) {
            return
        }
        
        try {
            await post('/voice/clones/delete', { id: clone.id })
            showAlert(t('voiceTraining.messages.deleteSuccess'), 'success')
            refreshVoiceClones()
        } catch (err: any) {
            console.error('删除音色失败:', err)
            showAlert(err?.message || t('voiceTraining.messages.deleteFailed'), 'error')
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
            await post('/voice/synthesize', {
                voiceCloneId: clone.id,
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
                            配置讯飞星火音色克隆服务
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
                            <FormField label="API Key" required>
                                <Input
                                    type="password"
                                    value={configForm.api_key}
                                    onChange={(e) => setConfigForm({ ...configForm, api_key: e.target.value })}
                                    placeholder="请输入 API Key"
                                />
                            </FormField>
                            <FormField label="Base URL">
                                <Input
                                    value={configForm.base_url}
                                    onChange={(e) => setConfigForm({ ...configForm, base_url: e.target.value })}
                                    placeholder="http://opentrain.xfyousheng.com"
                                />
                            </FormField>
                            <FormField label="WebSocket App ID">
                                <Input
                                    value={configForm.ws_app_id}
                                    onChange={(e) => setConfigForm({ ...configForm, ws_app_id: e.target.value })}
                                    placeholder="请输入 WebSocket App ID"
                                />
                            </FormField>
                            <FormField label="WebSocket API Key">
                                <Input
                                    type="password"
                                    value={configForm.ws_api_key}
                                    onChange={(e) => setConfigForm({ ...configForm, ws_api_key: e.target.value })}
                                    placeholder="请输入 WebSocket API Key"
                                />
                            </FormField>
                            <FormField label="WebSocket API Secret">
                                <Input
                                    type="password"
                                    value={configForm.ws_api_secret}
                                    onChange={(e) => setConfigForm({ ...configForm, ws_api_secret: e.target.value })}
                                    placeholder="请输入 WebSocket API Secret"
                                />
                            </FormField>
                        </div>
                    </CardContent>
                    <CardFooter>
                        <Button
                            onClick={handleSaveConfig}
                            disabled={savingConfig || !configForm.app_id || !configForm.api_key}
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
            {/* 背景装饰 */}
            <div className="relative max-w-6xl mx-auto px-4 py-8">
                {/* 页面头部 */}
                <div className="flex items-center justify-between mb-8">
                    <div className="flex items-center gap-4">
                        <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => navigate('/voice-training')}
                            className="text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
                        >
                            ← {t('voiceTraining.back')}
                        </Button>
                        <div className="h-6 w-px bg-gray-300 dark:bg-gray-600"></div>
                        <div className="space-y-1">
                            <h1 className="text-2xl font-semibold text-gray-900 dark:text-white flex items-center gap-2">
                                <Sparkles className="w-6 h-6 text-blue-500" />
                            {t('voiceTraining.title')}
                        </h1>
                        <p className="text-gray-600 dark:text-gray-400 text-sm">
                            {t('voiceTraining.subtitle')}
                        </p>
                    </div>
                    </div>
                </div>

                {/* 标签页导航 */}
                <div className="flex space-x-1 mb-8 bg-white/90 dark:bg-neutral-800/90 p-1 rounded-xl border border-gray-200 dark:border-gray-700">
                    <button
                        onClick={() => setActiveTab('training')}
                        className={`flex-1 flex items-center justify-center gap-2 px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                            activeTab === 'training'
                                ? 'bg-purple-50 text-purple-700 border border-purple-200'
                                : 'text-gray-600 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-700'
                        }`}
                    >
                        <Upload className="w-4 h-4" />
                        {t('voiceTraining.tab.training')}
                    </button>
                    <button
                        onClick={() => setActiveTab('clones')}
                        className={`flex-1 flex items-center justify-center gap-2 px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                            activeTab === 'clones'
                                ? 'bg-purple-50 text-purple-700 border border-purple-200'
                                : 'text-gray-600 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-700'
                        }`}
                    >
                        <Mic className="w-4 h-4" />
                        {t('voiceTraining.tab.clones')}
                    </button>
                    <button
                        onClick={() => setActiveTab('history')}
                        className={`flex-1 flex items-center justify-center gap-2 px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                            activeTab === 'history'
                                ? 'bg-purple-50 text-purple-700 border border-purple-200'
                                : 'text-gray-600 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-700'
                        }`}
                    >
                        <History className="w-4 h-4" />
                        {t('voiceTraining.tab.history')}
                    </button>
                </div>

                {/* 音色训练标签页 */}
                {activeTab === 'training' && (
                    <div className="grid lg:grid-cols-2 gap-6">
                        <Card
                            variant="elevated"
                            padding="lg"
                            className="backdrop-blur-sm bg-white/90 dark:bg-neutral-800/90 border border-gray-200 dark:border-gray-700 shadow-sm hover:border-purple-200 transition-colors"
                            animation="fade"
                            delay={0.1}
                        >
                            <CardHeader>
                                <div className="flex items-center gap-3 mb-2">
                                    <div className="w-8 h-8 bg-purple-50 text-purple-600 dark:bg-neutral-700 dark:text-purple-300 rounded-md flex items-center justify-center">
                                        <Upload className="w-5 h-5 text-white" />
                                    </div>
                                    <div>
                                        <CardTitle className="text-base font-semibold">{t('voiceTraining.createTask.title')}</CardTitle>
                                        <CardDescription className="text-sm">{t('voiceTraining.createTask.desc')}</CardDescription>
                                    </div>
                                </div>
                            </CardHeader>
                            <CardContent className="space-y-4">
                                <FormField label={t('voiceTraining.taskName')} required>
                                    <Input
                                        value={taskName}
                                        onValueChange={setTaskName}
                                        placeholder={t('voiceTraining.taskNamePlaceholder')}
                                        size="md"
                                    />
                                </FormField>

                                <div className="grid grid-cols-2 gap-4">
                                    <FormField label={t('voiceTraining.gender')} required>
                                        <Select value={sex.toString()} onValueChange={(value) => setSex(parseInt(value))}>
                                            <SelectTrigger selectedValue={sex === 1 ? t('voiceTraining.gender.female') : t('voiceTraining.gender.male')}>
                                                <SelectValue placeholder={t('voiceTraining.genderSelect')} />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectItem value="1">{t('voiceTraining.gender.female')}</SelectItem>
                                                <SelectItem value="2">{t('voiceTraining.gender.male')}</SelectItem>
                                            </SelectContent>
                                        </Select>
                                    </FormField>

                                    <FormField label={t('voiceTraining.ageGroup')} required>
                                        <Select value={ageGroup.toString()} onValueChange={(value) => setAgeGroup(parseInt(value))}>
                                            <SelectTrigger selectedValue={
                                                ageGroup === 1 ? t('voiceTraining.ageGroup.child') :
                                                    ageGroup === 2 ? t('voiceTraining.ageGroup.youth') :
                                                        ageGroup === 3 ? t('voiceTraining.ageGroup.middle') : t('voiceTraining.ageGroup.old')
                                            }>
                                                <SelectValue placeholder={t('voiceTraining.ageGroupSelect')} />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectItem value="1">{t('voiceTraining.ageGroup.child')}</SelectItem>
                                                <SelectItem value="2">{t('voiceTraining.ageGroup.youth')}</SelectItem>
                                                <SelectItem value="3">{t('voiceTraining.ageGroup.middle')}</SelectItem>
                                                <SelectItem value="4">{t('voiceTraining.ageGroup.old')}</SelectItem>
                                            </SelectContent>
                                        </Select>
                                    </FormField>
                                </div>

                                <FormField label={t('voiceTraining.language')} required>
                                    <Input
                                        value={language}
                                        onValueChange={setLanguage}
                                        placeholder={t('voiceTraining.languagePlaceholder')}
                                        size="md"
                                    />
                                </FormField>
                            </CardContent>
                            <CardFooter>
                                <Button
                                    onClick={createTask}
                                    loading={creating}
                                    variant="primary"
                                    size="md"
                                    fullWidth
                                    leftIcon={<Upload className="w-4 h-4" />}
                                >
                                    {creating ? t('voiceTraining.creating') : t('voiceTraining.createTask')}
                                </Button>
                            </CardFooter>
                            {currentTask && (
                                <Card
                                    variant="outlined"
                                    padding="sm"
                                    className="mt-6 border border-green-200 dark:border-green-800 bg-green-50/80 dark:bg-green-900/20"
                                >
                                    <CardContent>
                                        <div className="space-y-4">
                                            <div className="flex items-center gap-3">
                                                <div className="w-6 h-6 bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300 rounded-md flex items-center justify-center">
                                                    <Clock className="w-3.5 h-3.5" />
                                                </div>
                                                <span className="text-sm font-semibold text-green-700 dark:text-green-300">{t('voiceTraining.taskStatus')}</span>
                                            </div>
                                            <div className="space-y-3">
                                                <div className="flex justify-between items-center p-3 bg-white/50 dark:bg-neutral-800/50 rounded-lg">
                                                    <span className="text-sm text-gray-600 dark:text-gray-400">{t('voiceTraining.taskId')}</span>
                                                    <span className="font-mono text-xs bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded text-gray-800 dark:text-gray-200">
                          {currentTask.taskId}
                        </span>
                                                </div>
                                                <div className="flex justify-between items-center p-3 bg-white/50 dark:bg-neutral-800/50 rounded-lg">
                                                    <span className="text-sm text-gray-600 dark:text-gray-400">{t('voiceTraining.status')}</span>
                                                    <div className="flex items-center gap-2">
                          <span className={`text-sm font-medium px-2 py-1 rounded-full ${getStatusInfo(currentTask.status).color}`}>
                            {getStatusInfo(currentTask.status).text}
                          </span>
                                                        {currentTask.progress != null && (
                                                            <span className="text-xs bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 px-2 py-1 rounded-full">
                              {currentTask.progress}%
                            </span>
                                                        )}
                                                    </div>
                                                </div>
                                                {currentTask.progress != null && currentTask.progress > 0 && (
                                                    <div className="p-3 bg-white/50 dark:bg-neutral-800/50 rounded-lg">
                                                        <div className="flex justify-between items-center mb-2">
                                                            <span className="text-xs text-gray-600 dark:text-gray-400">{t('voiceTraining.progress')}</span>
                                                            <span className="text-xs text-gray-600 dark:text-gray-400">{currentTask.progress}%</span>
                                                        </div>
                                                        <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                                                            <div
                                                                className="bg-blue-500 h-2 rounded-full transition-all duration-500 ease-out"
                                                                style={{ width: `${currentTask.progress}%` }}
                                                            ></div>
                                                        </div>
                                                    </div>
                                                )}

                                                {/* 状态说明和下一步指导 */}
                                                <div className="p-3 bg-blue-50/80 dark:bg-blue-900/20 rounded-lg">
                                                    <div className="flex items-start gap-3">
                                                        <div className="flex-shrink-0 w-6 h-6 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center">
                                                            <span className="text-blue-600 dark:text-blue-400 text-sm">ℹ</span>
                                                        </div>
                                                        <div className="flex-1">
                                                            <h4 className="text-sm font-medium text-blue-900 dark:text-blue-100 mb-1">
                                                                {getStatusInfo(currentTask.status).text} - {t('voiceTraining.nextStep')}
                                                            </h4>
                                                            <div className="text-sm text-blue-700 dark:text-blue-300">
                                                                {currentTask.status === 2 && (
                                                                    <p>{t('voiceTraining.statusQueuedDesc')}</p>
                                                                )}
                                                                {currentTask.status === -1 && (
                                                                    <p>{t('voiceTraining.statusTrainingDesc')}</p>
                                                                )}
                                                                {currentTask.status === 1 && (
                                                                    <p>{t('voiceTraining.statusSuccessDesc')}</p>
                                                                )}
                                                                {currentTask.status === 0 && (
                                                                    <p>{t('voiceTraining.statusFailedDesc')}</p>
                                                                )}
                                                            </div>
                                                        </div>
                                                    </div>
                                                </div>

                                                {currentTask.message && (
                                                    <div className="p-3 bg-yellow-50/80 dark:bg-yellow-900/20 rounded-lg border border-yellow-200 dark:border-yellow-800">
                                                        <div className="text-xs text-yellow-700 dark:text-yellow-300">
                                                            <span className="font-medium">{t('voiceTraining.note')}</span>
                                                            {currentTask.message}
                                                        </div>
                                                    </div>
                                                )}
                                            </div>
                                            <div className="flex gap-3 pt-2">
                                                <Button
                                                    onClick={queryTask}
                                                    variant="outline"
                                                    size="sm"
                                                    leftIcon={<RefreshCw className="w-3 h-3" />}
                                                    className="flex-1"
                                                >
                                                    {t('voiceTraining.queryStatus')}
                                                </Button>
                                                <Button
                                                    onClick={startPolling}
                                                    variant="outline"
                                                    size="sm"
                                                    leftIcon={<Clock className="w-3 h-3" />}
                                                    className="flex-1"
                                                >
                                                    {t('voiceTraining.startPolling')}
                                                </Button>
                                            </div>
                                        </div>
                                    </CardContent>
                                </Card>
                            )}
                        </Card>

                        <Card
                            variant="elevated"
                            padding="lg"
                            className="backdrop-blur-sm bg-white/90 dark:bg-neutral-800/90 border border-gray-200 dark:border-gray-700 shadow-sm hover:border-purple-200 transition-colors"
                            animation="fade"
                            delay={0.2}
                        >
                            <CardHeader>
                                <div className="flex items-center gap-3 mb-2">
                                    <div className="w-8 h-8 bg-purple-50 text-purple-600 dark:bg-neutral-700 dark:text-purple-300 rounded-md flex items-center justify-center">
                                        <Upload className="w-4 h-4" />
                                    </div>
                                    <div>
                                        <CardTitle className="text-base font-semibold">{t('voiceTraining.uploadAudio.title')}</CardTitle>
                                        <CardDescription className="text-sm">{t('voiceTraining.uploadAudio.desc')}</CardDescription>
                                    </div>
                                </div>
                            </CardHeader>
                            <CardContent className="space-y-4">
                                <div className="p-4 bg-blue-50/70 dark:bg-blue-900/20 rounded-lg border border-blue-200/60 dark:border-blue-800/50">
                                    <div className="text-sm text-blue-700 dark:text-blue-300">
                                        <div className="font-semibold mb-3 flex items-center gap-2">
                                            <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                                            {t('voiceTraining.audioRequirements')}
                                        </div>
                                        <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 text-xs text-blue-600 dark:text-blue-400">
                                            <div className="flex items-center gap-2">
                                                <div className="w-1.5 h-1.5 bg-blue-400 rounded-full"></div>
                                                <span>{t('voiceTraining.audioReq.quiet')}</span>
                                            </div>
                                            <div className="flex items-center gap-2">
                                                <div className="w-1.5 h-1.5 bg-blue-400 rounded-full"></div>
                                                <span>{t('voiceTraining.audioReq.sampleRate')}</span>
                                            </div>
                                            <div className="flex items-center gap-2">
                                                <div className="w-1.5 h-1.5 bg-blue-400 rounded-full"></div>
                                                <span>{t('voiceTraining.audioReq.mono')}</span>
                                            </div>
                                            <div className="flex items-center gap-2">
                                                <div className="w-1.5 h-1.5 bg-blue-400 rounded-full"></div>
                                                <span>{t('voiceTraining.audioReq.duration')}</span>
                                            </div>
                                            <div className="flex items-center gap-2 sm:col-span-2">
                                                <div className="w-1.5 h-1.5 bg-blue-400 rounded-full"></div>
                                                <span>{t('voiceTraining.audioReq.multiple')}</span>
                                            </div>
                                        </div>
                                    </div>
                                </div>

                                {/* 选中的训练文本段落 */}
                                {selectedTextSegment && (
                                    <div className="p-4 bg-green-50/80 dark:bg-green-900/20 rounded-lg border border-green-200 dark:border-green-800 mb-4">
                                        <div className="flex items-start gap-3">
                                            <div className="flex-shrink-0 w-6 h-6 bg-green-500 rounded-full flex items-center justify-center">
                                                <span className="text-xs text-white">✓</span>
                                            </div>
                                            <div className="flex-1">
                                                <h4 className="text-sm font-medium text-green-900 dark:text-green-100 mb-1">
                                                    {t('voiceTraining.selectedSegment')}
                                                </h4>
                                                <p className="text-sm text-green-700 dark:text-green-300 leading-relaxed">
                                                    {selectedTextSegment.seg_text}
                                                </p>
                                                <p className="text-xs text-green-600 dark:text-green-400 mt-2">
                                                    {t('voiceTraining.selectedSegmentDesc')}
                                                </p>
                                            </div>
                                        </div>
                                    </div>
                                )}

                                {!selectedTextSegment && (
                                    <div className="p-4 bg-yellow-50/80 dark:bg-yellow-900/20 rounded-lg border border-yellow-200 dark:border-yellow-800 mb-4">
                                        <div className="flex items-start gap-3">
                                            <div className="flex-shrink-0 w-6 h-6 bg-yellow-500 rounded-full flex items-center justify-center">
                                                <span className="text-xs text-white">!</span>
                                            </div>
                                            <div className="flex-1">
                                                <h4 className="text-sm font-medium text-yellow-900 dark:text-yellow-100 mb-1">
                                                    {t('voiceTraining.selectSegmentFirst')}
                                                </h4>
                                                <p className="text-sm text-yellow-700 dark:text-yellow-300">
                                                    {t('voiceTraining.selectSegmentFirstDesc')}
                                                </p>
                                            </div>
                                        </div>
                                    </div>
                                )}

                                <FileUpload
                                    onFileSelect={(files) => {
                                        if (files.length > 0) {
                                            submitAudio(files[0])
                                        }
                                    }}
                                    accept="audio/*"
                                    multiple={false}
                                    maxSize={50}
                                    maxFiles={1}
                                    label={t('voiceTraining.selectAudioFile')}
                                    disabled={uploading || !selectedTextSegment}
                                />
                            </CardContent>
                        </Card>


                        <Card
                            variant="elevated"
                            padding="lg"
                            className="mt-8 backdrop-blur-sm bg-white/80 dark:bg-neutral-800/80 border-0 shadow-2xl hover:shadow-3xl transition-all duration-500"
                            animation="fade"
                            delay={0.3}
                        >
                            <CardHeader>
                                <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-3">
                                        <div className="w-10 h-10 bg-gradient-to-br from-green-500 to-emerald-600 rounded-xl flex items-center justify-center">
                                            <RefreshCw className="w-5 h-5 text-white" />
                                        </div>
                                        <div>
                                            <CardTitle className="text-xl">{t('voiceTraining.trainingTexts.title')}</CardTitle>
                                            <CardDescription className="text-sm">{t('voiceTraining.trainingTexts.desc')}</CardDescription>
                                        </div>
                                    </div>
                                    <Button
                                        onClick={refreshTrainingTexts}
                                        variant="outline"
                                        size="sm"
                                        loading={loadingTexts}
                                        leftIcon={<RefreshCw className="w-4 h-4" />}
                                        className="shadow-lg hover:shadow-xl transition-all duration-300"
                                    >
                                        {t('voiceTraining.refresh')}
                                    </Button>
                                </div>
                            </CardHeader>
                            <CardContent>
                                <div className="space-y-3 max-h-64 overflow-auto">
                                    {loadingTexts ? (
                                        <div className="flex items-center justify-center py-12">
                                            <div className="text-center">
                                                <div className="w-12 h-12 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-full flex items-center justify-center mx-auto mb-4">
                                                    <RefreshCw className="w-6 h-6 text-white animate-spin" />
                                                </div>
                                                <div className="text-sm text-gray-500">{t('voiceTraining.loadingTexts')}</div>
                                            </div>
                                        </div>
                                    ) : trainingTexts.length === 0 ? (
                                        <div className="flex items-center justify-center py-12">
                                            <div className="text-center">
                                                <div className="w-12 h-12 bg-gray-100 dark:bg-gray-700 rounded-full flex items-center justify-center mx-auto mb-4">
                                                    <RefreshCw className="w-6 h-6 text-gray-400" />
                                                </div>
                                                <div className="text-sm text-gray-500">{t('voiceTraining.noTexts')}</div>
                                            </div>
                                        </div>
                                    ) : (
                                        <div className="space-y-4">
                                            {trainingTexts.map((text, textIndex) => (
                                                <div key={text.id} className="space-y-3">
                                                    <div className="flex items-center gap-3 p-3 bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-blue-900/20 dark:to-indigo-900/20 rounded-lg border border-blue-200 dark:border-blue-800">
                                                        <div className="w-8 h-8 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-full flex items-center justify-center shadow-lg">
                          <span className="text-xs font-bold text-white">
                            {textIndex + 1}
                          </span>
                                                        </div>
                                                        <div className="flex-1">
                                                            <h4 className="font-medium text-blue-900 dark:text-blue-100">{text.text_name}</h4>
                                                            <p className="text-sm text-blue-700 dark:text-blue-300">{t('voiceTraining.segmentsCount').replace('{count}', String(text.text_segments?.length || 0))}</p>
                                                        </div>
                                                    </div>

                                                    {text.text_segments && text.text_segments.length > 0 && (
                                                        <div className="grid gap-2 ml-4">
                                                            {text.text_segments.map((segment, segmentIndex) => (
                                                                <Card
                                                                    key={segment.id}
                                                                    variant="outlined"
                                                                    padding="sm"
                                                                    className={`cursor-pointer transition-all duration-300 hover:border-blue-300 dark:hover:border-blue-600 ${
                                                                        selectedTextSegment?.id === segment.id
                                                                            ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 shadow-lg'
                                                                            : 'border-gray-200 dark:border-gray-700'
                                                                    }`}
                                                                    onClick={() => setSelectedTextSegment(segment)}
                                                                >
                                                                    <CardContent>
                                                                        <div className="flex items-start gap-3">
                                                                            <div className={`flex-shrink-0 w-6 h-6 rounded-full flex items-center justify-center ${
                                                                                selectedTextSegment?.id === segment.id
                                                                                    ? 'bg-blue-500 text-white'
                                                                                    : 'bg-gray-200 dark:bg-gray-700 text-gray-600 dark:text-gray-400'
                                                                            }`}>
                                                                                {selectedTextSegment?.id === segment.id ? (
                                                                                    <span className="text-xs">✓</span>
                                                                                ) : (
                                                                                    <span className="text-xs">{segmentIndex + 1}</span>
                                                                                )}
                                                                            </div>
                                                                            <div className="flex-1 text-sm text-gray-800 dark:text-gray-100 leading-relaxed">
                                                                                {segment.seg_text}
                                                                            </div>
                                                                        </div>
                                                                    </CardContent>
                                                                </Card>
                                                            ))}
                                                        </div>
                                                    )}
                                                </div>
                                            ))}
                                        </div>
                                    )}
                                </div>
                            </CardContent>
                        </Card>
                    </div>
                )}

                {/* 我的音色标签页 */}
                {activeTab === 'clones' && (
                    <div className="space-y-6">
                        <Card
                            variant="elevated"
                            padding="lg"
                            className="backdrop-blur-sm bg-white/90 dark:bg-neutral-800/90 border border-gray-200 dark:border-gray-700 shadow-sm"
                        >
                            <CardHeader>
                                <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-3">
                                        <div className="w-10 h-10 bg-gradient-to-br from-purple-500 to-pink-600 rounded-xl flex items-center justify-center">
                                            <Mic className="w-5 h-5 text-white" />
                                        </div>
                                        <div>
                                            <CardTitle className="text-xl">{t('voiceTraining.myVoices.title')}</CardTitle>
                                            <CardDescription className="text-sm">{t('voiceTraining.myVoices.desc')}</CardDescription>
                                        </div>
                                    </div>
                                    <Button
                                        onClick={refreshVoiceClones}
                                        variant="outline"
                                        size="sm"
                                        loading={loadingClones}
                                        leftIcon={<RefreshCw className="w-4 h-4" />}
                                        className="shadow-lg hover:shadow-xl transition-all duration-300"
                                    >
                                        {t('voiceTraining.refresh')}
                                    </Button>
                                </div>
                            </CardHeader>
                            <CardContent>
                                <div className="space-y-4">
                                    {loadingClones ? (
                                        <div className="flex items-center justify-center py-12">
                                            <div className="text-center">
                                                <div className="w-12 h-12 bg-gradient-to-br from-purple-500 to-pink-600 rounded-full flex items-center justify-center mx-auto mb-4">
                                                    <RefreshCw className="w-6 h-6 text-white animate-spin" />
                                                </div>
                                                <div className="text-sm text-gray-500">{t('voiceTraining.loadingTexts')}</div>
                                            </div>
                                        </div>
                                    ) : voiceClones.length === 0 ? (
                                        <div className="flex items-center justify-center py-12">
                                            <div className="text-center">
                                                <div className="w-12 h-12 bg-gray-100 dark:bg-gray-700 rounded-full flex items-center justify-center mx-auto mb-4">
                                                    <Mic className="w-6 h-6 text-gray-400" />
                                                </div>
                                                <div className="text-sm text-gray-500">{t('voiceTraining.noTexts')}</div>
                                                <div className="text-xs text-gray-400 mt-2">{t('voiceTraining.selectSegmentFirst')}</div>
                                            </div>
                                        </div>
                                    ) : (
                                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                                            {voiceClones.map((clone) => (
                                                <Card
                                                    key={clone.id}
                                                    variant="outlined"
                                                    padding="md"
                                                    className="transition-colors duration-200 border border-gray-200 dark:border-gray-700 hover:border-purple-200 bg-white dark:bg-neutral-800"
                                                >
                                                    <CardContent>
                                                        <div className="space-y-3">
                                                            <div className="flex items-start justify-between">
                                                                <div className="flex-1">
                                                                    <h3 className="font-semibold text-gray-900 dark:text-white text-sm">
                                                                        {clone.voiceName}
                                                                    </h3>
                                                                    <p className="text-xs text-gray-600 dark:text-gray-400 mt-1 line-clamp-2">
                                                                        {clone.voiceDescription || t('voiceTraining.noTexts')}
                                                                    </p>
                                                                </div>
                                                                <div className="flex items-center gap-1">
                                                                    {clone.isActive ? (
                                                                        <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                                                                    ) : (
                                                                        <div className="w-2 h-2 bg-gray-400 rounded-full"></div>
                                                                    )}
                                                                </div>
                                                            </div>
                                                            <div className="text-xs text-gray-500">
                                                                {new Date(clone.createdAt).toLocaleDateString()}
                                                            </div>
                                                            <div className="flex gap-2">
                                                                <Button
                                                                    size="sm"
                                                                    variant="outline"
                                                                    className="flex-1 text-xs"
                                                                    leftIcon={<Play className="w-3 h-3" />}
                                                                    onClick={() => auditionVoice(clone)}
                                                                >
                                                                    {t('voiceTraining.audition')}
                                                                </Button>
                                                                <Button
                                                                    size="sm"
                                                                    variant="outline"
                                                                    className="flex-1 text-xs"
                                                                    leftIcon={<Edit3 className="w-3 h-3" />}
                                                                    onClick={() => editVoice(clone)}
                                                                >
                                                                    {t('voiceTraining.edit')}
                                                                </Button>
                                                                <Button
                                                                    size="sm"
                                                                    variant="outline"
                                                                    className="text-xs text-red-600 hover:text-red-700"
                                                                    leftIcon={<Trash2 className="w-3 h-3" />}
                                                                    onClick={() => deleteVoice(clone)}
                                                                >
                                                                    {t('voiceTraining.delete')}
                                                                </Button>
                                                            </div>
                                                        </div>
                                                    </CardContent>
                                                </Card>
                                            ))}
                                        </div>
                                    )}
                                </div>
                            </CardContent>
                        </Card>

                        {/* 编辑音色模态框 */}
                        {editingClone && (
                            <Card
                                variant="elevated"
                                padding="lg"
                                className="mt-6 backdrop-blur-sm bg-white/90 dark:bg-neutral-800/90 border border-gray-200 dark:border-gray-700 shadow-sm"
                            >
                                <CardHeader>
                                    <div className="flex items-center gap-3">
                                        <div className="w-10 h-10 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-xl flex items-center justify-center">
                                            <Edit3 className="w-5 h-5 text-white" />
                                        </div>
                                        <div>
                                            <CardTitle className="text-xl">{t('voiceTraining.edit')} {t('voiceTraining.myVoices.title')}</CardTitle>
                                            <CardDescription className="text-sm">{t('voiceTraining.myVoices.desc')}</CardDescription>
                                        </div>
                                    </div>
                                </CardHeader>
                                <CardContent className="space-y-4">
                                    <FormField label={t('voiceTraining.voiceName')} required>
                                        <Input
                                            value={editName}
                                            onValueChange={setEditName}
                                            placeholder={t('voiceTraining.voiceName')}
                                            size="md"
                                        />
                                    </FormField>
                                    <FormField label={t('voiceTraining.voiceDescription')}>
                                        <Input
                                            value={editDescription}
                                            onValueChange={setEditDescription}
                                            placeholder={t('voiceTraining.voiceDescription')}
                                            size="md"
                                        />
                                    </FormField>
                                </CardContent>
                                <CardFooter className="flex gap-3">
                                    <Button
                                        onClick={saveEdit}
                                        variant="primary"
                                        size="md"
                                        leftIcon={<Edit3 className="w-4 h-4" />}
                                        className="flex-1"
                                    >
                                        {t('voiceTraining.saveEdit')}
                                    </Button>
                                    <Button
                                        onClick={() => {
                                            setEditingClone(null)
                                            setEditName('')
                                            setEditDescription('')
                                        }}
                                        variant="outline"
                                        size="md"
                                        className="flex-1"
                                    >
                                        {t('voiceTraining.cancel')}
                                    </Button>
                                </CardFooter>
                            </Card>
                        )}

                        {/* 合成语音功能 */}
                        {voiceClones.length > 0 && (
                            <Card
                                variant="elevated"
                                padding="lg"
                                className="mt-6 backdrop-blur-sm bg-white/90 dark:bg-neutral-800/90 border border-gray-200 dark:border-gray-700 shadow-sm"
                            >
                                <CardHeader>
                                    <div className="flex items-center gap-3">
                                        <div className="w-10 h-10 bg-gradient-to-br from-green-500 to-emerald-600 rounded-xl flex items-center justify-center">
                                            <Volume2 className="w-5 h-5 text-white" />
                                        </div>
                                        <div>
                                            <CardTitle className="text-xl">{t('voiceTraining.synthesize.title')}</CardTitle>
                                            <CardDescription className="text-sm">{t('voiceTraining.synthesize.desc')}</CardDescription>
                                        </div>
                                    </div>
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
                                        leftIcon={<Volume2 className="w-4 h-4" />}
                                        disabled={!selectedCloneForSynthesis || !synthesisText.trim()}
                                    >
                                        {synthesizing ? t('voiceTraining.synthesizing') : t('voiceTraining.startSynthesize')}
                                    </Button>
                                </CardFooter>
                            </Card>
                        )}
                    </div>
                )}

                {/* 合成历史标签页 */}
                {activeTab === 'history' && (
                    <div className="space-y-6">
                        <Card
                            variant="elevated"
                            padding="lg"
                            className="backdrop-blur-sm bg-white/90 dark:bg-neutral-800/90 border border-gray-200 dark:border-gray-700 shadow-sm"
                        >
                            <CardHeader>
                                <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-3">
                                        <div className="w-10 h-10 bg-gradient-to-br from-green-500 to-emerald-600 rounded-xl flex items-center justify-center">
                                            <History className="w-5 h-5 text-white" />
                                        </div>
                                        <div>
                                            <CardTitle className="text-xl">{t('voiceTraining.synthesisHistory.title')}</CardTitle>
                                            <CardDescription className="text-sm">{t('voiceTraining.synthesisHistory.desc')}</CardDescription>
                                        </div>
                                    </div>
                                    <Button
                                        onClick={refreshSynthesisHistory}
                                        variant="outline"
                                        size="sm"
                                        loading={loadingHistory}
                                        leftIcon={<RefreshCw className="w-4 h-4" />}
                                        className="shadow-lg hover:shadow-xl transition-all duration-300"
                                    >
                                        {t('voiceTraining.refresh')}
                                    </Button>
                                </div>
                            </CardHeader>
                            <CardContent>
                                <div className="space-y-4">
                                    {loadingHistory ? (
                                        <div className="flex items-center justify-center py-12">
                                            <div className="text-center">
                                                <div className="w-12 h-12 bg-gradient-to-br from-green-500 to-emerald-600 rounded-full flex items-center justify-center mx-auto mb-4">
                                                    <RefreshCw className="w-6 h-6 text-white animate-spin" />
                                                </div>
                                                <div className="text-sm text-gray-500">{t('voiceTraining.loadingHistory')}</div>
                                            </div>
                                        </div>
                                    ) : synthesisHistory.length === 0 ? (
                                        <div className="flex items-center justify-center py-12">
                                            <div className="text-center">
                                                <div className="w-12 h-12 bg-gray-100 dark:bg-gray-700 rounded-full flex items-center justify-center mx-auto mb-4">
                                                    <History className="w-6 h-6 text-gray-400" />
                                                </div>
                                                <div className="text-sm text-gray-500">{t('voiceTraining.noHistory')}</div>
                                                <div className="text-xs text-gray-400 mt-2">{t('voiceTraining.startSynthesizeDesc')}</div>
                                            </div>
                                        </div>
                                    ) : (
                                        <div className="space-y-3">
                                            {synthesisHistory.map((record) => (
                                                <Card
                                                    key={record.id}
                                                    variant="outlined"
                                                    padding="md"
                                                    className="transition-colors duration-200 border border-gray-200 dark:border-gray-700 hover:border-purple-200 bg-white dark:bg-neutral-800"
                                                >
                                                    <CardContent>
                                                        <div className="flex items-start gap-4">
                                                            <div className="flex-shrink-0 w-8 h-8 bg-purple-50 text-purple-600 dark:bg-neutral-700 dark:text-purple-300 rounded-md flex items-center justify-center">
                                                                <Volume2 className="w-4 h-4" />
                                                            </div>
                                                            <div className="flex-1 min-w-0">
                                                                <div className="flex items-start justify-between">
                                                                    <div className="flex-1">
                                                                        <p className="text-sm text-gray-800 dark:text-gray-100 line-clamp-2">
                                                                            {record.text}
                                                                        </p>
                                                                        <div className="flex items-center gap-4 mt-2 text-xs text-gray-500">
                                                                            <span>{t('voiceTraining.voiceCloneId')}: {record.voiceCloneId}</span>
                                                                            <span>{new Date(record.createdAt).toLocaleString()}</span>
                                                                        </div>
                                                                    </div>
                                                                    <div className="flex items-center gap-2 ml-4">
                                                                        {record.audioUrl ? (
                                                                            <Button
                                                                                size="sm"
                                                                                variant="outline"
                                                                                className="text-xs"
                                                                                onClick={() => {
                                                                                    if (playingAudio === record.audioUrl) {
                                                                                        stopAudio()
                                                                                    } else {
                                                                                        playAudio(record.audioUrl)
                                                                                    }
                                                                                }}
                                                                                leftIcon={
                                                                                    playingAudio === record.audioUrl ?
                                                                                        <Pause className="w-3 h-3" /> :
                                                                                        <Play className="w-3 h-3" />
                                                                                }
                                                                            >
                                                                                {playingAudio === record.audioUrl ? t('voiceTraining.pause') : t('voiceTraining.play')}
                                                                            </Button>
                                                                        ) : (
                                                                            <span className="text-xs text-gray-400">{t('voiceTraining.noAudio')}</span>
                                                                        )}
                                                                        <Button
                                                                            size="sm"
                                                                            variant="outline"
                                                                            className="text-xs text-red-600 hover:text-red-700"
                                                                            leftIcon={<Trash2 className="w-3 h-3" />}
                                                                            onClick={() => deleteSynthesisRecord(record.id)}
                                                                        >
                                                                            {t('voiceTraining.delete')}
                                                                        </Button>
                                                                    </div>
                                                                </div>
                                                            </div>
                                                        </div>
                                                    </CardContent>
                                                </Card>
                                            ))}
                                        </div>
                                    )}
                                </div>
                            </CardContent>
                        </Card>
                    </div>
                )}
            </div>
        </div>
    )
}

export default VoiceTrainingXunfei

