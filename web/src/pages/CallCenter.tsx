import { useState, useEffect } from 'react'
import { 
  Phone, 
  PhoneCall, 
  PhoneOff, 
  Search, 
  User, 
  Clock, 
  CheckCircle, 
  XCircle, 
  AlertCircle,
  Settings,
  Mail,
  Smartphone
} from 'lucide-react'
import { getSipUsers, makeOutgoingCall, getOutgoingCallStatus, cancelOutgoingCall, hangupOutgoingCall, getCallHistory, type SipUser, type SipCall } from '@/api/sip'
import { useI18nStore } from '@/stores/i18nStore'
import Button from '@/components/UI/Button'
import { showAlert } from '@/utils/notification'
import SchemeManager from './SchemeManager'
import PhoneNumberManager from './PhoneNumberManager'
import VoicemailBox from './VoicemailBox'

type TabType = 'call' | 'schemes' | 'numbers' | 'voicemail'

const CallCenter = () => {
  const { t } = useI18nStore()
  const [activeTab, setActiveTab] = useState<TabType>('call')
  const [sipUsers, setSipUsers] = useState<SipUser[]>([])
  const [selectedUser, setSelectedUser] = useState<SipUser | null>(null)
  const [searchTerm, setSearchTerm] = useState('')
  const [loading, setLoading] = useState(false)
  const [calling, setCalling] = useState(false)
  const [currentCallId, setCurrentCallId] = useState<string | null>(null)
  const [callStatus, setCallStatus] = useState<string>('')
  const [callHistory, setCallHistory] = useState<SipCall[]>([])
  const [showHistory, setShowHistory] = useState(false)

  // Tab 配置
  const tabs = [
    { id: 'call' as TabType, name: '通话控制', icon: PhoneCall },
    { id: 'schemes' as TabType, name: '代接方案', icon: Settings },
    { id: 'numbers' as TabType, name: '号码管理', icon: Smartphone },
    { id: 'voicemail' as TabType, name: '留言箱', icon: Mail },
  ]

  // 加载SIP用户列表
  useEffect(() => {
    loadSipUsers()
    loadCallHistory()
  }, [])

  // 轮询呼出状态
  useEffect(() => {
    if (currentCallId && calling) {
      const interval = setInterval(() => {
        checkCallStatus(currentCallId)
      }, 2000) // 每2秒检查一次

      return () => clearInterval(interval)
    }
  }, [currentCallId, calling])

  const loadSipUsers = async () => {
    try {
      setLoading(true)
      const res = await getSipUsers()
      if (res.code === 200 && res.data) {
        setSipUsers(res.data)
      }
    } catch (error) {
      console.error('加载SIP用户失败:', error)
      showAlert('加载SIP用户失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  const loadCallHistory = async () => {
    try {
      const res = await getCallHistory({ limit: 20 })
      if (res.code === 200 && res.data) {
        setCallHistory(res.data)
      }
    } catch (error) {
      console.error('加载通话历史失败:', error)
    }
  }

  const checkCallStatus = async (callId: string) => {
    try {
      const res = await getOutgoingCallStatus(callId)
      if (res.code === 200 && res.data) {
        setCallStatus(res.data.status)
        
        // 如果通话已结束或失败，停止轮询
        if (['ended', 'failed', 'cancelled'].includes(res.data.status)) {
          setCalling(false)
          setCurrentCallId(null)
          loadCallHistory() // 刷新历史记录
        }
      }
    } catch (error) {
      console.error('获取通话状态失败:', error)
    }
  }

  const handleMakeCall = async () => {
    if (!selectedUser) {
      showAlert('请选择要呼叫的用户', 'warning')
      return
    }

    if (calling) {
      showAlert('正在通话中，请稍候', 'warning')
      return
    }

    try {
      setCalling(true)
      setCallStatus('calling')
      
      // 构建目标URI
      const targetUri = selectedUser.contact || `sip:${selectedUser.username}@${selectedUser.contactIp || '127.0.0.1'}:${selectedUser.contactPort || 5060}`
      
      const res = await makeOutgoingCall({
        targetUri,
        notes: `外呼到 ${selectedUser.displayName || selectedUser.username}`,
      })

      if (res.code === 200 && res.data) {
        setCurrentCallId(res.data.callId)
        setCallStatus(res.data.status)
        showAlert('呼叫已发起', 'success')
        loadCallHistory() // 刷新历史记录
      } else {
        throw new Error(res.msg || '发起呼叫失败')
      }
    } catch (error: any) {
      console.error('发起呼叫失败:', error)
      showAlert(error.message || '发起呼叫失败', 'error')
      setCalling(false)
      setCallStatus('')
    }
  }

  const handleCancelCall = async () => {
    if (!currentCallId) return

    try {
      // 如果已接通，使用挂断；否则使用取消
      const res = callStatus === 'answered' 
        ? await hangupOutgoingCall(currentCallId)
        : await cancelOutgoingCall(currentCallId)
      
      if (res.code === 200) {
        setCalling(false)
        setCurrentCallId(null)
        setCallStatus('')
        showAlert(callStatus === 'answered' ? '通话已挂断' : '呼叫已取消', 'success')
        loadCallHistory() // 刷新历史记录
      } else {
        throw new Error(res.msg || (callStatus === 'answered' ? '挂断失败' : '取消呼叫失败'))
      }
    } catch (error: any) {
      console.error('操作失败:', error)
      showAlert(error.message || (callStatus === 'answered' ? '挂断失败' : '取消呼叫失败'), 'error')
    }
  }

  const filteredUsers = sipUsers.filter(user => {
    const search = searchTerm.toLowerCase()
    return (
      user.username.toLowerCase().includes(search) ||
      (user.displayName && user.displayName.toLowerCase().includes(search)) ||
      (user.alias && user.alias.toLowerCase().includes(search))
    )
  })

  const getStatusBadge = (status: string) => {
    const statusConfig = {
      calling: { icon: PhoneCall, color: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200', text: '呼叫中' },
      ringing: { icon: Phone, color: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200', text: '响铃中' },
      answered: { icon: CheckCircle, color: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200', text: '已接通' },
      failed: { icon: XCircle, color: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200', text: '失败' },
      cancelled: { icon: PhoneOff, color: 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200', text: '已取消' },
      ended: { icon: CheckCircle, color: 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200', text: '已结束' },
    }

    const config = statusConfig[status as keyof typeof statusConfig] || {
      icon: AlertCircle,
      color: 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200',
      text: status,
    }

    const Icon = config.icon
    return (
      <span className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium ${config.color}`}>
        <Icon className="w-3 h-3" />
        {config.text}
      </span>
    )
  }

  return (
    <div className="p-6 max-w-7xl mx-auto">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100 mb-2">
          {t('callCenter.title')}
        </h1>
        <p className="text-gray-600 dark:text-gray-400">
          通话控制、AI代接方案、号码管理和留言箱
        </p>
      </div>

      {/* Tabs */}
      <div className="mb-6 border-b border-gray-200 dark:border-gray-700">
        <div className="flex gap-1">
          {tabs.map((tab) => {
            const Icon = tab.icon
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`
                  flex items-center gap-2 px-4 py-3 font-medium text-sm rounded-t-lg transition-all
                  ${activeTab === tab.id
                    ? 'bg-white dark:bg-gray-800 text-blue-600 dark:text-blue-400 border-b-2 border-blue-600 dark:border-blue-400'
                    : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100 hover:bg-gray-50 dark:hover:bg-gray-800/50'
                  }
                `}
              >
                <Icon className="w-4 h-4" />
                {tab.name}
              </button>
            )
          })}
        </div>
      </div>

      {/* Tab Content */}
      <div className="mt-6">
        {activeTab === 'call' && (
          <CallControlTab
            sipUsers={sipUsers}
            selectedUser={selectedUser}
            setSelectedUser={setSelectedUser}
            searchTerm={searchTerm}
            setSearchTerm={setSearchTerm}
            loading={loading}
            calling={calling}
            currentCallId={currentCallId}
            callStatus={callStatus}
            callHistory={callHistory}
            showHistory={showHistory}
            setShowHistory={setShowHistory}
            handleMakeCall={handleMakeCall}
            handleCancelCall={handleCancelCall}
            filteredUsers={filteredUsers}
            getStatusBadge={getStatusBadge}
            t={t}
          />
        )}
        {activeTab === 'schemes' && <SchemeManager />}
        {activeTab === 'numbers' && <PhoneNumberManager />}
        {activeTab === 'voicemail' && <VoicemailBox />}
      </div>
    </div>
  )
}

// 通话控制 Tab 组件
const CallControlTab = ({
  sipUsers,
  selectedUser,
  setSelectedUser,
  searchTerm,
  setSearchTerm,
  loading,
  calling,
  currentCallId,
  callStatus,
  callHistory,
  showHistory,
  setShowHistory,
  handleMakeCall,
  handleCancelCall,
  filteredUsers,
  getStatusBadge,
  t
}: any) => {
  return (
    <>
      {/* 主操作区域 */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6">
        {/* 用户选择区域 */}
        <div className="lg:col-span-2 bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <div className="mb-4">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
              <input
                type="text"
                placeholder={t('callCenter.searchPlaceholder')}
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
          </div>

          <div className="space-y-2 max-h-96 overflow-y-auto">
            {loading ? (
              <div className="text-center py-8 text-gray-500">加载中...</div>
            ) : filteredUsers.length === 0 ? (
              <div className="text-center py-8 text-gray-500">没有找到SIP用户</div>
            ) : (
              filteredUsers.map((user) => (
                <div
                  key={user.id}
                  onClick={() => setSelectedUser(user)}
                  className={`p-4 rounded-lg border-2 cursor-pointer transition-all ${
                    selectedUser?.id === user.id
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                      : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-full bg-blue-100 dark:bg-blue-900 flex items-center justify-center">
                        <User className="w-5 h-5 text-blue-600 dark:text-blue-400" />
                      </div>
                      <div>
                        <div className="font-medium text-gray-900 dark:text-gray-100">
                          {user.displayName || user.alias || user.username}
                        </div>
                        <div className="text-sm text-gray-500 dark:text-gray-400">
                          {user.username}
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      {getStatusBadge(user.status)}
                      {user.enabled ? (
                        <span className="text-xs text-green-600 dark:text-green-400">已启用</span>
                      ) : (
                        <span className="text-xs text-red-600 dark:text-red-400">已禁用</span>
                      )}
                    </div>
                  </div>
                  {user.contact && (
                    <div className="mt-2 text-xs text-gray-500 dark:text-gray-400">
                      联系地址: {user.contact}
                    </div>
                  )}
                </div>
              ))
            )}
          </div>
        </div>

        {/* 呼叫控制区域 */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-gray-100">
            {t('callCenter.callControl')}
          </h2>

          {selectedUser ? (
            <div className="space-y-4">
              <div className="p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
                <div className="text-sm text-gray-600 dark:text-gray-400 mb-1">被叫用户</div>
                <div className="font-medium text-gray-900 dark:text-gray-100">
                  {selectedUser.displayName || selectedUser.alias || selectedUser.username}
                </div>
                <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  {selectedUser.username}
                </div>
              </div>

              {calling && currentCallId && (
                <div className="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                  <div className="text-sm text-gray-600 dark:text-gray-400 mb-1">通话状态</div>
                  <div className="flex items-center gap-2">
                    {getStatusBadge(callStatus)}
                  </div>
                  <div className="text-xs text-gray-500 dark:text-gray-400 mt-2">
                    通话ID: {currentCallId}
                  </div>
                </div>
              )}

              <div className="flex gap-2">
                {!calling ? (
                  <Button
                    onClick={handleMakeCall}
                    disabled={!selectedUser.enabled}
                    className="flex-1"
                  >
                    <Phone className="w-4 h-4 mr-2" />
                    {t('callCenter.makeCall')}
                  </Button>
                ) : (
                  <Button
                    onClick={handleCancelCall}
                    variant="destructive"
                    className="flex-1"
                  >
                    <PhoneOff className="w-4 h-4 mr-2" />
                    {callStatus === 'answered' ? '挂断' : t('callCenter.cancelCall')}
                  </Button>
                )}
              </div>

              {!selectedUser.enabled && (
                <div className="text-xs text-red-600 dark:text-red-400">
                  该用户已被禁用，无法发起呼叫
                </div>
              )}
            </div>
          ) : (
            <div className="text-center py-8 text-gray-500">
              请从左侧选择要呼叫的用户
            </div>
          )}
        </div>
      </div>

      {/* 通话历史 */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
        <div className="p-6 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 flex items-center gap-2">
            <Clock className="w-5 h-5" />
            {t('callCenter.callHistory')}
          </h2>
          <Button
            variant="ghost"
            onClick={() => setShowHistory(!showHistory)}
          >
            {showHistory ? '收起' : '展开'}
          </Button>
        </div>

        {showHistory && (
          <div className="p-6">
            {callHistory.length === 0 ? (
              <div className="text-center py-8 text-gray-500">暂无通话记录</div>
            ) : (
              <div className="space-y-2">
                {callHistory.map((call) => (
                  <div
                    key={call.id}
                    className="p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-4">
                        <div>
                          <div className="font-medium text-gray-900 dark:text-gray-100">
                            {call.direction === 'outbound' ? '呼出' : '呼入'}
                          </div>
                          <div className="text-sm text-gray-500 dark:text-gray-400">
                            {call.toUri || call.fromUri}
                          </div>
                        </div>
                        {getStatusBadge(call.status)}
                      </div>
                      <div className="text-right">
                        <div className="text-sm text-gray-600 dark:text-gray-400">
                          {new Date(call.startTime).toLocaleString()}
                        </div>
                        {call.duration > 0 && (
                          <div className="text-xs text-gray-500 dark:text-gray-400">
                            时长: {Math.floor(call.duration / 60)}分{call.duration % 60}秒
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </>
  )
}

export default CallCenter

