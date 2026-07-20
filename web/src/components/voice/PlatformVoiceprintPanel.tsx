import { useCallback, useEffect, useState } from 'react'
import { Modal, Tag } from '@arco-design/web-react'
import { UserCircle, Trash2, RefreshCw, Shield, Wifi } from 'lucide-react'
import { Button, DataList } from '@/components/ui'
import type { DataListColumn } from '@/components/ui'
import {
  deletePlatformVoiceprintProfile,
  listPlatformVoiceprintProfiles,
  runPlatformVoiceprintSelfTest,
} from '@/api/platformVoices'
import type { VoiceprintSelfTestReport } from '@/api/voiceprint'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'

const PAGE_SIZE = 20

export default function PlatformVoiceprintPanel() {
  const [rows, setRows] = useState<Record<string, unknown>[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [meta, setMeta] = useState({ enabled: false, provider: '', label: '' })
  const [selfTest, setSelfTest] = useState<VoiceprintSelfTestReport | null>(null)
  const [testing, setTesting] = useState(false)

  const reload = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listPlatformVoiceprintProfiles({ page, pageSize: PAGE_SIZE })
      if (res.code === 200 && res.data) {
        setRows((res.data.list || []) as unknown as Record<string, unknown>[])
        setTotal(res.data.total || 0)
        setMeta({
          enabled: Boolean(res.data.voiceprintEnabled),
          provider: res.data.voiceprintProvider || '',
          label: res.data.voiceprintProviderLabel || '',
        })
      }
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, '加载声纹记录失败'), 'error')
    } finally {
      setLoading(false)
    }
  }, [page])

  useEffect(() => {
    void reload()
  }, [reload])

  const handleSelfTest = async (probe: boolean) => {
    setTesting(true)
    try {
      const res = await runPlatformVoiceprintSelfTest(probe)
      if (res.code !== 200 || !res.data) {
        showAlert(res.msg || '自检失败', 'error')
        return
      }
      setSelfTest(res.data)
      showAlert(res.data.ok ? '自检通过' : '自检未通过', res.data.ok ? 'success' : 'warning')
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, '自检失败'), 'error')
    } finally {
      setTesting(false)
    }
  }

  const handleDelete = (rawId: unknown) => {
    const id = Number(rawId)
    if (!id) return
    Modal.confirm({
      title: '删除声纹',
      content: '确定删除该条声纹记录？将同步删除厂商侧特征。',
      onOk: async () => {
        const res = await deletePlatformVoiceprintProfile(id)
        if (res.code !== 200) {
          showAlert(res.msg || '删除失败', 'error')
          return
        }
        void reload()
      },
    })
  }

  const columns: DataListColumn<Record<string, unknown>>[] = [
    {
      key: 'icon',
      width: 40,
      render: () => (
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-emerald-50 text-emerald-600">
          <UserCircle size={18} strokeWidth={1.75} />
        </div>
      ),
    },
    {
      key: 'info',
      title: '名称',
      render: (_, r) => (
        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-medium text-neutral-900">{String(r.name || '—')}</div>
          <div className="mt-0.5 flex items-center gap-2 text-xs text-neutral-500">
            <span className="font-mono">{String(r.id || '').slice(0, 10)}</span>
            <span className="text-neutral-300">·</span>
            <span className="font-mono">{String(r.tenantId || '').slice(0, 10)}</span>
          </div>
        </div>
      ),
    },
    {
      key: 'meta',
      width: 220,
      render: (_, r) => (
        <div className="flex items-center gap-3 text-xs text-neutral-500">
          <span>{String(r.provider || '—')}</span>
          <span className="font-mono text-neutral-400">{String(r.featureId || '').slice(0, 14)}</span>
          <Tag color={r.status === 'active' ? 'green' : 'gray'} className="!rounded-full !text-xs">
            {String(r.status || '—')}
          </Tag>
        </div>
      ),
    },
    {
      key: 'actions',
      width: 80,
      align: 'right',
      render: (_, r) => (
        <Button
          size="mini"
          status="danger"
          icon={<Trash2 size={12} />}
          onClick={() => handleDelete(r.id)}
        >
          删除
        </Button>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        <div className="flex items-center gap-2 text-sm text-neutral-500">
          <Shield size={14} className="text-neutral-400" />
          声纹服务：
          {meta.enabled ? (
            <Tag color="green" className="!rounded-full">{meta.label || meta.provider}</Tag>
          ) : (
            <Tag color="gray" className="!rounded-full">未开启</Tag>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Button size="small" icon={<Shield size={12} />} loading={testing} onClick={() => void handleSelfTest(false)}>
            校验配置
          </Button>
          <Button size="small" icon={<Wifi size={12} />} loading={testing} type="outline" onClick={() => void handleSelfTest(true)}>
            连通性探测
          </Button>
          <Button size="small" icon={<RefreshCw size={12} />} onClick={() => void reload()}>
            刷新列表
          </Button>
        </div>
      </div>

      {selfTest ? (
        <div className="rounded-xl border border-neutral-100 bg-white p-4">
          <div className="mb-2 text-sm font-medium text-neutral-700">自检结果</div>
          <div className="space-y-1.5">
            {selfTest.checks.map((c) => (
              <div key={c.name} className="flex items-center gap-2 text-sm">
                <Tag color={c.ok ? 'green' : 'red'} className="!rounded-full !text-xs">{c.ok ? 'OK' : 'FAIL'}</Tag>
                <span className="text-neutral-700">{c.name}</span>
                {c.detail && <span className="text-neutral-400">— {c.detail}</span>}
              </div>
            ))}
          </div>
        </div>
      ) : null}

      <DataList
        data={rows}
        columns={columns}
        loading={loading}
        rowKey="id"
        emptyText="暂无声纹记录"
        pagination={total > 0 ? { current: page, pageSize: PAGE_SIZE, total, onChange: setPage } : null}
      />
    </div>
  )
}
