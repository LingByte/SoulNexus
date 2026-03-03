import { useState, useEffect } from 'react'
import { Search, Download, Star, Trash2, Check, Zap } from 'lucide-react'
import Button from '@/components/UI/Button'
import { listMarketplace, installMCP, uninstallMCP, getCategories, getUserInstalledMCPs } from '@/api/mcp'

interface MCPItem {
  id: number
  name: string
  description: string
  category: string
  version: string
  author: string
  icon?: string
  tags?: string[]
  features?: string[]
  downloads: number
  rating: number
  reviews: number
  is_official?: boolean
  is_featured?: boolean
}

interface Category {
  id: number
  name: string
  description?: string
}

const MCPMarketplace = () => {
  const [items, setItems] = useState<MCPItem[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedCategory, setSelectedCategory] = useState<string>('')
  const [installedIds, setInstalledIds] = useState<Set<number>>(new Set())
  const [page, setPage] = useState(1)
  const [pageSize] = useState(12)

  useEffect(() => {
    fetchData()
  }, [page, selectedCategory, searchQuery])

  const fetchData = async () => {
    try {
      setLoading(true)
      const res = await listMarketplace({
        query: searchQuery,
        category: selectedCategory,
        page,
        page_size: pageSize,
      })
      if (res.code === 200 && res.data) {
        setItems(res.data.items || [])
      }
    } catch (err) {
      console.error('Failed to fetch marketplace items', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchCategories()
    fetchUserInstalledMCPs()
  }, [])

  const fetchCategories = async () => {
    try {
      const res = await getCategories()
      if (res.code === 200) {
        setCategories(res.data || [])
      }
    } catch (err) {
      console.error('Failed to fetch categories', err)
    }
  }

  const fetchUserInstalledMCPs = async () => {
    try {
      const res = await getUserInstalledMCPs()
      if (res.code === 200 && res.data) {
        const installedMarketplaceIds = res.data.map((installation: any) => installation.marketplace_id)
        setInstalledIds(new Set(installedMarketplaceIds))
      }
    } catch (err) {
      console.error('Failed to fetch user installed MCPs', err)
    }
  }

  const handleInstall = async (id: number) => {
    try {
      const res = await installMCP(id, {})
      if (res.code === 200) {
        setInstalledIds(new Set([...installedIds, id]))
        // Show success message
        console.log('MCP installed successfully')
      }
    } catch (err) {
      console.error('Failed to install MCP', err)
    }
  }

  const handleUninstall = async (id: number) => {
    try {
      const res = await uninstallMCP(id)
      if (res.code === 200) {
        setInstalledIds(new Set([...installedIds].filter(x => x !== id)))
        // Show success message
        console.log('MCP uninstalled successfully')
      }
    } catch (err) {
      console.error('Failed to uninstall MCP', err)
    }
  }

  const renderStars = (rating: number) => {
    return (
      <div className="flex items-center gap-1">
        {[...Array(5)].map((_, i) => (
          <Star
            key={i}
            className={`w-4 h-4 ${
              i < Math.floor(rating)
                ? 'fill-yellow-400 text-yellow-400'
                : 'text-gray-300'
            }`}
          />
        ))}
        <span className="text-sm text-muted-foreground ml-1">({rating.toFixed(1)})</span>
      </div>
    )
  }

  if (loading && items.length === 0) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-muted-foreground">Loading marketplace...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6 p-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold">MCP Marketplace</h1>
        <p className="text-muted-foreground mt-1">Discover and install MCP services</p>
      </div>

      {/* Search and Filter */}
      <div className="space-y-4">
        <div className="flex gap-4 flex-col md:flex-row">
          {/* Search */}
          <div className="flex-1 relative">
            <Search className="absolute left-3 top-3 w-5 h-5 text-muted-foreground" />
            <input
              type="text"
              placeholder="Search MCP services..."
              value={searchQuery}
              onChange={(e) => {
                setSearchQuery(e.target.value)
                setPage(1)
              }}
              className="w-full pl-10 pr-4 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary"
            />
          </div>

          {/* Category Filter */}
          <select
            value={selectedCategory}
            onChange={(e) => {
              setSelectedCategory(e.target.value)
              setPage(1)
            }}
            className="px-4 py-2 border border-border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-primary"
          >
            <option value="">All Categories</option>
            {categories.map(cat => (
              <option key={cat.id} value={cat.name}>{cat.name}</option>
            ))}
          </select>
        </div>
      </div>

      {/* Items Grid */}
      {items.length === 0 ? (
        <div className="text-center py-12 bg-muted/50 rounded-lg">
          <p className="text-muted-foreground">No MCP services found</p>
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {items.map(item => (
              <div
                key={item.id}
                className="bg-card border border-border rounded-lg p-4 hover:shadow-lg transition-shadow flex flex-col"
              >
                {/* Header */}
                <div className="flex items-start justify-between mb-3">
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-1">
                      <h3 className="font-semibold text-lg">{item.name}</h3>
                      {item.is_official && (
                        <span className="px-2 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400 text-xs rounded font-medium flex items-center gap-1">
                          <Check className="w-3 h-3" />
                          Official
                        </span>
                      )}
                      {item.is_featured && (
                        <span className="px-2 py-1 bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-400 text-xs rounded font-medium flex items-center gap-1">
                          <Zap className="w-3 h-3" />
                          Featured
                        </span>
                      )}
                    </div>
                    <p className="text-sm text-muted-foreground">{item.author}</p>
                  </div>
                </div>

                {/* Description */}
                <p className="text-sm text-muted-foreground mb-3 flex-1">{item.description}</p>

                {/* Tags */}
                {item.tags && item.tags.length > 0 && (
                  <div className="flex flex-wrap gap-2 mb-3">
                    {item.tags.slice(0, 3).map(tag => (
                      <span
                        key={tag}
                        className="px-2 py-1 bg-muted text-muted-foreground text-xs rounded"
                      >
                        {tag}
                      </span>
                    ))}
                    {item.tags.length > 3 && (
                      <span className="px-2 py-1 text-xs text-muted-foreground">
                        +{item.tags.length - 3}
                      </span>
                    )}
                  </div>
                )}

                {/* Stats */}
                <div className="space-y-2 mb-4 pb-4 border-b border-border">
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">Rating:</span>
                    {renderStars(item.rating)}
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">Downloads:</span>
                    <span className="font-medium flex items-center gap-1">
                      <Download className="w-4 h-4" />
                      {item.downloads.toLocaleString()}
                    </span>
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">Version:</span>
                    <span className="font-medium">{item.version}</span>
                  </div>
                </div>

                {/* Action Button */}
                <Button
                  variant={installedIds.has(item.id) ? 'ghost' : 'primary'}
                  className="w-full"
                  onClick={() => {
                    if (installedIds.has(item.id)) {
                      handleUninstall(item.id)
                    } else {
                      handleInstall(item.id)
                    }
                  }}
                  leftIcon={
                    installedIds.has(item.id) ? (
                      <Trash2 className="w-4 h-4" />
                    ) : (
                      <Download className="w-4 h-4" />
                    )
                  }
                >
                  {installedIds.has(item.id) ? 'Uninstall' : 'Install'}
                </Button>
              </div>
            ))}
          </div>

          {/* Pagination */}
          <div className="flex items-center justify-center gap-2 mt-8">
            <Button
              variant="outline"
              onClick={() => setPage(Math.max(1, page - 1))}
              disabled={page === 1}
            >
              Previous
            </Button>
            <span className="text-sm text-muted-foreground">
              Page {page}
            </span>
            <Button
              variant="outline"
              onClick={() => setPage(page + 1)}
              disabled={items.length < pageSize}
            >
              Next
            </Button>
          </div>
        </>
      )}
    </div>
  )
}

export default MCPMarketplace
