import { test, expect } from './fixtures'

// Light is the default and it is decided before first paint. The pre-paint
// script (public/js/theme-init.js) runs ahead of hydration, so the class on
// <html> is already right when the document is first observable, and no later
// render flips it. The OS preference is deliberately not consulted: these run
// with the emulated OS set to dark and still expect light.

test.use({ colorScheme: 'dark' })

const STORAGE_KEY = 'mathua.theme'

async function themeAtFirstObservation(page: import('@playwright/test').Page, path: string) {
  // `commit` resolves as soon as the response starts and the document exists,
  // while the parser is still streaming the head. Samples are taken from the
  // moment <body> exists: by then every synchronous script in <head> has run,
  // and nothing has been painted before it. A theme script that is merely
  // preloaded and run by the framework later (next/script `beforeInteractive`
  // with a `src`, which is how this used to work) leaves the class wrong across
  // these samples, because the body is parsed and painted before it executes.
  await page.goto(path, { waitUntil: 'commit' })
  const seen: boolean[] = []
  for (let i = 0; i < 40; i++) {
    try {
      const dark = await page.evaluate(() =>
        document.body ? document.documentElement.classList.contains('dark') : null,
      )
      if (dark !== null) seen.push(dark)
    } catch {
      // the document can be mid-navigation for the first poll or two
    }
    await page.waitForTimeout(25)
  }
  await page.waitForLoadState('load')
  seen.push(await page.evaluate(() => document.documentElement.classList.contains('dark')))
  return seen
}

test('no stored choice: light from first paint, even when the OS prefers dark', async ({ page }) => {
  const seen = await themeAtFirstObservation(page, '/')
  expect(seen.length).toBeGreaterThan(0)
  expect(seen.every((dark) => dark === false)).toBe(true)
  await expect(page.locator('html')).not.toHaveClass(/dark/)
  // The native UI follows the page, not the OS.
  await expect(page.locator('html')).toHaveCSS('color-scheme', 'light')
  await expect(page.locator('meta[name="theme-color"]')).toHaveAttribute('content', '#ffffff')
});

test('stored dark: dark from first paint, no light flash', async ({ page }) => {
  await page.addInitScript((key) => {
    localStorage.setItem(key, 'dark')
  }, STORAGE_KEY)
  const seen = await themeAtFirstObservation(page, '/')
  expect(seen.length).toBeGreaterThan(0)
  expect(seen.every((dark) => dark === true)).toBe(true)
  await expect(page.locator('html')).toHaveCSS('color-scheme', 'dark')
  await expect(page.locator('meta[name="theme-color"]')).toHaveAttribute('content', '#0c0d0f')
})

test('the toggle persists the choice and it survives a reload', async ({ page }) => {
  await page.goto('/')
  const toggle = page.getByRole('button', { name: 'Toggle theme' }).first()
  await expect(toggle).toBeVisible()
  await toggle.click()
  await expect(page.locator('html')).toHaveClass(/dark/)
  expect(await page.evaluate((key) => localStorage.getItem(key), STORAGE_KEY)).toBe('dark')

  await page.reload()
  await expect(page.locator('html')).toHaveClass(/dark/)

  await page.getByRole('button', { name: 'Toggle theme' }).first().click()
  await expect(page.locator('html')).not.toHaveClass(/dark/)
  expect(await page.evaluate((key) => localStorage.getItem(key), STORAGE_KEY)).toBe('light')
})

test('a choice stored under the legacy key is honoured', async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.setItem('mathua-theme', 'dark')
  })
  await page.goto('/')
  await expect(page.locator('html')).toHaveClass(/dark/)
})
