// Fork change: the PDF reader (components/PdfReader) renders pages with
// pdf.js, whose worker must be served as its own file. With `output: 'export'`
// it is copied into `public/pdfjs` at build time, like the MathLive fonts.
// Runs from `prebuild` and `predev`; cheap and idempotent.

import { copyFile, mkdir } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const src = join(root, 'node_modules', 'pdfjs-dist', 'legacy', 'build', 'pdf.worker.min.mjs')
const dest = join(root, 'public', 'pdfjs')

await mkdir(dest, { recursive: true })
await copyFile(src, join(dest, 'pdf.worker.min.mjs'))
console.log('pdf.js worker: copied to public/pdfjs')
