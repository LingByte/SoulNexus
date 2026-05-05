import { useState, useEffect, useMemo } from 'react'
import { 
  Key, Plus, Trash2, Download, 
  Settings, CheckCircle,
  Brain, Globe, Lock
} from 'lucide-react'
import { useAuthStore } from '../stores/authStore'
import { useI18nStore } from '../stores/i18nStore'
import Button from '../components/UI/Button'
import Input from '../components/UI/Input'
import AutocompleteInput from '../components/UI/AutocompleteInput'
import Card from '../components/UI/Card'
import Badge from '../components/UI/Badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/UI/Tabs'
import ConfirmDialog from '../components/UI/ConfirmDialog'
import { showAlert } from '../utils/notification'
import {
  createCredential,
  fetchUserCredentials,
  deleteCredential,
  type Credential,
  type CreateCredentialForm
} from '../api/credential'
import { motion, AnimatePresence } from 'framer-motion'
import ProviderConfigForm from '../components/Credential/ProviderConfigForm'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/UI/Select'
import { 
  getTTSProviderConfig, 
  getASRProviderConfig,
  getTTSProviderOptions,
  getASRProviderOptions
} from '../config/providerConfig'
import {
  LLM_PROVIDER_SUGGESTIONS,
  getDefaultApiUrl,
  isCozeProvider,
  isOllamaProvider,
  isLMStudioProvider,
} from '../config/llmProviderConfig'

const DATE_NEVER = '1970-01-01 07:59:59'

const formatDateTimeInput = (value?: string): string => {
  if (!value) return ''
  const dt = new Date(value)
  if (Number.isNaN(dt.getTime())) return value
  const y = dt.getFullYear()
  const m = `${dt.getMonth() + 1}`.padStart(2, '0')
  const d = `${dt.getDate()}`.padStart(2, '0')
  const h = `${dt.getHours()}`.padStart(2, '0')
  const mi = `${dt.getMinutes()}`.padStart(2, '0')
  const s = `${dt.getSeconds()}`.padStart(2, '0')
  return `${y}-${m}-${d} ${h}:${mi}:${s}`
}

const addDays = (days: number): string => {
  const now = new Date()
  now.setDate(now.getDate() + days)
  return formatDateTimeInput(now.toISOString())
}

const CredentialManager = () => {
  const { t } = useI18nStore()
  const { isAuthenticated } = useAuthStore()
  const [credentials, setCredentials] = useState<Credential[]>([])
  const [isPageLoading, setIsPageLoading] = useState(true)
  const [isCreating, setIsCreating] = useState(false)
  const [activeTab, setActiveTab] = useState('list')
  const [generatedKey, setGeneratedKey] = useState<{ name: string; apiKey: string; apiSecret: string } | null>(null)
  const [form, setForm] = useState<CreateCredentialForm>({
    name: "",
    llmProvider: "",
    llmApiKey: "",
    llmApiUrl: "",
    expiresAt: "",
    tokenQuota: 0,
    requestQuota: 0,
    useNativeQuota: false,
    unlimitedQuota: true,
  })
  
  // Coze 专用配置字段
  const [cozeConfig, setCozeConfig] = useState({
    userId: "",    // 可选的 User ID
    baseUrl: "",   // 可选的 Base URL
  })
  
  // 动态配置字段的值
  const [asrConfigFields, setAsrConfigFields] = useState<Record<string, any>>({})
  const [ttsConfigFields, setTtsConfigFields] = useState<Record<string, any>>({})
  const [asrProvider, setAsrProvider] = useState('')
  const [ttsProvider, setTtsProvider] = useState('')
  
  // 删除确认对话框状态
  const [deleteConfirm, setDeleteConfirm] = useState<{ isOpen: boolean; credentialId: number | null; credentialName: string; isDeleting: boolean }>({
    isOpen: false,
    credentialId: null,
    credentialName: '',
    isDeleting: false
  })

  const credentialStats = useMemo(
    () => ({
      total: credentials.length,
      llm: credentials.filter((c) => c.llmProvider).length,
      asr: credentials.filter((c) => c.asrProvider).length,
      tts: credentials.filter((c) => c.ttsProvider).length,
    }),
    [credentials]
  )

  // 页面加载时获取密钥列表
  useEffect(() => {
    if (!isAuthenticated) {
      setIsPageLoading(false)
      return
    }

    const fetchCredentials = async () => {
      try {
        setIsPageLoading(true)
        const response = await fetchUserCredentials()
        if (response.code === 200) {
          setCredentials(response.data)
        } else {
          throw new Error(response.msg || t('credential.messages.fetchFailed'))
        }
      } catch (error: any) {
        // 处理API错误响应
        const errorMessage = error?.msg || error?.message || t('credential.messages.fetchFailed')
        showAlert(errorMessage, 'error', t('credential.messages.loadFailed'))
      } finally {
        setIsPageLoading(false)
      }
    }

    fetchCredentials()
  }, [isAuthenticated])

  // 构建配置对象
  const buildConfig = (provider: string, fields: Record<string, any>): { provider: string; [key: string]: any } | undefined => {
    if (!provider) return undefined
    
    const config: { provider: string; [key: string]: any } = {
      provider: provider
    }
    
    // 将字段添加到配置中，移除前缀
    Object.keys(fields).forEach(key => {
      const value = fields[key]
      if (value !== undefined && value !== null && value !== '') {
        // 移除前缀（如 asr_ 或 tts_）
        const configKey = key.replace(/^(asr|tts)_/, '')
        // 如果移除前缀后还有值，添加到配置中
        if (configKey) {
          config[configKey] = value
        }
      }
    })
    
    return Object.keys(config).length > 1 ? config : undefined // 至少要有 provider
  }

  const handleCreate = async () => {
    if (!form.name.trim()) {
      showAlert(t('credential.messages.enterName'), 'error', t('credential.messages.validationFailed'))
      return
    }

    setIsCreating(true)
    try {
      // 构建新格式的配置
      const asrConfig = buildConfig(asrProvider, asrConfigFields)
      const ttsConfig = buildConfig(ttsProvider, ttsConfigFields)
      
      // 处理 Coze 配置：如果有可选参数，组合成 JSON 格式
      let llmApiUrl = form.llmApiUrl
      if (isCozeProvider(form.llmProvider)) {
        const hasOptionalConfig = cozeConfig.userId || cozeConfig.baseUrl
        if (hasOptionalConfig) {
          // 如果有可选配置，组合成 JSON
          const cozeJsonConfig: any = {
            botId: form.llmApiUrl, // Bot ID 是必需的
          }
          if (cozeConfig.userId) {
            cozeJsonConfig.userId = cozeConfig.userId
          }
          if (cozeConfig.baseUrl) {
            cozeJsonConfig.baseUrl = cozeConfig.baseUrl
          }
          llmApiUrl = JSON.stringify(cozeJsonConfig)
        }
        // 如果没有可选配置，llmApiUrl 直接存储 Bot ID（简单格式）
      }
      
      const submitForm: CreateCredentialForm = {
        ...form,
        llmApiUrl, // 使用处理后的 llmApiUrl
        asrConfig,
        ttsConfig,
      }
      
      const response = await createCredential(submitForm)
      if (response.code === 200) {
        setGeneratedKey({
          name: response.data.name,
          apiKey: response.data.apiKey,
          apiSecret: response.data.apiSecret,
        })
        showAlert(t('credential.messages.createSuccess'), 'success', t('credential.messages.createSuccess'))
        
        // 重新获取列表
        try {
          const listResponse = await fetchUserCredentials()
          if (listResponse.code === 200) {
            setCredentials(listResponse.data)
          }
        } catch (error: any) {
          console.error('Failed to refresh credentials list:', error)
        }
        
        // 重置表单
        setForm({
          name: "",
          llmProvider: "",
          llmApiKey: "",
          llmApiUrl: "",
          expiresAt: "",
          tokenQuota: 0,
          requestQuota: 0,
          useNativeQuota: false,
          unlimitedQuota: true,
        })
        setCozeConfig({
          userId: "",
          baseUrl: "",
        })
        setAsrConfigFields({})
        setTtsConfigFields({})
        setAsrProvider('')
        setTtsProvider('')
      } else {
        throw new Error(response.msg || t('credential.messages.createFailed'))
      }
    } catch (error: any) {
      // 处理API错误响应
      const errorMessage = error?.msg || error?.message || t('credential.messages.createFailed')
      showAlert(errorMessage, 'error', t('credential.messages.operationFailed'))
    } finally {
      setIsCreating(false)
    }
  }

  const handleDelete = async (id: number) => {
    try {
      setDeleteConfirm(prev => ({ ...prev, isDeleting: true }))
      const response = await deleteCredential(id)
      if (response.code === 200) {
        setCredentials((prev) => prev.filter((c) => c.id !== id))
        showAlert(t('credential.messages.deleteSuccess'), 'success', t('credential.messages.deleteSuccess'))
        setDeleteConfirm({ isOpen: false, credentialId: null, credentialName: '', isDeleting: false })
      } else {
        throw new Error(response.msg || t('credential.messages.deleteFailed'))
      }
    } catch (error: any) {
      // 处理API错误响应
      const errorMessage = error?.msg || error?.message || t('credential.messages.deleteFailed')
      showAlert(errorMessage, 'error', t('credential.messages.operationFailed'))
      setDeleteConfirm(prev => ({ ...prev, isDeleting: false }))
    }
  }

  const openDeleteConfirm = (id: number, name: string) => {
    setDeleteConfirm({
      isOpen: true,
      credentialId: id,
      credentialName: name,
      isDeleting: false
    })
  }

  const closeDeleteConfirm = () => {
    setDeleteConfirm({ isOpen: false, credentialId: null, credentialName: '', isDeleting: false })
  }

  const confirmDelete = () => {
    if (deleteConfirm.credentialId !== null) {
      handleDelete(deleteConfirm.credentialId)
    }
  }

  const handleFormChange = (field: keyof CreateCredentialForm, value: string) => {
    setForm(prev => {
      const updated = { ...prev, [field]: value }
      
      // 当选择LLM Provider时，自动填充API URL（如果URL为空）
      // Coze 不需要自动填充 URL，其他 provider（包括 Ollama）会自动填充
      if (field === 'llmProvider' && value && !updated.llmApiUrl && !isCozeProvider(value)) {
        const defaultUrl = getDefaultApiUrl(value)
        if (defaultUrl) {
          updated.llmApiUrl = defaultUrl
        }
      }
      
      return updated
    })
  }

  const handleExport = () => {
    if (!generatedKey) return
    
    const blob = new Blob(
      [`Name: ${generatedKey.name}\nAPI Key: ${generatedKey.apiKey}\nAPI Secret: ${generatedKey.apiSecret}`],
      { type: "text/plain" }
    )
    const url = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = `${generatedKey.name || "credential"}.txt`
    a.click()
    URL.revokeObjectURL(url)
  }



  if (!isAuthenticated) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-2xl font-bold text-neutral-900 dark:text-neutral-100 mb-4">
            {t('credential.pleaseLogin')}
          </h1>
          <p className="text-neutral-600 dark:text-neutral-400">
            {t('credential.loginDesc')}
          </p>
        </div>
      </div>
    )
  }

  if (isPageLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto mb-4"></div>
          <h1 className="text-2xl font-bold text-neutral-900 dark:text-neutral-100 mb-4">
            {t('credential.loading')}
          </h1>
          <p className="text-neutral-600 dark:text-neutral-400">
            {t('credential.loadingDesc')}
          </p>
        </div>
      </div>
    )
  }

  // @ts-ignore
    return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* 头部操作栏 */}
        <div className="mb-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <Key className="w-3 h-3 mr-1" />
              <Badge variant="primary" className="text-xs">
                {t('credential.title')}
              </Badge>
              <div className="text-sm text-gray-500 dark:text-gray-400">
                {t('credential.totalCount', { count: credentials.length })}
              </div>
            </div>
            <div className="flex items-center space-x-2">
              <Button
                variant="primary"
                size="sm"
                leftIcon={<Plus className="w-4 h-4" />}
                onClick={() => setActiveTab('create')}
              >
                {t('credential.create')}
              </Button>
            </div>
          </div>
        </div>

        {/* 顶部：密钥统计（仅数字卡片，无标题区） */}
        <Card className="mb-6">
          <div className="p-5 md:p-6">
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
                <div className="rounded-xl border border-gray-200 dark:border-gray-700 bg-gray-50/80 dark:bg-gray-800/40 px-4 py-3">
                  <div className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">
                    {t('credential.totalKeys')}
                  </div>
                  <div className="text-2xl font-semibold font-mono tabular-nums text-gray-900 dark:text-white mt-1">
                    #{credentialStats.total}
                  </div>
                </div>
                <div className="rounded-xl border border-gray-200 dark:border-gray-700 bg-gray-50/80 dark:bg-gray-800/40 px-4 py-3">
                  <div className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">
                    {t('credential.llmKeys')}
                  </div>
                  <div className="text-2xl font-semibold tabular-nums text-gray-900 dark:text-white mt-1">
                    {credentialStats.llm}
                  </div>
                </div>
                <div className="rounded-xl border border-gray-200 dark:border-gray-700 bg-gray-50/80 dark:bg-gray-800/40 px-4 py-3">
                  <div className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">
                    {t('credential.asrKeys')}
                  </div>
                  <div className="text-2xl font-semibold tabular-nums text-gray-900 dark:text-white mt-1">
                    {credentialStats.asr}
                  </div>
                </div>
                <div className="rounded-xl border border-gray-200 dark:border-gray-700 bg-gray-50/80 dark:bg-gray-800/40 px-4 py-3">
                  <div className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">
                    {t('credential.ttsKeys')}
                  </div>
                  <div className="text-2xl font-semibold tabular-nums text-gray-900 dark:text-white mt-1">
                    {credentialStats.tts}
                  </div>
                </div>
            </div>
          </div>
        </Card>

        {/* 列表 / 创建 */}
        <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-0">
                <TabsList className="grid w-full grid-cols-2 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-1">
                  <TabsTrigger value="list" className="flex items-center space-x-2 text-sm py-2">
                    <Key className="w-4 h-4" />
                    <span>{t('credential.list')}</span>
                  </TabsTrigger>
                  <TabsTrigger value="create" className="flex items-center space-x-2 text-sm py-2">
                    <Plus className="w-4 h-4" />
                    <span>{t('credential.createTab')}</span>
                  </TabsTrigger>
                </TabsList>

                {/* 密钥列表标签页 */}
                <TabsContent value="list" className="mt-6">
                  <Card>
                    <div className="p-6">
                      <div className="flex items-center justify-between mb-4">
                        <h3 className="text-lg font-semibold text-gray-900 dark:text-white">{t('credential.myKeys')}</h3>
                        <Button
                          variant="outline"
                          size="sm"
                          leftIcon={<Plus className="w-4 h-4" />}
                          onClick={() => setActiveTab('create')}
                        >
                          {t('credential.newKey')}
                        </Button>
                      </div>
                      
                      {credentials.length === 0 ? (
                        <div className="text-center py-12">
                          <Key className="w-12 h-12 text-gray-400 mx-auto mb-4" />
                          <h4 className="text-lg font-medium text-gray-900 dark:text-white mb-2">{t('credential.empty')}</h4>
                          <p className="text-gray-600 dark:text-gray-400 mb-4">{t('credential.emptyDesc')}</p>
                          <Button
                            variant="primary"
                            leftIcon={<Plus className="w-4 h-4" />}
                            onClick={() => setActiveTab('create')}
                          >
                            {t('credential.create')}
                          </Button>
                        </div>
                      ) : (
                        <div className="space-y-4">
                          {credentials.map((cred) => (
                            <div key={cred.id} className="p-4 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
                              <div className="flex items-center justify-between">
                                <div className="flex-1 min-w-0">
                                  <div className="flex items-center space-x-3 mb-2">
                                    <h4 className="font-medium text-gray-900 dark:text-white truncate">{cred.name}</h4>
                                    <Badge variant="secondary" className="text-xs">
                                      {cred.llmProvider || '未配置'}
                                    </Badge>
                                  </div>
                                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                                    <div>
                                      <span className="text-gray-600 dark:text-gray-400">{t('credential.createdAt')}:</span>
                                      <div className="text-gray-900 dark:text-white">
                                        {cred.updatedAt ? new Date(cred.updatedAt).toLocaleDateString('zh-CN', {
                                          year: 'numeric',
                                          month: '2-digit',
                                          day: '2-digit',
                                          hour: '2-digit',
                                          minute: '2-digit'
                                        }) : t('credential.unknown')}
                                      </div>
                                    </div>
                                    <div>
                                      <span className="text-gray-600 dark:text-gray-400">{t('credential.status')}:</span>
                                      <div>
                                        <Badge variant="success" className="text-xs">{t('credential.active')}</Badge>
                                      </div>
                                    </div>
                                    <div>
                                      <span className="text-gray-600 dark:text-gray-400">过期时间:</span>
                                      <div className="text-gray-900 dark:text-white">
                                        {cred.expiresAt ? formatDateTimeInput(cred.expiresAt) : '-'}
                                      </div>
                                    </div>
                                    <div>
                                      <span className="text-gray-600 dark:text-gray-400">令牌可用额度:</span>
                                      <div className="text-gray-900 dark:text-white">{cred.tokenQuota ?? 0}</div>
                                    </div>
                                    <div>
                                      <span className="text-gray-600 dark:text-gray-400">令牌可用数量:</span>
                                      <div className="text-gray-900 dark:text-white">{cred.requestQuota ?? 0}</div>
                                    </div>
                                    <div>
                                      <span className="text-gray-600 dark:text-gray-400">使用原生额度输入:</span>
                                      <div className="text-gray-900 dark:text-white">{cred.useNativeQuota ? '是' : '否'}</div>
                                    </div>
                                    <div>
                                      <span className="text-gray-600 dark:text-gray-400">无限额度:</span>
                                      <div className="text-gray-900 dark:text-white">{cred.unlimitedQuota !== false ? '是' : '否'}</div>
                                    </div>
                                  </div>
                                </div>
                                <div className="flex items-center space-x-2 ml-4">
                                  <Button
                                    variant="outline"
                                    size="sm"
                                    leftIcon={<Download className="w-4 h-4" />}
                                    onClick={() => {
                                      const blob = new Blob(
                                        [`Name: ${cred.name}\nProvider: ${cred.llmProvider || t('credential.notConfigured')}\nUpdated: ${cred.updatedAt ? new Date(cred.updatedAt).toLocaleString('zh-CN') : t('credential.unknown')}`],
                                        { type: "text/plain" }
                                      )
                                      const url = URL.createObjectURL(blob)
                                      const a = document.createElement("a")
                                      a.href = url
                                      a.download = `${cred.name}.txt`
                                      a.click()
                                      URL.revokeObjectURL(url)
                                    }}
                                  >
                                    {t('credential.export')}
                                  </Button>
                                  <Button
                                    variant="destructive"
                                    size="sm"
                                    leftIcon={<Trash2 className="w-4 h-4" />}
                                    onClick={() => openDeleteConfirm(cred.id, cred.name)}
                                  >
                                    {t('credential.delete')}
                                  </Button>
                                </div>
                              </div>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  </Card>
                </TabsContent>

                {/* 创建密钥标签页 */}
                <TabsContent value="create" className="mt-6">
                  <Card>
                    <div className="p-6">
                      <div className="flex items-center justify-between mb-4">
                        <h3 className="text-lg font-semibold text-gray-900 dark:text-white">{t('credential.createNew')}</h3>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => setActiveTab('list')}
                        >
                          {t('credential.backToList')}
                        </Button>
                      </div>
                      
                      <div className="space-y-6">
                        {/* 通用设置 */}
                        <div className="space-y-4">
                          <h4 className="text-md font-semibold text-gray-700 dark:text-gray-300 border-b border-gray-200 dark:border-gray-700 pb-2">
                            {t('credential.generalSettings')}
                          </h4>
                          <Input
                            label={t('credential.keyName')}
                            value={form.name}
                            onChange={(e) => handleFormChange("name", e.target.value)}
                            leftIcon={<Key className="w-4 h-4" />}
                            placeholder={t('credential.keyNamePlaceholder')}
                          />
                        </div>

                        {/* LLM配置 */}
                        <div className="space-y-4">
                          <h4 className="text-md font-semibold text-gray-700 dark:text-gray-300 border-b border-gray-200 dark:border-gray-700 pb-2">
                            {t('credential.llmConfig')}
                          </h4>
                          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                            <AutocompleteInput
                              label={t('credential.provider')}
                              value={form.llmProvider}
                              onChange={(value) => {
                                handleFormChange("llmProvider", value)
                                // 如果是 coze，清空 apiUrl 和 coze 配置
                                if (isCozeProvider(value)) {
                                  handleFormChange("llmApiUrl", "")
                                  setCozeConfig({ userId: "", baseUrl: "" })
                                }
                                // Ollama 的 URL 自动填充已在 handleFormChange 中处理
                              }}
                              options={LLM_PROVIDER_SUGGESTIONS}
                              leftIcon={<Brain className="w-4 h-4" />}
                              placeholder={t('credential.providerPlaceholder')}
                              helperText={t('credential.providerHelper')}
                            />
                            <Input
                              label={
                                isCozeProvider(form.llmProvider) ? 'Coze API Token' : 
                                (isOllamaProvider(form.llmProvider) || isLMStudioProvider(form.llmProvider)) ? 'API Key (可选)' :
                                t('credential.apiKeyLabel')
                              }
                              value={form.llmApiKey}
                              onChange={(e) => handleFormChange("llmApiKey", e.target.value)}
                              leftIcon={<Lock className="w-4 h-4" />}
                              placeholder={
                                isCozeProvider(form.llmProvider) ? '请输入 Coze API Token' : 
                                (isOllamaProvider(form.llmProvider) || isLMStudioProvider(form.llmProvider))
                                  ? '本地 OpenAI 兼容服务可留空 API Key' :
                                t('credential.apiKeyPlaceholder')
                              }
                              type="password"
                              helperText={
                                isCozeProvider(form.llmProvider) ? '从 Coze 平台获取的个人访问令牌 (PAT)' : 
                                (isOllamaProvider(form.llmProvider) || isLMStudioProvider(form.llmProvider))
                                  ? 'Ollama/LM Studio 本地服务通常不要求 API Key，此字段可留空' :
                                undefined
                              }
                            />
                            {isCozeProvider(form.llmProvider) ? (
                              <>
                                <Input
                                  label="Bot ID"
                                  value={form.llmApiUrl}
                                  onChange={(e) => handleFormChange("llmApiUrl", e.target.value)}
                                  leftIcon={<Settings className="w-4 h-4" />}
                                  placeholder="请输入 Coze Bot ID"
                                  helperText="在 Coze 平台上创建的智能体 Bot ID（必需）"
                                />
                                <Input
                                  label="User ID（可选）"
                                  value={cozeConfig.userId}
                                  onChange={(e) => setCozeConfig(prev => ({ ...prev, userId: e.target.value }))}
                                  leftIcon={<Settings className="w-4 h-4" />}
                                  placeholder="自定义 User ID（留空则自动生成）"
                                  helperText="如果不填写，将自动使用 user_{您的用户ID} 格式"
                                />
                                <Input
                                  label="Base URL（可选）"
                                  value={cozeConfig.baseUrl}
                                  onChange={(e) => setCozeConfig(prev => ({ ...prev, baseUrl: e.target.value }))}
                                  leftIcon={<Globe className="w-4 h-4" />}
                                  placeholder="https://api.coze.com"
                                  helperText="Coze API 基础地址（留空使用默认值）"
                                />
                              </>
                            ) : (
                              <Input
                                label={t('credential.apiUrl')}
                                value={form.llmApiUrl}
                                onChange={(e) => handleFormChange("llmApiUrl", e.target.value)}
                                leftIcon={<Globe className="w-4 h-4" />}
                                placeholder={
                                  isOllamaProvider(form.llmProvider)
                                    ? 'http://localhost:11434/v1'
                                    : isLMStudioProvider(form.llmProvider)
                                      ? 'http://localhost:1234/v1'
                                    : t('credential.apiUrlPlaceholder')
                                }
                                helperText={
                                  isOllamaProvider(form.llmProvider)
                                    ? 'Ollama 服务的 API 地址，默认为 http://localhost:11434/v1。如果 Ollama 运行在其他地址，请修改此值。'
                                    : isLMStudioProvider(form.llmProvider)
                                      ? 'LM Studio OpenAI 兼容服务地址，默认为 http://localhost:1234/v1。'
                                    : t('credential.apiUrlHelper')
                                }
                              />
                            )}
                          </div>
                        </div>

                        {/* ASR配置 */}
                        <div className="space-y-4">
                          <h4 className="text-md font-semibold text-gray-700 dark:text-gray-300 border-b border-gray-200 dark:border-gray-700 pb-2">
                            {t('credential.asrConfig')}
                          </h4>
                          <div className="mb-4">
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                              {t('credential.serviceProvider')}
                            </label>
                            <Select
                              value={asrProvider}
                              onValueChange={(value) => {
                                setAsrProvider(value)
                                setAsrConfigFields({})
                              }}
                            >
                              <SelectTrigger className="w-full">
                                <SelectValue placeholder={t('credential.selectProvider')} />
                              </SelectTrigger>
                              <SelectContent searchable searchPlaceholder="搜索ASR服务商">
                                <SelectItem value="">{t('credential.selectProvider')}</SelectItem>
                                {getASRProviderOptions().map((opt) => (
                                  <SelectItem key={opt.value} value={opt.value}>
                                    {opt.label}
                                  </SelectItem>
                                ))}
                              </SelectContent>
                            </Select>
                          </div>
                          <ProviderConfigForm
                            provider={asrProvider}
                            config={getASRProviderConfig(asrProvider)}
                            values={asrConfigFields}
                            onChange={(key, value) => {
                              setAsrConfigFields(prev => ({ ...prev, [key]: value }))
                            }}
                            prefix="asr"
                          />
                        </div>

                        {/* TTS配置 */}
                        <div className="space-y-4">
                          <h4 className="text-md font-semibold text-gray-700 dark:text-gray-300 border-b border-gray-200 dark:border-gray-700 pb-2">
                            {t('credential.ttsConfig')}
                          </h4>
                          <div className="mb-4">
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                              {t('credential.serviceProvider')}
                            </label>
                            <Select
                              value={ttsProvider}
                              onValueChange={(value) => {
                                setTtsProvider(value)
                                setTtsConfigFields({})
                              }}
                            >
                              <SelectTrigger className="w-full">
                                <SelectValue placeholder={t('credential.selectProvider')} />
                              </SelectTrigger>
                              <SelectContent searchable searchPlaceholder="搜索TTS服务商">
                                <SelectItem value="">{t('credential.selectProvider')}</SelectItem>
                                {getTTSProviderOptions().map((opt) => (
                                  <SelectItem key={opt.value} value={opt.value}>
                                    {opt.label}
                                  </SelectItem>
                                ))}
                              </SelectContent>
                            </Select>
                          </div>
                          <ProviderConfigForm
                            provider={ttsProvider}
                            config={getTTSProviderConfig(ttsProvider)}
                            values={ttsConfigFields}
                            onChange={(key, value) => {
                              setTtsConfigFields(prev => ({ ...prev, [key]: value }))
                            }}
                            prefix="tts"
                          />
                        </div>

                        <div className="space-y-4">
                          <h4 className="text-md font-semibold text-gray-700 dark:text-gray-300 border-b border-gray-200 dark:border-gray-700 pb-2">
                            过期与额度设置
                          </h4>
                          <div>
                            <div className="text-sm font-medium mb-2">过期时间快捷设置</div>
                            <div className="flex flex-wrap gap-2 mb-2">
                              <Button size="sm" variant="outline" onClick={() => handleFormChange("expiresAt", addDays(1))}>+1天</Button>
                              <Button size="sm" variant="outline" onClick={() => handleFormChange("expiresAt", addDays(7))}>+7天</Button>
                              <Button size="sm" variant="outline" onClick={() => handleFormChange("expiresAt", addDays(30))}>+30天</Button>
                              <Button size="sm" variant="outline" onClick={() => handleFormChange("expiresAt", DATE_NEVER)}>1970-01-01 07:59:59</Button>
                              <Button size="sm" variant="ghost" onClick={() => handleFormChange("expiresAt", "")}>清空</Button>
                            </div>
                            <Input
                              label="过期时间"
                              value={form.expiresAt || ""}
                              onChange={(e) => handleFormChange("expiresAt", e.target.value)}
                              placeholder="YYYY-MM-DD HH:MM:SS"
                            />
                          </div>
                          <div className="space-y-2">
                            <div className="text-sm font-medium">令牌分组额度设置</div>
                            <label className="flex items-center gap-2 text-sm">
                              <input
                                type="checkbox"
                                checked={!!form.useNativeQuota}
                                onChange={(e) => setForm(prev => ({ ...prev, useNativeQuota: e.target.checked }))}
                              />
                              <span>▸ 使用原生额度输入</span>
                            </label>
                            <label className="flex items-center gap-2 text-sm">
                              <input
                                type="checkbox"
                                checked={form.unlimitedQuota !== false}
                                onChange={(e) => setForm(prev => ({ ...prev, unlimitedQuota: e.target.checked }))}
                              />
                              <span>无限额度</span>
                            </label>
                            <Input
                              label="设置令牌可用额度"
                              type="number"
                              value={String(form.tokenQuota ?? 0)}
                              onChange={(e) => setForm(prev => ({ ...prev, tokenQuota: Number(e.target.value || 0) }))}
                            />
                            <Input
                              label="设置令牌可用数量"
                              type="number"
                              value={String(form.requestQuota ?? 0)}
                              onChange={(e) => setForm(prev => ({ ...prev, requestQuota: Number(e.target.value || 0) }))}
                            />
                          </div>
                        </div>

                        <div className="flex justify-end space-x-3">
                          <Button
                            variant="outline"
                            onClick={() => setActiveTab('list')}
                            disabled={isCreating}
                          >
                            {t('credential.cancel')}
                          </Button>
                          <Button
                            variant="primary"
                            leftIcon={<Plus className="w-4 h-4" />}
                            onClick={handleCreate}
                            loading={isCreating}
                          >
                            {isCreating ? t('credential.creating') : t('credential.create')}
                          </Button>
                        </div>
                      </div>
                    </div>
                  </Card>
                </TabsContent>
        </Tabs>
      </div>

      {/* 密钥创建成功弹窗 */}
      <AnimatePresence>
        {generatedKey && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
          >
            <motion.div
              initial={{ scale: 0.9, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.9, opacity: 0 }}
              className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-lg max-w-md w-full mx-4"
            >
              <div className="flex items-center space-x-3 mb-4">
                <div className="p-2 bg-green-100 dark:bg-green-900/30 rounded-lg">
                  <CheckCircle className="w-6 h-6 text-green-600 dark:text-green-400" />
                </div>
                <h3 className="text-lg font-bold text-gray-900 dark:text-white">密钥创建成功</h3>
              </div>
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                请妥善保存以下信息：
              </p>
              <div className="space-y-3 mb-6">
                <div className="bg-gray-100 dark:bg-gray-700 p-3 rounded-md">
                  <div className="text-xs text-gray-500 dark:text-gray-400 mb-1">API Key</div>
                  <div className="font-mono text-sm break-all">{generatedKey.apiKey}</div>
                </div>
                <div className="bg-gray-100 dark:bg-gray-700 p-3 rounded-md">
                  <div className="text-xs text-gray-500 dark:text-gray-400 mb-1">API Secret</div>
                  <div className="font-mono text-sm break-all">{generatedKey.apiSecret}</div>
                </div>
              </div>
              <div className="flex justify-end space-x-3">
                <Button
                  variant="outline"
                  leftIcon={<Download className="w-4 h-4" />}
                  onClick={handleExport}
                >
                  导出
                </Button>
                <Button
                  variant="primary"
                  onClick={() => setGeneratedKey(null)}
                >
                  我已保存
                </Button>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* 删除确认对话框 */}
      <ConfirmDialog
        isOpen={deleteConfirm.isOpen}
        onClose={closeDeleteConfirm}
        onConfirm={confirmDelete}
        title={t('credential.deleteConfirmTitle') || '删除密钥'}
        message={t('credential.deleteConfirmMessage', { name: deleteConfirm.credentialName }) || `确定要删除密钥 "${deleteConfirm.credentialName}" 吗？此操作不可恢复。`}
        confirmText={t('credential.delete') || '删除'}
        cancelText={t('common.cancel') || '取消'}
        type="danger"
        loading={deleteConfirm.isDeleting}
      />

    </div>
  )
}

export default CredentialManager
