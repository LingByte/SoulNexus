// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 通知渠道独立编辑页（邮件 / 短信）。
// 路由：/notification-channels/new?type=email|sms、/notification-channels/:id/edit
import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { Button, Input, InputNumber, Switch, Select, Space, Card, Message } from '@arco-design/web-react'
import { ArrowLeft, Save } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import {
  getNotificationChannel,
  createNotificationChannel,
  updateNotificationChannel,
  type UpsertChannelReq,
  type EmailChannelForm,
  type SMSChannelForm,
} from '@/services/notificationsApi'

const Option = Select.Option

type ChannelType = 'email' | 'sms'

interface SMSProviderMeta {
  value: string
  label: string
  fields: Array<{ key: string; label: string; secret?: boolean; placeholder?: string }>
}

const SMS_PROVIDERS: SMSProviderMeta[] = [
  { value: 'aliyun', label: '阿里云短信', fields: [
    { key: 'accessKeyId', label: 'AccessKeyId' },
    { key: 'accessKeySecret', label: 'AccessKeySecret', secret: true },
    { key: 'signName', label: '签名' },
    { key: 'regionId', label: 'Region', placeholder: 'cn-hangzhou' },
  ]},
  { value: 'tencent', label: '腾讯云短信', fields: [
    { key: 'secretId', label: 'SecretId' },
    { key: 'secretKey', label: 'SecretKey', secret: true },
    { key: 'sdkAppId', label: 'SdkAppId' },
    { key: 'sign', label: '签名' },
    { key: 'region', label: 'Region', placeholder: 'ap-guangzhou' },
  ]},
  { value: 'huawei', label: '华为云短信', fields: [
    { key: 'appKey', label: 'AppKey' },
    { key: 'appSecret', label: 'AppSecret', secret: true },
    { key: 'sender', label: 'Sender' },
    { key: 'signature', label: '签名' },
    { key: 'endpoint', label: 'Endpoint' },
  ]},
  { value: 'baidu', label: '百度云短信', fields: [
    { key: 'accessKey', label: 'AccessKey' },
    { key: 'secretKey', label: 'SecretKey', secret: true },
    { key: 'signatureId', label: 'SignatureId' },
    { key: 'endpoint', label: 'Endpoint', placeholder: 'http://smsv3.bj.baidubce.com' },
  ]},
  { value: 'yunpian', label: '云片', fields: [
    { key: 'apiKey', label: 'ApiKey', secret: true },
  ]},
  { value: 'twilio', label: 'Twilio', fields: [
    { key: 'accountSid', label: 'AccountSid' },
    { key: 'token', label: 'AuthToken', secret: true },
    { key: 'from', label: 'From 号码' },
  ]},
  { value: 'submail', label: 'Submail', fields: [
    { key: 'appId', label: 'AppId' },
    { key: 'appKey', label: 'AppKey', secret: true },
  ]},
  { value: 'huyi', label: '互亿无线', fields: [
    { key: 'account', label: 'Account' },
    { key: 'apiKey', label: 'ApiKey', secret: true },
  ]},
  { value: 'luosimao', label: '螺丝帽', fields: [
    { key: 'apiKey', label: 'ApiKey', secret: true },
  ]},
  { value: 'juhe', label: '聚合数据', fields: [
    { key: 'appKey', label: 'AppKey', secret: true },
    { key: 'tplId', label: '默认模板 ID' },
  ]},
  { value: 'chuanglan', label: '创蓝', fields: [
    { key: 'account', label: 'Account' },
    { key: 'password', label: 'Password', secret: true },
  ]},
  { value: 'netease', label: '网易云信', fields: [
    { key: 'appKey', label: 'AppKey' },
    { key: 'appSecret', label: 'AppSecret', secret: true },
  ]},
  { value: 'rongcloud', label: '融云', fields: [
    { key: 'appKey', label: 'AppKey' },
    { key: 'appSecret', label: 'AppSecret', secret: true },
  ]},
  { value: 'yuntongxun', label: '容联云通讯', fields: [
    { key: 'accountSid', label: 'AccountSid' },
    { key: 'authToken', label: 'AuthToken', secret: true },
    { key: 'appId', label: 'AppId' },
  ]},
  { value: 'tiniyo', label: 'Tiniyo', fields: [
    { key: 'accountSid', label: 'AccountSid' },
    { key: 'authToken', label: 'AuthToken', secret: true },
  ]},
  { value: 'ucloud', label: 'UCloud', fields: [
    { key: 'publicKey', label: 'PublicKey' },
    { key: 'privateKey', label: 'PrivateKey', secret: true },
    { key: 'sigContent', label: '签名' },
  ]},
]

const Field = ({ label, required, children }: { label: string; required?: boolean; children: React.ReactNode }) => (
  <div>
    <label className="block text-sm font-medium text-[var(--color-text-2)] mb-1">
      {label}
      {required && <span className="text-red-500 ml-0.5">*</span>}
    </label>
    {children}
  </div>
)

const NotificationChannelEditPage = () => {
  const params = useParams<{ id?: string }>()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const idNum = params.id ? Number(params.id) : 0
  const isEdit = idNum > 0
  const initialType = (searchParams.get('type') as ChannelType) || 'email'

  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [channelName, setChannelName] = useState('')
  const [emailForm, setEmailForm] = useState<EmailChannelForm | undefined>()
  const [smsForm, setSmsForm] = useState<SMSChannelForm | undefined>()
  const [form, setForm] = useState<UpsertChannelReq>({
    channelType: initialType,
    name: '',
    sortOrder: 0,
    enabled: true,
    remark: '',
    driver: initialType === 'email' ? 'smtp' : undefined,
    smsProvider: initialType === 'sms' ? 'aliyun' : undefined,
    smsConfig: {},
    smtpPort: 587,
  })

  useEffect(() => {
    if (!isEdit) return
    setLoading(true)
    getNotificationChannel(idNum)
      .then((detail) => {
        const c = detail.channel
        setChannelName(c.name)
        setEmailForm(detail.emailForm)
        setSmsForm(detail.smsForm)
        const next: UpsertChannelReq = {
          channelType: c.type,
          name: c.name,
          sortOrder: c.sortOrder,
          enabled: c.enabled,
          remark: c.remark || '',
        }
        if (c.type === 'email' && detail.emailForm) {
          next.driver = detail.emailForm.driver
          next.smtpHost = detail.emailForm.smtpHost
          next.smtpPort = detail.emailForm.smtpPort
          next.smtpUsername = detail.emailForm.smtpUsername
          next.smtpFrom = detail.emailForm.smtpFrom
          next.fromDisplayName = detail.emailForm.fromDisplayName
          next.sendcloudApiUser = detail.emailForm.sendcloudApiUser
          next.sendcloudFrom = detail.emailForm.sendcloudFrom
        } else if (c.type === 'sms' && detail.smsForm) {
          next.smsProvider = detail.smsForm.provider
          next.smsConfig = { ...detail.smsForm.config }
        }
        setForm(next)
      })
      .catch((e) => Message.error(`读取渠道失败：${e?.msg || e?.message || e}`))
      .finally(() => setLoading(false))
  }, [idNum, isEdit])

  const update = <K extends keyof UpsertChannelReq>(k: K, v: UpsertChannelReq[K]) => {
    setForm((prev) => ({ ...prev, [k]: v }))
  }

  const updateSmsConfig = (key: string, val: string) => {
    setForm((prev) => ({ ...prev, smsConfig: { ...(prev.smsConfig || {}), [key]: val } }))
  }

  const currentSMSProvider = useMemo(
    () => SMS_PROVIDERS.find((p) => p.value === form.smsProvider),
    [form.smsProvider],
  )

  const handleSave = async () => {
    if (!form.name?.trim()) {
      Message.error('请填写渠道名称')
      return
    }
    setSaving(true)
    try {
      if (isEdit) {
        await updateNotificationChannel(idNum, form)
        Message.success('更新成功')
      } else {
        await createNotificationChannel(form)
        Message.success('创建成功')
      }
      navigate('/notification-channels')
    } catch (e: any) {
      Message.error(`保存失败：${e?.msg || e?.message || e}`)
    } finally {
      setSaving(false)
    }
  }

  const isEmail = form.channelType === 'email'

  return (
    <div className="space-y-4">
      <PageHeader
        title={isEdit ? `编辑渠道：${channelName}` : `新建${isEmail ? '邮件' : '短信'}渠道`}
        description="配置发送供应商凭据。已设置的密钥字段在编辑时留空表示保持原值。"
        actions={
          <Space>
            <Button onClick={() => navigate('/notification-channels')}>
              <span className="inline-flex items-center gap-1"><ArrowLeft size={14} /> 返回列表</span>
            </Button>
            <Button type="primary" loading={saving} onClick={handleSave}>
              <span className="inline-flex items-center gap-1"><Save size={14} /> 保存</span>
            </Button>
          </Space>
        }
      />

      {loading ? (
        <Card>
          <div className="p-8 text-center text-[var(--color-text-3)]">加载中...</div>
        </Card>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <Card title="基本信息" bordered>
            <div className="space-y-3">
              <Field label="渠道类型">
                <Select
                  value={form.channelType}
                  disabled={isEdit}
                  onChange={(v) => {
                    update('channelType', v)
                    if (v === 'email') update('driver', form.driver || 'smtp')
                    if (v === 'sms') update('smsProvider', form.smsProvider || 'aliyun')
                  }}
                >
                  <Option value="email">邮件</Option>
                  <Option value="sms">短信</Option>
                </Select>
              </Field>
              <Field label="渠道名称" required>
                <Input value={form.name} onChange={(v) => update('name', v)} placeholder="主邮箱 / 备用 SMS 等" />
              </Field>
              <Field label="排序权重">
                <InputNumber value={form.sortOrder ?? 0} onChange={(v) => update('sortOrder', Number(v ?? 0))} />
              </Field>
              <Field label="备注">
                <Input value={form.remark || ''} onChange={(v) => update('remark', v)} />
              </Field>
              <Field label="启用">
                <Switch checked={form.enabled !== false} onChange={(v) => update('enabled', v)} />
              </Field>
            </div>
          </Card>

          <Card title={isEmail ? '邮件供应商配置' : '短信供应商配置'} bordered>
            {isEmail ? (
              <div className="space-y-3">
                <Field label="驱动">
                  <Select value={form.driver || 'smtp'} onChange={(v) => update('driver', v)}>
                    <Option value="smtp">SMTP</Option>
                    <Option value="sendcloud">SendCloud</Option>
                  </Select>
                </Field>
                <Field label="发件人显示名">
                  <Input value={form.fromDisplayName || ''} onChange={(v) => update('fromDisplayName', v)} />
                </Field>
                {form.driver === 'smtp' ? (
                  <>
                    <Field label="SMTP Host">
                      <Input value={form.smtpHost || ''} onChange={(v) => update('smtpHost', v)} />
                    </Field>
                    <Field label="SMTP Port">
                      <InputNumber value={form.smtpPort ?? 587} onChange={(v) => update('smtpPort', Number(v ?? 587))} />
                    </Field>
                    <Field label="SMTP 用户名">
                      <Input value={form.smtpUsername || ''} onChange={(v) => update('smtpUsername', v)} />
                    </Field>
                    <Field
                      label={isEdit && emailForm?.smtpPasswordSet ? 'SMTP 密码（留空保持原值）' : 'SMTP 密码'}
                    >
                      <Input.Password
                        value={form.smtpPassword || ''}
                        onChange={(v) => update('smtpPassword', v)}
                      />
                    </Field>
                    <Field label="发件地址">
                      <Input value={form.smtpFrom || ''} onChange={(v) => update('smtpFrom', v)} />
                    </Field>
                  </>
                ) : (
                  <>
                    <Field label="API User">
                      <Input value={form.sendcloudApiUser || ''} onChange={(v) => update('sendcloudApiUser', v)} />
                    </Field>
                    <Field
                      label={isEdit && emailForm?.sendcloudApiKeySet ? 'API Key（留空保持原值）' : 'API Key'}
                    >
                      <Input.Password
                        value={form.sendcloudApiKey || ''}
                        onChange={(v) => update('sendcloudApiKey', v)}
                      />
                    </Field>
                    <Field label="发件地址">
                      <Input value={form.sendcloudFrom || ''} onChange={(v) => update('sendcloudFrom', v)} />
                    </Field>
                  </>
                )}
              </div>
            ) : (
              <div className="space-y-3">
                <Field label="供应商">
                  <Select
                    value={form.smsProvider}
                    onChange={(v) => {
                      setForm((prev) => ({ ...prev, smsProvider: v, smsConfig: {} }))
                    }}
                    showSearch
                    filterOption={(input, option) => {
                      const text = String(option?.props?.children ?? '').toLowerCase()
                      return text.includes(input.toLowerCase())
                    }}
                  >
                    {SMS_PROVIDERS.map((p) => (
                      <Option key={p.value} value={p.value}>{p.label}（{p.value}）</Option>
                    ))}
                  </Select>
                </Field>
                {currentSMSProvider?.fields.map((f) => {
                  const cfg = form.smsConfig || {}
                  const wasSet = isEdit && f.secret && smsForm?.secretKeys?.includes(f.key)
                  return (
                    <Field key={f.key} label={wasSet ? `${f.label}（留空保持原值）` : f.label}>
                      {f.secret ? (
                        <Input.Password
                          placeholder={f.placeholder}
                          value={(cfg[f.key] ?? '') as string}
                          onChange={(v) => updateSmsConfig(f.key, v)}
                        />
                      ) : (
                        <Input
                          placeholder={f.placeholder}
                          value={(cfg[f.key] ?? '') as string}
                          onChange={(v) => updateSmsConfig(f.key, v)}
                        />
                      )}
                    </Field>
                  )
                })}
              </div>
            )}
          </Card>
        </div>
      )}
    </div>
  )
}

export default NotificationChannelEditPage
