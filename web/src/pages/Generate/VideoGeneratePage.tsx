import { useState } from 'react'
import { Button, Card, Empty, Input, Select } from '@/components/UI'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Clapperboard, Film, History, Play, Sparkles, Timer, Download } from 'lucide-react'

export default function VideoGeneratePage() {
  const [prompt, setPrompt] = useState('')
  const [duration, setDuration] = useState('4s')
  const [ratio, setRatio] = useState('16:9')
  const [motion, setMotion] = useState('medium')
  const [fps, setFps] = useState('24')

  return (
    <BaseLayout
      title="视频生成"
      description="左侧参数，中间预览，右侧任务状态与历史"
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="outline" size="sm">上传参考素材</Button>
          <Button size="sm" className="gap-1.5"><Sparkles size={16} />开始生成</Button>
        </div>
      }
    >
      <div className="min-h-[calc(100vh-8rem)] bg-[radial-gradient(circle_at_top_right,rgba(99,102,241,0.14),transparent_36%)]">
        <div className="mx-auto max-w-[1600px]">
          <div className="grid gap-5 xl:grid-cols-[340px_minmax(0,1fr)_300px]">
            <aside className="space-y-4">
              <Card className="space-y-4 border border-border/60 bg-card/80 p-4 shadow-sm backdrop-blur">
                <div>
                  <div className="mb-2 text-sm font-medium text-foreground">提示词</div>
                  <Input.TextArea
                    value={prompt}
                    onChange={(e) => setPrompt(e.target.value)}
                    placeholder="角色从左侧冲入画面，镜头跟随推进，粒子特效散开，电影感光影"
                    rows={7}
                  />
                </div>
                <div className="grid gap-3">
                  <div>
                    <div className="mb-2 text-sm font-medium text-foreground">时长</div>
                    <Select
                      value={duration}
                    onValueChange={setDuration}
                    options={[
                      { label: '2 秒', value: '2s' },
                      { label: '4 秒', value: '4s' },
                      { label: '6 秒', value: '6s' },
                      { label: '8 秒', value: '8s' },
                    ]}
                  />
                </div>
                <div>
                  <div className="mb-2 text-sm font-medium text-foreground">比例</div>
                  <Select
                    value={ratio}
                    onValueChange={setRatio}
                    options={[
                      { label: '16:9 横屏', value: '16:9' },
                      { label: '1:1 方形', value: '1:1' },
                      { label: '9:16 竖屏', value: '9:16' },
                    ]}
                  />
                </div>
                <div>
                  <div className="mb-2 text-sm font-medium text-foreground">运动幅度</div>
                  <Select
                    value={motion}
                    onValueChange={setMotion}
                    options={[
                      { label: '轻微', value: 'low' },
                      { label: '中等', value: 'medium' },
                      { label: '强烈', value: 'high' },
                    ]}
                  />
                </div>
                <div>
                  <div className="mb-2 text-sm font-medium text-foreground">帧率</div>
                  <Select
                    value={fps}
                    onValueChange={setFps}
                    options={[
                      { label: '12 FPS', value: '12' },
                      { label: '24 FPS', value: '24' },
                      { label: '30 FPS', value: '30' },
                    ]}
                  />
                </div>
              </div>
              <Button className="mt-2 w-full gap-1.5">
                <Film size={16} />
                生成视频
              </Button>
            </Card>

            <Card className="border border-border/60 bg-card/70 p-4 shadow-sm">
              <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-foreground">
                <Timer size={16} className="text-indigo-400" />
                场景模板
              </div>
              <div className="flex flex-wrap gap-2">
                {['技能释放', '角色登场', '镜头推进', '粒子爆发', '过场动画'].map((item) => (
                  <button
                    key={item}
                    type="button"
                    className="rounded-full border border-border/60 bg-background/70 px-3 py-1.5 text-xs text-foreground transition hover:border-indigo-400/40 hover:bg-indigo-500/10 hover:text-indigo-300"
                    onClick={() => setPrompt((prev) => (prev ? `${prev}，${item}` : item))}
                  >
                    {item}
                  </button>
                ))}
              </div>
            </Card>
          </aside>

          <section className="space-y-4">
            <Card className="min-h-[640px] border border-border/60 bg-card/70 p-4 shadow-sm">
              <div className="mb-4 flex items-center justify-between">
                <div>
                  <h2 className="text-base font-semibold text-foreground">预览舞台</h2>
                  <p className="text-xs text-muted-foreground">中间大预览，后续可接播放器与生成进度条。</p>
                </div>
                <div className="flex gap-2">
                  <Button size="sm" variant="outline" className="gap-1"><Play size={14} />预览</Button>
                  <Button size="sm" variant="outline" className="gap-1"><Download size={14} />导出</Button>
                </div>
              </div>
              <div className="grid min-h-[540px] place-items-center rounded-2xl border border-dashed border-border/70 bg-[linear-gradient(160deg,rgba(99,102,241,0.08),transparent_45%),hsl(var(--background)/0.7)] p-8">
                <Empty
                  title="等待生成视频"
                  description="生成完成后会在这里展示播放器、帧预览与导出入口。"
                  icon={<Clapperboard className="size-10 text-indigo-400/80" />}
                />
              </div>
            </Card>
          </section>

          <aside className="space-y-4">
            <Card className="border border-border/60 bg-card/80 p-4 shadow-sm">
              <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-foreground">
                <History size={16} className="text-indigo-400" />
                历史记录
              </div>
              <div className="space-y-3">
                {[1, 2, 3].map((item) => (
                  <div key={item} className="rounded-2xl border border-border/60 bg-background/70 p-3">
                    <div className="mb-2 flex aspect-video items-center justify-center rounded-xl border border-border/50 bg-muted/40">
                      <Film className="size-6 text-muted-foreground" />
                    </div>
                    <p className="line-clamp-2 text-xs text-muted-foreground">示例视频 {item} · {duration} · {ratio}</p>
                  </div>
                ))}
              </div>
            </Card>

            <Card className="border border-border/60 bg-card/80 p-4 shadow-sm">
              <div className="mb-2 text-sm font-semibold text-foreground">任务状态</div>
              <div className="space-y-2 text-sm text-muted-foreground">
                <div className="flex items-center justify-between rounded-xl bg-background/70 px-3 py-2">
                  <span>排队中</span>
                  <span className="text-foreground">0</span>
                </div>
                <div className="flex items-center justify-between rounded-xl bg-background/70 px-3 py-2">
                  <span>渲染中</span>
                  <span className="text-foreground">0</span>
                </div>
                <div className="flex items-center justify-between rounded-xl bg-background/70 px-3 py-2">
                  <span>已完成</span>
                  <span className="text-foreground">0</span>
                </div>
              </div>
            </Card>
          </aside>
        </div>
        </div>
      </div>
    </BaseLayout>
  )
}
