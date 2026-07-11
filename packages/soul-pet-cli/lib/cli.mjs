import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawn } from 'node:child_process'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const ROOT = path.resolve(__dirname, '..')
const LINK_FILE = '.soulpet/link.json'
const CONFIG_FILE = '.soulpet/config.json'

export function cwd() {
  return process.cwd()
}

export function linkPath() {
  return path.join(cwd(), LINK_FILE)
}

export function configPath() {
  return path.join(cwd(), CONFIG_FILE)
}

export function readJson(p, fallback = null) {
  try {
    return JSON.parse(fs.readFileSync(p, 'utf8'))
  } catch {
    return fallback
  }
}

export function writeJson(p, data) {
  fs.mkdirSync(path.dirname(p), { recursive: true })
  fs.writeFileSync(p, JSON.stringify(data, null, 2), 'utf8')
}

export function loadLink() {
  return readJson(linkPath()) || readJson(configPath())
}

export function saveLink(data) {
  writeJson(linkPath(), data)
  writeJson(configPath(), { ...readJson(configPath(), {}), ...data })
}

const BINARY_EXT = /\.(png|jpe?g|webp|gif|moc3|bin|wav|mp3)$/i

export function isBinaryFile(name) {
  return BINARY_EXT.test(name)
}

export function encodeFile(rel, buf) {
  if (isBinaryFile(rel) || !isUtf8(buf)) {
    return 'base64:' + buf.toString('base64')
  }
  return buf.toString('utf8')
}

function isUtf8(buf) {
  try {
    new TextDecoder('utf-8', { fatal: true }).decode(buf)
    return true
  } catch {
    return false
  }
}

export function readDirPackage(root) {
  const files = {}
  function walk(dir, prefix = '') {
    for (const name of fs.readdirSync(dir)) {
      if (name.startsWith('.') && name !== '.soulpet') continue
      const full = path.join(dir, name)
      const rel = prefix ? `${prefix}/${name}` : name
      const st = fs.statSync(full)
      if (st.isDirectory()) {
        walk(full, rel)
      } else {
        files[rel.replace(/\\/g, '/')] = encodeFile(rel, fs.readFileSync(full))
      }
    }
  }
  walk(root)
  if (!files['manifest.json']) throw new Error('missing manifest.json')
  if (!files['pet.js']) throw new Error('missing pet.js')
  return files
}

export function writeDirPackage(root, files) {
  for (const [rel, body] of Object.entries(files)) {
    const dest = path.join(root, rel)
    fs.mkdirSync(path.dirname(dest), { recursive: true })
    let data
    if (body.startsWith('base64:')) {
      data = Buffer.from(body.slice(7), 'base64')
    } else {
      data = Buffer.from(body, 'utf8')
    }
    fs.writeFileSync(dest, data)
  }
}

export function templateDir(name) {
  const repoExamples = path.resolve(ROOT, '../../examples/soulpet', name)
  if (fs.existsSync(repoExamples)) return repoExamples
  return path.join(ROOT, 'templates', name)
}

export function copyDir(src, dest) {
  fs.mkdirSync(dest, { recursive: true })
  for (const name of fs.readdirSync(src)) {
    if (name.startsWith('.')) continue
    const s = path.join(src, name)
    const d = path.join(dest, name)
    if (fs.statSync(s).isDirectory()) copyDir(s, d)
    else fs.copyFileSync(s, d)
  }
}

export function apiBase() {
  const link = loadLink()
  const env = process.env.SOUL_PET_SERVER || process.env.SOUL_PET_API
  return (link?.serverBase || env || 'http://127.0.0.1:7072/api').replace(/\/+$/, '')
}

export function authToken() {
  return process.env.SOUL_PET_TOKEN || loadLink()?.token || ''
}

export async function apiFetch(pathname, opts = {}) {
  const url = apiBase() + pathname
  const headers = { ...(opts.headers || {}) }
  const token = authToken()
  if (token) headers.Authorization = `Bearer ${token}`
  if (opts.body && !(opts.body instanceof FormData)) {
    headers['Content-Type'] = 'application/json'
  }
  const res = await fetch(url, { ...opts, headers })
  const json = await res.json().catch(() => ({}))
  if (!res.ok || json.success === false) {
    const msg = json.message || json.msg || res.statusText
    throw new Error(typeof msg === 'string' ? msg : JSON.stringify(json))
  }
  return json
}

export function packZip(files) {
  // minimal zip via PowerShell/unzip fallback — use store-only for dev: write files + manifest for import API instead
  return files
}

export async function cmdInit(args) {
  const name = args[0] || 'my-pet'
  const template = args.includes('--template')
    ? args[args.indexOf('--template') + 1]
    : 'live2d-stub'
  const dest = path.resolve(cwd(), name)
  if (fs.existsSync(dest)) throw new Error(`already exists: ${dest}`)
  const src = templateDir(template)
  if (!fs.existsSync(src)) throw new Error(`unknown template: ${template}`)
  copyDir(src, dest)
  console.log(`[soul-pet] initialized ${template} → ${dest}`)
  console.log('  next: cd', name, '&& soul-pet validate')
}

export async function cmdValidate() {
  const files = readDirPackage(cwd())
  const json = await apiFetch('/pet-packages/validate', {
    method: 'POST',
    body: JSON.stringify({ files, entry: 'pet.js' }),
  })
  const data = json.data || json
  console.log('[soul-pet] kind:', data.kind, 'valid:', data.valid)
  for (const issue of data.issues || []) {
    console.log(`  [${issue.level}] ${issue.field}: ${issue.message}`)
  }
  if (!data.valid) process.exit(1)
}

export async function cmdPush(args) {
  const files = readDirPackage(cwd())
  const link = loadLink() || {}
  const create = args.includes('--create')
  const nameFlag = args.includes('--name') ? args[args.indexOf('--name') + 1] : null
  const messageFlag = args.includes('--message') ? args[args.indexOf('--message') + 1] : null
  const noBump = args.includes('--no-bump')

  if (create || !link.templateId) {
    const form = new FormData()
    const zipPath = path.join(cwd(), '.soulpet-upload.zip')
    await import('./zip.mjs').then(({ writeZip }) => writeZip(zipPath, files))
    form.append('package', new Blob([fs.readFileSync(zipPath)]), 'package.zip')
    if (nameFlag) form.append('name', nameFlag)
    fs.unlinkSync(zipPath)
    const json = await apiFetch('/pet-packages/import', { method: 'POST', body: form })
    const d = json.data || json
    const tpl = d.template
    saveLink({
      serverBase: apiBase(),
      templateId: tpl.id,
      jsSourceId: tpl.jsSourceId,
      name: tpl.name,
    })
    console.log('[soul-pet] created jsSourceId:', tpl.jsSourceId)
    return
  }

  const payload = {
    files,
    entry: 'pet.js',
    name: nameFlag || link.name,
  }
  if (messageFlag) payload.changeNote = messageFlag
  if (!noBump && messageFlag) payload.bumpVersion = true

  const json = await apiFetch(`/js-templates/${link.templateId}/push`, {
    method: 'PUT',
    body: JSON.stringify(payload),
  })
  const d = json.data || json
  const ver = d.packageVersion ? ` v${d.packageVersion}` : ''
  console.log('[soul-pet] pushed', d.jsSourceId || link.jsSourceId, ver)
}

export async function cmdPull(args) {
  const link = loadLink() || {}
  const id = args.includes('--id') ? args[args.indexOf('--id') + 1] : link.templateId
  if (!id) throw new Error('missing templateId — use --id or push --create first')
  const json = await apiFetch(`/js-templates/${id}/pull`)
  const d = json.data || json
  writeDirPackage(cwd(), d.files)
  saveLink({
    serverBase: apiBase(),
    templateId: d.templateId,
    jsSourceId: d.jsSourceId,
    name: d.name,
  })
  console.log('[soul-pet] pulled', d.name, d.jsSourceId)
}

export async function cmdPublish(args) {
  const files = readDirPackage(cwd())
  const form = new FormData()
  const zipPath = path.join(cwd(), '.soulpet-upload.zip')
  const { writeZip } = await import('./zip.mjs')
  writeZip(zipPath, files)
  form.append('package', new Blob([fs.readFileSync(zipPath)]), 'package.zip')
  const name = args.includes('--name') ? args[args.indexOf('--name') + 1] : undefined
  if (name) form.append('name', name)
  fs.unlinkSync(zipPath)
  const json = await apiFetch('/pet-market/listings', { method: 'POST', body: form })
  const d = json.data || json
  console.log('[soul-pet] published marketId:', d.marketId)
}

export async function cmdDev(args) {
  const port = args.includes('--port') ? Number(args[args.indexOf('--port') + 1]) : 5179
  const preview = path.join(ROOT, 'lib/dev-server.mjs')
  const child = spawn(process.execPath, [preview, String(port)], {
    cwd: cwd(),
    stdio: 'inherit',
    env: { ...process.env, SOUL_PET_DEV_ROOT: cwd() },
  })
  child.on('exit', (code) => process.exit(code ?? 0))
}

export async function run(argv) {
  const args = argv.slice(2)
  const cmd = args[0]
  if (!cmd || cmd === '--help' || cmd === '-h') {
    console.log(`Soul Pet CLI

Usage:
  soul-pet init [dir] [--template sprite-ghost|live2d-stub]
  soul-pet validate
  soul-pet push [--create] [--name "名称"] [--message "变更说明"] [--no-bump]
  soul-pet pull [--id templateId]
  soul-pet publish [--name "名称"]
  soul-pet dev [--port 5179]

Env: SOUL_PET_TOKEN, SOUL_PET_SERVER
Config: .soulpet/link.json
`)
    return
  }
  const map = {
    init: cmdInit,
    validate: cmdValidate,
    push: cmdPush,
    pull: cmdPull,
    publish: cmdPublish,
    dev: cmdDev,
  }
  const fn = map[cmd]
  if (!fn) throw new Error(`unknown command: ${cmd}`)
  await fn(args.slice(1))
}
