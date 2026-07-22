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
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Field } from '@/components/ui/field'
import { SegmentedControl } from '@/components/ui/segmented-control'
import { SectionCard } from '@/components/ui/section-card'
import type { PetConfig } from '@/vite-env'
import { cn } from '@/lib/utils'

const EMPTY: PetConfig = {
  serverBase: 'https://soulmy.top/api',
  jsSourceId: 'js_75cad2ab4f9f142a',
  assistantId: '8859281265343332864',
  apiKey: 'soulnexus_user_PI2mRsBxioqpTkAS3K3yG3Z_YzY2smCsidEWJRuamMI',
  transport: 'websocket',
  title: '懒懒',
  position: 'right',
  primaryColor: '#18181B',
  size: 160,
  autoWander: true,
  autoChat: true,
  watchCoding: true,
  settingsHotkey: 'CommandOrControl+Alt+P',
  panelHotkey: 'CommandOrControl+Alt+V',
  voiceHotkey: 'Alt+Shift+V',
  talkHotkey: 'Alt+Shift+T',
  openAtLogin: false,
}

function embedUrl(serverBase: string, jsSourceId: string) {
  const base = String(serverBase || '').replace(/\/+$/, '')
  const id = String(jsSourceId || '').trim()
  if (!id || id === 'YOUR_JS_SOURCE_ID' || id === 'default') {
    return `${base}/lingecho/embed/v1/embed.js`
  }
  return `${base}/lingecho/embed/v1/t/${encodeURIComponent(id)}/embed.js`
}

type StatusKind = 'idle' | 'ok' | 'err' | 'loading'

export default function SettingsPage() {
  const [form, setForm] = useState<PetConfig>(EMPTY)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [status, setStatus] = useState<{ kind: StatusKind; msg: string }>({
    kind: 'idle',
    msg: '',
  })
  const [appearanceOpen, setAppearanceOpen] = useState(true)

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
          setForm({ ...EMPTY, ...cfg })
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

  const configured = useMemo(() => {
    const id = form.assistantId.trim()
    return Boolean(id && id !== 'YOUR_ASSISTANT_ID' && form.serverBase.trim())
  }, [form.assistantId, form.serverBase])

  const patch = useCallback(<K extends keyof PetConfig>(key: K, value: PetConfig[K]) => {
    setForm((prev) => ({ ...prev, [key]: value }))
  }, [])

  const onTest = async () => {
    if (!form.serverBase.trim()) {
      setStatus({ kind: 'err', msg: '请先填写 API 地址' })
      return
    }
    const url = embedUrl(form.serverBase, form.jsSourceId)
    setTesting(true)
    setStatus({ kind: 'loading', msg: `正在测试 ${url} …` })
    try {
      const res = await fetch(url, { method: 'GET', cache: 'no-store' })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const text = await res.text()
      if (
        !text.includes('SoulNexus') &&
        !text.includes('lingecho-embed-root') &&
        !text.includes('lanlan-pet-root') &&
        !text.includes('__LingEchoConfig') &&
        !text.includes('__LanlanConfig')
      ) {
        throw new Error('响应不像有效的 SoulNexus embed.js')
      }
      setStatus({ kind: 'ok', msg: '连接成功，可以启动挂件' })
    } catch (e) {
      setStatus({
        kind: 'err',
        msg: `连接失败: ${(e as Error).message || e}\n请确认后端已启动且挂件已发布为 active`,
      })
    } finally {
      setTesting(false)
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
    setSaving(true)
    try {
      const saved = await api.saveConfig({
        serverBase: form.serverBase.trim(),
        jsSourceId: form.jsSourceId.trim(),
        assistantId: form.assistantId.trim(),
        apiKey: form.apiKey.trim(),
        transport: form.transport || 'websocket',
        title: form.title.trim() || '懒懒',
        position: form.position || 'right',
        primaryColor: form.primaryColor.trim() || '#18181B',
        size: Number(form.size) > 0 ? Number(form.size) : 160,
        autoWander: form.autoWander !== false,
        autoChat: form.autoChat !== false,
        watchCoding: form.watchCoding !== false,
        settingsHotkey: form.settingsHotkey.trim(),
        panelHotkey: form.panelHotkey.trim(),
        voiceHotkey: form.voiceHotkey.trim(),
        talkHotkey: form.talkHotkey.trim(),
        openAtLogin: Boolean(form.openAtLogin),
      })
      setForm((prev) => ({ ...prev, ...saved }))
      setStatus({ kind: 'ok', msg: '已保存并重新加载桌面挂件' })
    } catch (e) {
      setStatus({ kind: 'err', msg: `保存失败: ${(e as Error).message || e}` })
    } finally {
      setSaving(false)
    }
  }

  const onToggleLogin = async (enabled: boolean) => {
    patch('openAtLogin', enabled)
    if (!api) return
    try {
      const saved = await api.setOpenAtLogin(enabled)
      setForm((prev) => ({ ...prev, ...saved, openAtLogin: enabled }))
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

  return (
    <div className="min-h-screen bg-[rgb(232,232,235)]">
      <header className="sticky top-0 z-10 border-b border-black/[0.05] bg-[rgb(232,232,235)]/92 backdrop-blur-md px-5 py-4">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="flex items-center gap-2.5">
              <img
                src="./favicon.png"
                alt="SoulNexus"
                className="h-8 w-8 rounded-xl bg-white shadow-sm border border-black/[0.06] object-cover"
              />
              <div className="min-w-0">
                <h1 className="text-[15px] font-semibold tracking-tight text-[#18181B]">
                  SoulNexus Desktop
                </h1>
                <p className="mt-0.5 text-[11px] text-muted-foreground">
                  网页挂件挂到桌面 · 托盘常驻 · 开机自启
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

      <main className="space-y-3 p-4 pb-28">
        <SectionCard
          icon={<Plug className="h-4 w-4" />}
          title="连接"
          description="SoulNexus API 与 JS 模版"
        >
          <Field
            label="API 地址"
            htmlFor="serverBase"
            hint="需带 /api 后缀，默认 https://soulmy.top/api"
          >
            <Input
              id="serverBase"
              value={form.serverBase}
              onChange={(e) => patch('serverBase', e.target.value)}
              placeholder="https://soulmy.top/api"
            />
          </Field>
          <Field
            label="JS 模版 jsSourceId"
            htmlFor="jsSourceId"
            hint="控制台 → 网页挂件 → 挂件编号；留空使用内置默认 embed.js"
          >
            <Input
              id="jsSourceId"
              value={form.jsSourceId}
              onChange={(e) => patch('jsSourceId', e.target.value)}
              placeholder="js_…"
              spellCheck={false}
            />
          </Field>
        </SectionCard>

        <SectionCard
          icon={<Bot className="h-4 w-4" />}
          title="智能体"
          description="对话凭证与传输方式"
        >
          <Field label="智能体 ID（assistantId）" htmlFor="assistantId">
            <Input
              id="assistantId"
              value={form.assistantId}
              onChange={(e) => patch('assistantId', e.target.value)}
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
                onChange={(e) => patch('apiKey', e.target.value)}
                placeholder="soulnexus_…"
                autoComplete="off"
                className="pl-8"
              />
            </div>
          </Field>
          <Field label="传输方式" htmlFor="transport">
            <SegmentedControl
              value={(form.transport || 'websocket') as 'websocket' | 'webrtc'}
              onChange={(v) => patch('transport', v)}
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
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <Field
              label="控制面板快捷键"
              htmlFor="settingsHotkey"
              hint="默认 Ctrl+Alt+P（⌘⌥P）"
            >
              <Input
                id="settingsHotkey"
                value={form.settingsHotkey}
                onChange={(e) => patch('settingsHotkey', e.target.value)}
                spellCheck={false}
                className="font-mono text-xs"
              />
            </Field>
            <Field label="对话面板快捷键" htmlFor="panelHotkey" hint="全局；默认 Ctrl+Alt+V（与文字对话可并存）">
              <Input
                id="panelHotkey"
                value={form.panelHotkey}
                onChange={(e) => patch('panelHotkey', e.target.value)}
                spellCheck={false}
                className="font-mono text-xs"
              />
            </Field>
            <Field label="全局语音快捷键" htmlFor="voiceHotkey" hint="无需点桌宠；默认 Alt+Shift+V">
              <Input
                id="voiceHotkey"
                value={form.voiceHotkey}
                onChange={(e) => patch('voiceHotkey', e.target.value)}
                spellCheck={false}
                className="font-mono text-xs"
              />
            </Field>
            <Field label="全局文字对话快捷键" htmlFor="talkHotkey" hint="无需点桌宠；默认 Alt+Shift+T">
              <Input
                id="talkHotkey"
                value={form.talkHotkey}
                onChange={(e) => patch('talkHotkey', e.target.value)}
                spellCheck={false}
                className="font-mono text-xs"
              />
            </Field>
          </div>
        </SectionCard>

        <SectionCard
          icon={<Palette className="h-4 w-4" />}
          title="外观与行为"
          description="尺寸、位置与桌宠自主行为"
          collapsible
          open={appearanceOpen}
          onOpenChange={setAppearanceOpen}
        >
          <Field label="挂件标题" htmlFor="titleAppear" hint="显示名，默认「懒懒」">
            <Input
              id="titleAppear"
              value={form.title}
              onChange={(e) => patch('title', e.target.value)}
              placeholder="懒懒"
            />
          </Field>
          <Field label="初始位置">
            <SegmentedControl
              value={(form.position || 'right') as 'left' | 'right'}
              onChange={(v) => patch('position', v)}
              options={[
                { value: 'left', label: '左下角', description: '靠左停靠' },
                { value: 'right', label: '右下角', description: '靠右停靠' },
              ]}
            />
          </Field>
          <Field
            label="尺寸（px）"
            htmlFor="size"
            hint="96～256，默认 160"
          >
            <Input
              id="size"
              type="number"
              min={96}
              max={256}
              value={form.size || 160}
              onChange={(e) => patch('size', Number(e.target.value) || 160)}
            />
          </Field>
          <div className="space-y-2 pt-1">
            <div className="flex items-center justify-between gap-4 rounded-xl bg-[rgb(232,232,235)]/80 px-3 py-2.5">
              <div className="min-w-0">
                <p className="text-sm font-medium">自主游荡</p>
                <p className="text-[11px] text-muted-foreground mt-0.5">空闲时在屏幕上挪位置</p>
              </div>
              <Switch
                checked={form.autoWander !== false}
                onCheckedChange={(v) => patch('autoWander', v)}
              />
            </div>
            <div className="flex items-center justify-between gap-4 rounded-xl bg-[rgb(232,232,235)]/80 px-3 py-2.5">
              <div className="min-w-0">
                <p className="text-sm font-medium">主动聊天</p>
                <p className="text-[11px] text-muted-foreground mt-0.5">偶尔主动跟你说一句</p>
              </div>
              <Switch
                checked={form.autoChat !== false}
                onCheckedChange={(v) => patch('autoChat', v)}
              />
            </div>
            <div className="flex items-center justify-between gap-4 rounded-xl bg-[rgb(232,232,235)]/80 px-3 py-2.5">
              <div className="min-w-0">
                <p className="text-sm font-medium">敲代码监听</p>
                <p className="text-[11px] text-muted-foreground mt-0.5">
                  全局监听打字（macOS 需「辅助功能」权限；Windows 用预编译钩子，无需额外授权），驱动 coding / bug 等动作
                </p>
              </div>
              <Switch
                checked={form.watchCoding !== false}
                onCheckedChange={(v) => patch('watchCoding', v)}
              />
            </div>
          </div>
        </SectionCard>

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

      <footer className="fixed bottom-0 inset-x-0 border-t border-black/[0.05] bg-[rgb(232,232,235)]/95 backdrop-blur-md px-4 py-3">
        <div className="flex gap-2">
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
