const DB_NAME = 'soulnexus_pixel_assets_v1'
const DB_VERSION = 2
const STORE = 'assets'

export type LocalAsset = {
  id?: number
  name: string
  mimeType: string
  sizeBytes: number
  funcType: string
  matType: string
  folder: string
  style: string
  width: number | null
  height: number | null
  createdAt: string
  data: ArrayBuffer
}

function openDb(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, DB_VERSION)
    req.onerror = () => reject(req.error ?? new Error('无法打开本地素材库（IndexedDB）'))
    req.onsuccess = () => resolve(req.result)
    req.onupgradeneeded = (event) => {
      const db = (event.target as IDBOpenDBRequest).result
      if (!db.objectStoreNames.contains(STORE)) {
        const os = db.createObjectStore(STORE, { keyPath: 'id', autoIncrement: true })
        os.createIndex('funcType', 'funcType', { unique: false })
        os.createIndex('folder', 'folder', { unique: false })
      }
    }
  })
}

export async function listAssets(): Promise<LocalAsset[]> {
  const db = await openDb()
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE, 'readonly')
    const req = tx.objectStore(STORE).getAll()
    req.onsuccess = () => resolve((req.result as LocalAsset[]) ?? [])
    req.onerror = () => reject(req.error)
  })
}

export async function addAssetFromFile(
  file: File,
  meta: Partial<LocalAsset> = {},
): Promise<LocalAsset> {
  const buffer = await file.arrayBuffer()
  const record: Omit<LocalAsset, 'id'> = {
    name: meta.name ?? file.name,
    mimeType: file.type || 'application/octet-stream',
    sizeBytes: file.size,
    funcType: meta.funcType ?? '道具物品类',
    matType: meta.matType ?? '卡通极简材质',
    folder: meta.folder ?? '默认',
    style: meta.style ?? '像素风',
    width: meta.width ?? null,
    height: meta.height ?? null,
    createdAt: new Date().toISOString(),
    data: buffer,
  }
  const db = await openDb()
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE, 'readwrite')
    const req = tx.objectStore(STORE).add(record)
    req.onsuccess = () => resolve({ ...record, id: req.result as number })
    req.onerror = () => reject(req.error)
  })
}

export async function updateAsset(id: number, patch: Partial<LocalAsset>): Promise<LocalAsset> {
  const db = await openDb()
  const existing = await new Promise<LocalAsset | undefined>((resolve, reject) => {
    const tx = db.transaction(STORE, 'readonly')
    const req = tx.objectStore(STORE).get(id)
    req.onsuccess = () => resolve(req.result as LocalAsset | undefined)
    req.onerror = () => reject(req.error)
  })
  if (!existing) throw new Error('素材不存在')
  const next = { ...existing, ...patch, id }
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE, 'readwrite')
    const req = tx.objectStore(STORE).put(next)
    req.onsuccess = () => resolve(next)
    req.onerror = () => reject(req.error)
  })
}

export async function deleteAsset(id: number): Promise<void> {
  const db = await openDb()
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE, 'readwrite')
    const req = tx.objectStore(STORE).delete(id)
    req.onsuccess = () => resolve()
    req.onerror = () => reject(req.error)
  })
}

export function assetToBlob(asset: LocalAsset): Blob {
  return new Blob([asset.data], { type: asset.mimeType })
}

export async function getImageDimensions(file: Blob): Promise<{ width: number; height: number }> {
  const url = URL.createObjectURL(file)
  try {
    const img = await new Promise<HTMLImageElement>((resolve, reject) => {
      const el = new Image()
      el.onload = () => resolve(el)
      el.onerror = reject
      el.src = url
    })
    return { width: img.naturalWidth, height: img.naturalHeight }
  } finally {
    URL.revokeObjectURL(url)
  }
}

export async function saveCanvasToLibrary(
  canvas: HTMLCanvasElement,
  name = 'layer-merged.png',
  meta: Partial<LocalAsset> = {},
): Promise<LocalAsset> {
  const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/png'))
  if (!blob) throw new Error('导出画布失败')
  const file = new File([blob], name, { type: 'image/png' })
  return addAssetFromFile(file, {
    ...meta,
    width: canvas.width,
    height: canvas.height,
  })
}
