import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { 
  FileText, Download, RefreshCw, Brain, Mic, Volume2, HardDrive,
  Globe, Plus, Eye, Search, ChevronLeft, ChevronRight,
  BarChart3, List
} from 'lucide-react'
import { useI18nStore } from '@/stores/i18nStore'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Card, { CardContent, CardDescription, CardHeader, CardTitle } from '@/components/UI/Card'
import Badge from '@/components/UI/Badge'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/UI/Select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/UI/Tabs'
import FadeIn from '@/components/Animations/FadeIn'
import { showAlert } from '@/utils/notification'
import {
  getUsageStatistics,
  getDailyUsageData,
  getUsageRecords,
  exportUsageRecords,
  generateBill,
  getBills,
  getBill,
  exportBill,
  type UsageStatistics,
  type DailyUsageData,
  type UsageRecord,
  type Bill,
  type UsageType,
  type BillStatus,
  type GenerateBillRequest
} from '@/api/billing'
import UsageCharts from '@/components/Billing/UsageCharts'
import { fetchUserCredentials, type Credential } from '@/api/credential'
import { useAuthStore } from '@/stores/authStore'
import { getGroupList, type Group } from '@/api/group'
import { Building2, User } from 'lucide-react'

const Billing = () => {
  const { t } = useI18nStore()
  const { user } = useAuthStore()
  
  // 状态管理
  const [activeTab, setActiveTab] = useState<'statistics' | 'records' | 'bills'>('statistics')
  const [loading, setLoading] = useState(false)
  
  // 统计数据
  const [statistics, setStatistics] = useState<UsageStatistics | null>(null)
  const [dailyData, setDailyData] = useState<DailyUsageData[]>([])
  
  // 使用量记录
  const [usageRecords, setUsageRecords] = useState<UsageRecord[]>([])
  const [recordsTotal, setRecordsTotal] = useState(0)
  const [recordsPage, setRecordsPage] = useState(1)
  const [recordsSize] = useState(20)
  
  // 账单
  const [bills, setBills] = useState<Bill[]>([])
  const [billsTotal, setBillsTotal] = useState(0)
  const [billsPage, setBillsPage] = useState(1)
  const [billsSize] = useState(20)
  const [selectedBill, setSelectedBill] = useState<Bill | null>(null)
  const [showBillDetail, setShowBillDetail] = useState(false)
  
  // 凭证列表
  const [credentials, setCredentials] = useState<Credential[]>([])
  
  // 组织列表
  const [groups, setGroups] = useState<Group[]>([])
  
  // 筛选条件
  const [dateRange, setDateRange] = useState<'7d' | '30d' | '90d' | 'custom'>('30d')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [credentialFilter, setCredentialFilter] = useState<string>('all')
  const [usageTypeFilter, setUsageTypeFilter] = useState<string>('all')
  const [billStatusFilter, setBillStatusFilter] = useState<string>('all')
  const [billingScope, setBillingScope] = useState<'personal' | 'organization'>('personal')
  const [selectedGroupId, setSelectedGroupId] = useState<number | null>(null)
  
  // 生成账单表单
  const [showGenerateBill, setShowGenerateBill] = useState(false)
  const [generateBillForm, setGenerateBillForm] = useState<GenerateBillRequest>({
    startTime: '',
    endTime: '',
    title: '',
    credentialId: undefined,
    groupId: undefined
  })
  
  // 初始化日期范围
  useEffect(() => {
    const end = new Date()
    const start = new Date()
    start.setDate(start.getDate() - 30)
    setStartDate(start.toISOString().split('T')[0])
    setEndDate(end.toISOString().split('T')[0])
  }, [])
  
  // 加载凭证列表
  useEffect(() => {
    const loadCredentials = async () => {
      try {
        const response = await fetchUserCredentials()
        if (response.code === 200) {
          setCredentials(response.data || [])
        }
      } catch (error) {
        console.error('Failed to load credentials', error)
      }
    }
    loadCredentials()
  }, [])
  
  // 加载组织列表
  useEffect(() => {
    const loadGroups = async () => {
      try {
        const response = await getGroupList()
        if (response.code === 200) {
          setGroups(response.data || [])
        }
      } catch (error) {
        console.error('Failed to load groups', error)
      }
    }
    loadGroups()
  }, [])
  
  // 更新日期范围
  useEffect(() => {
    const end = new Date()
    const start = new Date()
    
    switch (dateRange) {
      case '7d':
        start.setDate(start.getDate() - 7)
        break
      case '30d':
        start.setDate(start.getDate() - 30)
        break
      case '90d':
        start.setDate(start.getDate() - 90)
        break
      case 'custom':
        return // 不自动更新
    }
    
    setStartDate(start.toISOString().split('T')[0])
    setEndDate(end.toISOString().split('T')[0])
  }, [dateRange])
  
  // 加载统计数据
  const loadStatistics = async () => {
    setLoading(true)
    try {
      const params: any = {
        startTime: startDate,
        endTime: endDate
      }
      if (credentialFilter !== 'all') {
        params.credentialId = parseInt(credentialFilter)
      }
      // 如果是组织账单，添加组织ID
      if (billingScope === 'organization' && selectedGroupId) {
        params.groupId = selectedGroupId
      }
      
      // 并行加载统计数据和每日数据
      const [statsResponse, dailyResponse] = await Promise.all([
        getUsageStatistics(params),
        getDailyUsageData(params)
      ])
      
      if (statsResponse.code === 200) {
        setStatistics(statsResponse.data)
      } else {
        throw new Error(statsResponse.msg || 'Failed to load statistics')
      }
      
      if (dailyResponse.code === 200) {
        setDailyData(dailyResponse.data || [])
      } else {
        console.warn('Failed to load daily usage data:', dailyResponse.msg)
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || t('billing.messages.loadStatsFailed'), 'error')
    } finally {
      setLoading(false)
    }
  }
  
  // 加载使用量记录
  const loadUsageRecords = async () => {
    setLoading(true)
    try {
      const params: any = {
        page: recordsPage,
        size: recordsSize,
        startTime: startDate,
        endTime: endDate,
        orderBy: 'usageTime DESC'
      }
      
      if (credentialFilter !== 'all') {
        params.credentialId = parseInt(credentialFilter)
      }
      if (usageTypeFilter !== 'all') {
        params.usageType = usageTypeFilter
      }
      // 如果是组织账单，添加组织ID
      if (billingScope === 'organization' && selectedGroupId) {
        params.groupId = selectedGroupId
      }
      
      const response = await getUsageRecords(params)
      if (response.code === 200) {
        setUsageRecords(response.data.list || [])
        setRecordsTotal(response.data.total || 0)
      } else {
        throw new Error(response.msg || 'Failed to load usage records')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || t('billing.messages.loadRecordsFailed'), 'error')
    } finally {
      setLoading(false)
    }
  }
  
  // 加载账单列表
  const loadBills = async () => {
    setLoading(true)
    try {
      const params: any = {
        page: billsPage,
        size: billsSize,
        orderBy: 'createdAt DESC'
      }
      
      if (credentialFilter !== 'all') {
        params.credentialId = parseInt(credentialFilter)
      }
      if (billStatusFilter !== 'all') {
        params.status = billStatusFilter
      }
      // 如果是组织账单，添加组织ID
      if (billingScope === 'organization' && selectedGroupId) {
        params.groupId = selectedGroupId
      }
      
      const response = await getBills(params)
      if (response.code === 200) {
        setBills(response.data.list || [])
        setBillsTotal(response.data.total || 0)
      } else {
        throw new Error(response.msg || 'Failed to load bills')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || t('billing.messages.loadBillsFailed'), 'error')
    } finally {
      setLoading(false)
    }
  }
  
  // 根据当前tab加载数据
  useEffect(() => {
    if (activeTab === 'statistics') {
      loadStatistics()
    } else if (activeTab === 'records') {
      loadUsageRecords()
    } else if (activeTab === 'bills') {
      loadBills()
    }
  }, [activeTab, dateRange, startDate, endDate, credentialFilter, usageTypeFilter, billStatusFilter, recordsPage, billsPage, billingScope, selectedGroupId])
  
  // 导出格式
  const [exportFormat, setExportFormat] = useState<'csv' | 'excel'>('csv')
  
  // 导出使用量记录
  const handleExportRecords = async () => {
    try {
      const params: any = {
        startTime: startDate,
        endTime: endDate,
        format: exportFormat
      }
      
      if (credentialFilter !== 'all') {
        params.credentialId = parseInt(credentialFilter)
      }
      if (usageTypeFilter !== 'all') {
        params.usageType = usageTypeFilter
      }
      // 如果是组织账单，添加组织ID
      if (billingScope === 'organization' && selectedGroupId) {
        params.groupId = selectedGroupId
      }
      
      await exportUsageRecords(params)
      showAlert('导出成功，文件将自动下载', 'success')
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '导出失败', 'error')
    }
  }
  
  // 生成账单
  const handleGenerateBill = async () => {
    if (!generateBillForm.startTime || !generateBillForm.endTime) {
      showAlert('请选择时间范围', 'error')
      return
    }
    
    // 如果是组织账单，必须选择组织
    if (billingScope === 'organization' && !selectedGroupId) {
      showAlert('请选择组织', 'error')
      return
    }
    
    try {
      const formData = {
        ...generateBillForm,
        groupId: billingScope === 'organization' ? selectedGroupId : undefined
      }
      const response = await generateBill(formData)
      if (response.code === 200) {
        showAlert(t('billing.generate.success'), 'success')
        setShowGenerateBill(false)
        setGenerateBillForm({ startTime: '', endTime: '', title: '', credentialId: undefined, groupId: undefined })
        loadBills()
      } else {
        throw new Error(response.msg || 'Generate bill failed')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || t('billing.generate.failed'), 'error')
    }
  }
  
  // 查看账单详情
  const handleViewBill = async (billId: number) => {
    try {
      const response = await getBill(billId)
      if (response.code === 200) {
        setSelectedBill(response.data)
        setShowBillDetail(true)
      } else {
        throw new Error(response.msg || 'Failed to load bill')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || t('billing.messages.loadBillFailed'), 'error')
    }
  }
  
  // 导出账单
  const handleExportBill = async (billId: number, format: 'csv' | 'excel' = 'csv') => {
    try {
      await exportBill(billId, format)
      showAlert('导出成功，文件将自动下载', 'success')
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '导出失败', 'error')
    }
  }
  
  // 格式化数字
  const formatNumber = (num: number) => {
    if (num >= 1000000) {
      return (num / 1000000).toFixed(2) + 'M'
    }
    if (num >= 1000) {
      return (num / 1000).toFixed(2) + 'K'
    }
    return num.toString()
  }
  
  // 格式化时长
  const formatDuration = (seconds: number) => {
    if (seconds < 60) {
      return `${seconds}秒`
    }
    if (seconds < 3600) {
      return `${Math.floor(seconds / 60)}分钟`
    }
    return `${Math.floor(seconds / 3600)}小时${Math.floor((seconds % 3600) / 60)}分钟`
  }
  
  // 格式化文件大小
  const formatFileSize = (bytes: number) => {
    if (bytes < 1024) {
      return `${bytes}B`
    }
    if (bytes < 1024 * 1024) {
      return `${(bytes / 1024).toFixed(2)}KB`
    }
    if (bytes < 1024 * 1024 * 1024) {
      return `${(bytes / (1024 * 1024)).toFixed(2)}MB`
    }
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)}GB`
  }
  
  // 获取使用类型标签
  const getUsageTypeLabel = (type: UsageType) => {
    const labels: Record<UsageType, string> = {
      llm: t('billing.usageType.llm'),
      call: t('billing.usageType.call'),
      asr: t('billing.usageType.asr'),
      tts: t('billing.usageType.tts'),
      api: t('billing.usageType.api')
    }
    return labels[type] || type
  }
  
  // 获取账单状态标签
  const getBillStatusLabel = (status: BillStatus) => {
    const labels: Record<BillStatus, string> = {
      draft: t('billing.status.draft'),
      generated: t('billing.status.generated'),
      exported: t('billing.status.exported'),
      archived: t('billing.status.archived')
    }
    return labels[status] || status
  }
  
  // 获取账单状态颜色
  const getBillStatusColor = (status: BillStatus) => {
    const colors: Record<BillStatus, string> = {
      draft: 'bg-gray-500',
      generated: 'bg-blue-500',
      exported: 'bg-green-500',
      archived: 'bg-gray-400'
    }
    return colors[status] || 'bg-gray-500'
  }
  
  return (
    <div className="container mx-auto px-4 py-6 space-y-6">
      <FadeIn>
        <div className="flex items-center justify-between">
          <div className="relative pl-4">
            <motion.div
              layoutId="pageTitleIndicator"
              className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-8 bg-primary rounded-r-full"
              transition={{ type: 'spring', bounce: 0.2, duration: 0.3 }}
            />
            <h1 className="text-3xl font-bold">{t('billing.title')}</h1>
            <p className="text-muted-foreground mt-1">{t('billing.subtitle')}</p>
          </div>
          <Button
            variant="primary"
            onClick={() => setShowGenerateBill(true)}
            leftIcon={<Plus className="w-4 h-4" />}
          >
            {t('billing.generateBill')}
          </Button>
        </div>
      </FadeIn>
      
      {/* 筛选栏 */}
      <FadeIn delay={0.1}>
        <Card>
          <CardContent className="pt-6">
            <div className="grid grid-cols-1 md:grid-cols-5 gap-4">
              {/* 账单范围选择 */}
              <div>
                <label className="text-sm font-medium mb-2 block">账单范围</label>
                <Select value={billingScope} onValueChange={(value: any) => {
                  setBillingScope(value)
                  if (value === 'personal') {
                    setSelectedGroupId(null)
                  } else if (value === 'organization' && groups.length > 0 && !selectedGroupId) {
                    // 默认选择第一个组织
                    setSelectedGroupId(groups[0].id)
                  }
                }}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="personal">
                      <div className="flex items-center gap-2">
                        <User className="w-4 h-4" />
                        <span>个人账单</span>
                      </div>
                    </SelectItem>
                    <SelectItem value="organization">
                      <div className="flex items-center gap-2">
                        <Building2 className="w-4 h-4" />
                        <span>组织账单</span>
                      </div>
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
              
              {/* 组织选择（仅当选择组织账单时显示） */}
              {billingScope === 'organization' && (
                <div>
                  <label className="text-sm font-medium mb-2 block">选择组织</label>
                  <Select 
                    value={selectedGroupId?.toString() || ''} 
                    onValueChange={(value) => setSelectedGroupId(value ? parseInt(value) : null)}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="请选择组织" />
                    </SelectTrigger>
                    <SelectContent>
                      {groups.map((group) => (
                        <SelectItem key={group.id} value={group.id.toString()}>
                          {group.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              )}
              
              <div>
                <label className="text-sm font-medium mb-2 block">{t('billing.filter.dateRange')}</label>
                <Select value={dateRange} onValueChange={(value: any) => setDateRange(value)}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="7d">{t('billing.filter.last7Days')}</SelectItem>
                    <SelectItem value="30d">{t('billing.filter.last30Days')}</SelectItem>
                    <SelectItem value="90d">{t('billing.filter.last90Days')}</SelectItem>
                    <SelectItem value="custom">{t('billing.filter.custom')}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              
              {dateRange === 'custom' && (
                <>
                  <div>
                    <label className="text-sm font-medium mb-2 block">{t('billing.filter.startDate')}</label>
                    <Input
                      type="date"
                      value={startDate}
                      onChange={(e) => setStartDate(e.target.value)}
                    />
                  </div>
                  <div>
                    <label className="text-sm font-medium mb-2 block">{t('billing.filter.endDate')}</label>
                    <Input
                      type="date"
                      value={endDate}
                      onChange={(e) => setEndDate(e.target.value)}
                    />
                  </div>
                </>
              )}
              
              {billingScope === 'personal' && (
              <div>
                <label className="text-sm font-medium mb-2 block">{t('billing.filter.credential')}</label>
                <Select value={credentialFilter} onValueChange={setCredentialFilter}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">{t('billing.filter.all')}</SelectItem>
                    {credentials.map((cred) => (
                      <SelectItem key={cred.id} value={cred.id.toString()}>
                        {cred.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              )}
              
              {activeTab === 'records' && (
                <div>
                  <label className="text-sm font-medium mb-2 block">{t('billing.filter.usageType')}</label>
                  <Select value={usageTypeFilter} onValueChange={setUsageTypeFilter}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">{t('billing.filter.all')}</SelectItem>
                      <SelectItem value="llm">{t('billing.usageType.llm')}</SelectItem>
                      <SelectItem value="asr">{t('billing.usageType.asr')}</SelectItem>
                      <SelectItem value="tts">{t('billing.usageType.tts')}</SelectItem>
                      <SelectItem value="api">{t('billing.usageType.api')}</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              )}
              
              {activeTab === 'bills' && (
                <div>
                  <label className="text-sm font-medium mb-2 block">{t('billing.filter.status')}</label>
                  <Select value={billStatusFilter} onValueChange={setBillStatusFilter}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">{t('billing.filter.all')}</SelectItem>
                      <SelectItem value="draft">{t('billing.status.draft')}</SelectItem>
                      <SelectItem value="generated">{t('billing.status.generated')}</SelectItem>
                      <SelectItem value="exported">{t('billing.status.exported')}</SelectItem>
                      <SelectItem value="archived">{t('billing.status.archived')}</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      </FadeIn>
      
      {/* 主要内容 */}
      <Tabs value={activeTab} onValueChange={(value: any) => setActiveTab(value)}>
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="statistics">
            <BarChart3 className="w-4 h-4 mr-2" />
            {t('billing.tabs.statistics')}
          </TabsTrigger>
          <TabsTrigger value="records">
            <List className="w-4 h-4 mr-2" />
            {t('billing.tabs.records')}
          </TabsTrigger>
          <TabsTrigger value="bills">
            <FileText className="w-4 h-4 mr-2" />
            {t('billing.tabs.bills')}
          </TabsTrigger>
        </TabsList>
        
        {/* 统计概览 */}
        <TabsContent value="statistics" className="space-y-6">
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <RefreshCw className="w-6 h-6 animate-spin text-muted-foreground" />
            </div>
          ) : statistics ? (
            <>
              {/* 统计卡片 */}
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Brain className="w-5 h-5 text-blue-500" />
                      {t('billing.stats.llm')}
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-2">
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">{t('billing.stats.callCount')}</span>
                      <span className="font-semibold">{formatNumber(statistics.llmCalls)}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">{t('billing.stats.totalTokens')}</span>
                      <span className="font-semibold">{formatNumber(statistics.llmTokens)}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">{t('billing.stats.promptTokens')}</span>
                      <span className="font-semibold">{formatNumber(statistics.promptTokens)}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">{t('billing.stats.completionTokens')}</span>
                      <span className="font-semibold">{formatNumber(statistics.completionTokens)}</span>
                    </div>
                  </CardContent>
                </Card>
                
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Mic className="w-5 h-5 text-purple-500" />
                      {t('billing.stats.asr')}
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-2">
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">{t('billing.stats.callCount')}</span>
                      <span className="font-semibold">{formatNumber(statistics.asrCount)}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">{t('billing.stats.totalDuration')}</span>
                      <span className="font-semibold">{formatDuration(statistics.asrDuration)}</span>
                    </div>
                  </CardContent>
                </Card>
                
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Volume2 className="w-5 h-5 text-orange-500" />
                      {t('billing.stats.tts')}
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-2">
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">{t('billing.stats.callCount')}</span>
                      <span className="font-semibold">{formatNumber(statistics.ttsCount)}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">{t('billing.stats.totalDuration')}</span>
                      <span className="font-semibold">{formatDuration(statistics.ttsDuration)}</span>
                    </div>
                  </CardContent>
                </Card>
                
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Globe className="w-5 h-5 text-cyan-500" />
                      {t('billing.stats.api')}
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">{t('billing.stats.apiCalls')}</span>
                      <span className="font-semibold">{formatNumber(statistics.apiCalls)}</span>
                    </div>
                  </CardContent>
                </Card>
              </div>
              
              {/* 图表可视化 */}
              {dailyData.length > 0 && (
                <UsageCharts dailyData={dailyData} statistics={statistics} />
              )}
            </>
          ) : (
            <Card>
              <CardContent className="py-12 text-center text-muted-foreground">
                {t('billing.stats.noData')}
              </CardContent>
            </Card>
          )}
        </TabsContent>
        
        {/* 使用记录 */}
        <TabsContent value="records" className="space-y-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Select value={exportFormat} onValueChange={(value: any) => setExportFormat(value)}>
                <SelectTrigger className="w-32">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="csv">CSV</SelectItem>
                  <SelectItem value="excel">Excel</SelectItem>
                </SelectContent>
              </Select>
              <Button
                variant="outline"
                onClick={handleExportRecords}
                leftIcon={<Download className="w-4 h-4" />}
              >
                {t('billing.records.export')}
              </Button>
            </div>
          </div>
          
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <RefreshCw className="w-6 h-6 animate-spin text-muted-foreground" />
            </div>
          ) : usageRecords.length > 0 ? (
            <>
              <Card>
                <CardContent className="p-0">
                  <div className="overflow-x-auto">
                    <table className="w-full">
                      <thead>
                        <tr className="border-b">
                          <th className="px-4 py-3 text-left text-sm font-medium">{t('billing.records.time')}</th>
                          <th className="px-4 py-3 text-left text-sm font-medium">{t('billing.records.type')}</th>
                          <th className="px-4 py-3 text-left text-sm font-medium">{t('billing.records.model')}</th>
                          <th className="px-4 py-3 text-left text-sm font-medium">{t('billing.records.tokens')}</th>
                          <th className="px-4 py-3 text-left text-sm font-medium">{t('billing.records.duration')}</th>
                          <th className="px-4 py-3 text-left text-sm font-medium">{t('billing.records.size')}</th>
                        </tr>
                      </thead>
                      <tbody>
                        {usageRecords.map((record) => (
                          <tr key={record.id} className="border-b hover:bg-accent/50">
                            <td className="px-4 py-3 text-sm">
                              {new Date(record.usageTime).toLocaleString()}
                            </td>
                            <td className="px-4 py-3 text-sm">
                              <Badge>{getUsageTypeLabel(record.usageType)}</Badge>
                            </td>
                            <td className="px-4 py-3 text-sm">{record.model || '-'}</td>
                            <td className="px-4 py-3 text-sm">
                              {record.totalTokens > 0 ? formatNumber(record.totalTokens) : '-'}
                            </td>
                            <td className="px-4 py-3 text-sm">
                              {record.callDuration > 0 || record.audioDuration > 0
                                ? formatDuration(record.callDuration || record.audioDuration)
                                : '-'}
                            </td>
                            <td className="px-4 py-3 text-sm">
                              {record.audioSize > 0
                                ? formatFileSize(record.audioSize)
                                : '-'}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </CardContent>
              </Card>
              
              {/* 分页 */}
              <div className="flex items-center justify-between">
                <div className="text-sm text-muted-foreground">
                  {t('billing.records.total', { count: recordsTotal })}
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setRecordsPage(Math.max(1, recordsPage - 1))}
                    disabled={recordsPage === 1}
                    leftIcon={<ChevronLeft className="w-4 h-4" />}
                  >
                    {t('billing.records.prev')}
                  </Button>
                  <span className="text-sm">
                    {t('billing.records.page', { current: recordsPage, total: Math.ceil(recordsTotal / recordsSize) })}
                  </span>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setRecordsPage(recordsPage + 1)}
                    disabled={recordsPage >= Math.ceil(recordsTotal / recordsSize)}
                    rightIcon={<ChevronRight className="w-4 h-4" />}
                  >
                    {t('billing.records.next')}
                  </Button>
                </div>
              </div>
            </>
          ) : (
            <Card>
              <CardContent className="py-12 text-center text-muted-foreground">
                {t('billing.records.noData')}
              </CardContent>
            </Card>
          )}
        </TabsContent>
        
        {/* 账单管理 */}
        <TabsContent value="bills" className="space-y-6">
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <RefreshCw className="w-6 h-6 animate-spin text-muted-foreground" />
            </div>
          ) : bills.length > 0 ? (
            <>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {bills.map((bill) => (
                  <Card key={bill.id}>
                    <CardHeader>
                      <div className="flex items-start justify-between">
                        <div>
                          <CardTitle className="text-lg">{bill.title}</CardTitle>
                          <CardDescription className="mt-1">{bill.billNo}</CardDescription>
                        </div>
                        <Badge className={getBillStatusColor(bill.status)}>
                          {getBillStatusLabel(bill.status)}
                        </Badge>
                      </div>
                    </CardHeader>
                    <CardContent className="space-y-3">
                      <div className="text-sm space-y-1">
                        <div className="flex justify-between">
                          <span className="text-muted-foreground">{t('billing.bills.timeRange')}</span>
                          <span>
                            {new Date(bill.startTime).toLocaleDateString()} - {new Date(bill.endTime).toLocaleDateString()}
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-muted-foreground">{t('billing.bills.llmCalls')}</span>
                          <span className="font-semibold">{formatNumber(bill.totalLLMCalls)}</span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-muted-foreground">{t('billing.bills.totalTokens')}</span>
                          <span className="font-semibold">{formatNumber(bill.totalLLMTokens)}</span>
                        </div>
                      </div>
                      <div className="flex gap-2 pt-2">
                        <Button
                          variant="outline"
                          size="sm"
                          className="flex-1"
                          onClick={() => handleViewBill(bill.id)}
                          leftIcon={<Eye className="w-4 h-4" />}
                        >
                          {t('billing.bills.view')}
                        </Button>
                        <div className="flex gap-1">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleExportBill(bill.id, 'csv')}
                            title="导出为CSV"
                          >
                            CSV
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleExportBill(bill.id, 'excel')}
                            title="导出为Excel"
                          >
                            Excel
                          </Button>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
              
              {/* 分页 */}
              <div className="flex items-center justify-between">
                <div className="text-sm text-muted-foreground">
                  {t('billing.bills.total', { count: billsTotal })}
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setBillsPage(Math.max(1, billsPage - 1))}
                    disabled={billsPage === 1}
                    leftIcon={<ChevronLeft className="w-4 h-4" />}
                  >
                    {t('billing.records.prev')}
                  </Button>
                  <span className="text-sm">
                    {t('billing.bills.page', { current: billsPage, total: Math.ceil(billsTotal / billsSize) })}
                  </span>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setBillsPage(billsPage + 1)}
                    disabled={billsPage >= Math.ceil(billsTotal / billsSize)}
                    rightIcon={<ChevronRight className="w-4 h-4" />}
                  >
                    {t('billing.records.next')}
                  </Button>
                </div>
              </div>
            </>
          ) : (
            <Card>
              <CardContent className="py-12 text-center text-muted-foreground">
                {t('billing.bills.noData')}
              </CardContent>
            </Card>
          )}
        </TabsContent>
      </Tabs>
      
      {/* 生成账单弹窗 */}
      {showGenerateBill && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <Card className="w-full max-w-md">
            <CardHeader>
              <CardTitle>{t('billing.generate.title')}</CardTitle>
              <CardDescription>{t('billing.generate.description')}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <label className="text-sm font-medium mb-2 block">{t('billing.filter.startDate')}</label>
                <Input
                  type="date"
                  value={generateBillForm.startTime}
                  onChange={(e) => setGenerateBillForm({ ...generateBillForm, startTime: e.target.value })}
                />
              </div>
              <div>
                <label className="text-sm font-medium mb-2 block">{t('billing.filter.endDate')}</label>
                <Input
                  type="date"
                  value={generateBillForm.endTime}
                  onChange={(e) => setGenerateBillForm({ ...generateBillForm, endTime: e.target.value })}
                />
              </div>
              <div>
                <label className="text-sm font-medium mb-2 block">{t('billing.generate.billTitle')}</label>
                <Input
                  value={generateBillForm.title || ''}
                  onChange={(e) => setGenerateBillForm({ ...generateBillForm, title: e.target.value })}
                  placeholder={t('billing.generate.billTitlePlaceholder')}
                />
              </div>
              {billingScope === 'personal' && (
              <div>
                <label className="text-sm font-medium mb-2 block">{t('billing.generate.credential')}</label>
                <Select
                  value={generateBillForm.credentialId?.toString() || 'all'}
                  onValueChange={(value) => setGenerateBillForm({
                    ...generateBillForm,
                    credentialId: value === 'all' ? undefined : parseInt(value)
                  })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">{t('billing.generate.allCredentials')}</SelectItem>
                    {credentials.map((cred) => (
                      <SelectItem key={cred.id} value={cred.id.toString()}>
                        {cred.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              )}
              {billingScope === 'organization' && (
                <div>
                  <label className="text-sm font-medium mb-2 block">选择组织</label>
                  <Select
                    value={selectedGroupId?.toString() || ''}
                    onValueChange={(value) => {
                      setSelectedGroupId(value ? parseInt(value) : null)
                      setGenerateBillForm({
                        ...generateBillForm,
                        groupId: value ? parseInt(value) : undefined
                      })
                    }}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="请选择组织" />
                    </SelectTrigger>
                    <SelectContent>
                      {groups.map((group) => (
                        <SelectItem key={group.id} value={group.id.toString()}>
                          {group.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              )}
              <div className="flex gap-2 pt-4">
                <Button
                  variant="outline"
                  className="flex-1"
                  onClick={() => {
                    setShowGenerateBill(false)
                    setGenerateBillForm({ startTime: '', endTime: '', title: '', credentialId: undefined, groupId: undefined })
                  }}
                >
                  {t('billing.generate.cancel')}
                </Button>
                <Button
                  variant="primary"
                  className="flex-1"
                  onClick={handleGenerateBill}
                >
                  {t('billing.generate.submit')}
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
      
      {/* 账单详情弹窗 */}
      {showBillDetail && selectedBill && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <Card className="w-full max-w-2xl max-h-[90vh] overflow-y-auto">
            <CardHeader>
              <div className="flex items-start justify-between">
                <div>
                  <CardTitle>{selectedBill.title}</CardTitle>
                  <CardDescription className="mt-1">{selectedBill.billNo}</CardDescription>
                </div>
                <Badge className={getBillStatusColor(selectedBill.status)}>
                  {getBillStatusLabel(selectedBill.status)}
                </Badge>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="text-sm text-muted-foreground">{t('billing.detail.startTime')}</div>
                  <div className="font-medium">{new Date(selectedBill.startTime).toLocaleString()}</div>
                </div>
                <div>
                  <div className="text-sm text-muted-foreground">{t('billing.detail.endTime')}</div>
                  <div className="font-medium">{new Date(selectedBill.endTime).toLocaleString()}</div>
                </div>
              </div>
              
              <div className="border-t pt-4">
                <h3 className="font-semibold mb-3">{t('billing.detail.usageStats')}</h3>
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">{t('billing.stats.callCount')} ({t('billing.usageType.llm')})</span>
                    <span className="font-semibold">{formatNumber(selectedBill.totalLLMCalls)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">{t('billing.stats.totalTokens')}</span>
                    <span className="font-semibold">{formatNumber(selectedBill.totalLLMTokens)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">{t('billing.stats.promptTokens')}</span>
                    <span className="font-semibold">{formatNumber(selectedBill.totalPromptTokens)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">{t('billing.stats.completionTokens')}</span>
                    <span className="font-semibold">{formatNumber(selectedBill.totalCompletionTokens)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">{t('billing.stats.totalDuration')} ({t('billing.usageType.asr')})</span>
                    <span className="font-semibold">{formatDuration(selectedBill.totalASRDuration)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">{t('billing.stats.callCount')} ({t('billing.usageType.asr')})</span>
                    <span className="font-semibold">{formatNumber(selectedBill.totalASRCount)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">{t('billing.stats.totalDuration')} ({t('billing.usageType.tts')})</span>
                    <span className="font-semibold">{formatDuration(selectedBill.totalTTSDuration)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">{t('billing.stats.callCount')} ({t('billing.usageType.tts')})</span>
                    <span className="font-semibold">{formatNumber(selectedBill.totalTTSCount)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">{t('billing.stats.apiCalls')}</span>
                    <span className="font-semibold">{formatNumber(selectedBill.totalAPICalls)}</span>
                  </div>
                </div>
              </div>
              
              {selectedBill.notes && (
                <div className="border-t pt-4">
                  <div className="text-sm text-muted-foreground mb-1">{t('billing.detail.notes')}</div>
                  <div className="text-sm">{selectedBill.notes}</div>
                </div>
              )}
              
              <div className="flex gap-2 pt-4">
                <Button
                  variant="outline"
                  className="flex-1"
                  onClick={() => {
                    setShowBillDetail(false)
                    setSelectedBill(null)
                  }}
                >
                  {t('billing.detail.close')}
                </Button>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    className="flex-1"
                    onClick={() => handleExportBill(selectedBill.id, 'csv')}
                    leftIcon={<Download className="w-4 h-4" />}
                  >
                    CSV
                  </Button>
                  <Button
                    variant="primary"
                    className="flex-1"
                    onClick={() => handleExportBill(selectedBill.id, 'excel')}
                    leftIcon={<Download className="w-4 h-4" />}
                  >
                    Excel
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  )
}

export default Billing

