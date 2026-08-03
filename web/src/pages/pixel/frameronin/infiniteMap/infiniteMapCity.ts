/**
 * @fileoverview 无限地图「小镇」布景：参数化生成 `CityPropDef[]`，再经世界锚点平移。
 *
 * ## 坐标系（本地 → 世界）
 * - 布局在 **本地坐标** 生成：东西栏以 `ewFenceZCenter` 为 Z 基准，南北栏关于 X=0 对称。
 * - `offsetCityPropLayout(layout, placeX, placeZ, ewFenceZCenter)` 把本地 `(0, ewFenceZCenter)` 对齐到世界 `(placeX, placeZ)`，
 *   得到 `InfiniteMapScene` 使用的世界 `wx,wz`。
 *
 * ## 生成顺序（`buildCityPropLayout`）
 * 1. `grassPatches`：围栏内陆草地格点。
 * 2. `fenceRing`：南北竖栏 + 东西横栏（东西栏可走透视四边形绘制）。
 * 3. `randomScatterInFence`：箱/火/路灯等小物，数量随内陆面积相对 `CITY_REF_INTERIOR_AREA` 缩放。
 * 4. `decorAndBuildings`：雪人槽位 + 两栋主建筑，受 `decorJitter` / `buildingWxJitter` 等扰动。
 *
 * ## 放置合法性
 * `findNearestFlatSnowTownPlacement` 在角色附近环上搜索，使围栏脚印内格子均为 **可走平地**（`isBlobTileWalkable`）。
 */

import { isBlobTileWalkable, type BlobWorld } from './blobTerrain'

export const CITY_BASE = `${import.meta.env.BASE_URL}map/city/`

/** 矩形围场基准：东西侧半宽（与南北 9 栏、步长 34 闭合：(9−1)/2×34） */
export const CITY_RECT_FENCE_X_ABS = 136
/** 南北栏相邻段沿 X 的间距（固定；与 9 段时总宽 2×136 闭合） */
export const CITY_NS_FENCE_X_STEP = 34
/** 东西栏 Z 向中心（与 6 段、间距 31 闭合） */
export const CITY_RECT_EW_Z_CENTER = 290
export type CityPropDef = {
  file: string
  wx: number
  wz: number
  feetDownSrcPx: number
  /** 同屏深度时的逻辑层级（草地 < 围栏 < 建筑） */
  layer: number
  /** 透视下缩放参考深度，类似树 */
  refDz: number
  scaleMul?: number
  /** 仅东西侧栏：透视四边形 + InfiniteMapScene 中固定宽高间距与左右偏移 */
  fenceEwUseAxisSliders?: true
}

/** 东西栏相邻段沿 Z 的间距（固定；与原先 3 段时 center±31 一致） */
export const CITY_EW_FENCE_Z_STEP = 31
/** 3 段时外侧相对中心的距离 (= STEP)；透视段间距等仍按此几何定义 */
export const CITY_EW_FENCE_Z_HALF_SPAN = CITY_EW_FENCE_Z_STEP
/** 与矩形基准一致 */
export const CITY_EW_FENCE_Z_CENTER = CITY_RECT_EW_Z_CENTER
/** 东西栏三段默认 Z（本地）；段数变化时由 `ewFenceExtentZ` 推导 */
export const CITY_EW_FENCE_Z_SIDE = [
  CITY_EW_FENCE_Z_CENTER - CITY_EW_FENCE_Z_HALF_SPAN,
  CITY_EW_FENCE_Z_CENTER,
  CITY_EW_FENCE_Z_CENTER + CITY_EW_FENCE_Z_HALF_SPAN,
] as const

/** 参数化生成小镇：可自行改规则；与 buildCityPropLayout 搭配试验 */
export type CityGenParams = {
  /** 随机种子（草地/装饰抖动可复现） */
  seed: number
  /** 草地：横向半宽格数，实际 ix ∈ [-n..n] */
  grassHalfIx: number
  /** 草地：纵向行数 */
  grassRows: number
  /** 草地：列方向世界步长 */
  grassWxStep: number
  /** 草地：奇偶行横向错开（原 12） */
  grassRowStagger: number
  /** 草地：行距（原 20） */
  grassWzStep: number
  /** 草地：首行 Z（保留字段；草地排布已随围栏内陆自动起算，此项不再参与定位） */
  grassBaseWz: number
  /** 草地：世界单位抖动幅度（0=关） */
  grassJitter: number
  /** 东西栏：各段 Z 以该中心为基准（基准 290） */
  ewFenceZCenter: number
  /** 东西侧每边栏段数量：与 CITY_EW_FENCE_Z_STEP 共同决定城镇 Z 向范围（段数↑则南北向总长↑） */
  ewFenceSegmentCount: number
  /** 南北侧每边栏段数量：与 CITY_NS_FENCE_X_STEP 共同决定城镇东西向范围（段数↑则东西向总长↑） */
  nsFenceSegmentCount: number
  /** 围栏内随机点缀基准数量（0=关）；实际数量按围栏内陆面积相对默认城镇比例缩放 */
  cityScatterCount: number
  /** 雪人/小树等装饰：世界坐标最大抖动 */
  decorJitter: number
  /** 建筑（hight/house）整体 Z 平移 */
  buildingWZShift: number
  /** 建筑 wx 额外抖动（在 decorJitter 之外单独调） */
  buildingWxJitter: number
}

export const CITY_GEN_DEFAULTS: CityGenParams = {
  seed: 42,
  grassHalfIx: 2,
  grassRows: 6,
  grassWxStep: 50,
  grassRowStagger: 12,
  grassWzStep: 20,
  grassBaseWz: 222,
  grassJitter: 0,
  ewFenceZCenter: CITY_RECT_EW_Z_CENTER,
  ewFenceSegmentCount: 6,
  nsFenceSegmentCount: 9,
  cityScatterCount: 14,
  decorJitter: 0,
  buildingWZShift: 0,
  buildingWxJitter: 0,
}

// ---------- 随机（与草地/点缀抖动复现） ----------

function mulberry32(a: number) {
  return function () {
    let t = (a += 0x6d2b79f5)
    t = Math.imul(t ^ (t >>> 15), t | 1)
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

/** UI 滑条 `Partial` 与默认合并 */
function mergeParams(p: Partial<CityGenParams>): CityGenParams {
  return { ...CITY_GEN_DEFAULTS, ...p }
}

/** 东西栏最南/最北 Z（本地坐标，未加世界放置偏移） */
export function ewFenceExtentZ(g: CityGenParams): { zSouth: number; zNorth: number } {
  const c = g.ewFenceZCenter
  const ewZStep = CITY_EW_FENCE_Z_STEP
  const n = Math.max(1, Math.min(20, Math.floor(g.ewFenceSegmentCount)))
  if (n <= 1) return { zSouth: c, zNorth: c }
  const half = ((n - 1) / 2) * ewZStep
  return { zSouth: c - half, zNorth: c + half }
}

/** 南北栏最西/最东 X（本地坐标，关于 0 对称；与东西栏 Z 排布方式一致） */
export function nsFenceExtentX(g: CityGenParams): { xWest: number; xEast: number } {
  const xStep = CITY_NS_FENCE_X_STEP
  const n = Math.max(1, Math.min(24, Math.floor(g.nsFenceSegmentCount)))
  if (n <= 1) return { xWest: 0, xEast: 0 }
  const half = ((n - 1) / 2) * xStep
  return { xWest: -half, xEast: half }
}

/** 围栏矩形在本地坐标的轴对齐包围盒（略含边栏占地） */
export function computeTownFenceLocalAabb(g: CityGenParams): {
  minWx: number
  maxWx: number
  minWz: number
  maxWz: number
} {
  const { zSouth, zNorth } = ewFenceExtentZ(g)
  const { xWest, xEast } = nsFenceExtentX(g)
  return {
    minWx: Math.min(xWest, xEast),
    maxWx: Math.max(xWest, xEast),
    minWz: Math.min(zSouth, zNorth),
    maxWz: Math.max(zSouth, zNorth),
  }
}

/**
 * 将本地布景平移到世界：使本地点 (0, ewFenceZCenter) 落到世界 (placeX, placeZ)。
 */
export function offsetCityPropLayout(
  layout: CityPropDef[],
  placeX: number,
  placeZ: number,
  ewZCenter: number,
): CityPropDef[] {
  const dz = placeZ - ewZCenter
  return layout.map((p) => ({ ...p, wx: p.wx + placeX, wz: p.wz + dz }))
}

/** 围栏内陆地区域（本地坐标），用于草地/装饰/点缀 */
function interiorLocalRect(
  g: CityGenParams,
  padX: number,
  padZ: number,
): { xLo: number; xHi: number; zLo: number; zHi: number } | null {
  const box = computeTownFenceLocalAabb(g)
  const xLo = box.minWx + padX
  const xHi = box.maxWx - padX
  const zLo = box.minWz + padZ
  const zHi = box.maxWz - padZ
  if (xHi <= xLo || zHi <= zLo) return null
  return { xLo, xHi, zLo, zHi }
}

/** 默认参数下围栏内陆近似面积（用于点缀数量比例）；与 pad≈12 的内陆同一量级 */
/** 默认参数下围栏内陆面积近似，用于按比例缩放 `cityScatterCount` */
const CITY_REF_INTERIOR_AREA = 33_500

function grassPatches(g: CityGenParams): CityPropDef[] {
  const out: CityPropDef[] = []
  const rand = mulberry32(g.seed >>> 0)
  const inner = interiorLocalRect(g, 10, 10)
  if (!inner) return []
  const { xLo, xHi, zLo, zHi } = inner
  const wSpan = xHi - xLo
  const hSpan = zHi - zLo
  const colFactor = Math.max(0.35, Math.max(0, Math.floor(g.grassHalfIx)) / 2)
  const rowFactor = Math.max(0.35, Math.max(1, Math.floor(g.grassRows)) / 6)
  let cols = Math.round((wSpan / g.grassWxStep) * colFactor)
  let rowCap = Math.round((hSpan / g.grassWzStep) * rowFactor)
  cols = Math.max(2, Math.min(28, cols))
  rowCap = Math.max(2, Math.min(24, rowCap))
  const staggerCap = Math.min(g.grassRowStagger, wSpan * 0.12)
  for (let iz = 0; iz < rowCap; iz++) {
    const wzRow = zLo + iz * g.grassWzStep
    if (wzRow > zHi) break
    for (let ci = 0; ci < cols; ci++) {
      const t = cols <= 1 ? 0.5 : ci / (cols - 1)
      let wx = xLo + t * (xHi - xLo)
      wx += (iz % 2) * staggerCap
      const jx = (rand() * 2 - 1) * g.grassJitter
      const jz = (rand() * 2 - 1) * g.grassJitter
      wx = Math.max(xLo, Math.min(xHi, wx + jx))
      const wz = Math.max(zLo, Math.min(zHi, wzRow + jz))
      out.push({
        file: 'ggrass1.png',
        wx,
        wz,
        feetDownSrcPx: 8,
        layer: 0,
        refDz: 128,
        scaleMul: 0.88,
      })
    }
  }
  return out
}

/** 围栏内随机点缀图（public/map/city）；在矩形内随机位置与缩放 */
export const CITY_SCATTER_FILES = [
  'box1.png',
  'box2.png',
  'fire1.png',
  'roadlight1.png',
  'wood1.png',
  'house2.png',
] as const

function scatterFeetForFile(file: string): number {
  if (file.startsWith('house')) return 12
  if (file.includes('roadlight')) return 10
  if (file.includes('fire')) return 8
  return 6
}

function scatterRefDzForFile(file: string): number {
  if (file.startsWith('house')) return 100
  if (file.includes('roadlight')) return 94
  return 90
}

/** 围栏矩形内部（与栏线留出边距）分层格内随机，数量随内陆面积缩放 */
function randomScatterInFence(g: CityGenParams): CityPropDef[] {
  const inner = interiorLocalRect(g, 12, 12)
  if (!inner) return []
  const { xLo: ix0, xHi: ix1, zLo: iz0, zHi: iz1 } = inner
  const area = (ix1 - ix0) * (iz1 - iz0)
  let n = Math.round(g.cityScatterCount * (area / CITY_REF_INTERIOR_AREA))
  n = Math.max(0, Math.min(64, n))
  if (n === 0) return []
  const rand = mulberry32((g.seed + 0x5c477) >>> 0)
  const w = ix1 - ix0
  const h = iz1 - iz0
  const ar = w / Math.max(1e-6, h)
  let gridCols = Math.max(1, Math.ceil(Math.sqrt(n * ar)))
  let gridRows = Math.max(1, Math.ceil(n / gridCols))
  while (gridCols * gridRows < n) {
    if (w >= h) gridCols++
    else gridRows++
  }
  const cells: { c: number; r: number }[] = []
  for (let r = 0; r < gridRows; r++) {
    for (let c = 0; c < gridCols; c++) cells.push({ c, r })
  }
  for (let i = cells.length - 1; i > 0; i--) {
    const j = Math.floor(rand() * (i + 1))
    const t = cells[i]!
    cells[i] = cells[j]!
    cells[j] = t
  }
  const out: CityPropDef[] = []
  for (let i = 0; i < n; i++) {
    const { c, r } = cells[i]!
    const file = CITY_SCATTER_FILES[Math.floor(rand() * CITY_SCATTER_FILES.length)]!
    const u0 = c / gridCols
    const u1 = (c + 1) / gridCols
    const v0 = r / gridRows
    const v1 = (r + 1) / gridRows
    const wx = ix0 + (u0 + rand() * (u1 - u0)) * w
    const wz = iz0 + (v0 + rand() * (v1 - v0)) * h
    const layer = file.startsWith('house') ? 3 : 2
    out.push({
      file,
      wx,
      wz,
      feetDownSrcPx: scatterFeetForFile(file),
      layer,
      refDz: scatterRefDzForFile(file),
      scaleMul: 0.82 + rand() * 0.28,
    })
  }
  return out
}

function fenceRing(g: CityGenParams): CityPropDef[] {
  const out: CityPropDef[] = []
  const { xWest, xEast } = nsFenceExtentX(g)
  const x0 = Math.min(xWest, xEast)
  const x1 = Math.max(xWest, xEast)
  const { zSouth, zNorth } = ewFenceExtentZ(g)
  const z0 = zSouth
  const z1 = zNorth
  const c = g.ewFenceZCenter
  const ewZStep = CITY_EW_FENCE_Z_STEP
  const ewN = Math.max(1, Math.min(20, Math.floor(g.ewFenceSegmentCount)))
  const zEw: number[] =
    ewN <= 1 ? [c] : Array.from({ length: ewN }, (_, i) => c + (i - (ewN - 1) / 2) * ewZStep)

  const nsN = Math.max(1, Math.min(24, Math.floor(g.nsFenceSegmentCount)))
  const xStep = CITY_NS_FENCE_X_STEP
  const xNs: number[] =
    nsN <= 1 ? [0] : Array.from({ length: nsN }, (_, i) => (i - (nsN - 1) / 2) * xStep)

  const fence = (wx: number, wz: number, ewSliders?: true, layer = 1): CityPropDef => ({
    file: 'fence1.png',
    wx,
    wz,
    feetDownSrcPx: 4,
    layer,
    refDz: 92,
    scaleMul: 0.95,
    fenceEwUseAxisSliders: ewSliders,
  })

  for (const x of xNs) {
    out.push(fence(x, z0))
    out.push(fence(x, z1))
  }
  for (const z of zEw) {
    out.push(fence(x0, z, true))
    out.push(fence(x1, z, true))
  }
  return out
}

function decorAndBuildings(g: CityGenParams): CityPropDef[] {
  const rand = mulberry32((g.seed + 1337) >>> 0)
  const inner = interiorLocalRect(g, 18, 16)
  if (!inner) return []
  const { xLo, xHi, zLo, zHi } = inner
  const area = (xHi - xLo) * (zHi - zLo)
  const dj = (() => (rand() * 2 - 1) * g.decorJitter) as () => number
  const bjx = (rand() * 2 - 1) * g.buildingWxJitter
  const bz = g.buildingWZShift
  const px = (u: number) => xLo + u * (xHi - xLo)
  const pz = (v: number) => zLo + v * (zHi - zLo)
  const snowSlots: { u: number; v: number }[] = [
    { u: 0.22, v: 0.22 },
    { u: 0.78, v: 0.28 },
    { u: 0.5, v: 0.48 },
    { u: 0.34, v: 0.66 },
    { u: 0.66, v: 0.7 },
    { u: 0.5, v: 0.3 },
    { u: 0.18, v: 0.52 },
    { u: 0.82, v: 0.55 },
  ]
  const extraSnow = Math.min(4, Math.max(0, Math.floor(area / 26_000) - 1))
  const out: CityPropDef[] = []
  const snowCount = Math.min(snowSlots.length, 3 + extraSnow)
  for (let i = 0; i < snowCount; i++) {
    const { u, v } = snowSlots[i]!
    out.push({
      file: 'snowman1.png',
      wx: px(u) + dj(),
      wz: pz(v) + dj(),
      feetDownSrcPx: 6,
      layer: 2,
      refDz: 88,
      scaleMul: 1,
    })
  }
  out.push(
    {
      file: 'hight1.png',
      wx: px(0.2) + dj() + bjx,
      wz: pz(0.45) + dj() + bz,
      feetDownSrcPx: 10,
      layer: 3,
      refDz: 96,
      scaleMul: 1.02,
    },
    {
      file: 'house1.png',
      wx: px(0.68) + dj() + bjx,
      wz: pz(0.52) + dj() + bz,
      feetDownSrcPx: 12,
      layer: 4,
      refDz: 100,
      scaleMul: 1,
    },
  )
  return out
}

/**
 * 由参数生成小镇布景表；可与 UI 滑条联动做随机/规则试验。
 * `getCityPropLayout()` 等价于默认参数。
 */
export function buildCityPropLayout(p: Partial<CityGenParams> = {}): CityPropDef[] {
  const g = mergeParams(p)
  return [...grassPatches(g), ...fenceRing(g), ...randomScatterInFence(g), ...decorAndBuildings(g)]
}

/** @deprecated 使用 buildCityPropLayout()；保留兼容 */
export function getCityPropLayout(): CityPropDef[] {
  return buildCityPropLayout({})
}

/** 树木等：是否在已放置小镇脚印内（本地围栏盒 + 边距，再换算世界） */
export function isInsidePlacedTownFootprint(
  wx: number,
  wz: number,
  g: CityGenParams,
  placeX: number,
  placeZ: number,
  pad = 24,
): boolean {
  const box = computeTownFenceLocalAabb(g)
  const lx = wx - placeX
  const lz = wz - (placeZ - g.ewFenceZCenter)
  return (
    lx >= box.minWx - pad &&
    lx <= box.maxWx + pad &&
    lz >= box.minWz - pad &&
    lz <= box.maxWz + pad
  )
}

/**
 * 小镇锚点 (placeX, placeZ)：本地 (0, ewFenceZCenter) 对齐到世界。
 * 检查围栏矩形脚印（略外扩）覆盖的每个地形格均为可走平地（非水、非山）。
 */
export function isTownAnchorWalkable(
  world: BlobWorld,
  g: CityGenParams,
  tileWorld: number,
  placeX: number,
  placeZ: number,
): boolean {
  const box = computeTownFenceLocalAabb(g)
  const padW = tileWorld * 1.5
  const zS = box.minWz
  const zN = box.maxWz
  const wx0 = box.minWx + placeX - padW
  const wx1 = box.maxWx + placeX + padW
  const wz0 = zS + (placeZ - g.ewFenceZCenter) - padW
  const wz1 = zN + (placeZ - g.ewFenceZCenter) + padW
  const tix0 = Math.floor(wx0 / tileWorld)
  const tix1 = Math.ceil(wx1 / tileWorld) - 1
  const tiz0 = Math.floor(wz0 / tileWorld)
  const tiz1 = Math.ceil(wz1 / tileWorld) - 1
  for (let tz = tiz0; tz <= tiz1; tz++) {
    for (let tx = tix0; tx <= tix1; tx++) {
      if (!isBlobTileWalkable(world, tx, tz)) return false
    }
  }
  return true
}

/**
 * 从人物附近由近到远（Chebyshev 环）找第一个合法锚点；同一环内顺序可由 ringShuffleSeed 打乱（随机小镇时换位置）。
 */
export function findNearestFlatSnowTownPlacement(
  world: BlobWorld,
  g: CityGenParams,
  tileWorld: number,
  originWx: number,
  originWz: number,
  maxRadiusTiles = 140,
  ringShuffleSeed?: number,
): { placeX: number; placeZ: number } | null {
  const rand = ringShuffleSeed !== undefined ? mulberry32(ringShuffleSeed >>> 0) : null
  const otx = Math.floor(originWx / tileWorld)
  const otz = Math.floor(originWz / tileWorld)

  for (let r = 0; r <= maxRadiusTiles; r++) {
    const ring: { dx: number; dz: number }[] = []
    for (let dz = -r; dz <= r; dz++) {
      for (let dx = -r; dx <= r; dx++) {
        if (Math.max(Math.abs(dx), Math.abs(dz)) !== r) continue
        ring.push({ dx, dz })
      }
    }
    if (rand && ring.length > 1) {
      for (let i = ring.length - 1; i > 0; i--) {
        const j = Math.floor(rand() * (i + 1))
        const t = ring[i]!
        ring[i] = ring[j]!
        ring[j] = t
      }
    }
    for (const { dx, dz } of ring) {
      const tx = otx + dx
      const tz = otz + dz
      const placeX = (tx + 0.5) * tileWorld
      const placeZ = (tz + 0.5) * tileWorld
      if (isTownAnchorWalkable(world, g, tileWorld, placeX, placeZ)) {
        return { placeX, placeZ }
      }
    }
  }
  return null
}

const CITY_FILES = [
  'fence1.png',
  'ggrass1.png',
  'hight1.png',
  'house1.png',
  'snowman1.png',
  ...CITY_SCATTER_FILES,
]

/**
 * 加载 `CITY_FILES` 中全部 PNG，按 **文件名** 存入 Map（与 `CityPropDef.file` 对应）。
 * @returns 取消加载的清理函数。
 */
export function loadCityPropImages(onDone: (map: Map<string, HTMLImageElement>) => void): () => void {
  let cancelled = false
  const map = new Map<string, HTMLImageElement>()
  let remaining = CITY_FILES.length
  if (remaining === 0) {
    onDone(map)
    return () => {}
  }
  const tryDone = () => {
    if (cancelled || remaining > 0) return
    onDone(map)
  }
  const cleanups: (() => void)[] = []
  for (const name of CITY_FILES) {
    const img = new Image()
    const done = () => {
      if (img.naturalWidth > 0) map.set(name, img)
      remaining--
      tryDone()
    }
    img.onload = done
    img.onerror = done
    img.src = CITY_BASE + name
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
