import { useEffect, useMemo, useRef, useState, type WheelEvent as ReactWheelEvent } from 'react'
import {
  ArrowDownOutlined,
  ArrowLeftOutlined,
  ArrowRightOutlined,
  ArrowUpOutlined,
  DeleteOutlined,
  DownloadOutlined,
  ExpandOutlined,
  EyeInvisibleOutlined,
  EyeOutlined,
  UploadOutlined,
} from '@ant-design/icons'
import { Button, message, Select, Slider, Space, Typography } from 'antd'
import JSZip from 'jszip'
import { removeGeminiWatermarkFromBlob } from '@/lib/pixel/geminiWatermark'
import MapStitchApiModal from './MapStitchApiModal'
import MapStitchRegionEditor from './MapStitchRegionEditor'
import './mapStitch.css'

const { Text, Title } = Typography

type Side = 'top' | 'right' | 'bottom' | 'left'
type TileKey = string
type ExpandSplit = 4 | 8 | 12

type LoadedImage = {
  file: File
  url: string
  image: HTMLImageElement
  width: number
  height: number
}

type Tile = {
  x: number
  y: number
  w: number
  h: number
}

type Feather = Record<Side, number>

const MAX_IMAGE_MB = 30
const CENTER_KEY: TileKey = '0,0'
const CENTER_TILE: Tile = { x: 0, y: 0, w: 1, h: 1 }
const DEFAULT_FEATHER: Feather = { top: 0, right: 0, bottom: 0, left: 0 }
const FEATHER_STEP = 5
const MAX_FEATHER = 50
const MIN_ZOOM = 0.05
const MAX_ZOOM = 8
const WHEEL_ZOOM_SENSITIVITY = 0.0015
const STATE_ARCHIVE_VERSION = 2
const STATE_MANIFEST_NAME = 'map_stitch_state.json'

type SavedImageState = {
  fileName?: string
  type?: string
  size?: number
  width?: number
  height?: number
  path?: string
  dataUrl?: string
}

type SavedMapStitchState = {
  version?: number
  source?: SavedImageState
  tiles?: Record<string, Partial<Tile>>
  tileUploads?: Record<string, SavedImageState>
  tileFeathers?: Partial<Record<TileKey, Partial<Feather>>>
  selectedKey?: unknown
  horizontalOverlapPercent?: unknown
  verticalOverlapPercent?: unknown
  expandSplit?: unknown
  pan?: unknown
  zoom?: unknown
  hidePreviewBorders?: unknown
  hiddenPreviewTiles?: unknown
}

function clampZoom(value: number): number {
  return Math.max(MIN_ZOOM, Math.min(MAX_ZOOM, value))
}

function tileKey(x: number, y: number): TileKey {
  const fmt = (n: number) => {
    const fixed = n.toFixed(4).replace(/\.?0+$/, '')
    return fixed === '-0' ? '0' : fixed
  }
  return `${fmt(x)},${fmt(y)}`
}

function splitSegments(start: number, size: number, count: number, overlapRatio: number): Array<{ start: number; size: number }> {
  if (count <= 1) return [{ start, size }]
  const clampedOverlap = Math.max(0, Math.min(0.95, overlapRatio))
  const segmentSize = size / (count - (count - 1) * clampedOverlap)
  const segmentStep = segmentSize * (1 - clampedOverlap)
  return Array.from({ length: count }, (_, i) => ({
    start: i === count - 1 ? start + size - segmentSize : start + i * segmentStep,
    size: segmentSize,
  }))
}

function expansionTilesFrom(origin: Tile, split: ExpandSplit, horizontalOverlapRatio: number, verticalOverlapRatio: number): Tile[] {
  const count = split / 4
  const clampedHorizontalOverlap = Math.max(0, Math.min(0.95, horizontalOverlapRatio))
  const clampedVerticalOverlap = Math.max(0, Math.min(0.95, verticalOverlapRatio))
  const horizontalSegments = splitSegments(origin.x, origin.w, count, horizontalOverlapRatio)
  const verticalSegments = splitSegments(origin.y, origin.h, count, verticalOverlapRatio)
  const out: Tile[] = []
  for (const segment of horizontalSegments) {
    const tileSize = segment.size
    out.push({ x: segment.start, y: origin.y - tileSize * (1 - clampedVerticalOverlap), w: tileSize, h: tileSize })
    out.push({ x: segment.start, y: origin.y + origin.h - tileSize * clampedVerticalOverlap, w: tileSize, h: tileSize })
  }
  for (const segment of verticalSegments) {
    const tileSize = segment.size
    out.push({ x: origin.x + origin.w - tileSize * clampedHorizontalOverlap, y: segment.start, w: tileSize, h: tileSize })
    out.push({ x: origin.x - tileSize * (1 - clampedHorizontalOverlap), y: segment.start, w: tileSize, h: tileSize })
  }
  return out
}

function isOutwardExpansionTile(origin: Tile, target: Tile): boolean {
  if (tileKey(origin.x, origin.y) === CENTER_KEY) return true
  if (origin.x > 0 && target.x < origin.x) return false
  if (origin.x < 0 && target.x > origin.x) return false
  if (origin.y > 0 && target.y < origin.y) return false
  if (origin.y < 0 && target.y > origin.y) return false
  return true
}

function tilesRecord(items: Tile[]): Record<TileKey, Tile> {
  return Object.fromEntries(items.map((tile) => [tileKey(tile.x, tile.y), tile])) as Record<TileKey, Tile>
}

function loadImageFile(file: File): Promise<LoadedImage> {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file)
    const image = new Image()
    image.onload = () => {
      resolve({
        file,
        url,
        image,
        width: image.naturalWidth,
        height: image.naturalHeight,
      })
    }
    image.onerror = () => {
      URL.revokeObjectURL(url)
      reject(new Error('图片读取失败'))
    }
    image.src = url
  })
}

function canvasToBlob(canvas: HTMLCanvasElement): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob((blob) => (blob ? resolve(blob) : reject(new Error('导出失败'))), 'image/png')
  })
}

type TemplateLayer = {
  image: LoadedImage
  offsetX: number
  offsetY: number
  drawWidth: number
  drawHeight: number
}

async function makeRelativeTemplate(layers: TemplateLayer[], canvasWidth: number, canvasHeight: number): Promise<Blob> {
  const canvas = document.createElement('canvas')
  canvas.width = canvasWidth
  canvas.height = canvasHeight
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('画布创建失败')
  ctx.imageSmoothingEnabled = false
  for (const layer of layers) {
    ctx.drawImage(layer.image.image, layer.offsetX, layer.offsetY, layer.drawWidth, layer.drawHeight)
  }
  return canvasToBlob(canvas)
}

function featheredImageCanvas(image: HTMLImageElement, width: number, height: number, feather: Feather): HTMLCanvasElement {
  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const ctx = canvas.getContext('2d')
  if (!ctx) return canvas
  ctx.imageSmoothingEnabled = false
  ctx.drawImage(image, 0, 0, width, height)

  const alphaForDistancePct = (distancePct: number, featherPct: number) => {
    if (featherPct <= 0) return 1
    const endPct = Math.max(0, Math.min(MAX_FEATHER, featherPct))
    const startPct = Math.max(0, endPct - FEATHER_STEP)
    if (distancePct <= startPct) return 0
    if (distancePct >= endPct) return 1
    return (distancePct - startPct) / Math.max(0.0001, endPct - startPct)
  }

  const imageData = ctx.getImageData(0, 0, width, height)
  const data = imageData.data
  const xDenom = Math.max(1, width - 1)
  const yDenom = Math.max(1, height - 1)
  for (let y = 0; y < height; y++) {
    const topPct = (y / yDenom) * 100
    const bottomPct = ((height - 1 - y) / yDenom) * 100
    const yAlpha =
      alphaForDistancePct(topPct, feather.top) *
      alphaForDistancePct(bottomPct, feather.bottom)
    for (let x = 0; x < width; x++) {
      const leftPct = (x / xDenom) * 100
      const rightPct = ((width - 1 - x) / xDenom) * 100
      const alpha =
        yAlpha *
        alphaForDistancePct(leftPct, feather.left) *
        alphaForDistancePct(rightPct, feather.right)
      data[(y * width + x) * 4 + 3] = Math.round((data[(y * width + x) * 4 + 3] ?? 0) * alpha)
    }
  }
  ctx.putImageData(imageData, 0, 0)
  return canvas
}

function FeatheredPreviewImage({
  upload,
  width,
  height,
  feather,
  alt,
}: {
  upload: LoadedImage
  width: number
  height: number
  feather: Feather
  alt: string
}) {
  const hasFeather = feather.top !== 0 || feather.right !== 0 || feather.bottom !== 0 || feather.left !== 0
  const previewKey = `${upload.url}:${width}:${height}:${feather.top}:${feather.right}:${feather.bottom}:${feather.left}`
  const [generatedPreview, setGeneratedPreview] = useState<{ key: string; url: string } | null>(null)

  useEffect(() => {
    if (!hasFeather) return

    let revokedUrl: string | null = null
    let cancelled = false
    const canvas = featheredImageCanvas(upload.image, Math.max(1, Math.round(width)), Math.max(1, Math.round(height)), feather)
    canvas.toBlob((blob) => {
      if (!blob || cancelled) return
      revokedUrl = URL.createObjectURL(blob)
      setGeneratedPreview({ key: previewKey, url: revokedUrl })
    }, 'image/png')

    return () => {
      cancelled = true
      if (revokedUrl) URL.revokeObjectURL(revokedUrl)
    }
  }, [feather, hasFeather, height, previewKey, upload.image, width])

  const src = hasFeather && generatedPreview?.key === previewKey ? generatedPreview.url : upload.url
  return <img src={src} alt={alt} />
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.style.display = 'none'
  document.body.appendChild(link)
  link.click()
  link.remove()
  window.setTimeout(() => URL.revokeObjectURL(url), 30_000)
}

function dataUrlToFile(dataUrl: string, fileName: string, fallbackType = 'image/png'): File {
  const [meta = '', payload = ''] = dataUrl.split(',', 2)
  if (!payload) throw new Error('拼接状态中的图片数据不完整')
  const mime = meta.match(/^data:([^;,]+)/)?.[1] || fallbackType
  const binary = atob(payload)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
  return new File([bytes], fileName, { type: mime })
}

function fileExtensionFor(file: File): string {
  const fromName = file.name.match(/\.[a-z0-9]+$/i)?.[0]?.toLowerCase()
  if (fromName && ['.png', '.jpg', '.jpeg', '.webp'].includes(fromName)) return fromName
  if (file.type === 'image/jpeg') return '.jpg'
  if (file.type === 'image/webp') return '.webp'
  return '.png'
}

function uniqueArchiveImagePath(used: Set<string>, baseName: string, file: File): string {
  const ext = fileExtensionFor(file)
  const safeBase = safeFilename(baseName.replace(/\.[^.]+$/, ''))
  let index = 0
  let name = `${safeBase}${ext}`
  while (used.has(`images/${name}`)) {
    index += 1
    name = `${safeBase}_${index}${ext}`
  }
  const path = `images/${name}`
  used.add(path)
  return path
}

async function savedImageStateToFile(
  image: SavedImageState | undefined,
  fallbackName: string,
  zip?: JSZip,
): Promise<File> {
  if (!image) throw new Error('拼接状态缺少图片数据')
  if (image.dataUrl) return dataUrlToFile(image.dataUrl, image.fileName || fallbackName, image.type || 'image/png')
  if (!image.path || !zip) throw new Error('拼接状态中的图片引用无效')

  const entry = zip.file(image.path)
  if (!entry) throw new Error(`拼接状态缺少图片文件：${image.path}`)
  const blob = await entry.async('blob')
  return new File([blob], image.fileName || image.path.split('/').pop() || fallbackName, {
    type: image.type || blob.type || 'image/png',
  })
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function numberOr(value: unknown, fallback: number): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback
}

function booleanRecord(value: unknown): Partial<Record<TileKey, boolean>> {
  if (!isRecord(value)) return {}
  return Object.fromEntries(Object.entries(value).filter(([, item]) => typeof item === 'boolean')) as Partial<Record<TileKey, boolean>>
}

function normalizeTilesState(value: unknown): Record<TileKey, Tile> {
  if (!isRecord(value)) throw new Error('拼接状态缺少地图块数据')
  const entries = Object.entries(value).flatMap(([key, raw]) => {
    if (!isRecord(raw)) return []
    const tile = {
      x: numberOr(raw.x, Number.NaN),
      y: numberOr(raw.y, Number.NaN),
      w: numberOr(raw.w, Number.NaN),
      h: numberOr(raw.h, Number.NaN),
    }
    return Number.isFinite(tile.x) && Number.isFinite(tile.y) && tile.w > 0 && tile.h > 0 ? [[key, tile] as const] : []
  })
  if (entries.length === 0) throw new Error('拼接状态中的地图块数据无效')
  return Object.fromEntries(entries) as Record<TileKey, Tile>
}

function normalizeFeathersState(value: unknown): Partial<Record<TileKey, Feather>> {
  if (!isRecord(value)) return {}
  return Object.fromEntries(
    Object.entries(value).flatMap(([key, raw]) => {
      if (!isRecord(raw)) return []
      return [[
        key,
        {
          top: Math.max(0, Math.min(MAX_FEATHER, numberOr(raw.top, 0))),
          right: Math.max(0, Math.min(MAX_FEATHER, numberOr(raw.right, 0))),
          bottom: Math.max(0, Math.min(MAX_FEATHER, numberOr(raw.bottom, 0))),
          left: Math.max(0, Math.min(MAX_FEATHER, numberOr(raw.left, 0))),
        },
      ] as const]
    }),
  ) as Partial<Record<TileKey, Feather>>
}

async function ensureImageReady(image: HTMLImageElement): Promise<void> {
  if (image.complete && image.naturalWidth > 0) return
  if (typeof image.decode === 'function') {
    await image.decode()
    return
  }
  await new Promise<void>((resolve, reject) => {
    image.onload = () => resolve()
    image.onerror = () => reject(new Error('鍥剧墖璇诲彇澶辫触'))
  })
}

function writeAscii(out: number[], value: string) {
  for (let i = 0; i < value.length; i++) out.push(value.charCodeAt(i) & 0xff)
}

function writeU16(out: number[], value: number) {
  out.push((value >>> 8) & 0xff, value & 0xff)
}

function writeU32(out: number[], value: number) {
  out.push((value >>> 24) & 0xff, (value >>> 16) & 0xff, (value >>> 8) & 0xff, value & 0xff)
}

function writeI16(out: number[], value: number) {
  writeU16(out, value < 0 ? 0x10000 + value : value)
}

function bytesFromWriter(write: (out: number[]) => void): Uint8Array<ArrayBuffer> {
  const out: number[] = []
  write(out)
  return new Uint8Array(out)
}

function psdPascalName(name: string): Uint8Array<ArrayBuffer> {
  const bytes = Array.from(name).map((ch) => ch.charCodeAt(0) & 0x7f)
  const len = Math.min(255, bytes.length)
  const total = 1 + len
  const padded = total + ((4 - (total % 4)) % 4)
  const out = new Uint8Array(padded)
  out[0] = len
  for (let i = 0; i < len; i++) out[i + 1] = bytes[i] ?? 0
  return out
}

function imageDataChannelBytes(imageData: ImageData, channel: 0 | 1 | 2 | 3): Uint8Array<ArrayBuffer> {
  const out = new Uint8Array(imageData.width * imageData.height)
  const data = imageData.data
  for (let src = channel, dst = 0; src < data.length; src += 4, dst++) {
    out[dst] = data[src] ?? 0
  }
  return out
}

type PsdLayer = {
  name: string
  left: number
  top: number
  imageData: ImageData
}

function layeredPsdBlob(width: number, height: number, composite: ImageData, layers: PsdLayer[]): Blob {
  if (width > 30000 || height > 30000) {
    throw new Error('PSD 尺寸超过 30000px，请减少扩展范围后再导出')
  }

  const chunks: BlobPart[] = []
  chunks.push(bytesFromWriter((out) => {
    writeAscii(out, '8BPS')
    writeU16(out, 1)
    out.push(0, 0, 0, 0, 0, 0)
    writeU16(out, 4)
    writeU32(out, height)
    writeU32(out, width)
    writeU16(out, 8)
    writeU16(out, 3)
    writeU32(out, 0)
    writeU32(out, 0)
  }))

  const layerRecordChunks: BlobPart[] = []
  const channelDataChunks: BlobPart[] = []
  let layerRecordsLen = 2
  let layerChannelsLen = 0

  for (const layer of layers) {
    const layerW = layer.imageData.width
    const layerH = layer.imageData.height
    const channelByteLen = layerW * layerH
    const channelDataLen = 2 + channelByteLen
    const nameBytes = psdPascalName(layer.name)
    const extraLen = 4 + 4 + nameBytes.length

    const record = bytesFromWriter((out) => {
      writeU32(out, layer.top)
      writeU32(out, layer.left)
      writeU32(out, layer.top + layerH)
      writeU32(out, layer.left + layerW)
      writeU16(out, 4)
      for (const channelId of [0, 1, 2, -1]) {
        writeI16(out, channelId)
        writeU32(out, channelDataLen)
      }
      writeAscii(out, '8BIM')
      writeAscii(out, 'norm')
      out.push(255, 0, 8, 0)
      writeU32(out, extraLen)
      writeU32(out, 0)
      writeU32(out, 0)
      for (const b of nameBytes) out.push(b)
    })
    layerRecordChunks.push(record)
    layerRecordsLen += record.length

    for (const channel of [0, 1, 2, 3] as const) {
      channelDataChunks.push(bytesFromWriter((out) => writeU16(out, 0)))
      channelDataChunks.push(imageDataChannelBytes(layer.imageData, channel))
      layerChannelsLen += channelDataLen
    }
  }

  const layerInfoLenRaw = layerRecordsLen + layerChannelsLen
  const layerInfoPad = layerInfoLenRaw % 2
  const layerInfoLen = layerInfoLenRaw + layerInfoPad
  const layerMaskLen = 4 + layerInfoLen + 4

  chunks.push(bytesFromWriter((out) => {
    writeU32(out, layerMaskLen)
    writeU32(out, layerInfoLen)
    writeI16(out, layers.length)
  }))
  chunks.push(...layerRecordChunks)
  chunks.push(...channelDataChunks)
  if (layerInfoPad) chunks.push(new Uint8Array([0]) as Uint8Array<ArrayBuffer>)
  chunks.push(bytesFromWriter((out) => writeU32(out, 0)))

  chunks.push(bytesFromWriter((out) => writeU16(out, 0)))
  for (const channel of [0, 1, 2, 3] as const) {
    chunks.push(imageDataChannelBytes(composite, channel))
  }

  return new Blob(chunks, { type: 'image/vnd.adobe.photoshop' })
}

function getBaseName(file: File | null): string {
  return file?.name.replace(/\.[^.]+$/, '') || 'map_tile'
}

function safeFilename(value: string): string {
  return value.replace(/[^a-z0-9._-]+/gi, '_').replace(/^_+|_+$/g, '') || 'tile'
}

function tscnString(value: string): string {
  return JSON.stringify(value)
}

function godotSceneTscn(
  sceneName: string,
  tiles: Array<{ name: string; image: string; pixel: { x: number; y: number } }>,
): string {
  const lines = ['[gd_scene load_steps=' + (tiles.length + 1) + ' format=3 uid="uid://pixelwork_map_stitch"]', '']
  tiles.forEach((tile, index) => {
    lines.push(`[ext_resource type="Texture2D" path=${tscnString(`res://${tile.image}`)} id=${tscnString(`tex_${index + 1}`)}]`)
  })
  lines.push('', `[node name=${tscnString(sceneName)} type="Node2D"]`, '')
  tiles.forEach((tile, index) => {
    lines.push(`[node name=${tscnString(safeFilename(tile.name))} type="Sprite2D" parent="."]`)
    lines.push(`position = Vector2(${tile.pixel.x}, ${tile.pixel.y})`)
    lines.push('centered = false')
    lines.push(`texture = ExtResource(${tscnString(`tex_${index + 1}`)})`)
    lines.push('')
  })
  return lines.join('\n')
}

function toneForTile(tile: Tile): Side {
  if (Math.abs(tile.x) >= Math.abs(tile.y) && tile.x !== 0) return tile.x > 0 ? 'right' : 'left'
  if (tile.y !== 0) return tile.y > 0 ? 'bottom' : 'top'
  return 'right'
}

function labelForTile(tile: Tile): string {
  const near = (a: number, b: number) => Math.abs(a - b) < 0.0001
  if (near(tile.x, 0) && tile.y < 0 && near(tile.w, 1) && near(tile.h, 1)) return '上方'
  if (tile.x > 0 && near(tile.y, 0) && near(tile.w, 1) && near(tile.h, 1)) return '右侧'
  if (near(tile.x, 0) && tile.y > 0 && near(tile.w, 1) && near(tile.h, 1)) return '下方'
  if (tile.x < 0 && near(tile.y, 0) && near(tile.w, 1) && near(tile.h, 1)) return '左侧'
  const format = (n: number) => Number.isInteger(n) ? String(Math.abs(n)) : Math.abs(n).toFixed(2)
  const xText = tile.x === 0 ? '' : tile.x > 0 ? `右${format(tile.x)}` : `左${format(tile.x)}`
  const yText = tile.y === 0 ? '' : tile.y > 0 ? `下${format(tile.y)}` : `上${format(tile.y)}`
  return [xText, yText].filter(Boolean).join(' ')
}

interface Props {
  onBack?: () => void
}

export default function MapStitch({ onBack }: Props) {
  const [source, setSource] = useState<LoadedImage | null>(null)
  const [tiles, setTiles] = useState<Record<TileKey, Tile>>(() =>
    tilesRecord(expansionTilesFrom(CENTER_TILE, 4, 0.15, 0.15)),
  )
  const [tileUploads, setTileUploads] = useState<Partial<Record<TileKey, LoadedImage>>>({})
  const [tileFeathers, setTileFeathers] = useState<Partial<Record<TileKey, Feather>>>({})
  const [selectedKey, setSelectedKey] = useState<TileKey | null>(null)
  const [horizontalOverlapPercent, setHorizontalOverlapPercent] = useState(15)
  const [verticalOverlapPercent, setVerticalOverlapPercent] = useState(15)
  const [expandSplit, setExpandSplit] = useState<ExpandSplit>(4)
  const [processingKey, setProcessingKey] = useState<TileKey | null>(null)
  const [viewport, setViewport] = useState({ width: window.innerWidth, height: window.innerHeight })
  const [pan, setPan] = useState({ x: 0, y: 0 })
  const [zoom, setZoom] = useState(1)
  const [hidePreviewBorders, setHidePreviewBorders] = useState(false)
  const [hiddenPreviewTiles, setHiddenPreviewTiles] = useState<Partial<Record<TileKey, boolean>>>({})
  const [pendingUploadKey, setPendingUploadKey] = useState<TileKey | null>(null)
  const [apiModalKey, setApiModalKey] = useState<TileKey | null>(null)
  const [apiTemplateFile, setApiTemplateFile] = useState<File | null>(null)
  const [apiTemplateUrl, setApiTemplateUrl] = useState<string | null>(null)
  const [regionKey, setRegionKey] = useState<TileKey | null>(null)
  const [regionImageUrl, setRegionImageUrl] = useState<string | null>(null)
  const [regionSize, setRegionSize] = useState({ w: 512, h: 512 })
  const hostRef = useRef<HTMLDivElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const tileFileInputRef = useRef<HTMLInputElement>(null)
  const stateFileInputRef = useRef<HTMLInputElement>(null)
  const panDragRef = useRef<{ startX: number; startY: number; panX: number; panY: number } | null>(null)
  const cleanupRef = useRef<{
    source: LoadedImage | null
    tileUploads: Partial<Record<TileKey, LoadedImage>>
  }>({ source: null, tileUploads: {} })
  cleanupRef.current = { source, tileUploads }

  useEffect(() => {
    const el = hostRef.current
    const apply = () => {
      const r = hostRef.current?.getBoundingClientRect()
      if (r && r.width > 0 && r.height > 0) {
        setViewport({ width: Math.floor(r.width), height: Math.floor(r.height) })
      } else {
        setViewport({ width: window.innerWidth, height: Math.max(480, window.innerHeight - 180) })
      }
    }
    apply()
    const ro = el ? new ResizeObserver(apply) : null
    if (el && ro) ro.observe(el)
    window.addEventListener('resize', apply)
    return () => {
      window.removeEventListener('resize', apply)
      ro?.disconnect()
    }
  }, [])

  useEffect(
    () => () => {
      const current = cleanupRef.current
      if (current.source) URL.revokeObjectURL(current.source.url)
      Object.values(current.tileUploads).forEach((item) => item && URL.revokeObjectURL(item.url))
    },
    [],
  )

  const allTiles = useMemo(() => [CENTER_TILE, ...Object.values(tiles)], [tiles])
  const horizontalOverlapRatio = horizontalOverlapPercent / 100
  const verticalOverlapRatio = verticalOverlapPercent / 100
  const initialTilesForSplit = useMemo(
    () => expansionTilesFrom(CENTER_TILE, expandSplit, horizontalOverlapRatio, verticalOverlapRatio),
    [expandSplit, horizontalOverlapRatio, verticalOverlapRatio],
  )

  const stage = useMemo(() => {
    if (!source) return null
    const minX = Math.min(...allTiles.map((tile) => tile.x))
    const maxX = Math.max(...allTiles.map((tile) => tile.x + tile.w))
    const minY = Math.min(...allTiles.map((tile) => tile.y))
    const maxY = Math.max(...allTiles.map((tile) => tile.y + tile.h))
    const stageW = (maxX - minX) * source.width
    const stageH = (maxY - minY) * source.height
    return {
      stageW,
      stageH,
      scale: zoom,
      unitW: source.width,
      unitH: source.height,
      minX,
      minY,
    }
  }, [source, allTiles, zoom])

  const completedImageForKey = (key: TileKey): LoadedImage | null => {
    if (key === CENTER_KEY) return source
    return tileUploads[key] ?? null
  }

  const hasCompletedImage = (key: TileKey): boolean => Boolean(completedImageForKey(key))

  const featherForKey = (key: TileKey): Feather => tileFeathers[key] ?? DEFAULT_FEATHER

  const completedTiles = () => allTiles.filter((tile) => hasCompletedImage(tileKey(tile.x, tile.y)))

  const exportBoundsFor = (items: Tile[]) => {
    if (!source || items.length === 0) return null
    const minX = Math.min(...items.map((tile) => tile.x))
    const maxX = Math.max(...items.map((tile) => tile.x + tile.w))
    const minY = Math.min(...items.map((tile) => tile.y))
    const maxY = Math.max(...items.map((tile) => tile.y + tile.h))
    return {
      minX,
      minY,
      unitW: source.width,
      unitH: source.height,
      width: Math.max(1, Math.round((maxX - minX) * source.width)),
      height: Math.max(1, Math.round((maxY - minY) * source.height)),
    }
  }

  const tilePosition = (tile: Tile) => {
    if (!stage) return {}
    return {
      left: (tile.x - stage.minX) * stage.unitW,
      top: (tile.y - stage.minY) * stage.unitH,
    }
  }

  const tileRect = (tile: Tile) => {
    if (!stage) return null
    const left = (tile.x - stage.minX) * stage.unitW
    const top = (tile.y - stage.minY) * stage.unitH
    const width = tile.w * stage.unitW
    const height = tile.h * stage.unitH
    return {
      left,
      top,
      width,
      height,
      right: left + width,
      bottom: top + height,
    }
  }

  const tilesIntersect = (a: Tile, b: Tile): boolean => {
    const ar = tileRect(a)
    const br = tileRect(b)
    if (!ar || !br) return false
    return ar.left < br.right && ar.right > br.left && ar.top < br.bottom && ar.bottom > br.top
  }

  const createStitchedCanvas = async (): Promise<HTMLCanvasElement | null> => {
    if (!source) return null
    const exportTiles = completedTiles()
    const bounds = exportBoundsFor(exportTiles)
    if (!bounds) return null
    const canvas = document.createElement('canvas')
    canvas.width = bounds.width
    canvas.height = bounds.height
    const ctx = canvas.getContext('2d')
    if (!ctx) return null
    ctx.imageSmoothingEnabled = false

    for (const tile of exportTiles) {
      const key = tileKey(tile.x, tile.y)
      const image = completedImageForKey(key)
      if (!image) continue
      await ensureImageReady(image.image)
      const left = Math.round((tile.x - bounds.minX) * bounds.unitW)
      const top = Math.round((tile.y - bounds.minY) * bounds.unitH)
      const drawWidth = Math.max(1, Math.round(tile.w * bounds.unitW))
      const drawHeight = Math.max(1, Math.round(tile.h * bounds.unitH))
      const feather = featherForKey(key)
      const drawable = key === CENTER_KEY || (feather.top === 0 && feather.right === 0 && feather.bottom === 0 && feather.left === 0)
        ? image.image
        : featheredImageCanvas(image.image, drawWidth, drawHeight, feather)
      ctx.drawImage(drawable, left, top, drawWidth, drawHeight)
    }

    return canvas
  }

  const createPsdLayers = (): PsdLayer[] | null => {
    if (!source) return null
    const exportTiles = completedTiles()
    const bounds = exportBoundsFor(exportTiles)
    if (!bounds) return null
    const layers: PsdLayer[] = []
    for (const tile of exportTiles) {
      const key = tileKey(tile.x, tile.y)
      const image = completedImageForKey(key)
      if (!image) continue
      void image.image.decode?.().catch(() => undefined)
      const layerCanvas = document.createElement('canvas')
      layerCanvas.width = Math.max(1, Math.round(tile.w * bounds.unitW))
      layerCanvas.height = Math.max(1, Math.round(tile.h * bounds.unitH))
      const layerCtx = layerCanvas.getContext('2d')
      if (!layerCtx) continue
      layerCtx.imageSmoothingEnabled = false
      const feather = featherForKey(key)
      const drawable = key === CENTER_KEY || (feather.top === 0 && feather.right === 0 && feather.bottom === 0 && feather.left === 0)
        ? image.image
        : featheredImageCanvas(image.image, layerCanvas.width, layerCanvas.height, feather)
      layerCtx.drawImage(drawable, 0, 0, layerCanvas.width, layerCanvas.height)
      layers.push({
        name: key === CENTER_KEY ? 'source' : `tile_${key.replace(',', '_')}`,
        left: Math.round((tile.x - bounds.minX) * bounds.unitW),
        top: Math.round((tile.y - bounds.minY) * bounds.unitH),
        imageData: layerCtx.getImageData(0, 0, layerCanvas.width, layerCanvas.height),
      })
    }
    return layers
  }

  const selectSource = async (file: File | null) => {
    if (!file) return
    if (!file.type.startsWith('image/')) {
      message.error('请选择图片文件')
      return
    }
    if (file.size > MAX_IMAGE_MB * 1024 * 1024) {
      message.error(`图片不能超过 ${MAX_IMAGE_MB}MB`)
      return
    }
    try {
      const loaded = await loadImageFile(file)
      const initTiles = [CENTER_TILE, ...initialTilesForSplit]
      const minX = Math.min(...initTiles.map((tile) => tile.x))
      const maxX = Math.max(...initTiles.map((tile) => tile.x + tile.w))
      const minY = Math.min(...initTiles.map((tile) => tile.y))
      const maxY = Math.max(...initTiles.map((tile) => tile.y + tile.h))
      const initialStageW = (maxX - minX) * loaded.width
      const initialStageH = (maxY - minY) * loaded.height
      const availableW = Math.max(320, viewport.width - 48)
      const availableH = Math.max(320, viewport.height - 160)
      setSource((prev) => {
        if (prev) URL.revokeObjectURL(prev.url)
        return loaded
      })
      setSelectedKey(null)
      setPan({ x: 0, y: 0 })
      setZoom(Math.min(1, availableW / initialStageW, availableH / initialStageH))
      setTiles(tilesRecord(initialTilesForSplit))
      setTileUploads((prev) => {
        Object.values(prev).forEach((item) => item && URL.revokeObjectURL(item.url))
        return {}
      })
      setTileFeathers({})
      setHiddenPreviewTiles({})
      setHidePreviewBorders(false)
    } catch (error) {
      message.error(String(error))
    }
  }

  const selectTileUpload = async (key: TileKey, file: File | null) => {
    if (!file) return
    if (!file.type.startsWith('image/')) {
      message.error('请选择图片文件')
      return
    }
    if (file.size > MAX_IMAGE_MB * 1024 * 1024) {
      message.error(`图片不能超过 ${MAX_IMAGE_MB}MB`)
      return
    }
    try {
      const upload = await loadImageFile(file)
      setTileUploads((prev) => {
        if (prev[key]) URL.revokeObjectURL(prev[key]!.url)
        return { ...prev, [key]: upload }
      })
      setSelectedKey(key)
    } catch (error) {
      message.error(String(error))
    }
  }

  const openTileUpload = (key: TileKey) => {
    setPendingUploadKey(key)
    tileFileInputRef.current?.click()
  }

  const findTemplateNeighbors = (target: Tile): Array<{ image: LoadedImage; tile: Tile }> => {
    const targetKey = tileKey(target.x, target.y)
    const neighbors: Array<{ image: LoadedImage; tile: Tile }> = []
    for (const completedTile of allTiles) {
      const completedKey = tileKey(completedTile.x, completedTile.y)
      if (completedKey === targetKey) continue
      const image = completedImageForKey(completedKey)
      if (!image) continue
      if (tilesIntersect(target, completedTile)) neighbors.push({ image, tile: completedTile })
    }
    return neighbors
  }

  const buildTileTemplateBlob = async (key: TileKey): Promise<{ blob: Blob; width: number; height: number } | null> => {
    const tile = tiles[key]
    if (!tile) return null
    const neighbors = findTemplateNeighbors(tile)
    if (neighbors.length === 0) {
      message.warning('周边没有可用于生成重叠像素的已完成图片（请先完成相邻格）')
      return null
    }
    const targetRect = tileRect(tile)
    if (!targetRect || !stage) return null
    const layers = neighbors.flatMap((neighbor): TemplateLayer[] => {
      const neighborRect = tileRect(neighbor.tile)
      if (!neighborRect) return []
      return [{
        image: neighbor.image,
        offsetX: Math.round(neighborRect.left - targetRect.left),
        offsetY: Math.round(neighborRect.top - targetRect.top),
        drawWidth: Math.max(1, Math.round(neighborRect.width)),
        drawHeight: Math.max(1, Math.round(neighborRect.height)),
      }]
    })
    const width = Math.max(1, Math.round(targetRect.width))
    const height = Math.max(1, Math.round(targetRect.height))
    const blob = await makeRelativeTemplate(layers, width, height)
    return { blob, width, height }
  }

  const downloadTileTemplate = async (key: TileKey) => {
    try {
      const built = await buildTileTemplateBlob(key)
      if (!built) return
      downloadBlob(built.blob, `${getBaseName(source?.file ?? null)}_${key.replace(',', '_')}_overlap_template.png`)
      message.success('模板已下载')
    } catch (error) {
      message.error(String(error))
    }
  }

  const openApiGenerate = async (key: TileKey) => {
    try {
      const built = await buildTileTemplateBlob(key)
      if (!built) return
      if (apiTemplateUrl) URL.revokeObjectURL(apiTemplateUrl)
      const file = new File([built.blob], `template_${key.replace(',', '_')}.png`, { type: 'image/png' })
      const url = URL.createObjectURL(built.blob)
      setApiTemplateFile(file)
      setApiTemplateUrl(url)
      setApiModalKey(key)
    } catch (error) {
      message.error(String(error))
    }
  }

  const openRegionDraw = async (key: TileKey) => {
    try {
      const built = await buildTileTemplateBlob(key)
      if (!built) return
      if (regionImageUrl) URL.revokeObjectURL(regionImageUrl)
      setRegionSize({ w: built.width, h: built.height })
      setRegionImageUrl(URL.createObjectURL(built.blob))
      setRegionKey(key)
    } catch (error) {
      message.error(String(error))
    }
  }

  const removeWatermarkForTile = async (key: TileKey) => {
    const upload = tileUploads[key]
    if (!upload) return
    setProcessingKey(key)
    try {
      const blob = await removeGeminiWatermarkFromBlob(upload.file)
      const cleanFile = new File([blob], `${getBaseName(upload.file)}_no_watermark.png`, { type: 'image/png' })
      const cleanImage = await loadImageFile(cleanFile)
      setTileUploads((prev) => {
        if (prev[key]) URL.revokeObjectURL(prev[key]!.url)
        return { ...prev, [key]: cleanImage }
      })
      message.success('去水印完成')
    } catch (error) {
      message.error(`去水印失败：${String(error)}`)
    } finally {
      setProcessingKey(null)
    }
  }

  const increaseFeather = (key: TileKey, side: Side) => {
    setTileFeathers((prev) => {
      const current = prev[key] ?? DEFAULT_FEATHER
      return {
        ...prev,
        [key]: {
          ...current,
          [side]: Math.min(MAX_FEATHER, current[side] + FEATHER_STEP),
        },
      }
    })
  }

  const decreaseFeather = (key: TileKey, side: Side) => {
    setTileFeathers((prev) => {
      const current = prev[key] ?? DEFAULT_FEATHER
      return {
        ...prev,
        [key]: {
          ...current,
          [side]: Math.max(0, current[side] - FEATHER_STEP),
        },
      }
    })
  }

  const togglePreviewTileVisibility = (key: TileKey) => {
    setHiddenPreviewTiles((prev) => ({ ...prev, [key]: !prev[key] }))
  }

  const unloadTileUpload = (key: TileKey) => {
    setTileUploads((prev) => {
      const current = prev[key]
      if (!current) return prev
      URL.revokeObjectURL(current.url)
      const next = { ...prev }
      delete next[key]
      return next
    })
    setTileFeathers((prev) => {
      if (!prev[key]) return prev
      const next = { ...prev }
      delete next[key]
      return next
    })
    setHiddenPreviewTiles((prev) => {
      if (!prev[key]) return prev
      const next = { ...prev }
      delete next[key]
      return next
    })
    message.success('图片已卸载')
  }

  const expandFromTile = (key: TileKey) => {
    const origin = key === CENTER_KEY ? CENTER_TILE : tiles[key]
    if (!origin || !hasCompletedImage(key)) return
    const split = key === CENTER_KEY ? expandSplit : 4
    let added = 0
    setTiles((prev) => {
      const next = { ...prev }
      for (const target of expansionTilesFrom(origin, split, horizontalOverlapRatio, verticalOverlapRatio)) {
        if (!isOutwardExpansionTile(origin, target)) continue
        const targetKey = tileKey(target.x, target.y)
        if (hasCompletedImage(targetKey)) continue
        if (!next[targetKey]) {
          next[targetKey] = target
          added += 1
        }
      }
      return next
    })
    if (added > 0) message.success(`已扩展 ${added} 个空框`)
    else message.info('周边已有空框或已完成图片，无需扩展')
  }

  const downloadStitchedPng = async () => {
    const canvas = await createStitchedCanvas()
    if (!canvas || !source) return
    try {
      const blob = await canvasToBlob(canvas)
      downloadBlob(blob, `${getBaseName(source.file)}_stitched.png`)
      message.success('拼接 PNG 已下载')
    } catch (error) {
      message.error(String(error))
    }
  }

  const downloadStitchedPsd = async () => {
    const canvas = await createStitchedCanvas()
    if (!canvas || !source) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    try {
      const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height)
      const layers = createPsdLayers()
      if (!layers || layers.length === 0) throw new Error('没有可导出的图层')
      downloadBlob(layeredPsdBlob(canvas.width, canvas.height, imageData, layers), `${getBaseName(source.file)}_stitched.psd`)
      message.success('PSD 已下载')
    } catch (error) {
      message.error(String(error))
    }
  }

  const downloadGodotPackage = async () => {
    if (!source) return
    const exportTiles = completedTiles()
    const bounds = exportBoundsFor(exportTiles)
    if (!bounds) return

    try {
      const zip = new JSZip()
      const imageFolder = zip.folder('images')
      if (!imageFolder) throw new Error('Godot 图片目录创建失败')
      const manifestTiles: Array<{
        key: string
        name: string
        image: string
        pixel: { x: number; y: number; width: number; height: number }
        tile: Tile
        feather: Feather
      }> = []

      for (const tile of exportTiles) {
        const key = tileKey(tile.x, tile.y)
        const image = completedImageForKey(key)
        if (!image) continue
        await ensureImageReady(image.image)
        const width = Math.max(1, Math.round(tile.w * bounds.unitW))
        const height = Math.max(1, Math.round(tile.h * bounds.unitH))
        const feather = featherForKey(key)
        const tileCanvas = document.createElement('canvas')
        tileCanvas.width = width
        tileCanvas.height = height
        const tileCtx = tileCanvas.getContext('2d')
        if (!tileCtx) continue
        tileCtx.imageSmoothingEnabled = false
        const drawable =
          key === CENTER_KEY || (feather.top === 0 && feather.right === 0 && feather.bottom === 0 && feather.left === 0)
            ? image.image
            : featheredImageCanvas(image.image, width, height, feather)
        tileCtx.drawImage(drawable, 0, 0, width, height)
        const blob = await canvasToBlob(tileCanvas)
        const name = key === CENTER_KEY ? 'source' : `tile_${key.replace(',', '_')}`
        const filename = `images/${safeFilename(name)}.png`
        imageFolder.file(filename.replace(/^images\//, ''), blob)
        manifestTiles.push({
          key,
          name,
          image: filename,
          pixel: {
            x: Math.round((tile.x - bounds.minX) * bounds.unitW),
            y: Math.round((tile.y - bounds.minY) * bounds.unitH),
            width,
            height,
          },
          tile,
          feather,
        })
      }

      zip.file(
        'map_stitch_godot.json',
        JSON.stringify(
          {
            version: 1,
            coordinate_system: 'top_left_origin_y_down_pixels',
            canvas: { width: bounds.width, height: bounds.height },
            source: {
              file: source.file.name,
              width: source.width,
              height: source.height,
            },
            overlap: {
              horizontal_percent: horizontalOverlapPercent,
              vertical_percent: verticalOverlapPercent,
            },
            tiles: manifestTiles,
          },
          null,
          2,
        ),
      )
      zip.file('map_stitch_godot.tscn', godotSceneTscn(safeFilename(getBaseName(source.file)), manifestTiles))
      const blob = await zip.generateAsync({ type: 'blob' })
      downloadBlob(blob, `${safeFilename(getBaseName(source.file))}_godot_package.zip`)
      message.success('Godot 打包已下载')
    } catch (error) {
      message.error(`Godot 打包失败：${String(error)}`)
    }
  }

  const downloadEditStateJson = async () => {
    if (!source) return
    try {
      const zip = new JSZip()
      const imageFolder = zip.folder('images')
      if (!imageFolder) throw new Error('拼接状态图片目录创建失败')

      const usedImagePaths = new Set<string>()
      const addImageToArchive = (image: LoadedImage, baseName: string): SavedImageState => {
        const path = uniqueArchiveImagePath(usedImagePaths, baseName, image.file)
        imageFolder.file(path.replace(/^images\//, ''), image.file)
        return {
          fileName: image.file.name,
          type: image.file.type,
          size: image.file.size,
          width: image.width,
          height: image.height,
          path,
        }
      }

      const uploads = Object.entries(tileUploads).flatMap(([key, upload]) => {
        if (!upload) return []
        return [[key, addImageToArchive(upload, `tile_${key.replace(',', '_')}_${upload.file.name}`)] as const]
      })

      const state = {
        version: STATE_ARCHIVE_VERSION,
        savedAt: new Date().toISOString(),
        format: 'pixelwork-map-stitch-state',
        source: addImageToArchive(source, `source_${source.file.name}`),
        tiles,
        tileUploads: Object.fromEntries(uploads),
        tileFeathers,
        selectedKey,
        horizontalOverlapPercent,
        verticalOverlapPercent,
        expandSplit,
        pan,
        zoom,
        hidePreviewBorders,
        hiddenPreviewTiles,
      }

      zip.file(STATE_MANIFEST_NAME, JSON.stringify(state, null, 2))
      downloadBlob(
        await zip.generateAsync({ type: 'blob', compression: 'STORE' }),
        `${safeFilename(getBaseName(source.file))}_map_stitch_state.zip`,
      )
      message.success('拼接状态已保存')
    } catch (error) {
      message.error(`保存拼接状态失败：${String(error)}`)
    }
  }

  const applyEditState = async (state: SavedMapStitchState, zip?: JSZip) => {
    if ((state.version !== 1 && state.version !== STATE_ARCHIVE_VERSION) || !state.source) {
      throw new Error('不是有效的拼接状态文件')
    }

    const nextSource = await loadImageFile(await savedImageStateToFile(
      state.source,
      'map_tile.png',
      zip,
    ))

    const uploads = await Promise.all(
      Object.entries(state.tileUploads ?? {}).map(async ([key, upload]) => {
        const loaded = await loadImageFile(await savedImageStateToFile(
          upload,
          `tile_${key.replace(',', '_')}.png`,
          zip,
        ))
        return [key, loaded] as const
      }),
    )

    const nextTiles = normalizeTilesState(state.tiles)
    const nextTileUploads = Object.fromEntries(uploads) as Partial<Record<TileKey, LoadedImage>>
    const nextExpandSplit: ExpandSplit = state.expandSplit === 8 || state.expandSplit === 12 ? state.expandSplit : 4
    const nextPan = isRecord(state.pan)
      ? { x: numberOr(state.pan.x, 0), y: numberOr(state.pan.y, 0) }
      : { x: 0, y: 0 }

    setSource((prev) => {
      if (prev) URL.revokeObjectURL(prev.url)
      return nextSource
    })
    setTileUploads((prev) => {
      Object.values(prev).forEach((item) => item && URL.revokeObjectURL(item.url))
      return nextTileUploads
    })
    setTiles(nextTiles)
    setTileFeathers(normalizeFeathersState(state.tileFeathers))
    setSelectedKey(typeof state.selectedKey === 'string' ? state.selectedKey : null)
    setHorizontalOverlapPercent(Math.max(0, Math.min(50, numberOr(state.horizontalOverlapPercent, 15))))
    setVerticalOverlapPercent(Math.max(0, Math.min(50, numberOr(state.verticalOverlapPercent, 15))))
    setExpandSplit(nextExpandSplit)
    setPan(nextPan)
    setZoom(clampZoom(numberOr(state.zoom, 1)))
    setHidePreviewBorders(typeof state.hidePreviewBorders === 'boolean' ? state.hidePreviewBorders : false)
    setHiddenPreviewTiles(booleanRecord(state.hiddenPreviewTiles))
    setProcessingKey(null)
    setPendingUploadKey(null)
  }

  const loadEditStateJson = async (file: File | null) => {
    if (!file) return
    try {
      if (file.name.toLowerCase().endsWith('.zip') || file.type === 'application/zip' || file.type === 'application/x-zip-compressed') {
        const zip = await JSZip.loadAsync(file)
        const manifest = zip.file(STATE_MANIFEST_NAME)
        if (!manifest) throw new Error('拼接状态 ZIP 缺少 map_stitch_state.json')
        await applyEditState(JSON.parse(await manifest.async('text')) as SavedMapStitchState, zip)
      } else {
        await applyEditState(JSON.parse(await file.text()) as SavedMapStitchState)
      }
      message.success('拼接状态已加载')
    } catch (error) {
      message.error(`加载拼接状态失败：${String(error)}`)
    }
  }

  const handlePanMouseDown = (event: React.MouseEvent<HTMLElement>) => {
    if (event.button !== 2) return
    event.preventDefault()
    panDragRef.current = { startX: event.clientX, startY: event.clientY, panX: pan.x, panY: pan.y }
  }

  const handlePanMouseMove = (event: React.MouseEvent<HTMLElement>) => {
    const drag = panDragRef.current
    if (!drag) return
    event.preventDefault()
    setPan({
      x: drag.panX + event.clientX - drag.startX,
      y: drag.panY + event.clientY - drag.startY,
    })
  }

  const stopPan = () => {
    panDragRef.current = null
  }

  const handleWorkspaceWheel = (event: ReactWheelEvent<HTMLElement>) => {
    if (!stage) return
    event.preventDefault()
    const rect = event.currentTarget.getBoundingClientRect()
    const oldZoom = zoom
    const deltaPx = event.deltaMode === 1 ? event.deltaY * 16 : event.deltaMode === 2 ? event.deltaY * rect.height : event.deltaY
    const nextZoom = clampZoom(oldZoom * Math.exp(-deltaPx * WHEEL_ZOOM_SENSITIVITY))
    if (Math.abs(nextZoom - oldZoom) < 0.0001) return

    const baseLeft = rect.left + (rect.width - stage.stageW * oldZoom) / 2
    const baseTop = rect.top + (rect.height - stage.stageH * oldZoom) / 2
    const worldX = (event.clientX - baseLeft - pan.x) / oldZoom
    const worldY = (event.clientY - baseTop - pan.y) / oldZoom
    const nextBaseLeft = rect.left + (rect.width - stage.stageW * nextZoom) / 2
    const nextBaseTop = rect.top + (rect.height - stage.stageH * nextZoom) / 2

    setZoom(nextZoom)
    setPan({
      x: event.clientX - nextBaseLeft - worldX * nextZoom,
      y: event.clientY - nextBaseTop - worldY * nextZoom,
    })
  }

  return (
    <div className="map-stitch-shell" ref={hostRef}>
      <div className="map-stitch-float-actions">
        <Button
          className="map-stitch-generate-float"
          type="primary"
          icon={<UploadOutlined />}
          onClick={() => stateFileInputRef.current?.click()}
        >
          加载拼接状态
        </Button>
        <Button
          className="map-stitch-generate-float"
          type="primary"
          disabled={!source}
          icon={<DownloadOutlined />}
          onClick={() => void downloadEditStateJson()}
        >
          保存拼接状态
        </Button>
        <Button
          className="map-stitch-generate-float"
          type="primary"
          disabled={!source}
          icon={<DownloadOutlined />}
          onClick={() => void downloadGodotPackage()}
        >
          godot打包下载
        </Button>
      </div>
      <header className="map-stitch-toolbar">
        {onBack ? (
          <Button type="text" icon={<ArrowLeftOutlined />} onClick={onBack}>
            返回首页
          </Button>
        ) : (
          <span />
        )}
        <div className="map-stitch-title">
          <Title level={4}>地图拼接</Title>
          <Text type="secondary" className="map-stitch-title-hint">
            右键拖动画布，滚轮缩放视图
          </Text>
        </div>
        <Space wrap>
          <div className="map-stitch-split-control">
            <Text type="secondary">扩图细分数</Text>
            <Select<ExpandSplit>
              value={expandSplit}
              onChange={setExpandSplit}
              popupClassName="map-stitch-split-popup"
              getPopupContainer={(trigger) => trigger.parentElement ?? document.body}
              options={[
                { value: 4, label: '4分扩图' },
                { value: 8, label: '8分扩图' },
                { value: 12, label: '12分扩图' },
              ]}
            />
          </div>
          <div className="map-stitch-overlap-control">
            <Text type="secondary">左右重叠 {horizontalOverlapPercent}%</Text>
            <Slider
              min={0}
              max={50}
              step={1}
              value={horizontalOverlapPercent}
              onChange={setHorizontalOverlapPercent}
              tooltip={{ formatter: (value) => `${value}%` }}
            />
          </div>
          <div className="map-stitch-overlap-control">
            <Text type="secondary">上下重叠 {verticalOverlapPercent}%</Text>
            <Slider
              min={0}
              max={50}
              step={1}
              value={verticalOverlapPercent}
              onChange={setVerticalOverlapPercent}
              tooltip={{ formatter: (value) => `${value}%` }}
            />
          </div>
          <Button icon={<UploadOutlined />} onClick={() => fileInputRef.current?.click()}>
            {source ? '重新导入图片' : '导入图片'}
          </Button>
          <Button icon={<DownloadOutlined />} disabled={!source} onClick={() => void downloadStitchedPng()}>
            下载全部PNG
          </Button>
          <Button icon={<DownloadOutlined />} disabled={!source} onClick={() => void downloadStitchedPsd()}>
            下载PSD
          </Button>
        </Space>
      </header>

      <input
        ref={fileInputRef}
        type="file"
        accept="image/png,image/jpeg,image/webp"
        hidden
        onChange={(event) => {
          void selectSource(event.target.files?.[0] ?? null)
          event.currentTarget.value = ''
        }}
      />

      <input
        ref={tileFileInputRef}
        type="file"
        accept="image/png,image/jpeg,image/webp"
        hidden
        onChange={(event) => {
          if (pendingUploadKey) {
            void selectTileUpload(pendingUploadKey, event.target.files?.[0] ?? null)
          }
          event.currentTarget.value = ''
          setPendingUploadKey(null)
        }}
      />

      <input
        ref={stateFileInputRef}
        type="file"
        accept="application/zip,application/x-zip-compressed,application/json,.zip,.json"
        hidden
        onChange={(event) => {
          void loadEditStateJson(event.target.files?.[0] ?? null)
          event.currentTarget.value = ''
        }}
      />

      {!source || !stage ? (
        <main
          className="map-stitch-empty"
          onClick={() => fileInputRef.current?.click()}
          onDragOver={(event) => event.preventDefault()}
          onDrop={(event) => {
            event.preventDefault()
            void selectSource(event.dataTransfer.files?.[0] ?? null)
          }}
        >
          <UploadOutlined />
          <Title level={3}>导入地图图片</Title>
          <Button type="primary" size="large" icon={<UploadOutlined />}>
            点击上传图片
          </Button>
          <Text type="secondary">也可以把 PNG / JPG / WebP 拖到这里上传。</Text>
        </main>
      ) : (
        <main
          className="map-stitch-workspace"
          onContextMenu={(event) => event.preventDefault()}
          onMouseDown={handlePanMouseDown}
          onMouseMove={handlePanMouseMove}
          onMouseUp={stopPan}
          onMouseLeave={stopPan}
          onWheel={handleWorkspaceWheel}
        >
          <div
            className="map-stitch-stage-shell"
            style={{
              width: stage.stageW * stage.scale,
              height: stage.stageH * stage.scale,
              transform: `translate(${pan.x}px, ${pan.y}px)`,
            }}
          >
            <div
              className={`map-stitch-stage ${hidePreviewBorders ? 'map-stitch-stage-hide-borders' : ''}`}
              style={{
                width: stage.stageW,
                height: stage.stageH,
                transform: `scale(${stage.scale})`,
              }}
            >
              <div
                className="map-stitch-source-frame"
                style={{
                  ...tilePosition(CENTER_TILE),
                  width: stage.unitW,
                  height: stage.unitH,
                }}
                onClick={() => setSelectedKey(CENTER_KEY)}
              >
                <img src={source.url} alt="原图" />
                <span
                  className="map-stitch-frame-label"
                  style={{ transform: `translate(-50%, -50%) scale(${1 / stage.scale})` }}
                >
                  主图
                  <strong>点击操作</strong>
                  <small>整体层已就绪，可处理或扩展</small>
                </span>
                {selectedKey === CENTER_KEY ? (
                  <span
                    className="map-stitch-frame-actions map-stitch-frame-actions-wide"
                    style={{ transform: `translate(-50%, -50%) scale(${1 / stage.scale})` }}
                    onClick={(event) => event.stopPropagation()}
                  >
                    <Button className="map-stitch-action-btn" icon={<UploadOutlined />} onClick={() => fileInputRef.current?.click()}>
                      上传整体层
                    </Button>
                    <Button className="map-stitch-action-btn" type="primary" icon={<DownloadOutlined />} onClick={() => void downloadStitchedPng()}>
                      下载
                    </Button>
                    <Button className="map-stitch-action-btn" icon={<ExpandOutlined />} onClick={() => expandFromTile(CENTER_KEY)}>
                      扩展
                    </Button>
                  </span>
                ) : null}
              </div>

              {Object.entries(tiles).map(([key, tile]) => {
                const tileId = key as TileKey
                const upload = tileUploads[tileId]
                const isSelected = selectedKey === tileId
                const tone = toneForTile(tile)
                const rect = tileRect(tile)
                const feather = featherForKey(tileId)
                const isImageHidden = Boolean(hiddenPreviewTiles[tileId])
                return (
                  <div
                    key={tileId}
                    role="button"
                    tabIndex={0}
                    className={`map-stitch-neighbor-frame map-stitch-neighbor-${tone} ${upload ? 'map-stitch-neighbor-uploaded' : ''} ${isSelected ? 'selected' : ''} ${isImageHidden ? 'map-stitch-tile-image-hidden' : ''}`}
                    style={{
                      ...tilePosition(tile),
                      width: rect?.width ?? stage.unitW,
                      height: rect?.height ?? stage.unitH,
                    }}
                    onClick={(event) => {
                      event.preventDefault()
                      setSelectedKey(tileId)
                    }}
                    onKeyDown={(event) => {
                      if (event.key !== 'Enter' && event.key !== ' ') return
                      event.preventDefault()
                      setSelectedKey(tileId)
                    }}
                  >
                    {upload && rect ? (
                      <FeatheredPreviewImage upload={upload} width={rect.width} height={rect.height} feather={feather} alt={`${labelForTile(tile)} preview`} />
                    ) : upload ? (
                      <img src={upload.url} alt={`${labelForTile(tile)}预览`} />
                    ) : null}
                    <span
                      className="map-stitch-frame-label"
                      style={{ transform: `translate(-50%, -50%) scale(${1 / stage.scale})` }}
                    >
                      {labelForTile(tile)}
                      {!isSelected && <strong>{upload ? '点击操作' : '点击激活'}</strong>}
                      <small>{upload ? '整体层已就绪，可处理或扩展' : '当前：整体层，可上传或生成'}</small>
                    </span>
                    {isSelected && (
                      <span
                        className="map-stitch-frame-actions map-stitch-frame-actions-wide"
                        style={{ transform: `translate(-50%, -50%) scale(${1 / stage.scale})` }}
                        onClick={(event) => event.stopPropagation()}
                      >
                        <Button
                          className="map-stitch-action-btn"
                          icon={<UploadOutlined />}
                          onMouseDown={(event) => event.stopPropagation()}
                          onClick={(event) => {
                            event.stopPropagation()
                            openTileUpload(tileId)
                          }}
                        >
                          上传整体层
                        </Button>
                        <Button className="map-stitch-action-btn" type="primary" icon={<DownloadOutlined />} onClick={() => void downloadTileTemplate(tileId)}>
                          下载
                        </Button>
                        <Button className="map-stitch-action-btn" type="primary" onClick={() => void openApiGenerate(tileId)}>
                          API生成
                        </Button>
                        <Button className="map-stitch-action-btn" icon={<ExpandOutlined />} onClick={() => expandFromTile(tileId)}>
                          扩展
                        </Button>
                        <Button className="map-stitch-action-btn" onClick={() => void openRegionDraw(tileId)}>
                          区域绘制
                        </Button>
                        {upload && (
                          <>
                            <Button
                              className="map-stitch-action-btn"
                              loading={processingKey === tileId}
                              onClick={() => void removeWatermarkForTile(tileId)}
                            >
                              去水印
                            </Button>
                            <Button
                              className="map-stitch-unload-btn"
                              danger
                              icon={<DeleteOutlined />}
                              title="卸载图片"
                              onClick={(event) => {
                                event.stopPropagation()
                                unloadTileUpload(tileId)
                              }}
                            >
                              卸载
                            </Button>
                          </>
                        )}
                        {upload && (
                          <span className="map-stitch-feather-actions">
                            <Button
                              className="map-stitch-feather-btn map-stitch-feather-top"
                              icon={<ArrowUpOutlined />}
                              title="左键增加羽化，右键减少羽化"
                              onClick={() => increaseFeather(tileId, 'top')}
                              onContextMenu={(event) => {
                                event.preventDefault()
                                decreaseFeather(tileId, 'top')
                              }}
                            >
                              {feather.top}%
                            </Button>
                            <span className="map-stitch-feather-cell map-stitch-feather-left">
                              <Button
                                className="map-stitch-feather-btn"
                                icon={<ArrowLeftOutlined />}
                                title="左键增加羽化，右键减少羽化"
                                onClick={() => increaseFeather(tileId, 'left')}
                                onContextMenu={(event) => {
                                  event.preventDefault()
                                  decreaseFeather(tileId, 'left')
                                }}
                              >
                                {feather.left}%
                              </Button>
                              <Button
                                className="map-stitch-feather-image-toggle"
                                icon={isImageHidden ? <EyeOutlined /> : <EyeInvisibleOutlined />}
                                title={isImageHidden ? '显示图片' : '隐藏图片'}
                                onClick={(event) => {
                                  event.stopPropagation()
                                  togglePreviewTileVisibility(tileId)
                                }}
                              />
                            </span>
                            <Button
                              className="map-stitch-feather-border-toggle"
                              type={hidePreviewBorders ? 'primary' : 'default'}
                              onClick={() => setHidePreviewBorders((prev) => !prev)}
                            >
                              不显示边框
                            </Button>
                            <Button
                              className="map-stitch-feather-btn map-stitch-feather-right"
                              icon={<ArrowRightOutlined />}
                              title="左键增加羽化，右键减少羽化"
                              onClick={() => increaseFeather(tileId, 'right')}
                              onContextMenu={(event) => {
                                event.preventDefault()
                                decreaseFeather(tileId, 'right')
                              }}
                            >
                              {feather.right}%
                            </Button>
                            <span className="map-stitch-feather-cell map-stitch-feather-bottom">
                              <Button
                                className="map-stitch-feather-btn"
                                icon={<ArrowDownOutlined />}
                                title="左键增加羽化，右键减少羽化"
                                onClick={() => increaseFeather(tileId, 'bottom')}
                                onContextMenu={(event) => {
                                  event.preventDefault()
                                  decreaseFeather(tileId, 'bottom')
                                }}
                              >
                                {feather.bottom}%
                              </Button>
                              <Button
                                className="map-stitch-feather-image-toggle"
                                icon={isImageHidden ? <EyeOutlined /> : <EyeInvisibleOutlined />}
                                title={isImageHidden ? '显示图片' : '隐藏图片'}
                                onClick={(event) => {
                                  event.stopPropagation()
                                  togglePreviewTileVisibility(tileId)
                                }}
                              />
                            </span>
                          </span>
                        )}
                      </span>
                    )}
                  </div>
                )
              })}
            </div>
          </div>
        </main>
      )}
      <MapStitchApiModal
        open={Boolean(apiModalKey)}
        tileKey={apiModalKey ?? ''}
        templateFile={apiTemplateFile}
        templatePreviewUrl={apiTemplateUrl}
        onCancel={() => {
          setApiModalKey(null)
          if (apiTemplateUrl) URL.revokeObjectURL(apiTemplateUrl)
          setApiTemplateUrl(null)
          setApiTemplateFile(null)
        }}
        onGenerated={({ file }) => {
          const key = apiModalKey
          setApiModalKey(null)
          if (apiTemplateUrl) URL.revokeObjectURL(apiTemplateUrl)
          setApiTemplateUrl(null)
          setApiTemplateFile(null)
          if (key) void selectTileUpload(key, file)
        }}
      />
      <MapStitchRegionEditor
        open={Boolean(regionKey)}
        imageUrl={regionImageUrl}
        width={regionSize.w}
        height={regionSize.h}
        onCancel={() => {
          setRegionKey(null)
          if (regionImageUrl) URL.revokeObjectURL(regionImageUrl)
          setRegionImageUrl(null)
        }}
        onConfirm={(blob) => {
          const key = regionKey
          if (regionImageUrl) URL.revokeObjectURL(regionImageUrl)
          setRegionImageUrl(null)
          setRegionKey(null)
          if (!key) return
          const file = new File([blob], `region_${key.replace(',', '_')}.png`, { type: 'image/png' })
          const url = URL.createObjectURL(blob)
          setApiTemplateFile(file)
          setApiTemplateUrl(url)
          setApiModalKey(key)
        }}
      />
    </div>
  )
}
