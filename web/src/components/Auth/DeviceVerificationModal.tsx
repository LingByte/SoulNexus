import { useState, useEffect } from 'react'
import { Modal, Input, Button, Typography, Alert, Space, ConfigProvider } from '@arco-design/web-react'
import { IconEmail, IconSafe } from '@arco-design/web-react/icon'
import { sendDeviceVerificationCode, verifyDevice } from '@/api/auth'
import { showAlert } from '@/utils/notification'
import { AUTH_PRIMARY_COLOR } from './CaptchaModal'

interface DeviceVerificationModalProps {
  isOpen: boolean
  onClose: () => void
  onSuccess: () => void
  email: string
  deviceId: string
  message?: string
}

const DeviceVerificationModal = ({
  isOpen,
  onClose,
  onSuccess,
  email,
  deviceId,
  message = '此设备不受信任，需要验证',
}: DeviceVerificationModalProps) => {
  const [verificationCode, setVerificationCode] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [isSendingCode, setIsSendingCode] = useState(false)
  const [countdown, setCountdown] = useState(0)

  useEffect(() => {
    if (countdown <= 0) return
    const timer = window.setTimeout(() => setCountdown(countdown - 1), 1000)
    return () => clearTimeout(timer)
  }, [countdown])

  useEffect(() => {
    if (!isOpen) {
      setVerificationCode('')
      setCountdown(0)
    }
  }, [isOpen])

  const handleSendCode = async () => {
    if (!email || !deviceId) {
      showAlert('设备信息不完整', 'error', '验证失败')
      return
    }
    setIsSendingCode(true)
    try {
      const response = await sendDeviceVerificationCode({ email, deviceId })
      if (response.code === 200) {
        showAlert('设备验证码已发送到您的邮箱，请在5分钟内验证', 'success', '发送成功')
        setCountdown(60)
      } else {
        throw new Error(response.msg || '设备验证码发送失败')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '设备验证码发送失败，请重试', 'error', '发送失败')
    } finally {
      setIsSendingCode(false)
    }
  }

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
        verifyCode: verificationCode,
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
      showAlert(error?.msg || error?.message || '设备验证失败，请重试', 'error', '验证失败')
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <ConfigProvider theme={{ primaryColor: AUTH_PRIMARY_COLOR }}>
    <Modal
      visible={isOpen}
      title="设备验证"
      onCancel={onClose}
      onOk={handleVerifyDevice}
      okText="验证设备"
      cancelText="取消"
      confirmLoading={isLoading}
      okButtonProps={{ disabled: !verificationCode.trim() }}
      style={{ width: 440 }}
      unmountOnExit
    >
      <Typography.Paragraph type="secondary">{message}</Typography.Paragraph>

      <div
        style={{
          padding: 12,
          borderRadius: 8,
          background: 'var(--color-fill-2)',
          marginBottom: 16,
        }}
      >
        <Space direction="vertical" size={4}>
          <Typography.Text>
            <IconEmail style={{ marginRight: 6 }} />
            邮箱：{email}
          </Typography.Text>
          <Typography.Text type="secondary">
            设备ID：{deviceId ? `${deviceId.substring(0, 8)}...` : '-'}
          </Typography.Text>
        </Space>
      </div>

      <Input
        prefix={<IconSafe />}
        placeholder="请输入邮箱中的验证码"
        value={verificationCode}
        onChange={setVerificationCode}
        addAfter={
          <Button
            type="text"
            size="small"
            loading={isSendingCode}
            disabled={countdown > 0}
            onClick={handleSendCode}
          >
            {countdown > 0 ? `${countdown}s` : '发送验证码'}
          </Button>
        }
      />

      <Alert
        type="info"
        style={{ marginTop: 16 }}
        content="验证码 5 分钟内有效。验证后此设备将被标记为受信任设备。"
      />
    </Modal>
    </ConfigProvider>
  )
}

export default DeviceVerificationModal
