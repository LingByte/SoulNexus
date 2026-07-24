import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Bot,
  CheckCircle2,
  KeyRound,
  Loader2,
  Palette,
  Plug,
  Power,
  Save,
  Star,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Field } from '@/components/ui/field'
import { SegmentedControl } from '@/components/ui/segmented-control'
import { SectionCard } from '@/components/ui/section-card'
import { PetWarehouseSidebar } from '@/components/pet-warehouse-sidebar'
import type { DesktopPetConfig, PetEntry } from '@/vite-env'
import {
  EMPTY_CONFIG,
  createPetEntry,
  embedUrl,
  normalizeConfig,
  payloadForSave,
} from '@/lib/pet-config'
import { cn } from '@/lib/utils'

type SectionKey = 'connect' | 'agent' | 'resident' | 'appearance'

type StatusKind = 'idle' | 'ok' | 'err' | 'loading'

export default function SettingsPage() {
  const [form, setForm] = useState<DesktopPetConfig>(EMPTY_CONFIG)
  const [selectedId, setSelectedId] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [previewLoadingId, setPreviewLoadingId] = useState<string | null>(null)
  const [status, setStatus] = useState<{ kind: StatusKind; msg: string }>({
    kind: 'idle',
    msg: '',
  })
  const [openSections, setOpenSections] = useState<Record<SectionKey, boolean>>({
    connect: false,
    agent: false,
    resident: false,
    appearance: false,
  })

  const api = window.electronPet

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      if (!api) {
        setStatus({ kind: 'err', msg: 'Electron preload 未就绪（浏览器预览模式下部分功能不可用）' })
        setLoading(false)
        return
      }
      try {
        const cfg = await api.getConfig()
        if (!cancelled) {
          const normalized = normalizeConfig(cfg as Record<string, unknown>)
          setForm(normalized)
          setSelectedId(normalized.primaryPetId || normalized.pets[0]?.id || '')
        }
      } catch (e) {
        if (!cancelled) {
          setStatus({ kind: 'err', msg: `读取配置失败: ${(e as Error).message || e}` })
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [api])

  const selectedPet = useMemo(
    () => form.pets.find((p) => p.id === selectedId) || form.pets[0],
    [form.pets, selectedId],
  )

  const configured = useMemo(() => {
    const id = form.assistantId.trim()
    return Boolean(id && id !== 'YOUR_ASSISTANT_ID' && form.serverBase.trim())
  }, [form.assistantId, form.serverBase])

  const patchGlobal = useCallback(<K extends keyof DesktopPetConfig>(key: K, value: DesktopPetConfig[K]) => {
    setForm((prev) => ({ ...prev, [key]: value }))
  }, [])

  const patchPet = useCallback((id: string, patch: Partial<PetEntry>) => {
    setForm((prev) => ({
      ...prev,
      pets: prev.pets.map((p) => (p.id === id ? { ...p, ...patch } : p)),
    }))
  }, [])

  const toggleSection = (key: SectionKey, open: boolean) => {
    setOpenSections((prev) => ({ ...prev, [key]: open }))
  }

  const onAddPet = () => {
    const pet = createPetEntry({ name: `桌宠 ${form.pets.length + 1}` })
    setForm((prev) => ({ ...prev, pets: [...prev.pets, pet] }))
    setSelectedId(pet.id)
  }

  const onRemovePet = (id: string) => {
    setForm((prev) => {
      const next = prev.pets.filter((p) => p.id !== id)
      if (next.length === 0) next.push(createPetEntry({ name: '懒懒' }))
      return { ...prev, pets: next }
    })
    setSelectedId((cur) => {
      if (cur !== id) return cur
      const rest = form.pets.filter((p) => p.id !== id)
      return rest[0]?.id || ''
    })
  }

  const onTest = async () => {
    if (!form.serverBase.trim()) {
      setStatus({ kind: 'err', msg: '请先填写 API 地址' })
      return
    }
    const pet = selectedPet
    if (!pet?.jsSourceId.trim()) {
      setStatus({ kind: 'err', msg: '请先在左侧仓库填写 jsSourceId' })
      return
    }
    const url = embedUrl(form.serverBase, pet.jsSourceId)
    setTesting(true)
    setStatus({ kind: 'loading', msg: `正在测试 ${url} …` })
    try {
      const res = await fetch(url, { method: 'GET', cache: 'no-store' })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const text = await res.text()
      if (
        !text.includes('lingecho-embed-root') &&
        !text.includes('lanlan-pet-root') &&
        !text.includes('__LingEchoConfig') &&
        !text.includes('__LanlanConfig')
      ) {
        throw new Error('响应不像有效的 embed.js')
      }
      setStatus({ kind: 'ok', msg: `「${pet.name}」连接成功，可以保存并启动` })
    } catch (e) {
      setStatus({
        kind: 'err',
        msg: `连接失败: ${(e as Error).message || e}\n请确认后端已启动且挂件已发布为 active`,
      })
    } finally {
      setTesting(false)
    }
  }

  const onPreview = async (petId: string) => {
    if (!api?.openEmbedPreview) {
      setStatus({ kind: 'err', msg: '预览需要 Electron 环境' })
      return
    }
    setPreviewLoadingId(petId)
    try {
      const res = await api.openEmbedPreview(petId)
      if (!res.ok) throw new Error(res.error || '预览失败')
    } catch (e) {
      setStatus({ kind: 'err', msg: `预览失败: ${(e as Error).message || e}` })
    } finally {
      setPreviewLoadingId(null)
    }
  }

  const onSave = async () => {
    if (!form.serverBase.trim()) {
      setStatus({ kind: 'err', msg: '请填写 API 地址' })
      return
    }
    if (!form.assistantId.trim() || form.assistantId.trim() === 'YOUR_ASSISTANT_ID') {
      setStatus({ kind: 'err', msg: '请填写智能体 ID（assistantId）' })
      return
    }
    if (!api) {
      setStatus({ kind: 'err', msg: 'Electron preload 未就绪' })
      return
    }
    const payload = payloadForSave({
      ...form,
      primaryPetId: selectedId || form.primaryPetId,
    })
    setSaving(true)
    try {
      const saved = await api.saveConfig(payload)
      const normalized = normalizeConfig(saved as Record<string, unknown>)
      setForm(normalized)
      setSelectedId(normalized.primaryPetId || normalized.pets[0]?.id || '')
      setStatus({ kind: 'ok', msg: '已保存并同步桌面桌宠（已启用的会同时显示）' })
    } catch (e) {
      setStatus({ kind: 'err', msg: `保存失败: ${(e as Error).message || e}` })
    } finally {
      setSaving(false)
    }
  }

  const onToggleLogin = async (enabled: boolean) => {
    patchGlobal('openAtLogin', enabled)
    if (!api) return
    try {
      const saved = await api.setOpenAtLogin(enabled)
      setForm((prev) => ({ ...prev, ...normalizeConfig(saved as Record<string, unknown>), openAtLogin: enabled }))
    } catch (e) {
      setStatus({ kind: 'err', msg: `设置开机自启失败: ${(e as Error).message || e}` })
    }
  }

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center gap-2 text-sm text-muted-foreground bg-[rgb(232,232,235)]">
        <Loader2 className="h-4 w-4 animate-spin" />
        加载配置…
      </div>
    )
  }

  const pet = selectedPet

  return (
    <div className="flex min-h-screen flex-col bg-[rgb(232,232,235)]">
      <header className="sticky top-0 z-10 border-b border-black/[0.05] bg-[rgb(232,232,235)]/92 backdrop-blur-md px-5 py-4 shrink-0">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="flex items-center gap-2.5">
              <img
                src="./favicon.png"
                alt="SoulMy"
                className="h-8 w-8 rounded-xl bg-white shadow-sm border border-black/[0.06] object-cover"
              />
              <div className="min-w-0">
                <h1 className="text-[15px] font-semibold tracking-tight text-[#18181B]">SoulMy</h1>
                <p className="mt-0.5 text-[11px] text-muted-foreground">
                  桌面桌宠 · 托盘常驻 · 多挂件仓库
                </p>
              </div>
            </div>
          </div>
          <span
            className={cn(
              'shrink-0 inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-[11px] font-medium border',
              configured
                ? 'bg-[#18181B] text-white border-[#18181B]'
                : 'bg-white text-amber-700 border-amber-200/80',
            )}
          >
            {configured ? (
              <>
                <CheckCircle2 className="h-3 w-3" />
                已配置
              </>
            ) : (
              '未配置'
            )}
          </span>
        </div>
      </header>

      <div className="flex flex-1 min-h-0">
        <PetWarehouseSidebar
          pets={form.pets}
          selectedId={selectedId || pet?.id || ''}
          onSelect={setSelectedId}
          onChangePet={patchPet}
          onAdd={onAddPet}
          onRemove={onRemovePet}
          onPreview={onPreview}
          previewLoadingId={previewLoadingId}
        />

        <main className="flex-1 overflow-y-auto space-y-3 p-4 pb-28 min-w-0">
          {pet ? (
            <div className="rounded-xl border border-dashed border-black/[0.08] bg-white/60 px-3 py-2 flex items-center justify-between gap-2">
              <div className="min-w-0">
                <p className="text-xs font-medium truncate">正在编辑：{pet.name}</p>
                <p className="text-[10px] text-muted-foreground font-mono truncate">{pet.jsSourceId}</p>
              </div>
              <Button
                type="button"
                size="sm"
                variant={form.primaryPetId === pet.id ? 'default' : 'outline'}
                className="shrink-0 h-8 text-xs gap-1"
                onClick={() => patchGlobal('primaryPetId', pet.id)}
              >
                <Star className={cn('h-3.5 w-3.5', form.primaryPetId === pet.id && 'fill-current')} />
                {form.primaryPetId === pet.id ? '主桌宠' : '设为主桌宠'}
              </Button>
            </div>
          ) : null}

          <SectionCard
            icon={<Plug className="h-4 w-4" />}
            title="连接"
            description="SoulMy API 与 JS 模版"
            collapsible
            open={openSections.connect}
            onOpenChange={(o) => toggleSection('connect', o)}
          >
            <Field
              label="API 地址"
              htmlFor="serverBase"
              hint="需带 /api 后缀，默认 https://soulmy.top/api"
            >
              <Input
                id="serverBase"
                value={form.serverBase}
                onChange={(e) => patchGlobal('serverBase', e.target.value)}
                placeholder="https://soulmy.top/api"
              />
            </Field>
            {pet ? (
              <Field label="当前桌宠名称" htmlFor="petName" hint="左侧仓库也可改 jsSourceId">
                <Input
                  id="petName"
                  value={pet.name}
                  onChange={(e) => patchPet(pet.id, { name: e.target.value, title: e.target.value })}
                  placeholder="显示名"
                />
              </Field>
            ) : null}
          </SectionCard>

          <SectionCard
            icon={<Bot className="h-4 w-4" />}
            title="智能体"
            description="对话凭证与传输方式"
            collapsible
            open={openSections.agent}
            onOpenChange={(o) => toggleSection('agent', o)}
          >
            <Field label="智能体 ID（assistantId）" htmlFor="assistantId">
              <Input
                id="assistantId"
                value={form.assistantId}
                onChange={(e) => patchGlobal('assistantId', e.target.value)}
                placeholder="必填"
                spellCheck={false}
              />
            </Field>
            <Field label="API Key" htmlFor="apiKey" hint="Access Keys 中创建的密钥">
              <div className="relative">
                <KeyRound className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
                <Input
                  id="apiKey"
                  value={form.apiKey}
                  onChange={(e) => patchGlobal('apiKey', e.target.value)}
                  placeholder="soulnexus_…"
                  autoComplete="off"
                  className="pl-8"
                />
              </div>
            </Field>
            <Field label="传输方式" htmlFor="transport">
              <SegmentedControl
                value={(form.transport || 'websocket') as 'websocket' | 'webrtc'}
                onChange={(v) => patchGlobal('transport', v)}
                options={[
                  { value: 'websocket', label: 'WS', description: 'websocket' },
                  { value: 'webrtc', label: 'RTC', description: 'webrtc' },
                ]}
              />
            </Field>
          </SectionCard>

          <SectionCard
            icon={<Power className="h-4 w-4" />}
            title="常驻"
            description="开机自启与系统级全局快捷键（无需点选桌宠）"
            collapsible
            open={openSections.resident}
            onOpenChange={(o) => toggleSection('resident', o)}
          >
            <div className="flex items-center justify-between gap-4 rounded-xl bg-[rgb(232,232,235)]/80 px-3 py-2.5">
              <div className="min-w-0">
                <p className="text-sm font-medium">开机自启</p>
                <p className="text-[11px] text-muted-foreground mt-0.5">
                  登录后后台启动，不弹控制面板
                </p>
              </div>
              <Switch
                id="openAtLogin"
                checked={Boolean(form.openAtLogin)}
                onCheckedChange={(v) => void onToggleLogin(v)}
              />
            </div>
            <p className="text-[10px] text-muted-foreground px-0.5">
              全局快捷键作用于「主桌宠」（带星标）；多桌宠时文字/语音仍只控制主桌宠。
            </p>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <Field label="控制面板快捷键" htmlFor="settingsHotkey" hint="默认 Ctrl+Alt+P（⌘⌥P）">
                <Input
                  id="settingsHotkey"
                  value={form.settingsHotkey}
                  onChange={(e) => patchGlobal('settingsHotkey', e.target.value)}
                  spellCheck={false}
                  className="font-mono text-xs"
                />
              </Field>
              <Field label="对话面板快捷键" htmlFor="panelHotkey">
                <Input
                  id="panelHotkey"
                  value={form.panelHotkey}
                  onChange={(e) => patchGlobal('panelHotkey', e.target.value)}
                  spellCheck={false}
                  className="font-mono text-xs"
                />
              </Field>
              <Field label="全局语音快捷键" htmlFor="voiceHotkey">
                <Input
                  id="voiceHotkey"
                  value={form.voiceHotkey}
                  onChange={(e) => patchGlobal('voiceHotkey', e.target.value)}
                  spellCheck={false}
                  className="font-mono text-xs"
                />
              </Field>
              <Field label="全局文字对话快捷键" htmlFor="talkHotkey">
                <Input
                  id="talkHotkey"
                  value={form.talkHotkey}
                  onChange={(e) => patchGlobal('talkHotkey', e.target.value)}
                  spellCheck={false}
                  className="font-mono text-xs"
                />
              </Field>
            </div>
          </SectionCard>

          {pet ? (
            <SectionCard
              icon={<Palette className="h-4 w-4" />}
              title="外观与行为"
              description="尺寸、位置与桌宠自主行为（当前选中）"
              collapsible
              open={openSections.appearance}
              onOpenChange={(o) => toggleSection('appearance', o)}
            >
              <Field label="挂件标题" htmlFor="titleAppear">
                <Input
                  id="titleAppear"
                  value={pet.title}
                  onChange={(e) => patchPet(pet.id, { title: e.target.value })}
                  placeholder="懒懒"
                />
              </Field>
              <Field label="初始位置">
                <SegmentedControl
                  value={pet.position || 'right'}
                  onChange={(v) => patchPet(pet.id, { position: v })}
                  options={[
                    { value: 'left', label: '左下角', description: '靠左停靠' },
                    { value: 'right', label: '右下角', description: '靠右停靠' },
                  ]}
                />
              </Field>
              <Field label="尺寸（px）" htmlFor="size" hint="96～256，默认 160">
                <Input
                  id="size"
                  type="number"
                  min={96}
                  max={256}
                  value={pet.size || 160}
                  onChange={(e) => patchPet(pet.id, { size: Number(e.target.value) || 160 })}
                />
              </Field>
              <div className="space-y-2 pt-1">
                <div className="flex items-center justify-between gap-4 rounded-xl bg-[rgb(232,232,235)]/80 px-3 py-2.5">
                  <div className="min-w-0">
                    <p className="text-sm font-medium">自主游荡</p>
                    <p className="text-[11px] text-muted-foreground mt-0.5">空闲时在屏幕上挪位置</p>
                  </div>
                  <Switch
                    checked={pet.autoWander !== false}
                    onCheckedChange={(v) => patchPet(pet.id, { autoWander: v })}
                  />
                </div>
                <div className="flex items-center justify-between gap-4 rounded-xl bg-[rgb(232,232,235)]/80 px-3 py-2.5">
                  <div className="min-w-0">
                    <p className="text-sm font-medium">主动聊天</p>
                    <p className="text-[11px] text-muted-foreground mt-0.5">偶尔主动跟你说一句</p>
                  </div>
                  <Switch
                    checked={pet.autoChat !== false}
                    onCheckedChange={(v) => patchPet(pet.id, { autoChat: v })}
                  />
                </div>
                <div className="flex items-center justify-between gap-4 rounded-xl bg-[rgb(232,232,235)]/80 px-3 py-2.5">
                  <div className="min-w-0">
                    <p className="text-sm font-medium">敲代码监听</p>
                    <p className="text-[11px] text-muted-foreground mt-0.5">
                      全局监听打字（macOS 需辅助功能权限）
                    </p>
                  </div>
                  <Switch
                    checked={pet.watchCoding !== false}
                    onCheckedChange={(v) => patchPet(pet.id, { watchCoding: v })}
                  />
                </div>
              </div>
            </SectionCard>
          ) : null}

          {status.msg ? (
            <div
              className={cn(
                'rounded-xl border px-3.5 py-2.5 text-xs whitespace-pre-wrap shadow-sm',
                status.kind === 'ok' && 'border-emerald-200/80 bg-white text-emerald-800',
                status.kind === 'err' && 'border-red-200/80 bg-white text-red-800',
                (status.kind === 'loading' || status.kind === 'idle') &&
                  'border-black/[0.06] bg-white text-muted-foreground',
              )}
            >
              {status.msg}
            </div>
          ) : null}
        </main>
      </div>

      <footer className="fixed bottom-0 inset-x-0 border-t border-black/[0.05] bg-[rgb(232,232,235)]/95 backdrop-blur-md px-4 py-3 z-20">
        <div className="flex gap-2 max-w-full pl-[248px]">
          <Button
            type="button"
            variant="outline"
            className="flex-1 bg-white"
            disabled={testing || saving}
            onClick={() => void onTest()}
          >
            {testing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plug className="h-4 w-4" />}
            测试连接
          </Button>
          <Button
            type="button"
            className="flex-1"
            disabled={saving || testing}
            onClick={() => void onSave()}
          >
            {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
            保存并启动
          </Button>
        </div>
      </footer>
    </div>
  )
}
