// What Mathua's math editor sends for each LaTeX answer.
//
//   node --experimental-strip-types scripts/mathlive-plain.mjs in.json out.json
//
// in.json is a JSON list of LaTeX strings; out.json gets the same list with
// each one replaced by the plain answer the editor would submit: MathLive's
// getValue('plain-text') in Chromium, then components/math/plainAnswer.ts.
// scripts/import_anki.py uses it so an imported typed answer is stored in
// exactly the form a learner who types the book's answer sends.
import { chromium } from '@playwright/test'
import { createServer } from 'node:http'
import { readFile, writeFile } from 'node:fs/promises'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const appRoot = join(here, '..')
const mathliveDir = join(appRoot, 'node_modules', 'mathlive')
const { toPlainAnswer } = await import(join(appRoot, 'components', 'math', 'plainAnswer.ts'))

const PAGE = `<!doctype html><html><head><meta charset="utf-8"></head><body>
<script type="module">
import './mathlive.min.mjs'
window.__probe = async (list) => {
  const mf = document.createElement('math-field')
  document.body.appendChild(mf)
  await new Promise(r => setTimeout(r, 60))
  const out = []
  for (const latex of list) {
    try {
      mf.value = latex
      out.push(mf.getValue('plain-text') ?? '')
    } catch (e) {
      out.push('')
    }
  }
  return out
}
window.__ready = true
</script></body></html>`

const [inPath, outPath] = process.argv.slice(2)
if (!inPath || !outPath) {
  console.error('usage: mathlive-plain.mjs in.json out.json')
  process.exit(2)
}
const list = JSON.parse(await readFile(inPath, 'utf8'))

const srv = createServer(async (req, res) => {
  if (req.url.startsWith('/mathlive.min.mjs')) {
    res.writeHead(200, { 'Content-Type': 'text/javascript' })
    res.end(await readFile(join(mathliveDir, 'mathlive.min.mjs')))
    return
  }
  res.writeHead(200, { 'Content-Type': 'text/html' })
  res.end(PAGE)
})
const port = 4600 + (process.pid % 200)
await new Promise(r => srv.listen(port, r))
const launch = process.env.MATHLIVE_CHROMIUM ? { executablePath: process.env.MATHLIVE_CHROMIUM } : {}
const browser = await chromium.launch(launch)
const page = await browser.newPage()
await page.goto(`http://localhost:${port}/`)
await page.waitForFunction(() => window.__ready === true, { timeout: 30000 })
const plain = []
for (let i = 0; i < list.length; i += 500) {
  plain.push(...(await page.evaluate(l => window.__probe(l), list.slice(i, i + 500))))
}
await browser.close()
srv.close()
await writeFile(outPath, JSON.stringify(plain.map(p => (p ? toPlainAnswer(p) : ''))))
console.log(`mathlive-plain: ${list.length} answers`)
