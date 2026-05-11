// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 账号锁定管理：Arco Table，支持邮箱过滤、是否激活筛选与解锁。
import { useEffect, useState } from 'react'
import {
  Table,
  Input,
  Select,
  Button,
  Tag,
  Popconfirm,
  Message,
  Space,
} from '@arco-design/web-react'
import { Search, RefreshCw, Unlock } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import { getAccountLocks, unlockAccount, type AccountLock } from '@/services/adminApi'

const Option = Select.Option

const ACTIVE_OPTIONS = [
  { label: '全部', value: '' },
  { label: '锁定中', value: 'true' },
  { label: '已解锁', value: 'false' },
]

const AccountLocks = () => {
  const [locks, setLocks] = useState<AccountLock[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [email, setEmail] = useState('')
  const [active, setActive] = useState('')

  const fetchLocks = async () => {
    setLoading(true)
    try {
      const res = await getAccountLocks({
        page,
        page_size: pageSize,
        email: email || undefined,
        is_active: active === '' ? undefined : active === 'true',
      })
      setLocks(res.locks || [])
      setTotal(res.total || 0)
    } catch (e: any) {
      Message.error(`获取账号锁定记录失败：${e?.msg || e?.message || e}`)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchLocks()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, active])

  const handleSearch = () => {
    setPage(1)
    fetchLocks()
  }

  const onUnlock = async (id: number) => {
    try {
      await unlockAccount(id)
      Message.success('账号解锁成功')
      fetchLocks()
    } catch (e: any) {
      Message.error(`账号解锁失败：${e?.msg || e?.message || e}`)
    }
  }

  const columns = [
    { title: '邮箱', dataIndex: 'email', ellipsis: true, width: 240 },
    { title: '失败次数', dataIndex: 'failedAttempts', width: 100 },
    { title: 'IP', dataIndex: 'ipAddress', width: 140 },
    {
      title: '解锁时间',
      dataIndex: 'unlockAt',
      width: 180,
      render: (v: string) => (v ? new Date(v).toLocaleString('zh-CN') : '-'),
    },
    {
      title: '状态',
      dataIndex: 'isActive',
      width: 110,
      render: (v: boolean) =>
        v ? <Tag color="red">锁定中</Tag> : <Tag color="green">已解锁</Tag>,
    },
    {
      title: '操作',
      key: '__actions__',
      width: 110,
      fixed: 'right' as const,
      render: (_: any, r: AccountLock) => (
        <Popconfirm
          title={`确认解锁 ${r.email}？`}
          okText="解锁"
          cancelText="取消"
          disabled={!r.isActive}
          onOk={() => onUnlock(r.id)}
        >
          <Button size="mini" type="text" disabled={!r.isActive}>
            <span className="inline-flex items-center gap-1"><Unlock size={14} /> 解锁</span>
          </Button>
        </Popconfirm>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <PageHeader
        title="账号锁定管理"
        description="查看并解锁被风控规则锁定的账号；解锁后该账号可立即重新登录。"
        actions={
          <Space>
            <Input
              prefix={<Search size={14} />}
              placeholder="邮箱"
              value={email}
              onChange={(v) => setEmail(v)}
              onPressEnter={handleSearch}
              allowClear
              style={{ width: 220 }}
            />
            <Select
              value={active}
              onChange={(v) => { setPage(1); setActive(v) }}
              style={{ width: 140 }}
            >
              {ACTIVE_OPTIONS.map((o) => (
                <Option key={o.value} value={o.value}>{o.label}</Option>
              ))}
            </Select>
            <Button type="primary" onClick={handleSearch}>搜索</Button>
            <Button onClick={fetchLocks}>
              <span className="inline-flex items-center gap-1">
                <RefreshCw size={14} className={loading ? 'animate-spin' : ''} /> 刷新
              </span>
            </Button>
          </Space>
        }
      />

      <Table
        rowKey={(r: AccountLock) => String(r.id)}
        loading={loading}
        columns={columns}
        data={locks}
        scroll={{ x: 1000 }}
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
    </div>
  )
}

export default AccountLocks
