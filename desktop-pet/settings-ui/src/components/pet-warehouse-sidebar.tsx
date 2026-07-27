import {
  Eye,
  Plus,
  Sparkles,
  Star,
  Trash2,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import type { PetEntry } from '@/vite-env'
import { cn } from '@/lib/utils'

export const WAREHOUSE_SIDEBAR_WIDTH = 280
/** 桌宠列表区域最大高度，超出后纵向滚动 */
export const WAREHOUSE_LIST_MAX_HEIGHT = 'min(560px, calc(100vh - 200px))'

type Props = {
  pets: PetEntry[]
  selectedId: string
  primaryPetId?: string
  onSelect: (id: string) => void
  onChangePet: (id: string, patch: Partial<PetEntry>) => void
  onAdd: () => void
  onRemove: (id: string) => void
  onPreview: (id: string) => void
  previewLoadingId: string | null
}

export function PetWarehouseSidebar({
  pets,
  selectedId,
  primaryPetId,
  onSelect,
  onChangePet,
  onAdd,
  onRemove,
  onPreview,
  previewLoadingId,
}: Props) {
  return (
    <aside
      className="flex h-full min-h-0 shrink-0 flex-col overflow-hidden border-r border-border bg-card"
      style={{ width: WAREHOUSE_SIDEBAR_WIDTH }}
    >
      <div className="border-b border-border px-4 py-4">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <h2 className="text-sm font-semibold text-foreground">桌宠仓库</h2>
            <p className="mt-0.5 text-xs text-muted-foreground leading-relaxed">
              右上角打开挂件市场，或手动填写 jsSourceId
            </p>
          </div>
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="h-8 w-8 shrink-0 bg-background"
            title="新建空白桌宠"
            onClick={onAdd}
          >
            <Plus className="h-4 w-4" />
          </Button>
        </div>
      </div>

      <ul
        className="min-h-0 overflow-y-auto overflow-x-hidden p-3 space-y-2 overscroll-contain"
        style={{ maxHeight: WAREHOUSE_LIST_MAX_HEIGHT }}
      >
        {pets.map((pet) => {
          const selected = pet.id === selectedId
          const isPrimary = primaryPetId === pet.id
          return (
            <li key={pet.id}>
              <div
                className={cn(
                  'rounded-xl border bg-background transition-all',
                  selected
                    ? 'border-primary/35 ring-1 ring-primary/20 shadow-sm'
                    : 'border-border hover:border-primary/20',
                )}
              >
                <button
                  type="button"
                  className="flex w-full items-center gap-2 px-3 py-2.5 text-left"
                  onClick={() => onSelect(pet.id)}
                >
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-1.5">
                      <p className="text-xs font-medium truncate text-foreground">{pet.name || '未命名'}</p>
                      {isPrimary ? (
                        <Star className="h-3 w-3 shrink-0 fill-primary text-primary" aria-label="主桌宠" />
                      ) : null}
                    </div>
                    <p className="text-[10px] font-mono text-muted-foreground truncate mt-0.5">
                      {pet.jsSourceId || '未设置 jsSourceId'}
                    </p>
                  </div>
                  <Switch
                    className="scale-90 shrink-0"
                    checked={pet.enabled !== false}
                    onCheckedChange={(v) => onChangePet(pet.id, { enabled: v })}
                    onClick={(e) => e.stopPropagation()}
                  />
                </button>
                <div className="flex gap-1.5 px-3 pb-3 pt-0">
                  <Input
                    value={pet.jsSourceId}
                    onChange={(e) => onChangePet(pet.id, { jsSourceId: e.target.value })}
                    placeholder="js_…"
                    spellCheck={false}
                    className="h-8 text-[11px] font-mono flex-1 bg-card"
                    onClick={(e) => e.stopPropagation()}
                  />
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="h-8 w-8 p-0 shrink-0"
                    disabled={previewLoadingId === pet.id}
                    title="预览"
                    onClick={(e) => {
                      e.stopPropagation()
                      onPreview(pet.id)
                    }}
                  >
                    {previewLoadingId === pet.id ? (
                      <Sparkles className="h-3.5 w-3.5 animate-pulse text-primary" />
                    ) : (
                      <Eye className="h-3.5 w-3.5" />
                    )}
                  </Button>
                  {pets.length > 1 ? (
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="h-8 w-8 p-0 shrink-0 text-muted-foreground hover:text-destructive"
                      title="移除"
                      onClick={(e) => {
                        e.stopPropagation()
                        onRemove(pet.id)
                      }}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  ) : null}
                </div>
              </div>
            </li>
          )
        })}
      </ul>
    </aside>
  )
}
