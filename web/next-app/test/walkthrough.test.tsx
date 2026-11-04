import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useEffect } from 'react'
import { render, screen, fireEvent, within } from '@testing-library/react'
import Walkthrough from '../components/Walkthrough'
import PlacementWalkthroughs from '../components/PlacementWalkthroughs'
import LearnStepper from '../components/LearnStepper'
import ReviewHost from '../app/review/ReviewHost'
import QuizHost from '../app/goals/QuizHost'
import { splitWalkthrough, walkthroughFromExplanation, type Walkthrough as WalkthroughData } from '../lib/walkthrough'
import {
  getLessonKPs, getLessonPractice, getLessonReadiness, submitStudyAnswer, getActivity, getProgress, getWeaknesses, getDueReviews, getScores, getNext,
  startReviewSession, submitReviewAnswer, startQuizSession, submitQuizAnswer,
} from '../lib/api'
import { getUserInfo } from '../lib/auth'
import { findAnswerField, typeAnswer } from './helpers/answerInput'

// The walkthrough on every surface (fork change, FSRS_PORT_PLAN 0c): after a
// miss, or "I don't know", the first step is visible and the rest wait behind
// "Show the rest of the steps"; after a right answer every step waits behind
// "Show the steps". Learn, review, the mastery check and the placement
// results each render it from the server's contract.

vi.mock('../lib/api', async (importOriginal) => {
  const mod = await importOriginal<typeof import('../lib/api')>()
  return {
    ...mod,
    getLessonKPs: vi.fn(), getLessonPractice: vi.fn(), getLessonReadiness: vi.fn(), submitStudyAnswer: vi.fn(),
    getActivity: vi.fn(), getProgress: vi.fn(), getWeaknesses: vi.fn(), getDueReviews: vi.fn(), getScores: vi.fn(), getNext: vi.fn(),
    startReviewSession: vi.fn(), submitReviewAnswer: vi.fn(),
    startQuizSession: vi.fn(), submitQuizAnswer: vi.fn(), skipQuizQuestion: vi.fn(),
  }
})
vi.mock('../lib/auth', async (importOriginal) => {
  const mod = await importOriginal<typeof import('../lib/auth')>()
  return { ...mod, getUserInfo: vi.fn() }
})
vi.mock('next/link', () => ({
  default: ({ href, children, ...rest }: { href: string; children: React.ReactNode }) => (
    <a href={href} {...rest}>{children}</a>
  ),
}))
vi.mock('sonner', () => ({ toast: { error: vi.fn(), success: vi.fn() } }))
vi.mock('../components/PdfReader', () => ({
  default: ({ onUnavailable }: { onUnavailable?: () => void }) => {
    useEffect(() => { onUnavailable?.() }, [onUnavailable])
    return <p>The PDF could not be opened here.</p>
  },
}))
Element.prototype.scrollIntoView = vi.fn() as unknown as typeof Element.prototype.scrollIntoView

const STEP1 = 'Differentiate: C′(x) = 4x + 2.'
const STEP2 = 'At x = 2: 4·2 + 2 = 10.'
const W: WalkthroughData = {
  answer: '10',
  steps: [STEP1, STEP2],
  explain: 'The marginal cost is the derivative of cost at x = 2.',
  reading: { label: 'Hoy · 5.3 Marginal analysis', url: '/api/modules/ma-program/files/hoy-3e.pdf', page: 180 },
}
const WRONG: WalkthroughData = { ...W, diverged_at: { step: 1, reason: 'Your answer is the value at this step: the path was left here.' } }

function visibleSteps() {
  return screen.queryAllByText((_, el) => !!el && el.matches('[data-walkthrough-step]')).length
}

describe('Walkthrough', () => {
  it('shows the first step after a miss and the rest on request, with the divergence marked', () => {
    render(<Walkthrough walkthrough={WRONG} correct={false} />)
    expect(screen.getByText('Answer:').parentElement?.textContent).toContain('10')
    expect(screen.getByText(STEP1)).toBeTruthy()
    expect(screen.queryByText(STEP2)).toBeNull()
    expect(visibleSteps()).toBe(1)
    expect(screen.getByText(/Your answer stops here/)).toBeTruthy()
    expect(screen.getByText('How')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Show the rest of the steps (1)' }))
    expect(screen.getByText(STEP2)).toBeTruthy()
    expect(visibleSteps()).toBe(2)
    expect(screen.getByRole('link', { name: /Read: Hoy/ }).getAttribute('href')).toMatch(/hoy-3e\.pdf#page=180$/)
  })

  it('keeps every step behind the toggle after a right answer', () => {
    render(<Walkthrough walkthrough={W} correct />)
    expect(visibleSteps()).toBe(0)
    expect(screen.queryByText(STEP1)).toBeNull()
    expect(screen.getByText('Why')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Show the steps (2)' }))
    expect(visibleSteps()).toBe(2)
  })

  it('treats "I don\'t know" as a miss: first step visible, no divergence', () => {
    render(<Walkthrough walkthrough={W} correct={false} dontKnow />)
    expect(visibleSteps()).toBe(1)
    expect(screen.queryByText(/stops here/)).toBeNull()
  })

  it('shows a one-line reason and the reading when there are no steps', () => {
    render(<Walkthrough walkthrough={{ answer: 'supremum', steps: [], explain: 'The least upper bound is the supremum.', reading: { label: 'Abbott 1.3', url: 'https://example.test/abbott' } }} correct={false} />)
    expect(screen.queryByRole('button', { name: /steps/ })).toBeNull()
    expect(screen.getByText(/least upper bound/)).toBeTruthy()
    expect(screen.getByRole('link', { name: /Read: Abbott 1.3/ }).getAttribute('href')).toBe('https://example.test/abbott')
  })
})

describe('splitWalkthrough', () => {
  it('reads a steps block out of an explanation, for history rows and older servers', () => {
    const enc = 'why\n\n<!--steps-->\none\n<!--step-->\ntwo'
    expect(splitWalkthrough(enc)).toEqual({ explain: 'why', steps: ['one', 'two'] })
    expect(splitWalkthrough('plain')).toEqual({ explain: 'plain', steps: [] })
    expect(walkthroughFromExplanation(enc, '4')).toEqual({ answer: '4', steps: ['one', 'two'], explain: 'why' })
  })
})

const CID = 'calc.deriv.marginal_analysis'

function studyRes(correct: boolean, walkthrough: WalkthroughData) {
  return { correct, feedback: correct ? '' : 'Incorrect', explanation: walkthrough.explain, xp: correct ? 10 : 0, expected_answer: '10', to_master: 2, walkthrough }
}

describe('Learn', () => {
  beforeEach(() => {
    vi.mocked(getNext).mockResolvedValue({ primary: null, alternatives: [], generatedAt: '' } as unknown as Awaited<ReturnType<typeof getNext>>)
    vi.mocked(getLessonKPs).mockResolvedValue({ concept_id: CID, kps: [{ label: 'Marginal cost', subgoals: [], worked_example: 'Worked: C′(x).' }] })
    vi.mocked(getLessonReadiness).mockResolvedValue({ concept_id: CID, ready: true, weak: [], missing: [] })
    vi.mocked(getLessonPractice).mockResolvedValue({ questions: [{ question: 'C(x) = 2x² + 2x + 9. Marginal cost at x = 2?' }, { question: 'Next one?' }], concept_id: CID })
    vi.mocked(submitStudyAnswer).mockReset()
    vi.mocked(getUserInfo).mockReturnValue({ student_id: 's1', diagnostic_completed: true } as ReturnType<typeof getUserInfo>)
    vi.mocked(getActivity).mockResolvedValue([])
    vi.mocked(getProgress).mockResolvedValue({})
    vi.mocked(getWeaknesses).mockResolvedValue({ by_domain: {} })
    vi.mocked(getDueReviews).mockResolvedValue({ count: 0 })
    vi.mocked(getScores).mockResolvedValue({ concepts_mastered: 3 } as Awaited<ReturnType<typeof getScores>>)
  })

  it('shows the first step after a wrong answer', async () => {
    vi.mocked(submitStudyAnswer).mockResolvedValue(studyRes(false, WRONG))
    render(<LearnStepper conceptId={CID} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Next →' }))
    await screen.findByText(/Marginal cost at x = 2/)
    typeAnswer(await findAnswerField(), '8')
    fireEvent.click(screen.getByRole('button', { name: 'Check' }))
    await screen.findByText(/Not quite/)
    expect(screen.getByText(STEP1)).toBeTruthy()
    expect(screen.queryByText(STEP2)).toBeNull()
    expect(screen.getByRole('button', { name: 'Show the rest of the steps (1)' })).toBeTruthy()
    expect(screen.getByText(/stops here/)).toBeTruthy()
  })

  it('shows the first step after "I don\'t know"', async () => {
    vi.mocked(submitStudyAnswer).mockResolvedValue(studyRes(false, W))
    render(<LearnStepper conceptId={CID} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Next →' }))
    await screen.findByText(/Marginal cost at x = 2/)
    fireEvent.click(screen.getByRole('button', { name: "I don't know" }))
    await screen.findByText(/No answer/)
    expect(screen.getByText(STEP1)).toBeTruthy()
    expect(screen.queryByText(STEP2)).toBeNull()
  })

  it('keeps the steps collapsed after a right answer', async () => {
    vi.mocked(submitStudyAnswer).mockResolvedValue(studyRes(true, W))
    render(<LearnStepper conceptId={CID} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Next →' }))
    await screen.findByText(/Marginal cost at x = 2/)
    typeAnswer(await findAnswerField(), '10')
    fireEvent.click(screen.getByRole('button', { name: 'Check' }))
    await screen.findByText(/you answered/)
    expect(screen.getByText((_, el) => !!el && el.matches('[data-walkthrough-state="right"]'))).toBeTruthy()
    expect(screen.queryByText(STEP1)).toBeNull()
    expect(screen.getByRole('button', { name: 'Show the steps (2)' })).toBeTruthy()
    expect(screen.getByText(/derivative of cost/)).toBeTruthy()
  })
})

const RQ = { concept_id: CID, concept_name: 'Marginal analysis', question: 'C(x) = 2x² + 2x + 9. Marginal cost at x = 2?', is_review: true, attempt_id: 'a1' }

function reviewResult(correct: boolean, walkthrough: WalkthroughData) {
  return { correct, feedback: correct ? 'Correct!' : 'Incorrect', new_status: 'PRACTICING', evidence_score: 0.4, evidence_band: 'building', explanation: walkthrough.explain, streak: 1, required_streak: 2, xp: correct ? 5 : 0, expected_answer: '10', walkthrough }
}

describe('Review', () => {
  beforeEach(() => {
    vi.mocked(startReviewSession).mockResolvedValue({ student_id: 's1', session_id: 'r1', question: RQ })
    vi.mocked(submitReviewAnswer).mockReset()
  })

  it('shows the first step after a wrong answer', async () => {
    vi.mocked(submitReviewAnswer).mockResolvedValueOnce({ result: reviewResult(false, WRONG), next_question: null, done: true })
    render(<ReviewHost />)
    fireEvent.click(screen.getByRole('button', { name: /Start review/ }))
    await screen.findByText(/Marginal cost at x = 2/)
    typeAnswer(await findAnswerField(), '8')
    fireEvent.click(screen.getByRole('button', { name: 'Check Answer' }))
    await screen.findByText('✗ Not quite')
    expect(screen.getByText(STEP1)).toBeTruthy()
    expect(screen.queryByText(STEP2)).toBeNull()
    expect(screen.getByRole('button', { name: 'Show the rest of the steps (1)' })).toBeTruthy()
  })

  it('keeps the steps collapsed after a right answer', async () => {
    vi.mocked(submitReviewAnswer).mockResolvedValueOnce({ result: reviewResult(true, W), next_question: null, done: true })
    render(<ReviewHost />)
    fireEvent.click(screen.getByRole('button', { name: /Start review/ }))
    await screen.findByText(/Marginal cost at x = 2/)
    typeAnswer(await findAnswerField(), '10')
    fireEvent.click(screen.getByRole('button', { name: 'Check Answer' }))
    await screen.findByText('✓ Correct!')
    expect(screen.queryByText(STEP1)).toBeNull()
    expect(screen.getByRole('button', { name: 'Show the steps (2)' })).toBeTruthy()
  })
})

describe('Mastery check', () => {
  beforeEach(() => {
    vi.mocked(startQuizSession).mockResolvedValue({
      session_id: 'q1', concept_id: CID, concept_name: 'Marginal analysis', question: 'C(x) = 2x² + 2x + 9. Marginal cost at x = 2?',
      grading_type: 'numeric', done: false, questions_total: 2, time_limit_seconds: 60, closed_book: true,
    } as Awaited<ReturnType<typeof startQuizSession>>)
    vi.mocked(submitQuizAnswer).mockReset()
  })

  it('shows the first step after a wrong answer, and the rest on request', async () => {
    vi.mocked(submitQuizAnswer).mockResolvedValueOnce({ done: false, correct: false, feedback: 'Incorrect', xp: 0, concept_id: 'b', concept_name: 'B', question: 'Next?', walkthrough: WRONG } as Awaited<ReturnType<typeof submitQuizAnswer>>)
    render(<QuizHost />)
    fireEvent.click(screen.getByRole('button', { name: /start quiz/i }))
    await screen.findByText(/Marginal cost at x = 2/)
    typeAnswer(await findAnswerField(), '8')
    fireEvent.click(screen.getByRole('button', { name: /Check/ }))
    await screen.findByText('✗ Not quite')
    expect(screen.getByText(STEP1)).toBeTruthy()
    expect(screen.queryByText(STEP2)).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'Show the rest of the steps (1)' }))
    expect(screen.getByText(STEP2)).toBeTruthy()
  })

  it('keeps the steps collapsed after a right answer', async () => {
    vi.mocked(submitQuizAnswer).mockResolvedValueOnce({ done: false, correct: true, feedback: '', xp: 5, concept_id: 'b', concept_name: 'B', question: 'Next?', walkthrough: W } as Awaited<ReturnType<typeof submitQuizAnswer>>)
    render(<QuizHost />)
    fireEvent.click(screen.getByRole('button', { name: /start quiz/i }))
    await screen.findByText(/Marginal cost at x = 2/)
    typeAnswer(await findAnswerField(), '10')
    fireEvent.click(screen.getByRole('button', { name: /Check/ }))
    await screen.findByText('✓ Correct!')
    expect(screen.queryByText(STEP1)).toBeNull()
    expect(screen.getByRole('button', { name: 'Show the steps (2)' })).toBeTruthy()
  })
})

describe('Placement results', () => {
  it('lists each answered question: a miss with its first step, a right answer collapsed', () => {
    render(<PlacementWalkthroughs questions={[
      { concept_id: CID, concept_name: 'Marginal analysis', question: 'Marginal cost at x = 2?', answer: '8', correct: false, dont_know: false, walkthrough: WRONG },
      { concept_id: 'calc.deriv.partial', concept_name: 'Partial derivatives', question: 'f_y at (1,2)?', answer: '6', correct: true, dont_know: false, walkthrough: { ...W, answer: '6', steps: ['f_y = 3x²y.', 'At (1,2): 6.'] } },
      { concept_id: 'calc.deriv.total_diff', concept_name: 'Total differential', question: 'df?', answer: '', correct: false, dont_know: true, walkthrough: { ...W, answer: '-1' } },
    ]} />)
    expect(screen.getByText(/3 asked, 2 missed/)).toBeTruthy()
    const items = screen.getAllByText((_, el) => !!el && el.matches('[data-placement-question]'))
    expect(items.length).toBe(3)
    expect(within(items[0]).getByText(STEP1)).toBeTruthy()
    expect(within(items[0]).queryByText(STEP2)).toBeNull()
    expect(within(items[1]).queryByText('f_y = 3x²y.')).toBeNull()
    expect(within(items[1]).getByRole('button', { name: 'Show the steps (2)' })).toBeTruthy()
    expect(within(items[2]).getByText('No answer.')).toBeTruthy()
    expect(within(items[2]).getByText(STEP1)).toBeTruthy()
  })
})
