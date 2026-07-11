import { useEffect, useState, useCallback } from 'react'
import {
  Table,
  Card,
  Input,
  Button,
  Tag,
  Space,
  Modal,
  Rate,
  Select,
  Popconfirm,
  Descriptions,
  Typography,
  Tooltip,
  Image,
  Message,
} from '@arco-design/web-react'
import {
  Search,
  RefreshCw,
  Download,
  Eye,
  GitFork,
  Star,
  User,
} from 'lucide-react'
import {
  listMarketAgents,
  getMarketAgent,
  forkMarketAgent,
  rateMarketAgent,
  type MarketAgentRow,
} from '@/services/adminApi'

const { Text, Paragraph } = Typography

const Market = () => {
  const [list, setList] = useState<MarketAgentRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(12)
  const [search, setSearch] = useState('')
  const [sortBy, setSortBy] = useState('download_count')
  const [loading, setLoading] = useState(false)
  const [detailVisible, setDetailVisible] = useState(false)
  const [detailAgent, setDetailAgent] = useState<MarketAgentRow | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)

  const fetchList = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listMarketAgents({ page, pageSize, search, sortBy })
      setList(res.agents || [])
      setTotal(res.total || 0)
    } catch (e: any) {
      Message.error(e?.msg || e?.message || '加载失败')
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, search, sortBy])

  useEffect(() => {
    fetchList()
  }, [fetchList])

  const handleViewDetail = async (agent: MarketAgentRow) => {
    setDetailLoading(true)
    setDetailVisible(true)
    try {
      const detail = await getMarketAgent(agent.id)
      setDetailAgent(detail)
    } catch {
      setDetailAgent(agent)
    } finally {
      setDetailLoading(false)
    }
  }

  const handleFork = async (agent: MarketAgentRow) => {
    try {
      await forkMarketAgent(agent.id)
      Message.success('Fork 成功！角色已添加到您的组织')
      fetchList()
    } catch (e: any) {
      Message.error(e?.msg || e?.message || 'Fork 失败')
    }
  }

  const handleRate = async (agentId: number, score: number) => {
    try {
      const res = await rateMarketAgent(agentId, score)
      Message.success(`评分成功！当前评分 ${res.rating.toFixed(1)}`)
      fetchList()
    } catch (e: any) {
      Message.error(e?.msg || e?.message || '评分失败')
    }
  }

  const columns = [
    {
      title: '角色',
      dataIndex: 'name',
      render: (_: any, record: MarketAgentRow) => (
        <Space>
          <div
            style={{
              width: 40,
              height: 40,
              borderRadius: 8,
              overflow: 'hidden',
              background: record.avatarUrl ? undefined : 'var(--color-fill-2)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              flexShrink: 0,
            }}
          >
            {record.avatarUrl ? (
              <Image
                src={record.avatarUrl}
                width={40}
                height={40}
                preview={false}
                style={{ objectFit: 'cover' }}
              />
            ) : (
              <User size={20} style={{ color: 'var(--color-text-3)' }} />
            )}
          </div>
          <div>
            <Text bold>{record.name}</Text>
            <br />
            <Text type="secondary" style={{ fontSize: 12 }}>
              {record.description?.slice(0, 40) || '暂无描述'}
              {(record.description?.length || 0) > 40 ? '...' : ''}
            </Text>
          </div>
        </Space>
      ),
    },
    {
      title: '标签',
      dataIndex: 'tags',
      width: 200,
      render: (_: any, record: MarketAgentRow) => (
        <Space wrap size="mini">
          {record.tags
            ?.split(',')
            .filter(Boolean)
            .slice(0, 3)
            .map((tag) => (
              <Tag key={tag} size="small" color="arcoblue">
                {tag.trim()}
              </Tag>
            ))}
          {record.tags?.split(',').filter(Boolean).length > 3 && (
            <Text type="secondary" style={{ fontSize: 11 }}>
              +{record.tags.split(',').filter(Boolean).length - 3}
            </Text>
          )}
        </Space>
      ),
    },
    {
      title: '下载',
      dataIndex: 'downloadCount',
      width: 80,
      sorter: true,
      render: (val: number) => (
        <Space size={4}>
          <Download size={14} />
          <Text>{val || 0}</Text>
        </Space>
      ),
    },
    {
      title: '评分',
      dataIndex: 'rating',
      width: 150,
      sorter: true,
      render: (val: number, record: MarketAgentRow) => (
        <Tooltip content={`${val?.toFixed(1) || '0.0'} / 5 (${record.ratingCount || 0} 人评分)`}>
          <div onClick={(e) => e.stopPropagation()}>
            <Rate
              value={val || 0}
              readonly
              size={14}
              style={{ transform: 'scale(0.9)' }}
            />
          </div>
        </Tooltip>
      ),
    },
    {
      title: '版本',
      dataIndex: 'specVersion',
      width: 70,
      render: (val: string) => val ? <Tag size="small">{val}</Tag> : '-',
    },
    {
      title: '操作',
      width: 160,
      render: (_: any, record: MarketAgentRow) => (
        <Space>
          <Button
            size="small"
            type="text"
            icon={<Eye size={14} />}
            onClick={() => handleViewDetail(record)}
          >
            详情
          </Button>
          <Popconfirm
            title="确认 Fork 此角色？"
            content="将复制到您的个人组织"
            onOk={() => handleFork(record)}
          >
            <Button size="small" type="outline" icon={<GitFork size={14} />}>
              Fork
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Card style={{ marginBottom: 16 }}>
        <Space direction="vertical" style={{ width: '100%' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Space>
              <Text style={{ fontSize: 20, fontWeight: 600 }}>角色市场</Text>
              <Text type="secondary">浏览和发现公开的角色卡</Text>
            </Space>
            <Button
              icon={<RefreshCw size={16} />}
              onClick={fetchList}
              loading={loading}
            >
              刷新
            </Button>
          </div>
          <Space>
            <Input
              placeholder="搜索角色名称 / 描述 / 标签"
              prefix={<Search size={16} />}
              value={search}
              onChange={setSearch}
              onPressEnter={fetchList}
              style={{ width: 320 }}
              allowClear
            />
            <Select
              value={sortBy}
              onChange={setSortBy}
              style={{ width: 150 }}
              options={[
                { label: '按下载量', value: 'download_count' },
                { label: '按评分', value: 'rating' },
                { label: '最新发布', value: 'created_at' },
              ]}
            />
          </Space>
        </Space>
      </Card>

      <Card>
        <Table
          columns={columns}
          data={list}
          loading={loading}
          rowKey="id"
          pagination={{
            current: page,
            pageSize,
            total,
            onChange: (p) => setPage(p),
            showTotal: true,
          }}
          onChange={(pagination, _filters, sorter) => {
            if (sorter && 'field' in sorter) {
              setSortBy(sorter.field === 'rating' ? 'rating' : 'download_count')
            }
          }}
        />
      </Card>

      {/* Detail Modal */}
      <Modal
        title={detailAgent?.name || '角色详情'}
        visible={detailVisible}
        onCancel={() => {
          setDetailVisible(false)
          setDetailAgent(null)
        }}
        footer={
          <Space>
            <Button onClick={() => setDetailVisible(false)}>关闭</Button>
            {detailAgent && (
              <>
                <Popconfirm
                  title="确认 Fork 此角色？"
                  onOk={() => {
                    handleFork(detailAgent)
                    setDetailVisible(false)
                  }}
                >
                  <Button type="outline" icon={<GitFork size={16} />}>
                    Fork
                  </Button>
                </Popconfirm>
                <Modal
                  trigger={
                    <Button type="primary" icon={<Star size={16} />}>
                      评分
                    </Button>
                  }
                  title="为角色评分"
                  footer={null}
                >
                  <div style={{ textAlign: 'center', padding: 20 }}>
                    <Text style={{ marginBottom: 12, display: 'block' }}>
                      请为 "{detailAgent.name}" 评分
                    </Text>
                    <Rate
                      defaultValue={0}
                      size={30}
                      onChange={(val) => {
                        handleRate(detailAgent.id, val)
                      }}
                    />
                  </div>
                </Modal>
              </>
            )}
          </Space>
        }
        style={{ width: 700 }}
      >
        {detailLoading ? (
          <div style={{ textAlign: 'center', padding: 40 }}>加载中...</div>
        ) : detailAgent ? (
          <Descriptions
            column={2}
            border={false}
            data={[
              { label: '名称', value: detailAgent.name },
              { label: '版本', value: detailAgent.specVersion || '-' },
              { label: '描述', value: detailAgent.description || '-' },
              {
                label: '标签',
                value: detailAgent.tags
                  ?.split(',')
                  .filter(Boolean)
                  .map((t) => <Tag key={t} size="small">{t.trim()}</Tag>) || '-',
              },
              {
                label: '下载次数',
                value: (
                  <Space size={4}>
                    <Download size={14} />
                    <Text>{detailAgent.downloadCount || 0}</Text>
                  </Space>
                ),
              },
              {
                label: '评分',
                value: (
                  <Space>
                    <Rate value={detailAgent.rating || 0} readonly size={16} />
                    <Text type="secondary">({detailAgent.ratingCount || 0})</Text>
                  </Space>
                ),
              },
            ]}
          />
        ) : null}

        {detailAgent?.personality && (
          <Card title="人格设定" style={{ marginTop: 16 }} size="small">
            <Paragraph>{detailAgent.personality}</Paragraph>
          </Card>
        )}

        {detailAgent?.scenario && (
          <Card title="世界观/场景" style={{ marginTop: 16 }} size="small">
            <Paragraph>{detailAgent.scenario}</Paragraph>
          </Card>
        )}

        {detailAgent?.creatorNote && (
          <Card title="创作者备注" style={{ marginTop: 16 }} size="small">
            <Paragraph>{detailAgent.creatorNote}</Paragraph>
          </Card>
        )}

        {detailAgent?.systemPrompt && (
          <Card title="系统提示词" style={{ marginTop: 16 }} size="small">
            <Paragraph
              style={{
                whiteSpace: 'pre-wrap',
                maxHeight: 200,
                overflow: 'auto',
                background: 'var(--color-fill-1)',
                padding: 12,
                borderRadius: 8,
              }}
            >
              {detailAgent.systemPrompt}
            </Paragraph>
          </Card>
        )}
      </Modal>
    </div>
  )
}

export default Market
