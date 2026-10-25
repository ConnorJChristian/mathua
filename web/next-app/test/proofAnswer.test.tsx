import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { useState } from 'react'
import PythonAnswer from '../components/python/PythonAnswer'
import ProofReport from '../components/ProofReport'
import ReviewHost from '../app/review/ReviewHost'
import { startReviewSession, submitReviewAnswer, type ProofReport as Report } from '../lib/api'
import { CARD_MARKER, PROOF_MARKER, displayAnswer, displayQuestion, isBankServed, isProofQuestion, pythonHint, pythonKind } from '../lib/pythonQuestion'

vi.mock('../lib/api', async (importOriginal) => {
  const mod = await importOriginal<typeof import('../lib/api')>()
  return { ...mod, startReviewSession: vi.fn(), submitReviewAnswer: vi.fn() }
})

// The shape internal/proof serves inside a generated topic's practice: the
// card marker, the statement, the proof marker. Kept literal so a change to
// the server's format fails here rather than as a one-line number box.
const PROOF_Q = `${CARD_MARKER}\n**Prove.** Every convergent sequence is bounded.\n\n${PROOF_MARKER}`

const REPORT: Report = {
  score: 4,
  max: 7,
  runs: [4, 3, 5, 4, 4],
  fully_correct: false,
  milestones: [
    { id: 'm1', desc: 'Applies the definition with a specific epsilon', points: 2, met: 'yes', note: 'uses 1' },
    { id: 'm2', desc: 'Bounds the tail', points: 2, met: 'yes' },
    { id: 'm3', desc: 'Handles the terms before N', points: 2, met: 'no', note: 'ignored' },
    { id: 'm4', desc: 'One bound for all terms', points: 1, met: 'partial' },
  ],
  first_error: 'The terms before N are never bounded.',
  feedback: 'Take a maximum over the first N - 1 terms.',
  counts_for_mastery: false,
}

describe('proof question shape', () => {
  it('is recognised from the question alone, whatever the topic', () => {
    expect(pythonKind(PROOF_Q)).toBe('proof')
    expect(isProofQuestion(PROOF_Q)).toBe(true)
    expect(isProofQuestion('2 + 3 = ?')).toBe(false)
    // A ra.* topic's grading type is numeric or multiple choice; the proof
    // still takes the question-bank inputs.
    expect(isBankServed('numeric', PROOF_Q, 'proof:abbott.x')).toBe(true)
    expect(isBankServed('numeric', '2 + 3 = ?', 'proof:abbott.x')).toBe(true)
  })

  it('shows the statement without markers and never the answer key', () => {
    expect(displayQuestion(PROOF_Q)).toBe('**Prove.** Every convergent sequence is bounded.')
    expect(displayAnswer('proof:abbott.convergent_bounded')).toBe('')
    expect(pythonHint('proof')).toMatch(/up to a minute/)
  })
})

function Box({ onSubmit }: { onSubmit: () => void }) {
  const [v, setV] = useState('')
  return (
    <>
      <PythonAnswer id="p" question={PROOF_Q} value={v} onChange={setV} onSubmit={onSubmit} />
      <output data-testid="value">{v}</output>
    </>
  )
}

describe('proof box', () => {
  it('is a multi-line box where Enter is a new line and Ctrl+Enter checks', () => {
    const submit = vi.fn()
    render(<Box onSubmit={submit} />)
    const box = screen.getByRole('textbox', { name: 'Your proof' }) as HTMLTextAreaElement
    expect(box.tagName).toBe('TEXTAREA')
    fireEvent.change(box, { target: { value: 'Let \\(\\epsilon = 1\\).\nThen N exists.' } })
    expect(screen.getByTestId('value').textContent).toBe('Let \\(\\epsilon = 1\\).\nThen N exists.')
    fireEvent.keyDown(box, { key: 'Enter' })
    expect(submit).not.toHaveBeenCalled()
    fireEvent.keyDown(box, { key: 'Enter', ctrlKey: true })
    expect(submit).toHaveBeenCalledTimes(1)
  })

  it('previews the LaTeX it will send', () => {
    render(<Box onSubmit={() => {}} />)
    fireEvent.change(screen.getByRole('textbox', { name: 'Your proof' }), { target: { value: 'So \\(|x_n| < M\\).' } })
    fireEvent.click(screen.getByRole('button', { name: 'Preview' }))
    const preview = document.querySelector('[data-proof-preview]')
    expect(preview).toBeTruthy()
    expect(preview!.querySelector('.katex')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Write' }))
    expect((screen.getByRole('textbox', { name: 'Your proof' }) as HTMLTextAreaElement).value).toBe('So \\(|x_n| < M\\).')
  })
})

describe('proof report', () => {
  it('shows the score, the checklist, the first error and the feedback', () => {
    render(<ProofReport report={REPORT} />)
    expect(screen.getByText('4/7')).toBeTruthy()
    expect(screen.getByText('Not yet a complete proof')).toBeTruthy()
    expect(screen.getByText(/median of 5 gradings/)).toBeTruthy()
    const items = screen.getByRole('list', { name: 'Marking scheme' }).querySelectorAll('li')
    expect(items).toHaveLength(4)
    expect(items[2].textContent).toMatch(/Missing.*2 pt.*Handles the terms before N/)
    expect(items[3].textContent).toMatch(/Partly/)
    expect(screen.getByText('The terms before N are never bounded.')).toBeTruthy()
    expect(screen.getByText(/Take a maximum/)).toBeTruthy()
    expect(screen.getByText(/do not count toward mastery/)).toBeTruthy()
  })

  it('says nothing about mastery once proofs count', () => {
    render(<ProofReport report={{ ...REPORT, score: 7, fully_correct: true, counts_for_mastery: true }} />)
    expect(screen.getByText('Fully correct')).toBeTruthy()
    expect(screen.queryByText(/do not count toward mastery/)).toBeNull()
  })
})

describe('proof in a review', () => {
  beforeEach(() => {
    vi.mocked(startReviewSession).mockResolvedValue({
      student_id: 's1',
      session_id: 'r1',
      question: { concept_id: 'ra.seq.convergence', concept_name: 'Convergence', question: PROOF_Q, is_review: true, attempt_id: 'a1', grading_type: 'numeric' },
    })
    vi.mocked(submitReviewAnswer).mockReset()
  })

  it('takes a written proof, asks for the long timeout and shows the grade', async () => {
    vi.mocked(submitReviewAnswer).mockResolvedValueOnce({
      result: {
        correct: false, feedback: 'Score 4/7.', new_status: 'PRACTICING', evidence_score: 0, evidence_band: '',
        explanation: 'A reference proof.', streak: 0, required_streak: 3, xp: 0, expected_answer: 'proof:x', proof: REPORT,
      },
      next_question: null,
      done: true,
    })
    render(<ReviewHost />)
    fireEvent.click(screen.getByRole('button', { name: /Start review/ }))
    const box = await screen.findByRole('textbox', { name: 'Your proof' })
    fireEvent.change(box, { target: { value: 'Let x_n converge to l. Then ...' } })
    fireEvent.click(screen.getByRole('button', { name: 'Check Answer' }))
    expect(await screen.findByText('4/7')).toBeTruthy()
    expect(vi.mocked(submitReviewAnswer).mock.calls[0][4]).toEqual({ proof: true })
  })

  it('shows why a proof was not graded and keeps it for a retry', async () => {
    vi.mocked(submitReviewAnswer).mockResolvedValueOnce({
      result: {
        correct: false, ungraded: true, feedback: "this month's proof-grading budget is used up", new_status: '', evidence_score: 0,
        evidence_band: '', explanation: '', streak: 0, required_streak: 3, xp: 0, expected_answer: '',
      },
      next_question: null,
      done: false,
    })
    render(<ReviewHost />)
    fireEvent.click(screen.getByRole('button', { name: /Start review/ }))
    const box = await screen.findByRole('textbox', { name: 'Your proof' })
    fireEvent.change(box, { target: { value: 'My proof.' } })
    fireEvent.click(screen.getByRole('button', { name: 'Check Answer' }))
    expect(await screen.findByText(/budget is used up/)).toBeTruthy()
    expect((screen.getByRole('textbox', { name: 'Your proof' }) as HTMLTextAreaElement).value).toBe('My proof.')
  })
})
