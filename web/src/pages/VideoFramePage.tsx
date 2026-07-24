import { useState } from 'react'
import { Button, Card, Empty, Input, Select } from '@/components/UI'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Slider } from '@arco-design/web-react'
import {
  History,
  Image as ImageIcon,
  PlaySquare,
  Settings2,
  Upload,
  Video,
  Zap,
  Clock3,
} from 'lucide-react'

export default function VideoFramePage() {
  const [fps, setFps] = useState(8)
  const [maxFrames, setMaxFrames] = useState(24)
  const [interval, setIntervalMs] = useState(1)
  const [outputName, setOutputName] = useState('video_frames')
  const [status] = useState('未上传')

  return (
    <BaseLayout
      title="视频抽帧"
      description="上传角色动作视频，自动识别关键帧并导出 PNG 序列"
    >
      <div className="space-y-6">
        <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_300px]">
          <main className="space-y-5">
            <Card className="border border-border/60 bg-card p-4 shadow-sm">
              <div className="grid gap-3 md:grid-cols-3">
                {[
                  { icon: Video, title: '1. 上传动作视频' },
                  { icon: Zap, title: '2. AI 智能抽帧识别' },
                  { icon: ImageIcon, title: '3. 生成导出精灵资源' },
                ].map((step) => {
                  const Icon = step.icon
                  return (
                    <div
                      key={step.title}
                      className="flex items-center gap-3 rounded-2xl border border-border/60 bg-background p-4"
                    >
                    <div className="flex size-11 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                      <Icon size={18} />
                    </div>
                    <p className="text-sm font-semibold text-foreground">{step.title}</p>
                  </div>
                )
              })}
            </div>
            <p className="mt-3 text-center text-xs text-muted-foreground">
              全程自动处理，无需人工干预，几分钟内完成基础素材抽取。
            </p>
          </Card>

          <Card className="border border-border/60 bg-card p-4 shadow-sm">
            <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_280px]">
              <label className="grid cursor-pointer place-items-center gap-3 rounded-2xl border border-dashed border-border bg-background px-6 py-16 text-center transition hover:border-primary/40 hover:bg-primary/5">
                <Upload className="size-10 text-primary" />
                <div>
                  <p className="text-base font-semibold text-foreground">拖拽或点击上传角色动作视频</p>
                  <p className="mt-1 text-sm text-muted-foreground">支持格式：MP4 / AVI / MOV</p>
                  <p className="mt-1 text-sm text-muted-foreground">单文件上限：≤ 500MB</p>
                </div>
                <input hidden type="file" accept="video/*" />
              </label>

              <div className="space-y-3 rounded-2xl border border-border/60 bg-background p-4">
                <div className="text-sm font-semibold text-foreground">使用小贴士</div>
                <div className="space-y-3 text-sm text-muted-foreground">
                  <div className="rounded-xl border border-border/50 bg-card p-3">
                    <div className="mb-1 flex items-center gap-2 font-medium text-foreground">
                      <Video size={15} />
                      视频准备
                    </div>
                    <p>上传 5~30 秒动作视频，背景尽量纯净。</p>
                  </div>
                  <div className="rounded-xl border border-border/50 bg-card p-3">
                    <div className="mb-1 flex items-center gap-2 font-medium text-foreground">
                      <Settings2 size={15} />
                      帧率设置
                    </div>
                    <p>像素角色推荐 8~12 FPS，演示动画建议 12~24 FPS。</p>
                  </div>
                  <div className="rounded-xl border border-border/50 bg-card p-3">
                    <div className="mb-1 flex items-center gap-2 font-medium text-foreground">
                      <ImageIcon size={15} />
                      导出选项
                    </div>
                    <p>可导出透明 PNG 序列，再接入精灵图工具。</p>
                  </div>
                </div>
              </div>
            </div>
          </Card>

          <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_300px]">
            <Card className="min-h-[420px] border border-border/60 bg-card p-4 shadow-sm">
              <div className="mb-4">
                <h3 className="text-sm font-semibold text-foreground">帧预览区</h3>
                <p className="text-xs text-muted-foreground">抽出的关键帧会显示在这里。</p>
              </div>
              <div className="grid min-h-[320px] place-items-center rounded-2xl border border-dashed border-border bg-background p-8">
                <Empty title="等待上传视频" description="上传后展示关键帧结果。" />
              </div>
            </Card>

            <div className="space-y-4">
              <Card className="border border-border/60 bg-card p-4 shadow-sm">
                <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-foreground">
                  <Settings2 size={16} />
                  处理状态
                </div>
                <div className="space-y-2 text-sm">
                  <div className="flex items-center justify-between rounded-xl border border-border/50 bg-background px-3 py-2">
                    <span>视频</span>
                    <span className="text-muted-foreground">{status}</span>
                  </div>
                  <div className="flex items-center justify-between rounded-xl border border-border/50 bg-background px-3 py-2">
                    <span>关键帧</span>
                    <span className="text-muted-foreground">等待识别</span>
                  </div>
                  <div className="flex items-center justify-between rounded-xl border border-border/50 bg-background px-3 py-2">
                    <span>进度</span>
                    <span className="text-muted-foreground">待处理</span>
                  </div>
                </div>
              </Card>

              <Card className="border border-border/60 bg-card p-4 shadow-sm">
                <div className="mb-3 flex items-center justify-between">
                  <div className="flex items-center gap-2 text-sm font-semibold text-foreground">
                    <History size={16} />
                    历史记录
                  </div>
                  <span className="text-xs text-muted-foreground">共 2 条</span>
                </div>
                <div className="space-y-3">
                  {['proxy.mp4', '动作序列示例.mp4'].map((name, idx) => (
                    <div key={name} className="rounded-xl border border-border/60 bg-background p-3">
                      <div className="flex items-center gap-3">
                        <div className="flex size-10 items-center justify-center rounded-xl bg-primary/10 text-primary">
                          <Clock3 size={16} />
                        </div>
                        <div className="min-w-0 flex-1">
                          <div className="truncate text-sm font-medium text-foreground">{name}</div>
                          <div className="text-xs text-muted-foreground">{78 - idx * 10} 帧 · 8 FPS</div>
                          <div className="mt-1 inline-flex rounded-full bg-emerald-500/10 px-2 py-0.5 text-[11px] text-emerald-600">
                            已完成
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </Card>
            </div>
          </div>
        </main>

        <aside className="space-y-4">
          <Card className="space-y-4 border border-border/60 bg-card p-4 shadow-sm">
            <div className="flex items-center gap-2 text-sm font-semibold text-foreground">
              <Settings2 size={16} />
              抽帧参数
            </div>
            <div>
              <label className="mb-2 block text-xs text-muted-foreground">输出名称</label>
              <Input value={outputName} onChange={(e) => setOutputName(e.target.value)} />
            </div>
            <div>
              <label className="mb-2 block text-xs text-muted-foreground">抽帧方式</label>
              <Select
                value="frame-interval"
                onValueChange={() => undefined}
                options={[
                  { label: '按帧间隔', value: 'frame-interval' },
                  { label: '按每秒抽帧', value: 'fps' },
                  { label: '关键帧优先', value: 'keyframe' },
                ]}
              />
            </div>
            <div>
              <div className="mb-2 flex items-center justify-between text-xs text-muted-foreground">
                <span>FPS</span>
                <span>{fps}</span>
              </div>
              <Slider min={1} max={30} value={fps} onChange={setFps} />
            </div>
            <div>
              <div className="mb-2 flex items-center justify-between text-xs text-muted-foreground">
                <span>最大帧数</span>
                <span>{maxFrames}</span>
              </div>
              <Slider min={1} max={240} value={maxFrames} onChange={setMaxFrames} />
            </div>
            <div>
              <div className="mb-2 flex items-center justify-between text-xs text-muted-foreground">
                <span>抽帧间隔</span>
                <span>{interval}</span>
              </div>
              <Slider min={1} max={12} value={interval} onChange={setIntervalMs} />
            </div>
            <Button className="w-full gap-1.5">
              <PlaySquare size={16} />
              开始抽帧
            </Button>
            <Button className="w-full" variant="outline">
              导出 PNG 序列
            </Button>
            <Button className="w-full" variant="outline">
              导出 ZIP
            </Button>
          </Card>
        </aside>
        </div>
      </div>
    </BaseLayout>
  )
}
