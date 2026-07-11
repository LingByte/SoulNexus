/**
 * 手动下载 Electron 二进制（pnpm postinstall 失败或卡住时用）
 * 用法：pnpm run setup
 */
const { spawnSync } = require('child_process')
const fs = require('fs')
const path = require('path')

const MIRROR = process.env.ELECTRON_MIRROR || 'https://npmmirror.com/mirrors/electron/'

function findElectronInstallScript() {
  const root = path.join(__dirname, '..', 'node_modules', 'electron')
  if (fs.existsSync(path.join(root, 'install.js'))) return path.join(root, 'install.js')

  const pnpmGlob = path.join(__dirname, '..', 'node_modules', '.pnpm')
  if (!fs.existsSync(pnpmGlob)) return null
  for (const name of fs.readdirSync(pnpmGlob)) {
    if (!name.startsWith('electron@')) continue
    const candidate = path.join(pnpmGlob, name, 'node_modules', 'electron', 'install.js')
    if (fs.existsSync(candidate)) return candidate
  }
  return null
}

function electronBinaryExists() {
  try {
    const electron = require('electron')
    return typeof electron === 'string' && fs.existsSync(electron)
  } catch {
    return false
  }
}

if (electronBinaryExists()) {
  console.log('[setup] Electron 已就绪:', require('electron'))
  process.exit(0)
}

const installScript = findElectronInstallScript()
if (!installScript) {
  console.error('[setup] 未找到 electron 包，请先运行: pnpm install --ignore-scripts')
  process.exit(1)
}

console.log('[setup] 使用镜像:', MIRROR)
console.log('[setup] 正在下载 Electron（约 150MB，请耐心等待）…')

const env = {
  ...process.env,
  ELECTRON_MIRROR: MIRROR,
  ELECTRON_GET_USE_PROXY: '1',
}

const r = spawnSync('node', [installScript], { stdio: 'inherit', env })
if (r.status !== 0) {
  console.error('[setup] 下载失败，可尝试 export ELECTRON_MIRROR=' + MIRROR)
  process.exit(r.status || 1)
}

if (electronBinaryExists()) {
  console.log('[setup] 完成:', require('electron'))
} else {
  console.error('[setup] install.js 已运行但未找到二进制')
  process.exit(1)
}
