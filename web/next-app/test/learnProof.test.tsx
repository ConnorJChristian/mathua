import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import LearnStepper from '../components/LearnStepper'
import { getLessonKPs, getLessonPractice, getLessonReadiness, submitStudyAnswer, getActivity, getProgress, getWeaknesses, getDueReviews, getScores, getNext } from '../lib/api'
import { getUserInfo } from '../lib/auth'
import { track } from '../lib/events'
import { answerCard, answerField, typeAnswer } from './helpers/answerInput'

vi.mock('../lib/api', async (importOriginal) => {
  const mod = await importOriginal<typeof import('../lib/api')>()
  return {
    ...mod,
    getLessonKPs: vi.fn(), getLessonPractice: vi.fn(), getLessonReadiness: vi.fn(), submitStudyAnswer: vi.fn(),
    getActivity: vi.fn(), getProgress: vi.fn(), getWeaknesses: vi.fn(), getDueReviews: vi.fn(), getScores: vi.fn(), getNext: vi.fn(),
  }
})
vi.mock('../lib/auth', async (importOriginal) => {
  const mod = await importOriginal<typeof import('../lib/auth')>()
  return { ...mod, getUserInfo: vi.fn() }
})
vi.mock('../lib/events', async (importOriginal) => {
  const mod = await importOriginal<typeof import('../lib/events')>()
  return { ...mod, track: vi.fn() }
})
Element.prototype.scrollIntoView = vi.fn() as unknown as typeof Element.prototype.scrollIntoView

const CID = 'arith.add.single'
const PROOF_Q = '<!--card-->\n**Prove.** Every convergent sequence is bounded.\n\n<!--proof-answer-->'
// The served questions carry no answer and no explanation: those come back
// with the submission. A bank-served one says so with a flag.
const served = (qs: typeof QS) => qs.map(({ answer, explanation: _e, ...q }) => ({ ...q, ...(/^(mc|out|code|recall|math|term|proof):/.test(answer) ? { bank: true } : {}) }))
const QS = [
  { question: PROOF_Q, answer: 'proof:abbott.convergent_bounded', explanation: 'A reference proof.' },
  { question: '2 + 3 = ?', answer: '5', explanation: 'Add them.' },
  { question: '4 + 1 = ?', answer: '5', explanation: 'Add them.' },
  { question: '1 + 6 = ?', answer: '7', explanation: 'Add them.' },
]

beforeEach(() => {
  vi.mocked(getNext).mockResolvedValue({ primary: null, alternatives: [], generatedAt: '' } as unknown as Awaited<ReturnType<typeof getNext>>)
  vi.mocked(getLessonKPs).mockResolvedValue({ concept_id: CID, kps: [{ label: 'Add single digits', subgoals: [], worked_example: '2 + 3 = 5' }] })
  vi.mocked(getLessonPractice).mockResolvedValue({ questions: served(QS), concept_id: CID })
  vi.mocked(getLessonReadiness).mockResolvedValue({ concept_id: CID, ready: true, weak: [], missing: [] })
  vi.mocked(submitStudyAnswer).mockImplementation(async (_cid, answer, _elapsed, question) => {
    if (question === PROOF_Q) {
      return {
        correct: true, feedback: 'Score 7/7.', explanation: 'A reference proof.', xp: 0,
        proof: { score: 7, max: 7, runs: [7, 7, 6, 7, 7], fully_correct: true, milestones: [{ id: 'm1', desc: 'Bounds the tail', points: 7, met: 'yes' }], feedback: 'Complete.', counts_for_mastery: false },
      }
    }
    const key = QS.find(q => q.question === question)?.answer ?? ''
    return { correct: answer.trim() === key, feedback: 'Correct!', explanation: '', expected_answer: key, xp: 1 }
  })
  vi.mocked(getUserInfo).mockReturnValue({ student_id: 's1', diagnostic_completed: true } as ReturnType<typeof getUserInfo>)
  vi.mocked(getActivity).mockResolvedValue([])
  vi.mocked(getProgress).mockResolvedValue({})
  vi.mocked(getWeaknesses).mockResolvedValue({ by_domain: {} })
  vi.mocked(getDueReviews).mockResolvedValue({ count: 0 })
  vi.mocked(getScores).mockResolvedValue({ concepts_mastered: 3 } as Awaited<ReturnType<typeof getScores>>)
  vi.mocked(track).mockClear()
})

async function answerCurrent(value: string) {
  await waitFor(() => expect(answerField()).toBeTruthy())
  const input = answerField()
  typeAnswer(input, value)
  fireEvent.click(within(answerCard(input)).getByRole('button', { name: 'Check' }))
}

describe('a proof in Learn', () => {
  it('takes a written proof, shows its grade, logs its score, and does not count toward the in-a-row gate', async () => {
    render(<LearnStepper conceptId={CID} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Next →' }))
    const box = await screen.findByRole('textbox', { name: 'Your proof' })
    fireEvent.change(box, { target: { value: 'Let \\(x_n \\to l\\). With \\(\\epsilon = 1\\) ...' } })
    fireEvent.click(screen.getByRole('button', { name: 'Check' }))

    expect(await screen.findByText('7/7')).toBeTruthy()
    expect(screen.getByText('Fully correct')).toBeTruthy()
    expect(vi.mocked(submitStudyAnswer).mock.calls[0][3]).toBe(PROOF_Q)
    const ev = vi.mocked(track).mock.calls.map(c => c[0]).find(e => e.kind === 'answer')
    expect(ev?.data?.proof_score).toBe(7)

    // The practice-only proof moved no counter: two arithmetic answers are
    // still needed, and one is not enough.
    expect(await screen.findByText('2 + 3 = ?')).toBeTruthy()
    await answerCurrent('5')
    expect(await screen.findByText('4 + 1 = ?')).toBeTruthy()
    expect(screen.queryByText(/Session done/)).toBeNull()
    await answerCurrent('5')
    expect(await screen.findByText(/Session done · still learning — 2\/2 correct/)).toBeTruthy()
  }, 15000)

  it('says why a proof was not graded and keeps it', async () => {
    vi.mocked(submitStudyAnswer).mockResolvedValueOnce({ correct: false, ungraded: true, feedback: 'Proof grading is off: ANTHROPIC_API_KEY is not set on the server', xp: 0 })
    render(<LearnStepper conceptId={CID} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Next →' }))
    const box = await screen.findByRole('textbox', { name: 'Your proof' })
    fireEvent.change(box, { target: { value: 'My proof.' } })
    fireEvent.click(screen.getByRole('button', { name: 'Check' }))
    expect(await screen.findByText(/ANTHROPIC_API_KEY is not set/)).toBeTruthy()
    expect((screen.getByRole('textbox', { name: 'Your proof' }) as HTMLTextAreaElement).value).toBe('My proof.')
  }, 15000)

  it('says that a typed answer was not graded and keeps it (fork fix)', async () => {
    vi.mocked(getLessonPractice).mockResolvedValue({ questions: served(QS.slice(1)), concept_id: CID })
    vi.mocked(submitStudyAnswer).mockResolvedValueOnce({ correct: false, ungraded: true, feedback: 'Grading service timed out', xp: 0 })
    render(<LearnStepper conceptId={CID} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Next →' }))
    await answerCurrent('5')
    expect(await screen.findByText(/Grading service timed out/)).toBeTruthy()
    expect((answerField() as HTMLInputElement).value).toBe('5')
    // Nothing was recorded: the same question is still the one to answer.
    expect(screen.getByText('2 + 3 = ?')).toBeTruthy()
    expect(screen.queryByText('4 + 1 = ?')).toBeNull()
  }, 15000)
})
