import { useCallback, useEffect, useRef, useState } from 'react'
import type { ApiResponse } from '@/utils/request'
import type { Paginated } from '@/api/types'

interface UsePaginatedListOptions<T> {
  fetcher: (page: number, size: number) => Promise<ApiResponse<Paginated<T>>>
  pageSize?: number
}

interface UsePaginatedListResult<T> {
  rows: T[]
  total: number
  page: number
  loading: boolean
  setPage: (p: number | ((prev: number) => number)) => void
  refresh: () => void
}

export function usePaginatedList<T>({
  fetcher,
  pageSize = 20,
}: UsePaginatedListOptions<T>): UsePaginatedListResult<T> {
  const [rows, setRows] = useState<T[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [nonce, setNonce] = useState(0)
  const mountedRef = useRef(true)
  const fetcherRef = useRef(fetcher)
  fetcherRef.current = fetcher

  useEffect(() => {
    let cancelled = false
    mountedRef.current = true
    setLoading(true)
    fetcherRef.current(page, pageSize).then((res) => {
      if (cancelled || !mountedRef.current) return
      if (res.code === 200 && res.data) {
        setRows(res.data.list || [])
        setTotal(res.data.total || 0)
      }
    }).catch(() => { /* swallow */ })
      .finally(() => { if (!cancelled && mountedRef.current) setLoading(false) })
    return () => { cancelled = true; mountedRef.current = false }
  }, [page, pageSize, nonce])

  const refresh = useCallback(() => setNonce((n) => n + 1), [])

  return { rows, total, page, loading, setPage, refresh }
}
