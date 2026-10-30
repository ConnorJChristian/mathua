// What the typed-LaTeX answer box sends for each LaTeX string (fork check).
//
//   node --experimental-strip-types scripts/latex-plain.mjs in.json out.json
//
// in.json is a JSON list of LaTeX strings; out.json maps each to the plain
// answer components/math/LatexField.tsx submits: MathLive's
// convertLatexToAsciiMath, then components/math/plainAnswer.ts. The grading
// self-check (internal/gradecheck, -latex) grades these against the keys.
import { readFile, writeFile } from 'node:fs/promises'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const appRoot = join(here, '..')
const { toPlainAnswer } = await import(join(appRoot, 'components', 'math', 'plainAnswer.ts'))
const { convertLatexToAsciiMath } = await import('mathlive')

const [inPath, outPath] = process.argv.slice(2)
if (!inPath || !outPath) {
  console.error('usage: latex-plain.mjs in.json out.json')
  process.exit(2)
}
const list = JSON.parse(await readFile(inPath, 'utf8'))
const out = {}
for (const latex of list) {
  try {
    out[latex] = toPlainAnswer(convertLatexToAsciiMath(latex))
  } catch {
    out[latex] = latex
  }
}
await writeFile(outPath, JSON.stringify(out))
console.log(`latex-plain: ${list.length} answers`)
