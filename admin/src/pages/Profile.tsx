import { useState, useEffect } from 'react'
import { Save, User, Mail, Phone, MapPin, Calendar, Edit2, Lock } from 'lucide-react'
import AdminLayout from '@/components/Layout/AdminLayout'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import { useAuthStore } from '@/stores/authStore'
import { getCurrentUser, updateProfile, changePassword, ProfileUpdateRequest, ChangePasswordRequest } from '@/services/adminApi'
import { showAlert } from '@/utils/notification'

const Profile = () => {
  const { user, refreshUserInfo } = useAuthStore()
  const [isEditing, setIsEditing] = useState(false)
  const [isChangingPassword, setIsChangingPassword] = useState(false)
  const [loading, setLoading] = useState(false)
  const [passwordForm, setPasswordForm] = useState({
    oldPassword: '',
    newPassword: '',
    confirmPassword: '',
  })
  
  const [formData, setFormData] = useState<ProfileUpdateRequest>({
    displayName: '',
    email: '',
    phone: '',
    timezone: '',
    gender: '',
    extra: '',
  })

  useEffect(() => {
    fetchUserInfo()
  }, [])

  const fetchUserInfo = async () => {
    try {
      setLoading(true)
      const userData = await getCurrentUser()
      setFormData({
        displayName: userData.displayName || userData.display_name || '',
        email: userData.email || '',
        phone: userData.phone || '',
        timezone: userData.timezone || '',
        gender: userData.gender || '',
        extra: userData.extra || userData.bio || '',
      })
    } catch (error) {
      console.error('获取用户信息失败:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleSave = async () => {
    try {
      setLoading(true)
      await updateProfile(formData)
      await refreshUserInfo()
      setIsEditing(false)
      showAlert('保存成功', 'success')
    } catch (error: any) {
      console.error('更新用户信息失败:', error)
      showAlert(error.msg || '保存失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  const handleCancel = () => {
    fetchUserInfo()
    setIsEditing(false)
  }

  const handleChangePassword = async () => {
    if (!passwordForm.oldPassword || !passwordForm.newPassword) {
      showAlert('请填写完整信息', 'warning')
      return
    }

    if (passwordForm.newPassword !== passwordForm.confirmPassword) {
      showAlert('两次输入的密码不一致', 'warning')
      return
    }

    if (passwordForm.newPassword.length < 6) {
      showAlert('密码长度至少6位', 'warning')
      return
    }

    try {
      setLoading(true)
      const data: ChangePasswordRequest = {
        oldPassword: passwordForm.oldPassword,
        newPassword: passwordForm.newPassword,
      }
      await changePassword(data)
      setIsChangingPassword(false)
      setPasswordForm({ oldPassword: '', newPassword: '', confirmPassword: '' })
      showAlert('密码修改成功', 'success')
    } catch (error: any) {
      console.error('修改密码失败:', error)
      showAlert(error.msg || '修改失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  const currentUser = user || {
    displayName: formData.displayName,
    email: formData.email,
  }

  return (
    <AdminLayout
      title="个人中心"
      description="管理您的个人信息和账户设置"
      actions={
        isEditing ? (
          <div className="flex gap-2">
            <Button variant="outline" onClick={handleCancel} disabled={loading}>
              取消
            </Button>
            <Button variant="primary" leftIcon={<Save className="w-4 h-4" />} onClick={handleSave} disabled={loading}>
              {loading ? '保存中...' : '保存'}
            </Button>
          </div>
        ) : (
          <Button variant="primary" leftIcon={<Edit2 className="w-4 h-4" />} onClick={() => setIsEditing(true)}>
            编辑资料
          </Button>
        )
      }
    >
      <div className="space-y-6">
        {/* 用户信息卡片 */}
        <Card className="p-6">
          <div className="flex flex-col sm:flex-row items-center sm:items-start gap-6">
            <div className="flex-1 text-center sm:text-left">
              <h2 className="text-2xl font-bold text-slate-900 dark:text-white mb-2">
                {formData.displayName || currentUser.email || '管理员'}
              </h2>
              <p className="text-slate-600 dark:text-slate-400 mb-4">
                {formData.email || currentUser.email || 'admin@example.com'}
              </p>
              <div className="flex flex-wrap gap-4 justify-center sm:justify-start">
                <div className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400">
                  <Calendar className="w-4 h-4" />
                  <span>注册时间: {user?.createdAt ? new Date(user.createdAt).toLocaleDateString('zh-CN') : '未知'}</span>
                </div>
                <div className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400">
                  <User className="w-4 h-4" />
                  <span>管理员</span>
                </div>
              </div>
            </div>
          </div>
        </Card>

        {/* 个人信息 */}
        <Card className="p-6">
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-6">
            个人信息
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                显示名称
              </label>
              {isEditing ? (
                <Input
                  value={formData.displayName || ''}
                  onChange={(e) => setFormData({ ...formData, displayName: e.target.value })}
                  leftIcon={<User className="w-4 h-4" />}
                  placeholder="请输入显示名称"
                />
              ) : (
                <div className="flex items-center gap-2 text-slate-900 dark:text-white">
                  <User className="w-4 h-4 text-slate-400" />
                  <span>{formData.displayName || '未设置'}</span>
                </div>
              )}
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                邮箱
              </label>
              {isEditing ? (
                <Input
                  type="email"
                  value={formData.email || ''}
                  onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                  leftIcon={<Mail className="w-4 h-4" />}
                  placeholder="请输入邮箱"
                />
              ) : (
                <div className="flex items-center gap-2 text-slate-900 dark:text-white">
                  <Mail className="w-4 h-4 text-slate-400" />
                  <span>{formData.email || '未设置'}</span>
                </div>
              )}
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                手机号
              </label>
              {isEditing ? (
                <Input
                  type="tel"
                  value={formData.phone || ''}
                  onChange={(e) => setFormData({ ...formData, phone: e.target.value })}
                  leftIcon={<Phone className="w-4 h-4" />}
                  placeholder="请输入手机号"
                />
              ) : (
                <div className="flex items-center gap-2 text-slate-900 dark:text-white">
                  <Phone className="w-4 h-4 text-slate-400" />
                  <span>{formData.phone || '未设置'}</span>
                </div>
              )}
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                时区
              </label>
              {isEditing ? (
                <Input
                  value={formData.timezone || ''}
                  onChange={(e) => setFormData({ ...formData, timezone: e.target.value })}
                  placeholder="例如: Asia/Shanghai"
                />
              ) : (
                <div className="flex items-center gap-2 text-slate-900 dark:text-white">
                  <MapPin className="w-4 h-4 text-slate-400" />
                  <span>{formData.timezone || '未设置'}</span>
                </div>
              )}
            </div>
          </div>
          <div className="mt-6">
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              个人简介
            </label>
            {isEditing ? (
              <textarea
                value={formData.extra || ''}
                onChange={(e) => setFormData({ ...formData, extra: e.target.value })}
                className="w-full px-4 py-3 rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                rows={4}
                placeholder="介绍一下自己..."
              />
            ) : (
              <p className="text-slate-600 dark:text-slate-400">
                {formData.extra || '暂无简介'}
              </p>
            )}
          </div>
        </Card>

        {/* 账户安全 */}
        <Card className="p-6">
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-6">
            账户安全
          </h3>
          <div className="space-y-4">
            <div className="p-4 rounded-lg border border-slate-200 dark:border-slate-800">
              <div className="flex items-center justify-between mb-4">
                <div>
                  <p className="font-medium text-slate-900 dark:text-white">修改密码</p>
                  <p className="text-sm text-slate-500 dark:text-slate-400">定期更新密码以保护账户安全</p>
                </div>
                <Button 
                  variant="outline" 
                  size="sm"
                  onClick={() => setIsChangingPassword(!isChangingPassword)}
                >
                  {isChangingPassword ? '取消' : '修改'}
                </Button>
              </div>
              {isChangingPassword && (
                <div className="space-y-4 pt-4 border-t border-slate-200 dark:border-slate-800">
                  <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                      当前密码
                    </label>
                    <Input
                      type="password"
                      value={passwordForm.oldPassword}
                      onChange={(e) => setPasswordForm({ ...passwordForm, oldPassword: e.target.value })}
                      leftIcon={<Lock className="w-4 h-4" />}
                      placeholder="请输入当前密码"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                      新密码
                    </label>
                    <Input
                      type="password"
                      value={passwordForm.newPassword}
                      onChange={(e) => setPasswordForm({ ...passwordForm, newPassword: e.target.value })}
                      leftIcon={<Lock className="w-4 h-4" />}
                      placeholder="请输入新密码（至少6位）"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                      确认新密码
                    </label>
                    <Input
                      type="password"
                      value={passwordForm.confirmPassword}
                      onChange={(e) => setPasswordForm({ ...passwordForm, confirmPassword: e.target.value })}
                      leftIcon={<Lock className="w-4 h-4" />}
                      placeholder="请再次输入新密码"
                    />
                  </div>
                  <Button 
                    variant="primary" 
                    onClick={handleChangePassword}
                    disabled={loading}
                    className="w-full"
                  >
                    {loading ? '修改中...' : '确认修改'}
                  </Button>
                </div>
              )}
            </div>
          </div>
        </Card>
      </div>
    </AdminLayout>
  )
}

export default Profile
