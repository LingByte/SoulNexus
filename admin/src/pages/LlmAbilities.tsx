// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// LLM 模型路由（abilities）：(group, model, channel_id) 复合主键，由渠道侧 sync 自动产生，
// 这里只做查看 + 跳转回渠道。
import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Table, Input, Select, Button, Tag, Message, Space } from '@arco-design/web-react'
import { Search, RefreshCw, Edit } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import {
  listLLMAbilities,
  listLLMChannels,
  type LLMAbility,
  type LLMChannel,
} from '@/services/adminApi'

const Option = Select.Option

const LlmAbilities = () => {
  const nav = useNavigate()
  const [rows, setRows] = useState<LLMAbility[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [channels, setChannels] = useState<LLMChannel[]>([])

  const [groupFilter, setGroupFilter] = useState('')
  const [modelInput, setModelInput] = useState('')
  const [model, setModel] = useState('')
  const [channelFilter, setChannelFilter] = useState<number | ''>('')
  const [enabledFilter, setEnabledFilter] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)

  const channelMap = useMemo(() => {
    const m = new Map<number, LLMChannel>()
    channels.forEach((c) => m.set(c.id, c))
    return m
  }, [channels])

  const fetchChannels = async () => {
    try {
      const r = await listLLMChannels({ page: 1, pageSize: 200, mask_key: 'true' })
      setChannels(r.channels || [])
    } catch (e: any) {
      Message.error(`加载渠道失败：${e?.msg || e?.message || e}`)
    }
  }

  const fetchRows = async () => {
    setLoading(true)
    try {
      const r = await listLLMAbilities({
        page,
        pageSize,
        group: groupFilter || undefined,
        model: model || undefined,
        channel_id: channelFilter === '' ? undefined : channelFilter,
        enabled: enabledFilter || undefined,
      })
      setRows(r.abilities || [])
      setTotal(r.total || 0)
    } catch (e: any) {
      Message.error(`加载失败：${e?.msg || e?.message || e}`)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchChannels()
  }, [])

  useEffect(() => {
    fetchRows()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, groupFilter, model, channelFilter, enabledFilter])

  const onSearch = () => { setPage(1); setModel(modelInput.trim()) }

  const groupOptions = useMemo(() => {
    const set = new Set<string>()
    channels.forEach((c) => String(c.group || '').split(',').map((s) => s.trim()).filter(Boolean).forEach((g) => set.add(g)))
    return Array.from(set).sort()
  }, [channels])

  const columns = [
    {
      title: '分组',
      dataIndex: 'group',
      width: 140,
      render: (v: string) => <Tag>{v || 'default'}</Tag>,
    },
    { title: '模型', dataIndex: 'model', ellipsis: true },
    {
      title: '渠道',
      dataIndex: 'channel_id',
      width: 240,
      render: (v: number) => {
        const ch = channelMap.get(v)
        if (!ch) return <span className="text-[var(--color-text-3)]">#{v}（未找到）</span>
        return (
          <Space size="mini" wrap>
            <span className="font-medium">#{ch.id}</span>
            <span>{ch.name || '-'}</span>
            <Tag color={ch.protocol === 'openai' ? 'arcoblue' : ch.protocol === 'anthropic' ? 'orangered' : 'gray'} size="small">
              {ch.protocol}
            </Tag>
          </Space>
        )
      },
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 90,
      render: (v: boolean) => <Tag color={v ? 'green' : 'gray'}>{v ? '启用' : '停用'}</Tag>,
    },
    { title: '优先级', dataIndex: 'priority', width: 90 },
    { title: '权重', dataIndex: 'weight', width: 80 },
    { title: '标签', dataIndex: 'tag', width: 110, render: (v: string) => v || '-' },
    {
      title: '操作',
      key: '__actions__',
      width: 120,
      render: (_: any, row: LLMAbility) => (
        <Button size="mini" type="text" onClick={() => nav('/llm-channels')}>
          <span className="inline-flex items-center gap-1"><Edit size={14} /> 编辑渠道</span>
        </Button>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <PageHeader
        title="LLM 模型路由"
        description="(group, model, channel_id) 复合主键，由渠道侧 models/group 字段在保存时自动同步。要修改请回到渠道编辑页。"
        actions={
          <Space wrap>
            <Input
              prefix={<Search size={14} />}
              placeholder="按模型名搜索"
              value={modelInput}
              onChange={(v) => setModelInput(v)}
              onPressEnter={onSearch}
              allowClear
              onClear={() => { setPage(1); setModel('') }}
              style={{ width: 220 }}
            />
            <Select value={groupFilter} onChange={(v) => { setPage(1); setGroupFilter(v) }} placeholder="分组" style={{ width: 140 }}>
              <Option value="">全部分组</Option>
              {groupOptions.map((g) => <Option key={g} value={g}>{g}</Option>)}
            </Select>
            <Select
              value={channelFilter as any}
              onChange={(v) => { setPage(1); setChannelFilter(v === '' ? '' : Number(v)) }}
              placeholder="渠道"
              style={{ width: 220 }}
              allowClear
              onClear={() => setChannelFilter('')}
            >
              <Option value="">全部渠道</Option>
              {channels.map((c) => (
                <Option key={c.id} value={c.id}>#{c.id} {c.name}</Option>
              ))}
            </Select>
            <Select value={enabledFilter} onChange={(v) => { setPage(1); setEnabledFilter(v) }} placeholder="状态" style={{ width: 110 }}>
              <Option value="">全部</Option>
              <Option value="true">启用</Option>
              <Option value="false">停用</Option>
            </Select>
            <Button type="primary" onClick={onSearch}>搜索</Button>
            <Button onClick={fetchRows}>
              <span className="inline-flex items-center gap-1"><RefreshCw size={14} className={loading ? 'animate-spin' : ''} /> 刷新</span>
            </Button>
          </Space>
        }
      />

      <Table
        rowKey={(r: LLMAbility) => `${r.group}|${r.model}|${r.channel_id}`}
        loading={loading}
        columns={columns}
        data={rows}
        pagination={{
          current: page,
          pageSize,
          total,
          showTotal: (t: number) => `共 ${t} 条`,
          sizeCanChange: true,
          sizeOptions: [20, 50, 100, 200],
          onChange: (p: number, ps: number) => { setPage(p); setPageSize(ps) },
        }}
      />
    </div>
  )
}

export default LlmAbilities
