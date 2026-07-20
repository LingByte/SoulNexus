import { Table, Typography } from '@arco-design/web-react'
import { useTranslation } from '@/i18n'
import { formatSchemaJSON, schemaParamRows, type JsonSchemaObject } from './toolSchema'

type ToolSchemaDetailProps = {
  schema?: JsonSchemaObject | null
  compact?: boolean
  showJson?: boolean
}

export default function ToolSchemaDetail({ schema, compact = false, showJson = !compact }: ToolSchemaDetailProps) {
  const { t } = useTranslation()
  const rows = schemaParamRows(schema)

  if (!rows.length) {
    return (
      <Typography.Text type="secondary" className="!text-xs">
        {t('assistantTools.noSchema')}
      </Typography.Text>
    )
  }

  if (compact) {
    return (
      <div className="mt-1 space-y-0.5">
        {rows.map((r) => (
          <div key={r.name} className="text-xs text-muted-foreground font-mono">
            <span className="text-foreground">{r.name}</span>
            {r.required ? <span className="text-red-500">*</span> : null}
            <span className="text-muted-foreground"> ({r.type})</span>
            {r.description ? <span> — {r.description}</span> : null}
          </div>
        ))}
      </div>
    )
  }

  const json = showJson ? formatSchemaJSON(schema) : ''

  return (
    <div className="space-y-3">
      <Table
        size="small"
        pagination={false}
        rowKey="name"
        columns={[
          { title: t('assistantTools.schemaParam'), dataIndex: 'name', width: 140 },
          {
            title: t('assistantTools.schemaType'),
            dataIndex: 'type',
            width: 90,
            render: (v: string, r) => (
              <span>
                {v}
                {r.required ? <span className="text-red-500 ml-0.5">*</span> : null}
              </span>
            ),
          },
          { title: t('assistantTools.schemaDesc'), dataIndex: 'description', ellipsis: true },
        ]}
        data={rows}
      />
      {json ? (
        <pre className="rounded-md bg-muted/40 p-3 text-xs font-mono overflow-x-auto max-h-48">{json}</pre>
      ) : null}
    </div>
  )
}
