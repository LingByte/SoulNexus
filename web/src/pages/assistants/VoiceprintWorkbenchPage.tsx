import { useCallback, useEffect, useMemo, useState } from 'react'
import { Input, Modal, Popover, Select, Space, Tag, Typography, Upload } from '@arco-design/web-react'
import type { UploadItem } from '@arco-design/web-react/es/Upload'
import { Trash2, RefreshCw, Upload as UploadIcon, Mic, CheckCircle2, XCircle, UserCog } from 'lucide-react'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Button, DataList } from '@/components/ui'
import SpeakerContextEditor from '@/components/voice/SpeakerContextEditor'
import VoiceprintAudioRecorder from '@/components/voice/VoiceprintAudioRecorder'
import { useSiteConfig } from '@/contexts/siteConfig'
import {
  createVoiceprint,
  deleteVoiceprint,
  getVoiceprintConfig,
  identifyVoiceprint,
  listVoiceprints,
  runVoiceprintSelfTest,
  snowflakeStr,
  isSnowflakeSet,
  type VoiceprintConfig,
  type VoiceprintIdentifyResult,
  type VoiceprintProfile,
  type VoiceprintSelfTestReport,
} from '@/api/voiceprint'
import { showAlert } from '@/utils/notification'

function toUploadList(file: File | null): UploadItem[] {
  if (!file) return []
  return [{ uid: 'voiceprint-audio', name: file.name, originFile: file }]
}

function pickFromFileList(fileList: UploadItem[] | undefined): File | null {
  if (!fileList?.length) return null
  const item = fileList[fileList.length - 1]
  if (item?.originFile instanceof File) return item.originFile
  return null
}

type IdentifyView = {
  result: VoiceprintIdentifyResult
  profile: VoiceprintProfile | null
}

export default function VoiceprintWorkbenchPage() {
  const { config } = useSiteConfig()
  const [cfg, setCfg] = useState<VoiceprintConfig | null>(null)
  const [rows, setRows] = useState<VoiceprintProfile[]>([])
  const [loading, setLoading] = useState(true)
  const [enrollName, setEnrollName] = useState('')
  const [enrollDesc, setEnrollDesc] = useState('')
  const [enrollFile, setEnrollFile] = useState<File | null>(null)
  const [enrolling, setEnrolling] = useState(false)
  const [identifyFile, setIdentifyFile] = useState<File | null>(null)
  const [identifyScope, setIdentifyScope] = useState<string[]>([])
  const [identifying, setIdentifying] = useState(false)
  const [identifyView, setIdentifyView] = useState<IdentifyView | null>(null)
  const [selfTest, setSelfTest] = useState<VoiceprintSelfTestReport | null>(null)
  const [testing, setTesting] = useState(false)
  const [speakerEdit, setSpeakerEdit] = useState<VoiceprintProfile | null>(null)
  const [enrollRecKey, setEnrollRecKey] = useState(0)
  const [identifyRecKey, setIdentifyRecKey] = useState(0)

  const providerLabel = config.VOICEPRINT_LABEL || cfg?.label || cfg?.provider || '声纹识别'
  const identifyDisabled = cfg?.supportsIdentify === false

  const scopeOptions = useMemo(
    () =>
      rows
        .filter((r) => r.status === 'active')
        .map((r) => ({ label: r.name, value: r.featureId })),
    [rows],
  )

  const reload = useCallback(async () => {
    setLoading(true)
    try {
      const [cfgRes, listRes] = await Promise.all([getVoiceprintConfig(), listVoiceprints()])
      if (cfgRes.code === 200 && cfgRes.data) setCfg(cfgRes.data)
      if (listRes.code === 200 && listRes.data) setRows(listRes.data)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void reload()
  }, [reload])

  const handleEnroll = async () => {
    if (!enrollName.trim()) return showAlert('请填写说话人名称', 'error')
    if (!enrollFile) return showAlert('请先现场录音或上传 WAV 音频', 'error')
    setEnrolling(true)
    try {
      const res = await createVoiceprint(enrollName.trim(), enrollFile, {
        description: enrollDesc.trim(),
      })
      if (res.code !== 200) {
        showAlert(res.msg || '注册失败', 'error')
        return
      }
      showAlert('声纹注册成功', 'success')
      setEnrollName('')
      setEnrollDesc('')
      setEnrollFile(null)
      setEnrollRecKey((k) => k + 1)
      await reload()
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || '注册失败', 'error')
    } finally {
      setEnrolling(false)
    }
  }

  const handleIdentify = async () => {
    if (!identifyFile) return showAlert('请先现场录音或上传 WAV 音频', 'error')
    if (identifyDisabled) return showAlert('当前厂商不支持 1:N 识别', 'error')
    setIdentifying(true)
    setIdentifyView(null)
    try {
      const res = await identifyVoiceprint(identifyFile, {
        featureIds: identifyScope.length > 0 ? identifyScope : undefined,
      })
      if (res.code !== 200 || !res.data?.result) {
        showAlert(res.msg || '识别失败', 'error')
        return
      }
      setIdentifyView({ result: res.data.result, profile: res.data.profile ?? null })
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || '识别失败', 'error')
    } finally {
      setIdentifying(false)
    }
  }

  const handleSelfTest = async (probe: boolean) => {
    setTesting(true)
    try {
      const res = await runVoiceprintSelfTest(probe)
      if (res.code !== 200 || !res.data) {
        showAlert(res.msg || '自检失败', 'error')
        return
      }
      setSelfTest(res.data)
      showAlert(res.data.ok ? '自检通过' : '自检未通过', res.data.ok ? 'success' : 'warning')
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || '自检失败', 'error')
    } finally {
      setTesting(false)
    }
  }

  const handleDelete = (row: VoiceprintProfile) => {
    Modal.confirm({
      title: '删除声纹',
      content: `确定删除「${row.name}」？删除后无法用于 1:N 识别。`,
      onOk: async () => {
        const res = await deleteVoiceprint(row.id)
        if (res.code !== 200) {
          showAlert(res.msg || '删除失败', 'error')
          return
        }
        showAlert('已删除', 'success')
        setIdentifyScope((prev) => prev.filter((id) => id !== row.featureId))
        await reload()
      },
    })
  }

  const selfTestBadge = selfTest ? (
    <Tag color={selfTest.ok ? 'green' : 'red'}>{selfTest.ok ? 'OK' : 'FAIL'}</Tag>
  ) : null

  return (
    <BaseLayout>
      <div className="mx-auto max-w-6xl space-y-6 p-6">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0">
            <Typography.Title heading={5} style={{ margin: 0 }}>
              声纹识别
            </Typography.Title>
            <Typography.Text type="secondary" style={{ fontSize: 13 }}>
              厂商：{providerLabel}
              {cfg?.similarityThreshold != null ? ` · 默认阈值 ${cfg.similarityThreshold}` : ''}
            </Typography.Text>
            <p className="mt-1 max-w-2xl text-xs text-neutral-500">
              ① 上方注册声纹 → ② 下方列表点「配置上下文」填写 LLM 属性与工具凭证（如云阶
              token）→ ③ 会话识别命中后自动注入本通身份卡片。
            </p>
          </div>
          <Space>
            <Popover
              trigger="click"
              position="br"
              title={
                <Space>
                  <span>环境自检</span>
                  {selfTestBadge}
                </Space>
              }
              content={
                <div className="w-72 space-y-2">
                  <Space wrap>
                    <Button size="mini" loading={testing} onClick={() => void handleSelfTest(false)}>
                      校验配置
                    </Button>
                    <Button size="mini" loading={testing} type="outline" onClick={() => void handleSelfTest(true)}>
                      连通性探测
                    </Button>
                  </Space>
                  {selfTest?.checks?.length ? (
                    <div className="max-h-48 space-y-1 overflow-y-auto text-xs">
                      {selfTest.checks.map((c) => (
                        <div key={c.name} className="flex gap-2">
                          <Tag size="small" color={c.ok ? 'green' : 'red'}>
                            {c.ok ? 'OK' : 'FAIL'}
                          </Tag>
                          <span className="min-w-0 break-all">
                            {c.name}
                            {c.detail ? ` — ${c.detail}` : ''}
                          </span>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                      点击按钮运行自检
                    </Typography.Text>
                  )}
                </div>
              }
            >
              <Button type="outline" loading={testing}>
                环境自检 {selfTestBadge}
              </Button>
            </Popover>
            <Button icon={<RefreshCw size={16} />} onClick={() => void reload()}>
              刷新
            </Button>
          </Space>
        </div>

        <div className="grid gap-6 lg:grid-cols-2">
          <div className="space-y-3 rounded-xl border border-border bg-card p-5">
            <div>
              <Typography.Text bold>注册声纹</Typography.Text>
              <Typography.Paragraph type="secondary" style={{ marginBottom: 0, fontSize: 12 }}>
                现场录音或上传清晰人声 WAV（建议 ≥3 秒）。注册成功后会出现在下方列表，再点「配置上下文」。
              </Typography.Paragraph>
            </div>
            <Input placeholder="说话人名称，如：张老师" value={enrollName} onChange={setEnrollName} allowClear />
            <Input placeholder="备注（可选）" value={enrollDesc} onChange={setEnrollDesc} allowClear />
            <div className="rounded-lg border border-dashed border-border bg-neutral-50/60 p-3 space-y-2">
              <Typography.Text style={{ fontSize: 12 }} type="secondary">
                方式一：现场录音
              </Typography.Text>
              <VoiceprintAudioRecorder
                key={enrollRecKey}
                mode="record-only"
                onAudioChange={(file) => {
                  setEnrollFile(file)
                }}
              />
            </div>
            <div className="rounded-lg border border-dashed border-border bg-neutral-50/60 p-3 space-y-2">
              <Typography.Text style={{ fontSize: 12 }} type="secondary">
                方式二：上传文件
              </Typography.Text>
              <Upload
                accept="audio/*,.wav"
                autoUpload={false}
                limit={1}
                fileList={toUploadList(enrollFile)}
                onChange={(fileList) => {
                  const f = pickFromFileList(fileList)
                  setEnrollFile(f)
                  if (f) setEnrollRecKey((k) => k + 1)
                }}
                onRemove={() => {
                  setEnrollFile(null)
                  setEnrollRecKey((k) => k + 1)
                  return true
                }}
              >
                <Button icon={<UploadIcon size={16} />}>选择 WAV 音频</Button>
              </Upload>
            </div>
            {enrollFile ? (
              <Typography.Text style={{ fontSize: 12 }} type="secondary">
                已选音频：{enrollFile.name}
              </Typography.Text>
            ) : null}
            <div className="flex flex-wrap gap-2">
              <Button type="primary" loading={enrolling} onClick={() => void handleEnroll()}>
                提交注册
              </Button>
              {enrollFile ? (
                <Button
                  type="outline"
                  onClick={() => {
                    setEnrollFile(null)
                    setEnrollRecKey((k) => k + 1)
                  }}
                >
                  清除音频
                </Button>
              ) : null}
            </div>
          </div>

          <div className="space-y-3 rounded-xl border border-border bg-card p-5">
            <div>
              <Typography.Text bold>1:N 识别</Typography.Text>
              <Typography.Paragraph type="secondary" style={{ marginBottom: 0, fontSize: 12 }}>
                {identifyDisabled
                  ? '当前厂商不支持识别（请检查配置）'
                  : '现场录音或上传待识别音频；可限定比对范围（不选则比对全部已注册声纹）'}
              </Typography.Paragraph>
            </div>
            <Select
              mode="multiple"
              allowClear
              placeholder="声纹范围（可选，多选）"
              options={scopeOptions}
              value={identifyScope}
              onChange={(v) => setIdentifyScope(v as string[])}
              disabled={identifyDisabled}
            />
            <div className="rounded-lg border border-dashed border-border bg-neutral-50/60 p-3 space-y-2">
              <Typography.Text style={{ fontSize: 12 }} type="secondary">
                方式一：现场录音
              </Typography.Text>
              <VoiceprintAudioRecorder
                key={identifyRecKey}
                mode="record-only"
                disabled={identifyDisabled}
                onAudioChange={(file) => {
                  setIdentifyFile(file)
                }}
              />
            </div>
            <div className="rounded-lg border border-dashed border-border bg-neutral-50/60 p-3 space-y-2">
              <Typography.Text style={{ fontSize: 12 }} type="secondary">
                方式二：上传文件
              </Typography.Text>
              <Upload
                accept="audio/*,.wav"
                autoUpload={false}
                limit={1}
                fileList={toUploadList(identifyFile)}
                onChange={(fileList) => {
                  const f = pickFromFileList(fileList)
                  setIdentifyFile(f)
                  if (f) setIdentifyRecKey((k) => k + 1)
                }}
                onRemove={() => {
                  setIdentifyFile(null)
                  setIdentifyRecKey((k) => k + 1)
                  return true
                }}
              >
                <Button icon={<UploadIcon size={16} />} disabled={identifyDisabled}>
                  选择 WAV 音频
                </Button>
              </Upload>
            </div>
            {identifyFile ? (
              <Typography.Text style={{ fontSize: 12 }} type="secondary">
                已选音频：{identifyFile.name}
              </Typography.Text>
            ) : null}
            <div className="flex flex-wrap gap-2">
              <Button
                type="primary"
                loading={identifying}
                disabled={identifyDisabled}
                onClick={() => void handleIdentify()}
              >
                开始识别
              </Button>
              {identifyFile ? (
                <Button
                  type="outline"
                  onClick={() => {
                    setIdentifyFile(null)
                    setIdentifyView(null)
                    setIdentifyRecKey((k) => k + 1)
                  }}
                >
                  清除音频
                </Button>
              ) : null}
            </div>

            {identifyView ? (
              <div
                className={`rounded-lg border px-3 py-3 text-sm ${
                  identifyView.result.isMatch
                    ? 'border-emerald-200 bg-emerald-50/80 text-emerald-900'
                    : 'border-amber-200 bg-amber-50/80 text-amber-950'
                }`}
              >
                <div className="mb-1 flex items-center gap-2 font-medium">
                  {identifyView.result.isMatch ? (
                    <CheckCircle2 size={16} className="shrink-0 text-emerald-600" />
                  ) : (
                    <XCircle size={16} className="shrink-0 text-amber-600" />
                  )}
                  {identifyView.result.isMatch
                    ? `命中：${identifyView.profile?.name || '未知说话人'}`
                    : '未命中已注册声纹'}
                </div>
                <div className="space-y-0.5 font-mono text-xs opacity-90">
                  <div>featureId：{identifyView.result.featureId || '—'}</div>
                  <div>
                    score：{identifyView.result.score.toFixed(4)} · threshold：
                    {identifyView.result.threshold}
                    {identifyView.result.confidence
                      ? ` · confidence：${identifyView.result.confidence}`
                      : ''}
                  </div>
                </div>
              </div>
            ) : null}
          </div>
        </div>

        <div className="rounded-xl border border-border bg-card p-5">
          <div className="mb-3 flex flex-wrap items-end justify-between gap-2">
            <div>
              <div className="text-sm font-medium text-neutral-900">已注册声纹</div>
              <p className="mt-0.5 text-xs text-neutral-500">
                在每一行右侧点「配置上下文」，设置显示名、LLM 属性（如 role）和工具凭证（如
                cloudsteps token）。
              </p>
            </div>
            <span className="text-xs text-neutral-500">{rows.length} 条</span>
          </div>
          <DataList
            data={rows as unknown as (VoiceprintProfile & Record<string, unknown>)[]}
            columns={[
              {
                key: 'icon',
                width: 40,
                render: () => (
                  <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-emerald-50 text-emerald-600">
                    <Mic size={18} strokeWidth={1.75} />
                  </div>
                ),
              },
              {
                key: 'info',
                title: '名称',
                render: (_, r) => (
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-sm font-medium text-neutral-900">
                      {String(r.name || '—')}
                    </div>
                    <div className="mt-0.5 font-mono text-xs text-neutral-500">
                      {String(r.featureId || '—')}
                    </div>
                    {r.description ? (
                      <div className="mt-0.5 truncate text-xs text-neutral-400">
                        {String(r.description)}
                      </div>
                    ) : null}
                  </div>
                ),
              },
              {
                key: 'status',
                title: '状态',
                width: 100,
                render: (_, r) => (
                  <span
                    className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                      String(r.status) === 'active'
                        ? 'bg-emerald-50 text-emerald-700'
                        : 'bg-neutral-100 text-neutral-600'
                    }`}
                  >
                    {String(r.status || '—')}
                  </span>
                ),
              },
              {
                key: 'assistant',
                title: '助手',
                width: 100,
                render: (_, r) => {
                  const aid = isSnowflakeSet(r.assistantId) ? snowflakeStr(r.assistantId) : ''
                  return (
                    <span className="text-xs text-neutral-600" title={aid || undefined}>
                      {aid ? `助手 #${aid}` : '未绑定'}
                    </span>
                  )
                },
              },
              {
                key: 'provider',
                title: '厂商',
                width: 100,
                render: (_, r) => (
                  <span className="text-sm text-neutral-700">{String(r.provider || '—')}</span>
                ),
              },
              {
                key: 'actions',
                width: 200,
                align: 'right',
                render: (_, r) => {
                  const row = r as unknown as VoiceprintProfile
                  return (
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        size="mini"
                        type="primary"
                        icon={<UserCog size={12} />}
                        onClick={() => setSpeakerEdit(row)}
                      >
                        配置上下文
                      </Button>
                      <Button
                        size="mini"
                        status="danger"
                        icon={<Trash2 size={12} />}
                        onClick={() => handleDelete(row)}
                      >
                        删除
                      </Button>
                    </div>
                  )
                },
              },
            ]}
            loading={loading}
            rowKey="id"
            emptyText="暂无声纹，请先在上方「注册声纹」提交后再来配置上下文"
          />
        </div>
      </div>
      <SpeakerContextEditor
        profile={speakerEdit}
        onClose={() => setSpeakerEdit(null)}
        onSaved={() => void reload()}
      />
    </BaseLayout>
  )
}
