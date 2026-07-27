import { useState } from 'react'
import { Button, Card, Empty, Input, Select } from '@/components/UI'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Image as ImageIcon, History, Sparkles, Download, Heart, Layers3 } from 'lucide-react'

export default function ImageGeneratePage() {
  const [prompt, setPrompt] = useState('')
  const [negative, setNegative] = useState('')
  const [size, setSize] = useState('1024x1024')
  const [style, setStyle] = useState('pixel')
  const [count, setCount] = useState('4')
  const [strength, setStrength] = useState('0.7')

  return (
    <BaseLayout
      title="图片生成"
      description="左侧调参，中间结果区，右侧历史与收藏"
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="outline" size="sm">清空结果</Button>
          <Button variant="outline" size="sm">导入参考图</Button>
          <Button size="sm" className="gap-1.5"><Sparkles size={16} />开始生成</Button>
        </div>
      }
    >
      <div className="min-h-[calc(100vh-8rem)] bg-[radial-gradient(circle_at_top,rgba(168,85,247,0.12),transparent_40%)]">
        <div className="mx-auto max-w-[1600px]">
          <div className="grid gap-5 xl:grid-cols-[340px_minmax(0,1fr)_300px]">
            <aside className="space-y-4">
              <Card className="space-y-4 border border-border/60 bg-card/80 p-4 shadow-sm backdrop-blur">
                <div>
                  <div className="mb-2 text-sm font-medium text-foreground">提示词</div>
                  <Input.TextArea
                    value={prompt}
                    onChange={(e) => setPrompt(e.target.value)}
                    placeholder="一只像素风格的冒险家，站在夜晚的霓虹街角，厚涂光影，高细节"
                    rows={7}
                  />
                </div>
                <div>
                  <div className="mb-2 text-sm font-medium text-foreground">反向提示词</div>
                  <Input.TextArea
                    value={negative}
                    onChange={(e) => setNegative(e.target.value)}
                    placeholder="模糊，低质量，文字，水印"
                    rows={3}
                  />
                </div>
                <div className="grid gap-3">
                  <div>
                    <div className="mb-2 text-sm font-medium text-foreground">尺寸</div>
                    <Select
                      value={size}
                      onValueChange={setSize}
                      options={[
                        { label: '512 × 512', value: '512x512' },
                        { label: '768 × 768', value: '768x768' },
                        { label: '1024 × 1024', value: '1024x1024' },
                        { label: '1280 × 720', value: '1280x720' },
                      ]}
                    />
                  </div>
                  <div>
                    <div className="mb-2 text-sm font-medium text-foreground">风格</div>
                    <Select
                      value={style}
                      onValueChange={setStyle}
                      options={[
                        { label: '像素风', value: 'pixel' },
                        { label: '卡通', value: 'cartoon' },
                        { label: '写实', value: 'realistic' },
                        { label: '二次元', value: 'anime' },
                      ]}
                    />
                  </div>
                <div>
                  <div className="mb-2 text-sm font-medium text-foreground">生成数量</div>
                  <Select
                    value={count}
                    onValueChange={setCount}
                    options={[
                      { label: '1 张', value: '1' },
                      { label: '2 张', value: '2' },
                      { label: '4 张', value: '4' },
                    ]}
                  />
                </div>
                <div>
                  <div className="mb-2 text-sm font-medium text-foreground">创意强度</div>
                  <Select
                    value={strength}
                    onValueChange={setStrength}
                    options={[
                      { label: '保守 0.4', value: '0.4' },
                      { label: '均衡 0.7', value: '0.7' },
                      { label: '发散 0.9', value: '0.9' },
                    ]}
                  />
                </div>
              </div>
              <Button className="mt-2 w-full gap-1.5">
                <Sparkles size={16} />
                生成图片
              </Button>
            </Card>

            <Card className="border border-border/60 bg-card/70 p-4 shadow-sm">
              <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-foreground">
                <Layers3 size={16} className="text-primary" />
                快捷预设
              </div>
              <div className="flex flex-wrap gap-2">
                {['游戏道具', '角色立绘', 'UI 图标', '场景概念', '技能特效'].map((item) => (
                  <button
                    key={item}
                    type="button"
                    className="rounded-full border border-border/60 bg-background/70 px-3 py-1.5 text-xs text-foreground transition hover:border-primary/40 hover:bg-primary/10 hover:text-primary"
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
                  <h2 className="text-base font-semibold text-foreground">结果画廊</h2>
                  <p className="text-xs text-muted-foreground">大预览区 + 多图网格，后续可直接接生成结果。</p>
                </div>
                <div className="flex gap-2">
                  <Button size="sm" variant="outline" className="gap-1"><Download size={14} />批量下载</Button>
                  <Button size="sm" variant="outline" className="gap-1"><Heart size={14} />收藏</Button>
                </div>
              </div>
              <div className="grid min-h-[540px] place-items-center rounded-2xl border border-dashed border-border/70 bg-[linear-gradient(145deg,rgba(168,85,247,0.06),transparent_50%),hsl(var(--background)/0.7)] p-8">
                <Empty
                  title="等待生成"
                  description="生成结果会以大图 + 多图网格展示在这里。"
                  icon={<ImageIcon className="size-10 text-primary/70" />}
                />
              </div>
            </Card>
          </section>

          <aside className="space-y-4">
            <Card className="border border-border/60 bg-card/80 p-4 shadow-sm">
              <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-foreground">
                <History size={16} className="text-primary" />
                历史记录
              </div>
              <div className="space-y-3">
                {[1, 2, 3].map((item) => (
                  <div key={item} className="rounded-2xl border border-border/60 bg-background/70 p-3">
                    <div className="mb-2 flex aspect-video items-center justify-center rounded-xl border border-border/50 bg-muted/40">
                      <ImageIcon className="size-6 text-muted-foreground" />
                    </div>
                    <p className="line-clamp-2 text-xs text-muted-foreground">示例历史记录 {item} · 像素风 · 1024×1024</p>
                  </div>
                ))}
              </div>
            </Card>

            <Card className="border border-border/60 bg-card/80 p-4 shadow-sm">
              <div className="mb-2 text-sm font-semibold text-foreground">任务状态</div>
              <div className="space-y-2 text-sm text-muted-foreground">
                <div className="flex items-center justify-between rounded-xl bg-background/70 px-3 py-2">
                  <span>排队</span>
                  <span className="text-foreground">0</span>
                </div>
                <div className="flex items-center justify-between rounded-xl bg-background/70 px-3 py-2">
                  <span>生成中</span>
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
