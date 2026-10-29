import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import CourseTestsSection from '../components/CourseTestsSection'
import CourseTestSummary from '../components/CourseTestSummary'
import QuizHost from '../app/goals/QuizHost'
import type { CourseTests, CourseView } from '../lib/api'
import * as api from '../lib/api'
import { track } from '../lib/events'

vi.mock('../lib/api', async (importOriginal) => {
  const mod = await importOriginal<typeof import('../lib/api')>()
  return { ...mod, startQuizSession: vi.fn(), submitQuizAnswer: vi.fn(), skipQuizQuestion: vi.fn() }
})
vi.mock('../lib/events', async (importOriginal) => {
  const mod = await importOriginal<typeof import('../lib/events')>()
  return { ...mod, track: vi.fn() }
})
vi.mock('next/link', () => ({
  default: ({ href, children, ...rest }: { href: string; children: React.ReactNode }) => <a href={href} {...rest}>{children}</a>,
}))
vi.mock('sonner', () => ({ toast: { error: vi.fn(), success: vi.fn() } }))
vi.mock('../components/KatexContent', () => ({ default: ({ children }: { children: React.ReactNode }) => <span>{children}</span> }))
vi.mock('../components/math/MathInput', () => ({
  MathAnswerInput: ({ value, onChange }: { value: string; onChange: (v: string) => void }) => (
    <input aria-label="answer" value={value} onChange={(e) => onChange(e.target.value)} />
  ),
}))

function view(tests?: Partial<CourseTests>): CourseView {
  return {
    id: 'plan.x', name: 'A course', description: '', followed: true, modules: [], texts: [], videos: [], notes: [],
    total: 10, mastered: 0, started: 0, reviews_due: 0, prerequisites: [], needs_placement: false,
    tests: tests ? {
      placement: false, review: false, gradable_topics: 10, placement_topics: 20, placement_questions: 0,
      review_topics: 0, review_questions: 0, full_review_questions: 0, ...tests,
    } : undefined,
  }
}

describe('the course Tests section', () => {
  it('shows each test only when the course offers it', () => {
    const { rerender, container } = render(<CourseTestsSection view={view({ placement: true, placement_questions: 30 })} />)
    expect(screen.getByRole('link', { name: /placement test/i }).getAttribute('href')).toBe('/onboard?course=plan.x')
    expect(screen.getByText(/Up to 30 questions/)).toBeTruthy()
    expect(screen.queryByText(/review test/i)).toBeNull()

    rerender(<CourseTestsSection view={view({ review: true, review_topics: 20, review_questions: 15, full_review_questions: 20 })} />)
    expect(screen.queryByRole('link', { name: /placement test/i })).toBeNull()
    expect(screen.getByRole('link', { name: /review test/i }).getAttribute('href')).toBe('/goals?quiz=1&course=plan.x&scope=review')
    expect(screen.getByRole('link', { name: /full course review/i }).getAttribute('href')).toBe('/goals?quiz=1&course=plan.x&scope=full')
    expect(screen.getByText(/15 questions on topics this course has taught you/)).toBeTruthy()

    rerender(<CourseTestsSection view={view()} />)
    expect(container.textContent).toBe('')
  })

  it('does not offer a review before anything is studied', () => {
    render(<CourseTestsSection view={view({ review: true })} />)
    expect(screen.queryByRole('link', { name: /review test/i })).toBeNull()
    expect(screen.getByText(/Study a topic first/)).toBeTruthy()
  })
})

describe('the per-section summary', () => {
  it('names what each section placed and where to start', () => {
    render(<CourseTestSummary mode="placement" courseId="plan.x" sections={[
      { title: 'Unit one', total: 4, passed: [{ id: 'a', label: 'Topic A' }], missed: [{ id: 'b', label: 'Topic B' }], inferred: [{ id: 'c', label: 'Topic C' }] },
    ]} />)
    expect(screen.getByText(/2 placed, 1 to start/)).toBeTruthy()
    expect(screen.getByText('Topic B').closest('a')?.getAttribute('href')).toBe('/learn?concept=b&course=plan.x')
    expect(screen.getByText(/Placed below a right answer/)).toBeTruthy()
  })
})

describe('QuizHost as a course review test', () => {
  beforeEach(() => {
    vi.mocked(api.startQuizSession).mockReset()
    vi.mocked(api.submitQuizAnswer).mockReset()
    vi.mocked(track).mockReset()
  })

  it('starts the course review, records start and finish with the course, and summarises by section', async () => {
    vi.mocked(api.startQuizSession).mockResolvedValue({
      session_id: 's1', concept_id: 'a', concept_name: 'Topic A', question: '2 + 3', grading_type: 'numeric',
      done: false, questions_total: 1, closed_book: true, course_id: 'plan.x', test: 'full',
    })
    vi.mocked(api.submitQuizAnswer).mockResolvedValue({
      done: true, correct: false, feedback: 'Not equivalent', remedial: ['a'],
      sections: [{ title: 'Unit one', total: 3, passed: [], missed: [{ id: 'a', label: 'Topic A' }] }],
    })
    render(<QuizHost courseId="plan.x" scope="full" />)
    expect(screen.getByText('Full course review')).toBeTruthy()
    // The mastery check's briefing promised "up to 5 questions" from "your recent
    // lessons" on a course test that asks one per taught topic.
    expect(screen.queryByText(/Up to 5 questions/)).toBeNull()
    expect(screen.queryByText(/recent lessons/)).toBeNull()
    expect(screen.getByText(/One question per topic/)).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: /start quiz/i }))
    await screen.findByText(/quiz question 1 of 1/i)
    expect(api.startQuizSession).toHaveBeenCalledWith({ courseId: 'plan.x', scope: 'full' })
    expect(track).toHaveBeenCalledWith(expect.objectContaining({ kind: 'test_start', data: expect.objectContaining({ course: 'plan.x', test: 'full_review' }) }))

    fireEvent.change(screen.getByLabelText('answer'), { target: { value: '4' } })
    fireEvent.click(screen.getByRole('button', { name: /check answer/i }))
    await waitFor(() => expect(track).toHaveBeenCalledWith(expect.objectContaining({ kind: 'test_finish', data: expect.objectContaining({ course: 'plan.x', test: 'full_review', correct: 0, total: 1 }) })))
    // The ordinary answer event still fires.
    expect(track).toHaveBeenCalledWith(expect.objectContaining({ kind: 'answer', concept: 'a' }))
    expect(await screen.findByText(/Review test complete/, {}, { timeout: 3000 })).toBeTruthy()
    expect(screen.getByText(/Needs review \(1\)/)).toBeTruthy()
  })

  it('shows the XP an answer earned without an internal label', async () => {
    vi.mocked(api.startQuizSession).mockResolvedValue({
      session_id: 's2', concept_id: 'a', concept_name: 'Topic A', question: '2 + 3', grading_type: 'numeric',
      done: false, questions_total: 2, closed_book: true, course_id: 'plan.x', test: 'review',
    })
    vi.mocked(api.submitQuizAnswer).mockResolvedValue({
      done: false, correct: true, xp: 1, concept_id: 'b', concept_name: 'Topic B', question: '1 + 1', grading_type: 'numeric',
    })
    render(<QuizHost courseId="plan.x" scope="review" />)
    fireEvent.click(screen.getByRole('button', { name: /start quiz/i }))
    await screen.findByText(/quiz question 1 of 2/i)
    fireEvent.change(screen.getByLabelText('answer'), { target: { value: '5' } })
    fireEvent.click(screen.getByRole('button', { name: /check answer/i }))
    const xp = await screen.findByText(/\+1 XP/)
    // It read "+1 XP (TaskQuiz 20)": an engine constant's name, and a number
    // that was not the award.
    expect(xp.textContent).toBe('+1 XP')
  })

  it('says when the review test counted as the weekly mastery check', async () => {
    vi.mocked(api.startQuizSession).mockResolvedValue({
      session_id: 's3', concept_id: 'a', concept_name: 'Topic A', question: '2 + 3', grading_type: 'numeric',
      done: false, questions_total: 1, closed_book: true, course_id: 'plan.x', test: 'review',
    })
    vi.mocked(api.submitQuizAnswer).mockResolvedValue({
      done: true, correct: true, xp: 1, counted_as_mastery_check: true,
      sections: [{ title: 'Unit one', total: 1, passed: [{ id: 'a', label: 'Topic A' }], missed: [] }],
    })
    render(<QuizHost courseId="plan.x" scope="review" />)
    fireEvent.click(screen.getByRole('button', { name: /start quiz/i }))
    await screen.findByText(/quiz question 1 of 1/i)
    fireEvent.change(screen.getByLabelText('answer'), { target: { value: '5' } })
    fireEvent.click(screen.getByRole('button', { name: /check answer/i }))
    expect(await screen.findByText(/Review test complete/, {}, { timeout: 3000 })).toBeTruthy()
    expect(screen.getByText("This counted as this week's mastery check.")).toBeTruthy()
  })

  it('prints a right answer\'s verdict once', async () => {
    vi.mocked(api.startQuizSession).mockResolvedValue({
      session_id: 's4', concept_id: 'a', concept_name: 'Topic A', question: '2 + 3', grading_type: 'numeric',
      done: false, questions_total: 2, closed_book: true,
    })
    // The server holds back non-token feedback on an assessment, so a right
    // answer arrives with none; the screen used to fill in its own "Correct!"
    // under the "✓ Correct!" verdict.
    vi.mocked(api.submitQuizAnswer).mockResolvedValue({
      done: false, correct: true, xp: 1, concept_id: 'b', concept_name: 'Topic B', question: '1 + 1', grading_type: 'numeric',
    })
    render(<QuizHost />)
    fireEvent.click(screen.getByRole('button', { name: /start quiz/i }))
    await screen.findByText(/quiz question 1 of 2/i)
    fireEvent.change(screen.getByLabelText('answer'), { target: { value: '5' } })
    fireEvent.click(screen.getByRole('button', { name: /check answer/i }))
    await screen.findByRole('button', { name: /next/i })
    expect(screen.getAllByText(/Correct!/)).toHaveLength(1)
  })

  it('describes how XP is priced now, not speed and streaks', () => {
    render(<QuizHost />)
    expect(screen.queryByText(/streaks earn more/i)).toBeNull()
    expect(screen.getByText(/Harder questions/)).toBeTruthy()
  })

  it('keeps the mastery check unchanged without a course', async () => {
    vi.mocked(api.startQuizSession).mockResolvedValue({ session_id: 's', done: true })
    render(<QuizHost />)
    expect(screen.getByText('Mastery check')).toBeTruthy()
    expect(screen.getByText(/Up to 5 questions/)).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: /start quiz/i }))
    await waitFor(() => expect(api.startQuizSession).toHaveBeenCalledWith(undefined))
    expect(track).not.toHaveBeenCalledWith(expect.objectContaining({ kind: 'test_start' }))
  })
})
