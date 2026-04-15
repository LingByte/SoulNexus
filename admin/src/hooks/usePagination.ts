import { useState, useMemo } from 'react'

interface UsePaginationProps {
  data: any[]
  pageSize?: number
  initialPage?: number
}

interface PaginationResult {
  currentPage: number
  totalPages: number
  pageData: any[]
  totalItems: number
  hasNextPage: boolean
  hasPrevPage: boolean
  goToPage: (page: number) => void
  nextPage: () => void
  prevPage: () => void
  setPageSize: (size: number) => void
}

export const usePagination = ({
  data,
  pageSize = 10,
  initialPage = 1
}: UsePaginationProps): PaginationResult => {
  const [currentPage, setCurrentPage] = useState(initialPage)
  const [currentPageSize, setCurrentPageSize] = useState(pageSize)

  const totalPages = Math.ceil(data.length / currentPageSize)
  const totalItems = data.length

  const pageData = useMemo(() => {
    const startIndex = (currentPage - 1) * currentPageSize
    const endIndex = startIndex + currentPageSize
    return data.slice(startIndex, endIndex)
  }, [data, currentPage, currentPageSize])

  const hasNextPage = currentPage < totalPages
  const hasPrevPage = currentPage > 1

  const goToPage = (page: number) => {
    if (page >= 1 && page <= totalPages) {
      setCurrentPage(page)
    }
  }

  const nextPage = () => {
    if (hasNextPage) {
      setCurrentPage(prev => prev + 1)
    }
  }

  const prevPage = () => {
    if (hasPrevPage) {
      setCurrentPage(prev => prev - 1)
    }
  }

  const setPageSize = (size: number) => {
    setCurrentPageSize(size)
    setCurrentPage(1) // 重置到第一页
  }

  return {
    currentPage,
    totalPages,
    pageData,
    totalItems,
    hasNextPage,
    hasPrevPage,
    goToPage,
    nextPage,
    prevPage,
    setPageSize
  }
}
