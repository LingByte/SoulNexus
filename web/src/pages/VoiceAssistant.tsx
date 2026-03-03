import { useState, useEffect, useRef } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { showAlert } from '@/utils/notification'
import { Settings } from 'lucide-react'
import { useSearchHighlight } from '@/hooks/useSearchHighlight'
import { useI18nStore } from '@/stores/i18nStore'
// 导入语音助手组件
import VoiceBall from '@/components/Voice/VoiceBall'
import ChatHistory from '@/components/Voice/ChatHistory'
import ChatArea from '@/components/Voice/ChatArea'
import ControlPanel from '@/components/Voice/ControlPanel'
import AddAssistantModal from '@/components/Voice/AddAssistantModal'
import IntegrationModal from '@/components/Voice/IntegrationModal'
import GuideTooltip from '@/components/Voice/GuideTooltip'
import LineSelector from '@/components/Voice/LineSelector'
import TextInputBox from '@/components/Voice/TextInputBox'
import { getKnowledgeBaseByUser } from '@/api/knowledge'
// 导入API服务
import {
    createAssistant,
    getAssistantList,
    getAssistant,
    updateAssistant,
    updateAssistantJS,
    deleteAssistant,
    getVoiceClones,
    getAudioStatus,
    type VoiceClone,
} from '@/api/assistant'
import {
    getChatSessionLogDetail,
    getChatSessionLogsBySession,
    getChatSessionLogsByAssistant,
    type ToolCallInfo
} from '@/api/chat'
import { fetchUserCredentials, type Credential } from '@/api/credential'
// 导入类型定义
import type { Assistant, ChatMessage, VoiceChatSession, LineMode } from './VoiceAssistant/types'
// 导入自定义 Hooks
import { useVoiceAssistant } from '@/hooks/useVoiceAssistant'
// 导入配置
import { getUploadsBaseURL, buildWebSocketURL } from '@/config/apiConfig'

const VoiceAssistant = () => {
    const { t } = useI18nStore()
    const navigate = useNavigate()
    const { id } = useParams();
    const assistantId = id ? parseInt(id, 10) : 0;

    // 引导动画状态管理
    const [showOnboarding, setShowOnboarding] = useState(false)
    const [onboardingStep, setOnboardingStep] = useState(0)
    const [highlightedElement, setHighlightedElement] = useState<string | null>(null)

    // 状态管理
    const [isCalling, setIsCalling] = useState(false)
    const [assistants, setAssistants] = useState<Assistant[]>([])
    const [chatHistory, setChatHistory] = useState<VoiceChatSession[]>([])
    const [chatMessages, setChatMessages] = useState<ChatMessage[]>([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)

    // 获取搜索高亮信息
    const { searchKeyword, highlightFragments, highlightResultId } = useSearchHighlight()
    const [isGlobalMuted, setIsGlobalMuted] = useState(false)
    const [currentSessionId, setCurrentSessionId] = useState<string | null>(null)
    const currentPlayingAudioRef = useRef<HTMLAudioElement | null>(null)
    const currentTTSAudioSourceRef = useRef<AudioBufferSourceNode | null>(null) // 当前播放的TTS音频源
    const lastASRTextRef = useRef<string>('') // 上次已处理的ASR文本（用于去重）
    const lastLLMResponseRef = useRef<string>('') // 上次已处理的LLM回复（用于去重）
    const lastSelectedAgentRef = useRef<number | null>(null) // 防止重复选择同一个助手

    // 引用DOM元素用于引导动画
    const voiceBallRef = useRef<HTMLDivElement>(null)
    const assistantListRef = useRef<HTMLDivElement>(null)
    const chatAreaRef = useRef<HTMLDivElement>(null)
    const controlPanelRef = useRef<HTMLDivElement>(null)
    const textInputRef = useRef<HTMLDivElement>(null)

    // WebSocket 和 WebRTC 相关状态
    const [socket, setSocket] = useState<WebSocket | null>(null)
    const [peerConnection, setPeerConnection] = useState<RTCPeerConnection | null>(null)
    const [localStream, setLocalStream] = useState<MediaStream | null>(null)
    const [callDuration, setCallDuration] = useState(0)
    const [callTimer, setCallTimer] = useState<NodeJS.Timeout | null>(null)
    const [pendingCandidates, setPendingCandidates] = useState<any[]>([])

    // 线路选择：WebRTC / 七牛云ASR+TTS
    const [lineMode, setLineMode] = useState<LineMode>('webrtc')
    const mediaRecorderRef = useRef<MediaRecorder | null>(null)
    // 训练音色
    const [voiceClones, setVoiceClones] = useState<VoiceClone[]>([])
    const [selectedVoiceCloneId, setSelectedVoiceCloneId] = useState<number | null>(null)
    // 文本模式：语音输出 / 纯文本对话
    const [textMode, setTextMode] = useState<'voice' | 'text'>('voice')

    // 左右侧面板折叠状态管理
    const [isAssistantListCollapsed, setIsAssistantListCollapsed] = useState(false)
    const [isControlPanelCollapsed, setIsControlPanelCollapsed] = useState(true) // 默认隐藏配置面板

    // 控制面板状态 - API密钥使用localStorage持久化
    const [apiKey, setApiKey] = useState(() => {
        return localStorage.getItem('voiceAssistant_apiKey') || ''
    })
    const [apiSecret, setApiSecret] = useState(() => {
        return localStorage.getItem('voiceAssistant_apiSecret') || ''
    })

    // API密钥变化时自动保存到localStorage
    useEffect(() => {
        if (apiKey) {
            localStorage.setItem('voiceAssistant_apiKey', apiKey)
        } else {
            localStorage.removeItem('voiceAssistant_apiKey')
        }
    }, [apiKey])

    useEffect(() => {
        if (apiSecret) {
            localStorage.setItem('voiceAssistant_apiSecret', apiSecret)
        } else {
            localStorage.removeItem('voiceAssistant_apiSecret')
        }
    }, [apiSecret])

    // 根据API密钥查找对应的凭证，获取TTS Provider
    useEffect(() => {
        const fetchTTSProvider = async () => {
            if (!apiKey || !apiSecret) {
                setTtsProvider(undefined)
                return
            }

            try {
                const credentials = await fetchUserCredentials()
                if (credentials.code === 200 && credentials.data) {
                    // 查找匹配的凭证
                    const matchedCredential = credentials.data.find(
                        (cred: Credential) => cred.apiKey === apiKey && cred.apiSecret === apiSecret
                    )

                    if (matchedCredential && matchedCredential.ttsConfig?.provider) {
                        setTtsProvider(matchedCredential.ttsConfig.provider.toLowerCase())
                    } else {
                        // 如果没有找到匹配的凭证，默认使用腾讯云（向后兼容）
                        setTtsProvider('tencent')
                    }
                }
            } catch (error) {
                console.error('获取TTS Provider失败:', error)
                // 默认使用腾讯云（向后兼容）
                setTtsProvider('tencent')
            }
        }

        fetchTTSProvider()
    }, [apiKey, apiSecret])
    const [language, setLanguage] = useState('zh-cn')
    const [selectedSpeaker, setSelectedSpeaker] = useState('101016')
    const [systemPrompt, setSystemPrompt] = useState('')
    const [temperature, setTemperature] = useState(0.6)
    const [maxTokens, setMaxTokens] = useState(150)
    const [llmModel, setLlmModel] = useState('')

    // 助手基本信息
    const [assistantName, setAssistantName] = useState('')
    const [assistantDescription, setAssistantDescription] = useState('')
    const [assistantIcon, setAssistantIcon] = useState('Bot')
    const [enableGraphMemory, setEnableGraphMemory] = useState(false)
    // VAD 配置
    const [enableVAD, setEnableVAD] = useState(true)
    const [vadThreshold, setVadThreshold] = useState(500)
    const [vadConsecutiveFrames, setVadConsecutiveFrames] = useState(2)

    // 模态框状态
    const [showAddAssistantModal, setShowAddAssistantModal] = useState(false)
    const [showIntegrationModal, setShowIntegrationModal] = useState(false)
    const [showConfirmModal, setShowConfirmModal] = useState(false)
    const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
    const [selectedMethod, setSelectedMethod] = useState<string | null>(null)
    const [pendingAgent, setPendingAgent] = useState<number | null>(null)

    // 聊天记录详情
    const [selectedLogDetail, setSelectedLogDetail] = useState<any>(null)
    const [showLogModal, setShowLogModal] = useState(false)

    // 获取选中助手的 jsSourceId
    const jsSourceId = assistants.find(a => a.id === assistantId)?.jsSourceId || ''

    // 当选中助手变化时，更新JS模板选择和基础配置
    useEffect(() => {
        if (assistantId && assistants.length > 0) {
            const currentAssistant = assistants.find(a => a.id === assistantId)
            if (currentAssistant) {
                setSelectedJSTemplate(currentAssistant.jsSourceId || null)
                // 同步助手基础配置（包括图记忆开关和VAD配置）
                setAssistantName(currentAssistant.name || '')
                setAssistantDescription(currentAssistant.description || '')
                setAssistantIcon(currentAssistant.icon || 'Bot')
                setEnableGraphMemory(!!(currentAssistant as any).enableGraphMemory)
                // 同步 VAD 配置
                if ((currentAssistant as any).enableVAD !== undefined) {
                    setEnableVAD((currentAssistant as any).enableVAD)
                }
                if ((currentAssistant as any).vadThreshold !== undefined) {
                    setVadThreshold((currentAssistant as any).vadThreshold)
                }
                if ((currentAssistant as any).vadConsecutiveFrames !== undefined) {
                    setVadConsecutiveFrames((currentAssistant as any).vadConsecutiveFrames)
                }
            }
        }
    }, [assistantId, assistants])

    // 状态管理中新增
    const [selectedKnowledgeBase, setSelectedKnowledgeBase] = useState<string | null>(null)
    const [knowledgeBases, setKnowledgeBases] = useState<Array<{id: string, name: string}>>([]);
    const [selectedJSTemplate, setSelectedJSTemplate] = useState<string | null>(null)
    const [ttsProvider, setTtsProvider] = useState<string | undefined>(undefined)

    // 引导步骤配置（支持国际化）
    const onboardingSteps = [
        {
            element: 'voice-ball',
            text: t('voiceAssistant.onboarding.voiceBall'),
            position: 'right' as const
        },
        {
            element: 'assistant-list',
            text: t('voiceAssistant.onboarding.assistantList'),
            position: 'right' as const
        },
        {
            element: 'chat-area',
            text: t('voiceAssistant.onboarding.chatArea'),
            position: 'top' as const
        },
        {
            element: 'control-panel',
            text: t('voiceAssistant.onboarding.controlPanel'),
            position: 'left' as const
        },
        {
            element: 'text-input',
            text: t('voiceAssistant.onboarding.textInput'),
            position: 'top' as const,
            isLast: true
        }
    ]

    // 添加用户消息到聊天记录（去重）
    const addUserMessage = (text: string): string => {
        // 去重：如果与上次ASR文本相同，不重复添加
        if (text === lastASRTextRef.current && chatMessages.length > 0) {
            // 检查最后一条消息是否已经是这个文本
            const lastMessage = chatMessages[chatMessages.length - 1]
            if (lastMessage.type === 'user' && lastMessage.content === text) {
                console.log('[WebSocket语音] 重复的用户消息，跳过:', text)
                // 返回已存在的消息ID
                return lastMessage.id || `user-${Date.now()}`
            }
        }
        console.log('添加用户消息:', text)
        const messageId = `user-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
        const newMessage = {
            type: 'user' as const,
            content: text,
            timestamp: new Date().toLocaleTimeString(),
            id: messageId
        }
        console.log('新用户消息对象:', newMessage)
        setChatMessages(prev => {
            const updated = [...prev, newMessage]
            console.log('更新后的消息列表(用户):', updated)
            return updated
        })
        return messageId
    }

    // 添加AI消息到聊天记录（去重）
    const addAIMessage = (text: string, audioUrl?: string): string => {
        // 去重：如果与上次LLM回复相同，不重复添加
        if (text === lastLLMResponseRef.current && chatMessages.length > 0) {
            // 检查最后一条消息是否已经是这个文本
            const lastMessage = chatMessages[chatMessages.length - 1]
            if (lastMessage.type === 'agent' && lastMessage.content === text) {
                console.log('[WebSocket语音] 重复的AI消息，跳过:', text)
                // 返回已存在的消息ID
                return lastMessage.id || `ai-${Date.now()}`
            }
        }

        console.log('添加AI消息:', text, '音频URL:', audioUrl)

        // 如果有新消息且不是静音状态，停止当前播放的音频
        if (audioUrl && !isGlobalMuted) {
            console.log('停止当前播放的音频，准备播放新音频')
            stopCurrentAudio()
        }

        const messageId = `ai-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
        const newMessage = {
            type: 'agent' as const,
            content: text,
            timestamp: new Date().toLocaleTimeString(),
            id: messageId,
            audioUrl: audioUrl
        }
        console.log('新消息对象:', newMessage)
        setChatMessages(prev => {
            const updated = [...prev, newMessage]
            console.log('更新后的消息列表:', updated)
            return updated
        })

        return messageId
    }

    // 更新AI消息内容（用于流式输出）
    const updateAIMessage = (messageId: string, newText: string): void => {
        setChatMessages(prev => {
            return prev.map(msg => {
                if (msg.id === messageId && msg.type === 'agent') {
                    return { ...msg, content: newText }
                }
                return msg
            })
        })
    }

    // 添加AI loading消息
    const addAILoadingMessage = () => {
        const messageId = `loading-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
        const loadingMessage = {
            type: 'agent' as const,
            content: '',
            timestamp: new Date().toLocaleTimeString(),
            id: messageId,
            isLoading: true
        }
        setChatMessages(prev => [...prev, loadingMessage])
        return messageId
    }

    // 移除loading消息
    const removeLoadingMessage = (messageId: string) => {
        setChatMessages(prev => prev.filter(msg => msg.id !== messageId))
    }

    // 轮询获取音频URL
    const pollAudioStatus = async (requestId: string, messageId: string) => {
        console.log('开始轮询音频状态:', requestId, messageId)
        const pollInterval = 1000 // 1秒轮询一次
        const maxAttempts = 30 // 最多轮询30次（30秒）
        let attempts = 0

        const poll = async () => {
            console.log(`轮询第 ${attempts + 1} 次，requestId: ${requestId}`)
            try {
                const response = await getAudioStatus(requestId)
                console.log('轮询结果:', response)

                if (response.data?.status === 'completed' && response.data?.audioUrl) {
                    // 将 /media/ 路径替换为完整的 URL 路径
                    const uploadsBaseURL = getUploadsBaseURL()
                    const audioUrl = response.data.audioUrl.replace('/media/', `${uploadsBaseURL}/`);

                    // 更新消息的音频URL（不触发打字特效）
                    setChatMessages(prev => prev.map(msg =>
                        msg.id === messageId
                            ? { ...msg, audioUrl: audioUrl }
                            : msg
                    ))

                    // 如果不是静音状态，播放音频
                    if (!isGlobalMuted) {
                        console.log('准备播放轮询获取的音频:', audioUrl)
                        setTimeout(() => {
                            const audio = new Audio(audioUrl) // 使用处理后的完整URL
                            currentPlayingAudioRef.current = audio

                            audio.onended = () => {
                                currentPlayingAudioRef.current = null
                            }

                            audio.onerror = (error) => {
                                console.error('音频播放失败:', error)
                                console.error('音频URL:', audioUrl)
                                currentPlayingAudioRef.current = null
                            }

                            audio.play().catch(err => {
                                console.error('音频播放失败:', err)
                                console.error('音频URL:', audioUrl)
                                currentPlayingAudioRef.current = null
                            })
                        }, 500)
                    } else {
                        console.log('全局静音状态，跳过音频播放')
                    }
                    return
                }

                attempts++
                if (attempts < maxAttempts) {
                    setTimeout(poll, pollInterval)
                } else {
                    console.log('轮询超时，停止轮询')
                }
            } catch (error) {
                console.error('轮询音频状态失败:', error)
                attempts++
                if (attempts < maxAttempts) {
                    setTimeout(poll, pollInterval)
                }
            }
        }

        // 开始轮询
        setTimeout(poll, pollInterval)
    }

    // 使用文本输入相关 hook（在函数定义之后）
    const voiceAssistantHook = useVoiceAssistant({
        apiKey,
        apiSecret,
        assistantId,
        language,
        systemPrompt,
        selectedVoiceCloneId,
        selectedSpeaker,
        currentSessionId,
        setCurrentSessionId,
        addUserMessage,
        addAIMessage,
        addAILoadingMessage,
        removeLoadingMessage,
        pollAudioStatus,
        textMode,
        temperature,
        maxTokens,
        updateAIMessage,
        selectedKnowledgeBase,
    })

    // 处理引导下一步
    const handleNextStep = () => {
        if (onboardingStep < onboardingSteps.length - 1) {
            setOnboardingStep(onboardingStep + 1)
            setHighlightedElement(onboardingSteps[onboardingStep + 1].element)

            // 滚动到下一个元素
            const nextElement = getElementByStep(onboardingStep + 1)
            if (nextElement) {
                nextElement.scrollIntoView({ behavior: 'smooth', block: 'center' })
            }
        } else {
            setShowOnboarding(false)
            setHighlightedElement(null)
            showAlert('引导动画结束， 请用户开始进行语音聊天吧', 'success')
        }
    }

    // 处理跳过引导
    const handleSkipOnboarding = () => {
        setShowOnboarding(false)
        setHighlightedElement(null)
    }

    // 根据步骤获取元素
    const getElementByStep = (step: number) => {
        const elementId = onboardingSteps[step].element
        switch (elementId) {
            case 'voice-ball':
                return voiceBallRef.current
            case 'assistant-list':
                return assistantListRef.current
            case 'chat-area':
                return chatAreaRef.current
            case 'control-panel':
                return controlPanelRef.current
            case 'text-input':
                return textInputRef.current
            default:
                return null
        }
    }

    // 开始引导
    const startOnboarding = () => {
        setShowOnboarding(true)
        setOnboardingStep(0)
        setHighlightedElement(onboardingSteps[0].element)

        // 滚动到第一个元素
        const firstElement = getElementByStep(0)
        if (firstElement) {
            firstElement.scrollIntoView({ behavior: 'smooth', block: 'center' })
        }
    }


    const fetchKnowledgeBases = async () => {
        try {
            const response = await getKnowledgeBaseByUser();
            // 修改数据转换逻辑，适应新的返回格式
            const transformedData = response.data.map((item: { name: string; key: string }) => ({
                id: item.key,
                name: item.name
            }));
            setKnowledgeBases(transformedData);
        } catch (error) {
            console.error('获取知识库列表失败:', error);
        }
    };

    // 添加管理知识库的函数（导航到知识库管理页面）
    const handleManageKnowledgeBases = () => {
        navigate('/knowledge'); // 假设知识库管理页面的路径是 /knowledge
    };

    // 停止当前播放的音频
    const stopCurrentAudio = () => {
        if (currentPlayingAudioRef.current) {
            currentPlayingAudioRef.current.pause()
            currentPlayingAudioRef.current.currentTime = 0
            currentPlayingAudioRef.current = null
        }
        // 也停止页面上所有其他音频元素
        const audioElements = document.querySelectorAll('audio')
        audioElements.forEach(audio => {
            if (audio !== currentPlayingAudioRef.current) {
                audio.pause()
                audio.currentTime = 0
            }
        })
    }

    // 停止当前TTS播放
    const stopCurrentTTSPlayback = () => {
        // 停止AudioBufferSourceNode
        if (currentTTSAudioSourceRef.current) {
            try {
                currentTTSAudioSourceRef.current.stop()
                currentTTSAudioSourceRef.current.disconnect()
                console.log('[WebSocket语音] 已停止当前TTS播放')
            } catch (error) {
                // 可能已经停止了，忽略错误
                console.log('[WebSocket语音] 停止TTS播放:', error)
            }
            currentTTSAudioSourceRef.current = null
        }

        // 停止Audio元素
        if (currentPlayingAudioRef.current) {
            currentPlayingAudioRef.current.pause()
            currentPlayingAudioRef.current.currentTime = 0
            currentPlayingAudioRef.current = null
        }
    }

    // 播放TTS音频流（音频流方式）
    const playTTSAudioStream = (
        audioBuffer: Uint8Array[],
        format: { sampleRate?: number; channels?: number; bitDepth?: number },
        audioContext: AudioContext | null
    ) => {
        if (isGlobalMuted || audioBuffer.length === 0 || !audioContext) {
            console.warn('[WebSocket语音] 跳过播放:', { isGlobalMuted, bufferLength: audioBuffer.length, hasContext: !!audioContext })
            return
        }

        try {
            // 停止之前的播放（确保同一时间只有一个音频在播放）
            stopCurrentTTSPlayback()
            // 确保AudioContext处于running状态
            if (audioContext.state === 'suspended') {
                audioContext.resume().then(() => {
                    console.log('[WebSocket语音] AudioContext已恢复')
                }).catch(err => {
                    console.error('[WebSocket语音] 恢复AudioContext失败:', err)
                })
            }

            // 合并所有音频数据
            const totalLength = audioBuffer.reduce((sum, buf) => sum + buf.length, 0)
            if (totalLength === 0) {
                console.warn('[WebSocket语音] 音频数据为空')
                return
            }

            const mergedBuffer = new Uint8Array(totalLength)
            let offset = 0
            for (const buf of audioBuffer) {
                mergedBuffer.set(buf, offset)
                offset += buf.length
            }

            console.log('[WebSocket语音] 合并音频数据，总长度:', totalLength, '格式:', format)

            // 获取音频格式参数
            const sampleRate = format.sampleRate || 8000 // 后端TTS默认8000
            const channels = format.channels || 1
            const bitDepth = format.bitDepth || 16

            // 根据位深度计算样本数
            const bytesPerSample = bitDepth / 8
            const samples = Math.floor(mergedBuffer.length / bytesPerSample)

            if (samples === 0) {
                console.warn('[WebSocket语音] 样本数为0')
                return
            }

            // 转换为Int16Array（16位PCM）
            const pcmData = new Int16Array(samples)

            // 转换为小端序Int16（假设数据已经是小端序）
            for (let i = 0; i < samples; i++) {
                const byteOffset = i * bytesPerSample
                if (bytesPerSample === 2) {
                    // 16位：2字节
                    pcmData[i] = mergedBuffer[byteOffset] | (mergedBuffer[byteOffset + 1] << 8)
                    // 检查是否有符号位
                    if (pcmData[i] & 0x8000) {
                        pcmData[i] |= 0xFFFF0000 // 符号扩展
                    }
                } else if (bytesPerSample === 1) {
                    // 8位：1字节（转换为16位）
                    pcmData[i] = (mergedBuffer[byteOffset] - 128) * 256
                }
            }

            // 转换为Float32Array（Web Audio API使用，范围-1.0到1.0）
            const float32Data = new Float32Array(pcmData.length)
            for (let i = 0; i < pcmData.length; i++) {
                float32Data[i] = Math.max(-1, Math.min(1, pcmData[i] / 32768.0))
            }

            // 计算每个声道的样本数
            const samplesPerChannel = Math.floor(float32Data.length / channels)
            if (samplesPerChannel === 0) {
                console.warn('[WebSocket语音] 每声道样本数为0')
                return
            }

            // 创建AudioBuffer
            const audioBufferSource = audioContext.createBuffer(channels, samplesPerChannel, sampleRate)

            // 填充音频数据
            if (channels === 1) {
                // 单声道：直接填充
                audioBufferSource.getChannelData(0).set(float32Data)
            } else {
                // 多声道：交错数据需要解交错
                for (let ch = 0; ch < channels; ch++) {
                    const channelData = audioBufferSource.getChannelData(ch)
                    for (let i = 0; i < samplesPerChannel; i++) {
                        channelData[i] = float32Data[i * channels + ch]
                    }
                }
            }

            // 创建AudioBufferSourceNode并播放
            const source = audioContext.createBufferSource()
            source.buffer = audioBufferSource
            source.connect(audioContext.destination)

            // 保存当前播放的音频源（用于打断）
            currentTTSAudioSourceRef.current = source

            // 添加结束回调
            source.onended = () => {
                console.log('[WebSocket语音] TTS音频播放完成')
                if (currentTTSAudioSourceRef.current === source) {
                    currentTTSAudioSourceRef.current = null
                }
            }

            source.start(0)
            console.log('[WebSocket语音] 开始播放TTS音频流，时长:', (samplesPerChannel / sampleRate).toFixed(2), '秒')
        } catch (error) {
            console.error('[WebSocket语音] 播放TTS音频流失败:', error)
            console.error('[WebSocket语音] 错误详情:', {
                bufferCount: audioBuffer.length,
                format,
                contextState: audioContext?.state
            })
        }
    }

    // 连接WebSocket语音服务（通用模式，支持多服务商）
    const connectQiniuVoice = () => {
        // 先关闭现有连接
        if (socket && socket.readyState !== WebSocket.CLOSED) {
            socket.close()
        }

        // 校验API密钥（已经在startCall中校验，这里再次确认）
        if (!apiKey || !apiSecret) {
            showAlert('请先在配置面板中设置API Key和API Secret', 'warning')
            setIsCalling(false)
            return
        }

        // 连接通用WebSocket语音服务
        const wsUrl = `${buildWebSocketURL('/api/voice/websocket')}?apiKey=${encodeURIComponent(apiKey)}&apiSecret=${encodeURIComponent(apiSecret)}&assistantId=${assistantId}&language=${language}&speaker=${selectedSpeaker}`
        const newSocket = new WebSocket(wsUrl)

        newSocket.onopen = async () => {
            console.log('[WebSocket语音] WebSocket已连接')

            try {
                // 获取麦克风权限
                const stream = await navigator.mediaDevices.getUserMedia({
                    audio: {
                        echoCancellation: true,
                        noiseSuppression: true,
                        autoGainControl: true,
                        sampleRate: 16000 // 16kHz采样率
                    }
                })

                setLocalStream(stream)

                // 等待服务端确认连接后再开始录音
                console.log('[WebSocket语音] 等待服务端确认连接...')
                
                // 保存 stream，等待 connected 消息后再启动录音
                ;(newSocket as any)._pendingStream = stream
            } catch (error) {
                console.error('[WebSocket语音] 获取麦克风失败:', error)
                showAlert('获取麦克风权限失败', 'error')
                setIsCalling(false)
            }
        }

        // TTS音频流处理
        let ttsAudioBuffer: Uint8Array[] = []
        let ttsAudioFormat: { sampleRate?: number; channels?: number; bitDepth?: number } = {}
        let isRecordingStarted = false // 标记是否已开始录音
        let isTTSActive = false // 标记是否在TTS播放状态
        let audioContext: AudioContext | null = null

        newSocket.onmessage = async (event) => {
            // 处理二进制消息（TTS音频流）- 支持ArrayBuffer和Blob
            // WebSocket的二进制消息应该总是音频数据，不应该尝试解析为JSON
            if (event.data instanceof ArrayBuffer) {
                console.log('[WebSocket语音] 收到ArrayBuffer音频数据，大小:', event.data.byteLength, 'isTTSActive:', isTTSActive, 'sampleRate:', ttsAudioFormat.sampleRate)
                // 只有在TTS激活状态时才累积音频数据
                if (!isGlobalMuted && isTTSActive && ttsAudioFormat.sampleRate) {
                    // 累积音频数据
                    ttsAudioBuffer.push(new Uint8Array(event.data))
                    console.log('[WebSocket语音] 累积音频数据，当前缓冲区大小:', ttsAudioBuffer.length)
                }
                return
            }

            // 处理Blob对象（某些浏览器可能将二进制数据包装为Blob）
            if (event.data instanceof Blob) {
                console.log('[WebSocket语音] 收到Blob音频数据，大小:', event.data.size, 'isTTSActive:', isTTSActive, 'sampleRate:', ttsAudioFormat.sampleRate)
                // WebSocket的Blob消息在TTS激活状态下应该是音频数据
                if (!isGlobalMuted && isTTSActive && ttsAudioFormat.sampleRate) {
                    const arrayBuffer = await event.data.arrayBuffer()
                    ttsAudioBuffer.push(new Uint8Array(arrayBuffer))
                    console.log('[WebSocket语音] 累积音频数据，当前缓冲区大小:', ttsAudioBuffer.length)
                } else {
                    // 如果不是TTS激活状态，尝试作为文本消息处理（但这种情况应该很少）
                    // 先检查是否是有效的文本（尝试读取前几个字节）
                    try {
                        const text = await event.data.text()
                        // 检查是否是有效的JSON（以{或[开头）
                        const trimmed = text.trim()
                        if (trimmed.startsWith('{') || trimmed.startsWith('[')) {
                            const data = JSON.parse(text)
                            handleWebSocketMessage(data)
                        } else {
                            // 不是JSON，可能是其他文本，忽略或记录
                            console.warn('[WebSocket语音] 收到非JSON文本Blob:', text.substring(0, 100))
                        }
                    } catch (error) {
                        // 解析失败，可能是二进制数据，忽略
                        console.debug('[WebSocket语音] Blob不是文本消息，可能是音频数据')
                    }
                }
                return
            }

            // 处理文本消息
            try {
                // 确保是字符串
                const textData = typeof event.data === 'string' ? event.data : String(event.data)
                const data = JSON.parse(textData)
                handleWebSocketMessage(data)
            } catch (error) {
                // 如果不是JSON，可能是其他文本消息
                console.error('[WebSocket语音] 消息解析失败:', error, '原始数据:', event.data)
            }
        }

        // 处理WebSocket文本消息的内部函数
        const handleWebSocketMessage = (data: any) => {
            console.log('[WebSocket语音] 收到消息:', data.type, data)

            switch (data.type) {
                case 'connected':
                    // 服务端确认连接成功，现在可以开始录音了
                    console.log('[WebSocket语音]', data.message)
                    showAlert('WebSocket语音连接已建立', 'success')
                    
                    // 启动录音（如果还没开始）
                    if (!isRecordingStarted && (newSocket as any)._pendingStream) {
                        const stream = (newSocket as any)._pendingStream
                        console.log('[WebSocket语音] 服务端就绪，开始录音')
                        startQiniuRecording(stream, newSocket)
                        isRecordingStarted = true
                        delete (newSocket as any)._pendingStream
                    }
                    break
                case 'asr_result':
                    // 显示识别结果（去重：只显示新的文本）
                    if (data.text && data.text !== lastASRTextRef.current) {
                        lastASRTextRef.current = data.text
                        addUserMessage(data.text)
                        // 打断当前AI播放（用户说话时打断AI）
                        stopCurrentTTSPlayback()
                    }
                    break
                case 'llm_response':
                    // 显示LLM回复（去重：只显示新的回复）
                    if (data.text && data.text !== lastLLMResponseRef.current) {
                        lastLLMResponseRef.current = data.text
                        addAIMessage(data.text)
                        // 打断之前的AI播放（新的回复到来时，停止之前的播放）
                        stopCurrentTTSPlayback()
                    }
                    break
                case 'tts_start':
                    // TTS开始，记录音频格式
                    console.log('[WebSocket语音] TTS开始，音频格式:', data)

                    // 停止之前的播放（打断之前的TTS）
                    stopCurrentTTSPlayback()

                    // 暂停发送音频数据（防止回声/反馈触发VAD barge-in）
                    if ((newSocket as any)._audioProcessor) {
                        console.log('[WebSocket语音] TTS开始，暂停发送音频数据以防止回声')
                        ;(newSocket as any)._audioProcessor.pause()
                    }

                    // 标记TTS为激活状态
                    isTTSActive = true
                    ttsAudioFormat = {
                        sampleRate: data.sampleRate || 8000, // 默认8000，因为后端TTS是8000
                        channels: data.channels || 1,
                        bitDepth: data.bitDepth || 16
                    }
                    ttsAudioBuffer = []

                    // 创建AudioContext用于播放
                    if (!audioContext) {
                        audioContext = new (window.AudioContext || (window as any).webkitAudioContext)({
                            sampleRate: ttsAudioFormat.sampleRate
                        })
                    }
                    // 确保AudioContext处于running状态
                    if (audioContext.state === 'suspended') {
                        audioContext.resume().catch(err => {
                            console.error('[WebSocket语音] 恢复AudioContext失败:', err)
                        })
                    }
                    break
                case 'tts_end':
                    // TTS结束，播放累积的音频
                    console.log('[WebSocket语音] TTS结束，音频缓冲区大小:', ttsAudioBuffer.length)
                    // 标记TTS为非激活状态（在播放前就标记，防止新的音频数据进入）
                    isTTSActive = false
                    if (!isGlobalMuted && ttsAudioBuffer.length > 0 && ttsAudioFormat.sampleRate) {
                        // 复制缓冲区（因为playTTSAudioStream会清空它）
                        const audioDataToPlay = [...ttsAudioBuffer]
                        ttsAudioBuffer = [] // 立即清空，防止重复播放
                        playTTSAudioStream(audioDataToPlay, ttsAudioFormat, audioContext)
                    } else {
                        // 如果没有音频数据，清空缓冲区
                        ttsAudioBuffer = []
                    }
                    // 重置音频格式（防止后续二进制消息被误认为是音频）
                    ttsAudioFormat = {}
                    
                    // 恢复发送音频数据（TTS播放完成后，延迟500ms以确保TTS音频播放完成）
                    setTimeout(() => {
                        if ((newSocket as any)._audioProcessor) {
                            console.log('[WebSocket语音] TTS结束，恢复发送音频数据')
                            ;(newSocket as any)._audioProcessor.resume()
                        }
                    }, 500)
                    break
                case 'tts_audio':
                    // 兼容旧格式（音频URL）
                    if (data.audioUrl && !isGlobalMuted) {
                        playTTSAudio(data.audioUrl)
                    }
                    break
                case 'error':
                    console.error('[WebSocket语音] 错误:', data.message)
                    showAlert(data.message, 'error')
                    setIsCalling(false)
                    break
                case 'session_cleared':
                    console.log('[WebSocket语音]', data.message)
                    break
                case 'pong':
                    // 心跳响应
                    break
            }
        }

        newSocket.onerror = (error) => {
            console.error('[WebSocket语音] WebSocket错误:', error)
            showAlert('WebSocket语音连接出错', 'error')
            setIsCalling(false)
        }

        newSocket.onclose = () => {
            console.log('[WebSocket语音] WebSocket连接关闭')
            setIsCalling(false)
        }

        setSocket(newSocket)
    }

    // 开始录音 - 使用Web Audio API获取PCM数据
    const startQiniuRecording = (stream: MediaStream, ws: WebSocket) => {
        console.log('[WebSocket语音] 开始录音，使用Web Audio API获取PCM数据')

        try {
            // 创建AudioContext
            const audioContext = new (window.AudioContext || (window as any).webkitAudioContext)({
                sampleRate: 16000 // 设置采样率为16kHz
            })

            // 创建音频源
            const source = audioContext.createMediaStreamSource(stream)

            // 创建ScriptProcessorNode来处理音频数据
            const processor = audioContext.createScriptProcessor(4096, 1, 1)

            // 标记是否应该发送音频（TTS播放时不发送）
            let shouldSendAudio = true
            
            // 保存到ref以便外部控制
            ;(ws as any)._audioProcessor = {
                pause: () => { shouldSendAudio = false },
                resume: () => { shouldSendAudio = true }
            }

            processor.onaudioprocess = (event) => {
                if (ws.readyState === WebSocket.OPEN && shouldSendAudio) {
                    // 获取PCM数据
                    const inputBuffer = event.inputBuffer
                    const pcmData = inputBuffer.getChannelData(0)

                    // 将Float32Array转换为Int16Array (PCM 16-bit)
                    const pcm16Data = new Int16Array(pcmData.length)
                    for (let i = 0; i < pcmData.length; i++) {
                        // 将float32 (-1.0 到 1.0) 转换为int16 (-32768 到 32767)
                        pcm16Data[i] = Math.max(-32768, Math.min(32767, pcmData[i] * 32768))
                    }

                    // 发送PCM数据（二进制消息）
                    ws.send(pcm16Data.buffer)
                }
            }

            // 连接音频处理链
            source.connect(processor)
            processor.connect(audioContext.destination)

            // 保存引用以便停止
            mediaRecorderRef.current = {
                stop: () => {
                    processor.disconnect()
                    source.disconnect()
                    audioContext.close()
                }
            } as any

        } catch (error) {
            console.error('创建Web Audio API失败:', error)
            // 如果Web Audio API失败，回退到MediaRecorder
            startQiniuRecordingFallback(stream, ws)
        }
    }

    // 回退方案：使用MediaRecorder
    const startQiniuRecordingFallback = (stream: MediaStream, ws: WebSocket) => {
        console.log('[WebSocket语音] 使用MediaRecorder回退方案')

        const mediaRecorder = new MediaRecorder(stream, {
            mimeType: 'audio/webm;codecs=opus',
            audioBitsPerSecond: 128000
        })

        mediaRecorder.ondataavailable = (event) => {
            if (event.data.size > 0 && ws.readyState === WebSocket.OPEN) {
                const reader = new FileReader()
                reader.onload = () => {
                    const arrayBuffer = reader.result as ArrayBuffer
                    ws.send(arrayBuffer)
                }
                reader.readAsArrayBuffer(event.data)
            }
        }

        mediaRecorder.start(200)
        mediaRecorderRef.current = mediaRecorder
    }

    // 播放TTS音频（URL方式，兼容旧格式）
    const playTTSAudio = (audioUrl: string) => {
        console.log('准备播放TTS音频:', audioUrl)

        // 检查音频URL是否有效
        if (!audioUrl || audioUrl.trim() === '') {
            console.error('音频URL为空')
            return
        }

        const audio = new Audio(audioUrl)
        currentPlayingAudioRef.current = audio

        audio.onended = () => {
            console.log('音频播放完成')
            currentPlayingAudioRef.current = null
        }

        audio.onerror = (error) => {
            console.error('TTS音频播放失败:', error)
            console.error('音频URL:', audioUrl)
            currentPlayingAudioRef.current = null

            // 尝试检查音频文件是否可访问
            fetch(audioUrl, { method: 'HEAD' })
                .then(response => {
                    console.log('音频文件HTTP状态:', response.status)
                    console.log('音频文件Content-Type:', response.headers.get('content-type'))
                })
                .catch(fetchError => {
                    console.error('音频文件访问失败:', fetchError)
                })
        }

        audio.play().catch(err => {
            console.error('TTS音频播放失败:', err)
            console.error('音频URL:', audioUrl)
            currentPlayingAudioRef.current = null
        })
    }

    // 连接WebSocket
    const connectWebSocket = () => {
        // 先关闭现有连接
        if (socket && socket.readyState !== WebSocket.CLOSED) {
            socket.close()
        }

        // 将认证信息作为查询参数添加到URL中（使用表单配置的API密钥）
        const currentApiKey = apiKey
        const currentApiSecret = apiSecret
        // 添加 assistantId 参数（必需）
        const wsUrl = `${buildWebSocketURL('/api/chat/call')}?apiKey=${currentApiKey}&apiSecret=${currentApiSecret}&assistantId=${assistantId}`
        const newSocket = new WebSocket(wsUrl)

        // 用于存储会话信息和 ICE candidates
        let sessionId = ''
        let localPeerConnection: RTCPeerConnection | null = null
        const collectedCandidates: string[] = []

        // 初始化 WebRTC 连接的函数
        const initWebRTC = async () => {
            try {
                // 1. 创建 RTCPeerConnection
                const newPeerConnection = new RTCPeerConnection({
                    iceServers: [
                        { urls: 'stun:stun.l.google.com:19302' }
                    ]
                })
                localPeerConnection = newPeerConnection

                // 2. 获取麦克风音频
                const stream = await navigator.mediaDevices.getUserMedia({
                    audio: {
                        echoCancellation: true,
                        noiseSuppression: true,
                        sampleRate: 16000, // 请求 16kHz 采样率以匹配后端 ASR
                    }
                })
                setLocalStream(stream) // 保存到状态中，以便 stopCall 使用

                stream.getTracks().forEach(track => {
                    newPeerConnection.addTrack(track, stream)
                })

                // 3. 收集 ICE 候选（使用 ICE bundling 模式，收集完毕后一起发送）
                newPeerConnection.onicecandidate = (event) => {
                    if (event.candidate) {
                        // 收集 candidate 字符串
                        collectedCandidates.push(event.candidate.candidate)
                        console.log('[WebRTC] 收集 ICE 候选:', event.candidate.candidate)
                    } else {
                        // ICE 收集完成（event.candidate 为 null）
                        console.log('[WebRTC] ICE 候选收集完成，共收集:', collectedCandidates.length)
                        // 发送 offer（包含所有 candidates）
                        sendOfferWithCandidates(newPeerConnection)
                    }
                }

                // 处理远端音频轨道
                newPeerConnection.ontrack = (event) => {
                    console.log('[WebRTC] 收到远端音轨')
                    const remoteAudio = new Audio()
                    remoteAudio.srcObject = event.streams[0]
                    remoteAudio.play().catch(err => {
                        console.error('[WebRTC] 播放远端音频失败:', err)
                    })
                }

                // 连接状态变化处理
                newPeerConnection.onconnectionstatechange = () => {
                    console.log('[WebRTC] 连接状态:', newPeerConnection.connectionState)
                    switch (newPeerConnection.connectionState) {
                        case 'connected':
                            console.log('[WebRTC] 已连接')
                            // 发送 connected 确认消息
                            if (newSocket.readyState === WebSocket.OPEN) {
                                newSocket.send(JSON.stringify({
                                    type: 'connected',
                                    session_id: sessionId,
                                    data: {}
                                }))
                                console.log('[WebRTC] 已发送 connected 确认消息')
                            }
                            break
                        case 'disconnected':
                        case 'failed':
                        case 'closed':
                            console.log('[WebRTC] 连接关闭/失败')
                            break
                    }
                }

                // 4. 创建 offer
                const offer = await newPeerConnection.createOffer()
                await newPeerConnection.setLocalDescription(offer)
                console.log('[WebRTC] 已创建并设置本地 SDP offer')

                setPeerConnection(newPeerConnection)
                setLocalStream(stream)

                // 注意：offer 会在 ICE 收集完成后通过 sendOfferWithCandidates 发送

            } catch (error) {
                console.error('[WebRTC] 初始化失败:', error)
                showAlert('音频设备初始化失败', 'error')
            }
        }

        // 发送包含 candidates 的 offer
        const sendOfferWithCandidates = (pc: RTCPeerConnection) => {
            if (!pc.localDescription || newSocket.readyState !== WebSocket.OPEN) {
                console.error('[WebRTC] 无法发送 offer: localDescription 或 WebSocket 未就绪')
                return
            }

            // 按照后端期望的格式发送 offer
            const offerMsg = {
                type: 'offer',
                session_id: sessionId,
                data: {
                    sdp: pc.localDescription.sdp,
                    candidates: collectedCandidates,
                }
            }

            console.log('[WebRTC] 发送 offer，包含', collectedCandidates.length, '个 ICE 候选')
            newSocket.send(JSON.stringify(offerMsg))
        }

        // 处理 answer 消息
        const handleAnswer = async (data: any) => {
            if (!localPeerConnection) {
                console.error('[WebRTC] peerConnection 不存在')
                return
            }

            // 从 data.data 中获取 sdp 和 candidates（后端返回格式）
            const answerData = data.data || data
            let sdp = answerData.sdp || data.sdp
            const candidates = answerData.candidates || []

            console.log('[WebRTC] 收到 answer:')
            console.log('- data:', data)
            console.log('- answerData:', answerData)
            console.log('- SDP 类型:', typeof sdp)
            console.log('- SDP 存在:', !!sdp)
            console.log('- 服务器 ICE 候选数量:', candidates.length)

            if (!sdp) {
                console.error('[WebRTC] answer 中没有 SDP')
                return
            }

            // 如果 sdp 是对象（可能是 JSON 字符串被解析了），尝试提取 sdp 字段
            if (typeof sdp === 'object' && sdp !== null) {
                console.log('[WebRTC] SDP 是对象，尝试提取 sdp 字段')
                if (sdp.sdp) {
                    sdp = sdp.sdp
                } else {
                    console.error('[WebRTC] SDP 对象中没有 sdp 字段')
                    return
                }
            }

            // 如果 sdp 是字符串，但可能是 JSON 字符串（后端返回的是 SessionDescription 的 JSON）
            if (typeof sdp === 'string') {
                // 检查是否是 JSON 格式的字符串（以 { 开头）
                const trimmed = sdp.trim()
                if (trimmed.startsWith('{')) {
                    try {
                        const parsed = JSON.parse(sdp)
                        // 后端返回的格式是 {"type":"answer","sdp":"v=0\r\n..."}
                        if (parsed.sdp && typeof parsed.sdp === 'string') {
                            sdp = parsed.sdp
                            console.log('[WebRTC] 从 SessionDescription JSON 中提取 SDP')
                        } else if (parsed.type === 'answer' && parsed.sdp) {
                            sdp = parsed.sdp
                            console.log('[WebRTC] 从嵌套的 SessionDescription JSON 中提取 SDP')
                        } else {
                            console.warn('[WebRTC] JSON 解析成功但未找到 sdp 字段:', parsed)
                        }
                    } catch (e) {
                        console.warn('[WebRTC] SDP 字符串解析 JSON 失败，当作纯 SDP 处理:', e)
                        // 如果解析失败，继续使用原始字符串
                    }
                }
            }

            // 调试：打印 SDP 的前100个字符和换行符情况
            console.log('[WebRTC] 处理后的 SDP 类型:', typeof sdp)
            console.log('[WebRTC] SDP 前100字符:', typeof sdp === 'string' ? sdp.substring(0, 100) : 'N/A')
            if (typeof sdp === 'string') {
                console.log('[WebRTC] SDP 包含 \\r\\n:', sdp.includes('\r\n'))
                console.log('[WebRTC] SDP 包含 \\n:', sdp.includes('\n'))
            }

            // 确保 sdp 是字符串
            if (typeof sdp !== 'string') {
                console.error('[WebRTC] SDP 不是字符串类型:', typeof sdp, sdp)
                return
            }

            // 标准化 SDP 换行符为 CRLF（WebRTC SDP 规范要求）
            // 先统一为 LF，再转换为 CRLF
            sdp = sdp.replace(/\r\n/g, '\n').replace(/\r/g, '\n').replace(/\n/g, '\r\n')

            // 确保 SDP 以 CRLF 结尾
            if (!sdp.endsWith('\r\n')) {
                sdp += '\r\n'
            }

            console.log('[WebRTC] 标准化后 SDP 前100字符:', sdp.substring(0, 100))

            try {
                // 设置远端 SDP（直接使用对象，不需要 new RTCSessionDescription）
                await localPeerConnection.setRemoteDescription({ type: 'answer', sdp: sdp })
                console.log('[WebRTC] 已设置远端 SDP answer')

                // 添加服务器的 ICE 候选
                // 注意：必须在 setRemoteDescription 之后添加 candidates
                if (candidates && candidates.length > 0) {
                    for (const candidateStr of candidates) {
                        // 验证 candidate 字符串
                        if (!candidateStr || typeof candidateStr !== 'string' || candidateStr.trim() === '') {
                            console.warn('[WebRTC] 跳过无效的 ICE 候选:', candidateStr)
                            continue
                        }

                        try {
                            // 尝试解析 candidate 字符串
                            // candidate 格式通常是: "candidate:1 1 UDP 2130706431 192.168.1.1 54321 typ host"
                            // 如果包含 "a=" 前缀，需要去掉
                            let cleanCandidate = candidateStr.trim()
                            if (cleanCandidate.startsWith('a=')) {
                                cleanCandidate = cleanCandidate.substring(2)
                            }

                            // 创建 RTCIceCandidate 对象
                            // 不指定 sdpMid，让浏览器自动匹配
                            const candidate = new RTCIceCandidate({
                                candidate: cleanCandidate,
                                sdpMLineIndex: 0,
                            })

                            // 检查连接状态，避免在已关闭的连接上添加
                            if (localPeerConnection.connectionState === 'closed' ||
                                localPeerConnection.connectionState === 'failed') {
                                console.warn('[WebRTC] 连接已关闭或失败，跳过添加 ICE 候选')
                                break
                            }

                            await localPeerConnection.addIceCandidate(candidate)
                            console.log('[WebRTC] 添加服务器 ICE 候选成功:', cleanCandidate.substring(0, 50))
                        } catch (err: any) {
                            // 某些错误是可以忽略的（如 candidate 已过期、重复等）
                            if (err?.message?.includes('already been added') ||
                                err?.message?.includes('InvalidStateError') ||
                                err?.message?.includes('candidate')) {
                                console.warn('[WebRTC] ICE 候选添加失败（可忽略）:', err.message)
                            } else {
                                console.error('[WebRTC] 添加服务器 ICE 候选失败:', err, 'candidate:', candidateStr)
                            }
                        }
                    }
                }

                // 处理之前缓存的 ICE 候选（如果有）
                if (pendingCandidates && pendingCandidates.length > 0) {
                    for (const candidate of pendingCandidates) {
                        try {
                            // 检查连接状态
                            if (localPeerConnection.connectionState === 'closed' ||
                                localPeerConnection.connectionState === 'failed') {
                                break
                            }

                            await localPeerConnection.addIceCandidate(new RTCIceCandidate(candidate))
                            console.log('[WebRTC] 添加缓存 ICE 候选成功')
                        } catch (err: any) {
                            if (err?.message?.includes('already been added') ||
                                err?.message?.includes('InvalidStateError')) {
                                console.warn('[WebRTC] 缓存 ICE 候选添加失败（可忽略）:', err.message)
                            } else {
                                console.error('[WebRTC] 添加缓存 ICE 候选失败:', err)
                            }
                        }
                    }
                    setPendingCandidates([])
                }
            } catch (err) {
                console.error('[WebRTC] 处理 answer 失败:', err)
            }
        }

        // WebSocket 消息处理
        newSocket.onmessage = async (event) => {
            console.log('[WebSocket] 收到消息:', event.data)
            const data = JSON.parse(event.data)
            console.log('[WebSocket] 解析后的数据:', data)

            switch (data.type) {
                case 'init':
                    // 收到 init 消息，获取 sessionId，然后初始化 WebRTC
                    sessionId = data.session_id || ''
                    console.log('[WebSocket] 收到 init 消息，sessionId:', sessionId)
                    // 开始初始化 WebRTC
                    await initWebRTC()
                    break

                case 'answer':
                    await handleAnswer(data)
                    break

                case 'asrFinal':
                    console.log('[WebSocket] 收到 ASR 结果:', data.text)
                    addAIMessage(data.text)
                    break

                case 'ice-candidate':
                    // 服务器可能单独发送 ICE 候选（虽然当前后端使用 bundling 模式）
                    if (localPeerConnection && data.candidate) {
                        // 检查连接状态
                        if (localPeerConnection.connectionState === 'closed' ||
                            localPeerConnection.connectionState === 'failed') {
                            console.warn('[WebRTC] 连接已关闭或失败，跳过 ICE 候选')
                            break
                        }

                        try {
                            const candidate = new RTCIceCandidate(data.candidate)
                            if (localPeerConnection.remoteDescription && localPeerConnection.remoteDescription.type) {
                                await localPeerConnection.addIceCandidate(candidate)
                                console.log('[WebRTC] 添加 ICE 候选成功')
                            } else {
                                setPendingCandidates(prev => [...prev, data.candidate])
                                console.log('[WebRTC] 缓存 ICE 候选，等待 remoteDescription 设置')
                            }
                        } catch (err: any) {
                            if (err?.message?.includes('already been added') ||
                                err?.message?.includes('InvalidStateError')) {
                                console.warn('[WebRTC] ICE 候选添加失败（可忽略）:', err.message)
                            } else {
                                console.error('[WebRTC] 添加 ICE 候选失败:', err)
                            }
                        }
                    }
                    break

                default:
                    console.log('[WebSocket] 未知消息类型:', data.type)
            }
        }

        newSocket.onopen = () => {
            console.log('[WebSocket] 已连接，等待服务器 init 消息...')
            // 不再在 onopen 中立即初始化 WebRTC，而是等待 init 消息
        }

        newSocket.onerror = (error) => {
            console.error('[WebSocket] 连接出错:', error)
            showAlert('WebSocket连接出错', 'error')
        }

        newSocket.onclose = () => {
            console.log('[WebSocket] 连接关闭')
        }

        setSocket(newSocket)
    }

    // 使用 ref 防止 StrictMode 导致的重复请求
    const initializedRef = useRef(false)
    const loadingRef = useRef(false)
    
    // 初始化数据
    useEffect(() => {
        // 防止 StrictMode 导致的重复初始化
        if (initializedRef.current || loadingRef.current) {
            return
        }
        initializedRef.current = true
        loadingRef.current = true
        
        const initializeData = async () => {
            try {
                setLoading(true)

                // 检查是否为首次访问
                const hasVisited = localStorage.getItem('hasVisitedVoiceAssistant');
                if (!hasVisited) {
                    // 首次访问，自动启动引导
                    setTimeout(() => {
                        startOnboarding();
                    }, 1000);
                    // 标记已访问
                    localStorage.setItem('hasVisitedVoiceAssistant', 'true');
                }

                // 如果有特定的 assistantId（从 URL 参数获取），优先直接加载该助手配置
                // 这样可以更快地显示配置，而不是等待助手列表加载完成
                if (assistantId && assistantId > 0) {
                    try {
                        const response = await getAssistant(assistantId)
                        const detail = response.data
                        if (detail) {
                            // 使用 ?? 来判断，确保空字符串也能正确赋值
                            setSystemPrompt(detail.systemPrompt ?? '')
                            setSelectedSpeaker(detail.speaker !== undefined && detail.speaker !== null && detail.speaker !== '' ? detail.speaker : '101016')
                            setLanguage(detail.language !== undefined && detail.language !== null && detail.language !== '' ? detail.language : 'zh-cn')
                            setTemperature(detail.temperature ?? 0.6)
                            setLlmModel(detail.llmModel ?? '')
                            setMaxTokens(detail.maxTokens ?? 150)
                            // 加载训练音色配置（从后端读取，不自动选择）
                            if (detail.voiceCloneId !== null && detail.voiceCloneId !== undefined) {
                                setSelectedVoiceCloneId(detail.voiceCloneId)
                            } else {
                                setSelectedVoiceCloneId(null)
                            }
                            // 加载知识库配置
                            if (detail.knowledgeBaseId) {
                                setSelectedKnowledgeBase(detail.knowledgeBaseId)
                            } else {
                                setSelectedKnowledgeBase(null)
                            }
                            // 加载API密钥配置（即使为空字符串也要设置）
                            setApiKey(detail.apiKey ?? '')
                            setApiSecret(detail.apiSecret ?? '')
                            // 加载TTS提供商（即使为空字符串也要设置）
                            setTtsProvider(detail.ttsProvider ?? undefined)
                            // 加载助手基本信息
                            setAssistantName(detail.name ?? '')
                            setAssistantDescription(detail.description ?? '')
                            setAssistantIcon(detail.icon ?? 'Bot')
                            // 加载 VAD 配置
                            setEnableVAD(detail.enableVAD !== undefined ? detail.enableVAD : true)
                            setVadThreshold(detail.vadThreshold ?? 500)
                            setVadConsecutiveFrames(detail.vadConsecutiveFrames ?? 2)
                        }
                    } catch (err) {
                        console.warn('加载助手配置失败:', err)
                    }
                }

                // 并行加载助手列表（用于侧边栏显示，不阻塞配置加载）
                void getAssistantList().then(response => {
                    setAssistants(response.data as Assistant[])
                }).catch(err => {
                    console.warn('获取助手列表失败:', err)
                })

                // 获取聊天历史（只获取当前助手的记录）
                try {
                    if (assistantId && assistantId > 0) {
                        const historyResponse = await getChatSessionLogsByAssistant(assistantId, { pageSize: 20 })
                        if (historyResponse.data && historyResponse.data.logs) {
                            setChatHistory(historyResponse.data.logs.map(log => ({
                                id: log.id,
                                sessionId: log.sessionId,
                                content: log.preview,
                                createdAt: log.createdAt,
                                assistantName: log.assistantName,
                                chatType: log.chatType,
                                messageCount: log.messageCount || 1
                            })))
                        } else {
                            setChatHistory([])
                        }
                    } else {
                        setChatHistory([])
                    }
                } catch (historyErr) {
                    console.warn('获取聊天历史失败:', historyErr)
                    setChatHistory([])
                }

                // 获取知识库列表
                await fetchKnowledgeBases();

                // 获取音色列表（但不自动选择，等待从助手配置加载）
                try {
                    const response = await getVoiceClones()
                    const list = response.data || []
                    setVoiceClones(list)
                    // 移除自动选择逻辑，训练音色应该从后端助手配置中读取
                } catch (err) {
                    console.warn('获取音色列表失败:', err)
                    setVoiceClones([])
                }
            } catch (err) {
                console.error('初始化数据失败:', err)
                setError('加载数据失败')
                showAlert('加载数据失败', 'error')
            } finally {
                setLoading(false)
                loadingRef.current = false
            }
        }

        initializeData()
        
        // 清理函数：在组件卸载或 StrictMode 重新挂载时重置
        return () => {
            // 注意：在 StrictMode 下，这个清理函数会在第二次渲染前执行
            // 但我们不重置 initializedRef，因为第二次渲染时我们仍然希望跳过初始化
            loadingRef.current = false
        }
    }, [])

    // 当 assistantId 变化时（比如从 URL 参数或用户点击助手），自动加载助手配置
    // 注意：初始化时的加载已经在 initializeData 中直接处理了，这里主要处理切换助手的情况
    // 但是为了避免重复加载，我们只在助手列表加载完成后，且 assistantId 发生变化时才加载
    useEffect(() => {
        // 只在助手列表加载完成后才处理切换，避免初始化时重复加载
        if (assistantId && assistantId > 0 && assistants.length > 0) {
            // 防止重复调用同一个 agentId（StrictMode 会导致重复执行）
            if (lastSelectedAgentRef.current !== assistantId) {
                lastSelectedAgentRef.current = assistantId
                handleSelectAgent(assistantId)
            }
        }
    }, [assistantId, assistants.length])


    // 键盘快捷键支持
    useEffect(() => {
        const handleKeyDown = (event: KeyboardEvent) => {
            // Ctrl/Cmd + 1: 切换助手列表
            if ((event.ctrlKey || event.metaKey) && event.key === '1') {
                event.preventDefault()
                setIsAssistantListCollapsed(!isAssistantListCollapsed)
            }
            // Ctrl/Cmd + 2: 切换控制面板
            if ((event.ctrlKey || event.metaKey) && event.key === '2') {
                event.preventDefault()
                setIsControlPanelCollapsed(!isControlPanelCollapsed)
            }
            // Ctrl/Cmd + 0: 展开所有面板
            if ((event.ctrlKey || event.metaKey) && event.key === '0') {
                event.preventDefault()
                setIsAssistantListCollapsed(false)
                setIsControlPanelCollapsed(false)
            }
        }

        window.addEventListener('keydown', handleKeyDown)
        return () => window.removeEventListener('keydown', handleKeyDown)
    }, [isAssistantListCollapsed, isControlPanelCollapsed])

    // 组件卸载时清理资源
    useEffect(() => {
        return () => {
            console.log('[Cleanup] 组件卸载，清理资源')
            // 组件卸载时清理所有连接
            if (callTimer) {
                clearInterval(callTimer)
            }
            if (socket && socket.readyState !== WebSocket.CLOSED) {
                socket.close()
            }
            if (peerConnection && peerConnection.connectionState !== 'closed') {
                peerConnection.close()
            }
            if (localStream) {
                localStream.getTracks().forEach(track => track.stop())
            }
        }
    }, [])

    // 开始通话
    const startCall = async () => {
        if (assistantId === 0) {
            showAlert('请先选择一个AI助手', 'warning')
            return
        }

        // 校验API密钥（WebRTC和WebSocket模式都需要）
        if (!apiKey || !apiSecret) {
            showAlert('请先配置API密钥和密钥（右侧控制面板）', 'warning')
            // 展开控制面板提示用户
            setIsControlPanelCollapsed(false)
            return
        }

        try {
            setIsCalling(true)
            setCallDuration(0)
            setChatMessages([]) // 清空当前聊天记录

            // 根据线路选择连接方式
            if (lineMode === 'webrtc') {
                // 线路1：WebRTC实时通信
                connectWebSocket()
            } else if (lineMode === 'websocket') {
                // 线路2：通用WebSocket语音服务
                connectQiniuVoice()
            }

            // 开始通话计时器
            const timer = setInterval(() => {
                setCallDuration(prev => prev + 1)
            }, 1000)
            setCallTimer(timer)

            // 注意：成功提示将在WebSocket连接成功后显示，而不是在这里
            // 这样可以避免在没有密钥时仍然显示"通话已开始"的问题
        } catch (err: any) {
            console.error('通话启动失败:', err)
            setIsCalling(false)
            showAlert('通话启动失败', 'error')
        }
    }

    // 停止通话
    const stopCall = async () => {
        try {
            console.log('[StopCall] 开始停止通话')

            // 停止通话计时器
            if (callTimer) {
                clearInterval(callTimer)
                setCallTimer(null)
            }

            // 根据线路类型停止相应的连接
            if (lineMode === 'webrtc') {
                // 停止WebRTC连接
                if (peerConnection && peerConnection.connectionState !== 'closed') {
                    peerConnection.close()
                    setPeerConnection(null)
                }
            } else if (lineMode === 'websocket') {
                // 停止WebSocket录音
                if (mediaRecorderRef.current && mediaRecorderRef.current.state !== 'inactive') {
                    mediaRecorderRef.current.stop()
                    mediaRecorderRef.current = null
                }
            }

            // 关闭WebSocket连接（先发送关闭消息）
            if (socket && socket.readyState !== WebSocket.CLOSED) {
                try {
                    // 发送关闭消息通知后端
                    socket.send(JSON.stringify({
                        type: 'close',
                        session_id: currentSessionId || '',
                        data: {}
                    }))
                } catch (err) {
                    console.warn('[StopCall] 发送关闭消息失败:', err)
                }
                // 等待一小段时间让消息发送，然后关闭连接
                setTimeout(() => {
                    if (socket && socket.readyState !== WebSocket.CLOSED) {
                        socket.close()
                    }
                    setSocket(null)
                }, 100)
            } else {
                setSocket(null)
            }

            // 停止本地音频流
            if (localStream) {
                localStream.getTracks().forEach(track => track.stop())
                setLocalStream(null)
            }

            setIsCalling(false)
            setCallDuration(0)
            setPendingCandidates([])

            console.log('[StopCall] 通话已结束')
            showAlert('通话已结束', 'success')
            return true
        } catch (err: any) {
            console.error('终止通话失败:', err)
            showAlert('终止通话失败', 'error')
            return false
        }
    }

    // 开始新会话
    const startNewSession = async () => {
        setCurrentSessionId(null)
        setChatMessages([])
        showAlert('已开始新会话', 'success')

        // 刷新聊天记录
        try {
            if (assistantId && assistantId > 0) {
                const historyResponse = await getChatSessionLogsByAssistant(assistantId, { pageSize: 20 })
                if (historyResponse.data && historyResponse.data.logs) {
                    setChatHistory(historyResponse.data.logs.map(log => ({
                        id: log.id,
                        sessionId: log.sessionId,
                        content: log.preview,
                        createdAt: log.createdAt,
                        assistantName: log.assistantName,
                        chatType: log.chatType,
                        messageCount: log.messageCount || 1
                    })))
                } else {
                    setChatHistory([])
                }
            } else {
                setChatHistory([])
            }
        } catch (err) {
            console.warn('刷新聊天记录失败:', err)
        }
    }

    // 切换助手
    const handleSelectAgent = async (agentId: number) => {
        if (isCalling) {
            setPendingAgent(agentId)
            setShowConfirmModal(true)
            return
        }

        try {
            const response = await getAssistant(agentId)
            const detail = response.data

            if (detail) {
                // 使用 ?? 来判断，确保空字符串也能正确赋值
                setSystemPrompt(detail.systemPrompt ?? '')
                setSelectedSpeaker(detail.speaker !== undefined && detail.speaker !== null && detail.speaker !== '' ? detail.speaker : '101016')
                setLanguage(detail.language !== undefined && detail.language !== null && detail.language !== '' ? detail.language : 'zh-cn')
                setTemperature(detail.temperature ?? 0.6)
                setLlmModel(detail.llmModel ?? '')
                setMaxTokens(detail.maxTokens ?? 150)
                // 加载训练音色配置（从后端读取，不自动选择）
                if (detail.voiceCloneId !== null && detail.voiceCloneId !== undefined) {
                    setSelectedVoiceCloneId(detail.voiceCloneId)
                } else {
                    setSelectedVoiceCloneId(null)
                }
                // 加载知识库配置
                if (detail.knowledgeBaseId) {
                    setSelectedKnowledgeBase(detail.knowledgeBaseId)
                } else {
                    setSelectedKnowledgeBase(null)
                }
                // 加载API密钥配置（即使为空字符串也要设置）
                setApiKey(detail.apiKey ?? '')
                setApiSecret(detail.apiSecret ?? '')
                // 加载TTS提供商（即使为空字符串也要设置）
                setTtsProvider(detail.ttsProvider ?? undefined)
                // 加载助手基本信息
                setAssistantName(detail.name ?? '')
                setAssistantDescription(detail.description ?? '')
                setAssistantIcon(detail.icon ?? 'Bot')
                // 加载 VAD 配置
                setEnableVAD(detail.enableVAD !== undefined ? detail.enableVAD : true)
                setVadThreshold(detail.vadThreshold ?? 500)
                setVadConsecutiveFrames(detail.vadConsecutiveFrames ?? 2)

                // 重新加载当前助手的聊天记录
                try {
                    const historyResponse = await getChatSessionLogsByAssistant(agentId, { pageSize: 20 })
                    if (historyResponse.data && historyResponse.data.logs) {
                        setChatHistory(historyResponse.data.logs.map(log => ({
                            id: log.id,
                            sessionId: log.sessionId,
                            content: log.preview,
                            createdAt: log.createdAt,
                            assistantName: log.assistantName,
                            chatType: log.chatType,
                            messageCount: log.messageCount || 1
                        })))
                    } else {
                        setChatHistory([])
                    }
                } catch (historyErr) {
                    console.warn('获取聊天历史失败:', historyErr)
                    setChatHistory([])
                }
            }
        } catch (err: any) {
            console.error('获取助手详情失败:', err)

            // 检查是否是API错误响应
            if (err.response && err.response.data && err.response.data.msg) {
                showAlert(err.response.data.msg, 'error')
            } else if (err.message) {
                showAlert(err.message, 'error')
            } else {
                showAlert('获取助手详情失败', 'error')
            }
        }
    }

    // 确认切换助手
    const confirmSwitch = async () => {
        setShowConfirmModal(false)
        const success = await stopCall()
        if (success && pendingAgent) {
            handleSelectAgent(pendingAgent)
            setPendingAgent(null)
        }
    }

    // 添加助手
    const handleAddAssistant = async (assistant: { name: string; description: string; icon: string }) => {
        try {
            await createAssistant(assistant)

            // 刷新助手列表
            const assistantsResponse = await getAssistantList()
            setAssistants(assistantsResponse.data as Assistant[])

            showAlert('助手创建成功', 'success')
        } catch (err: any) {
            console.error('创建助手失败:', err)

            // 检查是否是API错误响应
            if (err.response && err.response.data && err.response.data.msg) {
                showAlert(err.response.data.msg, 'error')
            } else if (err.message) {
                showAlert(err.message, 'error')
            } else {
                showAlert('创建助手失败', 'error')
            }
        }
    }

    // 保存设置
    const handleSaveSettings = async () => {
        try {
            await updateAssistant(assistantId, {
                name: assistantName,
                description: assistantDescription,
                icon: assistantIcon,
                systemPrompt,
                persona_tag: assistants.find(a => a.id === assistantId)?.name || '',
                temperature: temperature,
                maxTokens: maxTokens,
                language,
                speaker: selectedSpeaker,
                voiceCloneId: selectedVoiceCloneId,
                knowledgeBaseId: selectedKnowledgeBase,
                ttsProvider: ttsProvider || '',
                apiKey,
                apiSecret,
                llmModel,
                enableGraphMemory,
                enableVAD,
                vadThreshold,
                vadConsecutiveFrames,
            })

            // 更新JS模板
            if (selectedJSTemplate !== null) {
                await updateAssistantJS(assistantId, selectedJSTemplate)
            }

            // 刷新助手列表以更新显示
            const assistantsResponse = await getAssistantList()
            setAssistants(assistantsResponse.data as Assistant[])

            showAlert('设置保存成功', 'success')
        } catch (err: any) {
            console.error('保存设置失败:', err)

            // 检查是否是API错误响应
            if (err.response && err.response.data && err.response.data.msg) {
                showAlert(err.response.data.msg, 'error')
            } else if (err.message) {
                showAlert(err.message, 'error')
            } else {
                showAlert('保存设置失败', 'error')
            }
        }
    }

    // 删除助手
    const handleDeleteAssistant = async () => {
        try {
            await deleteAssistant(assistantId)
            setShowDeleteConfirm(false)
            setSystemPrompt('')
            setLlmModel('')

            showAlert('助手删除成功', 'success')
            // 跳转到助手列表页面
            navigate('/assistants')
        } catch (err: any) {
            console.error('删除助手失败:', err)

            // 检查是否是API错误响应
            if (err.response && err.response.data && err.response.data.msg) {
                showAlert(err.response.data.msg, 'error')
            } else if (err.message) {
                showAlert(err.message, 'error')
            } else {
                showAlert('删除助手失败', 'error')
            }
        }
    }

    // 查看聊天记录详情
    const handleSessionClick = async (logId: number, sessionId?: string) => {
        try {
            console.log('点击聊天记录:', logId, 'sessionId:', sessionId)

            // 如果有 sessionId，获取该 session 的所有记录
            if (sessionId) {
                const response = await getChatSessionLogsBySession(sessionId)
                console.log('会话记录响应:', response)

                if (response.data && response.data.length > 0) {
                    // 创建一个包含所有记录的详情对象
                    const sessionDetails = {
                        sessionId: sessionId,
                        logs: response.data,
                        assistantName: response.data[0]?.assistantName || '未知助手',
                        chatType: response.data[0]?.chatType || 'text'
                    }
                    setSelectedLogDetail(sessionDetails as any)
                    setShowLogModal(true)
                } else {
                    showAlert('未找到会话记录', 'warning')
                }
            } else {
                // 如果没有 sessionId，使用原来的方式获取单条记录
                const response = await getChatSessionLogDetail(logId)
                console.log('聊天记录详情响应:', response)

                if (response.data) {
                    setSelectedLogDetail(response.data)
                    setShowLogModal(true)
                } else {
                    showAlert('未找到聊天记录详情', 'warning')
                }
            }
        } catch (err: any) {
            console.error('获取聊天记录详情失败:', err)

            // 检查是否是API错误响应
            if (err.response && err.response.data && err.response.data.msg) {
                showAlert(err.response.data.msg, 'error')
            } else if (err.message) {
                showAlert(err.message, 'error')
            } else {
                showAlert('获取聊天记录详情失败', 'error')
            }
        }
    }

    // 处理接入方法点击
    const handleMethodClick = (method: string) => {
        if (assistantId === 0) {
            showAlert('请先选择一个AI助手', 'warning')
            return
        }
        setSelectedMethod(method)
        setShowIntegrationModal(true)
    }

    const handleJSTemplateChange = (value: string) => {
        setSelectedJSTemplate(value)
    }

    return (
        <div className="h-[100vh] overflow-hidden flex flex-col bg-gradient-to-br from-blue-50 to-purple-50 dark:from-neutral-900 dark:to-neutral-800">
            <div className="flex flex-1 overflow-hidden">
                {/* 左侧边栏（语音球+历史会话） */}
                <div
                    className="w-64 flex flex-col border-r dark:border-neutral-700 bg-white dark:bg-neutral-800 overflow-hidden">
                    {/* 语音球区域 */}
                    <div className="p-4 border-b dark:border-neutral-700">
                        {/* 线路选择 */}
                        <LineSelector
                            lineMode={lineMode}
                            onLineModeChange={setLineMode}
                        />

                        <div
                            ref={voiceBallRef}
                            data-highlighted={showOnboarding && highlightedElement === 'voice-ball' ? 'true' : 'false'}
                            className={highlightedElement === 'voice-ball' ? 'ring-2 ring-blue-500 rounded-xl p-1' : ''}
                        >
                            <VoiceBall
                                isCalling={isCalling}
                                onToggleCall={() => {
                                    return isCalling ? stopCall() : startCall()
                                }}
                            />
                            {showOnboarding && highlightedElement === 'voice-ball' && (
                                <GuideTooltip
                                    text={onboardingSteps[onboardingStep].text}
                                    position={onboardingSteps[onboardingStep].position as any}
                                    onNext={handleNextStep}
                                    onClose={handleSkipOnboarding}
                                />
                            )}
                        </div>

                        {/* 状态指示 */}
                        <div className="mt-4 text-center">
                            <h2 className={`text-lg font-bold transition-all duration-300 ${
                                isCalling ? 'text-purple-600' : 'text-blue-500'
                            }`}>
                                {isCalling ? '通话中...' : '待机中'}
                            </h2>
                            <p className="text-xs text-gray-500 mt-1">
                                {assistants.find(a => a.id === assistantId)?.name || '未选择助手'}
                            </p>
                            {isCalling && (
                                <p className="text-sm text-purple-600 mt-2 font-mono">
                                    {Math.floor(callDuration / 60).toString().padStart(2, '0')}:
                                    {(callDuration % 60).toString().padStart(2, '0')}
                                </p>
                            )}
                        </div>
                    </div>

                    {/* 历史会话列表 */}
                    <ChatHistory
                        chatHistory={chatHistory}
                        loading={loading}
                        error={error}
                        onSessionClick={handleSessionClick}
                    />
                </div>

                {/* 聊天主区域 */}
                <main className="flex-1 flex overflow-hidden">
                    {/* 左侧客服选择栏 */}
                    <div
                        ref={chatAreaRef}
                        data-highlighted={showOnboarding && highlightedElement === 'chat-area' ? 'true' : 'false'}
                        className={`flex-1 flex flex-col overflow-hidden relative ${showOnboarding && highlightedElement === 'chat-area' ? 'border-2 border-blue-500' : ''}`}
                    >
                        <ChatArea
                            messages={chatMessages}
                            isCalling={isCalling}
                            isGlobalMuted={isGlobalMuted}
                            onMuteToggle={setIsGlobalMuted}
                            onNewSession={startNewSession}
                            assistantName={assistants.find(a => a.id === assistantId)?.name}
                        />
                        {showOnboarding && highlightedElement === 'chat-area' && (
                            <GuideTooltip
                                text={onboardingSteps[onboardingStep].text}
                                position={onboardingSteps[onboardingStep].position as any}
                                onNext={handleNextStep}
                                onClose={handleSkipOnboarding}
                                isLast={onboardingSteps[onboardingStep].isLast}
                            />
                        )}

                        {/* 文本输入框 */}
                        <TextInputBox
                            inputValue={voiceAssistantHook.inputValue}
                            onInputChange={voiceAssistantHook.setInputValue}
                            isWaitingForResponse={voiceAssistantHook.isWaitingForResponse}
                            onEnter={voiceAssistantHook.handleInputEnter}
                            onSend={voiceAssistantHook.handleSendClick}
                            textMode={textMode}
                            onTextModeChange={setTextMode}
                            inputRef={voiceAssistantHook.inputRef}
                            textInputRef={textInputRef}
                        />
                        {showOnboarding && highlightedElement === 'text-input' && (
                            <GuideTooltip
                                text={onboardingSteps[onboardingStep].text}
                                position={onboardingSteps[onboardingStep].position as any}
                                onNext={handleNextStep}
                                onClose={handleSkipOnboarding}
                                isLast={onboardingSteps[onboardingStep].isLast}
                            />
                        )}
                    </div>
                </main>

                {/* 右侧控制面板 */}
                <div
                    ref={controlPanelRef}
                    data-highlighted={showOnboarding && highlightedElement === 'control-panel' ? 'true' : 'false'}
                    className={`transition-all duration-300 ease-in-out ${
                        isControlPanelCollapsed ? 'w-12' : 'w-[28rem]'
                    } flex flex-col border-l dark:border-neutral-700 bg-white dark:bg-neutral-800 overflow-hidden`}
                >
                    {/* 折叠按钮 */}
                    <div className="p-2 border-b dark:border-neutral-700">
                        <button
                            onClick={() => setIsControlPanelCollapsed(!isControlPanelCollapsed)}
                            className="w-full p-2 hover:bg-gray-100 dark:hover:bg-neutral-700 rounded-lg transition-colors"
                            title={`${isControlPanelCollapsed ? '展开控制面板' : '折叠控制面板'} (Ctrl+2)`}
                        >
                            <div className="flex items-center justify-center">
                                {isControlPanelCollapsed ? (
                                    <svg className="w-5 h-5 text-gray-600 dark:text-gray-300" fill="none" stroke="currentColor"
                                         viewBox="0 0 24 24">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7"/>
                                    </svg>
                                ) : (
                                    <svg className="w-5 h-5 text-gray-600 dark:text-gray-300" fill="none" stroke="currentColor"
                                         viewBox="0 0 24 24">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7"/>
                                    </svg>
                                )}
                            </div>
                        </button>
                    </div>

                    {/* 控制面板内容 */}
                    {!isControlPanelCollapsed && assistantId !== 0 && (
                        <div className={`flex-1 overflow-y-auto custom-scrollbar ${showOnboarding && highlightedElement === 'control-panel' ? 'border-2 border-blue-500' : ''}`}>
                            <ControlPanel
                                apiKey={apiKey}
                                apiSecret={apiSecret}
                                onApiKeyChange={setApiKey}
                                onApiSecretChange={setApiSecret}
                                ttsProvider={ttsProvider}
                                language={language}
                                selectedSpeaker={selectedSpeaker}
                                systemPrompt={systemPrompt}
                                temperature={temperature}
                                maxTokens={maxTokens}
                                llmModel={llmModel}
                                onLanguageChange={setLanguage}
                                onSpeakerChange={setSelectedSpeaker}
                                onSystemPromptChange={setSystemPrompt}
                                onTemperatureChange={setTemperature}
                                onMaxTokensChange={setMaxTokens}
                                onLlmModelChange={setLlmModel}
                                assistantName={assistantName}
                                assistantDescription={assistantDescription}
                                assistantIcon={assistantIcon}
                                enableGraphMemory={enableGraphMemory}
                                onAssistantNameChange={setAssistantName}
                                onAssistantDescriptionChange={setAssistantDescription}
                                onAssistantIconChange={setAssistantIcon}
                                onEnableGraphMemoryChange={setEnableGraphMemory}
                                enableVAD={enableVAD}
                                vadThreshold={vadThreshold}
                                vadConsecutiveFrames={vadConsecutiveFrames}
                                onEnableVADChange={setEnableVAD}
                                onVADThresholdChange={setVadThreshold}
                                onVADConsecutiveFramesChange={setVadConsecutiveFrames}
                                onSaveSettings={handleSaveSettings}
                                onDeleteAssistant={() => setShowDeleteConfirm(true)}
                                searchKeyword={searchKeyword}
                                highlightFragments={highlightFragments}
                                highlightResultId={highlightResultId}
                                selectedJSTemplate={selectedJSTemplate}
                                onJSTemplateChange={handleJSTemplateChange}
                                onMethodClick={handleMethodClick}
                                selectedKnowledgeBase={selectedKnowledgeBase}
                                onKnowledgeBaseChange={setSelectedKnowledgeBase}
                                knowledgeBases={knowledgeBases}
                                onManageKnowledgeBases={handleManageKnowledgeBases}
                                onRefreshKnowledgeBases={fetchKnowledgeBases}
                                // 添加训练音色相关配置
                                selectedVoiceCloneId={selectedVoiceCloneId}
                                onVoiceCloneChange={setSelectedVoiceCloneId}
                                voiceClones={voiceClones}
                                onRefreshVoiceClones={async () => {
                                    try {
                                        const response = await getVoiceClones()
                                        const list = response.data || []
                                        setVoiceClones(list)
                                        // 移除自动选择逻辑，训练音色应该从后端助手配置中读取
                                        showAlert('音色已刷新', 'success')
                                    } catch (err: any) {
                                        console.error('刷新音色失败:', err)
                                        showAlert(err?.msg || err?.message || '刷新音色失败', 'error')
                                    }
                                }}
                                onNavigateToVoiceTraining={() => navigate('/voice-training')}
                            />
                            {showOnboarding && highlightedElement === 'control-panel' && (
                                <GuideTooltip
                                    text={onboardingSteps[onboardingStep].text}
                                    position={onboardingSteps[onboardingStep].position as any}
                                    onNext={handleNextStep}
                                    onClose={handleSkipOnboarding}
                                />
                            )}
                        </div>
                    )}

                    {/* 未选择助手时的提示 */}
                    {!isControlPanelCollapsed && assistantId === 0 && (
                        <div className="flex-1 flex items-center justify-center p-6">
                            <div className="text-center text-gray-500 dark:text-gray-400">
                                <Settings className="w-12 h-12 mx-auto mb-4 opacity-50" />
                                <p className="text-lg font-medium mb-2">请先选择一个助手</p>
                                <p className="text-sm">选择助手后可以配置相关参数</p>
                            </div>
                        </div>
                    )}
                </div>

            </div>

            {/* 模态框 */}
            <AddAssistantModal
                isOpen={showAddAssistantModal}
                onClose={() => setShowAddAssistantModal(false)}
                onAdd={handleAddAssistant}
            />

            <IntegrationModal
                isOpen={showIntegrationModal}
                onClose={() => setShowIntegrationModal(false)}
                selectedMethod={selectedMethod}
                selectedAgent={assistantId}
                jsSourceId={jsSourceId}
            />

            {/* 确认切换助手模态框 */}
            {showConfirmModal && (
                <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
                    <div className="bg-white dark:bg-neutral-800 p-6 rounded-xl max-w-md w-full mx-4">
                        <h3 className="text-lg font-semibold mb-4">切换助手确认</h3>
                        <p className="text-gray-600 dark:text-gray-300 mb-6">
                            当前正在通话中，切换助手将结束当前通话。确定要切换吗？
                        </p>
                        <div className="flex justify-end space-x-4">
                            <button
                                onClick={() => setShowConfirmModal(false)}
                                className="px-4 py-2 text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-neutral-700 rounded-lg"
                            >
                                取消
                            </button>
                            <button
                                onClick={confirmSwitch}
                                className="px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700"
                            >
                                确定切换
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {/* 删除确认模态框 */}
            {showDeleteConfirm && (
                <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
                    <div className="bg-white dark:bg-neutral-800 p-6 rounded-xl max-w-md w-full mx-4">
                        <h3 className="text-lg font-semibold mb-4">删除确认</h3>
                        <p className="text-gray-600 dark:text-gray-300 mb-6">
                            确定要删除当前助手吗？此操作不可撤销。
                        </p>
                        <div className="flex justify-end space-x-4">
                            <button
                                onClick={() => setShowDeleteConfirm(false)}
                                className="px-4 py-2 text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-neutral-700 rounded-lg"
                            >
                                取消
                            </button>
                            <button
                                onClick={handleDeleteAssistant}
                                className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700"
                            >
                                确定删除
                            </button>
                        </div>
                    </div>
                </div>
            )}


            {/* 聊天记录详情模态框 */}
            {showLogModal && selectedLogDetail && (
                <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
                    <div className="bg-white dark:bg-neutral-800 p-6 rounded-xl w-full max-w-2xl mx-4">
                        <div className="flex justify-between items-center mb-4">
                            <h2 className="text-lg font-semibold">
                                {selectedLogDetail.logs ? `对话详情 (${selectedLogDetail.logs.length} 条消息)` : '对话详情'}
                            </h2>
                            <button
                                onClick={() => setShowLogModal(false)}
                                className="text-gray-400 hover:text-gray-700 dark:hover:text-white"
                            >
                                ✕
                            </button>
                        </div>
                        <div
                            className="max-h-[60vh] overflow-y-auto border rounded-lg p-4 bg-gray-50 dark:bg-neutral-700 space-y-4 custom-scrollbar">
                            {/* 如果有多条记录，显示所有记录 */}
                            {selectedLogDetail.logs && Array.isArray(selectedLogDetail.logs) ? (
                                selectedLogDetail.logs.map((log: any, index: number) => (
                                    <div key={log.id || index} className="space-y-3">
                                        {/* 分隔线（除了第一条） */}
                                        {index > 0 && (
                                            <div className="border-t border-gray-300 dark:border-neutral-600 my-4"></div>
                                        )}

                                        {/* 用户消息 */}
                                        {log.userMessage && (
                                            <div className="bg-blue-50 dark:bg-blue-900/20 p-3 rounded-lg">
                                                <div className="flex items-center mb-2">
                                                    <div className="w-6 h-6 bg-blue-500 rounded-full flex items-center justify-center text-white text-xs font-medium mr-2">
                                                        我
                                                    </div>
                                                    <span className="text-sm font-medium text-blue-700 dark:text-blue-300">用户</span>
                                                    <span className="ml-auto text-xs text-gray-400 dark:text-gray-500">
                                {new Date(log.createdAt).toLocaleString('zh-CN')}
                              </span>
                                                </div>
                                                <div className="text-sm text-gray-800 dark:text-gray-100 whitespace-pre-wrap">
                                                    {log.userMessage}
                                                </div>
                                            </div>
                                        )}

                                        {/* AI回复 */}
                                        {log.agentMessage && (
                                            <div className="bg-green-50 dark:bg-green-900/20 p-3 rounded-lg">
                                                <div className="flex items-center mb-2">
                                                    <div className="w-6 h-6 bg-green-500 rounded-full flex items-center justify-center text-white text-xs font-medium mr-2">
                                                        AI
                                                    </div>
                                                    <span className="text-sm font-medium text-green-700 dark:text-green-300">
                                {log.assistantName || selectedLogDetail.assistantName || 'AI助手'}
                              </span>
                                                    <span className="ml-auto text-xs text-gray-400 dark:text-gray-500">
                                {new Date(log.createdAt).toLocaleString('zh-CN')}
                              </span>
                                                </div>
                                                <div className="text-sm text-gray-800 dark:text-gray-100 whitespace-pre-wrap">
                                                    {log.agentMessage}
                                                </div>
                                            </div>
                                        )}

                                        {/* LLM Usage 信息 */}
                                        {log.llmUsage && (
                                            <div className="mt-2 p-3 bg-purple-50 dark:bg-purple-900/20 border border-purple-200 dark:border-purple-800 rounded-lg">
                                                <h5 className="text-xs font-semibold text-purple-900 dark:text-purple-100 mb-2">
                                                    LLM 使用信息
                                                </h5>
                                                <div className="grid grid-cols-2 gap-2 text-xs">
                                                    <div>
                                                        <span className="text-purple-700 dark:text-purple-300 font-medium">模型:</span>
                                                        <span className="ml-2 text-purple-900 dark:text-purple-100">{log.llmUsage.model}</span>
                                                    </div>
                                                    <div>
                                                        <span className="text-purple-700 dark:text-purple-300 font-medium">Total Tokens:</span>
                                                        <span className="ml-2 text-purple-900 dark:text-purple-100 font-semibold">{log.llmUsage.totalTokens}</span>
                                                    </div>
                                                    <div>
                                                        <span className="text-purple-700 dark:text-purple-300 font-medium">Prompt:</span>
                                                        <span className="ml-2 text-purple-900 dark:text-purple-100">{log.llmUsage.promptTokens}</span>
                                                    </div>
                                                    <div>
                                                        <span className="text-purple-700 dark:text-purple-300 font-medium">Completion:</span>
                                                        <span className="ml-2 text-purple-900 dark:text-purple-100">{log.llmUsage.completionTokens}</span>
                                                    </div>
                                                    {log.llmUsage.duration !== undefined && (
                                                        <div>
                                                            <span className="text-purple-700 dark:text-purple-300 font-medium">耗时:</span>
                                                            <span className="ml-2 text-purple-900 dark:text-purple-100">
                                    {log.llmUsage.duration >= 1000
                                        ? `${(log.llmUsage.duration / 1000).toFixed(2)}s`
                                        : `${log.llmUsage.duration}ms`}
                                  </span>
                                                        </div>
                                                    )}
                                                    {log.llmUsage.hasToolCalls && (
                                                        <div>
                                                            <span className="text-purple-700 dark:text-purple-300 font-medium">工具调用:</span>
                                                            <span className="ml-2 text-purple-900 dark:text-purple-100">
                                    {log.llmUsage.toolCallCount || 0} 个
                                  </span>
                                                        </div>
                                                    )}
                                                </div>
                                                {/* 工具调用详情 */}
                                                {log.llmUsage.hasToolCalls && log.llmUsage.toolCalls && log.llmUsage.toolCalls.length > 0 && (
                                                    <div className="mt-2 pt-2 border-t border-purple-200 dark:border-purple-700">
                                                        <div className="text-xs text-purple-700 dark:text-purple-300 font-medium mb-1">工具调用详情:</div>
                                                        <div className="space-y-1">
                                                            {log.llmUsage.toolCalls.map((toolCall: ToolCallInfo, idx: number) => (
                                                                <div key={idx} className="text-xs text-purple-900 dark:text-purple-100">
                                                                    <span className="font-medium">{toolCall.name}</span>
                                                                    {toolCall.arguments && (
                                                                        <span className="ml-1 text-purple-600 dark:text-purple-400">
                                          ({toolCall.arguments.length > 30 ? `${toolCall.arguments.substring(0, 30)}...` : toolCall.arguments})
                                        </span>
                                                                    )}
                                                                </div>
                                                            ))}
                                                        </div>
                                                    </div>
                                                )}
                                            </div>
                                        )}
                                    </div>
                                ))
                            ) : (
                                <>
                                    {/* 单条记录显示（兼容旧格式） */}
                                    {/* 用户消息 */}
                                    {selectedLogDetail.userMessage && (
                                        <div className="bg-blue-50 dark:bg-blue-900/20 p-3 rounded-lg">
                                            <div className="flex items-center mb-2">
                                                <div
                                                    className="w-6 h-6 bg-blue-500 rounded-full flex items-center justify-center text-white text-xs font-medium mr-2">
                                                    我
                                                </div>
                                                <span className="text-sm font-medium text-blue-700 dark:text-blue-300">用户</span>
                                            </div>
                                            <div className="text-sm text-gray-800 dark:text-gray-100 whitespace-pre-wrap">
                                                {selectedLogDetail.userMessage}
                                            </div>
                                        </div>
                                    )}

                                    {/* AI回复 */}
                                    {selectedLogDetail.agentMessage && (
                                        <div className="bg-green-50 dark:bg-green-900/20 p-3 rounded-lg">
                                            <div className="flex items-center mb-2">
                                                <div
                                                    className="w-6 h-6 bg-green-500 rounded-full flex items-center justify-center text-white text-xs font-medium mr-2">
                                                    AI
                                                </div>
                                                <span className="text-sm font-medium text-green-700 dark:text-green-300">
                            {selectedLogDetail.assistantName || 'AI助手'}
                          </span>
                                            </div>
                                            <div className="text-sm text-gray-800 dark:text-gray-100 whitespace-pre-wrap">
                                                {selectedLogDetail.agentMessage}
                                            </div>
                                        </div>
                                    )}

                                    {/* LLM Usage 信息 */}
                                    {selectedLogDetail.llmUsage && (
                                        <div className="mt-6 p-4 bg-purple-50 dark:bg-purple-900/20 border border-purple-200 dark:border-purple-800 rounded-lg">
                                            <h4 className="text-sm font-semibold text-purple-900 dark:text-purple-100 mb-3">
                                                LLM 使用信息
                                            </h4>
                                            <div className="grid grid-cols-2 gap-3 text-xs">
                                                <div>
                                                    <span className="text-purple-700 dark:text-purple-300 font-medium">模型:</span>
                                                    <span className="ml-2 text-purple-900 dark:text-purple-100">{selectedLogDetail.llmUsage.model}</span>
                                                </div>
                                                <div>
                                                    <span className="text-purple-700 dark:text-purple-300 font-medium">Prompt Tokens:</span>
                                                    <span className="ml-2 text-purple-900 dark:text-purple-100">{selectedLogDetail.llmUsage.promptTokens}</span>
                                                </div>
                                                <div>
                                                    <span className="text-purple-700 dark:text-purple-300 font-medium">Completion Tokens:</span>
                                                    <span className="ml-2 text-purple-900 dark:text-purple-100">{selectedLogDetail.llmUsage.completionTokens}</span>
                                                </div>
                                                <div>
                                                    <span className="text-purple-700 dark:text-purple-300 font-medium">Total Tokens:</span>
                                                    <span className="ml-2 text-purple-900 dark:text-purple-100 font-semibold">{selectedLogDetail.llmUsage.totalTokens}</span>
                                                </div>
                                                {selectedLogDetail.llmUsage.temperature !== undefined && (
                                                    <div>
                                                        <span className="text-purple-700 dark:text-purple-300 font-medium">Temperature:</span>
                                                        <span className="ml-2 text-purple-900 dark:text-purple-100">{selectedLogDetail.llmUsage.temperature}</span>
                                                    </div>
                                                )}
                                                {selectedLogDetail.llmUsage.maxTokens !== undefined && (
                                                    <div>
                                                        <span className="text-purple-700 dark:text-purple-300 font-medium">Max Tokens:</span>
                                                        <span className="ml-2 text-purple-900 dark:text-purple-100">{selectedLogDetail.llmUsage.maxTokens}</span>
                                                    </div>
                                                )}
                                                {selectedLogDetail.llmUsage.topP !== undefined && (
                                                    <div>
                                                        <span className="text-purple-700 dark:text-purple-300 font-medium">Top P:</span>
                                                        <span className="ml-2 text-purple-900 dark:text-purple-100">{selectedLogDetail.llmUsage.topP}</span>
                                                    </div>
                                                )}
                                                {selectedLogDetail.llmUsage.finishReason && (
                                                    <div>
                                                        <span className="text-purple-700 dark:text-purple-300 font-medium">Finish Reason:</span>
                                                        <span className="ml-2 text-purple-900 dark:text-purple-100">{selectedLogDetail.llmUsage.finishReason}</span>
                                                    </div>
                                                )}
                                                {selectedLogDetail.llmUsage.stream !== undefined && (
                                                    <div>
                                                        <span className="text-purple-700 dark:text-purple-300 font-medium">Stream:</span>
                                                        <span className="ml-2 text-purple-900 dark:text-purple-100">{selectedLogDetail.llmUsage.stream ? '是' : '否'}</span>
                                                    </div>
                                                )}
                                                {/* 时间统计 */}
                                                {selectedLogDetail.llmUsage.duration !== undefined && (
                                                    <div>
                                                        <span className="text-purple-700 dark:text-purple-300 font-medium">调用耗时:</span>
                                                        <span className="ml-2 text-purple-900 dark:text-purple-100">
                                  {selectedLogDetail.llmUsage.duration >= 1000
                                      ? `${(selectedLogDetail.llmUsage.duration / 1000).toFixed(2)}s`
                                      : `${selectedLogDetail.llmUsage.duration}ms`}
                                </span>
                                                    </div>
                                                )}
                                                {selectedLogDetail.llmUsage.startTime && (
                                                    <div>
                                                        <span className="text-purple-700 dark:text-purple-300 font-medium">开始时间:</span>
                                                        <span className="ml-2 text-purple-900 dark:text-purple-100">
                                  {new Date(selectedLogDetail.llmUsage.startTime).toLocaleString('zh-CN')}
                                </span>
                                                    </div>
                                                )}
                                                {selectedLogDetail.llmUsage.endTime && (
                                                    <div>
                                                        <span className="text-purple-700 dark:text-purple-300 font-medium">结束时间:</span>
                                                        <span className="ml-2 text-purple-900 dark:text-purple-100">
                                  {new Date(selectedLogDetail.llmUsage.endTime).toLocaleString('zh-CN')}
                                </span>
                                                    </div>
                                                )}
                                                {/* 工具调用统计 */}
                                                {selectedLogDetail.llmUsage.hasToolCalls !== undefined && (
                                                    <div>
                                                        <span className="text-purple-700 dark:text-purple-300 font-medium">工具调用:</span>
                                                        <span className="ml-2 text-purple-900 dark:text-purple-100">
                                  {selectedLogDetail.llmUsage.hasToolCalls ? '是' : '否'}
                                                            {selectedLogDetail.llmUsage.hasToolCalls && selectedLogDetail.llmUsage.toolCallCount !== undefined && (
                                                                <span className="ml-1">({selectedLogDetail.llmUsage.toolCallCount} 个)</span>
                                                            )}
                                </span>
                                                    </div>
                                                )}
                                            </div>
                                            {/* 工具调用详情 */}
                                            {selectedLogDetail.llmUsage.hasToolCalls && selectedLogDetail.llmUsage.toolCalls && selectedLogDetail.llmUsage.toolCalls.length > 0 && (
                                                <div className="mt-3 pt-3 border-t border-purple-200 dark:border-purple-700">
                                                    <h5 className="text-xs font-semibold text-purple-900 dark:text-purple-100 mb-2">
                                                        工具调用详情
                                                    </h5>
                                                    <div className="space-y-2">
                                                        {selectedLogDetail.llmUsage.toolCalls.map((toolCall: ToolCallInfo, idx: number) => (
                                                            <div key={idx} className="p-2 bg-purple-100 dark:bg-purple-800/30 rounded text-xs">
                                                                <div className="font-medium text-purple-900 dark:text-purple-100 mb-1">
                                                                    {idx + 1}. {toolCall.name}
                                                                </div>
                                                                {toolCall.id && (
                                                                    <div className="text-purple-600 dark:text-purple-400 mb-1">
                                                                        ID: {toolCall.id}
                                                                    </div>
                                                                )}
                                                                {toolCall.arguments && (
                                                                    <div className="text-purple-700 dark:text-purple-300">
                                                                        <span className="font-medium">参数:</span>
                                                                        <pre className="mt-1 p-1 bg-white dark:bg-purple-900/50 rounded text-xs overflow-x-auto">
                                          {toolCall.arguments.length > 200
                                              ? `${toolCall.arguments.substring(0, 200)}...`
                                              : toolCall.arguments}
                                        </pre>
                                                                    </div>
                                                                )}
                                                            </div>
                                                        ))}
                                                    </div>
                                                </div>
                                            )}
                                        </div>
                                    )}

                                    {!selectedLogDetail.userMessage && !selectedLogDetail.agentMessage && selectedLogDetail.content && (
                                        <div className="text-sm text-gray-800 dark:text-gray-100 whitespace-pre-wrap">
                                            {selectedLogDetail.content}
                                        </div>
                                    )}
                                </>
                            )}
                        </div>
                    </div>
                </div>
            )}
        </div>
    )
}

export default VoiceAssistant