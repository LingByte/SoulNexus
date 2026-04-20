import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import { showAlert } from '@/utils/notification'
import {
  getAccountDeletionEligibility,
  sendAccountDeletionEmailCode,
  requestAccountDeletion,
  unbindGithubAccount,
  unbindWechatAccount,
} from '@/api/accountDeletion'
import { getUserServiceLoginPageURL } from '@/config/apiConfig'

const AccountDeletionRequest = () => {
  const [warnings, setWarnings] = useState<string[]>([])
  const [reasons, setReasons] = useState<string[]>([])
  const [githubBound, setGithubBound] = useState(false)
  const [wechatBound, setWechatBound] = useState(false)
  const [eligible, setEligible] = useState(false)
  const [password, setPassword] = useState('')
  const [emailCode, setEmailCode] = useState('')
  const [ack, setAck] = useState(false)
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [sendingCode, setSendingCode] = useState(false)

  const load = async () => {
    setLoading(true)
    try {
      const res = await getAccountDeletionEligibility()
      if (res.code !== 200 || !res.data) {
        throw new Error(res.msg || '加载失败')
      }
      const d = res.data as any
      setWarnings(Array.isArray(d.warnings) ? d.warnings : [])
      setReasons(Array.isArray(d.reasons) ? d.reasons : [])
      setGithubBound(!!d.githubBound)
      setWechatBound(!!d.wechatBound)
      setEligible(!!d.eligible)
      if (d.deletionPending) {
        window.location.assign(getUserServiceLoginPageURL())
      }
    } catch (e: any) {
      showAlert(e?.msg || e?.message || '加载失败', 'error', '错误')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const sendCode = async () => {
    setSendingCode(true)
    try {
      const res = await sendAccountDeletionEmailCode()
      if (res.code !== 200) throw new Error(res.msg || '发送失败')
      showAlert('验证码已发送到邮箱', 'success', '已发送')
    } catch (e: any) {
      showAlert(e?.msg || e?.message || '发送失败', 'error', '错误')
    } finally {
      setSendingCode(false)
    }
  }

  const unbindGh = async () => {
    try {
      const res = await unbindGithubAccount()
      if (res.code !== 200) throw new Error(res.msg || '解绑失败')
      showAlert('GitHub 已解绑', 'success', '完成')
      await load()
    } catch (e: any) {
      showAlert(e?.msg || e?.message || '解绑失败', 'error', '错误')
    }
  }

  const unbindWx = async () => {
    try {
      const res = await unbindWechatAccount()
      if (res.code !== 200) throw new Error(res.msg || '解绑失败')
      showAlert('微信已解绑', 'success', '完成')
      await load()
    } catch (e: any) {
      showAlert(e?.msg || e?.message || '解绑失败', 'error', '错误')
    }
  }

  const submit = async () => {
    if (!ack) {
      showAlert('请勾选确认已了解注销后果', 'warning', '提示')
      return
    }
    if (!password || !emailCode) {
      showAlert('请填写密码与邮箱验证码', 'warning', '提示')
      return
    }
    if (!eligible) {
      showAlert('当前不满足注销条件，请先处理上述提示', 'warning', '提示')
      return
    }
    setSubmitting(true)
    try {
      const res = await requestAccountDeletion({
        password,
        emailCode,
        acknowledgeConsequences: true,
      })
      if (res.code !== 200) throw new Error(res.msg || '申请失败')
      showAlert('已进入注销冷静期，将前往登录页', 'success', '已提交')
      try {
        localStorage.removeItem('auth_token')
        localStorage.removeItem('refresh_token')
      } catch {
        //
      }
      window.location.assign(getUserServiceLoginPageURL())
    } catch (e: any) {
      showAlert(e?.msg || e?.message || '申请失败', 'error', '错误')
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center text-gray-600 dark:text-gray-300">
        加载中…
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 py-10 px-4">
      <div className="max-w-lg mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-xl font-semibold text-gray-900 dark:text-white">注销账号</h1>
          <Link to="/profile" className="text-sm text-blue-600 dark:text-blue-400 hover:underline">
            返回资料
          </Link>
        </div>

        <Card>
          <div className="p-6 space-y-4">
            <p className="text-sm text-gray-600 dark:text-gray-400">
              注销前有 <strong>冷静期</strong>，期间将无法正常使用产品，仅可通过「撤销注销」恢复。冷静期结束后账号将被永久注销。
            </p>
            {warnings.length > 0 && (
              <ul className="list-disc pl-5 text-sm text-amber-800 dark:text-amber-200 space-y-1 bg-amber-50 dark:bg-amber-900/20 rounded-lg p-3">
                {warnings.map((w, i) => (
                  <li key={i}>{w}</li>
                ))}
              </ul>
            )}
            {(githubBound || wechatBound) && (
              <div className="rounded-lg border border-gray-200 dark:border-gray-700 p-3 text-sm space-y-2">
                <p className="font-medium text-gray-900 dark:text-white">第三方登录</p>
                <p className="text-gray-600 dark:text-gray-400">需先解绑 GitHub / 微信后再申请注销。</p>
                <div className="flex flex-wrap gap-2">
                  {githubBound && (
                    <Button type="button" size="sm" variant="outline" onClick={unbindGh}>
                      解绑 GitHub
                    </Button>
                  )}
                  {wechatBound && (
                    <Button type="button" size="sm" variant="outline" onClick={unbindWx}>
                      解绑微信
                    </Button>
                  )}
                </div>
              </div>
            )}
            {reasons.length > 0 && (
              <ul className="list-disc pl-5 text-sm text-red-700 dark:text-red-300 space-y-1">
                {reasons.map((r, i) => (
                  <li key={i}>{r}</li>
                ))}
              </ul>
            )}

            <div className="space-y-3 pt-2">
              <Input
                label="登录密码"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="用于核验身份"
              />
              <div className="flex gap-2 items-end">
                <div className="flex-1">
                  <Input
                    label="邮箱验证码"
                    value={emailCode}
                    onChange={(e) => setEmailCode(e.target.value)}
                    placeholder="6 位数字"
                  />
                </div>
                <Button type="button" variant="outline" onClick={sendCode} disabled={sendingCode}>
                  {sendingCode ? '发送中…' : '发送验证码'}
                </Button>
              </div>
              <label className="flex items-start gap-2 text-sm text-gray-700 dark:text-gray-300 cursor-pointer">
                <input
                  type="checkbox"
                  className="mt-1"
                  checked={ack}
                  onChange={(e) => setAck(e.target.checked)}
                />
                <span>我已知晓：数据不可恢复、权益清零、无法通过原账号找回。</span>
              </label>
              <Button type="button" variant="destructive" onClick={submit} disabled={submitting || !eligible}>
                {submitting ? '提交中…' : '确认申请注销'}
              </Button>
            </div>
          </div>
        </Card>
      </div>
    </div>
  )
}

export default AccountDeletionRequest
