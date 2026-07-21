import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import { Link, ExternalLink, ArrowLink } from '@/components/ui/link'
import { Card, Empty, Tooltip, Progress } from '@/components/ui'
import { Loading } from '@/components/ui/loading'
import { useToast } from '@/components/ui/toast'
import { Message } from '@arco-design/web-react'
import {
  Plus,
  Trash2,
  Edit2,
  Check,
  X,
  ArrowRight,
  Download,
  Upload,
  RefreshCw,
  Settings,
  Save,
  Search,
  ChevronRight,
  Loader2,
  Sparkles,
  Zap,
  Shield,
  Eye,
  EyeOff,
  Rocket,
  Globe,
  BookOpen,
  Github,
  Home,
  User,
  FileText,
} from 'lucide-react'

const SectionCard = ({
  title,
  description,
  children,
}: {
  title: string
  description?: string
  children: React.ReactNode
}) => (
  <section className="bg-[hsl(var(--card))] rounded-xl border border-[hsl(var(--border))] overflow-hidden">
    <div className="px-6 py-4 border-b border-[hsl(var(--border))] bg-[hsl(var(--card))]">
      <h2 className="text-lg font-semibold text-[hsl(var(--foreground))]">{title}</h2>
      {description && <p className="text-sm text-[hsl(var(--muted-foreground))] mt-1">{description}</p>}
    </div>
    <div className="p-6">{children}</div>
  </section>
)

const PreviewCard = ({ children }: { children: React.ReactNode }) => (
  <div className="flex flex-wrap items-center gap-3 p-5 rounded-lg bg-[hsl(var(--background))] border border-[hsl(var(--border))]">
    {children}
  </div>
)

const LabeledRow = ({ label, children }: { label: string; children: React.ReactNode }) => (
  <div className="flex items-start gap-4">
    <div className="w-28 shrink-0 pt-2.5 text-sm text-[hsl(var(--muted-foreground))]">{label}</div>
    <div className="flex-1 min-w-0">{children}</div>
  </div>
)

function InputFeedbackDemo() {
  const [value, setValue] = useState('')
  const [touched, setTouched] = useState(false)

  const isValid = value.length >= 3
  const showError = touched && !isValid

  return (
    <div className="p-5 rounded-lg bg-[hsl(var(--background))] border border-[hsl(var(--border))] space-y-3">
      <LabeledRow label="用户名">
        <div className="space-y-1.5">
          <Input
            placeholder="至少 3 个字符"
            value={value}
            onChange={(v) => setValue(v)}
            onBlur={() => setTouched(true)}
            status={showError ? 'error' : undefined}
          />
          {showError && (
            <p className="text-xs text-red-500">用户名至少需要 3 个字符</p>
          )}
          {touched && isValid && (
            <p className="text-xs text-green-600">用户名可用</p>
          )}
        </div>
      </LabeledRow>
      <div className="text-xs text-[hsl(var(--muted-foreground))]">
        输入内容: <code className="text-[hsl(var(--primary))]">{value || '(空)'}</code> · 字符数: <code className="text-[hsl(var(--primary))]">{value.length}</code>
      </div>
    </div>
  )
}

function ToastDemo() {
  const { toast } = useToast()

  return (
    <div className="p-5 rounded-lg bg-[hsl(var(--background))] border border-[hsl(var(--border))] space-y-3">
      <div className="flex flex-wrap gap-2">
        <Button size="sm" onClick={() => toast('操作成功完成', 'success')}>
          Success
        </Button>
        <Button size="sm" variant="destructive" onClick={() => toast('网络请求失败，请重试', 'error')}>
          Error
        </Button>
        <Button size="sm" onClick={() => toast('磁盘空间不足 10%', 'warning')}>
          Warning
        </Button>
        <Button size="sm" variant="outline" onClick={() => toast('系统将于今晚 22:00 维护', 'info')}>
          Info
        </Button>
      </div>
      <div className="text-xs text-[hsl(var(--muted-foreground))]">
        点击按钮触发从右侧滑入的毛玻璃 Toast 提醒，3 秒后自动消失，也可点击 × 关闭
      </div>
    </div>
  )
}

export default function ComponentsDemo() {
  const [loading, setLoading] = useState(false)
  const [selectValue, setSelectValue] = useState<string | undefined>()
  const [multiValue, setMultiValue] = useState<string[]>([])
  const [formSelect, setFormSelect] = useState<string | undefined>()

  const handleLoadingDemo = () => {
    setLoading(true)
    setTimeout(() => {
      setLoading(false)
      Message.success('操作完成')
    }, 1500)
  }

  const countryOptions = [
    { label: '🇨🇳 中国', value: 'cn' },
    { label: '🇺🇸 美国', value: 'us' },
    { label: '🇯🇵 日本', value: 'jp' },
    { label: '🇰🇷 韩国', value: 'kr' },
    { label: '🇬🇧 英国', value: 'uk' },
    { label: '🇫🇷 法国', value: 'fr' },
    { label: '🇩🇪 德国', value: 'de' },
    { label: '🇸🇬 新加坡', value: 'sg' },
  ]

  const roleOptions = [
    { label: '超级管理员', value: 'admin' },
    { label: '运营人员', value: 'operator' },
    { label: '普通用户', value: 'user' },
    { label: '只读访客', value: 'viewer', disabled: true },
  ]

  return (
    <div className="p-6 md:p-10 max-w-7xl mx-auto space-y-8 bg-[hsl(var(--background))] min-h-screen">
      {/* Hero */}
      <div className="text-center py-10 border border-[hsl(var(--border))] rounded-2xl bg-gradient-to-br from-[hsl(var(--card))] to-[hsl(var(--card)/0.6)]">
        <div className="inline-flex items-center gap-2 px-4 py-1.5 bg-[hsl(var(--primary)/0.08)] text-[hsl(var(--primary))] rounded-full text-sm font-medium mb-4">
          <Sparkles size={14} />
          <span>UI Components</span>
        </div>
        <h1 className="text-4xl font-bold text-[hsl(var(--foreground))] mb-3 tracking-tight">
          Components Demo
        </h1>
        <p className="text-[hsl(var(--muted-foreground))] max-w-xl mx-auto">
          基于 Arco Design + Tailwind CSS 封装的现代化 UI 组件库。遵循设计系统规范，支持完整的暗色模式。
        </p>
      </div>

      {/* ============ Button 部分 ============ */}
      <SectionCard title="Button 变体" description="8 种预定义的按钮样式，覆盖绝大多数业务场景">
        <div className="grid gap-5">
          <PreviewCard>
            <Button variant="primary">Primary</Button>
            <Button variant="secondary">Secondary</Button>
            <Button variant="outline">Outline</Button>
            <Button variant="ghost">Ghost</Button>
            <Button variant="destructive">Destructive</Button>
            <Button variant="success">Success</Button>
            <Button variant="warning">Warning</Button>
            <Button variant="link">Link Button</Button>
          </PreviewCard>
        </div>
      </SectionCard>

      <SectionCard title="Button 尺寸" description="从紧凑的 xs 到醒目的 lg，还有专门的图标按钮尺寸">
        <PreviewCard>
          <Button size="xs">Extra Small</Button>
          <Button size="sm">Small</Button>
          <Button size="md">Medium</Button>
          <Button size="lg">Large</Button>
          <Button size="icon" variant="outline" title="新建">
            <Plus size={18} />
          </Button>
          <Button size="icon" variant="primary" title="搜索">
            <Search size={18} />
          </Button>
          <Button size="icon" variant="ghost" title="设置">
            <Settings size={18} />
          </Button>
        </PreviewCard>
      </SectionCard>

      <SectionCard title="带图标的按钮" description="通过 leftIcon / rightIcon 属性，灵活组合图标与文字">
        <div className="space-y-4">
          <PreviewCard>
            <Button leftIcon={<Plus size={16} />}>新建项目</Button>
            <Button variant="destructive" leftIcon={<Trash2 size={16} />}>删除</Button>
            <Button variant="outline" leftIcon={<Edit2 size={16} />}>编辑</Button>
            <Button variant="success" leftIcon={<Check size={16} />}>确认</Button>
            <Button variant="warning" leftIcon={<Zap size={16} />}>快速操作</Button>
            <Button variant="ghost" rightIcon={<ArrowRight size={16} />}>下一步</Button>
          </PreviewCard>
          <PreviewCard>
            <Button variant="outline" leftIcon={<Download size={16} />}>下载</Button>
            <Button variant="outline" leftIcon={<Upload size={16} />}>上传</Button>
            <Button variant="secondary" leftIcon={<RefreshCw size={16} />}>刷新</Button>
            <Button variant="primary" leftIcon={<Save size={16} />}>保存</Button>
            <Button variant="outline" leftIcon={<Search size={16} />}>搜索</Button>
            <Button variant="ghost" leftIcon={<Shield size={16} />}>安全中心</Button>
            <Button variant="success" leftIcon={<Rocket size={16} />}>立即发布</Button>
          </PreviewCard>
        </div>
      </SectionCard>

      <SectionCard title="圆角风格" description="5 种圆角选项，适应不同的设计风格">
        <PreviewCard>
          <Button rounded="none">Sharp</Button>
          <Button rounded="sm">Small</Button>
          <Button rounded="md">Medium</Button>
          <Button rounded="lg">Large</Button>
          <Button rounded="full" leftIcon={<ChevronRight size={16} />}>Pill</Button>
        </PreviewCard>
      </SectionCard>

      <SectionCard title="交互状态" description="Disabled / Loading / Hover / Focus 状态完整支持">
        <div className="space-y-4">
          <PreviewCard>
            <Button disabled>已禁用</Button>
            <Button variant="outline" disabled>已禁用</Button>
            <Button variant="ghost" disabled>已禁用</Button>
            <Button variant="destructive" disabled>已禁用</Button>
          </PreviewCard>
          <PreviewCard>
            <Button loading={loading} onClick={handleLoadingDemo} leftIcon={!loading ? <RefreshCw size={16} /> : undefined}>
              {loading ? '加载中...' : '点击加载'}
            </Button>
            <Button variant="outline" loading={loading}>处理中</Button>
            <Button variant="success" loading={loading} leftIcon={!loading ? <Check size={16} /> : undefined}>
              {loading ? '保存中' : '保存'}
            </Button>
            <Button size="icon" loading={loading}>
              {!loading && <Loader2 size={18} />}
            </Button>
          </PreviewCard>
        </div>
      </SectionCard>

      <SectionCard title="块级按钮" description="设置 block 属性可让按钮占满父容器宽度">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Button block variant="primary">全宽 Primary</Button>
          <Button block variant="outline">全宽 Outline</Button>
          <Button block variant="success" size="lg">全宽 Success (Large)</Button>
        </div>
      </SectionCard>

      <SectionCard title="实际场景组合" description="表单、向导、危险操作等常见场景的使用示例">
        <div className="space-y-6">
          <div className="flex items-center justify-between p-5 rounded-lg bg-[hsl(var(--background))] border border-[hsl(var(--border))]">
            <div>
              <div className="font-medium text-[hsl(var(--foreground))]">表单操作</div>
              <div className="text-sm text-[hsl(var(--muted-foreground))] mt-1">标准的保存 / 取消组合</div>
            </div>
            <div className="flex items-center gap-2">
              <Button variant="ghost">取消</Button>
              <Button leftIcon={<Save size={16} />}>保存</Button>
            </div>
          </div>

          <div className="flex items-center justify-between p-5 rounded-lg bg-[hsl(var(--background))] border border-[hsl(var(--border))]">
            <div>
              <div className="font-medium text-[hsl(var(--foreground))]">危险操作</div>
              <div className="text-sm text-[hsl(var(--muted-foreground))] mt-1">删除后不可恢复，请谨慎操作</div>
            </div>
            <div className="flex items-center gap-2">
              <Button variant="ghost">取消</Button>
              <Button variant="destructive" leftIcon={<Trash2 size={16} />}>删除</Button>
            </div>
          </div>

          <div className="flex items-center justify-between p-5 rounded-lg bg-[hsl(var(--background))] border border-[hsl(var(--border))]">
            <div>
              <div className="font-medium text-[hsl(var(--foreground))]">多步骤向导</div>
              <div className="text-sm text-[hsl(var(--muted-foreground))] mt-1">步骤 2 / 4</div>
            </div>
            <div className="flex items-center gap-2">
              <Button variant="outline" leftIcon={<X size={16} />}>取消</Button>
              <Button variant="secondary">上一步</Button>
              <Button variant="primary" rightIcon={<ArrowRight size={16} />}>下一步</Button>
            </div>
          </div>

          <div className="flex items-center justify-between p-5 rounded-lg bg-[hsl(var(--background))] border border-[hsl(var(--border))]">
            <div>
              <div className="font-medium text-[hsl(var(--foreground))]">空状态操作</div>
              <div className="text-sm text-[hsl(var(--muted-foreground))] mt-1">还没有任何数据，点击下方按钮创建</div>
            </div>
            <div className="flex items-center gap-2">
              <Button variant="outline" leftIcon={<Eye size={16} />}>查看文档</Button>
              <Button size="lg" variant="primary" leftIcon={<Plus size={16} />}>新建</Button>
            </div>
          </div>
        </div>
      </SectionCard>

      {/* ============ Select 部分 ============ */}
      <div id="select-demo" className="h-0" />
      <SectionCard title="Select 下拉选择" description="支持单选、多选、搜索、清空，完全贴合设计主题">
        <div className="space-y-6">
          <LabeledRow label="基础单选">
            <Select
              placeholder="请选择一个国家"
              options={countryOptions}
              value={selectValue}
              onChange={(val) => setSelectValue(val as string)}
            />
          </LabeledRow>

          <LabeledRow label="多选模式">
            <Select
              mode="multiple"
              placeholder="可以选择多个国家"
              options={countryOptions}
              value={multiValue}
              onChange={(val) => setMultiValue(val as string[])}
            />
          </LabeledRow>

          <LabeledRow label="禁用状态">
            <Select
              placeholder="此组件已禁用"
              options={countryOptions}
              disabled
            />
          </LabeledRow>

          <LabeledRow label="含禁用项">
            <Select
              placeholder="选择用户角色"
              options={roleOptions}
            />
          </LabeledRow>

          <LabeledRow label="带默认值">
            <Select
              options={countryOptions}
              defaultValue="cn"
            />
          </LabeledRow>

          <LabeledRow label="Small 尺寸">
            <Select size="sm" options={countryOptions} placeholder="紧凑版选择器" />
          </LabeledRow>

          <LabeledRow label="Large 尺寸">
            <Select size="lg" options={countryOptions} placeholder="大尺寸选择器" />
          </LabeledRow>

          <div className="pt-4">
            <div className="text-sm text-[hsl(var(--muted-foreground))] mb-3">在表单中的使用</div>
            <div className="p-5 rounded-lg bg-[hsl(var(--background))] border border-[hsl(var(--border))] space-y-4">
              <LabeledRow label="所属国家">
                <Select options={countryOptions} value={formSelect} onChange={(v) => setFormSelect(v as string)} placeholder="请选择" />
              </LabeledRow>
              <LabeledRow label="当前角色">
                <Select options={roleOptions} defaultValue="operator" placeholder="请选择" />
              </LabeledRow>
              <div className="flex items-center gap-2 pt-2">
                <Button variant="outline" onClick={() => { setFormSelect(undefined) }}>重置</Button>
                <Button leftIcon={<Check size={16} />}>提交</Button>
                {formSelect && (
                  <span className="text-sm text-[hsl(var(--muted-foreground))]">
                    当前选择: <code className="text-[hsl(var(--primary))]">{formSelect}</code>
                  </span>
                )}
              </div>
            </div>
          </div>
        </div>
      </SectionCard>

      {/* ============ Link 部分 ============ */}
      <div id="link-demo" className="h-0" />
      <SectionCard title="Link 链接" description="Link 组件提供路由跳转和外部链接，自动跟随主题配色，支持多种视觉变体">
        <div className="space-y-6">
          <div>
            <div className="text-sm text-[hsl(var(--muted-foreground))] mb-3">5 种视觉变体</div>
            <PreviewCard>
              <Link to="/assistant-manager">Default</Link>
              <Link to="/assistant-manager" variant="primary">Primary</Link>
              <Link to="/assistant-manager" variant="muted">Muted</Link>
              <Link to="/assistant-manager" variant="nav">Nav</Link>
              <Link to="/assistant-manager" variant="underline">Underline</Link>
            </PreviewCard>
          </div>

          <div>
            <div className="text-sm text-[hsl(var(--muted-foreground))] mb-3">4 种字号</div>
            <PreviewCard>
              <Link to="/assistant-manager" size="xs">xs 文本</Link>
              <Link to="/assistant-manager" size="sm">sm 文本</Link>
              <Link to="/assistant-manager" size="md">md 文本</Link>
              <Link to="/assistant-manager" size="lg">lg 文本</Link>
            </PreviewCard>
          </div>

          <div>
            <div className="text-sm text-[hsl(var(--muted-foreground))] mb-3">带图标</div>
            <PreviewCard>
              <Link to="/assistant-manager" leftIcon={<Home size={14} />}>返回首页</Link>
              <Link to="/profile" leftIcon={<User size={14} />} variant="primary">个人中心</Link>
              <Link to="/assistant-manager" rightIcon={<FileText size={14} />} variant="muted">查看文档</Link>
            </PreviewCard>
          </div>

          <div>
            <div className="text-sm text-[hsl(var(--muted-foreground))] mb-3">ArrowLink — 带箭头的快捷链接</div>
            <PreviewCard>
              <ArrowLink to="/assistant-manager">了解更多</ArrowLink>
              <ArrowLink to="/assistant-manager" variant="muted" size="sm">查看详情</ArrowLink>
              <ArrowLink to="/assistant-manager" variant="default" size="lg">开始使用</ArrowLink>
            </PreviewCard>
          </div>

          <div>
            <div className="text-sm text-[hsl(var(--muted-foreground))] mb-3">ExternalLink — 外部链接（自动打开新标签页）</div>
            <PreviewCard>
              <ExternalLink href="https://react.dev">React 官网</ExternalLink>
              <ExternalLink href="https://arco.design" variant="primary" leftIcon={<Globe size={14} />}>
                Arco Design
              </ExternalLink>
              <ExternalLink href="https://github.com" variant="muted" leftIcon={<Github size={14} />}>
                GitHub
              </ExternalLink>
              <ExternalLink href="https://tailwindcss.com/docs" variant="underline" leftIcon={<BookOpen size={14} />}>
                Tailwind Docs
              </ExternalLink>
            </PreviewCard>
          </div>

          <div>
            <div className="text-sm text-[hsl(var(--muted-foreground))] mb-3">外部链接不带图标</div>
            <PreviewCard>
              <ExternalLink href="https://react.dev" showExternalIcon={false}>跳转到 React</ExternalLink>
              <ExternalLink href="https://arco.design" showExternalIcon={false} variant="muted">跳转到 Arco</ExternalLink>
            </PreviewCard>
          </div>

          <div>
            <div className="text-sm text-[hsl(var(--muted-foreground))] mb-3">实际导航场景</div>
            <div className="flex flex-wrap gap-6 p-5 rounded-lg bg-[hsl(var(--background))] border border-[hsl(var(--border))]">
              <div className="flex flex-col gap-3">
                <div className="text-xs font-medium uppercase tracking-wider text-[hsl(var(--muted-foreground))]">主导航</div>
                <nav className="flex flex-col gap-2">
                  <Link to="/assistant-manager" variant="nav" leftIcon={<Home size={14} />}>控制台</Link>
                  <Link to="/profile" variant="nav" leftIcon={<User size={14} />}>账户设置</Link>
                  <Link to="/assistant-manager" variant="nav" leftIcon={<FileText size={14} />}>智能体</Link>
                </nav>
              </div>
              <div className="flex flex-col gap-3">
                <div className="text-xs font-medium uppercase tracking-wider text-[hsl(var(--muted-foreground))]">快捷操作</div>
                <nav className="flex flex-col gap-2">
                  <ArrowLink to="/assistant-manager" variant="primary">创建新项目</ArrowLink>
                  <ArrowLink to="/assistant-manager" variant="muted" size="sm">查看定价</ArrowLink>
                  <ArrowLink to="/assistant-manager" variant="default">浏览模板库</ArrowLink>
                </nav>
              </div>
              <div className="flex flex-col gap-3">
                <div className="text-xs font-medium uppercase tracking-wider text-[hsl(var(--muted-foreground))]">外部资源</div>
                <nav className="flex flex-col gap-2">
                  <ExternalLink href="https://react.dev" variant="muted">React 学习资源</ExternalLink>
                  <ExternalLink href="https://arco.design" variant="muted">Arco 设计系统</ExternalLink>
                  <ExternalLink href="https://tailwindcss.com" variant="muted">Tailwind CSS</ExternalLink>
                </nav>
              </div>
            </div>
          </div>
        </div>
      </SectionCard>

      {/* ============ Input 部分 ============ */}
      <div id="input-demo" className="h-0" />
      <SectionCard title="Input 输入框" description="3 种变体 × 3 种尺寸，覆盖表单、搜索、密码等场景">
        <div className="space-y-6">
          <div>
            <div className="text-sm text-[hsl(var(--muted-foreground))] mb-3">变体 Variants</div>
            <div className="space-y-4">
              <LabeledRow label="Default">
                <Input placeholder="默认样式" />
              </LabeledRow>
              <LabeledRow label="Filled">
                <Input variant="filled" placeholder="填充背景样式" />
              </LabeledRow>
              <LabeledRow label="Outline">
                <Input variant="outline" placeholder="透明背景样式" />
              </LabeledRow>
            </div>
          </div>

          <div>
            <div className="text-sm text-[hsl(var(--muted-foreground))] mb-3">尺寸 Sizes</div>
            <div className="space-y-4">
              <LabeledRow label="Small">
                <Input size="sm" placeholder="紧凑尺寸 (sm)" />
              </LabeledRow>
              <LabeledRow label="Medium">
                <Input size="md" placeholder="默认尺寸 (md)" />
              </LabeledRow>
              <LabeledRow label="Large">
                <Input size="lg" placeholder="大尺寸 (lg)" />
              </LabeledRow>
            </div>
          </div>

          <div>
            <div className="text-sm text-[hsl(var(--muted-foreground))] mb-3">Password 密码输入</div>
            <div className="space-y-4">
              <LabeledRow label="密码">
                <Input.Password placeholder="请输入密码" />
              </LabeledRow>
              <LabeledRow label="Filled 密码">
                <Input.Password variant="filled" placeholder="填充样式密码" />
              </LabeledRow>
            </div>
          </div>

          <div>
            <div className="text-sm text-[hsl(var(--muted-foreground))] mb-3">TextArea 多行文本</div>
            <div className="space-y-4">
              <LabeledRow label="默认">
                <Input.TextArea placeholder="请输入详细描述..." autoSize={{ minRows: 3, maxRows: 6 }} />
              </LabeledRow>
              <LabeledRow label="Filled">
                <Input.TextArea variant="filled" placeholder="填充样式的多行文本" autoSize={{ minRows: 2 }} />
              </LabeledRow>
              <LabeledRow label="Small">
                <Input.TextArea size="sm" placeholder="紧凑尺寸多行文本" autoSize={{ minRows: 2 }} />
              </LabeledRow>
            </div>
          </div>

          <div>
            <div className="text-sm text-[hsl(var(--muted-foreground))] mb-3">交互状态</div>
            <div className="space-y-4">
              <LabeledRow label="禁用">
                <Input disabled placeholder="此输入框已禁用" />
              </LabeledRow>
              <LabeledRow label="只读">
                <Input readOnly value="只读内容，不可编辑" />
              </LabeledRow>
              <LabeledRow label="带状态">
                <Input status="error" placeholder="输入有误" />
              </LabeledRow>
            </div>
          </div>

          <div>
            <div className="text-sm text-[hsl(var(--muted-foreground))] mb-3">前缀 / 后缀</div>
            <div className="space-y-4">
              <LabeledRow label="搜索框">
                <Input prefix={<Search size={16} />} placeholder="搜索..." />
              </LabeledRow>
              <LabeledRow label="带后缀">
                <Input suffix="@gmail.com" placeholder="用户名" />
              </LabeledRow>
              <LabeledRow label="数字输入">
                <Input prefix="¥" placeholder="0.00" />
              </LabeledRow>
            </div>
          </div>

          <div className="pt-4">
            <div className="text-sm text-[hsl(var(--muted-foreground))] mb-3">表单场景组合</div>
            <div className="p-5 rounded-lg bg-[hsl(var(--background))] border border-[hsl(var(--border))] space-y-4">
              <LabeledRow label="用户名">
                <Input placeholder="请输入用户名" prefix={<User size={16} />} />
              </LabeledRow>
              <LabeledRow label="邮箱">
                <Input placeholder="user@example.com" />
              </LabeledRow>
              <LabeledRow label="密码">
                <Input.Password placeholder="请输入密码" />
              </LabeledRow>
              <LabeledRow label="备注">
                <Input.TextArea placeholder="可选备注信息..." autoSize={{ minRows: 2 }} />
              </LabeledRow>
              <div className="flex items-center gap-2 pt-2">
                <Button variant="outline">重置</Button>
                <Button leftIcon={<Save size={16} />}>提交</Button>
              </div>
            </div>
          </div>

          <div>
            <div className="text-sm text-[hsl(var(--muted-foreground))] mb-3">实时输入反馈</div>
            <InputFeedbackDemo />
          </div>

          <div>
            <div className="text-sm text-[hsl(var(--muted-foreground))] mb-3">Wave 浮动标签 Input</div>
            <PreviewCard>
              <div className="w-full max-w-xs space-y-4">
                <Input variant="wave" label="Name" block />
                <Input variant="wave" label="Email" block defaultValue="demo@example.com" />
              </div>
            </PreviewCard>
          </div>

          <div>
            <div className="text-sm text-[hsl(var(--muted-foreground))] mb-3">Loading / Empty</div>
            <PreviewCard>
              <Loading tip="加载中..." />
            </PreviewCard>
            <div className="mt-4 grid gap-4 sm:grid-cols-2">
              <Empty preset="no-data" description="暂无数据" />
              <Empty preset="no-message" description="暂无新消息" />
            </div>
          </div>

          <div>
            <div className="text-sm text-[hsl(var(--muted-foreground))] mb-3">Progress 进度条</div>
            <div className="space-y-6 max-w-lg">
              <Progress percent={30} />
              <Progress percent={80} animation status="normal" />
              <Progress percent={100} status="success" />
              <div className="flex flex-wrap items-center gap-8">
                <Progress type="circle" percent={20} width={80} />
                <Progress type="circle" percent={65} status="warning" width={80} />
              </div>
              <Progress size="mini" percent={45} showText={false} />
            </div>
          </div>

          <div>
            <div className="text-sm text-[hsl(var(--muted-foreground))] mb-3">Hint Tooltip / Elevated Card</div>
            <div className="flex flex-wrap items-start gap-8">
              <Tooltip variant="hint" hintLabel="Tip" content="Use Navbar to navigate the website quickly and easily." />
              <Card
                variant="elevated"
                icon={<FileText size={28} />}
                style={{ maxWidth: 280 }}
              >
                <p className="text-sm leading-relaxed mb-0">
                  分层卡片动效，hover 时轻微上浮并展示背景层叠效果。
                </p>
              </Card>
            </div>
          </div>

          <div>
            <div className="text-sm text-[hsl(var(--muted-foreground))] mb-3">Toast 消息提醒</div>
            <ToastDemo />
          </div>
        </div>
      </SectionCard>

      {/* Footer note */}
      <div className="text-center text-sm text-[hsl(var(--muted-foreground))] py-6">
        <p className="inline-flex items-center gap-2">
          <Eye size={14} />
          <span>Button · Select · Link · Input · Loading · Empty · Tooltip · Card · Progress</span>
          <EyeOff size={14} />
        </p>
      </div>
    </div>
  )
}
