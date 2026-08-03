/**
 * @fileoverview 无限地图玩家角色：精灵表切帧元数据。
 *
 * ## 来源
 * 与 ControlTest topdown 使用同一套 `TINA.png` 布局；此处为 **独立副本**，避免与 ControlTest 模块循环依赖。
 *
 * ## 约定
 * - `REGIONS` 的键名为历史/工具生成的短 id，与 `ANIMS[].frames` 一一对应。
 * - 每个区域为源图上的像素矩形 `(x,y,w,h)`；`extractFrame` 裁到独立小 Canvas 供每帧绘制。
 * - `ANIMS`：`speed` 与 `InfiniteMapScene` 中 `accum` 逻辑配合，数值越大切帧越快。
 */
export const DEFAULT_CHAR_URL = `${import.meta.env.BASE_URL}map/TINA.png`

/**
 * 源图 `TINA.png` 上的帧矩形（像素坐标，左上为原点）。
 * 键名无语义，仅作稳定引用。
 */
export const REGIONS: Record<string, { x: number; y: number; w: number; h: number }> = {
  '72hcl': { x: 210, y: 126, w: 21, h: 42 },
  rydce: { x: 189, y: 126, w: 21, h: 42 },
  '1et3y': { x: 231, y: 126, w: 21, h: 42 },
  bbcvv: { x: 0, y: 126, w: 28, h: 42 },
  foxtp: { x: 28, y: 126, w: 28, h: 42 },
  aw8dg: { x: 56, y: 126, w: 28, h: 42 },
  evrtr: { x: 84, y: 126, w: 28, h: 42 },
  pyoh8: { x: 112, y: 126, w: 28, h: 42 },
  t4rff: { x: 140, y: 126, w: 28, h: 42 },
  koy62: { x: 0, y: 0, w: 21, h: 42 },
  '3ygc0': { x: 21, y: 0, w: 21, h: 42 },
  yfrrb: { x: 42, y: 0, w: 21, h: 42 },
  '2enbr': { x: 63, y: 0, w: 21, h: 42 },
  s2yql: { x: 84, y: 0, w: 21, h: 42 },
  idc64: { x: 105, y: 0, w: 21, h: 42 },
  '8mwul': { x: 0, y: 42, w: 21, h: 42 },
  snwwj: { x: 21, y: 42, w: 21, h: 42 },
  ynglr: { x: 42, y: 42, w: 21, h: 42 },
  p3oo0: { x: 63, y: 42, w: 21, h: 42 },
  pfwvy: { x: 84, y: 42, w: 21, h: 42 },
  tvkvf: { x: 105, y: 42, w: 21, h: 42 },
  '3c66l': { x: 0, y: 84, w: 21, h: 42 },
  wq5ia: { x: 21, y: 84, w: 21, h: 42 },
  '11gwb': { x: 42, y: 84, w: 21, h: 42 },
  iitav: { x: 63, y: 84, w: 21, h: 42 },
  '360a7': { x: 84, y: 84, w: 21, h: 42 },
  ffd0g: { x: 105, y: 84, w: 21, h: 42 },
  ahlcx: { x: 126, y: 0, w: 21, h: 42 },
  '4i3vm': { x: 147, y: 0, w: 21, h: 42 },
  '0qwcd': { x: 168, y: 0, w: 21, h: 42 },
  y1030: { x: 189, y: 0, w: 21, h: 42 },
  '3sl87': { x: 210, y: 0, w: 21, h: 42 },
  '8kwsb': { x: 231, y: 0, w: 21, h: 42 },
  umveo: { x: 126, y: 42, w: 21, h: 42 },
  v6ado: { x: 147, y: 42, w: 21, h: 42 },
  syfy0: { x: 168, y: 42, w: 21, h: 42 },
  us0w8: { x: 189, y: 42, w: 21, h: 42 },
  pf2m2: { x: 210, y: 42, w: 21, h: 42 },
  '876dv': { x: 231, y: 42, w: 21, h: 42 },
}

/**
 * 动画状态机列表：`name` 与键盘推导的下一状态名对应；
 * `frames` 为 `REGIONS` 键序列；`loop` 为 false 时播完停在最后一帧（本场景未用非循环跑走）。
 */
export const ANIMS: { name: string; frames: string[]; loop: boolean; speed: number }[] = [
  { name: 'idleL', frames: ['72hcl'], loop: true, speed: 5 },
  { name: 'idledown', frames: ['rydce'], loop: true, speed: 5 },
  { name: 'idleup', frames: ['1et3y'], loop: true, speed: 5 },
  { name: 'runL', frames: ['bbcvv', 'foxtp', 'aw8dg', 'evrtr', 'pyoh8', 't4rff'], loop: true, speed: 5 },
  { name: 'rundown', frames: ['koy62', '3ygc0', 'yfrrb', '2enbr', 's2yql', 'idc64'], loop: true, speed: 5 },
  { name: 'runup', frames: ['8mwul', 'snwwj', 'ynglr', 'p3oo0', 'pfwvy', 'tvkvf'], loop: true, speed: 5 },
  { name: 'walkL', frames: ['3c66l', 'wq5ia', '11gwb', 'iitav', '360a7', 'ffd0g'], loop: true, speed: 5 },
  { name: 'walkdown', frames: ['ahlcx', '4i3vm', '0qwcd', 'y1030', '3sl87', '8kwsb'], loop: true, speed: 5 },
  { name: 'walkup', frames: ['umveo', 'v6ado', 'syfy0', 'us0w8', 'pf2m2', '876dv'], loop: true, speed: 5 },
]

/**
 * 从整张贴图裁出一帧到透明背景小 Canvas（尺寸 = 区域 wh）。
 */
export function extractFrame(img: HTMLImageElement, key: string): HTMLCanvasElement | null {
  const r = REGIONS[key]
  if (!r) return null
  const c = document.createElement('canvas')
  c.width = r.w
  c.height = r.h
  const ctx = c.getContext('2d')!
  ctx.drawImage(img, r.x, r.y, r.w, r.h, 0, 0, r.w, r.h)
  return c
}
