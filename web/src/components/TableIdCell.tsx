import { EllipsisHoverCell } from '@/pages/assistants/EllipsisHoverCell'

/** Snowflake / numeric id in admin tables: single-line ellipsis, full value on hover. */
export function TableIdCell({ id }: { id: string | number | null | undefined }) {
  if (id == null || id === '') return <span className="text-muted-foreground">—</span>
  return (
    <div className="max-w-[88px]">
      <EllipsisHoverCell text={String(id)} lines={1} mono />
    </div>
  )
}
