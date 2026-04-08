import { del, get, post, put, type ApiResponse } from '@/utils/request'
import { getApiBaseURL } from '@/config/apiConfig'

export interface KnowledgeBase {
  id: number
  userId: number
  groupId?: number | null
  name: string
  description?: string
  provider: string
  endpointUrl?: string
  apiKey?: string
  apiSecret?: string
  indexName?: string
  namespace?: string
  extraConfig?: Record<string, any>
  isActive: boolean
  createdAt: string
  updatedAt: string
}

export interface KnowledgeBasePayload {
  name: string
  description?: string
  provider: string
  endpointUrl?: string
  apiKey?: string
  apiSecret?: string
  indexName?: string
  namespace?: string
  embeddingUrl?: string
  embeddingKey?: string
  embeddingModel?: string
  extraConfig?: Record<string, any>
}

export interface SupportedDocumentFormat {
  extension: string
  description: string
}

export interface SupportedDocumentTypesResponse {
  formats: SupportedDocumentFormat[]
  notes: string[]
}

export interface KnowledgeDocumentItem {
  docId: string
  filename: string
  chunks: number
  source: string
}

export interface KnowledgeDocumentListResponse {
  items: KnowledgeDocumentItem[]
  totalChunks: number
  totalDocs: number
}

export type KnowledgeUploadStreamEvent =
  | { type: 'progress'; phase: string; percent?: number; message?: string; chunks?: number }
  | { type: 'done'; percent?: number; message?: string; data: { knowledgeBaseId: number; docId: string; chunks: number; filename: string } }
  | { type: 'error'; message: string }

export const listKnowledgeBases = (): Promise<ApiResponse<KnowledgeBase[]>> =>
  get('/knowledge-base')

export const getKnowledgeBase = (id: number): Promise<ApiResponse<KnowledgeBase>> =>
  get(`/knowledge-base/${id}`)

/** 与 getKnowledgeBase 相同，便于页面侧语义区分 */
export const fetchKnowledgeBaseById = getKnowledgeBase

export const listSupportedDocumentTypes = (): Promise<ApiResponse<SupportedDocumentTypesResponse>> =>
  get('/knowledge-base/supported-document-types')

export const createKnowledgeBase = (data: KnowledgeBasePayload): Promise<ApiResponse<KnowledgeBase>> =>
  post('/knowledge-base', data)

export const updateKnowledgeBase = (id: number, data: Partial<KnowledgeBasePayload & { isActive: boolean }>): Promise<ApiResponse<KnowledgeBase>> =>
  put(`/knowledge-base/${id}`, data)

export const deleteKnowledgeBase = (id: number): Promise<ApiResponse<null>> =>
  del(`/knowledge-base/${id}`)

export const listKnowledgeDocuments = (id: number): Promise<ApiResponse<KnowledgeDocumentListResponse>> =>
  get(`/knowledge-base/${id}/documents`)

export const uploadKnowledgeDocument = (id: number, file: File): Promise<ApiResponse<{ knowledgeBaseId: number; docId: string; chunks: number; filename: string }>> => {
  const formData = new FormData()
  formData.append('file', file)
  return post(`/knowledge-base/${id}/documents/upload`, formData)
}

export interface UploadKnowledgeDocumentProgressHandlers {
  onUploadProgress?: (percent: number, loaded: number, total: number) => void
  onServerEvent?: (evt: KnowledgeUploadStreamEvent) => void
  onOverallProgress?: (percent: number, message: string) => void
}

/** 上传并解析 NDJSON 进度流（Accept: application/x-ndjson）；失败时抛出 { msg } */
export function uploadKnowledgeDocumentWithProgress(
  id: number,
  file: File,
  handlers: UploadKnowledgeDocumentProgressHandlers,
): Promise<{ knowledgeBaseId: number; docId: string; chunks: number; filename: string }> {
  const token = localStorage.getItem('auth_token')
  const base = getApiBaseURL().replace(/\/$/, '')
  const url = `${base}/knowledge-base/${id}/documents/upload?_t=${Date.now()}`

  const mapServerToOverall = (serverPercent: number) => {
    const s = Math.min(100, Math.max(0, serverPercent))
    if (s <= 15) return 22
    return Math.round(22 + ((s - 15) / (100 - 15)) * (100 - 22))
  }

  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    const formData = new FormData()
    formData.append('file', file)

    let lineCursor = 0
    let lastOverall = 0
    let settled = false

    const finish = (fn: () => void) => {
      if (settled) return
      settled = true
      fn()
    }

    const bumpOverall = (pct: number, msg: string) => {
      lastOverall = Math.max(lastOverall, Math.min(100, pct))
      handlers.onOverallProgress?.(lastOverall, msg)
    }

    const parseIncremental = () => {
      const text = xhr.responseText || ''
      const lines = text.split('\n')
      const completeThrough = xhr.readyState === 4 ? lines.length : lines.length - 1
      for (let i = lineCursor; i < completeThrough; i++) {
        const raw = lines[i]?.trim()
        if (!raw) continue
        let evt: KnowledgeUploadStreamEvent
        try {
          evt = JSON.parse(raw) as KnowledgeUploadStreamEvent
        } catch {
          continue
        }
        handlers.onServerEvent?.(evt)
        if (evt.type === 'progress' && typeof evt.percent === 'number') {
          bumpOverall(mapServerToOverall(evt.percent), evt.message || evt.phase)
        }
        if (evt.type === 'done' && evt.data) {
          bumpOverall(100, evt.message || '完成')
          finish(() => resolve(evt.data))
          return
        }
        if (evt.type === 'error') {
          finish(() => reject({ msg: evt.message || '上传失败' }))
          return
        }
      }
      lineCursor = completeThrough
    }

    xhr.open('POST', url)
    if (token) {
      xhr.setRequestHeader('Authorization', `Bearer ${token}`)
    }
    xhr.setRequestHeader('Accept', 'application/x-ndjson')

    xhr.upload.onprogress = e => {
      if (!e.lengthComputable || !e.total) return
      const up = Math.round((e.loaded / e.total) * 20)
      bumpOverall(up, '正在上传文件…')
      handlers.onUploadProgress?.(Math.round((e.loaded / e.total) * 100), e.loaded, e.total)
    }

    xhr.onreadystatechange = () => {
      if (xhr.readyState === 3 || xhr.readyState === 4) {
        parseIncremental()
      }
    }

    xhr.onload = () => {
      if (settled) return
      if (xhr.status >= 200 && xhr.status < 300) {
        parseIncremental()
        if (settled) return
        if (lastOverall < 100) {
          try {
            const j = JSON.parse(xhr.responseText)
            if (j?.code === 200 && j?.data) {
              bumpOverall(100, '完成')
              finish(() => resolve(j.data))
              return
            }
          } catch {
            /* fall through */
          }
        }
        finish(() => reject({ msg: '未收到完成事件，请稍后重试' }))
        return
      }
      let msg = `上传失败 (${xhr.status})`
      try {
        const j = JSON.parse(xhr.responseText)
        msg = j.msg || j.message || msg
      } catch {
        /* ignore */
      }
      finish(() => reject({ msg }))
    }

    xhr.onerror = () => finish(() => reject({ msg: '网络错误' }))
    xhr.ontimeout = () => finish(() => reject({ msg: '请求超时' }))
    xhr.timeout = 120000

    xhr.send(formData)
  })
}

export const deleteKnowledgeDocument = (id: number, payload: { docId?: string; ids?: string[] }): Promise<ApiResponse<{ deleted: number; ids?: string[] }>> =>
  del(`/knowledge-base/${id}/documents`, { data: payload })

export const recallTestKnowledgeBase = (id: number, query: string, topK = 5): Promise<ApiResponse<{ total: number; items: any[] }>> =>
  post(`/knowledge-base/${id}/recall-test`, { query, topK })
