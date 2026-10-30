import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useEffect } from 'react'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import LearnStepper, { sessionDoneLine } from '../components/LearnStepper'
import { getLessonKPs, getLessonPractice, getLessonReadiness, submitStudyAnswer, getActivity, getProgress, getWeaknesses, getDueReviews, getScores, getNext, type PracticeQuestion } from '../lib/api'
import { getUserInfo } from '../lib/auth'
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
// The PDF is not on this machine: the reader reports it, as pdf.js does on a 404.
vi.mock('../components/PdfReader', () => ({
  default: ({ onUnavailable }: { onUnavailable?: () => void }) => {
    useEffect(() => { onUnavailable?.() }, [onUnavailable])
    return <p>The PDF could not be opened here.</p>
  },
}))
Element.prototype.scrollIntoView = vi.fn() as unknown as typeof Element.prototype.scrollIntoView

const CID = 'arith.add.single'
const BACK = 'The back of the card: (∅ ∪ A) ∩ (B ∪ A) = A'
const CARD_Q = `<!--card-->\nWrite the dual of the set equation.\n\n<!--recall-back-->\n\n${BACK}`
const MC_Q = 'Which is the optimum?\n\nA) \\((x^*, y^*) = (1, 2)\\)\n\nB) \\((x^*, y^*) = (2, 1)\\)'

function serve(questions: PracticeQuestion[]) {
  vi.mocked(getLessonPractice).mockResolvedValue({ questions, concept_id: CID })
}

beforeEach(() => {
  vi.mocked(getNext).mockResolvedValue({ primary: null, alternatives: [], generatedAt: '' } as unknown as Awaited<ReturnType<typeof getNext>>)
  vi.mocked(getLessonKPs).mockResolvedValue({ concept_id: CID, kps: [{ label: 'Add single digits', subgoals: [], worked_example: 'Worked: 2 + 3 = 5' }] })
  vi.mocked(getLessonReadiness).mockResolvedValue({ concept_id: CID, ready: true, weak: [], missing: [] })
  serve([{ question: '2 + 3 = ?' }, { question: '4 + 1 = ?' }])
  vi.mocked(submitStudyAnswer).mockReset()
  vi.mocked(getUserInfo).mockReturnValue({ student_id: 's1', diagnostic_completed: true } as ReturnType<typeof getUserInfo>)
  vi.mocked(getActivity).mockResolvedValue([])
  vi.mocked(getProgress).mockResolvedValue({})
  vi.mocked(getWeaknesses).mockResolvedValue({ by_domain: {} })
  vi.mocked(getDueReviews).mockResolvedValue({ count: 0 })
  vi.mocked(getScores).mockResolvedValue({ concepts_mastered: 3 } as Awaited<ReturnType<typeof getScores>>)
})

describe('a gated topic', () => {
  beforeEach(() => {
    vi.mocked(getLessonReadiness).mockResolvedValue({
      concept_id: CID, ready: false, eligible: false, weak: [], missing: [],
      prerequisites: [
        { conceptId: 'calc.deriv.partial', title: 'Partial derivatives', state: 'due_for_review', met: true },
        { conceptId: 'calc.deriv.optimization', title: 'Optimization problems', state: 'unlocked', met: false },
      ],
    })
  })

  it('lists only the prerequisites that block it', async () => {
    render(<LearnStepper conceptId={CID} />)
    expect(await screen.findByText('Optimization problems')).toBeTruthy()
    expect(screen.queryByText('Partial derivatives')).toBeNull()
    expect(screen.getByText('Prerequisites (1)')).toBeTruthy()
  })

  it('keeps the reading open: the gate limits practice, not reading', async () => {
    render(<LearnStepper conceptId={CID} />)
    expect(await screen.findByRole('button', { name: /Practice anyway/ })).toBeTruthy()
    expect(screen.getByText(/Worked: 2 \+ 3 = 5/)).toBeTruthy()
    expect(getLessonPractice).not.toHaveBeenCalled()
  })
})

describe('a reading whose PDF cannot be opened', () => {
  it("opens Mathua's summary, so the lesson is not empty", async () => {
    vi.mocked(getLessonKPs).mockResolvedValue({
      concept_id: CID,
      kps: [{ label: 'Sets', subgoals: [], worked_example: 'Summary: a set is a collection.', reading: { title: 'Hoy §2.1', url: '/api/modules/m/files/hoy.pdf', page: 30 } }],
    })
    render(<LearnStepper conceptId={CID} />)
    await screen.findByText(/could not be opened/)
    const summary = screen.getByText(/summary and worked example/).closest('details')!
    await waitFor(() => expect(summary.open).toBe(true))
  })
})

describe('"I don\'t know" in a lesson', () => {
  it('submits an admitted unknown and shows the worked solution, not a wrong answer', async () => {
    vi.mocked(submitStudyAnswer).mockResolvedValue({ correct: false, feedback: '', explanation: 'Count on from 2: 3, 4, 5.', xp: 0, expected_answer: '5', to_master: 6 })
    render(<LearnStepper conceptId={CID} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Next →' }))
    await screen.findByText('2 + 3 = ?')
    fireEvent.click(screen.getByRole('button', { name: "I don't know" }))
    await screen.findByText(/No answer/)
    expect(submitStudyAnswer).toHaveBeenCalledWith(CID, '', expect.any(Number), '2 + 3 = ?', false, true)
    expect(screen.queryByText(/Not quite/)).toBeNull()
    expect(screen.getByText(/Count on from 2/)).toBeTruthy()
  })
})

describe('a flashcard', () => {
  it('submits on "I had it", has no Check button, and shows the back once', async () => {
    serve([{ question: CARD_Q, bank: true }])
    vi.mocked(submitStudyAnswer).mockResolvedValue({ correct: true, feedback: '', explanation: `${BACK}\n\nSolved Problem 1.20(a)`, xp: 0, to_master: -1 })
    render(<LearnStepper conceptId={CID} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Next →' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Show answer' }))
    expect(screen.queryByRole('button', { name: 'Check' })).toBeNull()
    expect(screen.queryByRole('button', { name: "I don't know" })).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'I had it' }))
    await screen.findByText(/you had it/)
    expect(submitStudyAnswer).toHaveBeenCalledWith(CID, 'right', expect.any(Number), CARD_Q, false, false)
    expect(screen.getAllByText(new RegExp(BACK.replace(/[()∪∩∅]/g, '.'))).length).toBe(1)
    expect(screen.getByText(/Solved Problem 1.20\(a\)/)).toBeTruthy()
  })
})

describe('a multiple-choice answer', () => {
  it('is echoed by its letter and rendered, not as LaTeX source', async () => {
    serve([{ question: MC_Q }])
    vi.mocked(submitStudyAnswer).mockResolvedValue({ correct: true, feedback: '', explanation: 'Set the gradient to zero.', xp: 1, to_master: 5 })
    render(<LearnStepper conceptId={CID} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Next →' }))
    const group = await screen.findByRole('group', { name: 'Answer choices' })
    fireEvent.click(within(group).getAllByRole('button')[0])
    const card = answerCard(answerField())
    fireEvent.click(within(card).getByRole('button', { name: 'Check' }))
    const echo = (await screen.findByText(/you answered/)).closest('p')!
    expect(echo.textContent).toContain('A)')
    expect(echo.textContent).not.toContain('\\(')
  })
})

describe('the end of a session', () => {
  it('says how far from mastery the topic still is', async () => {
    vi.mocked(submitStudyAnswer).mockImplementation(async (_c, answer) => ({ correct: answer === '5', feedback: '', explanation: '', xp: 1, new_status: 'LEARNING', to_master: 4 }))
    render(<LearnStepper conceptId={CID} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Next →' }))
    for (let i = 0; i < 2; i++) {
      await waitFor(() => expect(answerField()).toBeTruthy())
      const input = answerField()
      typeAnswer(input, '5')
      fireEvent.click(within(answerCard(input)).getByRole('button', { name: 'Check' }))
      if (i === 0) await screen.findByText('4 + 1 = ?')
    }
    expect(await screen.findByText(/Session done · still learning \(4 more correct to master\) — 2\/2 correct/)).toBeTruthy()
    expect(screen.queryByText(/Complete/)).toBeNull()
  }, 15000)

  it('says mastered only when the evidence says so', () => {
    expect(sessionDoneLine({ correct: 2, answered: 2, xp: 2, band: '', toMaster: 0, status: 'MASTERED' })).toBe('✓ Mastered — 2/2 correct · +2 XP')
    expect(sessionDoneLine({ correct: 2, answered: 2, xp: 0, band: 'early', toMaster: 2, status: 'PRACTICING' })).toBe('Session done · still learning (2 more correct to master) — 2/2 correct · early')
    expect(sessionDoneLine({ correct: 1, answered: 1, xp: 0, band: '', toMaster: null, status: '' })).toBe('Session done · still learning — 1/1 correct')
  })
})
