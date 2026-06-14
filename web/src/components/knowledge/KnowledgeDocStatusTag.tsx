// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import React from 'react'
import { Tag } from '@arco-design/web-react'
import { Loader2 } from 'lucide-react'
import { isKnowledgeDocInProgress } from '@/api/knowledge'
import { useI18nStore } from '@/stores/i18nStore'

interface Props {
  status?: string
  className?: string
}

export const KnowledgeDocStatusTag: React.FC<Props> = ({ status, className }) => {
  const { t } = useI18nStore()
  const s = (status || 'active').toLowerCase()
  const inProgress = isKnowledgeDocInProgress(s)

  const labelKey = `knowledge.status.${s}`
  const label = t(labelKey) !== labelKey ? t(labelKey) : s

  let color: string = 'gray'
  switch (s) {
    case 'active':
      color = 'green'
      break
    case 'failed':
      color = 'red'
      break
    case 'deleted':
      color = 'gray'
      break
    default:
      if (inProgress) color = 'arcoblue'
  }

  const spinner = inProgress ? (
    <Loader2 className="h-3 w-3 shrink-0 animate-spin" aria-hidden />
  ) : undefined

  return (
    <Tag
      size="small"
      color={color}
      icon={spinner}
      className={`!rounded-md !align-middle ${className || ''}`}
    >
      {label}
    </Tag>
  )
}
