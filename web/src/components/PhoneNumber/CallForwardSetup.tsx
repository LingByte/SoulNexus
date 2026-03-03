import React, { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { 
  Phone, 
  ArrowRight, 
  CheckCircle, 
  AlertCircle,
  Copy,
  RefreshCw,
  HelpCircle,
  PhoneForwarded
} from 'lucide-react'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/UI/Select'
import { showAlert } from '@/utils/notification'
import type { PhoneNumber } from '@/types/phoneNumber'

interface CallForwardSetupProps {
  phoneNumber: PhoneNumber
  onClose: () => void
  onSuccess: () => void
}

interface SetupStep {
  order: number
  description: string
  action?: string
}

interface SetupInstructions {
  code: string
  description: string
  steps: SetupStep[]
}

const CallForwardSetup: React.FC<CallForwardSetupProps> = ({
  phoneNumber,
  onClose,
  onSuccess
}) => {
  const [step, setStep] = useState<'input' | 'instructions' | 'verify'>('input')
  const [targetNumber, setTargetNumber] = useState('')
  const [carrier, setCarrier] = useState('移动')
  const [instructions, setInstructions] = useState<SetupInstructions | null>(null)
  const [loading, setLoading] = useState(false)
  const [verifying, setVerifying] = useState(false)

  // 获取设置指引
  const getInstructions = async () => {
    if (!targetNumber) {
      showAlert('请输入转移目标号码', 'warning')
      return
    }

    try {
      setLoading(true)
      const response = await fetch('/api/call-forward/setup-instructions', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('auth_token')}`
        },
        body: JSON.stringify({
          phoneNumberId: phoneNumber.id,
          targetNumber,
          carrier
        })
      })

      const data = await response.json()
      if (data.code === 200) {
        setInstructions(data.data)
        setStep('instructions')
      } else {
        showAlert(data.message || '获取指引失败', 'error')
      }
    } catch (error) {
      console.error('获取指引失败:', error)
      showAlert('获取指引失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  // 复制代码
  const copyCode = (code: string) => {
    navigator.clipboard.writeText(code)
    showAlert('已复制到剪贴板', 'success')
  }

  // 验证设置
  const verifySetup = async () => {
    try {
      setVerifying(true)
      const response = await fetch(`/api/call-forward/${phoneNumber.id}/status`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('auth_token')}`
        },
        body: JSON.stringify({
          enabled: true,
          targetNumber
        })
      })

      const data = await response.json()
      if (data.code === 200) {
        showAlert('呼叫转移已启用', 'success')
        onSuccess()
        onClose()
      } else {
        showAlert(data.message || '更新失败', 'error')
      }
    } catch (error) {
      console.error('更新失败:', error)
      showAlert('更新失败', 'error')
    } finally {
      setVerifying(false)
    }
  }

  // 测试呼叫转移
  const testForward = async () => {
    try {
      setLoading(true)
      const response = await fetch(`/api/call-forward/${phoneNumber.id}/test`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('auth_token')}`
        }
      })

      const data = await response.json()
      if (data.code === 200) {
        showAlert('测试呼叫已发起，请注意接听', 'success')
      } else {
        showAlert(data.message || '测试失败', 'error')
      }
    } catch (error) {
      console.error('测试失败:', error)
      showAlert('测试失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        exit={{ opacity: 0, scale: 0.95 }}
        className="bg-card rounded-lg shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto"
      >
        {/* Header */}
        <div className="sticky top-0 bg-card border-b border-border p-6 z-10">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <PhoneForwarded className="w-6 h-6 text-primary" />
              <div>
                <h2 className="text-xl font-semibold text-foreground">设置呼叫转移</h2>
                <p className="text-sm text-muted-foreground mt-1">
                  号码：{phoneNumber.phoneNumber}
                </p>
              </div>
            </div>
            <button
              onClick={onClose}
              className="text-muted-foreground hover:text-foreground transition-colors"
            >
              ✕
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="p-6">
          <AnimatePresence mode="wait">
            {/* Step 1: 输入目标号码 */}
            {step === 'input' && (
              <motion.div
                key="input"
                initial={{ opacity: 0, x: 20 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -20 }}
                className="space-y-6"
              >
                <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
                  <div className="flex items-start gap-3">
                    <HelpCircle className="w-5 h-5 text-blue-600 dark:text-blue-400 mt-0.5" />
                    <div className="text-sm text-blue-900 dark:text-blue-100">
                      <p className="font-medium mb-1">什么是呼叫转移？</p>
                      <p>将您的真实手机号来电自动转移到虚拟号码，由 AI 助手代接。</p>
                    </div>
                  </div>
                </div>

                <div className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-foreground mb-2">
                      转移目标号码（虚拟号码）
                    </label>
                    <Input
                      type="tel"
                      value={targetNumber}
                      onChange={(e) => setTargetNumber(e.target.value)}
                      placeholder="例如：13800138000"
                      className="w-full"
                    />
                    <p className="text-xs text-muted-foreground mt-1">
                      请输入您在 SIP 服务商处申请的虚拟号码
                    </p>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-foreground mb-2">
                      运营商
                    </label>
                    <Select value={carrier} onValueChange={setCarrier}>
                      <SelectTrigger className="w-full">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="移动">中国移动</SelectItem>
                        <SelectItem value="联通">中国联通</SelectItem>
                        <SelectItem value="电信">中国电信</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <div className="flex justify-end gap-3">
                  <Button variant="outline" onClick={onClose}>
                    取消
                  </Button>
                  <Button onClick={getInstructions} loading={loading}>
                    下一步
                    <ArrowRight className="w-4 h-4 ml-2" />
                  </Button>
                </div>
              </motion.div>
            )}

            {/* Step 2: 显示设置指引 */}
            {step === 'instructions' && instructions && (
              <motion.div
                key="instructions"
                initial={{ opacity: 0, x: 20 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -20 }}
                className="space-y-6"
              >
                <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4">
                  <p className="text-sm text-green-900 dark:text-green-100">
                    {instructions.description}
                  </p>
                </div>

                {/* 拨号代码 */}
                <div className="bg-card border border-border rounded-lg p-4">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-sm font-medium text-muted-foreground">拨号代码</span>
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => copyCode(instructions.code)}
                    >
                      <Copy className="w-4 h-4 mr-1" />
                      复制
                    </Button>
                  </div>
                  <div className="text-2xl font-mono font-bold text-primary">
                    {instructions.code}
                  </div>
                </div>

                {/* 操作步骤 */}
                <div className="space-y-3">
                  <h3 className="text-sm font-medium text-foreground">操作步骤</h3>
                  {instructions.steps.map((s) => (
                    <div
                      key={s.order}
                      className="flex items-start gap-3 p-3 bg-muted/50 rounded-lg"
                    >
                      <div className="flex-shrink-0 w-6 h-6 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-sm font-medium">
                        {s.order}
                      </div>
                      <div className="flex-1">
                        <p className="text-sm text-foreground">{s.description}</p>
                        {s.action && (
                          <div className="mt-2 flex items-center gap-2">
                            <code className="px-2 py-1 bg-card border border-border rounded text-sm font-mono">
                              {s.action}
                            </code>
                            <Button
                              size="sm"
                              variant="ghost"
                              onClick={() => copyCode(s.action!)}
                            >
                              <Copy className="w-3 h-3" />
                            </Button>
                          </div>
                        )}
                      </div>
                    </div>
                  ))}
                </div>

                <div className="flex justify-between gap-3">
                  <Button variant="outline" onClick={() => setStep('input')}>
                    上一步
                  </Button>
                  <div className="flex gap-3">
                    <Button variant="outline" onClick={testForward} loading={loading}>
                      <Phone className="w-4 h-4 mr-2" />
                      测试呼叫
                    </Button>
                    <Button onClick={verifySetup} loading={verifying}>
                      <CheckCircle className="w-4 h-4 mr-2" />
                      我已完成设置
                    </Button>
                  </div>
                </div>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </motion.div>
    </div>
  )
}

export default CallForwardSetup
