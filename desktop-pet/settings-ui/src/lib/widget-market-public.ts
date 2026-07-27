export type PublicWidgetMarketItem = {
  id: string
  displayName: string
  description?: string
  category: string
  avatarUrl?: string
  jsSourceId: string
  embedPath: string
  installCount: number
}

type PageBody = {
  list?: PublicWidgetMarketItem[]
  total?: number
}

export async function fetchPublicWidgetMarket(
  serverBase: string,
  opts?: { page?: number; size?: number; keyword?: string; category?: string },
): Promise<{ list: PublicWidgetMarketItem[]; total: number }> {
  const base = String(serverBase || '').replace(/\/+$/, '')
  const params = new URLSearchParams()
  params.set('page', String(opts?.page ?? 1))
  params.set('size', String(opts?.size ?? 48))
  if (opts?.keyword?.trim()) params.set('keyword', opts.keyword.trim())
  if (opts?.category?.trim() && opts.category !== 'all') params.set('category', opts.category.trim())
  const url = `${base}/public/widget-market/items?${params.toString()}`
  const res = await fetch(url, { method: 'GET', cache: 'no-store' })
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  const body = (await res.json()) as { code?: number; data?: PageBody; msg?: string }
  if (body.code !== 200 && body.code !== 0) {
    throw new Error(body.msg || '加载挂件市场失败')
  }
  return {
    list: body.data?.list || [],
    total: body.data?.total ?? 0,
  }
}

export async function trackPublicWidgetDownload(serverBase: string, id: string) {
  const base = String(serverBase || '').replace(/\/+$/, '')
  try {
    await fetch(`${base}/public/widget-market/items/${encodeURIComponent(id)}/download`, {
      method: 'POST',
    })
  } catch {
    /* metric optional */
  }
}
