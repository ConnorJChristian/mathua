import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import OnboardPage from '../app/onboard/page'
import * as api from '../lib/api'
import { track } from '../lib/events'

const push = vi.fn()
vi.mock('next/navigation', () => ({
  useRouter: () => ({ push, replace: vi.fn() }),
  useSearchParams: () => new URLSearchParams('course=plan.x'),
}))
vi.mock('next/link', () => ({
  default: ({ children, href }: { children: React.ReactNode; href: string }) => <a href={href}>{children}</a>,
}))
vi.mock('../lib/api', async (importOriginal) => {
  const mod = await importOriginal<typeof import('../lib/api')>()
  return { ...mod, startGoalDiagnostic: vi.fn(), getGoalPlan: vi.fn(), getCourseCatalog: vi.fn(), submitGoalAnswer: vi.fn() }
})
vi.mock('../lib/events', async (importOriginal) => {
  const mod = await importOriginal<typeof import('../lib/events')>()
  return { ...mod, track: vi.fn() }
})
vi.mock('../hooks/useTheme', () => ({ useTheme: () => ({ mounted: true }) }))
vi.mock('../components/Header', () => ({ default: () => null }))
vi.mock('../components/Footer', () => ({ default: () => null }))
vi.mock('../components/BottomTabs', () => ({ default: () => null }))
vi.mock('../components/KatexContent', () => ({ default: ({ children }: { children: React.ReactNode }) => <span>{children}</span> }))

beforeEach(() => {
  sessionStorage.clear()
  push.mockReset()
  vi.mocked(track).mockReset()
  vi.mocked(api.startGoalDiagnostic).mockReset()
  vi.mocked(api.getCourseCatalog).mockResolvedValue({
    courses: [{ id: 'plan.x', name: 'A course: study plan', plan: ['t1', 't2'] } as never],
    active: '', activeIds: [],
  })
})

// /onboard?course=<id> is that course's placement test: the server builds it
// from the course (its own topics, bank items included), not from a topic list.
describe('a course placement test', () => {
  it('starts from the course id and records the start with the course', async () => {
    vi.mocked(api.startGoalDiagnostic).mockResolvedValue({
      session_id: 's1', concept_id: 't1', concept_name: 'Topic one', grading_type: 'bank', course_id: 'plan.x',
      question: 'Which one?\n\nA) yes\n\nB) no',
      progress: { answered: 0, estimated_total: 25, min_total: 25, max_total: 30, cover_done: 0, cover_size: 8, done: false },
    })
    render(<OnboardPage />)
    fireEvent.click(await screen.findByRole('button', { name: /Start placement/ }))
    await waitFor(() => expect(api.startGoalDiagnostic).toHaveBeenCalledWith([], { courseId: 'plan.x' }))
    expect(track).toHaveBeenCalledWith(expect.objectContaining({ kind: 'test_start', data: expect.objectContaining({ course: 'plan.x', test: 'placement' }) }))
    // A bank multiple-choice item is answered with its options, not a math box.
    expect(await screen.findByRole('button', { name: /yes/ })).toBeTruthy()
  })

  it('says why when there is nothing to place', async () => {
    vi.mocked(api.startGoalDiagnostic).mockResolvedValue({ session_id: '', done: true, message: 'No placement here.' })
    const { toast } = await import('sonner')
    const err = vi.spyOn(toast, 'error')
    render(<OnboardPage />)
    fireEvent.click(await screen.findByRole('button', { name: /Start placement/ }))
    await waitFor(() => expect(err).toHaveBeenCalledWith('No placement here.'))
    expect(push).toHaveBeenCalledWith('/course?id=plan.x')
  })
})

// The server's answered count goes up as soon as an answer is graded. The
// header read "QUESTION 2" above question 1's feedback.
describe('the placement question counter', () => {
  it('names the question on screen while its feedback shows', async () => {
    vi.mocked(api.startGoalDiagnostic).mockResolvedValue({
      session_id: 's1', concept_id: 't1', concept_name: 'Topic one', grading_type: 'bank', course_id: 'plan.x',
      question: 'Which one?\n\nA) yes\n\nB) no',
      progress: { answered: 0, estimated_total: 25, min_total: 25, max_total: 30, cover_done: 0, cover_size: 8, done: false },
    })
    vi.mocked(api.submitGoalAnswer).mockResolvedValue({
      correct: true, done: false, concept_id: 't2', concept_name: 'Topic two', question: 'Next one?\n\nA) a\n\nB) b', grading_type: 'bank',
      progress: { answered: 1, estimated_total: 25, min_total: 25, max_total: 30, cover_done: 1, cover_size: 8, done: false },
    } as never)
    render(<OnboardPage />)
    fireEvent.click(await screen.findByRole('button', { name: /Start placement/ }))
    fireEvent.click(await screen.findByRole('button', { name: /yes/ }))
    expect(screen.getAllByText('Question 1').length).toBeGreaterThan(0)
    fireEvent.click(screen.getByRole('button', { name: /Check Answer/ }))
    await screen.findByRole('button', { name: /Next/ })
    expect(screen.queryByText('Question 2')).toBeNull()
    expect(screen.getAllByText('Question 1').length).toBeGreaterThan(0)
    fireEvent.click(screen.getByRole('button', { name: /Next/ }))
    expect((await screen.findAllByText('Question 2')).length).toBeGreaterThan(0)
  })
})
