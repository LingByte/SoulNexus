export type FileTreeNode =
  | { kind: 'file'; path: string; name: string }
  | { kind: 'folder'; path: string; name: string; children: FileTreeNode[] }

const HIDDEN_FILES = new Set(['.gitkeep', '.keep'])

export function normalizeProjectPath(raw: string): string {
  return raw
    .replace(/\\/g, '/')
    .replace(/^\/+/, '')
    .replace(/\/+/g, '/')
    .trim()
}

export function validateProjectPath(path: string, isFolder = false): string | null {
  const p = normalizeProjectPath(path)
  if (!p) return '路径不能为空'
  if (p.includes('..')) return '路径不能包含 ..'
  if (p.startsWith('.') && !p.endsWith('/.gitkeep') && p !== '.gitkeep') return '不能以 . 开头'
  if (isFolder && p.includes('.')) return '文件夹路径不应包含扩展名'
  if (!isFolder && p.endsWith('/')) return '文件路径不能以 / 结尾'
  if (!/^[\w./-]+$/.test(p)) return '路径仅允许字母、数字、_、-、/'
  return null
}

export function folderMarkerPath(folder: string): string {
  const base = normalizeProjectPath(folder).replace(/\/$/, '')
  return `${base}/.gitkeep`
}

export function defaultFileContent(path: string): string {
  if (path.endsWith('.json')) return '{\n  \n}\n'
  if (path.endsWith('.css')) return '/* styles */\n'
  if (path.endsWith('.md')) return `# ${path.split('/').pop()}\n`
  if (path.endsWith('.html')) return '<!DOCTYPE html>\n<html>\n<body>\n</body>\n</html>\n'
  return `// ${path}\n`
}

function isHiddenFile(name: string): boolean {
  return HIDDEN_FILES.has(name)
}

export function isHiddenProjectFile(path: string): boolean {
  return isHiddenFile(basename(path))
}

export function buildFileTree(paths: string[]): FileTreeNode[] {
  const root: FileTreeNode[] = []
  const folderIndex = new Map<string, FileTreeNode & { kind: 'folder' }>()

  const sorted = [...paths].sort((a, b) => a.localeCompare(b))

  for (const fullPath of sorted) {
    const parts = fullPath.split('/')
    const fileName = parts[parts.length - 1]
    if (isHiddenFile(fileName) && parts.length > 1) {
      // folder marker — ensure folder node exists
      const folderPath = parts.slice(0, -1).join('/')
      ensureFolder(root, folderIndex, folderPath)
      continue
    }
    if (isHiddenFile(fileName)) continue

    let parentList = root
    let acc = ''
    for (let i = 0; i < parts.length - 1; i++) {
      acc = acc ? `${acc}/${parts[i]}` : parts[i]
      let folder = folderIndex.get(acc)
      if (!folder) {
        folder = { kind: 'folder', path: acc, name: parts[i], children: [] }
        folderIndex.set(acc, folder)
        parentList.push(folder)
      }
      parentList = folder.children
    }

    parentList.push({ kind: 'file', path: fullPath, name: fileName })
  }

  // folders that only have .gitkeep
  for (const fullPath of sorted) {
    const parts = fullPath.split('/')
    if (parts.length > 1 && isHiddenFile(parts[parts.length - 1])) {
      const folderPath = parts.slice(0, -1).join('/')
      ensureFolder(root, folderIndex, folderPath)
    }
  }

  sortTree(root)
  return root
}

function ensureFolder(
  root: FileTreeNode[],
  folderIndex: Map<string, FileTreeNode & { kind: 'folder' }>,
  folderPath: string,
) {
  const parts = folderPath.split('/')
  let parentList = root
  let acc = ''
  for (const part of parts) {
    acc = acc ? `${acc}/${part}` : part
    let folder = folderIndex.get(acc)
    if (!folder) {
      folder = { kind: 'folder', path: acc, name: part, children: [] }
      folderIndex.set(acc, folder)
      parentList.push(folder)
    }
    parentList = folder.children
  }
}

function sortTree(nodes: FileTreeNode[]) {
  nodes.sort((a, b) => {
    if (a.kind !== b.kind) return a.kind === 'folder' ? -1 : 1
    return a.name.localeCompare(b.name)
  })
  for (const n of nodes) {
    if (n.kind === 'folder') sortTree(n.children)
  }
}

export function collectFolderPaths(paths: string[]): string[] {
  const folders = new Set<string>()
  for (const p of paths) {
    const parts = p.split('/')
    for (let i = 1; i < parts.length; i++) {
      folders.add(parts.slice(0, i).join('/'))
    }
  }
  return [...folders].sort()
}

/** Core files that must stay at project root. */
export const PROTECTED_PROJECT_FILES = new Set([
  'manifest.json',
  'pet.js',
  'style.css',
  'README.md',
])

export function basename(path: string): string {
  const p = normalizeProjectPath(path)
  const i = p.lastIndexOf('/')
  return i >= 0 ? p.slice(i + 1) : p
}

export function dirname(path: string): string {
  const p = normalizeProjectPath(path)
  const i = p.lastIndexOf('/')
  return i >= 0 ? p.slice(0, i) : ''
}

export function joinProjectPath(folder: string, name: string): string {
  const base = normalizeProjectPath(folder).replace(/\/$/, '')
  return base ? `${base}/${normalizeProjectPath(name)}` : normalizeProjectPath(name)
}

export function isProtectedProjectFile(path: string): boolean {
  return PROTECTED_PROJECT_FILES.has(normalizeProjectPath(path))
}

export function moveProjectFilePath(from: string, toFolder: string): string {
  return joinProjectPath(toFolder, basename(from))
}

/** After deleting paths, add .gitkeep for folders that still have children. */
export function reconcileFolderMarkers(files: Record<string, string>): Record<string, string> {
  const next = { ...files }
  const folders = collectFolderPaths(Object.keys(next).filter((p) => !isHiddenProjectFile(p)))
  for (const folder of folders) {
    const marker = folderMarkerPath(folder)
    const hasChild = Object.keys(next).some(
      (p) => p !== marker && p.startsWith(`${folder}/`) && !isHiddenProjectFile(p),
    )
    if (hasChild && !next[marker]) next[marker] = ''
  }
  for (const key of Object.keys(next)) {
    if (!isHiddenProjectFile(key)) continue
    const folder = dirname(key)
    if (!folder) continue
    const hasChild = Object.keys(next).some(
      (p) => p !== key && p.startsWith(`${folder}/`) && !isHiddenProjectFile(p),
    )
    if (!hasChild) delete next[key]
  }
  return next
}

export function deleteProjectFile(
  files: Record<string, string>,
  path: string,
): Record<string, string> | null {
  const p = normalizeProjectPath(path)
  if (!p || isProtectedProjectFile(p) || !files[p]) return null
  const next = { ...files }
  delete next[p]
  return reconcileFolderMarkers(next)
}

export function moveProjectFile(
  files: Record<string, string>,
  from: string,
  toFolder: string,
): { files: Record<string, string>; newPath: string } | null {
  const src = normalizeProjectPath(from)
  const folder = normalizeProjectPath(toFolder).replace(/\/$/, '')
  if (!src || isProtectedProjectFile(src) || !files[src]) return null
  if (folder && validateProjectPath(folder, true)) return null
  const dest = moveProjectFilePath(src, folder)
  if (validateProjectPath(dest, false)) return null
  if (files[dest] && dest !== src) return null
  if (dirname(src) === folder) return null
  const next = { ...files }
  next[dest] = next[src]
  delete next[src]
  return { files: reconcileFolderMarkers(next), newPath: dest }
}
