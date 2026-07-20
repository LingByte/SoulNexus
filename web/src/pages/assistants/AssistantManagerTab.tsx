import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Input,
  Modal,
  Space,
  Switch,
  Tag,
  Typography,
} from '@arco-design/web-react'
import { Button, Empty, Card } from '@/components/ui'
import { Loading } from '@/components/ui/loading'
import {
  IconSearch,
  IconPlus,
  IconList,
  IconApps,
} from '@arco-design/web-react/icon'
import { BookOpen, ClipboardList, Megaphone, Bot } from 'lucide-react'
import { showAlert } from '@/utils/notification'
import {
  ASSISTANT_SCENES,
  deleteAssistant,
  listAssistants,
  type AssistantRow,
} from '@/api/assistants'
import AssistantAvatar, { assistantAvatarSrc } from '@/components/assistant/AssistantAvatar'
import { cn } from '@/utils/cn'
import { t, useTranslation } from '@/i18n'

type ViewMode = 'list' | 'card'

export type AssistantManagerTabProps = {
  scopeTenantId?: string
  newPath?: string
  editPath?: (id: string) => string
}

export default function AssistantManagerTab({
  scopeTenantId,
  newPath = '/assistant-manager/new',
  editPath,
}: AssistantManagerTabProps) {
  const navigate = useNavigate()
  const { t: tt } = useTranslation()
  const [rows, setRows] = useState<AssistantRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [deleteId, setDeleteId] = useState<string | null>(null)
  const [deleteLoading, setDeleteLoading] = useState(false)
  const pageSize = 20

  // Search & filter
  const [keyword, setKeyword] = useState('')
  const [onlyMine, setOnlyMine] = useState(false)
  const [viewMode, setViewMode] = useState<ViewMode>('card')

  const resolveEditPath = (id: string) =>
    editPath ? editPath(id) : `/assistant-manager/${id}/edit`

  const tenantReady =
    scopeTenantId === undefined ||
    (scopeTenantId.trim() !== '' && scopeTenantId !== '0')

  // Debounced keyword search
  const [searchDebounce, setSearchDebounce] = useState('')
  useEffect(() => {
    const t = setTimeout(() => setSearchDebounce(keyword), 300)
    return () => clearTimeout(t)
  }, [keyword])

  const load = useCallback(async () => {
    if (!tenantReady) {
      setRows([])
      setTotal(0)
      return
    }
    setLoading(true)
    try {
      const res = await listAssistants(page, pageSize, {
        tenantId: scopeTenantId,
        name: searchDebounce || undefined,
      })
      if (res.code === 200 && res.data) {
        setRows(res.data.list || [])
        setTotal(res.data.total || 0)
      }
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || tt('assistant.loadFailed'), 'error')
    } finally {
      setLoading(false)
    }
  }, [page, scopeTenantId, tenantReady, searchDebounce])

  useEffect(() => {
    void load()
  }, [load])

  const sceneLabel = (scene?: string) =>
    ASSISTANT_SCENES.find((s) => s.value === scene)?.label || scene || '—'

  const confirmDelete = async () => {
    if (deleteId == null) return
    setDeleteLoading(true)
    try {
      const res = await deleteAssistant(deleteId)
      if (res.code !== 200) {
        showAlert(res.msg || tt('common.deleteFailed'), 'error')
        return
      }
      showAlert(tt('common.deleteSuccess'), 'success')
      setDeleteOpen(false)
      setDeleteId(null)
      await load()
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || tt('common.deleteFailed'), 'error')
    } finally {
      setDeleteLoading(false)
    }
  }

  const openNew = () => {
    if (scopeTenantId && scopeTenantId !== '0') {
      navigate(`${newPath}?tenantId=${encodeURIComponent(scopeTenantId)}`)
      return
    }
    navigate(newPath)
  }

  const renderEmpty = (
    <Empty preset="no-data" description={tt('assistant.emptyData')}>
      <Button type="primary" icon={<IconPlus />} onClick={openNew}>
        {tt('assistant.newAssistant')}
      </Button>
    </Empty>
  )

  const SceneIcon = ({ scene }: { scene?: string }) => {
    const cls = 'w-4 h-4'
    switch (scene) {
      case 'inbound_knowledge': return <BookOpen className={cls} />
      case 'outbound_collect': return <ClipboardList className={cls} />
      case 'outbound_notify': return <Megaphone className={cls} />
      default: return <Bot className={cls} />
    }
  }

  const CardItem = ({ r }: { r: AssistantRow }) => {
    const avatar = assistantAvatarSrc(r)
    return (
    <div
      className="h-full cursor-pointer"
      onClick={() => navigate(resolveEditPath(r.id))}
    >
      <Card
        variant="elevated"
        icon={
          avatar ? (
            <AssistantAvatar src={avatar} name={r.name} size="sm" rounded="lg" className="border-white/20 bg-white/10" />
          ) : (
            <SceneIcon scene={r.scene} />
          )
        }
        className="h-full"
      >
        <div className="flex w-full flex-col gap-3">
          <div className="flex flex-wrap items-center gap-2">
            <span className="inline-flex rounded-md bg-white/20 px-2 py-0.5 text-xs font-medium text-white">
              {r.enabled ? tt('common.enabled') : tt('common.disabled')}
            </span>
            <span className="inline-flex rounded-md bg-white/20 px-2 py-0.5 text-xs font-medium text-white">
              {r.publishedVersionId ? tt('common.published') : tt('common.unpublish')}
            </span>
          </div>

          <h3 className="mb-0 truncate text-base font-semibold text-white">{r.name}</h3>
          {r.description ? (
            <p className="mb-0 line-clamp-2 text-xs leading-relaxed text-white/85">{r.description}</p>
          ) : (
            <p className="mb-0 text-xs italic text-white/60">{tt('common.noDescription')}</p>
          )}

          <div className="flex flex-wrap items-center gap-2">
            <span className="inline-flex rounded-md bg-white/15 px-2 py-0.5 text-xs font-medium text-white">
              {sceneLabel(r.scene)}
            </span>
            {r.version ? (
              <span className="inline-flex rounded-md bg-white/10 px-2 py-0.5 font-mono text-xs text-white/90">
                {r.version}
              </span>
            ) : null}
          </div>

          <div
            className="mt-1 flex items-center gap-2 border-t border-white/20 pt-3"
            onClick={(e) => e.stopPropagation()}
          >
            <Button
              type="text"
              size="small"
              className="flex-1 !text-white hover:!bg-white/10"
              onClick={() => navigate(resolveEditPath(r.id))}
            >
              {tt('common.edit')}
            </Button>
            <Button
              type="text"
              status="danger"
              size="small"
              className="!text-white/90 hover:!bg-white/10"
              onClick={() => {
                setDeleteId(r.id)
                setDeleteOpen(true)
              }}
            >
              {tt('common.delete')}
            </Button>
          </div>
        </div>
      </Card>
    </div>
    )
  }

  // ---- List row ----
  const ListRow = ({ r }: { r: AssistantRow }) => (
    <tr key={r.id} className="border-t border-border transition-colors hover:bg-muted/30">
      <td className="p-3">
        <div className="flex items-center gap-3">
          <AssistantAvatar src={assistantAvatarSrc(r)} name={r.name} size="sm" rounded="full" />
          <div className="min-w-0">
            <div className="font-medium text-foreground">{r.name}</div>
            {r.description && (
              <div className="mt-0.5 truncate text-xs text-muted-foreground max-w-[240px]">
                {r.description}
              </div>
            )}
          </div>
        </div>
      </td>
      <td className="p-3"><Tag size="small">{sceneLabel(r.scene)}</Tag></td>
      <td className="p-3 font-mono text-xs">{r.version || '—'}</td>
      <td className="p-3">
        <Tag color={r.publishedVersionId ? 'arcoblue' : 'orange'} size="small">
          {r.publishedVersionId ? tt('common.published') : tt('common.unpublish')}
        </Tag>
      </td>
      <td className="p-3">
        <Tag color={r.enabled ? 'green' : 'red'} size="small">
          {r.enabled ? tt('common.enabled') : tt('common.disabled')}
        </Tag>
      </td>
      <td className="p-3 text-right">
        <Space>
          <Button type="outline" size="small" onClick={() => navigate(resolveEditPath(r.id))}>
            {tt('common.edit')}
          </Button>
          <Button
            type="outline"
            status="danger"
            size="small"
            onClick={() => {
              setDeleteId(r.id)
              setDeleteOpen(true)
            }}
          >
            {tt('common.delete')}
          </Button>
        </Space>
      </td>
    </tr>
  )

  return (
    <div className="flex min-h-full flex-col gap-4">
      {/* Header bar */}
      <div className="flex items-center justify-between gap-4">
        <h2 className="m-0 text-lg font-semibold text-foreground">{tt('assistant.breadcrumbLabel')}</h2>
        <Button type="primary" icon={<IconPlus />} onClick={openNew}>
          {tt('assistant.newAssistant')}
        </Button>
      </div>

      {/* Toolbar row */}
      <div className="flex items-center justify-between gap-3">
        {/* Right side: search + filter + view toggle */}
        <div className="flex-1" />
        <div className="flex flex-wrap items-center gap-3">
          <Input
            prefix={<IconSearch />}
            placeholder={tt('common.searchPlaceholder')}
            allowClear
            value={keyword}
            onChange={(v) => setKeyword(v)}
            style={{ width: 200 }}
          />
          <div className="flex items-center gap-1.5 text-sm">
            <Switch
              size="small"
              checked={onlyMine}
              onChange={setOnlyMine}
              disabled={!tenantReady}
            />
            <span className="text-muted-foreground whitespace-nowrap">{tt('common.onlyMine')}</span>
          </div>
          {/* View toggle icons */}
          <div className="flex items-center rounded-md border border-border bg-muted/40 p-0.5">
            <button
              type="button"
              onClick={() => setViewMode('card')}
              className={cn(
                'rounded px-2 py-1.5 transition-colors',
                viewMode === 'card'
                  ? 'bg-card shadow-sm text-foreground'
                  : 'text-muted-foreground hover:text-foreground',
              )}
              title={tt('common.cardView')}
            >
              <IconApps style={{ fontSize: 16 }} />
            </button>
            <button
              type="button"
              onClick={() => setViewMode('list')}
              className={cn(
                'rounded px-2 py-1.5 transition-colors',
                viewMode === 'list'
                  ? 'bg-card shadow-sm text-foreground'
                  : 'text-muted-foreground hover:text-foreground',
              )}
              title={tt('common.listView')}
            >
              <IconList style={{ fontSize: 16 }} />
            </button>
          </div>
        </div>
      </div>

      {/* Content area */}
      {!tenantReady ? (
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          {tt('common.selectFirstTenant')}
        </Typography.Text>
      ) : loading ? (
        <Loading block tip={tt('common.loading')} />
      ) : rows.length === 0 ? (
        renderEmpty
      ) : viewMode === 'card' ? (
        <>
          <div className="grid grid-cols-1 gap-4 overflow-visible sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {rows.map((r) => (
              <CardItem key={r.id} r={r} />
            ))}
          </div>
          {/* Pagination right-aligned at bottom-right */}
          <div className="flex items-center justify-end">
            <PaginationButtons page={page} pageSize={pageSize} total={total} onPageChange={setPage} />
          </div>
        </>
      ) : (
        <>
          <div className="overflow-x-auto rounded-xl border border-border bg-card">
            <table className="min-w-[800px] w-full text-sm">
              <thead className="bg-muted/40">
                <tr>
                  <th className="text-left p-3 font-medium text-muted-foreground">{tt('common.name')}</th>
                  <th className="text-left p-3 font-medium text-muted-foreground">{tt('common.scene')}</th>
                  <th className="text-left p-3 font-medium text-muted-foreground">{tt('common.version')}</th>
                  <th className="text-left p-3 font-medium text-muted-foreground">{tt('common.publish')}</th>
                  <th className="text-left p-3 font-medium text-muted-foreground">{tt('common.status')}</th>
                  <th className="text-right p-3 font-medium text-muted-foreground">{tt('common.actions')}</th>
                </tr>
              </thead>
              <tbody>{rows.map((r) => (
                <ListRow key={r.id} r={r} />
              ))}</tbody>
            </table>
            <div className="flex items-center justify-between border-t border-border px-4 py-3 text-sm">
              <span className="text-muted-foreground">{tt('common.pageRecord', { count: total })}</span>
              <PaginationButtons page={page} pageSize={pageSize} total={total} onPageChange={setPage} />
            </div>
          </div>
        </>
      )}

      <Modal
        title={tt('assistant.confirmDelete')}
        visible={deleteOpen}
        onOk={() => void confirmDelete()}
        onCancel={() => {
          setDeleteOpen(false)
          setDeleteId(null)
        }}
        okText={tt('common.confirmDelete')}
        okButtonProps={{ status: 'danger', loading: deleteLoading }}
      >
        <Typography.Text>{tt('assistant.deleteHint')}</Typography.Text>
      </Modal>
    </div>
  )
}

// ─── Pagination buttons ──────────────────────────────────────────────

function PaginationButtons({
  page,
  pageSize,
  total,
  onPageChange,
}: {
  page: number
  pageSize: number
  total: number
  onPageChange: (p: number) => void
}) {
  return (
    <Space>
      <Button
        size="small"
        disabled={page <= 1}
        onClick={() => onPageChange(Math.max(1, page - 1))}
      >
        {t('common.previous')}
      </Button>
      <Button
        size="small"
        disabled={page * pageSize >= total}
        onClick={() => onPageChange(page + 1)}
      >
        {t('common.next')}
      </Button>
    </Space>
  )
}
