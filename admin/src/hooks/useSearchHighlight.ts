import { useEffect, useState } from 'react'
import { getSearchParamsFromURL } from '@/utils/highlight'

/**
 * 用于从搜索结果跳转的页面获取搜索和高亮信息的 Hook
 * @returns 搜索关键词、高亮片段、结果ID
 */
export const useSearchHighlight = () => {
  const [searchKeyword, setSearchKeyword] = useState('')
  const [highlightFragments, setHighlightFragments] = useState<Record<string, string[]> | null>(null)
  const [highlightResultId, setHighlightResultId] = useState('')

  useEffect(() => {
    const { search, highlight, resultId } = getSearchParamsFromURL()
    if (search) {
      setSearchKeyword(search)
    }
    if (highlight) {
      setHighlightFragments(highlight)
    }
    if (resultId) {
      setHighlightResultId(resultId)
    }
  }, [])

  return {
    searchKeyword,
    highlightFragments,
    highlightResultId
  }
}

