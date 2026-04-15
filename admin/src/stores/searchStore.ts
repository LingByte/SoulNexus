import { create } from 'zustand'

export interface SearchResult {
  id: string
  title: string
  description?: string
  type: 'page' | 'component' | 'notification' | 'user' | 'content'
  url?: string
  icon?: string
  metadata?: Record<string, any>
  fragments?: Record<string, string[]> // 高亮片段
}

interface SearchState {
  isOpen: boolean
  query: string
  results: SearchResult[]
  isLoading: boolean
  selectedIndex: number
  searchEnabled: boolean | null // null 表示未检查，true/false 表示已检查
  isCheckingStatus: boolean
  
  // Actions
  openSearch: () => void
  closeSearch: () => void
  setQuery: (query: string) => void
  setResults: (results: SearchResult[]) => void
  setLoading: (loading: boolean) => void
  setSelectedIndex: (index: number) => void
  selectNext: () => void
  selectPrevious: () => void
  selectResult: (index: number) => void
  clearSearch: () => void
  checkSearchStatus: () => Promise<void>
  enableSearch: () => Promise<void>
}

// 防抖函数
let searchDebounceTimer: NodeJS.Timeout | null = null

export const useSearchStore = create<SearchState>((set, get) => ({
  isOpen: false,
  query: '',
  results: [],
  isLoading: false,
  selectedIndex: 0,
  searchEnabled: null,
  isCheckingStatus: false,

  openSearch: async () => {
    // 打开搜索时检查搜索状态
    const { searchEnabled } = get()
    if (searchEnabled === null) {
      await get().checkSearchStatus()
    }
    set({ isOpen: true, selectedIndex: 0 })
  },

  closeSearch: () => {
    // 清除防抖定时器
    if (searchDebounceTimer) {
      clearTimeout(searchDebounceTimer)
      searchDebounceTimer = null
    }
    set({ isOpen: false, query: '', results: [], selectedIndex: 0 })
  },

  setQuery: async (query: string) => {
    set({ query, selectedIndex: 0 })
    
    if (query.trim() === '') {
      // 清除之前的防抖定时器
      if (searchDebounceTimer) {
        clearTimeout(searchDebounceTimer)
        searchDebounceTimer = null
      }
      set({ results: [], isLoading: false })
      return
    }

    // 清除之前的防抖定时器
    if (searchDebounceTimer) {
      clearTimeout(searchDebounceTimer)
      searchDebounceTimer = null
    }

    // 设置防抖，延迟 300ms 执行搜索
    searchDebounceTimer = setTimeout(async () => {
      // 检查搜索是否启用
      const { searchEnabled } = get()
      if (searchEnabled === null) {
        await get().checkSearchStatus()
      }

      if (!get().searchEnabled) {
        // 搜索未启用，返回空结果
        set({ 
          results: [],
          isLoading: false,
          selectedIndex: 0
        })
        return
      }

      set({ isLoading: true })
      
      try {
        // 可以在这里调用后端搜索接口
        // const searchResponse = await search({
        //   keyword: query,
        //   size: 20,
        //   from: 0,
        //   highlight: true,
        //   highlightFields: ['title', 'description', 'content'],
        //   fragmentSize: 100,
        //   maxFragments: 3
        // })
        // 
        // if (searchResponse.code === 200 && searchResponse.data) {
        //   const data = searchResponse.data
        //   const hits = data.hits || data.Hits || []
        //   const results = hits.map(convertBackendHitToSearchResult)
        //   
        //   set({ 
        //     results,
        //     isLoading: false,
        //     selectedIndex: 0
        //   })
        // } else {
        //   set({ 
        //     results: [],
        //     isLoading: false,
        //     selectedIndex: 0
        //   })
        // }
        
        // 暂时返回空结果
        set({ 
          results: [],
          isLoading: false,
          selectedIndex: 0
        })
      } catch (error) {
        console.error('Search error:', error)
        set({ 
          results: [],
          isLoading: false,
          selectedIndex: 0
        })
      }
    }, 300) // 300ms 防抖延迟
  },

  checkSearchStatus: async () => {
    const { isCheckingStatus } = get()
    if (isCheckingStatus) return

    set({ isCheckingStatus: true })
    try {
      // 可以在这里调用API检查搜索状态
      // const response = await getSearchStatus()
      // if (response.code === 200 && response.data) {
      //   set({ searchEnabled: response.data.enabled })
      // } else {
      //   set({ searchEnabled: false })
      // }
      set({ searchEnabled: false })
    } catch (error) {
      console.error('Failed to check search status:', error)
      set({ searchEnabled: false })
    } finally {
      set({ isCheckingStatus: false })
    }
  },

  enableSearch: async () => {
    try {
      // 可以在这里调用API启用搜索
      // const response = await enableSearch()
      // if (response.code === 200) {
      //   set({ searchEnabled: true })
      // }
      set({ searchEnabled: true })
    } catch (error) {
      console.error('Failed to enable search:', error)
      throw error
    }
  },

  setResults: (results: SearchResult[]) => {
    set({ results })
  },

  setLoading: (isLoading: boolean) => {
    set({ isLoading })
  },

  setSelectedIndex: (selectedIndex: number) => {
    set({ selectedIndex })
  },

  selectNext: () => {
    const { results, selectedIndex } = get()
    if (results.length > 0) {
      set({ selectedIndex: (selectedIndex + 1) % results.length })
    }
  },

  selectPrevious: () => {
    const { results, selectedIndex } = get()
    if (results.length > 0) {
      set({ selectedIndex: selectedIndex === 0 ? results.length - 1 : selectedIndex - 1 })
    }
  },

  selectResult: (index: number) => {
    const { results } = get()
    if (results[index]) {
      set({ selectedIndex: index })
    }
  },

  clearSearch: () => {
    set({ query: '', results: [], selectedIndex: 0 })
  }
}))
