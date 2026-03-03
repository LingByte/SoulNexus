import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { Mic, Users, Shield, Save, Plus, Trash2, User, Volume2, Bot, AlertCircle, Play, TestTube, Edit2 } from 'lucide-react'
import Card, { CardContent, CardHeader, CardTitle } from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Modal, { ModalContent, ModalHeader, ModalTitle } from '@/components/UI/Modal'
import VoiceprintDeleteConfirm from '@/components/Voice/VoiceprintDeleteConfirm'
import { showAlert } from '@/utils/notification'
import { getSystemInit, SystemInitInfo } from '@/api/system'
import { getAssistantList, AssistantListItem } from '@/api/assistant'
import EmptyState from '@/components/UI/EmptyState'
import AudioFileUpload from '@/components/UI/AudioFileUpload'
import { useI18nStore } from '@/stores/i18nStore'
import {
  getVoiceprints,
  registerVoiceprint,
  deleteVoiceprint,
  identifyVoiceprint,
  verifyVoiceprint,
  updateVoiceprint,
  VoiceprintRecord,
  VoiceprintIdentifyResponse,
  VoiceprintVerifyResponse
} from '@/api/voiceprint'

const VoiceprintManagement = () => {
  const { t } = useI18nStore()
  const [loading, setLoading] = useState(false)
  const [assistants, setAssistants] = useState<AssistantListItem[]>([])
  const [selectedAssistantId, setSelectedAssistantId] = useState<string | null>(null)
  const [systemInfo, setSystemInfo] = useState<SystemInitInfo | null>(null)
  const [voiceprints, setVoiceprints] = useState<VoiceprintRecord[]>([])
  const [showAddModal, setShowAddModal] = useState(false)
  const [showTestModal, setShowTestModal] = useState(false)
  const [showVerifyModal, setShowVerifyModal] = useState(false)
  const [showEditModal, setShowEditModal] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [selectedVoiceprint, setSelectedVoiceprint] = useState<VoiceprintRecord | null>(null)
  const [voiceprintToDelete, setVoiceprintToDelete] = useState<VoiceprintRecord | null>(null)
  const [newSpeaker, setNewSpeaker] = useState({ name: '', description: '', audioFile: null as File | null })
  const [editingSpeaker, setEditingSpeaker] = useState({ name: '', description: '' })
  const [isEditingVoiceprint, setIsEditingVoiceprint] = useState(false)
  const [testAudioFile, setTestAudioFile] = useState<File | null>(null)
  const [verifyAudioFile, setVerifyAudioFile] = useState<File | null>(null)
  const [testResult, setTestResult] = useState<VoiceprintIdentifyResponse | null>(null)
  const [verifyResult, setVerifyResult] = useState<VoiceprintVerifyResponse | null>(null)
  const [isTestingVoiceprint, setIsTestingVoiceprint] = useState(false)
  const [isVerifyingVoiceprint, setIsVerifyingVoiceprint] = useState(false)
  const [isDeletingVoiceprint, setIsDeletingVoiceprint] = useState(false)

  // 获取助手列表
  useEffect(() => {
    const fetchAssistants = async () => {
      try {
        const res = await getAssistantList()
        if (res.code === 200) {
          setAssistants(res.data)
          if (res.data.length > 0 && !selectedAssistantId) {
            setSelectedAssistantId(String(res.data[0].id))
          }
        } else {
          showAlert('无法获取助手列表', 'error', '获取失败')
        }
      } catch (err: any) {
        showAlert(err?.msg || err?.message || '无法获取助手列表', 'error', '获取失败')
      }
    }
    fetchAssistants()
  }, [])

  // 加载系统配置和声纹数据
  useEffect(() => {
    loadSystemInfo()
  }, [])

  useEffect(() => {
    if (selectedAssistantId) {
      loadVoiceprints()
    }
  }, [selectedAssistantId])

  const loadSystemInfo = async () => {
    try {
      const response = await getSystemInit()
      if (response.code === 200) {
        setSystemInfo(response.data)
      }
    } catch (error) {
      console.error('Failed to load system info:', error)
      showAlert('无法获取系统配置信息', 'error', '加载失败')
    }
  }

  const loadVoiceprints = async () => {
    if (!selectedAssistantId) return
    
    setLoading(true)
    try {
      const response = await getVoiceprints(selectedAssistantId)
      if (response.code === 200) {
        setVoiceprints(response.data.voiceprints)
      } else {
        showAlert(response.msg || '无法获取声纹记录', 'error', '加载失败')
      }
    } catch (error: any) {
      console.error('Failed to load voiceprints:', error)
      showAlert(error?.msg || error?.message || '无法获取声纹记录', 'error', '加载失败')
    } finally {
      setLoading(false)
    }
  }

  const handleAddVoiceprint = async () => {
    if (!newSpeaker.name || !newSpeaker.audioFile) {
      showAlert('请填写姓名并上传音频文件', 'warning', '参数错误')
      return
    }

    if (!selectedAssistantId) {
      showAlert('请先选择助手', 'warning', '参数错误')
      return
    }

    setLoading(true)
    try {
      const response = await registerVoiceprint(
        selectedAssistantId,
        newSpeaker.name,
        newSpeaker.audioFile,
        newSpeaker.description
      )

      if (response.code === 200) {
        const selectedAssistant = assistants.find(a => String(a.id) === selectedAssistantId)
        showAlert(`${t('voiceprint.register.success')} ${selectedAssistant?.name}`, 'success', t('voiceprint.register.success'))
        setShowAddModal(false)
        setNewSpeaker({ name: '', description: '', audioFile: null })
        await loadVoiceprints()
      } else {
        throw new Error(response.msg || t('voiceprint.register.failed'))
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || t('voiceprint.register.failed'), 'error', t('voiceprint.register.failed'))
    } finally {
      setLoading(false)
    }
  }

  const handleDeleteVoiceprint = async (voiceprint: VoiceprintRecord) => {
    setVoiceprintToDelete(voiceprint)
    setShowDeleteConfirm(true)
  }

  const handleEditVoiceprint = (voiceprint: VoiceprintRecord) => {
    setSelectedVoiceprint(voiceprint)
    setEditingSpeaker({
      name: voiceprint.speaker_name,
      description: voiceprint.description || ''
    })
    setShowEditModal(true)
  }

  const handleSaveEdit = async () => {
    if (!selectedVoiceprint || !editingSpeaker.name) {
      showAlert('请填写姓名', 'warning', '参数错误')
      return
    }

    setIsEditingVoiceprint(true)
    try {
      const response = await updateVoiceprint(selectedVoiceprint.id, {
        speaker_name: editingSpeaker.name,
        description: editingSpeaker.description
      })

      if (response.code === 200) {
        showAlert(`${t('voiceprint.update.success')} ${editingSpeaker.name}`, 'success', t('voiceprint.update.success'))
        setShowEditModal(false)
        setSelectedVoiceprint(null)
        setEditingSpeaker({ name: '', description: '' })
        await loadVoiceprints()
      } else {
        throw new Error(response.msg || t('voiceprint.update.failed'))
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || t('voiceprint.update.failed'), 'error', t('voiceprint.update.failed'))
    } finally {
      setIsEditingVoiceprint(false)
    }
  }

  const confirmDeleteVoiceprint = async () => {
    if (!voiceprintToDelete) return

    setIsDeletingVoiceprint(true)
    try {
      const response = await deleteVoiceprint(voiceprintToDelete.id)
      if (response.code === 200) {
        showAlert(`${t('voiceprint.delete.success')} ${voiceprintToDelete.speaker_name}`, 'success', t('voiceprint.delete.success'))
        await loadVoiceprints()
        setShowDeleteConfirm(false)
        setVoiceprintToDelete(null)
      } else {
        throw new Error(response.msg || t('voiceprint.delete.failed'))
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || t('voiceprint.delete.failed'), 'error', t('voiceprint.delete.failed'))
    } finally {
      setIsDeletingVoiceprint(false)
    }
  }

  // 测试声纹识别（识别所有声纹）
  const handleTestVoiceprint = async () => {
    if (!testAudioFile || !selectedAssistantId) {
      showAlert('请选择音频文件', 'warning', '参数错误')
      return
    }

    setIsTestingVoiceprint(true)
    try {
      const response = await identifyVoiceprint(selectedAssistantId, testAudioFile)
      if (response.code === 200) {
        setTestResult(response.data)
        if (response.data.is_match) {
          const matchedVoiceprint = voiceprints.find(v => v.speaker_id === response.data.speaker_id)
          showAlert(`识别成功！匹配到：${matchedVoiceprint?.speaker_name || response.data.speaker_id}`, 'success', '识别结果')
        } else {
          showAlert('未找到匹配的声纹', 'info', '识别结果')
        }
      } else {
        throw new Error(response.msg || '识别失败')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '声纹识别失败', 'error', '识别失败')
      setTestResult(null)
    } finally {
      setIsTestingVoiceprint(false)
    }
  }

  // 验证特定声纹
  const handleVerifyVoiceprint = async () => {
    if (!verifyAudioFile || !selectedVoiceprint || !selectedAssistantId) {
      showAlert('请选择音频文件和要验证的声纹', 'warning', '参数错误')
      return
    }

    setIsVerifyingVoiceprint(true)
    try {
      const response = await verifyVoiceprint(selectedAssistantId, selectedVoiceprint.speaker_id, verifyAudioFile)
      if (response.code === 200) {
        setVerifyResult(response.data)
        if (response.data.verification_passed) {
          showAlert(`验证成功！确认是 ${selectedVoiceprint.speaker_name}`, 'success', '验证结果')
        } else if (response.data.is_match && !response.data.is_target_speaker) {
          const matchedVoiceprint = voiceprints.find(v => v.speaker_id === response.data.identified_speaker_id)
          showAlert(`验证失败！识别为其他人：${matchedVoiceprint?.speaker_name || response.data.identified_speaker_id}`, 'warning', '验证结果')
        } else {
          showAlert(`验证失败！未能识别为 ${selectedVoiceprint.speaker_name}`, 'error', '验证结果')
        }
      } else {
        throw new Error(response.msg || '验证失败')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '声纹验证失败', 'error', '验证失败')
      setVerifyResult(null)
    } finally {
      setIsVerifyingVoiceprint(false)
    }
  }

  const getConfidenceColor = (confidence: string) => {
    switch (confidence) {
      case 'very_high': return 'text-green-600 dark:text-green-400'
      case 'high': return 'text-blue-600 dark:text-blue-400'
      case 'medium': return 'text-yellow-600 dark:text-yellow-400'
      case 'low': return 'text-orange-600 dark:text-orange-400'
      case 'very_low': return 'text-red-600 dark:text-red-400'
      default: return 'text-gray-600 dark:text-gray-400'
    }
  }

  const getConfidenceText = (confidence: string) => {
    switch (confidence) {
      case 'very_high': return '非常高'
      case 'high': return '高'
      case 'medium': return '中等'
      case 'low': return '低'
      case 'very_low': return '非常低'
      default: return '未知'
    }
  }

  const selectedAssistant = assistants.find(a => String(a.id) === selectedAssistantId)

  return (
    <div className="container mx-auto px-4 py-8 max-w-6xl">
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className="space-y-6"
      >
        {/* 页面标题 */}
        <div className="flex items-center gap-3 mb-8">
          <div className="p-2 bg-purple-100 dark:bg-purple-900/20 rounded-lg">
            <Mic className="w-6 h-6 text-purple-600 dark:text-purple-400" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
              {t('voiceprint.title')}
            </h1>
            <p className="text-gray-600 dark:text-gray-400">
              {t('voiceprint.subtitle')}
            </p>
          </div>
        </div>

        {/* 助手选择器 - 使用按钮组样式 */}
        {assistants.length > 0 ? (
          <div className="w-full">
            <div className="flex flex-wrap gap-2 mb-6">
              {assistants.map(assistant => (
                <Button
                  key={assistant.id}
                  variant={selectedAssistantId === String(assistant.id) ? 'primary' : 'outline'}
                  size="md"
                  onClick={() => setSelectedAssistantId(String(assistant.id))}
                  leftIcon={<Bot className="w-4 h-4" />}
                  className="flex-shrink-0"
                >
                  {assistant.name}
                </Button>
              ))}
            </div>
            
            {assistants.map(assistant => (
              selectedAssistantId === String(assistant.id) && (
                <div key={assistant.id} className="space-y-6">
                  {/* 状态卡片 */}
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <Card>
                      <CardContent className="p-4">
                        <div className="flex items-center gap-3">
                          <div className={`p-2 rounded-lg ${
                            systemInfo?.voiceprint?.configured 
                              ? 'bg-green-100 dark:bg-green-900/20' 
                              : 'bg-red-100 dark:bg-red-900/20'
                          }`}>
                            <Shield className={`w-5 h-5 ${
                              systemInfo?.voiceprint?.configured
                                ? 'text-green-600 dark:text-green-400'
                                : 'text-red-600 dark:text-red-400'
                            }`} />
                          </div>
                          <div>
                            <p className="text-sm text-gray-600 dark:text-gray-400">服务状态</p>
                            <p className="font-semibold">
                              {systemInfo?.voiceprint?.configured ? '已配置' : '未配置'}
                            </p>
                          </div>
                        </div>
                      </CardContent>
                    </Card>

                    <Card>
                      <CardContent className="p-4">
                        <div className="flex items-center gap-3">
                          <div className="p-2 bg-blue-100 dark:bg-blue-900/20 rounded-lg">
                            <Users className="w-5 h-5 text-blue-600 dark:text-blue-400" />
                          </div>
                          <div>
                            <p className="text-sm text-gray-600 dark:text-gray-400">注册用户</p>
                            <p className="font-semibold">{voiceprints.length}</p>
                          </div>
                        </div>
                      </CardContent>
                    </Card>

                    <Card>
                      <CardContent className="p-4">
                        <div className="flex items-center gap-3">
                          <div className="p-2 bg-orange-100 dark:bg-orange-900/20 rounded-lg">
                            <Volume2 className="w-5 h-5 text-orange-600 dark:text-orange-400" />
                          </div>
                          <div>
                            <p className="text-sm text-gray-600 dark:text-gray-400">当前助手</p>
                            <p className="font-semibold">{assistant.name}</p>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  </div>

                  {/* 操作按钮 */}
                  <div className="flex justify-between items-center">
                    <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                      声纹记录 - {assistant.name}
                    </h2>
                    <div className="flex gap-3">
                      <Button
                        variant="outline"
                        onClick={() => setShowTestModal(true)}
                        leftIcon={<TestTube className="w-4 h-4" />}
                        disabled={!systemInfo?.voiceprint?.enabled || voiceprints.length === 0}
                      >
                        测试识别
                      </Button>
                      <Button
                        variant="primary"
                        onClick={() => setShowAddModal(true)}
                        leftIcon={<Plus className="w-4 h-4" />}
                        disabled={!systemInfo?.voiceprint?.enabled}
                      >
                        添加声纹
                      </Button>
                    </div>
                  </div>

                  {/* 声纹记录列表 */}
                  <Card>
                    <CardContent className="p-6">
                      {loading ? (
                        <div className="flex items-center justify-center py-8">
                          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-purple-600"></div>
                        </div>
                      ) : voiceprints.length === 0 ? (
                        <EmptyState
                          icon={Mic}
                          title="暂无声纹记录"
                          description={`还没有为助手 ${assistant.name} 注册任何声纹`}
                          iconClassName="text-purple-400 dark:text-purple-500"
                          action={systemInfo?.voiceprint?.enabled ? {
                            label: '添加声纹',
                            onClick: () => setShowAddModal(true)
                          } : undefined}
                        />
                      ) : (
                        <div className="space-y-4">
                          {voiceprints.map((voiceprint) => (
                            <div
                              key={voiceprint.id}
                              className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-800/50 rounded-lg"
                            >
                              <div className="flex items-center gap-4 flex-1">
                                <div className="p-2 bg-purple-100 dark:bg-purple-900/20 rounded-lg">
                                  <User className="w-5 h-5 text-purple-600 dark:text-purple-400" />
                                </div>
                                <div className="flex-1">
                                  <h3 className="font-medium text-gray-900 dark:text-white">
                                    {voiceprint.speaker_name}
                                  </h3>
                                  <p className="text-sm text-gray-600 dark:text-gray-400">
                                    ID: {voiceprint.speaker_id} • 注册时间: {new Date(voiceprint.created_at).toLocaleString()}
                                  </p>
                                  {voiceprint.description && (
                                    <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                                      描述: {voiceprint.description}
                                    </p>
                                  )}
                                  {voiceprint.last_used && (
                                    <p className="text-xs text-gray-500 dark:text-gray-500">
                                      最后使用: {new Date(voiceprint.last_used).toLocaleString()}
                                    </p>
                                  )}
                                </div>
                              </div>
                              <div className="flex items-center gap-4">
                                {voiceprint.confidence && (
                                  <div className="text-right">
                                    <p className="text-sm font-medium text-gray-900 dark:text-white">
                                      置信度: {(voiceprint.confidence * 100).toFixed(1)}%
                                    </p>
                                    <div className="w-20 bg-gray-200 dark:bg-gray-700 rounded-full h-2 mt-1">
                                      <div
                                        className="bg-purple-600 h-2 rounded-full"
                                        style={{ width: `${voiceprint.confidence * 100}%` }}
                                      />
                                    </div>
                                  </div>
                                )}
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => handleEditVoiceprint(voiceprint)}
                                  className="text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20"
                                  title="编辑声纹"
                                >
                                  <Edit2 className="w-4 h-4" />
                                </Button>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => {
                                    setSelectedVoiceprint(voiceprint)
                                    setShowVerifyModal(true)
                                  }}
                                  className="text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20"
                                  title="验证此声纹"
                                >
                                  <Play className="w-4 h-4" />
                                </Button>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => handleDeleteVoiceprint(voiceprint)}
                                  className="text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"
                                >
                                  <Trash2 className="w-4 h-4" />
                                </Button>
                              </div>
                            </div>
                          ))}
                        </div>
                      )}
                    </CardContent>
                  </Card>
                </div>
              )
            ))}
          </div>
        ) : (
          <EmptyState
            icon={Bot}
            title="暂无助手"
            description="请先创建助手后再进行声纹识别管理"
            iconClassName="text-gray-400"
          />
        )}

        {/* 使用说明 */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <AlertCircle className="w-5 h-5" />
              {t('voiceprint.usage.title')}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="bg-blue-50 dark:bg-blue-900/20 p-4 rounded-lg">
              <h4 className="font-medium text-blue-900 dark:text-blue-100 mb-2">
                {t('voiceprint.usage.feature.title')}
              </h4>
              <ul className="text-sm text-blue-800 dark:text-blue-200 space-y-1">
                <li>{t('voiceprint.usage.feature.item1')}</li>
                <li>{t('voiceprint.usage.feature.item2')}</li>
                <li>{t('voiceprint.usage.feature.item3')}</li>
                <li>{t('voiceprint.usage.feature.item4')}</li>
              </ul>
            </div>

            <div className="bg-yellow-50 dark:bg-yellow-900/20 p-4 rounded-lg">
              <h4 className="font-medium text-yellow-900 dark:text-yellow-100 mb-2">
                {t('voiceprint.usage.notes.title')}
              </h4>
              <ul className="text-sm text-yellow-800 dark:text-yellow-200 space-y-1">
                <li>{t('voiceprint.usage.notes.item1')}</li>
                <li>{t('voiceprint.usage.notes.item2')}</li>
                <li>{t('voiceprint.usage.notes.item3')}</li>
                <li>{t('voiceprint.usage.notes.item4')}</li>
              </ul>
            </div>
          </CardContent>
        </Card>
      </motion.div>

      {/* 添加声纹模态框 */}
      <Modal isOpen={showAddModal} onClose={() => setShowAddModal(false)}>
        <ModalContent className="max-w-2xl">
          <ModalHeader>
            <ModalTitle>添加声纹</ModalTitle>
          </ModalHeader>
          <div className="space-y-6 max-h-[70vh] overflow-y-auto">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                说话人姓名 <span className="text-red-500">*</span>
              </label>
              <Input
                type="text"
                value={newSpeaker.name}
                onChange={(e) => setNewSpeaker(prev => ({ ...prev, name: e.target.value }))}
                placeholder="输入说话人的姓名，如：张三"
                className="w-full"
              />
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                系统将自动为该说话人生成唯一标识符
              </p>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                描述信息
              </label>
              <textarea
                value={newSpeaker.description}
                onChange={(e) => setNewSpeaker(prev => ({ ...prev, description: e.target.value }))}
                placeholder="输入说话人的描述，如：身份、职业、成就等（可选）"
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-purple-500 dark:focus:ring-purple-400"
                rows={3}
              />
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                可以记录说话人的身份、职业、成就等信息，便于后续识别和管理
              </p>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
                音频文件 <span className="text-red-500">*</span>
              </label>
              <AudioFileUpload
                onFileSelect={(file) => setNewSpeaker(prev => ({ ...prev, audioFile: file }))}
                placeholder="选择WAV音频文件或拖拽到此处"
                maxSize={50}
                required
              />
              <div className="mt-3 p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                <h4 className="text-sm font-medium text-blue-900 dark:text-blue-100 mb-2">
                  {t('voiceTraining.audioRequirements')}
                </h4>
                <ul className="text-xs text-blue-800 dark:text-blue-200 space-y-1">
                  <li>{t('voiceTraining.audioReq.duration')}</li>
                  <li>{t('voiceTraining.audioReq.quiet')}</li>
                  <li>• 说话内容：清晰朗读一段文字</li>
                  <li>{t('voiceTraining.audioReq.sampleRate')}，{t('voiceTraining.audioReq.mono')}</li>
                </ul>
              </div>
            </div>

            <div className="bg-amber-50 dark:bg-amber-900/20 p-4 rounded-lg">
              <div className="flex items-start space-x-3">
                <Bot className="w-5 h-5 text-amber-600 dark:text-amber-400 mt-0.5 flex-shrink-0" />
                <div>
                  <p className="text-sm font-medium text-amber-900 dark:text-amber-100">
                    当前助手: {selectedAssistant?.name}
                  </p>
                  <p className="text-xs text-amber-700 dark:text-amber-300 mt-1">
                    声纹将注册到此助手下，仅在该助手的对话中生效
                  </p>
                </div>
              </div>
            </div>

            <div className="flex justify-end gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
              <Button
                variant="ghost"
                onClick={() => {
                  setShowAddModal(false)
                  setNewSpeaker({ name: '', description: '', audioFile: null })
                }}
              >
                取消
              </Button>
              <Button
                variant="primary"
                onClick={handleAddVoiceprint}
                loading={loading}
                leftIcon={<Save className="w-4 h-4" />}
                disabled={!newSpeaker.name || !newSpeaker.audioFile}
              >
                注册声纹
              </Button>
            </div>
          </div>
        </ModalContent>
      </Modal>

      {/* 测试识别模态框 */}
      <Modal isOpen={showTestModal} onClose={() => {
        setShowTestModal(false)
        setTestAudioFile(null)
        setTestResult(null)
      }}>
        <ModalContent className="max-w-3xl">
          <div className="space-y-6 max-h-[70vh] overflow-y-auto">
            <div className="bg-blue-50 dark:bg-blue-900/20 p-4 rounded-lg">
              <div className="flex items-start space-x-3">
                <Bot className="w-5 h-5 text-blue-600 dark:text-blue-400 mt-0.5 flex-shrink-0" />
                <div>
                  <p className="text-sm font-medium text-blue-900 dark:text-blue-100">
                    当前助手: {selectedAssistant?.name}
                  </p>
                  <p className="text-xs text-blue-700 dark:text-blue-300 mt-1">
                    将在该助手的 {voiceprints.length} 个声纹中进行识别
                  </p>
                </div>
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
                选择测试音频 <span className="text-red-500">*</span>
              </label>
              <AudioFileUpload
                onFileSelect={setTestAudioFile}
                placeholder="选择WAV音频文件进行声纹识别测试"
                maxSize={50}
                required
              />
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
                建议使用清晰、无噪音的音频文件，时长3-10秒
              </p>
            </div>

            <div className="flex justify-center">
              <Button
                variant="primary"
                onClick={handleTestVoiceprint}
                loading={isTestingVoiceprint}
                disabled={!testAudioFile}
                leftIcon={<Play className="w-4 h-4" />}
                className="px-8"
              >
                {isTestingVoiceprint ? '识别中...' : '开始识别'}
              </Button>
            </div>

            {testResult && (
              <div className="mt-6 p-4 bg-gray-50 dark:bg-gray-800/50 rounded-lg">
                <h4 className="font-medium text-gray-900 dark:text-white mb-3 flex items-center gap-2">
                  <Volume2 className="w-4 h-4" />
                  识别结果
                </h4>
                <div className="space-y-3">
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-600 dark:text-gray-400">识别的说话人:</span>
                    <span className="font-medium text-gray-900 dark:text-white">
                      {testResult.speaker_id ? (
                        voiceprints.find(v => v.speaker_id === testResult.speaker_id)?.speaker_name || testResult.speaker_id
                      ) : '未识别'}
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-600 dark:text-gray-400">相似度分数:</span>
                    <span className="font-medium text-gray-900 dark:text-white">
                      {(testResult.score * 100).toFixed(2)}%
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-600 dark:text-gray-400">置信度:</span>
                    <span className={`font-medium ${getConfidenceColor(testResult.confidence)}`}>
                      {getConfidenceText(testResult.confidence)}
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-600 dark:text-gray-400">匹配状态:</span>
                    <span className={`font-medium flex items-center gap-1 ${testResult.is_match ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'}`}>
                      {testResult.is_match ? (
                        <>
                          <Shield className="w-4 h-4" />
                          匹配成功
                        </>
                      ) : (
                        <>
                          <AlertCircle className="w-4 h-4" />
                          未匹配
                        </>
                      )}
                    </span>
                  </div>
                  
                  {/* 相似度进度条 */}
                  <div className="mt-4">
                    <div className="flex justify-between text-xs text-gray-500 dark:text-gray-400 mb-1">
                      <span>相似度</span>
                      <span>{(testResult.score * 100).toFixed(1)}%</span>
                    </div>
                    <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                      <div
                        className={`h-2 rounded-full transition-all duration-300 ${
                          testResult.score >= 0.8 ? 'bg-green-500' :
                          testResult.score >= 0.6 ? 'bg-blue-500' :
                          testResult.score >= 0.4 ? 'bg-yellow-500' :
                          testResult.score >= 0.2 ? 'bg-orange-500' : 'bg-red-500'
                        }`}
                        style={{ width: `${testResult.score * 100}%` }}
                      />
                    </div>
                  </div>
                </div>
              </div>
            )}
            <div className="flex justify-end gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
              <Button
                variant="ghost"
                onClick={() => {
                  setShowTestModal(false)
                  setTestAudioFile(null)
                  setTestResult(null)
                }}
              >
                关闭
              </Button>
            </div>
          </div>
        </ModalContent>
      </Modal>

      {/* 验证声纹模态框 */}
      <Modal isOpen={showVerifyModal} onClose={() => {
        setShowVerifyModal(false)
        setVerifyAudioFile(null)
        setVerifyResult(null)
        setSelectedVoiceprint(null)
      }}>
        <ModalContent className="max-w-3xl">
          <ModalHeader>
            <ModalTitle className="flex items-center gap-2">
              <Shield className="w-5 h-5" />
              验证声纹
            </ModalTitle>
          </ModalHeader>
          <div className="space-y-6 max-h-[70vh] overflow-y-auto">
            {selectedVoiceprint && (
              <div className="bg-purple-50 dark:bg-purple-900/20 p-4 rounded-lg">
                <div className="flex items-start space-x-3">
                  <User className="w-5 h-5 text-purple-600 dark:text-purple-400 mt-0.5 flex-shrink-0" />
                  <div>
                    <p className="text-sm font-medium text-purple-900 dark:text-purple-100">
                      验证目标: {selectedVoiceprint.speaker_name}
                    </p>
                    <p className="text-xs text-purple-700 dark:text-purple-300 mt-1">
                      ID: {selectedVoiceprint.speaker_id}
                    </p>
                    <p className="text-xs text-purple-700 dark:text-purple-300">
                      注册时间: {new Date(selectedVoiceprint.created_at).toLocaleString()}
                    </p>
                  </div>
                </div>
              </div>
            )}

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
                选择验证音频 <span className="text-red-500">*</span>
              </label>
              <AudioFileUpload
                onFileSelect={setVerifyAudioFile}
                placeholder="选择WAV音频文件验证是否为目标说话人"
                maxSize={50}
                required
              />
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
                上传音频将与目标声纹进行比对验证
              </p>
            </div>

            <div className="flex justify-center">
              <Button
                variant="primary"
                onClick={handleVerifyVoiceprint}
                loading={isVerifyingVoiceprint}
                disabled={!verifyAudioFile || !selectedVoiceprint}
                leftIcon={<Shield className="w-4 h-4" />}
                className="px-8"
              >
                {isVerifyingVoiceprint ? '验证中...' : '开始验证'}
              </Button>
            </div>

            {verifyResult && selectedVoiceprint && (
              <div className="mt-6 p-4 bg-gray-50 dark:bg-gray-800/50 rounded-lg">
                <h4 className="font-medium text-gray-900 dark:text-white mb-3 flex items-center gap-2">
                  <Shield className="w-4 h-4" />
                  验证结果
                </h4>
                <div className="space-y-3">
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-600 dark:text-gray-400">目标说话人:</span>
                    <span className="font-medium text-gray-900 dark:text-white">
                      {selectedVoiceprint.speaker_name}
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-600 dark:text-gray-400">识别结果:</span>
                    <span className="font-medium text-gray-900 dark:text-white">
                      {verifyResult.identified_speaker_id ? (
                        voiceprints.find(v => v.speaker_id === verifyResult.identified_speaker_id)?.speaker_name || verifyResult.identified_speaker_id
                      ) : '未识别'}
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-600 dark:text-gray-400">相似度分数:</span>
                    <span className="font-medium text-gray-900 dark:text-white">
                      {(verifyResult.score * 100).toFixed(2)}%
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-600 dark:text-gray-400">置信度:</span>
                    <span className={`font-medium ${getConfidenceColor(verifyResult.confidence)}`}>
                      {getConfidenceText(verifyResult.confidence)}
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-600 dark:text-gray-400">验证状态:</span>
                    <span className={`font-medium flex items-center gap-1 ${
                      verifyResult.verification_passed
                        ? 'text-green-600 dark:text-green-400'
                        : 'text-red-600 dark:text-red-400'
                    }`}>
                      {verifyResult.verification_passed ? (
                        <>
                          <Shield className="w-4 h-4" />
                          验证通过
                        </>
                      ) : (
                        <>
                          <AlertCircle className="w-4 h-4" />
                          验证失败
                        </>
                      )}
                    </span>
                  </div>
                  
                  {/* 相似度进度条 */}
                  <div className="mt-4">
                    <div className="flex justify-between text-xs text-gray-500 dark:text-gray-400 mb-1">
                      <span>相似度</span>
                      <span>{(verifyResult.score * 100).toFixed(1)}%</span>
                    </div>
                    <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                      <div
                        className={`h-2 rounded-full transition-all duration-300 ${
                          verifyResult.verification_passed
                            ? 'bg-green-500'
                            : verifyResult.score >= 0.6
                            ? 'bg-yellow-500'
                            : 'bg-red-500'
                        }`}
                        style={{ width: `${verifyResult.score * 100}%` }}
                      />
                    </div>
                  </div>

                  {/* 验证说明 */}
                  <div className={`mt-4 p-3 rounded-lg ${
                    verifyResult.verification_passed
                      ? 'bg-green-50 dark:bg-green-900/20'
                      : 'bg-red-50 dark:bg-red-900/20'
                  }`}>
                    <p className={`text-sm ${
                      verifyResult.verification_passed
                        ? 'text-green-800 dark:text-green-200'
                        : 'text-red-800 dark:text-red-200'
                    }`}>
                      {verifyResult.verification_passed
                        ? `✓ 验证成功！音频确实来自 ${selectedVoiceprint.speaker_name}`
                        : verifyResult.is_match && !verifyResult.is_target_speaker
                        ? `✗ 验证失败！音频来自其他说话人：${voiceprints.find(v => v.speaker_id === verifyResult.identified_speaker_id)?.speaker_name || '未知'}`
                        : `✗ 验证失败！音频不匹配任何已注册的声纹或置信度不足`
                      }
                    </p>
                  </div>
                </div>
              </div>
            )}

            <div className="text-xs text-gray-500 dark:text-gray-400 space-y-1">
              <p>• 验证功能用于确认音频是否来自特定的说话人</p>
              <p>• 验证通过需要同时满足：识别为目标说话人且置信度足够高</p>
              <p>• 建议使用与注册时相似的音频质量进行验证</p>
            </div>

            <div className="flex justify-end gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
              <Button
                variant="ghost"
                onClick={() => {
                  setShowVerifyModal(false)
                  setVerifyAudioFile(null)
                  setVerifyResult(null)
                  setSelectedVoiceprint(null)
                }}
              >
                关闭
              </Button>
            </div>
          </div>
        </ModalContent>
      </Modal>

      {/* 编辑声纹模态框 */}
      <Modal isOpen={showEditModal} onClose={() => {
        setShowEditModal(false)
        setSelectedVoiceprint(null)
        setEditingSpeaker({ name: '', description: '' })
      }}>
        <ModalContent className="max-w-2xl">
          <ModalHeader>
            <ModalTitle className="flex items-center gap-2">
              <Edit2 className="w-5 h-5" />
              编辑声纹
            </ModalTitle>
          </ModalHeader>
          <div className="space-y-6">
            {selectedVoiceprint && (
              <>
                <div className="bg-purple-50 dark:bg-purple-900/20 p-4 rounded-lg">
                  <div className="flex items-start space-x-3">
                    <User className="w-5 h-5 text-purple-600 dark:text-purple-400 mt-0.5 flex-shrink-0" />
                    <div>
                      <p className="text-sm font-medium text-purple-900 dark:text-purple-100">
                        声纹信息
                      </p>
                      <p className="text-xs text-purple-700 dark:text-purple-300 mt-1">
                        ID: {selectedVoiceprint.speaker_id}
                      </p>
                      <p className="text-xs text-purple-700 dark:text-purple-300">
                        注册时间: {new Date(selectedVoiceprint.created_at).toLocaleString()}
                      </p>
                    </div>
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    说话人姓名 <span className="text-red-500">*</span>
                  </label>
                  <Input
                    type="text"
                    value={editingSpeaker.name}
                    onChange={(e) => setEditingSpeaker(prev => ({ ...prev, name: e.target.value }))}
                    placeholder="输入说话人的姓名"
                    className="w-full"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    描述信息
                  </label>
                  <textarea
                    value={editingSpeaker.description}
                    onChange={(e) => setEditingSpeaker(prev => ({ ...prev, description: e.target.value }))}
                    placeholder="输入说话人的描述，如：身份、职业、成就等"
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-purple-500 dark:focus:ring-purple-400"
                    rows={4}
                  />
                </div>

                <div className="flex justify-end gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
                  <Button
                    variant="ghost"
                    onClick={() => {
                      setShowEditModal(false)
                      setSelectedVoiceprint(null)
                      setEditingSpeaker({ name: '', description: '' })
                    }}
                  >
                    取消
                  </Button>
                  <Button
                    variant="primary"
                    onClick={handleSaveEdit}
                    loading={isEditingVoiceprint}
                    leftIcon={<Save className="w-4 h-4" />}
                    disabled={!editingSpeaker.name}
                  >
                    保存修改
                  </Button>
                </div>
              </>
            )}
          </div>
        </ModalContent>
      </Modal>

      {/* 删除确认对话框 */}
      {voiceprintToDelete && (
        <VoiceprintDeleteConfirm
          isOpen={showDeleteConfirm}
          onClose={() => {
            setShowDeleteConfirm(false)
            setVoiceprintToDelete(null)
          }}
          onConfirm={confirmDeleteVoiceprint}
          voiceprintName={voiceprintToDelete.speaker_name}
          speakerId={voiceprintToDelete.speaker_id}
          createdAt={voiceprintToDelete.created_at}
          loading={isDeletingVoiceprint}
        />
      )}
    </div>
  )
}

export default VoiceprintManagement