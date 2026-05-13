// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import { get, post, put, del } from '@/utils/request'

export interface KnowledgeNamespaceRow {
  id: string
  groupId: number
  createdBy: number
  namespace: string
  name: string
  description?: string
  vectorProvider: string
  embedModel: string
  vectorDim: number
  status: string
  createdAt?: string
  updatedAt?: string
}

export interface KnowledgeNamespaceListPayload {
  list: KnowledgeNamespaceRow[]
  total: number
  page: number
  pageSize: number
  totalPage: number
}

export interface KnowledgeDocumentRow {
  id: string
  groupId: number
  createdBy: number
  namespace: string
  title: string
  source?: string
  fileHash: string
  textUrl?: string
  storedMarkdown?: string
  recordIds?: string
  status: string
  createdAt?: string
  updatedAt?: string
}

export interface KnowledgeDocumentListPayload {
  list: KnowledgeDocumentRow[]
  total: number
  page: number
  pageSize: number
  totalPage: number
}

export interface CreateKnowledgeNamespaceBody {
  namespace: string
  name: string
  description?: string
  vectorProvider?: string
  embedModel: string
  groupId?: number
  status?: string
}

export const listKnowledgeNamespaces = (params?: {
  page?: number
  pageSize?: number
  status?: string
  /** server-side filter on name / namespace */
  q?: string
}) => get<KnowledgeNamespaceListPayload>('/knowledge-namespaces', { params })

export const getKnowledgeNamespace = (id: string | number) =>
  get<{ namespace: KnowledgeNamespaceRow }>(`/knowledge-namespaces/${id}`)

export const getKnowledgeDocument = (id: string | number) =>
  get<{ document: KnowledgeDocumentRow; vectorProvider?: string }>(`/knowledge-documents/${id}`)

export const createKnowledgeNamespace = (body: CreateKnowledgeNamespaceBody) =>
  post<KnowledgeNamespaceRow>('/knowledge-namespaces', body)

export const deleteKnowledgeNamespace = (id: string | number) =>
  del<{ id: string | number }>(`/knowledge-namespaces/${id}`)

export const listKnowledgeDocuments = (params?: {
  page?: number
  pageSize?: number
  namespace?: string
  status?: string
  q?: string
}) => get<KnowledgeDocumentListPayload>('/knowledge-documents', { params })

export const uploadKnowledgeToNamespace = (namespaceId: string | number, file: File) => {
  const fd = new FormData()
  fd.append('file', file)
  return post<{ document: KnowledgeDocumentRow }>(`/knowledge-namespaces/${namespaceId}/upload`, fd)
}

export const reuploadKnowledgeDocument = (docId: string | number, file: File) => {
  const fd = new FormData()
  fd.append('file', file)
  return post<{ document: KnowledgeDocumentRow }>(`/knowledge-documents/${docId}/upload`, fd)
}

export const deleteKnowledgeDocument = (docId: string | number) =>
  del<{ id: string | number }>(`/knowledge-documents/${docId}`)

export const getKnowledgeDocumentText = (docId: string | number) =>
  get<{ url: string; markdown: string }>(`/knowledge-documents/${docId}/text`)

export const putKnowledgeDocumentText = (docId: string | number, markdown: string) =>
  put<{ document: KnowledgeDocumentRow }>(`/knowledge-documents/${docId}/text`, { markdown })

export const runKnowledgeRecallTest = (
  namespaceId: string | number,
  body: { query: string; topK?: number; minScore?: number; docId?: string },
) => post<Record<string, unknown>>(`/knowledge-namespaces/${namespaceId}/recall-test`, body)
