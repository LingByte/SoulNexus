import { get, post } from '@/utils/request'

export type WidgetMarketItem = {
  id: string
  slug: string
  name: string
  displayName: string
  description?: string
  category: string
  avatarUrl?: string
  tags?: string
  version: string
  author?: string
  jsSourceId: string
  embedPath: string
  installCount: number
  sourceJsTemplateId?: string
}

export type WidgetMarketCategory =
  | 'desktop_pet'
  | 'chat_widget'
  | 'live2d'
  | 'utility'
  | 'custom'
  | 'all'

type PageData = {
  list?: WidgetMarketItem[]
  total?: number
}

export const widgetMarketService = {
  async listPublished(params?: {
    page?: number
    pageSize?: number
    category?: string
    keyword?: string
    useTenantApi?: boolean
  }) {
    const path = params?.useTenantApi ? '/widget-market/items' : '/public/widget-market/items'
    return get<PageData>(path, {
      params: {
        page: params?.page ?? 1,
        size: params?.pageSize ?? 24,
        category: params?.category && params.category !== 'all' ? params.category : undefined,
        keyword: params?.keyword?.trim() || undefined,
      },
    })
  },

  getItem(id: string, useTenantApi?: boolean) {
    const path = useTenantApi
      ? `/widget-market/items/${encodeURIComponent(id)}`
      : `/public/widget-market/items/${encodeURIComponent(id)}`
    return get<WidgetMarketItem>(path)
  },

  trackDownload(id: string) {
    return post<{ ok: boolean }>(`/public/widget-market/items/${encodeURIComponent(id)}/download`)
  },

  publishFromTemplate(body: {
    jsTemplateId: string
    displayName?: string
    description?: string
    category?: string
    tags?: string
    author?: string
    publish?: boolean
  }) {
    return post<WidgetMarketItem>('/widget-market/publish', {
      ...body,
      publish: body.publish !== false,
    })
  },

  delist(jsTemplateId: string) {
    return post<{ ok: boolean }>('/widget-market/delist', { jsTemplateId })
  },
}

export default widgetMarketService
