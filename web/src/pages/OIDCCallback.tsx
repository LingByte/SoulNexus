import { useEffect, useRef } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useAuthStore } from '@/stores/authStore'
import { showAlert } from '@/utils/notification'
import { exchangeOIDCCode, resolveNextPathFromState } from '@/utils/sso'

const OIDCCallback = () => {
  const [searchParams] = useSearchParams()
  const { login } = useAuthStore()
  const startedRef = useRef(false)

  useEffect(() => {
    if (startedRef.current) {
      return
    }
    startedRef.current = true

    const run = async () => {
      const code = searchParams.get('code')
      const state = searchParams.get('state')
      const error = searchParams.get('error')
      const errorDescription = searchParams.get('error_description')

      if (error) {
        showAlert(errorDescription || error, 'error', 'SSO 登录失败')
        window.location.replace('/')
        return
      }

      if (!code) {
        showAlert('缺少授权码，请重新登录', 'error', 'SSO 登录失败')
        window.location.replace('/')
        return
      }

      try {
        const tokenData = await exchangeOIDCCode(code)
        const ok = await login(tokenData.access_token)
        if (!ok) {
          throw new Error('登录态初始化失败')
        }
        window.location.replace(resolveNextPathFromState(state))
      } catch (err: any) {
        showAlert(err?.msg || err?.message || 'SSO 换取令牌失败', 'error', 'SSO 登录失败')
        window.location.replace('/')
      }
    }

    run()

    return undefined
  }, [login, searchParams])

  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="text-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
        <p className="text-sm text-gray-600 dark:text-gray-400">SSO 登录处理中...</p>
      </div>
    </div>
  )
}

export default OIDCCallback
