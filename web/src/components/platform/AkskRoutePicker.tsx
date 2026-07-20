import { useMemo, useState } from 'react'
import { Checkbox, Collapse, Space, Tag, Typography } from '@arco-design/web-react'
import { Button, Input } from '@/components/ui'
import { useTranslation } from '@/i18n'
import type { AkskRouteCatalogGroup } from '@/api/akskRoutePolicy'

const CollapseItem = Collapse.Item

type Props = {
  groups: AkskRouteCatalogGroup[]
  selectedIds: string[]
  onChange: (ids: string[]) => void
  disabled?: boolean
  emptyHint?: string
  /** When false, all API groups start collapsed. Defaults to true. */
  defaultExpanded?: boolean
}

export default function AkskRoutePicker({
  groups,
  selectedIds,
  onChange,
  disabled,
  emptyHint,
  defaultExpanded = true,
}: Props) {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const selected = useMemo(() => new Set(selectedIds), [selectedIds])

  const filteredGroups = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) return groups
    return groups
      .map((g) => ({
        ...g,
        entries: g.entries.filter(
          (e) =>
            e.label.toLowerCase().includes(q) ||
            e.path.toLowerCase().includes(q) ||
            e.method.toLowerCase().includes(q) ||
            e.id.toLowerCase().includes(q),
        ),
      }))
      .filter((g) => g.entries.length > 0)
  }, [groups, search])

  const toggleOne = (id: string, checked: boolean) => {
    const next = new Set(selected)
    if (checked) next.add(id)
    else next.delete(id)
    onChange(Array.from(next))
  }

  const toggleGroup = (group: AkskRouteCatalogGroup, checked: boolean) => {
    const next = new Set(selected)
    for (const e of group.entries) {
      if (checked) next.add(e.id)
      else next.delete(e.id)
    }
    onChange(Array.from(next))
  }

  const selectAll = () => {
    const all = groups.flatMap((g) => g.entries.map((e) => e.id))
    onChange(all)
  }

  const clearAll = () => onChange([])

  if (groups.length === 0) {
    return (
      <Typography.Text type="secondary">
        {emptyHint || t('akskRoutePolicy.catalogEmpty')}
      </Typography.Text>
    )
  }

  return (
    <div>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <Input
          value={search}
          onChange={setSearch}
          placeholder={t('akskRoutePolicy.searchPlaceholder')}
          disabled={disabled}
          style={{ flex: '1 1 220px', maxWidth: 360 }}
        />
        <Button size="small" type="outline" disabled={disabled} onClick={selectAll}>
          {t('akskRoutePolicy.selectAll')}
        </Button>
        <Button size="small" type="outline" disabled={disabled} onClick={clearAll}>
          {t('akskRoutePolicy.clearAll')}
        </Button>
        <Tag>{selectedIds.length} / {groups.reduce((n, g) => n + g.entries.length, 0)}</Tag>
      </div>

      <Collapse
        bordered={false}
        defaultActiveKey={defaultExpanded ? filteredGroups.slice(0, 3).map((g) => g.id) : []}
      >
        {filteredGroups.map((group) => {
          const ids = group.entries.map((e) => e.id)
          const checkedCount = ids.filter((id) => selected.has(id)).length
          const allChecked = checkedCount === ids.length && ids.length > 0
          const indeterminate = checkedCount > 0 && !allChecked
          return (
            <CollapseItem
              key={group.id}
              name={group.id}
              header={
                <Space onClick={(e) => e.stopPropagation()}>
                  <Checkbox
                    checked={allChecked}
                    indeterminate={indeterminate}
                    disabled={disabled}
                    onChange={(v) => toggleGroup(group, v)}
                  />
                  <span>{group.label}</span>
                  <Tag size="small">{checkedCount}/{ids.length}</Tag>
                </Space>
              }
            >
              <div className="flex flex-col gap-2 pl-1">
                {group.entries.map((entry) => (
                  <label
                    key={entry.id}
                    className="flex cursor-pointer items-start gap-2 rounded-md px-2 py-1.5 hover:bg-muted/40"
                  >
                    <Checkbox
                      checked={selected.has(entry.id)}
                      disabled={disabled}
                      onChange={(v) => toggleOne(entry.id, v)}
                      style={{ marginTop: 2 }}
                    />
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <Typography.Text style={{ fontWeight: 500 }}>{entry.label}</Typography.Text>
                        <Tag size="small" color="arcoblue">{entry.method}</Tag>
                        {entry.permission ? (
                          <Tag size="small">{entry.permission}</Tag>
                        ) : null}
                      </div>
                      <Typography.Text type="secondary" style={{ fontFamily: 'monospace', fontSize: 12 }}>
                        {entry.path}
                      </Typography.Text>
                    </div>
                  </label>
                ))}
              </div>
            </CollapseItem>
          )
        })}
      </Collapse>
    </div>
  )
}
