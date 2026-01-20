/**
 * 高亮工具函数
 */

/**
 * 高亮文本中的关键词
 * @param text 原始文本
 * @param keyword 关键词
 * @returns 高亮后的 HTML 字符串
 */
export const highlightText = (text: string, keyword: string): string => {
  if (!text || !keyword) return text
  
  const regex = new RegExp(`(${keyword.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi')
  return text.replace(regex, '<mark class="bg-yellow-200 dark:bg-yellow-800 px-0.5 rounded">$1</mark>')
}

/**
 * 处理已包含高亮标签的文本（从后端返回的 fragments）
 * @param text 包含 <mark> 标签的文本
 * @returns 处理后的 HTML 字符串
 */
export const processHighlightedText = (text: string): string => {
  if (!text) return ''
  
  // 将后端的 <mark> 标签转换为前端样式
  return text.replace(
    /<mark>(.*?)<\/mark>/g,
    '<mark class="bg-yellow-200 dark:bg-yellow-800 px-0.5 rounded">$1</mark>'
  )
}

/**
 * 从 URL 参数中获取搜索和高亮信息
 * @returns 搜索关键词和高亮片段信息
 */
export const getSearchParamsFromURL = () => {
  const params = new URLSearchParams(window.location.search)
  const search = params.get('search') || ''
  const highlightStr = params.get('highlight')
  const resultId = params.get('resultId') || ''
  
  let highlight: Record<string, string[]> | null = null
  if (highlightStr) {
    try {
      highlight = JSON.parse(decodeURIComponent(highlightStr))
    } catch (e) {
      console.warn('Failed to parse highlight data:', e)
    }
  }
  
  return { search, highlight, resultId }
}

/**
 * 高亮显示文本内容
 * @param text 原始文本
 * @param searchKeyword 搜索关键词
 * @param highlightFragments 高亮片段（优先使用）
 * @returns 高亮后的 HTML 字符串
 */
export const highlightContent = (
  text: string,
  searchKeyword?: string,
  highlightFragments?: Record<string, string[]>
): string => {
  if (!text) return ''
  
  // 优先使用后端返回的高亮片段
  if (highlightFragments) {
    // 尝试从 fragments 中找到匹配的片段
    const fragments = highlightFragments.content || highlightFragments.description || highlightFragments.title
    if (fragments && fragments.length > 0) {
      return processHighlightedText(fragments[0])
    }
  }
  
  // 如果没有高亮片段，使用关键词高亮
  if (searchKeyword) {
    return highlightText(text, searchKeyword)
  }
  
  return text
}

