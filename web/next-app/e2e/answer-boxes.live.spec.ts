import { test, expect, type Page } from '@playwright/test'

// Q71 / Q72 against a real server: the Go binary serving this build's static
// export and the real question banks, nothing stubbed but the choice of item.
// Skipped unless MATHUA_LIVE_URL names that server, e.g.
//
//   MATHUA_ROOT=... MATHUA_DATA_DIR=<worktree>/data mathua --port 8411 --no-auth
//   MATHUA_LIVE_URL=http://localhost:8411 npx playwright test e2e/answer-boxes.live.spec.ts --project=chromium
//
// The practice request is passed through to the server and only narrowed to
// the one item under test, so serving, grading and the runner are all real.

const LIVE = process.env.MATHUA_LIVE_URL
test.skip(!LIVE, 'needs a live Mathua server (MATHUA_LIVE_URL)')
test.use({ baseURL: LIVE })

/** Serve the concept's practice with only the item whose question contains `marker`. */
async function onlyItem(page: Page, concept: string, marker: string) {
  await page.route(`**/api/lessons/${concept}/practice**`, async route => {
    for (let seed = 1; seed <= 60; seed++) {
      const url = new URL(route.request().url())
      url.searchParams.set('seed', String(seed))
      url.searchParams.set('override', '1')
      url.searchParams.delete('exclude')
      const res = await route.fetch({ url: url.toString() })
      const body = await res.json()
      const hit = (body.questions ?? []).find((q: { question: string }) => q.question.includes(marker))
      if (hit) return route.fulfill({ json: { ...body, questions: [hit] } })
    }
    return route.abort()
  })
}

// A guest learner with a server guest token, as lib/auth ensureGuestToken mints one.
test.beforeEach(async ({ page }) => {
  const res = await page.request.post('/api/auth/guest', { data: { student_id: `guest_${crypto.randomUUID()}` } })
  const { token, student_id } = await res.json()
  await page.addInitScript(([t, sid]) => {
    localStorage.setItem('mathua_guest_token', t)
    localStorage.setItem('mathua_guest_id', sid)
  }, [token, student_id])
})

async function openPractice(page: Page, concept: string) {
  await page.goto(`/learn?concept=${concept}`)
  // A topic with unmet prerequisites asks first; one without goes straight in.
  await page.getByRole('button', { name: /Practice anyway/ }).first().click({ timeout: 8_000 }).catch(() => {})
  // The reading comes first; Next moves on to the questions.
  for (let i = 0; i < 5 && !(await page.locator('textarea').count()); i++) {
    await page.getByRole('button', { name: /^Next/ }).first().click({ timeout: 8_000 }).catch(() => {})
    await page.waitForTimeout(800)
  }
}

test('a three-line output answer typed with Enter between the lines grades right (Q71)', async ({ page }) => {
  await onlyItem(page, 'py.basics.loops', 'for series in ["GDP", "CPI", "unemployment"]')
  await openPractice(page, 'py.basics.loops')
  const box = page.getByRole('textbox', { name: 'What the code prints' })
  await expect(box).toBeVisible({ timeout: 30_000 })
  await expect(page.locator('[data-submit-hint]').first()).toContainText('Ctrl+Enter to check')

  let submitted: string | null = null
  page.on('request', r => {
    if (r.url().includes('/api/study/answer')) submitted = (r.postDataJSON() as { answer: string }).answer
  })
  await box.click()
  await page.keyboard.type('GDP 3')
  await page.keyboard.press('Enter')
  await page.keyboard.type('CPI 3')
  await page.keyboard.press('Enter')
  await page.keyboard.type('unemployment 12')
  // Enter made new lines and checked nothing.
  await expect(box).toHaveValue('GDP 3\nCPI 3\nunemployment 12')
  expect(submitted).toBeNull()

  const graded = page.waitForResponse(r => r.url().includes('/api/study/answer'))
  await page.keyboard.press('Control+Enter')
  expect((await (await graded).json()).correct).toBe(true)
  await expect(page.locator('[data-code-echo]').first()).toBeVisible({ timeout: 20_000 })
  await expect(page.getByText('Not quite')).toHaveCount(0)
  expect(submitted).toBe('GDP 3\nCPI 3\nunemployment 12')
  // The echo keeps the three lines as typed.
  await expect(page.locator('[data-code-echo]').first()).toHaveText('GDP 3\nCPI 3\nunemployment 12')
})

test('a code task whose solution uses >= grades right when typed by hand, ligatures off (Q72)', async ({ page }) => {
  await onlyItem(page, 'py.basics.conditionals', 'task=py.basics.conditionals#c1')
  await openPractice(page, 'py.basics.conditionals')
  const box = page.getByRole('textbox', { name: 'Your Python code' })
  await expect(box).toBeVisible({ timeout: 30_000 })

  // The font draws `>=` as one glyph through `calt` unless it is turned off.
  const style = await box.evaluate(el => {
    const cs = getComputedStyle(el)
    return { family: cs.fontFamily, ligatures: cs.fontVariantLigatures, features: cs.fontFeatureSettings }
  })
  expect(style.family).toMatch(/JetBrains/i)
  expect(style.ligatures).toBe('none')
  expect(style.features).toMatch(/"liga" 0/)
  expect(style.features).toMatch(/"calt" 0/)
  // Fenced code in the prompt area and the run output use the same rule.
  const preStyle = await page.evaluate(() => {
    const pre = document.createElement('pre')
    document.body.appendChild(pre)
    const v = getComputedStyle(pre).fontVariantLigatures
    pre.remove()
    return v
  })
  expect(preStyle).toBe('none')

  let submitted: string | null = null
  page.on('request', r => {
    if (r.url().includes('/api/study/answer')) submitted = (r.postDataJSON() as { answer: string }).answer
  })
  await box.click()
  await page.keyboard.press('Control+A')
  await page.keyboard.press('Delete')
  // By hand: Enter keeps the indentation and indents after a colon, Shift+Tab dedents.
  await page.keyboard.type('if rate >= 6:')
  await page.keyboard.press('Enter')
  await page.keyboard.type('status = "high"')
  await page.keyboard.press('Enter')
  await page.keyboard.press('Shift+Tab')
  await page.keyboard.type('elif rate >= 4:')
  await page.keyboard.press('Enter')
  await page.keyboard.type('status = "moderate"')
  await page.keyboard.press('Enter')
  await page.keyboard.press('Shift+Tab')
  await page.keyboard.type('else:')
  await page.keyboard.press('Enter')
  await page.keyboard.type('status = "low"')
  const solution = 'if rate >= 6:\n    status = "high"\nelif rate >= 4:\n    status = "moderate"\nelse:\n    status = "low"'
  await expect(box).toHaveValue(solution)
  expect(submitted).toBeNull()
  await box.screenshot({ path: test.info().outputPath('code-box-ge.png') })

  const graded = page.waitForResponse(r => r.url().includes('/api/study/answer'))
  await page.getByRole('button', { name: 'Check', exact: true }).click()
  expect((await (await graded).json()).correct).toBe(true)
  await expect(page.getByText('checked against the hidden tests')).toBeVisible({ timeout: 30_000 })
  await expect(page.getByText('Not quite')).toHaveCount(0)
  expect(submitted).toBe(solution)
  // The "Your code" echo shows exactly what was sent.
  await page.locator('summary', { hasText: 'Your code' }).first().click()
  await expect(page.locator('pre[data-code-echo]').first()).toHaveText(solution)
})
