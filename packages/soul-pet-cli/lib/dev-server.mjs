import fs from 'node:fs'
import http from 'node:http'
import path from 'node:path'
import { readDirPackage } from './cli.mjs'

const PREVIEW_HTML = `<!DOCTYPE html>
<html><head><meta charset="utf-8"/><title>Soul Pet Dev</title>
<style>html,body{margin:0;height:100%;overflow:hidden;background:#1e293b}</style></head>
<body><div id="boot" style="color:#94a3b8;padding:16px;font:13px system-ui">Loading…</div>
<script>
window.__PET_EMBED_MODE__='desktop';
window.__AIPetConfig={mode:'desktop'};
</script>
<script src="/bundle.js"></script></body></html>`

function bundleScript(files) {
  const manifest = files['manifest.json'] || '{}'
  const petJs = files['pet.js'] || ''
  const css = files['style.css'] || ''
  return `
window.SERVER_BASE='';
window.__PET_MANIFEST__=${manifest};
window.__PET_PROJECT_BASE__='/assets/';
(function(){
  var s=document.createElement('style');s.textContent=${JSON.stringify(css)};document.head.appendChild(s);
})();
${petJs}
;(function(){var b=document.getElementById('boot');if(b)b.remove();})();
`
}

const port = Number(process.argv[2] || 5179)
const root = process.env.SOUL_PET_DEV_ROOT || process.cwd()
let files = readDirPackage(root)

const server = http.createServer((req, res) => {
  try {
    if (req.url === '/' || req.url === '/index.html') {
      res.writeHead(200, { 'Content-Type': 'text/html' })
      res.end(PREVIEW_HTML)
      return
    }
    if (req.url === '/bundle.js') {
      files = readDirPackage(root)
      res.writeHead(200, { 'Content-Type': 'application/javascript' })
      res.end(bundleScript(files))
      return
    }
    if (req.url?.startsWith('/assets/')) {
      const rel = decodeURIComponent(req.url.slice('/assets/'.length))
      const full = path.join(root, rel)
      if (!full.startsWith(root)) {
        res.writeHead(403)
        res.end()
        return
      }
      if (!fs.existsSync(full)) {
        res.writeHead(404)
        res.end()
        return
      }
      res.writeHead(200)
      fs.createReadStream(full).pipe(res)
      return
    }
    res.writeHead(404)
    res.end()
  } catch (e) {
    res.writeHead(500)
    res.end(String(e.message || e))
  }
})

server.listen(port, () => {
  console.log(`[soul-pet dev] http://127.0.0.1:${port}  root=${root}`)
})
