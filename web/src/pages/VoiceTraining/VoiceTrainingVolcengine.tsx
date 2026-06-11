import React, { useEffect, useState, useCallback, useRef } from 'react'
import { showAlert } from '@/utils/notification'
import { useI18nStore } from '@/stores/i18nStore'
import Card, { CardHeader, CardTitle, CardContent } from '@/components/UI/Card'
import { Input as ArcoInput } from '@arco-design/web-react'
import Button from '@/components/UI/Button'
import FileUpload from '@/components/UI/FileUpload'
import FormField from '@/components/Forms/FormField'
import { RefreshCw, Play, Pause, Trash2, Zap, Mic, Square, Upload, Download } from 'lucide-react'
import { get, post } from '@/utils/request'
import { getSystemInit } from '@/api/system'
import { getApiBaseURL } from '@/config/apiConfig'

interface VoiceClone {
    id: number
    voiceName: string
    voiceDescription: string
    isActive: boolean
    createdAt: string
    audioUrl?: string
    provider?: string
    assetId?: string
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

    // Training
    const [speakerId, setSpeakerId] = useState('')
    const [uploading, setUploading] = useState(false)
    const [querying, setQuerying] = useState(false)
    const [taskStatus, setTaskStatus] = useState<any>(null)

    // Recording
    const [recording, setRecording] = useState(false)
    const [mediaRecorder, setMediaRecorder] = useState<MediaRecorder | null>(null)
    const [recordedBlob, setRecordedBlob] = useState<Blob | null>(null)
    const [recordingTime, setRecordingTime] = useState(0)
    const recordingTimer = useRef<ReturnType<typeof setInterval> | null>(null)

    // Clones & Synthesis
    const [voiceClones, setVoiceClones] = useState<VoiceClone[]>([])
    const [loadingClones, setLoadingClones] = useState(false)
    const [synthesisHistory, setSynthesisHistory] = useState<SynthesisRecord[]>([])
    const [loadingHistory, setLoadingHistory] = useState(false)
    const [activeTab, setActiveTab] = useState<'training' | 'clones' | 'history'>('training')
    const [playingAudio, setPlayingAudio] = useState<string | null>(null)
    const [audioRef, setAudioRef] = useState<HTMLAudioElement | null>(null)
    const [synthesisText, setSynthesisText] = useState('')
    const [synthesizing, setSynthesizing] = useState(false)
    const [selectedCloneForSynthesis, setSelectedCloneForSynthesis] = useState<number | null>(null)

    // Config
    const [configChecked, setConfigChecked] = useState(false)
    const [configConfigured, setConfigConfigured] = useState(false)

    const checkConfig = useCallback(async () => {
        try {
            const response = await getSystemInit()
            if (response.code === 200 && response.data) {
                setConfigConfigured(response.data.voiceClone?.volcengine?.configured || false)
            }
        } catch (err) { console.error(err) }
        finally { setConfigChecked(true) }
    }, [])

    useEffect(() => { checkConfig() }, [checkConfig])
    useEffect(() => {
        if (configChecked && configConfigured) {
            refreshVoiceClones()
            refreshSynthesisHistory()
        }
    }, [configChecked, configConfigured])

    useEffect(() => {
        return () => { if (recordingTimer.current) clearInterval(recordingTimer.current) }
    }, [])

    // --- Data fetch ---
    const refreshVoiceClones = async () => {
        try {
            setLoadingClones(true)
            const res = await get('/voice/clones?provider=volcengine')
            setVoiceClones((res.data || []).map((x: any) => ({
                id: x.id ?? x.ID,
                voiceName: x.voiceName || x.voice_name || '',
                voiceDescription: x.voiceDescription || x.voice_description || '',
                isActive: x.IsActive ?? x.is_active ?? false,
                createdAt: x.createdAt || x.created_at || '',
                provider: x.provider || 'volcengine',
                assetId: x.assetId || x.asset_id || ''
            })))
        } catch (err: any) { showAlert(err?.message || t('voiceTraining.messages.fetchClonesFailed'), 'error') }
        finally { setLoadingClones(false) }
    }

    const refreshSynthesisHistory = async () => {
        try {
            setLoadingHistory(true)
            const res = await get('/voice/synthesis/history?provider=volcengine')
            setSynthesisHistory((res.data || []).map((x: any) => ({
                id: x.id ?? x.ID,
                voiceCloneId: x.voiceCloneId ?? x.voice_clone_id,
                text: x.text || '',
                audioUrl: x.audioUrl || x.audio_url || '',
                createdAt: x.createdAt || x.created_at || ''
            })))
        } catch (err: any) { showAlert(err?.message || t('voiceTraining.messages.fetchHistoryFailed'), 'error') }
        finally { setLoadingHistory(false) }
    }

    // --- Training: upload & record ---
    const doUpload = async (file: File) => {
        if (!speakerId.trim()) { showAlert(t('voiceTraining.volcengine.speakerIdPlaceholder'), 'warning'); return }
        try {
            setUploading(true)
            const fd = new FormData()
            fd.append('audio', file)
            fd.append('speakerId', speakerId)
            fd.append('language', 'zh-CN')
            await post('/volcengine/task/submit-audio', fd)
            showAlert(t('voiceTraining.messages.uploadSuccess'), 'success')
        } catch (err: any) { showAlert(err?.message || t('voiceTraining.messages.uploadFailed'), 'error') }
        finally { setUploading(false) }
    }

    const startRecording = async () => {
        try {
            const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
            const recorder = new MediaRecorder(stream, { mimeType: 'audio/webm' })
            const chunks: BlobPart[] = []
            recorder.ondataavailable = (e) => { if (e.data.size > 0) chunks.push(e.data) }
            recorder.onstop = () => {
                const blob = new Blob(chunks, { type: 'audio/webm' })
                setRecordedBlob(blob)
                stream.getTracks().forEach(t => t.stop())
            }
            recorder.start()
            setMediaRecorder(recorder)
            setRecording(true)
            setRecordingTime(0)
            recordingTimer.current = setInterval(() => setRecordingTime(p => p + 1), 1000)
        } catch (err: any) {
            showAlert(t('voiceTraining.volcengine.recordingFailed'), 'error')
        }
    }

    const stopRecording = () => {
        if (mediaRecorder && mediaRecorder.state !== 'inactive') mediaRecorder.stop()
        if (recordingTimer.current) { clearInterval(recordingTimer.current); recordingTimer.current = null }
        setRecording(false)
    }

    const submitRecording = async () => {
        if (!recordedBlob) return
        const file = new File([recordedBlob], `recording_${Date.now()}.webm`, { type: 'audio/webm' })
        setRecordedBlob(null)
        await doUpload(file)
    }

    const handleQueryStatus = async () => {
        if (!speakerId.trim()) return
        try {
            setQuerying(true)
            const res = await post('/volcengine/task/query', { speakerId })
            setTaskStatus(res.data)
            showAlert(t('voiceTraining.messages.querySuccess'), 'success')
        } catch (err: any) { showAlert(err?.message || t('voiceTraining.messages.queryFailed'), 'error') }
        finally { setQuerying(false) }
    }

    // --- Synthesis ---
    const synthesizeVoice = async (clone: VoiceClone) => {
        if (!synthesisText.trim()) { showAlert(t('voiceTraining.messages.enterSynthesisText'), 'warning'); return }
        const assetId = clone.assetId || clone.voiceName
        if (!assetId) { showAlert(t('voiceTraining.volcengine.messages.assetIdNotFound'), 'error'); return }
        try {
            setSynthesizing(true)
            await post('/volcengine/synthesize', { assetId, text: synthesisText, language: 'zh-CN' })
            showAlert(t('voiceTraining.messages.synthesisSuccess'), 'success')
            setSynthesisText('')
            refreshSynthesisHistory()
        } catch (err: any) { showAlert(err?.message || t('voiceTraining.messages.synthesisFailed'), 'error') }
        finally { setSynthesizing(false) }
    }

    // --- Audio playback ---
    const playAudio = (url: string) => {
        if (audioRef) { audioRef.pause(); audioRef.currentTime = 0 }
        let full = url
        if (url.startsWith('/media/') || url.startsWith('/uploads/')) {
            full = `${getApiBaseURL().replace('/api', '')}${url}`
        } else if (url.startsWith('/') && !url.startsWith('http')) {
            full = `${window.location.origin}${url}`
        }
        const a = new Audio(full)
        a.onended = () => setPlayingAudio(null)
        a.onerror = () => { setPlayingAudio(null); showAlert(t('voiceTraining.messages.audioPlayFailed'), 'error') }
        a.play(); setAudioRef(a); setPlayingAudio(url)
    }
    const stopAudio = () => { if (audioRef) { audioRef.pause(); audioRef.currentTime = 0 }; setPlayingAudio(null) }

    // --- Delete ---
    const deleteSynthesisRecord = async (id: number) => {
        try { await post('/voice/synthesis/delete', { id }); showAlert(t('voiceTraining.messages.deleteRecordSuccess'), 'success'); refreshSynthesisHistory() }
        catch (err: any) { showAlert(err?.message || t('voiceTraining.messages.deleteRecordFailed'), 'error') }
    }
    const deleteVoiceClone = async (id: number, name: string) => {
        if (!window.confirm(t('voiceTraining.messages.deleteConfirm', { name }))) return
        try { await post('/voice/clones/delete', { id }); showAlert(t('voiceTraining.messages.deleteSuccess'), 'success'); refreshVoiceClones() }
        catch (err: any) { showAlert(err?.message || t('voiceTraining.messages.deleteFailed'), 'error') }
    }

    const downloadAudio = (audioUrl: string, text: string) => {
        let fullUrl = audioUrl
        if (audioUrl.startsWith('/media/') || audioUrl.startsWith('/uploads/')) {
            fullUrl = `${getApiBaseURL().replace('/api', '')}${audioUrl}`
        } else if (audioUrl.startsWith('/') && !audioUrl.startsWith('http')) {
            fullUrl = `${window.location.origin}${audioUrl}`
        }
        const a = document.createElement('a')
        a.href = fullUrl
        a.download = `${text.slice(0, 30) || 'audio'}.wav`
        a.click()
    }

    const getStatusText = (s: number) => ({
        0: t('voiceTraining.volcengine.status.notFound'),
        1: t('voiceTraining.volcengine.status.training'),
        2: t('voiceTraining.volcengine.status.success'),
        3: t('voiceTraining.volcengine.status.failed'),
        4: t('voiceTraining.volcengine.status.available'),
    } as Record<number, string>)[s] || t('voiceTraining.volcengine.status.unknown')

    const formatTime = (s: number) => `${Math.floor(s / 60).toString().padStart(2, '0')}:${(s % 60).toString().padStart(2, '0')}`

    // --- Loading ---
    if (!configChecked) return (
        <div className="flex items-center justify-center min-h-[60vh]">
            <div className="animate-spin rounded-full h-8 w-8 border-2 border-gray-300 border-t-gray-600" />
        </div>
    )

    // --- Config not set ---
    if (!configConfigured) return (
        <div className="max-w-xl mx-auto py-12 px-4">
            <Card>
                <CardHeader><CardTitle className="flex items-center gap-2 text-lg"><Zap className="w-5 h-5 text-orange-500" />{t('voiceTraining.volcengine.config.title')}</CardTitle></CardHeader>
                <CardContent>
                    <div className="text-center py-8">
                        <p className="text-gray-500 mb-4">{t('voiceTraining.volcengine.config.envHint')}</p>
                        <div className="bg-gray-50 dark:bg-gray-800/50 rounded-lg p-4 text-left text-sm font-mono space-y-1">
                            <p>VOLCENGINE_CLONE_APP_ID=...</p>
                            <p>VOLCENGINE_CLONE_TOKEN=...</p>
                            <p>VOLCENGINE_CLONE_CLUSTER=volcano_icl</p>
                        </div>
                    </div>
                </CardContent>
            </Card>
        </div>
    )

    // --- Main render ---
    const selectedClone = voiceClones.find(c => c.id === selectedCloneForSynthesis)

    return (
        <div className="max-w-6xl mx-auto py-6 px-4 space-y-4">
            {/* Tabs */}
            <div className="flex gap-1 p-1 bg-gray-100 dark:bg-gray-800 rounded-lg w-fit">
                {(['training', 'clones', 'history'] as const).map(tab => (
                    <button key={tab} onClick={() => setActiveTab(tab)}
                        className={`px-3 py-1.5 rounded-md text-sm font-medium transition-all ${
                            activeTab === tab ? 'bg-white dark:bg-gray-700 shadow-sm text-gray-900 dark:text-white' : 'text-gray-500 hover:text-gray-700 dark:text-gray-400'
                        }`}>
                        {t(`voiceTraining.volcengine.tab.${tab}`)}
                    </button>
                ))}
            </div>

            {/* ===== Training Tab ===== */}
            {activeTab === 'training' && (
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                    {/* Left: Upload */}
                    <Card>
                        <CardHeader><CardTitle className="text-base flex items-center gap-2"><Upload className="w-4 h-4" />{t('voiceTraining.volcengine.submitAudio.title')}</CardTitle></CardHeader>
                        <CardContent className="space-y-3">
                            <FormField label={t('voiceTraining.volcengine.speakerId')} required>
                                <ArcoInput size="large" className="!h-9 !text-sm" value={speakerId}
                                    onChange={(e) => setSpeakerId(e)}
                                    placeholder={t('voiceTraining.volcengine.speakerIdPlaceholder')} />
                            </FormField>
                            <FormField label={t('voiceTraining.volcengine.uploadAudio')} required>
                                <FileUpload accept="audio/*"
                                    hint={t('voiceTraining.volcengine.uploadAudio.hint')}
                                    onFileSelect={(files: File[]) => { if (files[0]) doUpload(files[0]) }}
                                    disabled={uploading || !speakerId.trim()} />
                            </FormField>
                        </CardContent>
                    </Card>

                    {/* Right: Record + Query */}
                    <Card>
                        <CardHeader><CardTitle className="text-base flex items-center gap-2"><Mic className="w-4 h-4" />{t('voiceTraining.volcengine.record.title')}</CardTitle></CardHeader>
                        <CardContent className="space-y-3">
                            <div className="flex items-center gap-3">
                                {!recording ? (
                                    <Button onClick={startRecording} disabled={!speakerId.trim() || uploading}
                                        variant="primary" leftIcon={<Mic className="w-4 h-4" />}>
                                        {t('voiceTraining.volcengine.record.start')}
                                    </Button>
                                ) : (
                                    <Button onClick={stopRecording} variant="destructive" leftIcon={<Square className="w-4 h-4" />}>
                                        {formatTime(recordingTime)}
                                    </Button>
                                )}
                                {recordedBlob && !recording && (
                                    <Button onClick={submitRecording} loading={uploading}
                                        variant="primary" leftIcon={<Upload className="w-4 h-4" />}>
                                        {t('voiceTraining.volcengine.record.submit')}
                                    </Button>
                                )}
                            </div>
                            <p className="text-xs text-gray-400">{t('voiceTraining.volcengine.record.hint')}</p>

                            <div className="border-t border-gray-100 dark:border-gray-700 pt-3">
                                <Button onClick={handleQueryStatus} loading={querying} variant="outline"
                                    disabled={!speakerId.trim()} fullWidth size="sm">
                                    {t('voiceTraining.volcengine.queryStatus')}
                                </Button>
                            </div>
                            {taskStatus && (
                                <div className="p-3 bg-gray-50 dark:bg-gray-800/50 rounded-lg text-sm flex items-center justify-between">
                                    <span className="font-medium">{getStatusText(taskStatus.status)}</span>
                                    {taskStatus.failedDesc && <span className="text-red-500 text-xs">{taskStatus.failedDesc}</span>}
                                </div>
                            )}
                        </CardContent>
                    </Card>
                </div>
            )}

            {/* ===== Clones Tab ===== */}
            {activeTab === 'clones' && (
                <div className="grid grid-cols-1 lg:grid-cols-5 gap-4">
                    {/* Left: Voice List */}
                    <div className="lg:col-span-2">
                        <Card className="h-full">
                            <CardHeader>
                                <div className="flex items-center justify-between">
                                    <CardTitle className="text-base">{t('voiceTraining.volcengine.myVoices.title')}</CardTitle>
                                    <Button variant="ghost" size="sm" onClick={refreshVoiceClones}
                                        loading={loadingClones} leftIcon={<RefreshCw className="w-3.5 h-3.5" />}>
                                    </Button>
                                </div>
                            </CardHeader>
                            <CardContent>
                                {loadingClones ? (
                                    <p className="text-center py-6 text-gray-400 text-sm">{t('voiceTraining.loadingClones')}</p>
                                ) : voiceClones.length === 0 ? (
                                    <p className="text-center py-6 text-gray-400 text-sm">{t('voiceTraining.noClones')}</p>
                                ) : (
                                    <div className="space-y-2 max-h-[500px] overflow-y-auto">
                                        {voiceClones.map(clone => (
                                            <div key={clone.id}
                                                onClick={() => setSelectedCloneForSynthesis(clone.id)}
                                                className={`p-3 rounded-lg border cursor-pointer transition-all group ${
                                                    selectedCloneForSynthesis === clone.id
                                                        ? 'border-orange-400 bg-orange-50 dark:bg-orange-900/20'
                                                        : 'border-gray-200 dark:border-gray-700 hover:border-gray-300'
                                                }`}>
                                                <div className="flex items-center justify-between">
                                                    <div className="flex items-center gap-2 min-w-0">
                                                        <Zap className="w-4 h-4 text-orange-500 shrink-0" />
                                                        <span className="font-medium text-sm truncate">{clone.voiceName}</span>
                                                    </div>
                                                    <button onClick={(e) => { e.stopPropagation(); deleteVoiceClone(clone.id, clone.voiceName) }}
                                                        className="opacity-0 group-hover:opacity-100 transition-opacity text-gray-400 hover:text-red-500 shrink-0">
                                                        <Trash2 className="w-3.5 h-3.5" />
                                                    </button>
                                                </div>
                                            </div>
                                        ))}
                                    </div>
                                )}
                            </CardContent>
                        </Card>
                    </div>

                    {/* Right: Synthesis */}
                    <div className="lg:col-span-3">
                        <Card className="h-full">
                            <CardHeader><CardTitle className="text-base">{t('voiceTraining.synthesize.title')}</CardTitle></CardHeader>
                            <CardContent className="space-y-3">
                                {selectedClone ? (
                                    <>
                                        <div className="flex items-center gap-2 p-3 bg-orange-50 dark:bg-orange-900/20 rounded-lg">
                                            <Zap className="w-4 h-4 text-orange-500" />
                                            <span className="text-sm font-medium">{selectedClone.voiceName}</span>
                                            <span className="text-xs text-gray-400 ml-auto">{selectedClone.assetId}</span>
                                        </div>
                                        <FormField label={t('voiceTraining.synthesizeText')} required>
                                            <ArcoInput size="large" className="!h-10 !text-base" value={synthesisText}
                                                onChange={(e) => setSynthesisText(e)}
                                                placeholder={t('voiceTraining.synthesizeTextPlaceholder')} />
                                        </FormField>
                                        <Button onClick={() => synthesizeVoice(selectedClone)} loading={synthesizing}
                                            variant="primary" fullWidth disabled={!synthesisText.trim()}>
                                            {synthesizing ? t('voiceTraining.synthesizing') : t('voiceTraining.startSynthesize')}
                                        </Button>
                                    </>
                                ) : (
                                    <div className="text-center py-12 text-gray-400 text-sm">
                                        {t('voiceTraining.volcengine.selectVoiceHint')}
                                    </div>
                                )}
                            </CardContent>
                        </Card>
                    </div>
                </div>
            )}

            {/* ===== History Tab ===== */}
            {activeTab === 'history' && (
                <Card>
                    <CardHeader>
                        <div className="flex items-center justify-between">
                            <CardTitle className="text-base">{t('voiceTraining.synthesisHistory.title')}</CardTitle>
                            <Button variant="ghost" size="sm" onClick={refreshSynthesisHistory}
                                loading={loadingHistory} leftIcon={<RefreshCw className="w-3.5 h-3.5" />}>
                            </Button>
                        </div>
                    </CardHeader>
                    <CardContent>
                        {loadingHistory ? (
                            <p className="text-center py-6 text-gray-400 text-sm">{t('voiceTraining.loadingHistory')}</p>
                        ) : synthesisHistory.length === 0 ? (
                            <p className="text-center py-6 text-gray-400 text-sm">{t('voiceTraining.noHistory')}</p>
                        ) : (
                            <div className="space-y-1.5">
                                {synthesisHistory.map(record => (
                                    <div key={record.id}
                                        className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors">
                                        <div className="flex-1 min-w-0">
                                            <p className="text-sm truncate">{record.text}</p>
                                            <p className="text-xs text-gray-400">{new Date(record.createdAt).toLocaleString()}</p>
                                        </div>
                                        <div className="flex items-center gap-0.5 shrink-0">
                                            {record.audioUrl && (
                                                <>
                                                    <Button variant="ghost" size="sm"
                                                        onClick={() => playingAudio === record.audioUrl ? stopAudio() : playAudio(record.audioUrl)}
                                                        leftIcon={playingAudio === record.audioUrl ? <Pause className="w-3.5 h-3.5" /> : <Play className="w-3.5 h-3.5" />}>
                                                    </Button>
                                                    <Button variant="ghost" size="sm"
                                                        onClick={() => downloadAudio(record.audioUrl, record.text)}
                                                        leftIcon={<Download className="w-3.5 h-3.5" />}>
                                                    </Button>
                                                </>
                                            )}
                                            <Button variant="ghost" size="sm"
                                                onClick={() => deleteSynthesisRecord(record.id)}
                                                leftIcon={<Trash2 className="w-3.5 h-3.5 text-red-400" />}>
                                            </Button>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        )}
                    </CardContent>
                </Card>
            )}
        </div>
    )
}

export default VoiceTrainingVolcengine
