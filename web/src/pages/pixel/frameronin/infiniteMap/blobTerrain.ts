/**
 * @fileoverview RoninTileMap「Blob」程序化地形核心。
 *
 * ## 与 map.html 的关系
 * 逻辑与 `public/map/blob/map.html` 一致：同一套 Perlin/FBM 噪声、海平面与山地阈值、
 * 河谷走廊判水、以及 8 邻域掩码 → 图集子块索引（`MASK_TO_INDEX`）。
 *
 * ## 坐标系
 * - **世界整数格** `(tx, ty)`：与 map.html 的 `(wx, wy)` 同义；**北**为 `ty - 1`（上邻）。
 * - **BlobWorld** 按 **32×32 世界格** 分块缓存（`BLOB_CHUNK_SIZE`），按需生成、LRU 式淘汰。
 *
 * ## 数据流（简）
 * 1. `BlobWorld.landWorld` / `biomeWorld`：某格是否水、平地(IMG_NORM)或山(IMG_MTN)。
 * 2. `getMaskWater` / `getMaskLandBiome`：八邻同类 → 0–255 掩码。
 * 3. `nearestBlobTileIndex`：掩码 → `frame_004`/`007` 等图集中 **sheet 下标**（0-based）。
 * 4. `sampleBlobAtlas`：世界坐标在地块内 UV → 像素 RGB（供 Canvas 光栅化）。
 *
 * ## 可走与装饰
 * - `isBlobTileWalkable`：仅 **平地**（非水非山），角色/小镇锚点用。
 * - `isBlobTileLandNotWater`：含山地，树木等「陆上」装饰用。
 */

/** 从 PNG 解码后的图集：RGBA 交错、`cellW×cellH` 为单个子块尺寸（3 列 × 24 行布局） */
export type BlobAtlas = {
  data: Uint8ClampedArray
  stride: number
  cellW: number
  cellH: number
}

export const BLOB_TILE_COLS = 3
export const BLOB_TILE_ROWS = 24
/** 一块 Chunk 覆盖的世界格边长；与缓存键 `(cx,cy)` 对应 */
export const BLOB_CHUNK_SIZE = 32
/** 内存中最多保留的 Chunk 数；超出则从最早入队者淘汰 */
const MAX_CACHED_CHUNKS = 128

/** `biomeWorld` 取值：山地 → 采样 frame_001 类图集 */
const IMG_MTN = 0
/** 平地 → 采样 frame_004（及 X1/X2 变种） */
const IMG_NORM = 1

/**
 * 八邻掩码（0–255）→ 图集 **sheet 下标**。
 * 与 map.html / `frame_000` 示意一致；`001` / `004` / `007` 等共用同一邻接编码。
 */
const MASK_TO_INDEX: Record<number, number> = {
  0: 13,
  208: 0,
  248: 1,
  104: 2,
  214: 3,
  255: 4,
  107: 5,
  22: 6,
  31: 7,
  11: 8,
  80: 9,
  24: 10,
  72: 11,
  66: 12,
  18: 15,
  10: 17,
  64: 19,
  16: 21,
  90: 22,
  8: 23,
  2: 25,
  88: 28,
  82: 30,
  74: 32,
  26: 34,
  95: 37,
  123: 39,
  222: 41,
  250: 43,
  127: 45,
  223: 46,
  251: 48,
  254: 49,
  86: 51,
  75: 52,
  210: 54,
  106: 55,
  120: 57,
  216: 58,
  27: 60,
  30: 61,
  218: 63,
  122: 64,
  94: 66,
  91: 67,
  126: 69,
  219: 70,
}

const FORBIDDEN_TILE_INDEX = 71
const FALLBACK_TILE_INDEX = 4
const MASK_KEYS = Object.keys(MASK_TO_INDEX).map((k) => Number(k))

// ---------- 噪声与工具（内部） ----------

/** FNV-1a 风格：任意字符串 → 32 位种子，供 Perlin 初始化 */
function stringSeedToUint32(s: string): number {
  let h = 2166136261
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i)!
    h = Math.imul(h, 16777619)
  }
  return h >>> 0
}

/** Mulberry32：小体积、可复现的 [0,1) 均匀 PRNG */
function mulberry32(seed: number): () => number {
  let a = seed >>> 0
  return () => {
    a |= 0
    a = (a + 0x6d2b79f5) | 0
    let t = Math.imul(a ^ (a >>> 15), a | 1)
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

function fade(t: number): number {
  return t * t * t * (t * (t * 6 - 15) + 10)
}

function lerp(a: number, b: number, t: number): number {
  return a + t * (b - a)
}

function grad2(hash: number, x: number, y: number): number {
  const h = hash & 3
  const u = h < 2 ? x : y
  const v = h < 2 ? y : x
  return ((h & 1) === 0 ? u : -u) + ((h & 2) === 0 ? v : -v)
}

function createPerlin2D(seed: number): (x: number, y: number) => number {
  const rnd = mulberry32(seed ^ 0x9e3779b9)
  const perm = new Uint8Array(512)
  const pInit = new Uint8Array(256)
  for (let i = 0; i < 256; i++) pInit[i] = i
  for (let i = 255; i > 0; i--) {
    const j = Math.floor(rnd() * (i + 1))
    const tmp = pInit[i]!
    pInit[i] = pInit[j]!
    pInit[j] = tmp
  }
  for (let i = 0; i < 256; i++) {
    perm[i] = pInit[i]!
    perm[i + 256] = pInit[i]!
  }
  return (x: number, y: number) => {
    const X = Math.floor(x) & 255
    const Y = Math.floor(y) & 255
    const xf = x - Math.floor(x)
    const yf = y - Math.floor(y)
    const u = fade(xf)
    const v = fade(yf)
    const aa = perm[X + perm[Y]!]!
    const ab = perm[X + perm[Y + 1]!]!
    const ba = perm[X + 1 + perm[Y]!]!
    const bb = perm[X + 1 + perm[Y + 1]!]!
    const x1 = lerp(grad2(aa, xf, yf), grad2(ba, xf - 1, yf), u)
    const x2 = lerp(grad2(ab, xf, yf - 1), grad2(bb, xf - 1, yf - 1), u)
    return lerp(x1, x2, v)
  }
}

function fbm2D(
  noise: (x: number, y: number) => number,
  x: number,
  y: number,
  octaves: number,
  persistence: number,
  lacunarity: number,
): number {
  let sum = 0
  let amp = 1
  let freq = 1
  let norm = 0
  for (let i = 0; i < octaves; i++) {
    sum += amp * noise(x * freq, y * freq)
    norm += amp
    amp *= persistence
    freq *= lacunarity
  }
  return sum / norm
}

function clamp01(v: number): number {
  return Math.min(1, Math.max(0, v))
}

/** 8 位中 1 的个数；用于掩码汉明距离 */
function pop8(n: number): number {
  n &= 255
  let c = 0
  while (n) {
    c += n & 1
    n >>>= 1
  }
  return c
}

/**
 * 将 8 邻域掩码映射到合法图集下标；无精确匹配时取汉明距离最小的键。
 * @param mask `computeBlobMask` 返回值（已 `& 255`）
 */
export function nearestBlobTileIndex(mask: number): number {
  mask &= 255
  const direct = MASK_TO_INDEX[mask]
  if (direct !== undefined && direct !== FORBIDDEN_TILE_INDEX) return direct

  let bestKey = MASK_KEYS[0]!
  let bestD = 99
  for (let i = 0; i < MASK_KEYS.length; i++) {
    const k = MASK_KEYS[i]!
    const v = MASK_TO_INDEX[k]
    if (v === FORBIDDEN_TILE_INDEX) continue
    const d = pop8(mask ^ k)
    if (d < bestD) {
      bestD = d
      bestKey = k
    }
  }
  const out = MASK_TO_INDEX[bestKey]
  if (out === undefined || out === FORBIDDEN_TILE_INDEX) return FALLBACK_TILE_INDEX
  return out
}

/**
 * 八邻 Blob 掩码；pred(tx,ty) 为「该格与当前类属同类」。
 */
export function computeBlobMask(tx: number, ty: number, pred: (x: number, y: number) => boolean): number {
  const has = (dx: number, dy: number) => pred(tx + dx, ty + dy)

  let mask = 0
  const n = has(0, -1)
  const s = has(0, 1)
  const w = has(-1, 0)
  const e = has(1, 0)

  if (n) mask += 2
  if (s) mask += 64
  if (w) mask += 8
  if (e) mask += 16

  if (n && w && has(-1, -1)) mask += 1
  if (n && e && has(1, -1)) mask += 4
  if (s && w && has(-1, 1)) mask += 32
  if (s && e && has(1, 1)) mask += 128

  return mask
}

type ChunkData = {
  /** 1=陆 0=水（海/湖/河） */
  land: Uint8Array
  /** 陆上：IMG_MTN / IMG_NORM */
  biome: Uint8Array
}

/**
 * 宏观地形参数（世界连续坐标 wx,wy 输入 FBM）：
 * - **陆高**：大尺度 → 连片湖/陆；`seaLevel` 截断为水。
 * - **山场**：较慢噪声 → 成片山地；`mtnTh` 与平地分界。
 * - **河谷**：窄走廊 + 高度带 → 曲流河（仍为「格级」水）。
 * 海平面 / 山地阈值由 `BlobWorld.setParams`（UI 滑条）实时调整。
 */
const LAND_SCALE = 26
const LAND_OCTAVES = 3
const MOUNTAIN_SCALE = 42
const MOUNTAIN_OCTAVES = 3
const RIVER_SCALE = 12.5
const RIVER_OCTAVES = 2
const RIVER_ABS_THRESH = 0.1
const RIVER_HEIGHT_BAND = 0.16

/**
 * 无限延伸的程序化地形：按 Chunk 懒生成并缓存。
 * - 换种子：`reseed` 会重建三套 Perlin 场并 `clearCache`。
 * - 换参数：仅 `seaLevel`/`mtnTh` 变化时也应 `clearCache`（由调用方负责），否则旧 Chunk 与新区划不一致。
 */
export class BlobWorld {
  private readonly chunkCache = new Map<string, ChunkData>()
  private readonly chunkQueue: string[] = []
  /** 陆高噪声（归一化后截水） */
  private nHeight: (x: number, y: number) => number
  /** 山地权重噪声 */
  private nS4: (x: number, y: number) => number
  /** 河谷走廊噪声 */
  private nRiver: (x: number, y: number) => number

  /** 0–1：低于此高度的陆格视为水（湖/海） */
  seaLevel = 0.42
  /** 0–1：陆上高于此噪声值视为山地 */
  mtnTh = 0.48

  constructor(seedStr: string) {
    const base = stringSeedToUint32(seedStr.trim() || 'default')
    this.nHeight = createPerlin2D(base)
    this.nS4 = createPerlin2D(base ^ 0x414004)
    this.nRiver = createPerlin2D(base ^ 0x927b51c1)
  }

  /** 清空所有 Chunk；地形参数或种子变更后须调用 */
  clearCache(): void {
    this.chunkCache.clear()
    this.chunkQueue.length = 0
  }

  /** 更新海平面与山地阈值（不自动清缓存；InfiniteMapScene 中与 clearCache 配对使用） */
  setParams(seaLevel: number, mtnTh: number): void {
    this.seaLevel = seaLevel
    this.mtnTh = mtnTh
  }

  /** 生成单块 Chunk：`(cx,cy)` 为 Chunk 索引，内含 `BLOB_CHUNK_SIZE²` 个世界格 */
  private buildChunk(cx: number, cy: number): ChunkData {
    const land = new Uint8Array(BLOB_CHUNK_SIZE * BLOB_CHUNK_SIZE)
    const biome = new Uint8Array(BLOB_CHUNK_SIZE * BLOB_CHUNK_SIZE)

    for (let ly = 0; ly < BLOB_CHUNK_SIZE; ly++) {
      for (let lx = 0; lx < BLOB_CHUNK_SIZE; lx++) {
        const wx = cx * BLOB_CHUNK_SIZE + lx
        const wy = cy * BLOB_CHUNK_SIZE + ly
        const i = ly * BLOB_CHUNK_SIZE + lx

        const raw = fbm2D(this.nHeight, wx / LAND_SCALE, wy / LAND_SCALE, LAND_OCTAVES, 0.52, 2.05)
        const h01 = clamp01((raw + 1) * 0.5)

        const rv = fbm2D(
          this.nRiver,
          wx / RIVER_SCALE + 31.4,
          wy / RIVER_SCALE + 12.8,
          RIVER_OCTAVES,
          0.48,
          2.02,
        )
        const riverCorridor = Math.abs(rv) < RIVER_ABS_THRESH
        const lowEnoughForRiver = h01 < this.seaLevel + RIVER_HEIGHT_BAND
        const isLakeOrSea = h01 <= this.seaLevel
        const isRiver = riverCorridor && lowEnoughForRiver && !isLakeOrSea
        const isWater = isLakeOrSea || isRiver
        land[i] = isWater ? 0 : 1

        if (isWater) {
          biome[i] = 0
          continue
        }

        const u =
          (fbm2D(this.nS4, wx / MOUNTAIN_SCALE + 2.7, wy / MOUNTAIN_SCALE + 1.1, MOUNTAIN_OCTAVES, 0.5, 2.02) + 1) *
          0.5
        biome[i] = u > this.mtnTh ? IMG_MTN : IMG_NORM
      }
    }
    return { land, biome }
  }

  /** 取或创建 Chunk，并维护 FIFO 淘汰队列 */
  ensureChunk(cx: number, cy: number): ChunkData {
    const key = `${cx},${cy}`
    let ch = this.chunkCache.get(key)
    if (ch) return ch
    ch = this.buildChunk(cx, cy)
    this.chunkCache.set(key, ch)
    this.chunkQueue.push(key)
    while (this.chunkQueue.length > MAX_CACHED_CHUNKS) {
      const old = this.chunkQueue.shift()
      if (old) this.chunkCache.delete(old)
    }
    return ch
  }

  /** @returns 1 陆 / 0 水 */
  landWorld(wx: number, wy: number): number {
    const cx = Math.floor(wx / BLOB_CHUNK_SIZE)
    const cy = Math.floor(wy / BLOB_CHUNK_SIZE)
    const ch = this.ensureChunk(cx, cy)
    const lx = ((wx % BLOB_CHUNK_SIZE) + BLOB_CHUNK_SIZE) % BLOB_CHUNK_SIZE
    const ly = ((wy % BLOB_CHUNK_SIZE) + BLOB_CHUNK_SIZE) % BLOB_CHUNK_SIZE
    return ch.land[ly * BLOB_CHUNK_SIZE + lx]!
  }

  /** 仅当 `landWorld===1` 有意义：`IMG_MTN` 或 `IMG_NORM` */
  biomeWorld(wx: number, wy: number): number {
    const cx = Math.floor(wx / BLOB_CHUNK_SIZE)
    const cy = Math.floor(wy / BLOB_CHUNK_SIZE)
    const ch = this.ensureChunk(cx, cy)
    const lx = ((wx % BLOB_CHUNK_SIZE) + BLOB_CHUNK_SIZE) % BLOB_CHUNK_SIZE
    const ly = ((wy % BLOB_CHUNK_SIZE) + BLOB_CHUNK_SIZE) % BLOB_CHUNK_SIZE
    return ch.biome[ly * BLOB_CHUNK_SIZE + lx]!
  }

  preloadAroundTile(tx: number, ty: number, radiusTiles: number): void {
    const pad = BLOB_CHUNK_SIZE * 2
    const x0 = tx - radiusTiles - pad
    const x1 = tx + radiusTiles + pad
    const y0 = ty - radiusTiles - pad
    const y1 = ty + radiusTiles + pad
    const cx0 = Math.floor(x0 / BLOB_CHUNK_SIZE)
    const cx1 = Math.floor(x1 / BLOB_CHUNK_SIZE)
    const cy0 = Math.floor(y0 / BLOB_CHUNK_SIZE)
    const cy1 = Math.floor(y1 / BLOB_CHUNK_SIZE)
    for (let ccy = cy0; ccy <= cy1; ccy++) {
      for (let ccx = cx0; ccx <= cx1; ccx++) {
        this.ensureChunk(ccx, ccy)
      }
    }
  }

  /** 仅预加载视野内地块包络（世界格 tx/tz 与 map.html wy 一致） */
  preloadTileAABB(tix0: number, tix1: number, tiz0: number, tiz1: number): void {
    const loX = Math.min(tix0, tix1)
    const hiX = Math.max(tix0, tix1)
    const loZ = Math.min(tiz0, tiz1)
    const hiZ = Math.max(tiz0, tiz1)
    const pad = BLOB_CHUNK_SIZE * 2
    const cx0 = Math.floor((loX - pad) / BLOB_CHUNK_SIZE)
    const cx1 = Math.floor((hiX + pad) / BLOB_CHUNK_SIZE)
    const cy0 = Math.floor((loZ - pad) / BLOB_CHUNK_SIZE)
    const cy1 = Math.floor((hiZ + pad) / BLOB_CHUNK_SIZE)
    for (let ccy = cy0; ccy <= cy1; ccy++) {
      for (let ccx = cx0; ccx <= cx1; ccx++) {
        this.ensureChunk(ccx, ccy)
      }
    }
  }

  getMaskWater(tx: number, ty: number): number {
    return computeBlobMask(tx, ty, (nx, ny) => this.landWorld(nx, ny) === 0)
  }

  getMaskLandBiome(tx: number, ty: number, b: number): number {
    return computeBlobMask(tx, ty, (nx, ny) => this.landWorld(nx, ny) === 1 && this.biomeWorld(nx, ny) === b)
  }

  /**
   * 单格最终贴图类型 + 图集下标（渲染前再结合 `nearestBlobTileIndex` 的掩码逻辑）。
   * `(tx,ty)` 即世界格，与场景里 `tix/tiz` 一致。
   */
  sampleTileIndex(tx: number, ty: number): { kind: 'water' | 'mtn' | 'norm'; sheetIndex: number } {
    if (this.landWorld(tx, ty) === 0) {
      return { kind: 'water', sheetIndex: nearestBlobTileIndex(this.getMaskWater(tx, ty)) }
    }
    const b = this.biomeWorld(tx, ty)
    const mask = this.getMaskLandBiome(tx, ty, b)
    return b === IMG_MTN
      ? { kind: 'mtn', sheetIndex: nearestBlobTileIndex(mask) }
      : { kind: 'norm', sheetIndex: nearestBlobTileIndex(mask) }
  }

  /** 换随机种子重算噪声；保留 seaLevel / mtnTh，并清空 chunk 缓存 */
  reseed(seedStr: string): void {
    const base = stringSeedToUint32(seedStr.trim() || 'default')
    this.nHeight = createPerlin2D(base)
    this.nS4 = createPerlin2D(base ^ 0x414004)
    this.nRiver = createPerlin2D(base ^ 0x927b51c1)
    this.clearCache()
  }
}

/** 仅平地可走；水与山地不可走 */
export function isBlobTileWalkable(world: BlobWorld, tx: number, ty: number): boolean {
  if (world.landWorld(tx, ty) !== 1) return false
  return world.biomeWorld(tx, ty) === IMG_NORM
}

/** 陆地（平地或山地）；水域为 false — 用于树木等装饰 */
export function isBlobTileLandNotWater(world: BlobWorld, tx: number, ty: number): boolean {
  return world.landWorld(tx, ty) === 1
}

/** 从原点向外按 Chebyshev 距离扩环，找最近可站立格，返回格中心世界坐标 */
export function findWalkableTileCenter(
  world: BlobWorld,
  originTx: number,
  originTz: number,
  maxChebRadius: number,
  tileWorld: number,
): { wx: number; wz: number } | null {
  for (let r = 0; r <= maxChebRadius; r++) {
    for (let dz = -r; dz <= r; dz++) {
      for (let dx = -r; dx <= r; dx++) {
        if (Math.max(Math.abs(dx), Math.abs(dz)) !== r) continue
        const tx = originTx + dx
        const tz = originTz + dz
        if (isBlobTileWalkable(world, tx, tz)) {
          return { wx: (tx + 0.5) * tileWorld, wz: (tz + 0.5) * tileWorld }
        }
      }
    }
  }
  return null
}

/**
 * 将整张 `frame_001.png` 类图读入为 `BlobAtlas`（不切 Canvas，保留 ImageData）。
 * 子块尺寸 = 图宽/3 × 图高/24。
 */
export function decodeBlobAtlasFromImage(img: HTMLImageElement): BlobAtlas | null {
  const iw = img.naturalWidth
  const ih = img.naturalHeight
  const cellW = Math.floor(iw / BLOB_TILE_COLS)
  const cellH = Math.floor(ih / BLOB_TILE_ROWS)
  if (cellW < 1 || cellH < 1) return null
  const c = document.createElement('canvas')
  c.width = iw
  c.height = ih
  const x = c.getContext('2d')
  if (!x) return null
  x.drawImage(img, 0, 0)
  const id = x.getImageData(0, 0, iw, ih)
  return {
    data: id.data,
    stride: iw * 4,
    cellW,
    cellH,
  }
}

function worldFracInTile(world: number, tileWorld: number): number {
  let f = world % tileWorld
  if (f < 0) f += tileWorld
  return f / tileWorld
}

/** 与 map.html / Canvas 图集一致：ImageData 行 0 在上方，不对 wz 做垂直翻转 */
/**
 * 在世界位置 `(wx,wz)` 处，从指定 sheet 子块取 RGB（最近邻）。
 * `wz` 方向与 Canvas 图像 v 轴一致（不翻转），与 map.html 约定相同。
 */
export function sampleBlobAtlas(
  atlas: BlobAtlas,
  sheetIndex: number,
  wx: number,
  wz: number,
  tileWorld: number,
): [number, number, number] {
  const { data, stride, cellW, cellH } = atlas
  const col = sheetIndex % BLOB_TILE_COLS
  const row = (sheetIndex / BLOB_TILE_COLS) | 0
  const u = Math.min(cellW - 1, Math.floor(worldFracInTile(wx, tileWorld) * cellW))
  const v = Math.min(cellH - 1, Math.floor(worldFracInTile(wz, tileWorld) * cellH))
  const px = col * cellW + u
  const py = row * cellH + v
  const i = py * stride + px * 4
  return [data[i]!, data[i + 1]!, data[i + 2]!]
}

/** 调试用短标签：水/山/平 + 掩码对应下标 */
export function describeBlobTerrain(world: BlobWorld, tx: number, ty: number): string {
  if (world.landWorld(tx, ty) === 0) {
    const idx = nearestBlobTileIndex(world.getMaskWater(tx, ty))
    return `W:${idx}`
  }
  const b = world.biomeWorld(tx, ty)
  const tag = b === IMG_MTN ? '山' : '平'
  const idx = nearestBlobTileIndex(world.getMaskLandBiome(tx, ty, b))
  return `${tag}:${idx}`
}
