// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 通用实体管理页（搜索 + 分页表格 + 详情 + 删除）。
// 外部接口保持向后兼容：props 与旧版完全一致。内部改用 Arco Design 的
// Table / Modal / Input / Pagination / Button，统一管理端视觉。
import { useEffect, useMemo, useState } from 'react'
import {
  Table,
  Modal,
  Input,
  Button,
  Popconfirm,
  Message,
  Space,
} from '@arco-design/web-react'
import { Eye, RefreshCw, Trash2, Download, Search } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'

type AnyRecord = Record<string, any>
type ListResult = { items: AnyRecord[]; total: number; page: number; pageSize: number }

interface GenericEntityPageProps {
  title: string
  description: string
  searchPlaceholder: string
  exportName: string
  fetchList: (params: { page: number; pageSize: number; search?: string }) => Promise<ListResult>
  deleteItem: (id: string | number) => Promise<any>
  getId: (item: AnyRecord) => string | number
}

const prettyKey = (k: string) =>
  k
    .replaceAll('_', ' ')
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .replace(/\s+/g, ' ')
    .trim()

const renderValue = (value: any): string => {
  if (value === null || value === undefined || value === '') return '-'
  if (typeof value === 'boolean') return value ? '是' : '否'
  if (typeof value === 'string' && /(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})/.test(value)) {
    const d = new Date(value)
    if (!Number.isNaN(d.getTime())) return d.toLocaleString('zh-CN')
  }
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

const GenericEntityPage = ({
  title,
  description,
  searchPlaceholder,
  exportName,
  fetchList,
  deleteItem,
  getId,
}: GenericEntityPageProps) => {
  const [list, setList] = useState<AnyRecord[]>([])
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [detail, setDetail] = useState<AnyRecord | null>(null)

  const columnKeys = useMemo(() => {
    const keys = new Set<string>()
    list.forEach((item) => Object.keys(item).slice(0, 8).forEach((k) => keys.add(k)))
    const base = Array.from(keys)
    if (!base.includes('id')) base.unshift('id')
    return base.slice(0, 8)
  }, [list])

  const columns = useMemo(() => {
    const cols: any[] = columnKeys.map((key) => ({
      title: prettyKey(key),
      dataIndex: key,
      ellipsis: true,
      render: (val: any) => renderValue(val),
    }))
    cols.push({
      title: '操作',
      key: '__actions__',
      fixed: 'right',
      width: 160,
      render: (_: any, item: AnyRecord) => (
        <Space size="mini">
          <Button
            size="mini"
            type="text"
            onClick={() => setDetail(item)}
          >
            <span className="inline-flex items-center gap-1"><Eye size={14} /> 详情</span>
          </Button>
          <Popconfirm
            title="确认删除该条记录？"
            okText="删除"
            cancelText="取消"
            onOk={async () => {
              try {
                await deleteItem(getId(item))
                Message.success('删除成功')
                fetchData()
              } catch (e: any) {
                Message.error(`删除失败：${e?.msg || e?.message || e}`)
              }
            }}
          >
            <Button size="mini" type="text" status="danger">
              <span className="inline-flex items-center gap-1"><Trash2 size={14} /> 删除</span>
            </Button>
          </Popconfirm>
        </Space>
      ),
    })
    return cols
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [columnKeys])

  const fetchData = async () => {
    try {
      setLoading(true)
      const res = await fetchList({ page, pageSize, search: search || undefined })
      setList(res.items || [])
      setTotal(res.total || 0)
    } catch (e: any) {
      Message.error(`加载${title}失败：${e?.msg || e?.message || e}`)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, search])

  const exportCsv = () => {
    if (list.length === 0) {
      Message.info('当前页无数据可导出')
      return
    }
    const headers = columnKeys
    const rows = list.map((item) => headers.map((h) => item[h] ?? ''))
    const csv = [headers, ...rows]
      .map((row) => row.map((v) => `"${String(v).replaceAll('"', '""')}"`).join(','))
      .join('\n')
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${exportName}_${Date.now()}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  const detailEntries = useMemo(() => {
    if (!detail) return []
    return Object.entries(detail).sort(([a], [b]) => {
      if (a === 'id') return -1
      if (b === 'id') return 1
      if (a.endsWith('_at') && !b.endsWith('_at')) return 1
      if (!a.endsWith('_at') && b.endsWith('_at')) return -1
      return a.localeCompare(b)
    })
  }, [detail])

  const handleSearch = () => {
    setPage(1)
    setSearch(searchInput)
  }

  return (
    <div className="space-y-4">
      <PageHeader
        title={title}
        description={description}
        actions={
          <Space>
            <Input
              value={searchInput}
              onChange={(v) => setSearchInput(v)}
              onPressEnter={handleSearch}
              placeholder={searchPlaceholder}
              prefix={<Search size={14} />}
              allowClear
              style={{ width: 240 }}
            />
            <Button onClick={handleSearch}>搜索</Button>
            <Button onClick={fetchData}>
              <span className="inline-flex items-center gap-1"><RefreshCw size={14} /> 刷新</span>
            </Button>
            <Button onClick={exportCsv}>
              <span className="inline-flex items-center gap-1"><Download size={14} /> 导出 CSV</span>
            </Button>
          </Space>
        }
      />
      <Table
        rowKey={(record: AnyRecord) => String(getId(record))}
        loading={loading}
        columns={columns}
        data={list}
        scroll={{ x: true }}
        pagination={{
          current: page,
          pageSize,
          total,
          showTotal: (t: number) => `共 ${t} 条`,
          sizeCanChange: true,
          sizeOptions: [10, 20, 50, 100],
          onChange: (p: number, ps: number) => {
            setPage(p)
            setPageSize(ps)
          },
        }}
      />

      <Modal
        title={`${title}详情`}
        visible={!!detail}
        onCancel={() => setDetail(null)}
        footer={null}
        autoFocus={false}
        style={{ width: 720 }}
      >
        {detail && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-sm">
            {detailEntries.map(([k, v]) => (
              <div
                key={k}
                className="rounded-md border border-[var(--color-border-2)] p-3"
              >
                <div className="text-xs uppercase tracking-wide text-[var(--color-text-3)] mb-1">
                  {prettyKey(k)}
                </div>
                <span className="break-all text-[var(--color-text-1)]">{renderValue(v)}</span>
              </div>
            ))}
          </div>
        )}
      </Modal>
    </div>
  )
}

export default GenericEntityPage
