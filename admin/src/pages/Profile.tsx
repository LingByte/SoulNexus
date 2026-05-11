// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 个人中心：Arco Form + Form.Item，分卡片展示基本信息 / 通知偏好 / 安全密码。
import { useEffect, useState } from 'react'
import {
  Card,
  Form,
  Input,
  Button,
  Switch,
  Avatar,
  Modal,
  Message,
  Space,
} from '@arco-design/web-react'
import { Save, Edit2, Lock, Calendar, User as UserIcon } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import { useAuthStore } from '@/stores/authStore'
import {
  getCurrentUser,
  updateProfile,
  changePassword,
  updateNotificationSettings,
  type ProfileUpdateRequest,
  type ChangePasswordRequest,
} from '@/services/adminApi'

const FormItem = Form.Item

const Profile = () => {
  const { user, refreshUserInfo } = useAuthStore()
  const [isEditing, setIsEditing] = useState(false)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)

  const [profileForm] = Form.useForm<ProfileUpdateRequest>()
  const [pwModalOpen, setPwModalOpen] = useState(false)
  const [pwSaving, setPwSaving] = useState(false)
  const [pwForm] = Form.useForm<{ oldPassword: string; newPassword: string; confirmPassword: string }>()

  const fetchUserInfo = async () => {
    setLoading(true)
    try {
      const userData = await getCurrentUser()
      const pr = (userData as any)?.profile || {}
      profileForm.setFieldsValue({
        displayName: userData.displayName || (userData as any).display_name || pr.displayName || '',
        email: userData.email || '',
        phone: userData.phone || pr.phone || '',
        timezone: userData.timezone || pr.timezone || '',
        gender: (userData as any).gender || pr.gender || '',
        city: (userData as any).city || pr.city || '',
        region: (userData as any).region || pr.region || '',
        extra: (userData as any).extra || (userData as any).bio || pr.extra || '',
        avatar: (userData as any).avatar || pr.avatar || '',
        emailNotifications: (userData as any).emailNotifications ?? pr.emailNotifications ?? true,
        pushNotifications: (userData as any).pushNotifications ?? pr.pushNotifications ?? true,
      } as any)
    } catch (e: any) {
      Message.error(`获取用户信息失败：${e?.msg || e?.message || e}`)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchUserInfo()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const handleSave = async () => {
    try {
      const values = await profileForm.validate()
      setSaving(true)
      await updateProfile(values)
      await updateNotificationSettings({
        email_notifications: (values as any).emailNotifications,
        push_notifications: (values as any).pushNotifications,
      })
      await refreshUserInfo()
      setIsEditing(false)
      Message.success('保存成功')
    } catch (e: any) {
      if (e?.message || e?.msg) Message.error(`保存失败：${e?.msg || e?.message}`)
    } finally {
      setSaving(false)
    }
  }

  const handleCancel = () => {
    fetchUserInfo()
    setIsEditing(false)
  }

  const submitChangePassword = async () => {
    try {
      const v = await pwForm.validate()
      if (v.newPassword !== v.confirmPassword) {
        Message.error('两次输入的密码不一致')
        return
      }
      setPwSaving(true)
      const data: ChangePasswordRequest = {
        oldPassword: v.oldPassword,
        newPassword: v.newPassword,
      }
      await changePassword(data)
      pwForm.resetFields()
      setPwModalOpen(false)
      Message.success('密码修改成功')
    } catch (e: any) {
      if (e?.message || e?.msg) Message.error(`修改失败：${e?.msg || e?.message}`)
    } finally {
      setPwSaving(false)
    }
  }

  const avatar = profileForm.getFieldValue('avatar') || (user as any)?.avatar || ''
  const displayName = profileForm.getFieldValue('displayName') || user?.displayName || user?.email || '管理员'
  const email = profileForm.getFieldValue('email') || user?.email || ''

  const headerActions = isEditing ? (
    <Space>
      <Button onClick={handleCancel}>取消</Button>
      <Button type="primary" loading={saving} onClick={handleSave}>
        <span className="inline-flex items-center gap-1"><Save size={14} /> 保存</span>
      </Button>
    </Space>
  ) : (
    <Button type="primary" onClick={() => setIsEditing(true)}>
      <span className="inline-flex items-center gap-1"><Edit2 size={14} /> 编辑</span>
    </Button>
  )

  return (
    <div className="space-y-4">
      <PageHeader title="个人中心" description="管理您的个人信息和账户设置。" actions={headerActions} />

      <Card>
        <div className="flex flex-col sm:flex-row items-center sm:items-start gap-6 py-2">
          <Avatar size={80} style={{ backgroundColor: '#4ECDC4' }}>
            {avatar ? <img src={avatar} alt="avatar" /> : (displayName || 'A').slice(0, 1).toUpperCase()}
          </Avatar>
          <div className="flex-1 text-center sm:text-left">
            <div className="text-2xl font-semibold text-[var(--color-text-1)] mb-1">{displayName}</div>
            <div className="text-[var(--color-text-3)] mb-3">{email}</div>
            <div className="flex flex-wrap gap-4 text-sm text-[var(--color-text-3)] justify-center sm:justify-start">
              <span className="inline-flex items-center gap-1">
                <Calendar size={14} />
                注册时间：{(user as any)?.createdAt ? new Date((user as any).createdAt).toLocaleDateString('zh-CN') : '未知'}
              </span>
              <span className="inline-flex items-center gap-1">
                <UserIcon size={14} /> {user?.role || '管理员'}
              </span>
            </div>
          </div>
        </div>
      </Card>

      <Form
        form={profileForm}
        layout="vertical"
        autoComplete="off"
        disabled={!isEditing}
        loading={loading}
      >
        <Card title="个人信息">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-x-4">
            <FormItem label="显示名称" field="displayName">
              <Input placeholder="请输入显示名称" />
            </FormItem>
            <FormItem label="邮箱" field="email" rules={[{ type: 'email', message: '请输入有效邮箱' }]}>
              <Input placeholder="user@example.com" />
            </FormItem>
            <FormItem label="手机号" field="phone">
              <Input placeholder="请输入手机号" />
            </FormItem>
            <FormItem label="头像 URL" field="avatar">
              <Input placeholder="https://example.com/avatar.png" />
            </FormItem>
            <FormItem label="城市" field="city"><Input /></FormItem>
            <FormItem label="地区" field="region"><Input /></FormItem>
            <FormItem label="性别" field="gender"><Input placeholder="male / female / other" /></FormItem>
            <FormItem label="时区" field="timezone"><Input placeholder="Asia/Shanghai" /></FormItem>
          </div>
          <FormItem label="个人简介" field="extra">
            <Input.TextArea rows={4} placeholder="介绍一下自己..." />
          </FormItem>
        </Card>

        <Card title="通知偏好" style={{ marginTop: 16 }}>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <FormItem label="邮件通知" field="emailNotifications" triggerPropName="checked"><Switch /></FormItem>
            <FormItem label="推送通知" field="pushNotifications" triggerPropName="checked"><Switch /></FormItem>
          </div>
        </Card>
      </Form>

      <Card title="账户安全">
        <div className="flex items-center justify-between p-3 rounded-md border border-[var(--color-border-2)]">
          <div>
            <div className="font-medium text-[var(--color-text-1)]">修改密码</div>
            <div className="text-sm text-[var(--color-text-3)]">定期更新密码以保护账户安全。</div>
          </div>
          <Button onClick={() => { pwForm.resetFields(); setPwModalOpen(true) }}>
            <span className="inline-flex items-center gap-1"><Lock size={14} /> 修改</span>
          </Button>
        </div>
      </Card>

      <Modal
        title="修改密码"
        visible={pwModalOpen}
        onCancel={() => setPwModalOpen(false)}
        onOk={submitChangePassword}
        confirmLoading={pwSaving}
        okText="确认修改"
        cancelText="取消"
        autoFocus={false}
      >
        <Form form={pwForm} layout="vertical" autoComplete="off">
          <FormItem label="当前密码" field="oldPassword" rules={[{ required: true, message: '请输入当前密码' }]}>
            <Input.Password placeholder="当前密码" />
          </FormItem>
          <FormItem
            label="新密码"
            field="newPassword"
            rules={[
              { required: true, message: '请输入新密码' },
              { minLength: 6, message: '密码长度至少 6 位' },
            ]}
          >
            <Input.Password placeholder="新密码（至少 6 位）" />
          </FormItem>
          <FormItem
            label="确认新密码"
            field="confirmPassword"
            rules={[{ required: true, message: '请再次输入新密码' }]}
          >
            <Input.Password placeholder="再次输入新密码" />
          </FormItem>
        </Form>
      </Modal>
    </div>
  )
}

export default Profile
