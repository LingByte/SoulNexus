export const BINARY_PREFIX = 'base64:'

const BINARY_EXT = /\.(png|jpe?g|webp|gif|moc3|bin|wav|mp3)$/i

export function isBinaryProjectFile(path: string, content: string): boolean {
  return content.startsWith(BINARY_PREFIX) || BINARY_EXT.test(path)
}

export function shouldStoreAsBinary(path: string, file: File): boolean {
  return BINARY_EXT.test(path) || BINARY_EXT.test(file.name) || !file.type.startsWith('text/')
}

function readFileAsBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const result = reader.result
      if (typeof result !== 'string') {
        reject(new Error('读取文件失败'))
        return
      }
      const comma = result.indexOf(',')
      resolve(comma >= 0 ? result.slice(comma + 1) : result)
    }
    reader.onerror = () => reject(reader.error ?? new Error('读取文件失败'))
    reader.readAsDataURL(file)
  })
}

/** Encode a dropped/uploaded file for project storage (text or base64:). */
export async function encodeFileForProject(path: string, file: File): Promise<string> {
  if (shouldStoreAsBinary(path, file)) {
    return BINARY_PREFIX + (await readFileAsBase64(file))
  }
  return file.text()
}

function readAllDirectoryEntries(reader: FileSystemDirectoryReader): Promise<FileSystemEntry[]> {
  return new Promise((resolve, reject) => {
    const acc: FileSystemEntry[] = []
    const readBatch = () => {
      reader.readEntries(
        (entries) => {
          if (!entries.length) resolve(acc)
          else {
            acc.push(...entries)
            readBatch()
          }
        },
        (err) => reject(err),
      )
    }
    readBatch()
  })
}

async function walkFileEntry(
  entry: FileSystemEntry,
  prefix: string,
  out: Map<string, File>,
): Promise<void> {
  if (entry.isFile) {
    const file = await new Promise<File>((resolve, reject) => {
      ;(entry as FileSystemFileEntry).file(resolve, reject)
    })
    const rel = prefix ? `${prefix}/${file.name}` : file.name
    out.set(rel.replace(/\\/g, '/'), file)
    return
  }
  if (entry.isDirectory) {
    const reader = (entry as FileSystemDirectoryEntry).createReader()
    const children = await readAllDirectoryEntries(reader)
    const nextPrefix = prefix ? `${prefix}/${entry.name}` : entry.name
    for (const child of children) {
      await walkFileEntry(child, nextPrefix, out)
    }
  }
}

/** Collect files from drag-and-drop; relative paths joined under basePrefix. */
export async function collectFilesFromDataTransfer(
  dt: DataTransfer,
  basePrefix = 'assets/sprites/',
): Promise<Map<string, File>> {
  const out = new Map<string, File>()
  const prefix = basePrefix.replace(/\/+$/, '') + '/'

  const items = dt.items
  if (items && items.length > 0) {
    const entries: FileSystemEntry[] = []
    for (let i = 0; i < items.length; i++) {
      const item = items[i]
      if (item.kind !== 'file') continue
      const entry = item.webkitGetAsEntry?.()
      if (entry) entries.push(entry)
      else {
        const file = item.getAsFile()
        if (file) out.set(prefix + file.name, file)
      }
    }
    for (const entry of entries) {
      const nested = new Map<string, File>()
      await walkFileEntry(entry, '', nested)
      for (const [rel, file] of nested) {
        out.set(prefix + rel.replace(/^\/+/, ''), file)
      }
    }
    return out
  }

  for (let i = 0; i < dt.files.length; i++) {
    const file = dt.files[i]
    const rel = (file as File & { webkitRelativePath?: string }).webkitRelativePath || file.name
    out.set(prefix + rel.replace(/^\/+/, ''), file)
  }
  return out
}

/** Encode all dropped files to project file map. */
export async function encodeDroppedFiles(
  files: Map<string, File>,
): Promise<Record<string, string>> {
  const out: Record<string, string> = {}
  for (const [path, file] of files) {
    out[path] = await encodeFileForProject(path, file)
  }
  return out
}

export function mimeForProjectPath(path: string): string {
  const lower = path.toLowerCase()
  if (lower.endsWith('.png')) return 'image/png'
  if (lower.endsWith('.jpg') || lower.endsWith('.jpeg')) return 'image/jpeg'
  if (lower.endsWith('.webp')) return 'image/webp'
  if (lower.endsWith('.gif')) return 'image/gif'
  if (lower.endsWith('.json')) return 'application/json'
  return 'application/octet-stream'
}

export function toDataUrl(path: string, content: string): string | null {
  if (!content.startsWith(BINARY_PREFIX)) return null
  const b64 = content.slice(BINARY_PREFIX.length)
  return `data:${mimeForProjectPath(path)};base64,${b64}`
}

export function formatBinarySize(content: string): string {
  if (!content.startsWith(BINARY_PREFIX)) return `${content.length} chars`
  const bytes = Math.floor((content.length - BINARY_PREFIX.length) * 3 / 4)
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
