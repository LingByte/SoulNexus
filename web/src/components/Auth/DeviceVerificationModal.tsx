import React, { useState } from 'react'
import { motion } from 'framer-motion'
import { Shield, Mail, AlertTriangle } from 'lucide-react'
import Modal, { ModalContent, ModalFooter } from '../UI/Modal'
import Button from '../UI/Button'
import Input from '../UI/Input'
import { sendDeviceVerificationCode, verifyDevice } from '@/api/auth'
import { showAlert } from '@/utils/notification'

interface DeviceVerificationModalProps {
  isOpen: boolean
  onClose: () => void
  onSuccess: () => void
  email: string
  deviceId: string
  message?: string
}

const DeviceVerificationModal: React.FC<DeviceVerificationModalProps> = ({
  isOpen,
  onClose,
  onSuccess,
  email,
  deviceId,
  message = '此设备不受信任，需要验证'
}) => {
  const [verificationCode, setVerificationCode] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [isSendingCode, setIsSendingCode] = useState(false)
  const [countdown, setCountdown] = useState(0)

  // 发送设备验证码
  const handleSendCode = async () => {
    if (!email || !deviceId) {
      showAlert('设备信息不完整', 'error', '验证失败')
      return
    }

    setIsSendingCode(true)
    try {
      const response = await sendDeviceVerificationCode({
        email,
        deviceId
      })
      
      if (response.code === 200) {
        showAlert('设备验证码已发送到您的邮箱，请在5分钟内验证', 'success', '发送成功')
        setCountdown(60) // 60秒倒计时
        
        // 启动倒计时
        const timer = setInterval(() => {
          setCountdown((prev) => {
            if (prev <= 1) {
              clearInterval(timer)
              return 0
            }
            return prev - 1
          })
        }, 1000)
      } else {
        throw new Error(response.msg || '设备验证码发送失败')
      }
    } catch (error: any) {
      console.error('Send device code error:', error)
      showAlert(error?.msg || error?.message || '设备验证码发送失败，请重试', 'error', '发送失败')
    } finally {
      setIsSendingCode(false)
    }
  }

  // 验证设备
  const handleVerifyDevice = async () => {
    if (!verificationCode.trim()) {
      showAlert('请输入设备验证码', 'error', '验证失败')
      return
    }

    setIsLoading(true)
    try {
      const response = await verifyDevice({
        email,
        deviceId,
        verifyCode: verificationCode
      })
      
      if (response.code === 200) {
        showAlert('设备验证成功，现在可以重新登录', 'success', '验证成功')
        setVerificationCode('')
        onSuccess()
        onClose()
      } else {
        throw new Error(response.msg || '设备验证失败')
      }
    } catch (error: any) {
      console.error('Device verification error:', error)
      showAlert(error?.msg || error?.message || '设备验证失败，请重试', 'error', '验证失败')
    } finally {
      setIsLoading(false)
    }
  }

  const handleClose = () => {
    setVerificationCode('')
    setCountdown(0)
    onClose()
  }

  return (
    <Modal
      isOpen={isOpen}
      onClose={handleClose}
      size="sm"
      closeOnOverlayClick={!isLoading}
      closeOnEscape={!isLoading}
      showCloseButton={false}
    >
      <ModalContent>
        <div className="text-center mb-6">
          <motion.div
            initial={{ scale: 0 }}
            animate={{ scale: 1 }}
            transition={{ delay: 0.1, type: "spring", stiffness: 200 }}
            className="w-16 h-16 bg-yellow-100 dark:bg-yellow-900/20 rounded-full flex items-center justify-center mx-auto mb-4"
          >
            <Shield className="w-8 h-8 text-yellow-600 dark:text-yellow-400" />
          </motion.div>
          
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
            设备验证
          </h3>
          
          <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
            {message}
          </p>

          <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 mb-4 text-left">
            <div className="flex items-center space-x-2 mb-2">
              <Mail className="w-4 h-4 text-gray-500" />
              <span className="text-sm text-gray-600 dark:text-gray-400">
                <strong>邮箱：</strong>{email}
              </span>
            </div>
            <div className="flex items-center space-x-2">
              <AlertTriangle className="w-4 h-4 text-gray-500" />
              <span className="text-sm text-gray-600 dark:text-gray-400">
                <strong>设备ID：</strong>{deviceId.substring(0, 8)}...
              </span>
            </div>
          </div>
        </div>

        <div className="space-y-4">
          <div className="flex space-x-2">
            <Input
              label="设备验证码"
              placeholder="请输入邮箱中的验证码"
              value={verificationCode}
              onChange={(e) => setVerificationCode(e.target.value)}
              leftIcon={<Mail className="w-5 h-5" />}
              className="flex-1"
              required
            />
            <Button
              type="button"
              variant="outline"
              onClick={handleSendCode}
              disabled={isSendingCode || countdown > 0}
              className="mt-6 px-3 whitespace-nowrap"
            >
              {isSendingCode ? '发送中...' : countdown > 0 ? `${countdown}s` : '发送验证码'}
            </Button>
          </div>

          <div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-3">
            <div className="flex items-start space-x-2">
              <Shield className="w-4 h-4 text-blue-600 dark:text-blue-400 mt-0.5 flex-shrink-0" />
              <div className="text-xs text-blue-700 dark:text-blue-300">
                <p className="font-medium mb-1">安全提醒：</p>
                <ul className="space-y-1">
                  <li>• 验证码将在5分钟后过期</li>
                  <li>• 如果这不是您的操作，请立即更改密码</li>
                  <li>• 验证后，此设备将被标记为受信任设备</li>
                </ul>
              </div>
            </div>
          </div>
        </div>
      </ModalContent>
      
      <ModalFooter>
        <Button
          variant="outline"
          onClick={handleClose}
          disabled={isLoading}
        >
          取消
        </Button>
        <Button
          variant="primary"
          onClick={handleVerifyDevice}
          disabled={isLoading || !verificationCode.trim()}
          loading={isLoading}
        >
          验证设备
        </Button>
      </ModalFooter>
    </Modal>
  )
}

export default DeviceVerificationModal