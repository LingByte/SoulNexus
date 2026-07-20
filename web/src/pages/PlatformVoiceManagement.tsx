import { useCallback, useEffect, useState } from 'react'
import { Modal, Space, Tabs, Tag } from '@arco-design/web-react'
import { Mic, Volume2, UserCircle, RefreshCw, Play, Trash2, History, Loader2 } from 'lucide-react'
import { Button, DataList } from '@/components/ui'
import type { DataListColumn } from '@/components/ui'
import BaseLayout from '@/components/Layout/BaseLayout'
import { cn } from '@/utils/cn'
import {
  deletePlatformVoiceSynthesisHistory,
  listPlatformVoiceCloneProfiles,
  listPlatformVoiceSynthesisHistory,
  listPlatformVoiceCatalog,
} from '@/api/platformVoices'
import type { VoiceCatalogResponse } from '@/api/voices'
import PlatformVoiceprintPanel from '@/components/voice/PlatformVoiceprintPanel'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'

const CATALOG_PROVIDERS = [
  'aliyun',
  'volcengine',
  'xunfei',
  'qcloud',
  'baidu',
  'google',
  'azure',
  'aws',
  'openai',
  'elevenlabs',
  'qiniu',
  'minimax',
  'fishspeech',
  'fishaudio',
  'coqui',
  'local',
  'local_gospeech',
  'aliyun_omni',
  'volcengine_clone',
  'xunfei_clone',
  'volcengine_dialogue',
]

export default function PlatformVoiceManagement() {
  const [tab, setTab] = useState('catalog')
  const [catalogProvider, setCatalogProvider] = useState('aliyun')
  const [catalog, setCatalog] = useState<VoiceCatalogResponse | null>(null)
  const [catalogLoading, setCatalogLoading] = useState(false)

  const [cloneRows, setCloneRows] = useState<Record<string, unknown>[]>([])
  const [cloneTotal, setCloneTotal] = useState(0)
  const [clonePage, setClonePage] = useState(1)
  const [cloneLoading, setCloneLoading] = useState(false)
  const [cloneMeta, setCloneMeta] = useState({ enabled: false, provider: '', label: '' })

  const [histRows, setHistRows] = useState<Record<string, unknown>[]>([])
  const [histTotal, setHistTotal] = useState(0)
  const [histPage, setHistPage] = useState(1)
  const [histLoading, setHistLoading] = useState(false)

  const pageSize = 20

  const loadCatalog = useCallback(async () => {
    setCatalogLoading(true)
    try {
      const res = await listPlatformVoiceCatalog(catalogProvider, 'tts')
      if (res.code === 200 && res.data) setCatalog(res.data)
      else setCatalog(null)
    } catch {
      setCatalog(null)
    } finally {
      setCatalogLoading(false)
    }
  }, [catalogProvider])

  const loadClones = useCallback(async () => {
    setCloneLoading(true)
    try {
      const res = await listPlatformVoiceCloneProfiles({ page: clonePage, pageSize })
      if (res.code === 200 && res.data) {
        setCloneRows((res.data.list || []) as unknown as Record<string, unknown>[])
        setCloneTotal(res.data.total || 0)
        setCloneMeta({
          enabled: Boolean(res.data.cloneEnabled),
          provider: res.data.cloneProvider || '',
          label: res.data.cloneProviderLabel || '',
        })
      }
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, '加载克隆音色失败'), 'error')
    } finally {
      setCloneLoading(false)
    }
  }, [clonePage])

  const loadHistory = useCallback(async () => {
    setHistLoading(true)
    try {
      const res = await listPlatformVoiceSynthesisHistory({ page: histPage, pageSize })
      if (res.code === 200 && res.data) {
        setHistRows((res.data.list || []) as unknown as Record<string, unknown>[])
        setHistTotal(res.data.total || 0)
      }
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, '加载合成记录失败'), 'error')
    } finally {
      setHistLoading(false)
    }
  }, [histPage])

  useEffect(() => {
    if (tab === 'catalog') void loadCatalog()
  }, [tab, loadCatalog])

  useEffect(() => {
    if (tab === 'clones') void loadClones()
  }, [tab, loadClones])

  useEffect(() => {
    if (tab === 'history') void loadHistory()
  }, [tab, loadHistory])

  const handleDeleteHistory = (rawId: unknown) => {
    const id = String(rawId ?? '').trim()
    if (!id) return
    Modal.confirm({
      title: '删除合成记录',
      content: '确定删除该条合成记录？',
      onOk: async () => {
        const res = await deletePlatformVoiceSynthesisHistory(id)
        if (res.code !== 200) {
          showAlert(res.msg || '删除失败', 'error')
          return
        }
        void loadHistory()
      },
    })
  }

  const cloneColumns: DataListColumn<Record<string, unknown>>[] = [
    {
      key: 'icon',
      width: 40,
      render: () => (
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-violet-50 text-violet-600">
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
      width: 180,
      render: (_, r) => (
        <div className="flex items-center gap-3 text-xs text-neutral-500">
          <span>{String(r.provider || '—')}</span>
          <Tag color={r.status === 'active' ? 'green' : 'gray'} className="!rounded-full !text-xs">
            {String(r.status || '—')}
          </Tag>
          <span className="font-mono text-neutral-400">{String(r.assetId || r.speakerId || '').slice(0, 14)}</span>
        </div>
      ),
    },
  ]

  const historyColumns: DataListColumn<Record<string, unknown>>[] = [
    {
      key: 'icon',
      width: 40,
      render: () => (
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-blue-50 text-blue-600">
          <Volume2 size={18} strokeWidth={1.75} />
        </div>
      ),
    },
    {
      key: 'info',
      title: '音色 / 文本',
      render: (_, r) => (
        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-medium text-neutral-900">{String(r.voiceName || '—')}</div>
          <div className="mt-0.5 flex items-center gap-2 text-xs text-neutral-500">
            <span className="truncate">{String(r.text || '—')}</span>
          </div>
        </div>
      ),
    },
    {
      key: 'meta',
      width: 200,
      render: (_, r) => (
        <div className="flex items-center gap-3 text-xs text-neutral-500">
          <span className="font-mono">{String(r.tenantId || '').slice(0, 10)}</span>
          <Tag color={r.status === 'success' ? 'green' : 'red'} className="!rounded-full !text-xs">
            {String(r.status || '—')}
          </Tag>
        </div>
      ),
    },
    {
      key: 'actions',
      width: 100,
      align: 'right',
      render: (_, r) => (
        <Space size="mini">
          {r.audioUrl ? (
            <Button
              size="mini"
              icon={<Play size={12} />}
              onClick={() => {
                const audio = new Audio(String(r.audioUrl))
                void audio.play()
              }}
            >
              播放
            </Button>
          ) : null}
          <Button
            size="mini"
            status="danger"
            icon={<Trash2 size={12} />}
            onClick={() => handleDeleteHistory(r.id)}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ]

  return (
    <BaseLayout title="音色管理" description="平台全局音色目录、租户克隆音色、声纹识别与合成记录管理">
      <div className="mx-auto max-w-6xl space-y-4">
        <Tabs activeTab={tab} onChange={setTab} type="rounded">
          <Tabs.TabPane
            key="catalog"
            title={
              <span className="flex items-center gap-1.5">
                <Volume2 size={14} className="text-blue-500" />
                音色目录
              </span>
            }
          >
            <div className="mt-4 space-y-4">
              <div className="flex flex-wrap gap-2">
                {CATALOG_PROVIDERS.map((p) => (
                  <Button
                    key={p}
                    type={catalogProvider === p ? 'primary' : 'secondary'}
                    size="small"
                    onClick={() => setCatalogProvider(p)}
                  >
                    {p}
                  </Button>
                ))}
                <Button size="small" icon={<RefreshCw size={12} />} onClick={() => void loadCatalog()}>
                  刷新
                </Button>
              </div>
              {catalogLoading ? (
                <div className="flex items-center justify-center py-12 text-neutral-400">
                  <Loader2 size={20} className="mr-2 animate-spin" /> 加载中...
                </div>
              ) : (catalog?.voices || []).length === 0 ? (
                <div className="py-12 text-center text-sm text-neutral-400">暂无音色</div>
              ) : (
                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
                  {(catalog?.voices || []).map((v: any) => (
                    <div key={String(v.id)} className="flex items-center gap-3 rounded-xl border border-neutral-100 bg-white px-4 py-3 transition hover:border-neutral-200 hover:shadow-sm">
                      <div className={cn(
                        'flex h-10 w-10 shrink-0 items-center justify-center rounded-lg',
                        String(v.gender || '').toLowerCase() === 'female' ? 'bg-emerald-50 text-emerald-600' :
                        String(v.gender || '').toLowerCase() === 'male' ? 'bg-sky-50 text-sky-600' :
                        'bg-violet-50 text-violet-600',
                      )}>
                        <Volume2 size={18} />
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="truncate text-sm font-medium text-neutral-900">{String(v.label || '—')}</div>
                        <div className="mt-0.5 flex items-center gap-2 text-xs text-neutral-500">
                          {v.gender && <span className="rounded-full bg-neutral-100 px-2 py-0.5">{String(v.gender)}</span>}
                          {v.locale && <span className="rounded-full bg-neutral-100 px-2 py-0.5">{String(v.locale)}</span>}
                        </div>
                      </div>
                      <span className="shrink-0 font-mono text-xs text-neutral-400">{String(v.id || '').slice(0, 12)}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </Tabs.TabPane>

          <Tabs.TabPane
            key="clones"
            title={
              <span className="flex items-center gap-1.5">
                <UserCircle size={14} className="text-violet-500" />
                克隆音色
              </span>
            }
          >
            <div className="mt-4">
              <div className="mb-4 flex items-center gap-2 text-sm text-neutral-500">
                克隆服务：
                {cloneMeta.enabled ? (
                  <Tag color="green" className="!rounded-full">{cloneMeta.label || cloneMeta.provider}</Tag>
                ) : (
                  <Tag color="gray" className="!rounded-full">未开启</Tag>
                )}
              </div>
              <DataList
                data={cloneRows}
                columns={cloneColumns}
                loading={cloneLoading}
                rowKey="id"
                emptyText="暂无克隆音色"
                pagination={cloneTotal > 0 ? { current: clonePage, pageSize, total: cloneTotal, onChange: setClonePage } : null}
              />
            </div>
          </Tabs.TabPane>

          <Tabs.TabPane
            key="voiceprints"
            title={
              <span className="flex items-center gap-1.5">
                <Mic size={14} className="text-emerald-500" />
                声纹识别
              </span>
            }
          >
            <div className="mt-4">
              <PlatformVoiceprintPanel />
            </div>
          </Tabs.TabPane>

          <Tabs.TabPane
            key="history"
            title={
              <span className="flex items-center gap-1.5">
                <History size={14} className="text-amber-500" />
                合成记录
              </span>
            }
          >
            <div className="mt-4">
              <DataList
                data={histRows}
                columns={historyColumns}
                loading={histLoading}
                rowKey="id"
                emptyText="暂无合成记录"
                pagination={histTotal > 0 ? { current: histPage, pageSize, total: histTotal, onChange: setHistPage } : null}
              />
            </div>
          </Tabs.TabPane>
        </Tabs>
      </div>
    </BaseLayout>
  )
}
