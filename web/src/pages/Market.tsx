import React, { useEffect, useMemo, useState } from 'react'
import { listMarketAgents, getMarketAgent, forkMarketAgent, rateMarketAgent, getMarketShareInfo, type MarketAgent, type ShareInfo } from '@/api/assistant'
import { showAlert } from '@/utils/notification'
import { Bot, Search, Download, Star, GitFork, TrendingUp, Clock, Eye, Tag, Share2, Copy, Check, X, ImageDown } from 'lucide-react'
import { Input as ArcoInput, Drawer, Pagination, Rate, Tag as ArcoTag, Spin, Tooltip } from '@arco-design/web-react'
import { motion, AnimatePresence } from 'framer-motion'
import PageHeader from '@/components/Layout/PageHeader'
import Button from '@/components/UI/Button'
import { useAuthStore } from '@/stores/authStore'

const GRADIENT_POOL = [
  'from-purple-500 to-pink-500',
  'from-blue-500 to-cyan-500',
  'from-emerald-500 to-teal-500',
  'from-amber-500 to-orange-500',
  'from-rose-500 to-red-500',
  'from-indigo-500 to-violet-500',
  'from-sky-500 to-blue-600',
  'from-fuchsia-500 to-purple-600',
]

const pickGradient = (id: number) => GRADIENT_POOL[Math.abs(id) % GRADIENT_POOL.length]

const sortOptions = [
  { value: 'download_count', label: '下载量', icon: Download },
  { value: 'rating', label: '评分', icon: Star },
  { value: 'created_at', label: '最新', icon: Clock },
]

const Market: React.FC = () => {
  const { isAuthenticated } = useAuthStore()
  const [agents, setAgents] = useState<MarketAgent[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(12)
  const [keyword, setKeyword] = useState('')
  const [searchInput, setSearchInput] = useState('')
  const [sortBy, setSortBy] = useState<'download_count' | 'rating' | 'created_at'>('download_count')
  const [loading, setLoading] = useState(false)

  // 详情抽屉
  const [detailVisible, setDetailVisible] = useState(false)
  const [detailAgent, setDetailAgent] = useState<MarketAgent | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)

  // Fork 状态
  const [forkingId, setForkingId] = useState<number | null>(null)

  // 评分弹窗
  const [ratingVisible, setRatingVisible] = useState(false)
  const [ratingAgent, setRatingAgent] = useState<MarketAgent | null>(null)
  const [ratingValue, setRatingValue] = useState(0)
  const [ratingLoading, setRatingLoading] = useState(false)

  // 分享弹窗 + 海报
  const [shareVisible, setShareVisible] = useState(false)
  const [shareInfo, setShareInfo] = useState<ShareInfo | null>(null)
  const [shareLoading, setShareLoading] = useState(false)
  const [shareCopied, setShareCopied] = useState(false)
  const [posterDataUrl, setPosterDataUrl] = useState<string>('')
  const posterCanvasRef = React.useRef<HTMLCanvasElement>(null)

  const fetchList = async () => {
    setLoading(true)
    try {
      const res = await listMarketAgents({
        page,
        pageSize,
        search: keyword || undefined,
        sortBy,
      })
      setAgents(res.data?.agents || [])
      setTotal(res.data?.total || 0)
    } catch {
      showAlert('获取市场列表失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchList()
  }, [page, keyword, sortBy])

  const handleSearch = () => {
    setPage(1)
    setKeyword(searchInput)
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') handleSearch()
  }

  // ---- 详情 ----
  const openDetail = async (agent: MarketAgent) => {
    setDetailLoading(true)
    setDetailVisible(true)
    try {
      const res = await getMarketAgent(agent.id)
      setDetailAgent(res.data as MarketAgent)
    } catch {
      showAlert('获取详情失败', 'error')
      setDetailVisible(false)
    } finally {
      setDetailLoading(false)
    }
  }

  // ---- Fork ----
  const handleFork = async (id: number) => {
    if (!isAuthenticated) {
      showAlert('请先登录后再 Fork 角色', 'warning')
      return
    }
    setForkingId(id)
    try {
      const res = await forkMarketAgent(id)
      showAlert(`Fork 成功！已创建「${res.data?.name || '新角色'}」`, 'success')
      await fetchList()
    } catch (err: any) {
      showAlert(err?.msg || 'Fork 失败', 'error')
    } finally {
      setForkingId(null)
    }
  }

  // ---- 评分 ----
  const openRating = (agent: MarketAgent) => {
    if (!isAuthenticated) {
      showAlert('请先登录后再评分', 'warning')
      return
    }
    setRatingAgent(agent)
    setRatingValue(0)
    setRatingVisible(true)
  }

  const handleRating = async () => {
    if (!ratingAgent || ratingValue < 1) return
    setRatingLoading(true)
    try {
      await rateMarketAgent(ratingAgent.id, ratingValue)
      showAlert('评分成功', 'success')
      setRatingVisible(false)
      await fetchList()
    } catch (err: any) {
      showAlert(err?.msg || '评分失败', 'error')
    } finally {
      setRatingLoading(false)
    }
  }

  // 分享
  const openShare = async (agent: MarketAgent) => {
    setShareVisible(true)
    setShareInfo(null)
    setShareCopied(false)
    setShareLoading(true)
    try {
      const res = await getMarketShareInfo(agent.id)
      if (res.code === 200 && res.data) {
        setShareInfo(res.data)
        // 异步生成海报
        setTimeout(() => generatePoster(agent, res.data), 100)
      }
    } catch {
      // 使用客户端信息兜底
      const fallback: ShareInfo = {
        url: `${window.location.origin}/market?open=${agent.id}`,
        title: agent.name || '',
        description: (agent as any).description || '',
        avatar: (agent as any).avatarUrl || '',
        rating: agent.rating || 0,
        ratingCount: agent.ratingCount || 0,
        downloadCount: agent.downloadCount || 0,
        agentId: agent.id,
      }
      setShareInfo(fallback)
    } finally {
      setShareLoading(false)
    }
  }

  const copyShareLink = async () => {
    if (!shareInfo?.url) return
    try {
      await navigator.clipboard.writeText(shareInfo.url)
      setShareCopied(true)
      setTimeout(() => setShareCopied(false), 2000)
    } catch {
      // fallback
      const ta = document.createElement('textarea')
      ta.value = shareInfo.url
      document.body.appendChild(ta)
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
      setShareCopied(true)
      setTimeout(() => setShareCopied(false), 2000)
    }
  }

  const generatePoster = (agent: MarketAgent, info: ShareInfo) => {
    const canvas = posterCanvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    const w = 800
    const h = 500
    canvas.width = w
    canvas.height = h

    // Background gradient
    const gradient = ctx.createLinearGradient(0, 0, w, h)
    gradient.addColorStop(0, '#1e1b4b')
    gradient.addColorStop(0.5, '#312e81')
    gradient.addColorStop(1, '#4338ca')
    ctx.fillStyle = gradient
    ctx.fillRect(0, 0, w, h)

    // Decorative circles
    ctx.fillStyle = 'rgba(255,255,255,0.03)'
    ctx.beginPath()
    ctx.arc(650, 100, 180, 0, Math.PI * 2)
    ctx.fill()
    ctx.beginPath()
    ctx.arc(100, 400, 120, 0, Math.PI * 2)
    ctx.fill()

    // Title
    ctx.fillStyle = '#ffffff'
    ctx.font = 'bold 32px system-ui, -apple-system, sans-serif'
    ctx.fillText(truncateForCanvas(ctx, info.title, 500), 60, 100)

    // Description
    ctx.font = '16px system-ui, -apple-system, sans-serif'
    ctx.fillStyle = 'rgba(255,255,255,0.75)'
    const desc = info.description || 'Discover this amazing AI assistant on SoulNexus'
    wrapText(ctx, desc, 60, 140, 500, 22)

    // Stats
    ctx.font = '14px system-ui, -apple-system, sans-serif'
    ctx.fillStyle = 'rgba(255,255,255,0.6)'
    const ratingStr = `★ ${(info.rating || 0).toFixed(1)} (${info.ratingCount || 0})`
    const dlStr = `↓ ${(info.downloadCount || 0)} downloads`
    ctx.fillText(ratingStr, 60, 260)
    ctx.fillText(dlStr, 60, 285)

    // Brand
    ctx.font = 'bold 18px system-ui, -apple-system, sans-serif'
    ctx.fillStyle = 'rgba(255,255,255,0.9)'
    ctx.fillText('SoulNexus', 60, 400)

    ctx.font = '12px system-ui, -apple-system, sans-serif'
    ctx.fillStyle = 'rgba(255,255,255,0.4)'
    ctx.fillText('Discover & Share AI Assistants', 60, 422)

    // QR placeholder (right side)
    ctx.fillStyle = 'rgba(255,255,255,0.1)'
    ctx.fillRect(590, 310, 150, 150)
    ctx.fillStyle = 'rgba(255,255,255,0.5)'
    ctx.font = '12px system-ui, -apple-system, sans-serif'
    ctx.textAlign = 'center'
    ctx.fillText('Scan to visit', 665, 380)
    ctx.fillText(info.url.length > 40 ? info.url.slice(0, 38) + '...' : info.url, 665, 400)
    ctx.textAlign = 'left'

    // Save as data URL for display
    setPosterDataUrl(canvas.toDataURL('image/png'))
  }

  const downloadPoster = () => {
    if (!posterDataUrl) return
    const link = document.createElement('a')
    link.download = `soulnx-share-${shareInfo?.agentId || 'agent'}.png`
    link.href = posterDataUrl
    link.click()
  }

  const tagsToList = (tags?: string): string[] => {
    if (!tags) return []
    return tags.split(',').map(t => t.trim()).filter(Boolean)
  }

  const fmtDate = (iso?: string) => (iso ? iso.slice(0, 10) : '')

  return (
    <div className="flex flex-col h-full dark:bg-neutral-900">
      <PageHeader
        title="角色市场"
        subtitle="发现和 Fork 社区创作的公开角色"
        actions={
          <div className="flex items-center gap-3">
            {/* 搜索 */}
            <div className="relative">
              <ArcoInput
                size="large"
                className="!h-10 !w-64 !text-base"
                value={searchInput}
                onChange={(val) => setSearchInput(val)}
                onKeyDown={handleKeyDown}
                placeholder="搜索角色名称 / 描述 / 标签..."
                prefix={<Search className="h-4 w-4 text-muted-foreground" />}
                allowClear
              />
            </div>

            {/* 排序切换 */}
            <div className="flex items-center gap-1 bg-gray-100 dark:bg-neutral-800 rounded-lg p-1">
              {sortOptions.map((opt) => {
                const Icon = opt.icon
                return (
                  <button
                    key={opt.value}
                    onClick={() => { setSortBy(opt.value as any); setPage(1) }}
                    className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-all ${
                      sortBy === opt.value
                        ? 'bg-white dark:bg-neutral-700 text-primary shadow-sm'
                        : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200'
                    }`}
                  >
                    <Icon className="w-3.5 h-3.5" />
                    {opt.label}
                  </button>
                )
              })}
            </div>
          </div>
        }
      />

      <div className="flex-1 overflow-auto">
        <div className="max-w-7xl w-full mx-auto px-4 pt-6 pb-10">
          <Spin loading={loading} className="w-full">
            {/* 卡片网格 */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
              <AnimatePresence>
                {agents.map((agent, index) => {
                  const gradient = pickGradient(agent.id)
                  return (
                    <motion.div
                      key={agent.id}
                      initial={{ opacity: 0, y: 12 }}
                      animate={{ opacity: 1, y: 0 }}
                      exit={{ opacity: 0, scale: 0.95 }}
                      transition={{ delay: index * 0.03, duration: 0.2 }}
                      whileHover={{ y: -4 }}
                      className="group bg-white dark:bg-neutral-800/80 rounded-2xl overflow-hidden border border-gray-200/70 dark:border-neutral-700/60 hover:border-primary/40 dark:hover:border-primary/40 shadow-[0_1px_2px_rgba(0,0,0,0.03)] hover:shadow-lg hover:shadow-primary/5 transition-all duration-200"
                    >
                      {/* 卡片头部 - 渐变背景 + 头像 */}
                      <div className={`h-24 bg-gradient-to-br ${gradient} relative`}>
                        <div className="absolute inset-0 bg-black/10" />
                        <div className="absolute -bottom-6 left-4 w-16 h-16 rounded-2xl border-4 border-white dark:border-neutral-800 overflow-hidden shadow-lg bg-white dark:bg-neutral-800">
                          {agent.avatarUrl ? (
                            <img src={agent.avatarUrl} alt={agent.name} className="w-full h-full object-cover" />
                          ) : (
                            <div className={`w-full h-full bg-gradient-to-br ${gradient} flex items-center justify-center`}>
                              <Bot className="h-8 w-8" />
                            </div>
                          )}
                        </div>
                      </div>

                      {/* 卡片内容 */}
                      <div className="p-4 pt-8">
                        <h3
                          className="font-semibold text-[15px] text-gray-900 dark:text-gray-100 truncate cursor-pointer hover:text-primary transition-colors"
                          onClick={() => openDetail(agent)}
                        >
                          {agent.name}
                        </h3>

                        {/* 描述 */}
                        {agent.description && (
                          <p className="mt-2 text-xs text-gray-500 dark:text-gray-400 line-clamp-2 leading-relaxed">
                            {agent.description}
                          </p>
                        )}

                        {/* 标签 */}
                        {agent.tags && (
                          <div className="mt-2.5 flex flex-wrap gap-1">
                            {tagsToList(agent.tags).slice(0, 4).map(tag => (
                              <span
                                key={tag}
                                className="px-2 py-0.5 rounded-md bg-indigo-50 dark:bg-indigo-900/20 text-indigo-600 dark:text-indigo-300 text-[11px] font-medium"
                              >
                                {tag.length > 8 ? tag.slice(0, 8) + '…' : tag}
                              </span>
                            ))}
                            {tagsToList(agent.tags).length > 4 && (
                              <span className="px-2 py-0.5 text-[11px] text-gray-400">+{tagsToList(agent.tags).length - 4}</span>
                            )}
                          </div>
                        )}

                        {/* 底栏：统计 + 操作 */}
                        <div className="mt-4 pt-3 border-t border-gray-100 dark:border-neutral-700/60 flex items-center justify-between">
                          <div className="flex items-center gap-3 text-[11px] text-gray-400 dark:text-gray-500">
                            <span className="flex items-center gap-1">
                              <Download className="w-3 h-3" />
                              {agent.downloadCount || 0}
                            </span>
                            <span className="flex items-center gap-1 text-amber-500">
                              <Star className="w-3 h-3 fill-current" />
                              {typeof agent.rating === 'number' ? agent.rating.toFixed(1) : '0.0'}
                            </span>
                          </div>
                          <div className="flex items-center gap-1">
                            <button
                              onClick={(e) => { e.stopPropagation(); openDetail(agent) }}
                              className="p-1.5 rounded-lg text-gray-400 hover:text-primary hover:bg-primary/10 transition-colors"
                              title="查看详情"
                            >
                              <Eye className="w-4 h-4" />
                            </button>
                            <button
                              onClick={(e) => { e.stopPropagation(); openShare(agent) }}
                              className="p-1.5 rounded-lg text-gray-400 hover:text-green-500 hover:bg-green-50 dark:hover:bg-green-900/20 transition-colors"
                              title="分享"
                            >
                              <Share2 className="w-4 h-4" />
                            </button>
                            <button
                              onClick={(e) => { e.stopPropagation(); openRating(agent) }}
                              className="p-1.5 rounded-lg text-gray-400 hover:text-amber-500 hover:bg-amber-50 dark:hover:bg-amber-900/20 transition-colors"
                              title="评分"
                            >
                              <Star className="w-4 h-4" />
                            </button>
                            <button
                              onClick={(e) => { e.stopPropagation(); handleFork(agent.id) }}
                              disabled={forkingId === agent.id}
                              className={`p-1.5 rounded-lg transition-colors ${
                                forkingId === agent.id
                                  ? 'text-primary/50 cursor-not-allowed'
                                  : 'text-gray-400 hover:text-primary hover:bg-primary/10'
                              }`}
                              title="Fork 到我的组织"
                            >
                              {forkingId === agent.id ? (
                                <div className="w-4 h-4 border-2 border-primary/30 border-t-primary rounded-full animate-spin" />
                              ) : (
                                <GitFork className="w-4 h-4" />
                              )}
                            </button>
                          </div>
                        </div>
                      </div>
                    </motion.div>
                  )
                })}
              </AnimatePresence>

              {/* 空状态 */}
              {!loading && agents.length === 0 && (
                <div className="col-span-full text-center py-20">
                  <div className="inline-flex items-center justify-center w-20 h-20 rounded-full bg-gray-100 dark:bg-neutral-800 mb-4">
                    <Bot className="w-10 h-10 text-gray-300 dark:text-gray-600" />
                  </div>
                  <p className="text-gray-500 dark:text-gray-400 text-sm">
                    {keyword ? '没有找到匹配的角色' : '市场暂无公开角色'}
                  </p>
                  {keyword && (
                    <Button
                      variant="ghost"
                      size="sm"
                      className="mt-3"
                      onClick={() => { setSearchInput(''); setKeyword(''); setPage(1) }}
                    >
                      清除搜索
                    </Button>
                  )}
                </div>
              )}
            </div>

            {/* 分页 */}
            {total > pageSize && (
              <div className="flex justify-center mt-8">
                <Pagination
                  current={page}
                  total={total}
                  pageSize={pageSize}
                  onChange={(p) => setPage(p)}
                  showTotal
                  sizeCanChange={false}
                />
              </div>
            )}
          </Spin>
        </div>
      </div>

      {/* ========== 详情抽屉 ========== */}
      <Drawer
        width={480}
        title={null}
        visible={detailVisible}
        onCancel={() => { setDetailVisible(false); setDetailAgent(null) }}
        footer={null}
        className="!p-0"
      >
        {detailLoading ? (
          <div className="flex items-center justify-center py-20">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
          </div>
        ) : detailAgent ? (
          <div className="flex flex-col h-full">
            {/* 头部 */}
            <div className="relative px-6 pt-8 pb-6 bg-gradient-to-b from-primary/5 to-transparent">
              <div className="flex items-center gap-4">
                <div className="relative">
                  {detailAgent.avatarUrl ? (
                    <img src={detailAgent.avatarUrl} alt={detailAgent.name} className="w-16 h-16 rounded-2xl object-cover shadow-md" />
                  ) : (
                    <div className={`w-16 h-16 rounded-2xl bg-gradient-to-br ${pickGradient(detailAgent.id)} flex items-center justify-center shadow-md`}>
                      <Bot className="h-8 w-8" />
                    </div>
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <h2 className="text-xl font-bold text-gray-900 dark:text-gray-100 truncate">{detailAgent.name}</h2>
                  <div className="flex items-center gap-2 mt-1">
                    <span className="text-xs text-gray-400 font-mono">#{detailAgent.id}</span>
                    <ArcoTag size="small" color="green">公开</ArcoTag>
                  </div>
                </div>
              </div>

              {/* 统计 + 操作 */}
              <div className="flex items-center gap-4 mt-4">
                <div className="flex items-center gap-1.5 text-sm text-gray-500 dark:text-gray-400">
                  <Download className="w-4 h-4" />
                  <span>{detailAgent.downloadCount || 0} 次下载</span>
                </div>
                <div className="flex items-center gap-1.5 text-sm text-amber-500">
                  <Star className="w-4 h-4 fill-current" />
                  <span>{typeof detailAgent.rating === 'number' ? detailAgent.rating.toFixed(1) : '0.0'}</span>
                  <span className="text-gray-400">({detailAgent.ratingCount || 0})</span>
                </div>
                <div className="flex-1" />
                <button
                  onClick={() => openShare(detailAgent)}
                  className="flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs font-medium text-green-600 bg-green-50 dark:bg-green-900/20 hover:bg-green-100 dark:hover:bg-green-900/40 transition-colors"
                >
                  <Share2 className="w-3.5 h-3.5" /> 分享
                </button>
                <button
                  onClick={() => openRating(detailAgent)}
                  className="flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs font-medium text-amber-600 bg-amber-50 dark:bg-amber-900/20 hover:bg-amber-100 dark:hover:bg-amber-900/40 transition-colors"
                >
                  <Star className="w-3.5 h-3.5" /> 评分
                </button>
                <Button
                  variant="primary"
                  size="sm"
                  loading={forkingId === detailAgent.id}
                  onClick={() => handleFork(detailAgent.id)}
                  leftIcon={<GitFork className="w-4 h-4" />}
                >
                  Fork
                </Button>
              </div>
            </div>

            {/* 内容 */}
            <div className="flex-1 overflow-auto px-6 pb-8 space-y-5">
              {detailAgent.description && (
                <section>
                  <h4 className="text-xs font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider mb-2">描述</h4>
                  <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed whitespace-pre-wrap">{detailAgent.description}</p>
                </section>
              )}

              {detailAgent.personality && (
                <section>
                  <h4 className="text-xs font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider mb-2">人格设定</h4>
                  <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed whitespace-pre-wrap">{detailAgent.personality}</p>
                </section>
              )}

              {detailAgent.scenario && (
                <section>
                  <h4 className="text-xs font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider mb-2">世界观 / 场景</h4>
                  <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed whitespace-pre-wrap">{detailAgent.scenario}</p>
                </section>
              )}

              {detailAgent.tags && (
                <section>
                  <h4 className="text-xs font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider mb-2">标签</h4>
                  <div className="flex flex-wrap gap-1.5">
                    {tagsToList(detailAgent.tags).map(tag => (
                      <span key={tag} className="px-2.5 py-1 rounded-md bg-indigo-50 dark:bg-indigo-900/20 text-indigo-600 dark:text-indigo-300 text-xs font-medium">{tag}</span>
                    ))}
                  </div>
                </section>
              )}

              <section>
                <h4 className="text-xs font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider mb-2">基本信息</h4>
                <div className="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
                  {detailAgent.speaker && (
                    <>
                      <span className="text-gray-400 dark:text-gray-500">音色</span>
                      <span className="text-gray-700 dark:text-gray-300">{detailAgent.speaker}</span>
                    </>
                  )}
                  {detailAgent.llmModel && (
                    <>
                      <span className="text-gray-400 dark:text-gray-500">模型</span>
                      <span className="text-gray-700 dark:text-gray-300">{detailAgent.llmModel}</span>
                    </>
                  )}
                  {detailAgent.specVersion && (
                    <>
                      <span className="text-gray-400 dark:text-gray-500">规范版本</span>
                      <span className="text-gray-700 dark:text-gray-300">{detailAgent.specVersion}</span>
                    </>
                  )}
                  <span className="text-gray-400 dark:text-gray-500">创建时间</span>
                  <span className="text-gray-700 dark:text-gray-300">{fmtDate(detailAgent.createdAt)}</span>
                  {detailAgent.forkedFrom && (
                    <>
                      <span className="text-gray-400 dark:text-gray-500">Fork 自</span>
                      <span className="text-gray-700 dark:text-gray-300">#{detailAgent.forkedFrom}</span>
                    </>
                  )}
                </div>
              </section>
            </div>
          </div>
        ) : null}
      </Drawer>

      {/* ========== 评分弹窗 ========== */}
      <AnimatePresence>
        {ratingVisible && ratingAgent && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
            onClick={() => setRatingVisible(false)}
          >
            <motion.div
              initial={{ scale: 0.9, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.9, opacity: 0 }}
              className="bg-white dark:bg-neutral-800 rounded-2xl shadow-2xl p-6 w-80 mx-4"
              onClick={(e) => e.stopPropagation()}
            >
              <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-1">评分</h3>
              <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">{ratingAgent.name}</p>
              <div className="flex justify-center mb-6">
                <Rate
                  value={ratingValue}
                  onChange={(val) => setRatingValue(val)}
                  size={32}
                  style={{ gap: 8 }}
                />
              </div>
              <div className="flex justify-end gap-3">
                <Button variant="ghost" size="sm" onClick={() => setRatingVisible(false)}>
                  取消
                </Button>
                <Button
                  variant="primary"
                  size="sm"
                  onClick={handleRating}
                  loading={ratingLoading}
                  disabled={ratingValue < 1}
                >
                  提交评分
                </Button>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* ========== 分享弹窗 + 海报 ========== */}
      <AnimatePresence>
        {shareVisible && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
            onClick={() => setShareVisible(false)}
          >
            <motion.div
              initial={{ scale: 0.9, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.9, opacity: 0 }}
              className="bg-white dark:bg-neutral-800 rounded-2xl shadow-2xl p-6 w-[420px] max-w-[95vw] mx-4 max-h-[90vh] overflow-y-auto"
              onClick={(e) => e.stopPropagation()}
            >
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                  分享角色
                </h3>
                <button
                  onClick={() => setShareVisible(false)}
                  className="p-1 rounded-lg text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              {shareLoading ? (
                <div className="flex items-center justify-center py-12">
                  <Spin size={24} />
                </div>
              ) : shareInfo ? (
                <div className="space-y-4">
                  {/* 角色信息 */}
                  <div className="flex items-center gap-3 p-3 rounded-xl bg-gray-50 dark:bg-neutral-700/50">
                    <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-indigo-500 to-purple-500 flex items-center justify-center text-white font-bold text-sm shrink-0">
                      {shareInfo.title?.charAt(0) || '?'}
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
                        {shareInfo.title}
                      </p>
                      <p className="text-xs text-gray-400 dark:text-gray-500 truncate">
                        ★ {(shareInfo.rating || 0).toFixed(1)} · ↓ {shareInfo.downloadCount || 0} 次下载
                      </p>
                    </div>
                  </div>

                  {/* 分享链接 */}
                  <div>
                    <label className="text-xs font-medium text-gray-500 dark:text-gray-400 mb-1.5 block">
                      分享链接
                    </label>
                    <div className="flex gap-2">
                      <input
                        readOnly
                        value={shareInfo.url}
                        className="flex-1 px-3 py-2 text-xs rounded-lg border border-gray-200 dark:border-neutral-600 bg-gray-50 dark:bg-neutral-700 text-gray-700 dark:text-gray-300 focus:outline-none"
                      />
                      <button
                        onClick={copyShareLink}
                        className="shrink-0 px-3 py-2 rounded-lg text-xs font-medium bg-primary/10 text-primary hover:bg-primary/20 transition-colors flex items-center gap-1"
                      >
                        {shareCopied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
                        {shareCopied ? '已复制' : '复制'}
                      </button>
                    </div>
                  </div>

                  {/* 海报预览 + 下载 */}
                  <div>
                    <label className="text-xs font-medium text-gray-500 dark:text-gray-400 mb-1.5 block">
                      分享海报
                    </label>
                    <div className="rounded-xl overflow-hidden border border-gray-200 dark:border-neutral-600 bg-gray-100 dark:bg-neutral-900">
                      {posterDataUrl ? (
                        <img src={posterDataUrl} alt="分享海报" className="w-full h-auto block" />
                      ) : (
                        <div className="flex items-center justify-center h-40 text-gray-400 text-xs">
                          生成中…
                        </div>
                      )}
                    </div>
                    <button
                      onClick={downloadPoster}
                      disabled={!posterDataUrl}
                      className="mt-2 w-full py-2 rounded-lg text-xs font-medium bg-green-50 dark:bg-green-900/20 text-green-600 dark:text-green-400 hover:bg-green-100 dark:hover:bg-green-900/40 transition-colors flex items-center justify-center gap-1.5 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      <ImageDown className="w-4 h-4" />
                      下载海报图片
                    </button>
                  </div>
                </div>
              ) : (
                <p className="text-sm text-gray-400 text-center py-8">获取分享信息失败</p>
              )}
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* 隐藏 canvas 用于海报生成 */}
      <canvas ref={posterCanvasRef} className="hidden" />
    </div>
  )
}

// 辅助函数
function truncateForCanvas(ctx: CanvasRenderingContext2D, text: string, maxWidth: number): string {
  if (ctx.measureText(text).width <= maxWidth) return text
  let result = text
  while (ctx.measureText(result + '…').width > maxWidth && result.length > 0) {
    result = result.slice(0, -1)
  }
  return result + '…'
}

function wrapText(ctx: CanvasRenderingContext2D, text: string, x: number, y: number, maxWidth: number, lineHeight: number) {
  const words = text.split('')
  let line = ''
  let cy = y
  for (let i = 0; i < words.length; i++) {
    const testLine = line + words[i]
    if (ctx.measureText(testLine).width > maxWidth && line.length > 0) {
      ctx.fillText(line, x, cy)
      line = words[i]
      cy += lineHeight
    } else {
      line = testLine
    }
  }
  if (line.length > 0) {
    ctx.fillText(line, x, cy)
  }
}

export default Market
