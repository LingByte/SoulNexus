import { get, post, put, del, type ApiResponse } from '@/utils/request'

/** Snowflake IDs are serialized as strings — never use Number() in JS. */
export interface KnowledgeNamespace {
  id: string
  groupId: string
  createdBy: string
  namespace: string
  name: string
  description: string
  vectorProvider: string
  embedModel: string
  vectorDim: number
  status: string
  createdAt: string
  updatedAt: string
}

export interface KnowledgeDocument {
  id: string
  namespaceId: string
  title: string
  source: string
  sourceType: string
  sourceFileName?: string
  textUrl?: string
  chunkCount: number
  chunkStrategy?: string
  charCount: number
  status: 'queued' | 'parsing' | 'preview' | 'indexing' | 'processing' | 'active' | 'failed' | string
  indexError?: string
  indexMode?: string
  docType?: string
  tags?: string[]
  campaignId?: string
  productLine?: string
  summaryText?: string
  createdAt: string
  updatedAt: string
}

export interface KnowledgeDocumentContent {
  id: string
  title: string
  content: string
}

export interface KnowledgeDocumentChunk {
  index: number
  title: string
  preview: string
  charCount: number
}

export interface KnowledgeDocumentChunkDetail {
  docId: string
  index: number
  title: string
  preview: string
  content: string
  charCount: number
  chunkStrategy?: string
}

export interface KnowledgeDocumentChunksResult {
  docId: string
  chunkCount: number
  chunkStrategy: string
  chunks: KnowledgeDocumentChunk[]
}

export interface RecallScores {
  final?: number
  base?: number
  model?: number
  composite?: number
  rrf?: number
  vectorRrf?: number
  keywordRrf?: number
  vectorRank?: number
  keywordRank?: number
}

export interface RecallPipeline {
  strategy?: string
  rerankEnabled?: boolean
  compositeRerank?: boolean
  enableMMR?: boolean
  enableDedup?: boolean
  rrfK?: number
  rrfVectorWeight?: number
  rrfKeywordWeight?: number
  rerankModelWeight?: number
  rerankBaseWeight?: number
  rerankModel?: string
}

export interface RecallHit {
  id: string
  source: string
  title: string
  content: string
  score: number
  scores?: RecallScores
}

export interface RecallResult {
  query: string
  topK: number
  strategy?: string
  pipeline?: RecallPipeline
  results: RecallHit[]
  elapsed: string
  hitCount: number
}

export interface CreateNamespaceReq {
  name: string
  description?: string
}

export interface UpdateNamespaceReq {
  name?: string
  description?: string
}

export interface UpdateDocumentReq {
  title: string
  content: string
}

export interface RecallReq {
  query: string
  topK?: number
  minScore?: number
  docIds?: string[]
  docTypes?: string[]
  tags?: string[]
  campaignId?: string
  productLine?: string
  createdFrom?: string
  createdTo?: string
}

export interface DocumentPreviewChunk {
  index: number
  title: string
  preview: string
  content: string
  charCount: number
  parentIndex?: number
  level?: string
}

export interface DocumentPreviewPayload {
  mode: string
  strategy: string
  charCount: number
  parentCount: number
  childCount: number
  summary?: string
  parse?: { format?: string; charCount?: number; preview?: string }
  children: DocumentPreviewChunk[]
  parents?: DocumentPreviewChunk[]
}

export interface DocumentPreviewResult {
  docId: string
  status: string
  indexMode?: string
  summary?: string
  preview?: DocumentPreviewPayload
  childCount?: number
  parentCount?: number
  chunkStrategy?: string
}

export async function listKnowledgeNamespaces(): Promise<ApiResponse<KnowledgeNamespace[]>> {
  return get('/knowledge-namespaces')
}

export async function getKnowledgeNamespace(id: string | number): Promise<ApiResponse<KnowledgeNamespace>> {
  return get(`/knowledge-namespaces/${id}`)
}

export async function createKnowledgeNamespace(data: CreateNamespaceReq): Promise<ApiResponse<KnowledgeNamespace>> {
  return post('/knowledge-namespaces', data)
}

export async function updateKnowledgeNamespace(id: string | number, data: UpdateNamespaceReq): Promise<ApiResponse<KnowledgeNamespace>> {
  return put(`/knowledge-namespaces/${id}`, data)
}

export async function deleteKnowledgeNamespace(id: string | number): Promise<ApiResponse<null>> {
  return del(`/knowledge-namespaces/${id}`)
}

export async function listKnowledgeDocuments(nsId: string | number): Promise<ApiResponse<KnowledgeDocument[]>> {
  return get(`/knowledge-namespaces/${nsId}/documents`)
}

export async function getKnowledgeDocument(nsId: string | number, docId: string | number): Promise<ApiResponse<KnowledgeDocument>> {
  return get(`/knowledge-namespaces/${nsId}/documents/${docId}`)
}

export async function getKnowledgeDocumentContent(nsId: string | number, docId: string | number): Promise<ApiResponse<KnowledgeDocumentContent>> {
  return get(`/knowledge-namespaces/${nsId}/documents/${docId}/content`)
}

export async function listKnowledgeDocumentChunks(nsId: string | number, docId: string | number): Promise<ApiResponse<KnowledgeDocumentChunksResult>> {
  return get(`/knowledge-namespaces/${nsId}/documents/${docId}/chunks`)
}

export async function getKnowledgeDocumentChunk(
  nsId: string | number,
  docId: string | number,
  chunkIndex: number,
): Promise<ApiResponse<KnowledgeDocumentChunkDetail>> {
  return get(`/knowledge-namespaces/${nsId}/documents/${docId}/chunks/${chunkIndex}`)
}

export async function uploadKnowledgeDocument(
  nsId: string | number,
  payload: {
    title?: string
    content?: string
    file?: File
    docType?: string
    tags?: string[]
    campaignId?: string
    productLine?: string
    indexMode?: string
  },
): Promise<ApiResponse<KnowledgeDocument>> {
  const fd = new FormData()
  if (payload.file) {
    fd.append('file', payload.file)
    if (payload.title) fd.append('title', payload.title)
    if (payload.docType) fd.append('docType', payload.docType)
    if (payload.tags?.length) fd.append('tags', payload.tags.join(','))
    if (payload.campaignId) fd.append('campaignId', payload.campaignId)
    if (payload.productLine) fd.append('productLine', payload.productLine)
    if (payload.indexMode) fd.append('indexMode', payload.indexMode)
  } else {
    return post(`/knowledge-namespaces/${nsId}/documents`, {
      title: payload.title,
      content: payload.content,
      docType: payload.docType,
      tags: payload.tags,
      campaignId: payload.campaignId,
      productLine: payload.productLine,
      indexMode: payload.indexMode,
    })
  }
  return post(`/knowledge-namespaces/${nsId}/documents`, fd)
}

export async function updateKnowledgeDocument(
  nsId: string | number,
  docId: string | number,
  data: UpdateDocumentReq,
): Promise<ApiResponse<KnowledgeDocument>> {
  return put(`/knowledge-namespaces/${nsId}/documents/${docId}`, data)
}

export async function deleteKnowledgeDocument(nsId: string | number, docId: string | number): Promise<ApiResponse<null>> {
  return del(`/knowledge-namespaces/${nsId}/documents/${docId}`)
}

export async function recallKnowledgeDocuments(nsId: string | number, data: RecallReq): Promise<ApiResponse<RecallResult>> {
  return post(`/knowledge-namespaces/${nsId}/recall`, data)
}

export async function getKnowledgeDocumentPreview(
  nsId: string | number,
  docId: string | number,
): Promise<ApiResponse<DocumentPreviewResult>> {
  return get(`/knowledge-namespaces/${nsId}/documents/${docId}/preview`)
}

export async function confirmKnowledgeDocumentIndex(
  nsId: string | number,
  docId: string | number,
): Promise<ApiResponse<KnowledgeDocument>> {
  return post(`/knowledge-namespaces/${nsId}/documents/${docId}/confirm-index`)
}
