// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 组织管理（组织列表）：Arco Table + 归档切换 + 删除。
import { useEffect, useState } from 'react'
import { Table, Input, Button, Tag, Popconfirm, Message, Space } from '@arco-design/web-react'
import { Search, RefreshCw, Archive, ArchiveRestore, Trash2 } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import {
  listAdminGroups,
  updateAdminGroup,
  deleteAdminGroup,
  type AdminGroup,
} from '@/services/adminApi'

const Groups = () => {
  const [groups, setGroups] = useState<AdminGroup[]>([])
  const [loading, setLoading] = useState(false)
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)

  const fetchData = async () => {
    setLoading(true)
    try {
      const res = await listAdminGroups({ page, pageSize, search })
      setGroups(res.groups || [])
      setTotal(res.total || 0)
    } catch (e: any) {
      Message.error(`加载组织列表失败：${e?.msg || e?.message || e}`)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, search])

  const onArchiveToggle = async (g: AdminGroup) => {
    try {
      await updateAdminGroup(g.id, { isArchived: !g.isArchived })
      Message.success(g.isArchived ? '已取消归档' : '已归档')
      fetchData()
    } catch (e: any) {
      Message.error(`操作失败：${e?.msg || e?.message || e}`)
    }
  }

  const onDelete = async (g: AdminGroup) => {
    try {
      await deleteAdminGroup(g.id)
      Message.success('已删除')
      fetchData()
    } catch (e: any) {
      Message.error(`删除失败：${e?.msg || e?.message || e}`)
    }
  }

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 100 },
    { title: '名称', dataIndex: 'name', ellipsis: true },
    { title: '类型', dataIndex: 'type', width: 130, render: (v: string) => v || '-' },
    { title: '成员数', dataIndex: 'memberCount', width: 100, render: (v: number) => v || 0 },
    {
      title: '状态',
      dataIndex: 'isArchived',
      width: 110,
      render: (v: boolean) => (v ? <Tag color="orange">已归档</Tag> : <Tag color="green">正常</Tag>),
    },
    {
      title: '操作',
      key: '__actions__',
      width: 200,
      fixed: 'right' as const,
      render: (_: any, g: AdminGroup) => (
        <Space size="mini">
          <Button size="mini" type="text" onClick={() => onArchiveToggle(g)}>
            <span className="inline-flex items-center gap-1">
              {g.isArchived ? <ArchiveRestore size={14} /> : <Archive size={14} />}
              {g.isArchived ? '取消归档' : '归档'}
            </span>
          </Button>
          <Popconfirm title={`确认删除组织 ${g.name}？`} okText="删除" cancelText="取消" onOk={() => onDelete(g)}>
            <Button size="mini" type="text" status="danger">
              <span className="inline-flex items-center gap-1"><Trash2 size={14} /> 删除</span>
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <PageHeader
        title="组织管理"
        description="管理组织信息；归档后该组织成员仍可访问只读资源，但不能再创建新内容。"
        actions={
          <Space>
            <Input
              prefix={<Search size={14} />}
              placeholder="搜索名称 / 类型"
              value={searchInput}
              onChange={(v) => setSearchInput(v)}
              onPressEnter={() => { setPage(1); setSearch(searchInput) }}
              allowClear
              onClear={() => { setPage(1); setSearch('') }}
              style={{ width: 240 }}
            />
            <Button type="primary" onClick={() => { setPage(1); setSearch(searchInput) }}>搜索</Button>
            <Button onClick={fetchData}>
              <span className="inline-flex items-center gap-1">
                <RefreshCw size={14} className={loading ? 'animate-spin' : ''} /> 刷新
              </span>
            </Button>
          </Space>
        }
      />

      <Table
        rowKey={(g: AdminGroup) => String(g.id)}
        loading={loading}
        columns={columns}
        data={groups}
        scroll={{ x: 900 }}
        pagination={{
          current: page,
          pageSize,
          total,
          showTotal: (t: number) => `共 ${t} 条`,
          sizeCanChange: true,
          sizeOptions: [10, 20, 50, 100],
          onChange: (p: number, ps: number) => { setPage(p); setPageSize(ps) },
        }}
      />
    </div>
  )
}

export default Groups
