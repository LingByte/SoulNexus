import {
  Eye,
  GripVertical,
  Plus,
  Sparkles,
  Trash2,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import type { PetEntry } from '@/vite-env'
import { cn } from '@/lib/utils'

type Props = {
  pets: PetEntry[]
  selectedId: string
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
  onSelect,
  onChangePet,
  onAdd,
  onRemove,
  onPreview,
  previewLoadingId,
}: Props) {
  return (
    <aside className="flex h-full w-[248px] shrink-0 flex-col border-r border-black/[0.06] bg-white/80">
      <div className="flex items-center justify-between gap-2 border-b border-black/[0.05] px-3 py-3">
        <div className="min-w-0">
          <p className="text-xs font-semibold text-[#18181B]">桌宠仓库</p>
          <p className="text-[10px] text-muted-foreground mt-0.5">多个 jsSourceId · 可同时启用</p>
        </div>
        <Button type="button" size="sm" variant="outline" className="h-8 w-8 p-0 shrink-0" onClick={onAdd}>
          <Plus className="h-4 w-4" />
        </Button>
      </div>

      <ul className="flex-1 overflow-y-auto p-2 space-y-1.5">
        {pets.map((pet) => {
          const selected = pet.id === selectedId
          return (
            <li key={pet.id}>
              <div
                className={cn(
                  'rounded-xl border transition-colors',
                  selected
                    ? 'border-[#18181B]/20 bg-[#18181B]/[0.04] shadow-sm'
                    : 'border-black/[0.06] bg-white hover:border-black/[0.1]',
                )}
              >
                <button
                  type="button"
                  className="flex w-full items-start gap-2 px-2.5 py-2 text-left"
                  onClick={() => onSelect(pet.id)}
                >
                  <GripVertical className="mt-1 h-3.5 w-3.5 shrink-0 text-muted-foreground/60" />
                  <div className="min-w-0 flex-1">
                    <p className="text-xs font-medium truncate text-[#18181B]">{pet.name || '未命名'}</p>
                    <p className="text-[10px] font-mono text-muted-foreground truncate mt-0.5">
                      {pet.jsSourceId || '—'}
                    </p>
                  </div>
                  <Switch
                    className="scale-90 shrink-0"
                    checked={pet.enabled !== false}
                    onCheckedChange={(v) => onChangePet(pet.id, { enabled: v })}
                    onClick={(e) => e.stopPropagation()}
                  />
                </button>
                <div className="flex gap-1 px-2 pb-2 pt-0">
                  <Input
                    value={pet.jsSourceId}
                    onChange={(e) => onChangePet(pet.id, { jsSourceId: e.target.value })}
                    placeholder="js_…"
                    spellCheck={false}
                    className="h-7 text-[10px] font-mono flex-1"
                    onClick={(e) => e.stopPropagation()}
                  />
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="h-7 px-2 shrink-0"
                    disabled={previewLoadingId === pet.id}
                    onClick={(e) => {
                      e.stopPropagation()
                      onPreview(pet.id)
                    }}
                  >
                    {previewLoadingId === pet.id ? (
                      <Sparkles className="h-3.5 w-3.5 animate-pulse" />
                    ) : (
                      <Eye className="h-3.5 w-3.5" />
                    )}
                  </Button>
                  {pets.length > 1 ? (
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="h-7 w-7 p-0 shrink-0 text-muted-foreground hover:text-red-600"
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

      <div className="border-t border-black/[0.05] px-3 py-2.5 text-[10px] text-muted-foreground leading-relaxed">
        勾选启用后保存，会在桌面同时挂载对应 embed。预览为独立小窗。
      </div>
    </aside>
  )
}
