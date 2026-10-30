import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import * as api from '../lib/api'
import CoursePage from '../app/course/page'
import LibraryPage from '../app/library/page'

vi.mock('../lib/api', async (importOriginal) => {
  const mod = await importOriginal<typeof import('../lib/api')>()
  return {
    ...mod,
    getCourseView: vi.fn(),
    getPins: vi.fn(async () => []),
    getLibrary: vi.fn(),
    getProgress: vi.fn(async () => ({})),
    getSettings: vi.fn(async () => ({})),
    getMe: vi.fn(async () => { throw new Error('offline') }),
  }
})

vi.mock('../lib/auth', async (importOriginal) => {
  const mod = await importOriginal<typeof import('../lib/auth')>()
  return { ...mod, getUserInfo: vi.fn(() => null) }
})

let search = new URLSearchParams()
vi.mock('next/navigation', () => ({
  usePathname: () => '/',
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), prefetch: vi.fn() }),
  useSearchParams: () => search,
}))

const course = {
  id: 'econ5153', name: 'ECON 5153', description: '', followed: true, modules: [], texts: [], videos: [], notes: [],
  total: 0, mastered: 0, started: 0, reviews_due: 0, prerequisites: [], needs_placement: false,
} as api.CourseView

// A page whose data request fails says that it failed, and Retry asks again.
// "Not found" and "No books yet" are for a course or a library that really
// is missing or empty, not for a server that did not answer.
describe('pages when their data request fails', () => {
  beforeEach(() => { vi.mocked(api.getCourseView).mockReset(); vi.mocked(api.getLibrary).mockReset() })

  it('course: a failed load is not "not found", and Retry recovers', async () => {
    search = new URLSearchParams('id=econ5153')
    vi.mocked(api.getCourseView).mockRejectedValueOnce(new TypeError('Failed to fetch')).mockResolvedValueOnce(course)
    render(<CoursePage />)
    expect((await screen.findByRole('alert')).textContent).toMatch(/could not be loaded/)
    expect(screen.queryByText(/was not found/)).toBeNull()
    expect(vi.mocked(api.getCourseView).mock.calls[0][1]).toEqual({ throwOnError: true })
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect((await screen.findAllByText(/ECON 5153/)).length).toBeGreaterThan(0)
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('course: a course that does not exist still reads as not found', async () => {
    search = new URLSearchParams('id=nope')
    vi.mocked(api.getCourseView).mockResolvedValueOnce(null)
    render(<CoursePage />)
    expect(await screen.findByText(/That course was not found/)).toBeTruthy()
  })

  it('library: a failed load is not "No books yet", and Retry recovers', async () => {
    search = new URLSearchParams()
    vi.mocked(api.getLibrary).mockRejectedValueOnce(new Error('Library fetch failed: 500')).mockResolvedValueOnce([
      { id: 'b1', kind: 'web', track: 'econ', title: 'A Book', available: true, sections: [], topic_count: 1 },
    ])
    render(<LibraryPage />)
    expect((await screen.findByRole('alert')).textContent).toMatch(/could not be loaded/)
    expect(screen.queryByText(/No books yet/)).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(await screen.findByText('A Book')).toBeTruthy()
  })
})
