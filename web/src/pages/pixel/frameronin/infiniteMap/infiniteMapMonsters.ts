/**
 * @fileoverview 无限地图野怪：贴图加载、群体生成与简单游荡 AI。
 *
 * ## 资源
 * `map/monster/*.png` 列表由 Vite 插件 `virtual:infinite-map-monster-files` 注入为 `MONSTER_PUBLIC_FILENAMES`，
 * 避免手写文件名列表与目录不同步。
 *
 * ## 精灵布局
 * 每张图 **4 列 × 8 行**：列 = 走路帧循环；行 = **八方向**（与 `velocityToMonsterRow` 一致）。
 *
 * ## 运动
 * - 八方向匀速；定时 `repathIn` 到期后随机换新方向。
 * - 撞水或出陆地时速度取反并缩短下次转向间隔。
 */
import { isBlobTileLandNotWater, type BlobWorld } from './blobTerrain'
import { MONSTER_PUBLIC_FILENAMES } from 'virtual:infinite-map-monster-files'

/** 单张怪物表横向帧数（走路循环） */
export const MONSTER_COLS = 4
/** 单张怪物表纵向行数（8 向） */
export const MONSTER_ROWS = 8

/** 单只野怪实例状态（世界坐标 + 动画） */
export type MonsterInst = {
  wx: number
  wz: number
  vx: number
  vz: number
  /** MONSTER_PUBLIC_FILENAMES 下标 */
  sheetIndex: number
  frame: number
  /** 秒，用于走路帧 */
  animT: number
  /** 秒，到时换方向 */
  repathIn: number
}

const EIGHT_DIRS: [number, number][] = [
  [0, -1],
  [1, -1],
  [1, 0],
  [1, 1],
  [0, 1],
  [-1, 1],
  [-1, 0],
  [-1, -1],
]

/** 与 velocityToMonsterRow 一致：各行动画在屏幕上的朝向中心角（弧度） */
const MONSTER_ROW_ANGLE_CENTERS = [
  Math.PI, // 0 下
  (3 * Math.PI) / 4, // 1 右下
  Math.PI / 2, // 2 右
  Math.PI / 4, // 3 右上
  0, // 4 上
  -Math.PI / 4, // 5 左上
  -Math.PI / 2, // 6 左
  (-3 * Math.PI) / 4, // 7 左下
] as const

/**
 * 行从 0 起：0下、1右下、2右、3右上、4上、5左上、6左、7左下（与素材第1–8行一致）
 *
 * 透视里 world +X 对应屏幕左侧，不能直接用 (vx,vz) 与数学八象限点积，否则右上会错成左上。
 * 用 atan2(-vx,-vz)：0 为朝屏幕上、π/2 为朝屏幕右，与上表一致。
 */
export function velocityToMonsterRow(vx: number, vz: number): number {
  const s = Math.hypot(vx, vz)
  if (s < 0.08) return 0
  const ang = Math.atan2(-vx / s, -vz / s)
  let best = 0
  let bestD = Infinity
  for (let i = 0; i < 8; i++) {
    const c = MONSTER_ROW_ANGLE_CENTERS[i]!
    let d = Math.abs(ang - c)
    if (d > Math.PI) d = 2 * Math.PI - d
    if (d < bestD) {
      bestD = d
      best = i
    }
  }
  return best
}

function randomEightDirSpeed(): { vx: number; vz: number } {
  const [dx, dz] = EIGHT_DIRS[Math.floor(Math.random() * 8)]!
  const sp = 1.4 + Math.random() * 1.8
  return { vx: dx * sp, vz: dz * sp }
}

function monsterUrls(): string[] {
  const base = `${import.meta.env.BASE_URL}map/monster/`
  return [...MONSTER_PUBLIC_FILENAMES].map((n) => base + n)
}

/**
 * 并行加载全部怪物 PNG；完成时回调「有效图片」数组（加载失败槽位被 filter 掉）。
 * @returns 取消函数：卸载 onload、清空 src，避免泄漏。
 */
export function loadMonsterImages(onDone: (imgs: HTMLImageElement[]) => void): () => void {
  let cancelled = false
  const urls = monsterUrls()
  const imgs: HTMLImageElement[] = new Array(urls.length)
  let remaining = urls.length
  if (remaining === 0) {
    onDone([])
    return () => {}
  }
  const cleanups: (() => void)[] = []
  const tryFinish = () => {
    if (cancelled || remaining > 0) return
    onDone(imgs.filter((x) => x && x.naturalWidth > 0))
  }
  for (let i = 0; i < urls.length; i++) {
    const img = new Image()
    const idx = i
    const doneOne = () => {
      imgs[idx] = img
      remaining--
      tryFinish()
    }
    img.onload = doneOne
    img.onerror = doneOne
    img.src = urls[i]!
    cleanups.push(() => {
      img.onload = null
      img.onerror = null
      img.src = ''
    })
  }
  return () => {
    cancelled = true
    cleanups.forEach((f) => f())
  }
}

/**
 * 在陆地（平地+山地）上随机撒点；中心取角色附近世界格。
 */
export function createMonsterSwarm(
  count: number,
  world: BlobWorld,
  centerWx: number,
  centerWz: number,
  sheetCount: number,
  tileWorld: number,
): MonsterInst[] {
  if (count <= 0 || sheetCount <= 0) return []
  const out: MonsterInst[] = []
  const tcx = Math.floor(centerWx / tileWorld)
  const tcz = Math.floor(centerWz / tileWorld)
  let attempts = 0
  const maxAttempts = Math.max(count * 100, 400)
  while (out.length < count && attempts < maxAttempts) {
    attempts++
    const tix = tcx + Math.floor((Math.random() - 0.5) * 100)
    const tiz = tcz + Math.floor((Math.random() - 0.5) * 100)
    if (!isBlobTileLandNotWater(world, tix, tiz)) continue
    const { vx, vz } = randomEightDirSpeed()
    out.push({
      wx: (tix + 0.5) * tileWorld + (Math.random() - 0.5) * (tileWorld * 0.5),
      wz: (tiz + 0.5) * tileWorld + (Math.random() - 0.5) * (tileWorld * 0.5),
      vx,
      vz,
      sheetIndex: Math.floor(Math.random() * sheetCount),
      frame: Math.floor(Math.random() * 4),
      animT: Math.random() * 3,
      repathIn: 1.5 + Math.random() * 4,
    })
  }
  return out
}

/**
 * 积分位置、更新走路帧、定时换向；陆地碰撞时反弹。
 */
export function stepMonster(
  m: MonsterInst,
  world: BlobWorld,
  dt: number,
  tileWorld: number,
): void {
  m.animT += dt
  m.repathIn -= dt
  if (m.repathIn <= 0) {
    m.repathIn = 2 + Math.random() * 5
    const { vx, vz } = randomEightDirSpeed()
    m.vx = vx
    m.vz = vz
  }
  const frameDur = 0.125
  if (m.animT >= frameDur) {
    const adv = Math.floor(m.animT / frameDur)
    m.animT -= adv * frameDur
    m.frame = (m.frame + adv) % MONSTER_COLS
  }

  const nx = m.wx + m.vx * dt
  const nz = m.wz + m.vz * dt
  const tix = Math.floor(nx / tileWorld)
  const tiz = Math.floor(nz / tileWorld)
  if (!isBlobTileLandNotWater(world, tix, tiz)) {
    m.vx *= -1
    m.vz *= -1
    m.repathIn = 0.4 + Math.random() * 0.6
    return
  }
  m.wx = nx
  m.wz = nz
}
