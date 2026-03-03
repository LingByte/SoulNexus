import { useState, useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { motion } from 'framer-motion'
import { Lock, Eye, EyeOff, CheckCircle, AlertTriangle } from 'lucide-react'
import Button from '../components/UI/Button'
import Input from '../components/UI/Input'
import Card, { CardContent, CardHeader, CardTitle } from '../components/UI/Card'
import PasswordStrength from '../components/Auth/PasswordStrength'
import { resetPasswordConfirm } from '../api/auth'
import { showAlert } from '../utils/notification'
import { useI18nStore } from '../stores/i18nStore'

const ResetPassword = () => {
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token')

  const [formData, setFormData] = useState({
    password: '',
    confirmPassword: ''
  })
  const [showPassword, setShowPassword] = useState(false)
  const [showConfirmPassword, setShowConfirmPassword] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [isSuccess, setIsSuccess] = useState(false)
  const [isTokenValid, setIsTokenValid] = useState(true)

  useEffect(() => {
    if (!token) {
      setIsTokenValid(false)
      showAlert(t('resetPassword.invalidLink'), 'error', t('resetPassword.invalidLinkTitle'))
    }
  }, [token, t])

  const handleInputChange = (field: string, value: string) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!token) {
      showAlert(t('resetPassword.invalidToken'), 'error', t('resetPassword.validationFailed'))
      return
    }

    if (formData.password !== formData.confirmPassword) {
      showAlert(t('resetPassword.passwordMismatch'), 'error', t('resetPassword.validationFailed'))
      return
    }

    if (formData.password.length < 6) {
      showAlert(t('resetPassword.passwordTooShort'), 'error', t('resetPassword.validationFailed'))
      return
    }

    setIsLoading(true)
    try {
      const response = await resetPasswordConfirm(token, formData.password)
      
      if (response.code === 200) {
        setIsSuccess(true)
        showAlert(t('resetPassword.resetSuccess'), 'success', t('resetPassword.resetSuccessTitle'))
        
        // 3秒后跳转到登录页面
        setTimeout(() => {
          navigate('/', { replace: true })
        }, 3000)
      } else {
        throw new Error(response.msg || t('resetPassword.resetFailed'))
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || t('resetPassword.resetError'), 'error', t('resetPassword.resetFailedTitle'))
    } finally {
      setIsLoading(false)
    }
  }

  if (!isTokenValid) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900 flex items-center justify-center p-4">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="w-full max-w-md"
        >
          <Card className="shadow-xl border-0 bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm">
            <CardHeader className="text-center pb-6">
              <div className="w-16 h-16 bg-red-100 dark:bg-red-900/20 rounded-full flex items-center justify-center mx-auto mb-4">
                <AlertTriangle className="w-8 h-8 text-red-600 dark:text-red-400" />
              </div>
              <CardTitle className="text-2xl font-bold text-gray-900 dark:text-white">
                {t('resetPassword.invalidTitle')}
              </CardTitle>
            </CardHeader>
            <CardContent className="text-center">
              <p className="text-gray-600 dark:text-gray-400 mb-6">
                {t('resetPassword.invalidMessage')}
              </p>
              <Button
                variant="primary"
                onClick={() => navigate('/', { replace: true })}
                className="w-full"
              >
                {t('resetPassword.backToHome')}
              </Button>
            </CardContent>
          </Card>
        </motion.div>
      </div>
    )
  }

  if (isSuccess) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900 flex items-center justify-center p-4">
        <motion.div
          initial={{ opacity: 0, scale: 0.9 }}
          animate={{ opacity: 1, scale: 1 }}
          className="w-full max-w-md"
        >
          <Card className="shadow-xl border-0 bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm">
            <CardHeader className="text-center pb-6">
              <motion.div
                initial={{ scale: 0 }}
                animate={{ scale: 1 }}
                transition={{ delay: 0.2, type: "spring", stiffness: 200 }}
                className="w-16 h-16 bg-green-100 dark:bg-green-900/20 rounded-full flex items-center justify-center mx-auto mb-4"
              >
                <CheckCircle className="w-8 h-8 text-green-600 dark:text-green-400" />
              </motion.div>
              <CardTitle className="text-2xl font-bold text-gray-900 dark:text-white">
                {t('resetPassword.successTitle')}
              </CardTitle>
            </CardHeader>
            <CardContent className="text-center">
              <p className="text-gray-600 dark:text-gray-400 mb-6">
                {t('resetPassword.successMessage')}
              </p>
              <Button
                variant="primary"
                onClick={() => navigate('/', { replace: true })}
                className="w-full"
              >
                {t('resetPassword.loginNow')}
              </Button>
            </CardContent>
          </Card>
        </motion.div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900 flex items-center justify-center p-4">
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className="w-full max-w-md"
      >
        <Card className="shadow-xl border-0 bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm">
          <CardHeader className="text-center pb-6">
            <div className="w-16 h-16 bg-blue-100 dark:bg-blue-900/20 rounded-full flex items-center justify-center mx-auto mb-4">
              <Lock className="w-8 h-8 text-blue-600 dark:text-blue-400" />
            </div>
            <CardTitle className="text-2xl font-bold text-gray-900 dark:text-white">
              {t('resetPassword.title')}
            </CardTitle>
            <p className="text-gray-600 dark:text-gray-400 mt-2">
              {t('resetPassword.subtitle')}
            </p>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-6">
              <div>
                <Input
                  label={t('resetPassword.newPassword')}
                  type={showPassword ? 'text' : 'password'}
                  placeholder={t('resetPassword.newPasswordPlaceholder')}
                  value={formData.password}
                  onChange={(e) => handleInputChange('password', e.target.value)}
                  leftIcon={<Lock className="w-5 h-5" />}
                  rightIcon={
                    <button
                      type="button"
                      onClick={() => setShowPassword(!showPassword)}
                      className="text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-300"
                    >
                      {showPassword ? <EyeOff className="w-5 h-5" /> : <Eye className="w-5 h-5" />}
                    </button>
                  }
                  required
                />
                <PasswordStrength password={formData.password} />
              </div>

              <Input
                label={t('resetPassword.confirmPassword')}
                type={showConfirmPassword ? 'text' : 'password'}
                placeholder={t('resetPassword.confirmPasswordPlaceholder')}
                value={formData.confirmPassword}
                onChange={(e) => handleInputChange('confirmPassword', e.target.value)}
                leftIcon={<Lock className="w-5 h-5" />}
                rightIcon={
                  <button
                    type="button"
                    onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                    className="text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-300"
                  >
                    {showConfirmPassword ? <EyeOff className="w-5 h-5" /> : <Eye className="w-5 h-5" />}
                  </button>
                }
                required
              />

              <Button
                type="submit"
                variant="primary"
                className="w-full"
                disabled={isLoading}
              >
                {isLoading ? t('resetPassword.resetting') : t('resetPassword.resetButton')}
              </Button>
            </form>
          </CardContent>
        </Card>
      </motion.div>
    </div>
  )
}

export default ResetPassword