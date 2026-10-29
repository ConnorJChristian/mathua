import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

// `make local` built the UI with `npx next build`, which skips npm's prebuild:
// the graph's precomputed layout was never regenerated for your own modules, so
// the Graph page fell back to laying out all ~2,000 concepts with dagre in the
// browser (83 s measured on the owner's install), and the PDF worker and the
// math fonts were not copied either. It must run the build script.
describe('make local', () => {
  it('runs npm run build, so the prebuild steps run', () => {
    const makefile = readFileSync(join(__dirname, '..', '..', '..', 'Makefile'), 'utf8')
    const local = makefile.split(/\n(?=\S)/).find(block => block.startsWith('local:'))
    expect(local).toBeTruthy()
    expect(local).toContain('npm run build')
    expect(local).not.toMatch(/npx next build/)
  })
})
