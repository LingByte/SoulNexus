import {
  Modal,
  Form,
  Input,
  Button,
  Alert,
  Result,
  Typography,
  Space,
  Link,
  Grid,
  ConfigProvider,
} from '@arco-design/web-react'
import {
  IconEmail,
  IconLock,
  IconUser,
  IconSafe,
} from '@arco-design/web-react/icon'
import { useNavigate } from 'react-router-dom'
import PasswordStrength from './PasswordStrength'
import CaptchaModal, { AUTH_PRIMARY_COLOR } from './CaptchaModal'
import DeviceVerificationModal from './DeviceVerificationModal'
import { useAuthModalLogic } from './useAuthModalLogic'
import { useI18nStore } from '@/stores/i18nStore'
import type { LoginType } from './useAuthModalLogic'

const { Row, Col } = Grid

function LoginTypeSwitch({
  value,
  onChange,
}: {
  value: LoginType
  onChange: (value: LoginType) => void
}) {
  const itemStyle = (active: boolean): React.CSSProperties => ({
    flex: 1,
    height: 40,
    border: 'none',
    cursor: 'pointer',
    fontSize: 14,
    fontWeight: 500,
    transition: 'all 0.2s ease',
    background: active ? AUTH_PRIMARY_COLOR : 'var(--color-fill-2)',
    color: active ? '#fff' : 'var(--color-text-2)',
  })

  return (
    <div
      style={{
        display: 'flex',
        marginBottom: 16,
        borderRadius: 8,
        overflow: 'hidden',
        border: '1px solid var(--color-border-2)',
      }}
    >
      <button type="button" style={itemStyle(value === 'email')} onClick={() => onChange('email')}>
        验证码登录
      </button>
      <button type="button" style={itemStyle(value === 'password')} onClick={() => onChange('password')}>
        密码登录
      </button>
    </div>
  )
}

function SendCodeButton({
  loading,
  countdown,
  onClick,
}: {
  loading: boolean
  countdown: number
  onClick: () => void
}) {
  const label = loading ? '发送中' : countdown > 0 ? `${countdown}s` : '发送验证码'
  return (
    <Button
      type="text"
      size="small"
      loading={loading}
      disabled={countdown > 0}
      onClick={onClick}
      style={{ color: AUTH_PRIMARY_COLOR }}
    >
      {label}
    </Button>
  )
}

const AuthModal = () => {
  const { t } = useI18nStore()
  const navigate = useNavigate()
  const logic = useAuthModalLogic()

  const {
    isOpen,
    required,
    mode,
    setMode,
    loginType,
    setLoginType,
    isLoading,
    isSendingCode,
    countdown,
    isRegisterSuccess,
    registerSuccessData,
    isLoginSuccess,
    loginSuccessData,
    showTwoFactorInput,
    twoFactorCode,
    setTwoFactorCode,
    setShowTwoFactorInput,
    showCaptchaModal,
    setShowCaptchaModal,
    setPendingAction,
    isForgotPasswordSuccess,
    forgotPasswordEmail,
    showDeviceVerification,
    setShowDeviceVerification,
    deviceVerificationData,
    setDeviceVerificationData,
    showMemoryDBWarning,
    emailEnabled,
    formData,
    patchForm,
    formRef,
    modalTitle,
    handleCloseModal,
    handleDismissMemoryDBWarning,
    handleTwoFactorSubmit,
    sendVerificationCode,
    handleCaptchaVerify,
    handleSubmit,
    resetForm,
    close,
    nextPath,
    goToLoginMode,
    dismissRegisterSuccess,
  } = logic

  const showForm = !isRegisterSuccess && !isLoginSuccess && !isForgotPasswordSuccess

  return (
    <ConfigProvider theme={{ primaryColor: AUTH_PRIMARY_COLOR }}>
    <>
      <Modal
        visible={isOpen}
        title={modalTitle}
        onCancel={handleCloseModal}
        maskClosable={!required}
        escToExit={!required}
        footer={null}
        unmountOnExit
        style={{ width: 480, maxWidth: 'calc(100vw - 32px)' }}
        autoFocus={false}
      >
        {showMemoryDBWarning && (
          <Alert
            type="warning"
            closable
            onClose={handleDismissMemoryDBWarning}
            content="检测到您目前使用的是内存数据库，数据可能会丢失。如有需要请配置 MySQL 或 PostgreSQL。"
            style={{ marginBottom: 16 }}
          />
        )}

        {isRegisterSuccess && registerSuccessData && (
          <Result
            status="success"
            title="注册成功"
            subTitle={
              registerSuccessData.activation === false
                ? `激活邮件已发送至 ${registerSuccessData.email}，请查收并激活账号。`
                : `欢迎 ${registerSuccessData.displayName || registerSuccessData.email}，您的账号已创建完成。`
            }
            extra={
              <Space>
                <Button onClick={dismissRegisterSuccess}>继续注册</Button>
                <Button type="primary" onClick={goToLoginMode}>
                  立即登录
                </Button>
              </Space>
            }
          />
        )}

        {isLoginSuccess && loginSuccessData && (
          <Result
            status="success"
            title="登录成功"
            subTitle={`欢迎回来，${
              loginSuccessData.user?.displayName ||
              loginSuccessData.user?.DisplayName ||
              loginSuccessData.displayName ||
              '用户'
            }`}
            extra={
              <Button
                type="primary"
                long
                onClick={() => {
                  if (nextPath) navigate(nextPath)
                  close()
                  resetForm()
                }}
              >
                进入应用
              </Button>
            }
          />
        )}

        {isForgotPasswordSuccess && (
          <Result
            status="success"
            title={t('forgotPassword.successTitle')}
            subTitle={t('forgotPassword.successMessage').replace('{email}', forgotPasswordEmail)}
            extra={
              <Button type="primary" long onClick={goToLoginMode}>
                {t('forgotPassword.backToLogin')}
              </Button>
            }
          />
        )}

        {showForm && (
          <>
            {mode === 'login' && emailEnabled && (
              <LoginTypeSwitch value={loginType} onChange={setLoginType} />
            )}

            <div ref={formRef}>
            <Form
              layout="vertical"
              onSubmit={(_, e) => handleSubmit(e as unknown as React.FormEvent)}
              autoComplete="off"
            >
              {mode === 'login' && (
                <>
                  <Form.Item label="邮箱" required>
                    <Input
                      prefix={<IconEmail />}
                      placeholder="请输入邮箱"
                      value={formData.email}
                      onChange={(v) => patchForm('email', v)}
                      allowClear
                    />
                  </Form.Item>

                  {loginType === 'email' ? (
                    <Form.Item label="验证码" required>
                      <Input
                        prefix={<IconSafe />}
                        placeholder="请输入验证码"
                        value={formData.verificationCode}
                        onChange={(v) => patchForm('verificationCode', v)}
                        addAfter={
                          <SendCodeButton
                            loading={isSendingCode}
                            countdown={countdown}
                            onClick={sendVerificationCode}
                          />
                        }
                      />
                    </Form.Item>
                  ) : (
                    <Form.Item label="密码" required>
                      <Input.Password
                        prefix={<IconLock />}
                        placeholder="请输入密码"
                        value={formData.password}
                        onChange={(v) => patchForm('password', v)}
                      />
                    </Form.Item>
                  )}

                  {showTwoFactorInput && (
                    <>
                      <Form.Item label="两步验证码" required>
                        <Input
                          prefix={<IconSafe />}
                          placeholder="请输入两步验证码"
                          value={twoFactorCode}
                          onChange={setTwoFactorCode}
                        />
                      </Form.Item>
                      <Space direction="vertical" style={{ width: '100%' }}>
                        <Button type="primary" long loading={isLoading} onClick={handleTwoFactorSubmit}>
                          验证登录
                        </Button>
                        <Button long onClick={() => setShowTwoFactorInput(false)}>
                          返回
                        </Button>
                      </Space>
                    </>
                  )}
                </>
              )}

              {mode === 'register' && (
                <>
                  <Row gutter={12}>
                    {emailEnabled && (
                      <Col span={12}>
                        <Form.Item label="用户名" required>
                          <Input
                            prefix={<IconUser />}
                            placeholder="请输入用户名"
                            value={formData.userName}
                            onChange={(v) => patchForm('userName', v)}
                          />
                        </Form.Item>
                      </Col>
                    )}
                    <Col span={emailEnabled ? 12 : 24}>
                      <Form.Item label="显示名" required>
                        <Input
                          prefix={<IconUser />}
                          placeholder="请输入显示名"
                          value={formData.displayName}
                          onChange={(v) => patchForm('displayName', v)}
                        />
                      </Form.Item>
                    </Col>
                  </Row>

                  <Form.Item label="邮箱" required>
                    <Input
                      prefix={<IconEmail />}
                      placeholder="请输入邮箱"
                      value={formData.email}
                      onChange={(v) => patchForm('email', v)}
                      allowClear
                    />
                  </Form.Item>

                  {emailEnabled && (
                    <Form.Item label="验证码" required>
                      <Input
                        prefix={<IconSafe />}
                        placeholder="请输入验证码"
                        value={formData.verificationCode}
                        onChange={(v) => patchForm('verificationCode', v)}
                        addAfter={
                          <SendCodeButton
                            loading={isSendingCode}
                            countdown={countdown}
                            onClick={sendVerificationCode}
                          />
                        }
                      />
                    </Form.Item>
                  )}

                  <Form.Item label="密码" required>
                    <Input.Password
                      prefix={<IconLock />}
                      placeholder="请输入密码（至少8位）"
                      value={formData.password}
                      onChange={(v) => patchForm('password', v)}
                    />
                    <PasswordStrength password={formData.password} />
                  </Form.Item>

                  <Form.Item label="确认密码" required>
                    <Input.Password
                      prefix={<IconLock />}
                      placeholder="请再次输入密码"
                      value={formData.confirmPassword}
                      onChange={(v) => patchForm('confirmPassword', v)}
                    />
                  </Form.Item>
                </>
              )}

              {mode === 'forgot-password' && (
                <>
                  <Typography.Paragraph type="secondary" style={{ marginBottom: 16 }}>
                    {t('forgotPassword.description')}
                  </Typography.Paragraph>
                  <Form.Item label={t('forgotPassword.emailLabel')} required>
                    <Input
                      prefix={<IconEmail />}
                      placeholder={t('forgotPassword.emailPlaceholder')}
                      value={formData.email}
                      onChange={(v) => patchForm('email', v)}
                      allowClear
                    />
                  </Form.Item>
                </>
              )}

              {!showTwoFactorInput && (
                <Form.Item style={{ marginBottom: 8, marginTop: 8 }}>
                  <Button type="primary" htmlType="submit" long loading={isLoading}>
                    {mode === 'login' ? '登录' : mode === 'register' ? '注册' : t('forgotPassword.sendButton')}
                  </Button>
                </Form.Item>
              )}
            </Form>
            </div>

            <div style={{ textAlign: 'center', marginTop: 8 }}>
              {mode === 'login' && (
                <Space split={<Typography.Text type="secondary">·</Typography.Text>}>
                  <Link onClick={() => setMode('register')} style={{ color: AUTH_PRIMARY_COLOR }}>
                    注册账号
                  </Link>
                  <Link onClick={() => setMode('forgot-password')} style={{ color: AUTH_PRIMARY_COLOR }}>
                    忘记密码
                  </Link>
                </Space>
              )}
              {mode === 'register' && (
                <Typography.Text type="secondary">
                  已有账号？<Link onClick={() => setMode('login')} style={{ color: AUTH_PRIMARY_COLOR }}>立即登录</Link>
                </Typography.Text>
              )}
              {mode === 'forgot-password' && (
                <Space direction="vertical" size={4}>
                  <Typography.Text type="secondary">
                    {t('forgotPassword.rememberPassword')}
                    <Link onClick={() => setMode('login')}> {t('forgotPassword.backToLogin')}</Link>
                  </Typography.Text>
                  <Typography.Text type="secondary">
                    <Link onClick={() => setMode('register')}>{t('forgotPassword.createAccount')}</Link>
                  </Typography.Text>
                </Space>
              )}
            </div>
          </>
        )}
      </Modal>

      <CaptchaModal
        isOpen={showCaptchaModal}
        onClose={() => {
          setShowCaptchaModal(false)
          setPendingAction(null)
        }}
        onVerify={handleCaptchaVerify}
        title={mode === 'login' ? '登录验证' : mode === 'register' ? '注册验证' : '重置密码验证'}
      />

      <DeviceVerificationModal
        isOpen={showDeviceVerification}
        onClose={() => {
          setShowDeviceVerification(false)
          setDeviceVerificationData({ email: '', deviceId: '', message: '' })
        }}
        onSuccess={() => {
          setShowDeviceVerification(false)
          setDeviceVerificationData({ email: '', deviceId: '', message: '' })
        }}
        email={deviceVerificationData.email}
        deviceId={deviceVerificationData.deviceId}
        message={deviceVerificationData.message}
      />
    </>
    </ConfigProvider>
  )
}

export default AuthModal
