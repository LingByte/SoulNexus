import type { DiscoveredMCPTool } from '@/api/assistantTools'

export type JsonSchemaObject = Record<string, unknown>

export type SchemaParamRow = {
  name: string
  type: string
  required: boolean
  description: string
}

export function schemaParamRows(schema?: JsonSchemaObject | null): SchemaParamRow[] {
  if (!schema || typeof schema !== 'object') return []
  const props = (schema.properties as Record<string, JsonSchemaObject>) || {}
  const required = new Set(
    Array.isArray(schema.required) ? schema.required.map((x) => String(x)) : [],
  )
  return Object.entries(props).map(([name, def]) => ({
    name,
    type: String(def?.type || 'any'),
    required: required.has(name),
    description: typeof def?.description === 'string' ? def.description : '',
  }))
}

export function schemaSummary(schema?: JsonSchemaObject | null): string {
  const rows = schemaParamRows(schema)
  if (!rows.length) return ''
  return rows
    .map((r) => `${r.name}${r.required ? '*' : ''} (${r.type})${r.description ? `: ${r.description}` : ''}`)
    .join(' · ')
}

export function formatSchemaJSON(schema?: JsonSchemaObject | null): string {
  if (!schema || typeof schema !== 'object') return ''
  try {
    return JSON.stringify(schema, null, 2)
  } catch {
    return ''
  }
}

export function httpToolSchema(tool: { parameters?: JsonSchemaObject | null }): JsonSchemaObject | null {
  return tool.parameters && typeof tool.parameters === 'object' ? tool.parameters : null
}

export function mcpToolSchema(tool: DiscoveredMCPTool): JsonSchemaObject | null {
  return tool.inputSchema && typeof tool.inputSchema === 'object' ? tool.inputSchema : null
}
