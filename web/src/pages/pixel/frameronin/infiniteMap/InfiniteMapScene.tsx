import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'
import { Button, Checkbox, Select, Slider, Typography } from 'antd'
import { useLanguage } from '../i18n'
import { ANIMS, DEFAULT_CHAR_URL, extractFrame, REGIONS } from './infiniteMapSpriteData'
import {
  BlobWorld,
  decodeBlobAtlasFromImage,
  describeBlobTerrain,
  findWalkableTileCenter,
  isBlobTileLandNotWater,
  isBlobTileWalkable,
  sampleBlobAtlas,
  type BlobAtlas,
} from './blobTerrain'
import {
  createMonsterSwarm,
  loadMonsterImages,
  MONSTER_COLS,
  MONSTER_ROWS,
  stepMonster,
  velocityToMonsterRow,
  type MonsterInst,
} from './infiniteMapMonsters'
import {
  buildCityPropLayout,
  CITY_EW_FENCE_Z_CENTER,
  CITY_EW_FENCE_Z_STEP,
  CITY_GEN_DEFAULTS,
  ewFenceExtentZ,
  findNearestFlatSnowTownPlacement,
  isInsidePlacedTownFootprint,
  loadCityPropImages,
  offsetCityPropLayout,
  type CityGenParams,
  type CityPropDef,
} from './infiniteMapCity'

/**
 * @fileoverview 无限地图主场景：程序化地形 + 透视/俯视双模式 + 小镇/树/野怪/粒子/兽人小屋。
 *
 * ## 渲染管线（每帧 `gameLoop`）
 * 1. **地形光栅**：对离屏 `W×H` 逐像素 `screenToWorld` / `screenToWorldTopdown`，`BlobWorld.sampleTileIndex` + 图集 RGB 写入 `ImageData`。
 * 2. **背景合成**：`ctx.drawImage(off, …)` 裁出下方 16:9 区域（`DISPLAY_H`，去掉 `CROP_TOP`）。
 * 3. **深度批处理**：小镇道具、树、兽人小屋、野怪、玩家按 **透视 sy**（或俯视 sy）排序后绘制。
 * 4. **天气粒子**：世界坐标 `wx,wz` + `pixelFallY` 投影（避免与「钉屏中」的角色同动）；`blob` 雪 / `tileg` 樱花 / `tiler` 树叶。
 * 5. **后处理**：压暗与暗角 `drawScenePostFx`。
 *
 * ## 世界与相机
 * - 世界水平面 **XZ**：`TILE_WORLD` 为格宽；角色 `posRef` 为脚底；相机在 `camZ = pos.z + PLAYER_DZ`，朝 −Z 看（与旧版地图 180° 一致）。
 * - **透视**公式见 `worldToScreen` / `screenToWorld`（针孔模型 + 行号对应深度）。
 *
 * ## 状态与 ref
 * 动画循环内大量读 `*Ref.current`，避免 `useCallback([state])` 重建 `gameLoop`；与 UI 同步的标量用 `useEffect` 写入 ref。
 */
const { Text } = Typography

// ---------- 资源 URL / 地形贴图集切换 ----------

const BGM_URL = `${import.meta.env.BASE_URL}map/ff6.ogg`
const TREE_SNOW_URL = `${import.meta.env.BASE_URL}map/trees/treesnow.png`
const TREE_GRASS_URL = `${import.meta.env.BASE_URL}map/trees/treegrass.png`
const MONSTER_HUT_URL = `${import.meta.env.BASE_URL}map/monsterhut/monsterhut.png`

/**
 * Blob 地形贴图集：与 map/blob 一致；
 * tileg / tiler 为另两套同布局素材（005/006 对应原 004X1/004X2，与 tileg 一致）。
 */
export type InfiniteMapTerrainTextureSetId = 'blob' | 'tileg' | 'tiler'

function terrainTextureUrls(id: InfiniteMapTerrainTextureSetId): {
  mtn: string
  norm: string
  normX1: string
  normX2: string
  water: string[]
} {
  if (id === 'blob') {
    const base = `${import.meta.env.BASE_URL}map/blob/`
    return {
      mtn: `${base}frame_001.png`,
      norm: `${base}frame_004.png`,
      normX1: `${base}frame_004X1.png`,
      normX2: `${base}frame_004X2.png`,
      water: [`${base}frame_007.png`],
    }
  }
  const folder = id === 'tileg' ? 'tileg' : 'tiler'
  const base = `${import.meta.env.BASE_URL}map/${folder}/`
  return {
    mtn: `${base}frame_001.png`,
    norm: `${base}frame_004.png`,
    normX1: `${base}frame_005.png`,
    normX2: `${base}frame_006.png`,
    water: [`${base}frame_007.png`],
  }
}

/** 与地形套对应的飘落粒子种类 */
type FallParticleKind = 'snow' | 'petal' | 'leaf'

function fallParticleKindForTerrain(id: InfiniteMapTerrainTextureSetId): FallParticleKind {
  if (id === 'blob') return 'snow'
  if (id === 'tileg') return 'petal'
  return 'leaf'
}
/** 平地 004 中与「四周同类」最常见子格对应的图集下标（0-based，即 map 示意里的中心块） */
const NORM_CENTER_SHEET_INDEX = 4
/**
 * 仅对上述中心格：004X1 / 004X2 点缀概率（千分比，0–1000）。
 * 例：各 25 → 约 2.5% 用 X1、2.5% 用 X2，其余约 95% 仍为 frame_004。
 * 调大数字 = 变种略多；两者不必相等。
 */
const NORM_CENTER_X1_PERMILLE = 28
const NORM_CENTER_X2_PERMILLE = 28
/** 与 public/map/blob/map.html 滑条一致：value/100 → BlobWorld.seaLevel / mtnTh */
const TERRAIN_SEA_MIN = 0
const TERRAIN_SEA_MAX = 58
const TERRAIN_MTN_MIN = 25
const TERRAIN_MTN_MAX = 75

/** 逻辑分辨率（与 ControlTest topdown 一致） */
const W = 480
const H = 320
/** 显示为 16:9：从画面上方裁掉一条，只保留下方区域 */
const DISPLAY_H = Math.round((W * 9) / 16)
const CROP_TOP = H - DISPLAY_H
/** 透视：地平线提高 = 俯仰略抬，远景地面条带移出可渲染带，少算 blob */
const HORIZON = Math.floor(H * 0.1)
const FOCAL = 250
const CAM_HEIGHT = 100
const PLAYER_DZ = 120
/** 透视/俯视：投影整体下移（逻辑像素，Y 向下为正），人物在 16:9 视窗内略靠下 */
const CAMERA_FRAME_NUDGE_SY = 24
/**
 * 距地平线过近的扫描行：dz 极大，对应极远地面；不跑 blob（与视野外一致，省算力）。
 * 这些行改画远景雾色。
 */
const TERRAIN_MIN_ROW = 14

/** 世界格尺寸（blob 地块） */
const TILE_WORLD = 16
/** 大地图兽人小屋：数量少、仅可走平地、避开小镇与栏 */
const MONSTER_HUT_COUNT = 7
const MONSTER_HUT_MIN_SEP_WORLD = TILE_WORLD * 24
const MONSTER_HUT_REF_DZ = 102
const MONSTER_HUT_DISPLAY_SCALE = 1.32
const MONSTER_HUT_FEET_DOWN_SRC_PX = 10
const MOVE_SPEED = 1
const RUN_MUL = 2

// ---------- 飘落粒子（雪 / 樱花 / 树叶，世界坐标 + 屏显下落） ----------

/** FF6 开篇风格：斜飘、远景小点 / 近景大块，整数像素绘制 */
const FALL_SNOW_COUNT = 340
type FallLayer = 0 | 1 | 2
/**
 * 世界水平坐标 + 屏显下落偏移。若用纯屏幕 x/y，会与始终投影在画面固定处的角色一样「钉在视口上」。
 */
type FallFlake = {
  wx: number
  wz: number
  /** 叠在 worldToScreen 的 sy 上，模拟下落（逻辑像素） */
  pixelFallY: number
  /** 屏空间水平漂移（px/s），经 dz/FOCAL 换成世界 wx */
  vxScreen: number
  vy: number
  wobble: number
  layer: FallLayer
  /** 同层内形状变体（像素图案） */
  variant: number
}

/**
 * 在天空射线与地面的交点附近生成新粒子；失败则放在相机前方兜底。
 * `particleKind` 决定水平/垂直速度与后续摆动参数族。
 */
function respawnFallFlake(
  camX: number,
  camZ: number,
  layer: FallLayer,
  particleKind: FallParticleKind,
): FallFlake {
  const slow = layer === 0
  const mid = layer === 1
  let vxScreen: number
  let vy: number
  if (particleKind === 'snow') {
    vxScreen = 8 + (slow ? 3 : mid ? 12 : 20) + Math.random() * 14
    vy = (slow ? 18 : mid ? 42 : 72) + Math.random() * 38
  } else if (particleKind === 'petal') {
    vxScreen = 5 + (slow ? 4 : mid ? 12 : 20) + Math.random() * 16
    vy = (slow ? 11 : mid ? 26 : 44) + Math.random() * 26
  } else {
    vxScreen = 6 + (slow ? 6 : mid ? 14 : 22) + Math.random() * 18
    vy = (slow ? 13 : mid ? 30 : 50) + Math.random() * 28
  }
  for (let tries = 0; tries < 24; tries++) {
    const sx = Math.random() * W
    const syRow = HORIZON + 4 + Math.random() * 56
    const hit = screenToWorld(sx + 0.5, syRow + 0.5, camX, camZ)
    if (!hit) continue
    return {
      wx: hit.wx + (Math.random() - 0.5) * 36,
      wz: hit.wz + (Math.random() - 0.5) * 32,
      pixelFallY: -(32 + Math.random() * 150),
      vxScreen,
      vy,
      wobble: Math.random() * Math.PI * 2,
      layer,
      variant: Math.floor(Math.random() * 16),
    }
  }
  return {
    wx: camX + (Math.random() - 0.5) * 200,
    wz: camZ - 60 - Math.random() * 220,
    pixelFallY: -(40 + Math.random() * 100),
    vxScreen,
    vy,
    wobble: Math.random() * Math.PI * 2,
    layer,
    variant: Math.floor(Math.random() * 16),
  }
}

/**
 * 风力用屏空间速度经 `dz/FOCAL` 换到世界 wx；`pixelFallY` 叠加在投影行上模拟下落。
 * 出界或落到相机后方则 `respawnFallFlake`。
 */
function updateFallFlake(
  f: FallFlake,
  camX: number,
  camZ: number,
  dt: number,
  t0: number,
  particleKind: FallParticleKind,
): FallFlake {
  let swayAmp: number
  let twMul: number
  if (particleKind === 'snow') {
    swayAmp = f.layer === 0 ? 10 : f.layer === 1 ? 6 : 4
    twMul = 0.0018
  } else if (particleKind === 'petal') {
    swayAmp = f.layer === 0 ? 13 : f.layer === 1 ? 9 : 6
    twMul = 0.0021
  } else {
    swayAmp = f.layer === 0 ? 15 : f.layer === 1 ? 11 : 7
    twMul = 0.00235
  }
  const tw = t0 * twMul
  let { wx, wz, pixelFallY, vxScreen, vy, wobble, layer, variant } = f
  const dz = camZ - wz
  if (dz <= 0) return respawnFallFlake(camX, camZ, layer, particleKind)
  wx += (dz / FOCAL) * (vxScreen + Math.sin(tw + wobble) * swayAmp) * dt
  pixelFallY += vy * dt
  const scr = worldToScreen(wx, wz, camX, camZ)
  if (!scr || scr.sy < HORIZON - 8) return respawnFallFlake(camX, camZ, layer, particleKind)
  const syDisp = scr.sy - CROP_TOP + pixelFallY
  const margin = layer === 2 ? 6 : layer === 1 ? 3 : 2
  if (
    syDisp > DISPLAY_H + margin ||
    scr.sx < -margin - 70 ||
    scr.sx > W + margin + 70
  ) {
    return respawnFallFlake(camX, camZ, layer, particleKind)
  }
  return { wx, wz, pixelFallY, vxScreen, vy, wobble, layer, variant }
}

// ---------- 粒子像素画（按层远/中/近） ----------

function rgbForSnowFlake(layer: FallLayer, wobble: number): [number, number, number] {
  const t = Math.sin(wobble * 0.37) * 14
  if (layer === 0) return [148 + t, 168 + t, 202 + t]
  if (layer === 1) return [188 + t, 208 + t, 238 + t]
  return [228 + t, 236 + t, 252 + t]
}

/** 单像素点（远景） */
function drawSnowFar(ctx: CanvasRenderingContext2D, xi: number, yi: number, rgb: [number, number, number]) {
  const [r, g, b] = rgb
  ctx.fillStyle = `rgb(${Math.floor(r)},${Math.floor(g)},${Math.floor(b)})`
  ctx.fillRect(xi, yi, 1, 1)
}

/** 2×2 实块（中景） */
function drawSnowMid(ctx: CanvasRenderingContext2D, xi: number, yi: number, rgb: [number, number, number]) {
  const [r, g, b] = rgb
  ctx.fillStyle = `rgb(${Math.floor(r)},${Math.floor(g)},${Math.floor(b)})`
  ctx.fillRect(xi, yi, 2, 2)
}

/** 近景：3×3 / 4×4 整数像素簇（非矢量缩放，保持 SNES 块感） */
function drawSnowNear(
  ctx: CanvasRenderingContext2D,
  xi: number,
  yi: number,
  rgb: [number, number, number],
  variant: number,
  cw: number,
  ch: number,
) {
  const [r0, g0, b0] = rgb
  const r = Math.floor(r0)
  const g = Math.floor(g0)
  const b = Math.floor(b0)
  const rHi = Math.min(255, r + 24)
  const gHi = Math.min(255, g + 20)
  const bHi = Math.min(255, b + 14)
  const pat = variant % 4
  const plot = (dx: number, dy: number, hi: boolean) => {
    const x = xi + dx
    const y = yi + dy
    if (x < 0 || x >= cw || y < 0 || y >= ch) return
    ctx.fillStyle = hi ? `rgb(${rHi},${gHi},${bHi})` : `rgb(${r},${g},${b})`
    ctx.fillRect(x, y, 1, 1)
  }
  if (pat === 0) {
    for (let dy = 0; dy < 3; dy++) {
      for (let dx = 0; dx < 3; dx++) {
        plot(dx, dy, dx === 1 && dy === 1)
      }
    }
  } else if (pat === 1) {
    for (let dy = 0; dy < 3; dy++) {
      for (let dx = 0; dx < 3; dx++) {
        plot(dx, dy, (dx + dy) % 2 === 0)
      }
    }
  } else if (pat === 2) {
    const pts: [number, number][] = [
      [1, 0],
      [2, 0],
      [0, 1],
      [1, 1],
      [2, 1],
      [3, 1],
      [1, 2],
      [2, 2],
      [3, 2],
      [2, 3],
      [3, 3],
    ]
    for (let i = 0; i < pts.length; i++) {
      const [dx, dy] = pts[i]!
      plot(dx, dy, i % 2 === 0)
    }
  } else {
    for (let dy = 0; dy < 4; dy++) {
      for (let dx = 0; dx < 4; dx++) {
        const d = Math.abs(dx - 1.5) + Math.abs(dy - 1.5)
        if (d < 2.65) plot(dx, dy, d < 1.25)
      }
    }
  }
}

function rgbForPetalFlake(layer: FallLayer, wobble: number): [number, number, number] {
  const t = Math.sin(wobble * 0.41) * 16
  if (layer === 0) return [172 + t * 0.4, 128 + t * 0.25, 152 + t * 0.35]
  if (layer === 1) return [236 + t * 0.35, 186 + t * 0.3, 206 + t * 0.3]
  return [255, 218 + t * 0.2, 228 + t * 0.18]
}

function drawPetalFar(ctx: CanvasRenderingContext2D, xi: number, yi: number, rgb: [number, number, number]) {
  const [r, g, b] = rgb
  ctx.fillStyle = `rgb(${Math.floor(r)},${Math.floor(g)},${Math.floor(b)})`
  ctx.fillRect(xi, yi, 1, 2)
}

function drawPetalMid(ctx: CanvasRenderingContext2D, xi: number, yi: number, rgb: [number, number, number]) {
  const [r0, g0, b0] = rgb
  const r = Math.floor(r0)
  const g = Math.floor(g0)
  const b = Math.floor(b0)
  ctx.fillStyle = `rgb(${r},${g},${b})`
  ctx.fillRect(xi, yi, 4, 1)
  ctx.fillStyle = `rgb(${Math.min(255, r + 36)},${Math.min(255, g + 28)},${Math.min(255, b + 22)})`
  ctx.fillRect(xi + 1, yi - 1, 2, 1)
}

function drawPetalNear(
  ctx: CanvasRenderingContext2D,
  xi: number,
  yi: number,
  rgb: [number, number, number],
  variant: number,
  cw: number,
  ch: number,
) {
  const [r0, g0, b0] = rgb
  const r = Math.floor(r0)
  const g = Math.floor(g0)
  const b = Math.floor(b0)
  const rHi = Math.min(255, r + 40)
  const gHi = Math.min(255, g + 32)
  const bHi = Math.min(255, b + 26)
  const pat = variant % 4
  const plot = (dx: number, dy: number, hi: boolean) => {
    const x = xi + dx
    const y = yi + dy
    if (x < 0 || x >= cw || y < 0 || y >= ch) return
    ctx.fillStyle = hi ? `rgb(${rHi},${gHi},${bHi})` : `rgb(${r},${g},${b})`
    ctx.fillRect(x, y, 1, 1)
  }
  if (pat === 0) {
    for (let dx = 0; dx < 5; dx++) plot(dx, 1, dx === 2 || dx === 3)
    plot(1, 0, true)
    plot(3, 0, true)
    plot(2, 2, false)
  } else if (pat === 1) {
    const pts: [number, number][] = [
      [0, 1],
      [1, 1],
      [2, 1],
      [3, 1],
      [4, 1],
      [1, 0],
      [2, 0],
      [3, 0],
      [2, 2],
    ]
    for (let i = 0; i < pts.length; i++) {
      const [dx, dy] = pts[i]!
      plot(dx, dy, i % 3 === 0)
    }
  } else if (pat === 2) {
    for (let dx = 0; dx < 4; dx++) plot(dx, 1, dx >= 1 && dx <= 2)
    plot(0, 0, false)
    plot(3, 0, false)
    plot(1, 2, true)
    plot(2, 2, true)
  } else {
    for (let dx = 0; dx < 6; dx++) plot(dx, 2, dx === 2 || dx === 3)
    plot(1, 1, true)
    plot(4, 1, true)
    plot(2, 3, false)
    plot(3, 3, false)
  }
}

function rgbForLeafFlake(layer: FallLayer, wobble: number): [number, number, number] {
  const t = Math.sin(wobble * 0.33) * 20
  const u = Math.cos(wobble * 0.51) * 14
  const autumn = Math.sin(wobble * 0.17) > 0.65 ? 1 : 0
  if (layer === 0) {
    if (autumn) return [88 + t * 0.2, 112 + u * 0.15, 42 + t * 0.15]
    return [38 + t * 0.2, 98 + u * 0.18, 44 + t * 0.18]
  }
  if (layer === 1) {
    if (autumn) return [118 + t * 0.25, 142 + u * 0.2, 48 + t * 0.2]
    return [62 + t * 0.28, 154 + u * 0.25, 52 + t * 0.22]
  }
  if (autumn) return [148 + t * 0.3, 168 + u * 0.25, 58 + t * 0.22]
  return [78 + t * 0.32, 188 + u * 0.3, 58 + t * 0.28]
}

function drawLeafFar(ctx: CanvasRenderingContext2D, xi: number, yi: number, rgb: [number, number, number]) {
  const [r, g, b] = rgb
  const rf = Math.floor(r)
  const gf = Math.floor(g)
  const bf = Math.floor(b)
  ctx.fillStyle = `rgb(${rf},${gf},${bf})`
  ctx.fillRect(xi, yi, 1, 1)
  ctx.fillStyle = `rgb(${Math.min(255, rf + 18)},${Math.min(255, gf + 22)},${Math.min(255, bf + 10)})`
  ctx.fillRect(xi + 1, yi - 1, 1, 1)
}

function drawLeafMid(ctx: CanvasRenderingContext2D, xi: number, yi: number, rgb: [number, number, number]) {
  const [r0, g0, b0] = rgb
  const r = Math.floor(r0)
  const g = Math.floor(g0)
  const b = Math.floor(b0)
  const rHi = Math.min(255, r + 28)
  const gHi = Math.min(255, g + 32)
  const bHi = Math.min(255, b + 16)
  const plot = (dx: number, dy: number, hi: boolean) => {
    ctx.fillStyle = hi ? `rgb(${rHi},${gHi},${bHi})` : `rgb(${r},${g},${b})`
    ctx.fillRect(xi + dx, yi + dy, 1, 1)
  }
  plot(0, 1, false)
  plot(1, 1, true)
  plot(2, 1, false)
  plot(1, 0, true)
  plot(2, 0, false)
  plot(1, 2, false)
}

function drawLeafNear(
  ctx: CanvasRenderingContext2D,
  xi: number,
  yi: number,
  rgb: [number, number, number],
  variant: number,
  cw: number,
  ch: number,
) {
  const [r0, g0, b0] = rgb
  const r = Math.floor(r0)
  const g = Math.floor(g0)
  const b = Math.floor(b0)
  const rHi = Math.min(255, r + 34)
  const gHi = Math.min(255, g + 38)
  const bHi = Math.min(255, b + 20)
  const pat = variant % 4
  const plot = (dx: number, dy: number, hi: boolean) => {
    const x = xi + dx
    const y = yi + dy
    if (x < 0 || x >= cw || y < 0 || y >= ch) return
    ctx.fillStyle = hi ? `rgb(${rHi},${gHi},${bHi})` : `rgb(${r},${g},${b})`
    ctx.fillRect(x, y, 1, 1)
  }
  if (pat === 0) {
    const pts: [number, number][] = [
      [1, 0],
      [2, 0],
      [0, 1],
      [1, 1],
      [2, 1],
      [3, 1],
      [1, 2],
      [2, 2],
      [2, 3],
    ]
    for (let i = 0; i < pts.length; i++) {
      const [dx, dy] = pts[i]!
      plot(dx, dy, i % 3 === 1)
    }
  } else if (pat === 1) {
    for (let dx = 0; dx < 4; dx++) plot(dx, 2, dx === 1 || dx === 2)
    plot(1, 1, true)
    plot(2, 1, true)
    plot(0, 3, false)
    plot(3, 3, false)
    plot(1, 3, false)
    plot(2, 3, false)
  } else if (pat === 2) {
    plot(2, 0, true)
    plot(1, 1, false)
    plot(2, 1, true)
    plot(3, 1, false)
    plot(0, 2, false)
    plot(1, 2, true)
    plot(2, 2, true)
    plot(3, 2, false)
    plot(1, 3, false)
    plot(2, 3, true)
  } else {
    for (let dy = 0; dy < 4; dy++) {
      for (let dx = 0; dx < 4; dx++) {
        const d = Math.abs(dx - 1.5) + Math.abs(dy - 1.5)
        if (d < 2.4) plot(dx, dy, d < 1.2)
      }
    }
  }
}

// ---------- 玩家脚底椭圆阴影（小离屏纹理） ----------

let shadowTexCache: HTMLCanvasElement | null = null
function getShadowTexture(): HTMLCanvasElement {
  if (shadowTexCache) return shadowTexCache
  const c = document.createElement('canvas')
  const w = 28
  const h = 10
  c.width = w
  c.height = h
  const ctx = c.getContext('2d')!
  ctx.fillStyle = 'rgba(0,0,0,0.45)'
  ctx.beginPath()
  ctx.ellipse(w / 2, h / 2, w / 2 - 1, h / 2 - 1, 0, 0, Math.PI * 2)
  ctx.fill()
  shadowTexCache = c
  return c
}

// ---------- 地形像素着色辅助（逐像素 raycast 用） ----------

/** 黑白马赛克：图集未加载时的兜底 */
function mosaicRgb(tileIx: number, tileIz: number): [number, number, number] {
  const dark = ((tileIx + tileIz) & 1) === 0
  const v = dark ? 28 : 228
  return [v, v, v]
}

function pickWaterAtlas(waterFrames: (BlobAtlas | null)[], animT: number): BlobAtlas | null {
  const n = waterFrames.length
  if (n === 0) return null
  const phase = Math.floor(animT / 220) % n
  for (let k = 0; k < n; k++) {
    const a = waterFrames[(phase + k) % n]
    if (a) return a
  }
  return null
}

type BlobTilePick = ReturnType<BlobWorld['sampleTileIndex']>

function pickNormAtlasForSheet(
  sheetIndex: number,
  tix: number,
  tiz: number,
  base: BlobAtlas | null,
  x1: BlobAtlas | null,
  x2: BlobAtlas | null,
): BlobAtlas | null {
  if (sheetIndex !== NORM_CENTER_SHEET_INDEX) return base
  const h = (Math.imul(tix, 92837111) ^ Math.imul(tiz, 689287499)) >>> 0
  const u = h % 1000
  const c1 = Math.min(1000, Math.max(0, NORM_CENTER_X1_PERMILLE))
  const c2 = Math.min(Math.max(0, NORM_CENTER_X2_PERMILLE), 1000 - c1)
  if (u < c1 && x1) return x1
  if (u < c1 + c2 && x2) return x2
  return base
}

function sampleBlobTerrainRgbResolved(
  atlasMtn: BlobAtlas | null,
  atlasNorm: BlobAtlas | null,
  atlasNormX1: BlobAtlas | null,
  atlasNormX2: BlobAtlas | null,
  waterFrames: (BlobAtlas | null)[],
  animT: number,
  pick: BlobTilePick,
  tix: number,
  tiz: number,
  wx: number,
  wz: number,
): [number, number, number] {
  const { kind, sheetIndex } = pick
  if (kind === 'water') {
    const wa = pickWaterAtlas(waterFrames, animT)
    if (!wa) return mosaicRgb(tix, tiz)
    return sampleBlobAtlas(wa, sheetIndex, wx, wz, TILE_WORLD)
  }
  if (kind === 'mtn') {
    if (!atlasMtn) return mosaicRgb(tix, tiz)
    return sampleBlobAtlas(atlasMtn, sheetIndex, wx, wz, TILE_WORLD)
  }
  const land = pickNormAtlasForSheet(sheetIndex, tix, tiz, atlasNorm, atlasNormX1, atlasNormX2)
  if (!land) return mosaicRgb(tix, tiz)
  return sampleBlobAtlas(land, sheetIndex, wx, wz, TILE_WORLD)
}

const TERRAIN_SY_START = HORIZON + TERRAIN_MIN_ROW

// ---------- 投影：透视（游玩）与正交俯视（编辑） ----------

/** 编辑模式：正交俯视，世界单位 → 屏幕像素；数值越小视野越大（同屏显示更多格） */
const TOPDOWN_PX_PER_WORLD = 1.72 / 2

function terrainScreenMidY(): number {
  return (TERRAIN_SY_START + H - 1) / 2
}

/** 与俯视一致：角色脚底对齐 terrain 带垂直中线（俯视用 mid 作投影中心行），再加 CAMERA_FRAME_NUDGE_SY 整体下移 */
const PERSP_SY_OFFSET =
  terrainScreenMidY() - (HORIZON + (FOCAL * CAM_HEIGHT) / PLAYER_DZ) + CAMERA_FRAME_NUDGE_SY

/** 透视：相机在玩家 +Z 侧，朝 -Z 看；相对原实现等于水平面转 180° */
function screenToWorld(sx: number, sy: number, camX: number, camZ: number): { wx: number; wz: number } | null {
  const row = sy - HORIZON - PERSP_SY_OFFSET
  if (row <= 0) return null
  const dz = (FOCAL * CAM_HEIGHT) / row
  const wx = camX - ((sx - W / 2) * dz) / FOCAL
  const wz = camZ - dz
  return { wx, wz }
}

function worldToScreen(wx: number, wz: number, camX: number, camZ: number): { sx: number; sy: number } | null {
  const dz = camZ - wz
  if (dz <= 0) return null
  const sx = W / 2 - ((wx - camX) * FOCAL) / dz
  const row = (FOCAL * CAM_HEIGHT) / dz
  const sy = HORIZON + row + PERSP_SY_OFFSET
  return { sx, sy }
}

function screenToWorldTopdown(
  sx: number,
  sy: number,
  camX: number,
  camZ: number,
): { wx: number; wz: number } | null {
  if (sy < TERRAIN_SY_START) return null
  const k = TOPDOWN_PX_PER_WORLD
  const mid = terrainScreenMidY() + CAMERA_FRAME_NUDGE_SY
  /** 与透视同一套水平面 180°：东↔西、北↔南 */
  return {
    wx: camX - (sx - W / 2) / k,
    wz: camZ + (sy - mid) / k,
  }
}

function worldToScreenTopdown(wx: number, wz: number, camX: number, camZ: number): { sx: number; sy: number } {
  const k = TOPDOWN_PX_PER_WORLD
  const mid = terrainScreenMidY() + CAMERA_FRAME_NUDGE_SY
  return {
    sx: W / 2 - (wx - camX) * k,
    sy: mid + (wz - camZ) * k,
  }
}

function visibleGroundTileBoundsTopdown(
  camX: number,
  camZ: number,
): { tix0: number; tix1: number; tiz0: number; tiz1: number } | null {
  const corners: [number, number][] = [
    [0, TERRAIN_SY_START],
    [W - 1, TERRAIN_SY_START],
    [0, H - 1],
    [W - 1, H - 1],
  ]
  let minWx = Infinity
  let maxWx = -Infinity
  let minWz = Infinity
  let maxWz = -Infinity
  for (const [sx, sy] of corners) {
    const h = screenToWorldTopdown(sx + 0.5, sy + 0.5, camX, camZ)
    if (!h) continue
    minWx = Math.min(minWx, h.wx)
    maxWx = Math.max(maxWx, h.wx)
    minWz = Math.min(minWz, h.wz)
    maxWz = Math.max(maxWz, h.wz)
  }
  if (!Number.isFinite(minWx)) return null
  const pad = TILE_WORLD * 6
  return {
    tix0: Math.floor((minWx - pad) / TILE_WORLD),
    tix1: Math.floor((maxWx + pad) / TILE_WORLD),
    tiz0: Math.floor((minWz - pad) / TILE_WORLD),
    tiz1: Math.floor((maxWz + pad) / TILE_WORLD),
  }
}

/** 地平线以下、尚未进入 blob 带的行：远景雾（无地块采样） */
function farHazeRgb(sy: number): [number, number, number] {
  const t = (sy - HORIZON) / Math.max(1, TERRAIN_MIN_ROW)
  const g = Math.floor(14 + t * 22)
  const b = Math.floor(22 + t * 28)
  return [g, b, Math.min(48, b + 8)]
}

/** 当前帧地面在屏幕上的世界格包络，用于 chunk 预加载（不加载视野外） */
function visibleGroundTileBounds(camX: number, camZ: number): { tix0: number; tix1: number; tiz0: number; tiz1: number } | null {
  const corners: [number, number][] = [
    [0, TERRAIN_SY_START],
    [W - 1, TERRAIN_SY_START],
    [0, H - 1],
    [W - 1, H - 1],
  ]
  let minWx = Infinity
  let maxWx = -Infinity
  let minWz = Infinity
  let maxWz = -Infinity
  for (const [sx, sy] of corners) {
    const h = screenToWorld(sx + 0.5, sy + 0.5, camX, camZ)
    if (!h) continue
    minWx = Math.min(minWx, h.wx)
    maxWx = Math.max(maxWx, h.wx)
    minWz = Math.min(minWz, h.wz)
    maxWz = Math.max(maxWz, h.wz)
  }
  if (!Number.isFinite(minWx)) return null
  const pad = TILE_WORLD * 6
  return {
    tix0: Math.floor((minWx - pad) / TILE_WORLD),
    tix1: Math.floor((maxWx + pad) / TILE_WORLD),
    tiz0: Math.floor((minWz - pad) / TILE_WORLD),
    tiz1: Math.floor((maxWz + pad) / TILE_WORLD),
  }
}

/** 7×7 世界格为一块；块内 hash 决定是否密林，格内 hash 决定是否落树 */
const TREE_CLUSTER_BLOCK = 7
const TREE_HASH_BASE = 0x54726565

function treeCellHash(tix: number, tiz: number, salt: number): number {
  let h = Math.imul(tix, 374761393) ^ Math.imul(tiz, 668265263) ^ salt
  h ^= h >>> 16
  h = Math.imul(h, 2246822519)
  h ^= h >>> 13
  h = Math.imul(h, 3266489917)
  return h >>> 0
}

function treeU01(h: number): number {
  return h / 4294967296
}

/**
 * 平地与山地（陆地）均可种树；仅水域不种。
 * 独立树：全陆地按 lonePct 缩放；成片树：仅块 hash 落入林带时按 patchPct 缩放（第二哈希，与独立树独立）。
 */
function shouldPlaceSnowTreeOnTile(
  tix: number,
  tiz: number,
  patchDensityPct: number,
  loneDensityPct: number,
): boolean {
  const patchMul = Math.max(0, Math.min(100, patchDensityPct)) / 100
  const loneMul = Math.max(0, Math.min(100, loneDensityPct)) / 100
  const pLone = loneMul * 0.12
  const localLone = treeU01(treeCellHash(tix, tiz, TREE_HASH_BASE + 3331))
  if (localLone < pLone) return true

  const bx = Math.floor(tix / TREE_CLUSTER_BLOCK)
  const bz = Math.floor(tiz / TREE_CLUSTER_BLOCK)
  const blockV = treeU01(treeCellHash(bx, bz, TREE_HASH_BASE))
  let pPatchCap = 0
  if (blockV > 0.64) pPatchCap = 0.52
  else if (blockV > 0.5) pPatchCap = 0.24
  if (pPatchCap <= 0 || patchMul <= 0) return false
  const pPatch = pPatchCap * patchMul
  const localPatch = treeU01(treeCellHash(tix, tiz, TREE_HASH_BASE + 7777))
  return localPatch < pPatch
}

/** 树脚底落在地块内随机偏移（同一格可区分多棵树） */
function snowTreeFeetWorld(tix: number, tiz: number): { wx: number; wz: number } {
  const h1 = treeCellHash(tix, tiz, TREE_HASH_BASE + 11)
  const h2 = treeCellHash(tix, tiz, TREE_HASH_BASE + 22)
  const margin = 3
  const span = Math.max(1, TILE_WORLD - margin * 2)
  return {
    wx: tix * TILE_WORLD + margin + (h1 % span),
    wz: tiz * TILE_WORLD + margin + (h2 % span),
  }
}

const TREE_REF_DZ = 92
const TREE_DISPLAY_SCALE = 1.5
const MONSTER_REF_DZ = 90
const MONSTER_DISPLAY_SCALE = 1.12
const MONSTER_FEET_DOWN_SRC = 6
const MONSTER_COUNT_MAX = 80
/** 小镇元素相对原广告牌比例的整体缩放（0.2 = 20%） */
const CITY_DISPLAY_SCALE = 1.12 * 0.2
/**
 * 东西栏透视四边形：世界 XZ 底边半宽乘此系数，略压屏上宽度（透视下易显胖）。
 * 1 = 与南北栏同一套 dw 推导；小于 1 仅收窄左右栏。
 */
const FENCE_EW_PERSP_WIDTH_MUL = 0.84

/** 东西栏透视默认（宽/高/段间距）；与先前调好的视觉一致 */
const FENCE_EW_LAYOUT_FIXED = {
  widthPct: 75,
  heightPct: 109,
  spacingPct: 69,
  posZ: 0,
  /** 固定：西侧 +、东侧 −（负值两侧外扩） */
  posX: -16,
} as const

type CityPropDraw = CityPropDef & { img: HTMLImageElement }

/** 东西栏：固定宽高段间距 + 可调左右偏移；绕 Y 固定 90° 透视四边形 */
type CityFenceEwLayoutTweak = {
  widthPct: number
  heightPct: number
  spacingPct: number
  posZ: number
  /** 世界 X：西侧栏 +posX、东侧栏 −posX（正值两侧向中心收） */
  posX: number
}

/** 东西栏绕世界竖轴转角固定为 90°（侧向） */
const FENCE_EW_YAW_RAD = Math.PI / 2

function fenceEwEffectiveWorld(
  t: CityPropDef,
  tw: CityFenceEwLayoutTweak,
  ewZCenter: number,
): { wx: number; wz: number } {
  const wz = ewZCenter + (t.wz - ewZCenter) * (tw.spacingPct / 100) + tw.posZ
  let wx = t.wx
  if (tw.posX !== 0) {
    wx += t.wx < 0 ? tw.posX : -tw.posX
  }
  return { wx, wz }
}

/** 南北栏柱：与段间距 34 闭合，相邻柱碰撞圆相切，不留缝 */
const CITY_FENCE_NS_COLLIDE_R = 17
/** 东西栏：沿 Z 的半长（按透视段间距比例）；厚度为 X 向胶囊半径 */
const CITY_FENCE_EW_HALFLEN_MUL = 0.46
const CITY_FENCE_EW_THICK_R = 10
/** 角色脚底简化为圆，与栏检测半径相加 */
const PLAYER_FENCE_BODY_R = 3

function distSqPointSegment2D(
  px: number,
  pz: number,
  ax: number,
  az: number,
  bx: number,
  bz: number,
): number {
  const abx = bx - ax
  const abz = bz - az
  const apx = px - ax
  const apz = pz - az
  const abLen2 = abx * abx + abz * abz
  if (abLen2 < 1e-8) return apx * apx + apz * apz
  let t = (apx * abx + apz * abz) / abLen2
  t = Math.max(0, Math.min(1, t))
  const qx = ax + t * abx
  const qz = az + t * abz
  const dx = px - qx
  const dz = pz - qz
  return dx * dx + dz * dz
}

/** 与 drawFenceEwPerspectiveYawQuad 一致：底边 Z 限制在南北栏之间 */
function clipEwFenceSegmentToClampZ(
  wx: number,
  wz0: number,
  wz1: number,
  clampMin: number,
  clampMax: number,
): { ax: number; az: number; bx: number; bz: number } | null {
  const lo = Math.min(clampMin, clampMax)
  const hi = Math.max(clampMin, clampMax)
  let zA = Math.min(wz0, wz1)
  let zB = Math.max(wz0, wz1)
  zA = Math.max(zA, lo)
  zB = Math.min(zB, hi)
  if (zA >= zB - 1e-2) return null
  return { ax: wx, az: zA, bx: wx, bz: zB }
}

function worldPosBlockedByCityFence(
  px: number,
  pz: number,
  layout: readonly CityPropDef[],
  ewZCenter: number,
  ewTw: CityFenceEwLayoutTweak,
  ewClampZ: { min: number; max: number },
): boolean {
  const pr = PLAYER_FENCE_BODY_R
  const rNs = CITY_FENCE_NS_COLLIDE_R + pr
  const rNsSq = rNs * rNs
  const rEw = CITY_FENCE_EW_THICK_R + pr
  const rEwSq = rEw * rEw

  for (const def of layout) {
    if (def.file !== 'fence1.png') continue
    if (def.fenceEwUseAxisSliders) {
      const { wx, wz } = fenceEwEffectiveWorld(def, ewTw, ewZCenter)
      const dzStep = CITY_EW_FENCE_Z_STEP * (ewTw.spacingPct / 100)
      const halfLen = Math.max(8, dzStep * CITY_FENCE_EW_HALFLEN_MUL)
      const seg = clipEwFenceSegmentToClampZ(wx, wz - halfLen, wz + halfLen, ewClampZ.min, ewClampZ.max)
      if (!seg) continue
      if (distSqPointSegment2D(px, pz, seg.ax, seg.az, seg.bx, seg.bz) < rEwSq) return true
    } else {
      const dx = px - def.wx
      const dz = pz - def.wz
      if (dx * dx + dz * dz < rNsSq) return true
    }
  }
  return false
}

function mulberry32Hut(seed: number): () => number {
  let a = seed >>> 0
  return () => {
    a |= 0
    a = (a + 0x6d2b79f5) | 0
    let t = Math.imul(a ^ (a >>> 15), a | 1)
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

/**
 * 在世界原点邻域随机抽样可走平地格，摆少量兽人小屋；互斥最小间距、排除小镇脚印与栏碰撞。
 */
function buildMonsterHutSites(
  bw: BlobWorld,
  seedNonce: number,
  layout: readonly CityPropDef[],
  cg: CityGenParams,
  placeX: number,
  placeZ: number,
  ewTw: CityFenceEwLayoutTweak,
  ewClampZ: { min: number; max: number },
): { wx: number; wz: number }[] {
  const rand = mulberry32Hut((seedNonce >>> 0) ^ 0x48757421 ^ Math.imul(cg.seed >>> 0, 2654435761))
  const sites: { wx: number; wz: number }[] = []
  const minD2 = MONSTER_HUT_MIN_SEP_WORLD * MONSTER_HUT_MIN_SEP_WORLD
  const townPad = 56
  let attempts = 0
  const span = 112
  while (sites.length < MONSTER_HUT_COUNT && attempts < 14000) {
    attempts++
    const tix = Math.floor(rand() * (span * 2 + 1)) - span
    const tiz = Math.floor(rand() * (span * 2 + 1)) - span
    if (!isBlobTileWalkable(bw, tix, tiz)) continue
    const wx = (tix + 0.5) * TILE_WORLD
    const wz = (tiz + 0.5) * TILE_WORLD
    if (isInsidePlacedTownFootprint(wx, wz, cg, placeX, placeZ, townPad)) continue
    if (worldPosBlockedByCityFence(wx, wz, layout, cg.ewFenceZCenter, ewTw, ewClampZ)) continue
    if (sites.some((s) => (s.wx - wx) ** 2 + (s.wz - wz) ** 2 < minD2)) continue
    sites.push({ wx, wz })
  }
  return sites
}

/** 深度排序用绘制项：`key` 越大越靠近相机（越后画） */
type DepthSprite = { key: number; draw: () => void }

// ---------- 透视贴图：三角形仿射、东西栏、小镇道具、树、野怪、兽人小屋 ----------

/** 纹理三角形仿射映射到屏幕三角形（Canvas2D 真透视四边形的一半） */
function drawTexturedTriangleAffine(
  ctx: CanvasRenderingContext2D,
  img: CanvasImageSource,
  su0: number,
  sv0: number,
  su1: number,
  sv1: number,
  su2: number,
  sv2: number,
  x0: number,
  y0: number,
  x1: number,
  y1: number,
  x2: number,
  y2: number,
) {
  const x10 = x1 - x0
  const y10 = y1 - y0
  const x20 = x2 - x0
  const y20 = y2 - y0
  const u10 = su1 - su0
  const v10 = sv1 - sv0
  const u20 = su2 - su0
  const v20 = sv2 - sv0
  const det = u10 * v20 - u20 * v10
  if (Math.abs(det) < 1e-5) return
  const idet = 1 / det
  const a = (x10 * v20 - x20 * v10) * idet
  const b = (y10 * v20 - y20 * v10) * idet
  const c = (x20 * u10 - x10 * u20) * idet
  const d = (y20 * u10 - y10 * u20) * idet
  const e = x0 - a * su0 - c * sv0
  const f = y0 - b * su0 - d * sv0
  ctx.save()
  ctx.setTransform(a, b, c, d, e, f)
  ctx.beginPath()
  ctx.moveTo(su0, sv0)
  ctx.lineTo(su1, sv1)
  ctx.lineTo(su2, sv2)
  ctx.closePath()
  ctx.clip()
  ctx.drawImage(img, 0, 0)
  ctx.restore()
}

/**
 * 东西栏：绕世界 Y（竖轴）在 XZ 上转 θ，底边两端 worldToScreen；上边沿沿**屏幕竖直方向**（-Y）抬 dh。
 * 若沿底边法线抬高度，底边在屏上接近竖直时法线接近水平，围栏会像躺倒且随视角/人物位置剧变。
 */
function drawFenceEwPerspectiveYawQuad(
  ctx: CanvasRenderingContext2D,
  img: HTMLImageElement,
  camX: number,
  camZ: number,
  wx: number,
  wz: number,
  dz: number,
  dw: number,
  dh: number,
  yOffDisp: number,
  thetaRad: number,
  cosRx: number,
  /** 世界 Z：将底边两端限制在南北栏之间，避免透视底边沿 Z 超出矩形 */
  clampZMin?: number,
  clampZMax?: number,
) {
  const halfWWorld = ((dw * 0.5 * dz) / FOCAL) * FENCE_EW_PERSP_WIDTH_MUL
  const c = Math.cos(thetaRad)
  const s = Math.sin(thetaRad)
  const dxL = -halfWWorld * c
  const dzL = -halfWWorld * s
  const dxR = halfWWorld * c
  const dzR = halfWWorld * s
  let wxL = wx + dxL
  let wzL = wz + dzL
  let wxR = wx + dxR
  let wzR = wz + dzR
  if (clampZMin !== undefined && clampZMax !== undefined) {
    const lo = Math.min(clampZMin, clampZMax)
    const hi = Math.max(clampZMin, clampZMax)
    const z0 = Math.min(wzL, wzR)
    const z1 = Math.max(wzL, wzR)
    const nz0 = Math.max(z0, lo)
    const nz1 = Math.min(z1, hi)
    if (nz0 >= nz1 - 1e-2) return
    wzL = nz0
    wzR = nz1
    wxL = wx
    wxR = wx
  }
  if (camZ - wzL <= 0 || camZ - wzR <= 0) return
  const pl = worldToScreen(wxL, wzL, camX, camZ)
  const pr = worldToScreen(wxR, wzR, camX, camZ)
  if (!pl || !pr) return
  const xbl = pl.sx
  const ybl = pl.sy - CROP_TOP + yOffDisp
  const xbr = pr.sx
  const ybr = pr.sy - CROP_TOP + yOffDisp
  if (Math.hypot(xbr - xbl, ybr - ybl) < 1e-2) return
  const dhEff = Math.max(4, dh * Math.max(0.06, Math.abs(cosRx)))
  const xtl = xbl
  const xtr = xbr
  const ytl = ybl - dhEff
  const ytr = ybr - dhEff
  const iw = img.naturalWidth || 1
  const ih = img.naturalHeight || 1
  drawTexturedTriangleAffine(ctx, img, 0, 0, iw, 0, iw, ih, xtl, ytl, xtr, ytr, xbr, ybr)
  drawTexturedTriangleAffine(ctx, img, 0, 0, iw, ih, 0, ih, xtl, ytl, xbr, ybr, xbl, ybl)
}

/** 透视：脚底 sy 越大离相机越近，应越晚画；layer 仅作同深度微调 */
function cityPropDepthKeyPerspective(
  t: CityPropDraw,
  camX: number,
  camZ: number,
  fenceEwTweak?: CityFenceEwLayoutTweak,
  ewZCenter: number = CITY_EW_FENCE_Z_CENTER,
): number | null {
  const { wx, wz } =
    fenceEwTweak && t.fenceEwUseAxisSliders
      ? fenceEwEffectiveWorld(t, fenceEwTweak, ewZCenter)
      : { wx: t.wx, wz: t.wz }
  const feet = worldToScreen(wx, wz, camX, camZ)
  if (!feet) return null
  if (camZ - wz <= 0) return null
  return feet.sy + t.layer * 1e-4
}

function cityPropDepthKeyTopdown(
  t: CityPropDraw,
  topCx: number,
  topCz: number,
  fenceEwTweak?: CityFenceEwLayoutTweak,
  ewZCenter: number = CITY_EW_FENCE_Z_CENTER,
): number {
  const { wx, wz } =
    fenceEwTweak && t.fenceEwUseAxisSliders
      ? fenceEwEffectiveWorld(t, fenceEwTweak, ewZCenter)
      : { wx: t.wx, wz: t.wz }
  const feet = worldToScreenTopdown(wx, wz, topCx, topCz)
  return feet.sy + t.layer * 1e-4
}

function drawCityPropPerspectiveOne(
  ctx: CanvasRenderingContext2D,
  t: CityPropDraw,
  camX: number,
  camZ: number,
  fenceEwTweak?: CityFenceEwLayoutTweak,
  ewZCenter: number = CITY_EW_FENCE_Z_CENTER,
  ewFenceClampZ?: { min: number; max: number },
) {
  if (!t.img.complete || t.img.naturalWidth < 2) return
  const ew = t.fenceEwUseAxisSliders && fenceEwTweak
  const { wx: pwx, wz: pwz } = ew
    ? fenceEwEffectiveWorld(t, fenceEwTweak, ewZCenter)
    : { wx: t.wx, wz: t.wz }
  const feet = worldToScreen(pwx, pwz, camX, camZ)
  if (!feet) return
  const dz = camZ - pwz
  if (dz <= 0) return
  const row = (FOCAL * CAM_HEIGHT) / dz
  const syDisp = feet.sy - CROP_TOP
  if (syDisp < -220 || syDisp > DISPLAY_H + 220) return
  if (feet.sx < -220 || feet.sx > W + 220) return
  const refRow = (FOCAL * CAM_HEIGHT) / t.refDz
  const iw = t.img.naturalWidth || 1
  const ih = t.img.naturalHeight || 1
  const sc = Math.max(0.2, Math.min(1.55, row / refRow)) * CITY_DISPLAY_SCALE * (t.scaleMul ?? 1)
  const wMul = ew ? fenceEwTweak.widthPct / 100 : 1
  const hMul = ew ? fenceEwTweak.heightPct / 100 : 1
  const dw = iw * sc * wMul
  const dh = ih * sc * hMul
  const yOff = t.feetDownSrcPx * sc * hMul
  ctx.save()
  ctx.imageSmoothingEnabled = false
  if (ew) {
    drawFenceEwPerspectiveYawQuad(
      ctx,
      t.img,
      camX,
      camZ,
      pwx,
      pwz,
      dz,
      dw,
      dh,
      yOff,
      FENCE_EW_YAW_RAD,
      1,
      ewFenceClampZ?.min,
      ewFenceClampZ?.max,
    )
  } else {
    const dw0 = iw * sc
    const dh0 = ih * sc
    const yOff0 = t.feetDownSrcPx * sc
    ctx.drawImage(t.img, feet.sx - dw0 * 0.5, syDisp - dh0 + yOff0, dw0, dh0)
  }
  ctx.restore()
}

function drawCityPropTopdownOne(
  ctx: CanvasRenderingContext2D,
  t: CityPropDraw,
  camX: number,
  camZ: number,
  fenceEwTweak?: CityFenceEwLayoutTweak,
  ewZCenter: number = CITY_EW_FENCE_Z_CENTER,
) {
  const k = TOPDOWN_PX_PER_WORLD
  if (!t.img.complete || t.img.naturalWidth < 2) return
  const ew = t.fenceEwUseAxisSliders && fenceEwTweak
  const { wx: pwx, wz: pwz } = ew
    ? fenceEwEffectiveWorld(t, fenceEwTweak, ewZCenter)
    : { wx: t.wx, wz: t.wz }
  const feet = worldToScreenTopdown(pwx, pwz, camX, camZ)
  const syDisp = feet.sy - CROP_TOP
  if (syDisp < -220 || syDisp > DISPLAY_H + 220) return
  if (feet.sx < -220 || feet.sx > W + 220) return
  const iw = t.img.naturalWidth || 1
  const ih = t.img.naturalHeight || 1
  const refTilePx = TILE_WORLD * k
  const sc =
    Math.max(0.16, Math.min(1.02, (refTilePx * 1.15) / ih)) * CITY_DISPLAY_SCALE * (t.scaleMul ?? 1)
  const wMul = ew ? fenceEwTweak.widthPct / 100 : 1
  const hMul = ew ? fenceEwTweak.heightPct / 100 : 1
  const dw = iw * sc * wMul
  const dh = ih * sc * hMul
  const yOff = t.feetDownSrcPx * sc * hMul
  ctx.save()
  ctx.imageSmoothingEnabled = false
  if (ew) {
    ctx.translate(feet.sx, syDisp)
    ctx.rotate(FENCE_EW_YAW_RAD)
    ctx.drawImage(t.img, -dw * 0.5, -dh + yOff, dw, dh)
  } else {
    const dw0 = iw * sc
    const dh0 = ih * sc
    const yOff0 = t.feetDownSrcPx * sc
    ctx.drawImage(t.img, feet.sx - dw0 * 0.5, syDisp - dh0 + yOff0, dw0, dh0)
  }
  ctx.restore()
}

function treeDepthKeyPerspective(wx: number, wz: number, camX: number, camZ: number): number | null {
  const feet = worldToScreen(wx, wz, camX, camZ)
  if (!feet) return null
  if (camZ - wz <= 0) return null
  return feet.sy
}

function drawSnowTreePerspectiveOne(
  ctx: CanvasRenderingContext2D,
  treeImg: HTMLImageElement,
  wx: number,
  wz: number,
  feetDownSrcPx: number,
  camX: number,
  camZ: number,
) {
  const refRow = (FOCAL * CAM_HEIGHT) / TREE_REF_DZ
  const iw = treeImg.naturalWidth || 1
  const ih = treeImg.naturalHeight || 1
  const feet = worldToScreen(wx, wz, camX, camZ)
  if (!feet) return
  const dz = camZ - wz
  if (dz <= 0) return
  const row = (FOCAL * CAM_HEIGHT) / dz
  const syDisp = feet.sy - CROP_TOP
  if (syDisp < -110 || syDisp > DISPLAY_H + 110) return
  if (feet.sx < -110 || feet.sx > W + 110) return
  const sc = Math.max(0.26, Math.min(1.28, row / refRow)) * TREE_DISPLAY_SCALE
  const dw = iw * sc
  const dh = ih * sc
  const yOff = feetDownSrcPx * sc
  ctx.save()
  ctx.imageSmoothingEnabled = false
  ctx.drawImage(treeImg, feet.sx - dw * 0.5, syDisp - dh + yOff, dw, dh)
  ctx.restore()
}

function treeDepthKeyTopdown(wx: number, wz: number, topCx: number, topCz: number): number {
  return worldToScreenTopdown(wx, wz, topCx, topCz).sy
}

function drawSnowTreeTopdownOne(
  ctx: CanvasRenderingContext2D,
  treeImg: HTMLImageElement,
  wx: number,
  wz: number,
  feetDownSrcPx: number,
  camX: number,
  camZ: number,
) {
  const k = TOPDOWN_PX_PER_WORLD
  const iw = treeImg.naturalWidth || 1
  const ih = treeImg.naturalHeight || 1
  const refTilePx = TILE_WORLD * k
  const sc = Math.max(0.2, Math.min(0.92, (refTilePx * 1.35) / ih)) * TREE_DISPLAY_SCALE
  const feet = worldToScreenTopdown(wx, wz, camX, camZ)
  const syDisp = feet.sy - CROP_TOP
  if (syDisp < -110 || syDisp > DISPLAY_H + 110) return
  if (feet.sx < -110 || feet.sx > W + 110) return
  const dw = iw * sc
  const dh = ih * sc
  const yOff = feetDownSrcPx * sc
  ctx.save()
  ctx.imageSmoothingEnabled = false
  ctx.drawImage(treeImg, feet.sx - dw * 0.5, syDisp - dh + yOff, dw, dh)
  ctx.restore()
}

function drawMonsterHutPerspectiveOne(
  ctx: CanvasRenderingContext2D,
  img: HTMLImageElement,
  wx: number,
  wz: number,
  feetDownSrcPx: number,
  camX: number,
  camZ: number,
) {
  const refRow = (FOCAL * CAM_HEIGHT) / MONSTER_HUT_REF_DZ
  const iw = img.naturalWidth || 1
  const ih = img.naturalHeight || 1
  const feet = worldToScreen(wx, wz, camX, camZ)
  if (!feet) return
  const dz = camZ - wz
  if (dz <= 0) return
  const row = (FOCAL * CAM_HEIGHT) / dz
  const syDisp = feet.sy - CROP_TOP
  if (syDisp < -130 || syDisp > DISPLAY_H + 130) return
  if (feet.sx < -130 || feet.sx > W + 130) return
  const sc = Math.max(0.22, Math.min(1.22, row / refRow)) * MONSTER_HUT_DISPLAY_SCALE
  const dw = iw * sc
  const dh = ih * sc
  const yOff = feetDownSrcPx * sc
  ctx.save()
  ctx.imageSmoothingEnabled = false
  ctx.drawImage(img, feet.sx - dw * 0.5, syDisp - dh + yOff, dw, dh)
  ctx.restore()
}

function drawMonsterHutTopdownOne(
  ctx: CanvasRenderingContext2D,
  img: HTMLImageElement,
  wx: number,
  wz: number,
  feetDownSrcPx: number,
  camX: number,
  camZ: number,
) {
  const k = TOPDOWN_PX_PER_WORLD
  const iw = img.naturalWidth || 1
  const ih = img.naturalHeight || 1
  const refTilePx = TILE_WORLD * k
  const sc = Math.max(0.18, Math.min(0.88, (refTilePx * 1.42) / ih)) * MONSTER_HUT_DISPLAY_SCALE
  const feet = worldToScreenTopdown(wx, wz, camX, camZ)
  const syDisp = feet.sy - CROP_TOP
  if (syDisp < -130 || syDisp > DISPLAY_H + 130) return
  if (feet.sx < -130 || feet.sx > W + 130) return
  const dw = iw * sc
  const dh = ih * sc
  const yOff = feetDownSrcPx * sc
  ctx.save()
  ctx.imageSmoothingEnabled = false
  ctx.drawImage(img, feet.sx - dw * 0.5, syDisp - dh + yOff, dw, dh)
  ctx.restore()
}

function monsterDepthKeyPerspective(m: MonsterInst, camX: number, camZ: number): number | null {
  const feet = worldToScreen(m.wx, m.wz, camX, camZ)
  if (!feet) return null
  if (camZ - m.wz <= 0) return null
  return feet.sy + 5e-3
}

function drawMonsterPerspectiveOne(
  ctx: CanvasRenderingContext2D,
  imgs: HTMLImageElement[],
  m: MonsterInst,
  camX: number,
  camZ: number,
) {
  const refRow = (FOCAL * CAM_HEIGHT) / MONSTER_REF_DZ
  const img = imgs[m.sheetIndex]
  if (!img?.complete || img.naturalWidth < MONSTER_COLS || img.naturalHeight < MONSTER_ROWS) return
  const cw = Math.floor(img.naturalWidth / MONSTER_COLS)
  const ch = Math.floor(img.naturalHeight / MONSTER_ROWS)
  if (cw < 1 || ch < 1) return
  const row = velocityToMonsterRow(m.vx, m.vz)
  const col = m.frame % MONSTER_COLS
  const feet = worldToScreen(m.wx, m.wz, camX, camZ)
  if (!feet) return
  const dz = camZ - m.wz
  if (dz <= 0) return
  const rowH = (FOCAL * CAM_HEIGHT) / dz
  const syDisp = feet.sy - CROP_TOP
  if (syDisp < -130 || syDisp > DISPLAY_H + 130) return
  if (feet.sx < -130 || feet.sx > W + 130) return
  const sc = Math.max(0.22, Math.min(1.35, rowH / refRow)) * MONSTER_DISPLAY_SCALE
  const dw = cw * sc
  const dh = ch * sc
  const yOff = MONSTER_FEET_DOWN_SRC * sc
  ctx.save()
  ctx.imageSmoothingEnabled = false
  ctx.drawImage(
    img,
    col * cw,
    row * ch,
    cw,
    ch,
    feet.sx - dw * 0.5,
    syDisp - dh + yOff,
    dw,
    dh,
  )
  ctx.restore()
}

/** 树根世界坐标对齐地块上的点；贴地偏移为额外下移（纹理像素 × 透视缩放 sc），修正图底部透明留白 */
const TREE_FEET_DOWN_MIN = -24
const TREE_FEET_DOWN_MAX = 40

/**
 * 全屏后处理：先可选半透明黑层压暗，再叠径向渐变暗角（中心透明、边缘变暗）。
 * `dimPct` / `vignettePct` 为 0–100，来自 UI 滑条 ref。
 */
function drawScenePostFx(
  ctx: CanvasRenderingContext2D,
  cw: number,
  ch: number,
  dimPct: number,
  vignettePct: number,
) {
  const d = Math.max(0, Math.min(100, dimPct))
  const v = Math.max(0, Math.min(100, vignettePct))
  if (d < 0.5 && v < 0.5) return
  ctx.save()
  if (d > 0.5) {
    const a = (d / 100) * 0.62
    ctx.fillStyle = `rgba(0,0,0,${a})`
    ctx.fillRect(0, 0, cw, ch)
  }
  if (v > 0.5) {
    const cx = cw * 0.5
    const cy = ch * 0.5
    /** 内圈尽量小，外圈盖住四角，暗角才明显 */
    const r0 = Math.min(cw, ch) * 0.1
    const r1 = Math.hypot(cw * 0.52, ch * 0.52)
    const t = Math.pow(v / 100, 0.92)
    const edgeA = Math.min(0.94, t * 0.93)
    const g = ctx.createRadialGradient(cx, cy, r0, cx, cy, r1)
    g.addColorStop(0, 'rgba(0,0,0,0)')
    g.addColorStop(0.32, 'rgba(0,0,0,0)')
    g.addColorStop(0.62, `rgba(0,0,0,${edgeA * 0.22})`)
    g.addColorStop(0.82, `rgba(0,0,0,${edgeA * 0.62})`)
    g.addColorStop(1, `rgba(0,0,0,${edgeA})`)
    ctx.fillStyle = g
    ctx.fillRect(0, 0, cw, ch)
  }
  ctx.restore()
}

function createFallSnowFlakes(camX: number, camZ: number): FallFlake[] {
  const flakes: FallFlake[] = []
  for (let i = 0; i < FALL_SNOW_COUNT; i++) {
    const roll = Math.random()
    const layer: FallLayer = roll < 0.48 ? 0 : roll < 0.82 ? 1 : 2
    flakes.push(respawnFallFlake(camX, camZ, layer, 'snow'))
  }
  return flakes
}

function createFallPetals(camX: number, camZ: number): FallFlake[] {
  const flakes: FallFlake[] = []
  for (let i = 0; i < FALL_SNOW_COUNT; i++) {
    const roll = Math.random()
    const layer: FallLayer = roll < 0.48 ? 0 : roll < 0.82 ? 1 : 2
    flakes.push(respawnFallFlake(camX, camZ, layer, 'petal'))
  }
  return flakes
}

function createFallLeaves(camX: number, camZ: number): FallFlake[] {
  const flakes: FallFlake[] = []
  for (let i = 0; i < FALL_SNOW_COUNT; i++) {
    const roll = Math.random()
    const layer: FallLayer = roll < 0.48 ? 0 : roll < 0.82 ? 1 : 2
    flakes.push(respawnFallFlake(camX, camZ, layer, 'leaf'))
  }
  return flakes
}

/**
 * 无限地图 UI + Canvas：所有交互状态在此声明，耗时逻辑在 `gameLoop`（`ready` 后每帧）。
 */
export default function InfiniteMapScene() {
  const { t } = useLanguage()
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const offRef = useRef<HTMLCanvasElement | null>(null)
  const keysRef = useRef<Set<string>>(new Set())
  const posRef = useRef({ x: 0, z: 400 })
  const animRef = useRef({ name: 'idledown', frameIdx: 0, accum: 0 })
  const facingRef = useRef(-1)
  const rafRef = useRef(0)
  const lastTimeRef = useRef(0)
  const frameMapRef = useRef<Map<string, HTMLCanvasElement>>(new Map())
  const blobWorldRef = useRef<BlobWorld | null>(null)
  const blobAtlasMtnRef = useRef<BlobAtlas | null>(null)
  const blobAtlasNormRef = useRef<BlobAtlas | null>(null)
  const blobAtlasNormX1Ref = useRef<BlobAtlas | null>(null)
  const blobAtlasNormX2Ref = useRef<BlobAtlas | null>(null)
  /** 与 map.html 一致：水用 007；可选 000–007 多帧动画 */
  const blobWaterFramesRef = useRef<(BlobAtlas | null)[]>([])
  const [ready, setReady] = useState(false)
  const [musicOn, setMusicOn] = useState(true)
  const [showTerrainLabels, setShowTerrainLabels] = useState(false)
  const [editMode, setEditMode] = useState(false)
  const [terrainTextureSet, setTerrainTextureSet] = useState<InfiniteMapTerrainTextureSetId>('blob')
  const [terrainSeaPct, setTerrainSeaPct] = useState(25)
  const [terrainMtnPct, setTerrainMtnPct] = useState(56)
  const [fxDimPct, setFxDimPct] = useState(12)
  const [fxVignettePct, setFxVignettePct] = useState(100)
  const [treePatchDensityPct, setTreePatchDensityPct] = useState(85)
  const [treeLoneDensityPct, setTreeLoneDensityPct] = useState(45)
  const [treeFeetDownSrcPx, setTreeFeetDownSrcPx] = useState(8)
  const [monsterCount, setMonsterCount] = useState(24)
  const [monsterAssetsReady, setMonsterAssetsReady] = useState(false)
  const fxDimRef = useRef(12)
  const fxVignetteRef = useRef(100)
  /** 成片林、独立树密度（0–100），供 rAF 循环读取 */
  const treePatchDensityRef = useRef(85)
  const treeLoneDensityRef = useRef(45)
  const treeFeetDownSrcPxRef = useRef(8)
  const monsterCountRef = useRef(24)
  const monsterImgsRef = useRef<HTMLImageElement[]>([])
  const monstersRef = useRef<MonsterInst[]>([])
  const showTerrainLabelsRef = useRef(false)
  const editModeRef = useRef(false)
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const gameWrapRef = useRef<HTMLDivElement>(null)
  const fallSnowRef = useRef<FallFlake[] | null>(null)
  /** 与 `fallSnowRef` 中粒子对应的地形套，切换贴图集时重建粒子 */
  const fallParticlesTerrainRef = useRef<InfiniteMapTerrainTextureSetId | null>(null)
  const treeSnowImgRef = useRef<HTMLImageElement | null>(null)
  const treeGrassImgRef = useRef<HTMLImageElement | null>(null)
  const monsterHutImgRef = useRef<HTMLImageElement | null>(null)
  const monsterHutSitesRef = useRef<{ wx: number; wz: number }[]>([])
  const terrainTextureSetRef = useRef<InfiniteMapTerrainTextureSetId>('blob')
  const cityImgsRef = useRef<Map<string, HTMLImageElement>>(new Map())
  const cityGenRef = useRef<CityGenParams>({ ...CITY_GEN_DEFAULTS })
  const cityLayoutRef = useRef<CityPropDef[]>(buildCityPropLayout(CITY_GEN_DEFAULTS))
  const townPlaceRef = useRef<{ placeX: number; placeZ: number }>({
    placeX: 0,
    placeZ: CITY_GEN_DEFAULTS.ewFenceZCenter,
  })
  const [cityGen, setCityGen] = useState<CityGenParams>(() => ({ ...CITY_GEN_DEFAULTS }))
  const [townPlaceNonce, setTownPlaceNonce] = useState(0)
  /** 与随机地图联动，刷新兽人小屋世界坐标 */
  const [hutSitesNonce, setHutSitesNonce] = useState(0)
  const fenceEwLayoutRef = useRef<CityFenceEwLayoutTweak>({ ...FENCE_EW_LAYOUT_FIXED })
  const [displayScale, setDisplayScale] = useState(1)

  useLayoutEffect(() => {
    cityGenRef.current = cityGen
    const bw = blobWorldRef.current
    const local = buildCityPropLayout(cityGen)
    const px = posRef.current.x
    const pz = posRef.current.z
    const tw = TILE_WORLD
    let place: { placeX: number; placeZ: number }
    if (bw && ready) {
      const ringShuffle = townPlaceNonce > 0 ? townPlaceNonce * 494_137 + cityGen.seed * 1_009 : undefined
      const found = findNearestFlatSnowTownPlacement(bw, cityGen, tw, px, pz, 160, ringShuffle)
      place =
        found ?? {
          placeX: (Math.floor(px / tw) + 0.5) * tw,
          placeZ: (Math.floor(pz / tw) + 0.5) * tw,
        }
    } else {
      place = { placeX: px, placeZ: pz }
    }
    townPlaceRef.current = place
    cityLayoutRef.current = offsetCityPropLayout(local, place.placeX, place.placeZ, cityGen.ewFenceZCenter)
  }, [cityGen, ready, terrainSeaPct, terrainMtnPct, townPlaceNonce])

  useLayoutEffect(() => {
    const bw = blobWorldRef.current
    if (!bw || !ready) return
    const cg = cityGenRef.current
    const tp = townPlaceRef.current
    const ewZ = cg.ewFenceZCenter
    const dzPlace = tp.placeZ - ewZ
    const extZ = ewFenceExtentZ(cg)
    const ewClamp = { min: extZ.zSouth + dzPlace, max: extZ.zNorth + dzPlace }
    monsterHutSitesRef.current = buildMonsterHutSites(
      bw,
      hutSitesNonce,
      cityLayoutRef.current,
      cg,
      tp.placeX,
      tp.placeZ,
      fenceEwLayoutRef.current,
      ewClamp,
    )
  }, [ready, hutSitesNonce, townPlaceNonce, cityGen, terrainSeaPct, terrainMtnPct])

  useLayoutEffect(() => {
    const el = gameWrapRef.current
    if (!el) return
    const update = () => {
      const r = el.getBoundingClientRect()
      const sx = r.width / W
      const sy = r.height / DISPLAY_H
      const s = Math.min(sx, sy)
      setDisplayScale(Number.isFinite(s) && s > 0 ? s : 1)
    }
    update()
    const ro = new ResizeObserver(() => update())
    ro.observe(el)
    return () => ro.disconnect()
  }, [])

  useEffect(() => {
    const audio = new Audio(BGM_URL)
    audio.loop = true
    audio.preload = 'auto'
    audioRef.current = audio
    return () => {
      audio.pause()
      audio.src = ''
      audioRef.current = null
    }
  }, [])

  useEffect(() => {
    const audio = audioRef.current
    if (!audio) return
    if (musicOn) {
      void audio.play().catch(() => {})
    } else {
      audio.pause()
    }
  }, [musicOn])

  useEffect(() => {
    showTerrainLabelsRef.current = showTerrainLabels
  }, [showTerrainLabels])

  useEffect(() => {
    editModeRef.current = editMode
  }, [editMode])

  useEffect(() => {
    terrainTextureSetRef.current = terrainTextureSet
  }, [terrainTextureSet])

  useEffect(() => {
    monsterCountRef.current = monsterCount
  }, [monsterCount])

  useEffect(() => {
    const cancel = loadMonsterImages((imgs) => {
      monsterImgsRef.current = imgs
      setMonsterAssetsReady(true)
    })
    return () => {
      cancel()
      monsterImgsRef.current = []
      monstersRef.current = []
      setMonsterAssetsReady(false)
    }
  }, [])

  const rebuildMonsterSwarm = useCallback(() => {
    const w = blobWorldRef.current
    const imgs = monsterImgsRef.current
    if (!w || imgs.length === 0) {
      monstersRef.current = []
      return
    }
    const n = Math.max(0, Math.min(MONSTER_COUNT_MAX, Math.floor(monsterCountRef.current)))
    monstersRef.current = createMonsterSwarm(n, w, posRef.current.x, posRef.current.z, imgs.length, TILE_WORLD)
  }, [])

  useEffect(() => {
    if (!ready || !monsterAssetsReady) return
    rebuildMonsterSwarm()
  }, [ready, monsterAssetsReady, monsterCount, rebuildMonsterSwarm])

  useEffect(() => {
    blobWorldRef.current = new BlobWorld('ronin-2026')
    return () => {
      blobWorldRef.current?.clearCache()
      blobWorldRef.current = null
    }
  }, [])

  useEffect(() => {
    const w = blobWorldRef.current
    if (!w) return
    w.setParams(terrainSeaPct / 100, terrainMtnPct / 100)
    w.clearCache()
  }, [terrainSeaPct, terrainMtnPct])

  /** 出生点若落在水或山上，挪到最近平地 */
  useEffect(() => {
    if (!ready) return
    const w = blobWorldRef.current
    if (!w) return
    const tix = Math.floor(posRef.current.x / TILE_WORLD)
    const tiz = Math.floor(posRef.current.z / TILE_WORLD)
    if (!isBlobTileWalkable(w, tix, tiz)) {
      const p =
        findWalkableTileCenter(w, tix, tiz, 240, TILE_WORLD) ??
        findWalkableTileCenter(w, 0, 0, 400, TILE_WORLD)
      if (p) {
        posRef.current.x = p.wx
        posRef.current.z = p.wz
      }
    }
  }, [ready])

  const handleRandomMap = useCallback(() => {
    const w = blobWorldRef.current
    if (!w) return
    const seed = `w${Math.random().toString(36).slice(2, 11)}${Date.now().toString(36)}`
    w.reseed(seed)
    w.setParams(terrainSeaPct / 100, terrainMtnPct / 100)
    const ox = Math.floor(posRef.current.x / TILE_WORLD)
    const oz = Math.floor(posRef.current.z / TILE_WORLD)
    const p =
      findWalkableTileCenter(w, ox, oz, 260, TILE_WORLD) ??
      findWalkableTileCenter(w, 0, 0, 400, TILE_WORLD)
    if (p) {
      posRef.current.x = p.wx
      posRef.current.z = p.wz
    }
    rebuildMonsterSwarm()
    setTownPlaceNonce((n) => n + 1)
    setHutSitesNonce((n) => n + 1)
  }, [terrainSeaPct, terrainMtnPct, rebuildMonsterSwarm])

  useEffect(() => {
    const img = new Image()
    img.onload = () => {
      treeSnowImgRef.current = img
    }
    img.src = TREE_SNOW_URL
    return () => {
      if (treeSnowImgRef.current === img) treeSnowImgRef.current = null
    }
  }, [])

  useEffect(() => {
    const img = new Image()
    img.onload = () => {
      treeGrassImgRef.current = img
    }
    img.src = TREE_GRASS_URL
    return () => {
      if (treeGrassImgRef.current === img) treeGrassImgRef.current = null
    }
  }, [])

  useEffect(() => {
    const img = new Image()
    img.onload = () => {
      monsterHutImgRef.current = img
    }
    img.src = MONSTER_HUT_URL
    return () => {
      if (monsterHutImgRef.current === img) monsterHutImgRef.current = null
    }
  }, [])

  useEffect(() => {
    const cancel = loadCityPropImages((m) => {
      cityImgsRef.current = m
    })
    return () => {
      cancel()
      cityImgsRef.current = new Map()
    }
  }, [])

  /** Blob 图集 3×24：按所选地形套加载 URL（分区/blob 逻辑不变，仅贴图源切换） */
  useEffect(() => {
    let cancelled = false
    const urls = terrainTextureUrls(terrainTextureSet)
    blobWaterFramesRef.current = urls.water.map(() => null)

    const loadTo = (url: string, onOk: (a: BlobAtlas) => void) => {
      const img = new Image()
      img.onload = () => {
        if (cancelled) return
        const atlas = decodeBlobAtlasFromImage(img)
        if (atlas) onOk(atlas)
      }
      img.src = url
    }

    blobAtlasMtnRef.current = null
    blobAtlasNormRef.current = null
    blobAtlasNormX1Ref.current = null
    blobAtlasNormX2Ref.current = null

    loadTo(urls.mtn, (a) => {
      blobAtlasMtnRef.current = a
    })
    loadTo(urls.norm, (a) => {
      blobAtlasNormRef.current = a
    })
    loadTo(urls.normX1, (a) => {
      blobAtlasNormX1Ref.current = a
    })
    loadTo(urls.normX2, (a) => {
      blobAtlasNormX2Ref.current = a
    })

    urls.water.forEach((url, idx) => {
      loadTo(url, (a) => {
        const next = blobWaterFramesRef.current.slice()
        next[idx] = a
        blobWaterFramesRef.current = next
      })
    })

    return () => {
      cancelled = true
      blobAtlasMtnRef.current = null
      blobAtlasNormRef.current = null
      blobAtlasNormX1Ref.current = null
      blobAtlasNormX2Ref.current = null
      blobWaterFramesRef.current = urls.water.map(() => null)
    }
  }, [terrainTextureSet])

  /** 仅加载默认精灵表 map/TINA.png（与 topdown 同款切帧布局） */
  useEffect(() => {
    let cancelled = false
    const img = new Image()
    img.onload = () => {
      if (cancelled) return
      const map = new Map<string, HTMLCanvasElement>()
      for (const key of Object.keys(REGIONS)) {
        const c = extractFrame(img, key)
        if (c) map.set(key, c)
      }
      frameMapRef.current = map
      setReady(true)
    }
    img.src = DEFAULT_CHAR_URL
    return () => {
      cancelled = true
      setReady(false)
    }
  }, [])

  useEffect(() => {
    const kd = (e: KeyboardEvent) => {
      if (['KeyW', 'KeyA', 'KeyS', 'KeyD', 'ShiftLeft', 'ShiftRight'].includes(e.code)) {
        e.preventDefault()
      }
      keysRef.current.add(e.code)
    }
    const ku = (e: KeyboardEvent) => {
      keysRef.current.delete(e.code)
    }
    window.addEventListener('keydown', kd)
    window.addEventListener('keyup', ku)
    return () => {
      window.removeEventListener('keydown', kd)
      window.removeEventListener('keyup', ku)
    }
  }, [])

  /**
   * 主循环：输入 → 地形光栅 → 深度批处理 → 粒子 → 后效 → 动画计数。
   * 依赖数组仅 `[ready]`：其余一律读 ref，避免每改一个滑条就重建 RAF。
   */
  const gameLoop = useCallback(
    (now?: number) => {
      rafRef.current = requestAnimationFrame(gameLoop)
      const canvas = canvasRef.current
      if (!canvas || !ready) return
      const ctx = canvas.getContext('2d')
      if (!ctx) return
      ctx.imageSmoothingEnabled = false

      let off = offRef.current
      if (!off || off.width !== W || off.height !== H) {
        off = document.createElement('canvas')
        off.width = W
        off.height = H
        offRef.current = off
      }
      const octx = off.getContext('2d')!
      const imageData = octx.createImageData(W, H)
      const data = imageData.data

      const t0 = typeof now === 'number' ? now : performance.now()
      const dt = lastTimeRef.current ? Math.min((t0 - lastTimeRef.current) / 1000, 1 / 15) : 1 / 60
      lastTimeRef.current = t0

      const keys = keysRef.current
      const pos = posRef.current
      const bwMove = blobWorldRef.current
      const w = keys.has('KeyW')
      const a = keys.has('KeyA')
      const s = keys.has('KeyS')
      const d = keys.has('KeyD')
      const shift = keys.has('ShiftLeft') || keys.has('ShiftRight')
      const speed = MOVE_SPEED * (shift ? RUN_MUL : 1)
      const walkPrefix = shift ? 'run' : 'walk'

      let nx = pos.x
      let nz = pos.z
      let nextAnim: string = animRef.current.name
      // 地图水平面相对旧版转 180° 后，WASD 与屏幕直觉一致需取反位移
      if (w && !s) {
        nextAnim = `${walkPrefix}up`
        nz -= speed
      } else if (s && !w) {
        nextAnim = `${walkPrefix}down`
        nz += speed
      } else if (a && !d) {
        nextAnim = `${walkPrefix}L`
        facingRef.current = -1
        nx += speed
      } else if (d && !a) {
        nextAnim = `${walkPrefix}L`
        facingRef.current = 1
        nx -= speed
      } else {
        nextAnim = facingRef.current === -1 ? 'idleL' : 'idledown'
      }

      if (bwMove) {
        const tix = (x: number) => Math.floor(x / TILE_WORLD)
        const tiz = (z: number) => Math.floor(z / TILE_WORLD)
        const cgMove = cityGenRef.current
        const tpMove = townPlaceRef.current
        const ewZMove = cgMove.ewFenceZCenter
        const dzPlMove = tpMove.placeZ - ewZMove
        const extZMove = ewFenceExtentZ(cgMove)
        const ewClampMove = { min: extZMove.zSouth + dzPlMove, max: extZMove.zNorth + dzPlMove }
        const canStep = (wx: number, wz: number) =>
          isBlobTileWalkable(bwMove, tix(wx), tiz(wz)) &&
          !worldPosBlockedByCityFence(
            wx,
            wz,
            cityLayoutRef.current,
            ewZMove,
            fenceEwLayoutRef.current,
            ewClampMove,
          )
        if (canStep(nx, nz)) {
          pos.x = nx
          pos.z = nz
        } else if (canStep(nx, pos.z)) {
          pos.x = nx
        } else if (canStep(pos.x, nz)) {
          pos.z = nz
        }
      } else {
        pos.x = nx
        pos.z = nz
      }

      let anim = animRef.current
      if (nextAnim !== anim.name) {
        anim = { name: nextAnim, frameIdx: 0, accum: 0 }
        animRef.current = anim
      }

      const camX = pos.x
      const camZ = pos.z + PLAYER_DZ
      const edit = editModeRef.current
      /** 俯视以角色脚底为画面中心；透视相机在 pos.z + PLAYER_DZ，朝 -Z（地图相对旧版转 180°） */
      const topCx = pos.x
      const topCz = pos.z

      const bwPre = blobWorldRef.current
      if (bwPre) {
        const vb = edit ? visibleGroundTileBoundsTopdown(topCx, topCz) : visibleGroundTileBounds(camX, camZ)
        if (vb) bwPre.preloadTileAABB(vb.tix0, vb.tix1, vb.tiz0, vb.tiz1)
        else
          bwPre.preloadAroundTile(
            Math.floor((edit ? topCx : camX) / TILE_WORLD),
            Math.floor((edit ? topCz : camZ) / TILE_WORLD),
            48,
          )
      }

      const tilePickCache = new Map<string, BlobTilePick>()
      const bw = blobWorldRef.current
      const mis = monsterImgsRef.current
      const mons = monstersRef.current
      if (!edit && bw && mis.length > 0 && mons.length > 0) {
        for (const m of mons) stepMonster(m, bw, dt, TILE_WORLD)
      }

      let p = 0
      for (let sy = 0; sy < H; sy++) {
        for (let sx = 0; sx < W; sx++) {
          if (sy < HORIZON) {
            const g = sy / Math.max(1, HORIZON - 1)
            const v = Math.floor(18 + g * 40)
            data[p++] = v
            data[p++] = v
            data[p++] = v
            data[p++] = 255
          } else if (sy < TERRAIN_SY_START) {
            const [hr, hg, hb] = farHazeRgb(sy)
            data[p++] = hr
            data[p++] = hg
            data[p++] = hb
            data[p++] = 255
          } else {
            const hit = (edit ? screenToWorldTopdown : screenToWorld)(
              sx + 0.5,
              sy + 0.5,
              edit ? topCx : camX,
              edit ? topCz : camZ,
            )
            if (!hit) {
              data[p++] = 12
              data[p++] = 12
              data[p++] = 12
              data[p++] = 255
              continue
            }
            const tix = Math.floor(hit.wx / TILE_WORLD)
            const tiz = Math.floor(hit.wz / TILE_WORLD)
            const tKey = `${tix},${tiz}`
            let pick = tilePickCache.get(tKey)
            if (bw && pick === undefined) {
              pick = bw.sampleTileIndex(tix, tiz)
              tilePickCache.set(tKey, pick)
            }
            const [cr, cg, cb] =
              bw && pick !== undefined
                ? sampleBlobTerrainRgbResolved(
                    blobAtlasMtnRef.current,
                    blobAtlasNormRef.current,
                    blobAtlasNormX1Ref.current,
                    blobAtlasNormX2Ref.current,
                    blobWaterFramesRef.current,
                    t0,
                    pick,
                    tix,
                    tiz,
                    hit.wx,
                    hit.wz,
                  )
                : mosaicRgb(tix, tiz)
            data[p++] = cr
            data[p++] = cg
            data[p++] = cb
            data[p++] = 255
          }
        }
      }

      octx.putImageData(imageData, 0, 0)
      ctx.setTransform(1, 0, 0, 1, 0, 0)
      ctx.fillStyle = '#080a0f'
      ctx.fillRect(0, 0, canvas.width, canvas.height)
      ctx.drawImage(off, 0, CROP_TOP, W, DISPLAY_H, 0, 0, canvas.width, canvas.height)

      const depthBatch: DepthSprite[] = []
      const cityImgMap = cityImgsRef.current
      if (cityImgMap.size > 0) {
        const cg = cityGenRef.current
        const tp = townPlaceRef.current
        const ewZCenter = cg.ewFenceZCenter
        const dzPlace = tp.placeZ - ewZCenter
        const extZ = ewFenceExtentZ(cg)
        const ewFenceClampZ = { min: extZ.zSouth + dzPlace, max: extZ.zNorth + dzPlace }
        for (const def of cityLayoutRef.current) {
          const img = cityImgMap.get(def.file)
          if (!img?.complete || img.naturalWidth < 2) continue
          const item: CityPropDraw = { ...def, img }
          const ewTw = item.fenceEwUseAxisSliders ? fenceEwLayoutRef.current : undefined
          if (edit) {
            depthBatch.push({
              key: cityPropDepthKeyTopdown(item, topCx, topCz, ewTw, ewZCenter),
              draw: () => drawCityPropTopdownOne(ctx, item, topCx, topCz, ewTw, ewZCenter),
            })
          } else {
            const k = cityPropDepthKeyPerspective(item, camX, camZ, ewTw, ewZCenter)
            if (k === null) continue
            depthBatch.push({
              key: k,
              draw: () =>
                drawCityPropPerspectiveOne(
                  ctx,
                  item,
                  camX,
                  camZ,
                  ewTw,
                  ewZCenter,
                  ewTw ? ewFenceClampZ : undefined,
                ),
            })
          }
        }
      }

      const texSet = terrainTextureSetRef.current
      const treeImg =
        texSet === 'blob' ? treeSnowImgRef.current : treeGrassImgRef.current
      const treePlaces: { wx: number; wz: number }[] = []
      if (treeImg && treeImg.complete && treeImg.naturalWidth > 0 && bw) {
        let vbTrees = edit ? visibleGroundTileBoundsTopdown(topCx, topCz) : visibleGroundTileBounds(camX, camZ)
        if (!vbTrees) {
          const tcx = Math.floor((edit ? topCx : camX) / TILE_WORLD)
          const tcz = Math.floor((edit ? topCz : camZ) / TILE_WORLD)
          const sp = 26
          vbTrees = { tix0: tcx - sp, tix1: tcx + sp, tiz0: tcz - sp, tiz1: tcz + sp }
        }
        for (let tiz = vbTrees.tiz0; tiz <= vbTrees.tiz1; tiz++) {
          for (let tix = vbTrees.tix0; tix <= vbTrees.tix1; tix++) {
            if (!isBlobTileLandNotWater(bw, tix, tiz)) continue
            if (!shouldPlaceSnowTreeOnTile(tix, tiz, treePatchDensityRef.current, treeLoneDensityRef.current))
              continue
            const { wx, wz } = snowTreeFeetWorld(tix, tiz)
            const cg = cityGenRef.current
            const tp = townPlaceRef.current
            if (isInsidePlacedTownFootprint(wx, wz, cg, tp.placeX, tp.placeZ)) continue
            treePlaces.push({ wx, wz })
          }
        }
        const fd = treeFeetDownSrcPxRef.current
        for (const tr of treePlaces) {
          if (edit) {
            depthBatch.push({
              key: treeDepthKeyTopdown(tr.wx, tr.wz, topCx, topCz),
              draw: () => drawSnowTreeTopdownOne(ctx, treeImg, tr.wx, tr.wz, fd, topCx, topCz),
            })
          } else {
            const k = treeDepthKeyPerspective(tr.wx, tr.wz, camX, camZ)
            if (k === null) continue
            depthBatch.push({
              key: k,
              draw: () => drawSnowTreePerspectiveOne(ctx, treeImg, tr.wx, tr.wz, fd, camX, camZ),
            })
          }
        }
      }

      const hutImg = monsterHutImgRef.current
      const hutSites = monsterHutSitesRef.current
      if (hutImg?.complete && hutImg.naturalWidth > 0 && hutSites.length > 0) {
        const fdh = MONSTER_HUT_FEET_DOWN_SRC_PX
        for (let hi = 0; hi < hutSites.length; hi++) {
          const hut = hutSites[hi]!
          const tie = hi * 1e-5
          if (edit) {
            depthBatch.push({
              key: treeDepthKeyTopdown(hut.wx, hut.wz, topCx, topCz) + tie,
              draw: () => drawMonsterHutTopdownOne(ctx, hutImg, hut.wx, hut.wz, fdh, topCx, topCz),
            })
          } else {
            const k = treeDepthKeyPerspective(hut.wx, hut.wz, camX, camZ)
            if (k === null) continue
            depthBatch.push({
              key: k + tie,
              draw: () => drawMonsterHutPerspectiveOne(ctx, hutImg, hut.wx, hut.wz, fdh, camX, camZ),
            })
          }
        }
      }

      if (!edit && mis.length > 0 && mons.length > 0) {
        for (const m of mons) {
          const k = monsterDepthKeyPerspective(m, camX, camZ)
          if (k === null) continue
          depthBatch.push({
            key: k,
            draw: () => drawMonsterPerspectiveOne(ctx, mis, m, camX, camZ),
          })
        }
      }

      const aDef = ANIMS.find((x) => x.name === anim.name) ?? ANIMS.find((x) => x.name === 'idledown')!
      const frameKey = aDef.frames[anim.frameIdx % aDef.frames.length]!
      const frameCanvas = frameMapRef.current.get(frameKey)

      if (frameCanvas) {
        if (edit) {
          const feet = worldToScreenTopdown(pos.x, pos.z, topCx, topCz)
          depthBatch.push({
            key: feet.sy + 0.015,
            draw: () => {
              const { sx, sy } = feet
              const syDisp = sy - CROP_TOP
              const fw = frameCanvas.width
              const fh = frameCanvas.height
              const playerSc = 0.5
              const shadowTex = getShadowTexture()
              const shw = shadowTex.width * playerSc
              const shh = shadowTex.height * playerSc
              ctx.save()
              ctx.translate(sx, syDisp - 2)
              ctx.drawImage(shadowTex, -shw / 2, -shh / 2, shw, shh)
              ctx.restore()

              ctx.save()
              ctx.translate(sx, syDisp)
              ctx.scale(-facingRef.current * playerSc, playerSc)
              ctx.drawImage(frameCanvas, -fw / 2, -fh, fw, fh)
              ctx.restore()
            },
          })
        } else {
          const feet = worldToScreen(pos.x, pos.z, camX, camZ)
          if (feet) {
            depthBatch.push({
              key: feet.sy + 0.015,
              draw: () => {
                const { sx, sy } = feet
                const syDisp = sy - CROP_TOP
                const fw = frameCanvas.width
                const fh = frameCanvas.height
                const shadowTex = getShadowTexture()
                const shw = shadowTex.width * 0.85
                const shh = shadowTex.height * 0.85
                ctx.save()
                ctx.translate(sx, syDisp - 2)
                ctx.scale(0.85, 0.85)
                ctx.drawImage(shadowTex, -shw / 2, -shh / 2, shw, shh)
                ctx.restore()

                ctx.save()
                ctx.translate(sx, syDisp)
                ctx.scale(-facingRef.current, 1)
                ctx.drawImage(frameCanvas, -fw / 2, -fh, fw, fh)
                ctx.restore()
              },
            })
          }
        }
      }

      depthBatch.sort((a, b) => a.key - b.key)
      for (const d of depthBatch) d.draw()

      let flakes = fallSnowRef.current
      if (!edit) {
        const particleKind = fallParticleKindForTerrain(texSet)
        if (
          !flakes ||
          flakes.length !== FALL_SNOW_COUNT ||
          fallParticlesTerrainRef.current !== texSet
        ) {
          flakes =
            particleKind === 'snow'
              ? createFallSnowFlakes(camX, camZ)
              : particleKind === 'petal'
                ? createFallPetals(camX, camZ)
                : createFallLeaves(camX, camZ)
          fallSnowRef.current = flakes
          fallParticlesTerrainRef.current = texSet
        }
        for (let i = 0; i < flakes.length; i++) {
          flakes[i] = updateFallFlake(flakes[i]!, camX, camZ, dt, t0, particleKind)
        }
        ctx.save()
        ctx.imageSmoothingEnabled = false
        ctx.beginPath()
        ctx.rect(0, 0, W, DISPLAY_H)
        ctx.clip()
        for (const f of flakes) {
          const scr = worldToScreen(f.wx, f.wz, camX, camZ)
          if (!scr || scr.sy < HORIZON - 6) continue
          const xi = Math.floor(scr.sx)
          const yi = Math.floor(scr.sy - CROP_TOP + f.pixelFallY)
          if (yi < -8 || yi > DISPLAY_H + 8 || xi < -8 || xi > W + 8) continue
          if (particleKind === 'petal') {
            const rgb = rgbForPetalFlake(f.layer, f.wobble)
            if (f.layer === 0) drawPetalFar(ctx, xi, yi, rgb)
            else if (f.layer === 1) drawPetalMid(ctx, xi, yi, rgb)
            else drawPetalNear(ctx, xi, yi, rgb, f.variant, W, DISPLAY_H)
          } else if (particleKind === 'leaf') {
            const rgb = rgbForLeafFlake(f.layer, f.wobble)
            if (f.layer === 0) drawLeafFar(ctx, xi, yi, rgb)
            else if (f.layer === 1) drawLeafMid(ctx, xi, yi, rgb)
            else drawLeafNear(ctx, xi, yi, rgb, f.variant, W, DISPLAY_H)
          } else {
            const rgb = rgbForSnowFlake(f.layer, f.wobble)
            if (f.layer === 0) drawSnowFar(ctx, xi, yi, rgb)
            else if (f.layer === 1) drawSnowMid(ctx, xi, yi, rgb)
            else drawSnowNear(ctx, xi, yi, rgb, f.variant, W, DISPLAY_H)
          }
        }
        ctx.restore()
      }

      if (showTerrainLabelsRef.current) {
        const tcx = edit ? topCx : camX
        const tcz = edit ? topCz : camZ
        const span = 28
        const tix0 = Math.floor(tcx / TILE_WORLD) - span
        const tix1 = Math.floor(tcx / TILE_WORLD) + span
        const tiz0 = Math.floor(tcz / TILE_WORLD) - span
        const tiz1 = Math.floor(tcz / TILE_WORLD) + span
        ctx.save()
        ctx.imageSmoothingEnabled = false
        ctx.font = 'bold 9px monospace'
        ctx.textAlign = 'center'
        ctx.textBaseline = 'middle'
        for (let tiz = tiz0; tiz <= tiz1; tiz++) {
          for (let tix = tix0; tix <= tix1; tix++) {
            const wx = (tix + 0.5) * TILE_WORLD
            const wz = (tiz + 0.5) * TILE_WORLD
            const scr = edit ? worldToScreenTopdown(wx, wz, tcx, tcz) : worldToScreen(wx, wz, tcx, tcz)
            if (!scr) continue
            const y = scr.sy - CROP_TOP
            if (y < -10 || y > DISPLAY_H + 10) continue
            if (scr.sx < -30 || scr.sx > W + 30) continue
            const label = blobWorldRef.current
              ? describeBlobTerrain(blobWorldRef.current, tix, tiz)
              : '…'
            ctx.lineWidth = 3
            ctx.strokeStyle = 'rgba(0,0,0,0.92)'
            ctx.strokeText(label, scr.sx, y)
            ctx.fillStyle = 'rgba(255,255,100,0.95)'
            ctx.fillText(label, scr.sx, y)
          }
        }
        ctx.restore()
      }

      drawScenePostFx(ctx, W, DISPLAY_H, fxDimRef.current, fxVignetteRef.current)

      anim.accum += aDef.speed * dt
      while (anim.accum >= 1) {
        anim.accum -= 1
        anim.frameIdx += 1
        if (!aDef.loop && anim.frameIdx >= aDef.frames.length) {
          anim.frameIdx = aDef.frames.length - 1
          anim.accum = 0
          break
        }
        anim.frameIdx %= aDef.frames.length
      }
      animRef.current = anim
    },
    [ready],
  )

  useEffect(() => {
    lastTimeRef.current = 0
    rafRef.current = requestAnimationFrame(gameLoop)
    return () => cancelAnimationFrame(rafRef.current)
  }, [gameLoop])

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
        width: '100%',
      }}
    >
      <div style={{ flexShrink: 0, display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 12 }}>
        <Checkbox checked={musicOn} onChange={(e) => setMusicOn(e.target.checked)}>
          {t('infiniteMapMusic')}
        </Checkbox>
        <Checkbox checked={showTerrainLabels} onChange={(e) => setShowTerrainLabels(e.target.checked)}>
          {t('infiniteMapTerrainDebug')}
        </Checkbox>
        <Checkbox checked={editMode} onChange={(e) => setEditMode(e.target.checked)}>
          {t('infiniteMapEditMode')}
        </Checkbox>
        <Button type="default" size="small" onClick={handleRandomMap}>
          {t('infiniteMapRandomMap')}
        </Button>
        <Button type="default" size="small" onClick={() => setTownPlaceNonce((n) => n + 1)}>
          {t('infiniteMapRandomTownPlace')}
        </Button>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
          <Text type="secondary" style={{ fontSize: 12, whiteSpace: 'nowrap' }}>
            {t('infiniteMapTerrainTextureSet')}
          </Text>
          <Select<InfiniteMapTerrainTextureSetId>
            size="small"
            style={{ minWidth: 168 }}
            value={terrainTextureSet}
            onChange={setTerrainTextureSet}
            options={[
              { value: 'blob', label: t('infiniteMapTerrainTextureBlob') },
              { value: 'tileg', label: t('infiniteMapTerrainTextureTileg') },
              { value: 'tiler', label: t('infiniteMapTerrainTextureTiler') },
            ]}
          />
        </div>
      </div>
      <div
        style={{
          flexShrink: 0,
          display: 'flex',
          flexDirection: 'row',
          flexWrap: 'nowrap',
          alignItems: 'flex-end',
          gap: 12,
          width: '100%',
          padding: '4px 0 2px',
        }}
      >
        <div style={{ flex: '1 1 0', minWidth: 0 }}>
          <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4, lineHeight: 1.35 }}>
            {t('infiniteMapSeaLevel')} <span style={{ color: '#8b9dc3' }}>{(terrainSeaPct / 100).toFixed(2)}</span>
          </Text>
          <Slider
            min={TERRAIN_SEA_MIN}
            max={TERRAIN_SEA_MAX}
            value={terrainSeaPct}
            onChange={(v) => setTerrainSeaPct(v)}
            tooltip={{ formatter: (n) => (n !== undefined ? (n / 100).toFixed(2) : '') }}
          />
        </div>
        <div style={{ flex: '1 1 0', minWidth: 0 }}>
          <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4, lineHeight: 1.35 }}>
            {t('infiniteMapMtnThreshold')}{' '}
            <span style={{ color: '#8b9dc3' }}>{(terrainMtnPct / 100).toFixed(2)}</span>
          </Text>
          <Slider
            min={TERRAIN_MTN_MIN}
            max={TERRAIN_MTN_MAX}
            value={terrainMtnPct}
            onChange={(v) => setTerrainMtnPct(v)}
            tooltip={{ formatter: (n) => (n !== undefined ? (n / 100).toFixed(2) : '') }}
          />
        </div>
        <div style={{ flex: '1 1 0', minWidth: 0 }}>
          <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4, lineHeight: 1.35 }}>
            {t('infiniteMapFxDim')} <span style={{ color: '#8b9dc3' }}>{fxDimPct}%</span>
          </Text>
          <Slider
            min={0}
            max={100}
            value={fxDimPct}
            onChange={(v) => {
              fxDimRef.current = v
              setFxDimPct(v)
            }}
          />
        </div>
        <div style={{ flex: '1 1 0', minWidth: 0 }}>
          <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4, lineHeight: 1.35 }}>
            {t('infiniteMapFxVignette')} <span style={{ color: '#8b9dc3' }}>{fxVignettePct}%</span>
          </Text>
          <Slider
            min={0}
            max={100}
            value={fxVignettePct}
            onChange={(v) => {
              fxVignetteRef.current = v
              setFxVignettePct(v)
            }}
          />
        </div>
      </div>
      <div
        style={{
          flexShrink: 0,
          display: 'flex',
          flexDirection: 'row',
          flexWrap: 'wrap',
          alignItems: 'flex-end',
          gap: 12,
          width: '100%',
          padding: '2px 0 4px',
        }}
      >
        <div style={{ flex: '1 1 180px', minWidth: 0 }}>
          <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4, lineHeight: 1.35 }}>
            {t('infiniteMapTreePatchDensity')} <span style={{ color: '#8b9dc3' }}>{treePatchDensityPct}%</span>
          </Text>
          <Slider
            min={0}
            max={100}
            value={treePatchDensityPct}
            onChange={(v) => {
              treePatchDensityRef.current = v
              setTreePatchDensityPct(v)
            }}
          />
        </div>
        <div style={{ flex: '1 1 180px', minWidth: 0 }}>
          <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4, lineHeight: 1.35 }}>
            {t('infiniteMapTreeLoneDensity')} <span style={{ color: '#8b9dc3' }}>{treeLoneDensityPct}%</span>
          </Text>
          <Slider
            min={0}
            max={100}
            value={treeLoneDensityPct}
            onChange={(v) => {
              treeLoneDensityRef.current = v
              setTreeLoneDensityPct(v)
            }}
          />
        </div>
        <div style={{ flex: '1 1 180px', minWidth: 0 }}>
          <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4, lineHeight: 1.35 }}>
            {t('infiniteMapTreeFeetDown')}{' '}
            <span style={{ color: '#8b9dc3' }}>{treeFeetDownSrcPx}</span>
          </Text>
          <Slider
            min={TREE_FEET_DOWN_MIN}
            max={TREE_FEET_DOWN_MAX}
            value={treeFeetDownSrcPx}
            onChange={(v) => {
              treeFeetDownSrcPxRef.current = v
              setTreeFeetDownSrcPx(v)
            }}
          />
        </div>
        <div style={{ flex: '1 1 160px', minWidth: 0 }}>
          <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4, lineHeight: 1.35 }}>
            {t('infiniteMapMonsterCount')}{' '}
            <span style={{ color: '#8b9dc3' }}>{monsterCount}</span>
          </Text>
          <Slider
            min={0}
            max={MONSTER_COUNT_MAX}
            value={monsterCount}
            onChange={(v) => {
              monsterCountRef.current = v
              setMonsterCount(v)
            }}
          />
        </div>
      </div>
      <Text type="secondary" style={{ fontSize: 11, display: 'block', margin: '0 0 4px', lineHeight: 1.4 }}>
        {t('infiniteMapCityGenSection')}
      </Text>
      <div
        style={{
          flexShrink: 0,
          display: 'flex',
          flexDirection: 'row',
          flexWrap: 'wrap',
          alignItems: 'flex-end',
          gap: 12,
          width: '100%',
          padding: '0 0 6px',
        }}
      >
        <div style={{ flex: '1 1 130px', minWidth: 0 }}>
          <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4, lineHeight: 1.35 }}>
            {t('infiniteMapCityGenSeed')} <span style={{ color: '#8b9dc3' }}>{cityGen.seed}</span>
          </Text>
          <Slider
            min={0}
            max={9999}
            value={cityGen.seed}
            onChange={(v) => setCityGen((p) => ({ ...p, seed: Math.round(v) }))}
          />
        </div>
        <div style={{ flex: '1 1 130px', minWidth: 0 }}>
          <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4, lineHeight: 1.35 }}>
            {t('infiniteMapCityGenGrassHalfIx')} <span style={{ color: '#8b9dc3' }}>{cityGen.grassHalfIx}</span>
          </Text>
          <Slider
            min={1}
            max={5}
            value={cityGen.grassHalfIx}
            onChange={(v) => setCityGen((p) => ({ ...p, grassHalfIx: Math.round(v) }))}
          />
        </div>
        <div style={{ flex: '1 1 130px', minWidth: 0 }}>
          <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4, lineHeight: 1.35 }}>
            {t('infiniteMapCityGenGrassRows')} <span style={{ color: '#8b9dc3' }}>{cityGen.grassRows}</span>
          </Text>
          <Slider
            min={3}
            max={10}
            value={cityGen.grassRows}
            onChange={(v) => setCityGen((p) => ({ ...p, grassRows: Math.round(v) }))}
          />
        </div>
        <div style={{ flex: '1 1 130px', minWidth: 0 }}>
          <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4, lineHeight: 1.35 }}>
            {t('infiniteMapCityGenGrassJitter')} <span style={{ color: '#8b9dc3' }}>{cityGen.grassJitter}</span>
          </Text>
          <Slider
            min={0}
            max={24}
            value={cityGen.grassJitter}
            onChange={(v) => setCityGen((p) => ({ ...p, grassJitter: v }))}
          />
        </div>
        <div style={{ flex: '1 1 130px', minWidth: 0 }}>
          <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4, lineHeight: 1.35 }}>
            {t('infiniteMapCityGenDecorJitter')} <span style={{ color: '#8b9dc3' }}>{cityGen.decorJitter}</span>
          </Text>
          <Slider
            min={0}
            max={40}
            value={cityGen.decorJitter}
            onChange={(v) => setCityGen((p) => ({ ...p, decorJitter: v }))}
          />
        </div>
        <div style={{ flex: '1 1 130px', minWidth: 0 }}>
          <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4, lineHeight: 1.35 }}>
            {t('infiniteMapCityGenBuildingWZ')} <span style={{ color: '#8b9dc3' }}>{cityGen.buildingWZShift}</span>
          </Text>
          <Slider
            min={-50}
            max={50}
            value={cityGen.buildingWZShift}
            onChange={(v) => setCityGen((p) => ({ ...p, buildingWZShift: v }))}
          />
        </div>
        <div style={{ flex: '1 1 130px', minWidth: 0 }}>
          <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4, lineHeight: 1.35 }}>
            {t('infiniteMapCityGenBuildingWxJitter')}{' '}
            <span style={{ color: '#8b9dc3' }}>{cityGen.buildingWxJitter}</span>
          </Text>
          <Slider
            min={0}
            max={30}
            value={cityGen.buildingWxJitter}
            onChange={(v) => setCityGen((p) => ({ ...p, buildingWxJitter: v }))}
          />
        </div>
        <div style={{ flex: '1 1 130px', minWidth: 0 }}>
          <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4, lineHeight: 1.35 }}>
            {t('infiniteMapCityGenEwSegmentCount')}{' '}
            <span style={{ color: '#8b9dc3' }}>{cityGen.ewFenceSegmentCount}</span>
          </Text>
          <Slider
            min={2}
            max={14}
            value={cityGen.ewFenceSegmentCount}
            onChange={(v) => setCityGen((p) => ({ ...p, ewFenceSegmentCount: Math.round(v) }))}
          />
        </div>
        <div style={{ flex: '1 1 130px', minWidth: 0 }}>
          <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4, lineHeight: 1.35 }}>
            {t('infiniteMapCityGenNsSegmentCount')}{' '}
            <span style={{ color: '#8b9dc3' }}>{cityGen.nsFenceSegmentCount}</span>
          </Text>
          <Slider
            min={2}
            max={16}
            value={cityGen.nsFenceSegmentCount}
            onChange={(v) => setCityGen((p) => ({ ...p, nsFenceSegmentCount: Math.round(v) }))}
          />
        </div>
        <div style={{ flex: '1 1 130px', minWidth: 0 }}>
          <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4, lineHeight: 1.35 }}>
            {t('infiniteMapCityGenScatterCount')}{' '}
            <span style={{ color: '#8b9dc3' }}>{cityGen.cityScatterCount}</span>
          </Text>
          <Slider
            min={0}
            max={40}
            value={cityGen.cityScatterCount}
            onChange={(v) => setCityGen((p) => ({ ...p, cityScatterCount: Math.round(v) }))}
          />
        </div>
      </div>
      <div
        ref={gameWrapRef}
        style={{
          width: '100%',
          aspectRatio: '16 / 9',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          background: '#0a0c12',
          borderRadius: 8,
          border: '1px solid #2a3040',
          overflow: 'hidden',
        }}
      >
        <canvas
          ref={canvasRef}
          width={W}
          height={DISPLAY_H}
          style={{
            display: 'block',
            width: W * displayScale,
            height: DISPLAY_H * displayScale,
            maxWidth: '100%',
            maxHeight: '100%',
            imageRendering: 'pixelated',
          }}
        />
      </div>
      <Text type="secondary" style={{ display: 'block', fontSize: 12, flexShrink: 0 }}>
        {t('infiniteMapKeys')}
      </Text>
    </div>
  )
}
