import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { useState } from 'react'
import PythonAnswer from '../components/python/PythonAnswer'
import ProofReport from '../components/ProofReport'
import ReviewHost from '../app/review/ReviewHost'
import { startReviewSession, submitReviewAnswer, type ProofReport as Report } from '../lib/api'
import {
  CARD_MARKER, PROOF_MARKER, RECALL_MARKER, answeredKind, displayQuestion, isProofQuestion, isWrittenRecall, pythonKind,
} from '../lib/pythonQuestion'

vi.mock('../lib/api', async (importOriginal) => {
  const mod = await importOriginal<typeof import('../lib/api')>()
  return { ...mod, startReviewSession: vi.fn(), submitReviewAnswer: vi.fn() }
})

// The shape internal/pytrack/written.go serves: front, the proof marker, the
// recall marker, the back. Kept literal so a change to the server's format
// fails here.
const WRITTEN_Q = `${CARD_MARKER}\nProve that the sum of two odd numbers is even.\n\n${PROOF_MARKER}\n\n${RECALL_MARKER}\n\n\\((2a+1)+(2b+1) = 2(a+b+1)\\).`
const PLAIN_PROOF_Q = `**Prove.** Every convergent sequence is bounded.\n\n${PROOF_MARKER}`

const REPORT: Report = {
  score: 3, max: 7, runs: [3, 3, 4], fully_correct: false,
  milestones: [
    { id: 'm1', desc: 'Writes both numbers as 2k+1', points: 3, met: 'yes' },
    { id: 'm2', desc: 'Factors out 2', points: 4, met: 'no' },
  ],
  first_error: 'The sum is asserted to be even.', feedback: 'Factor out the 2.', counts_for_mastery: false,
}

describe('written-answer flashcard shape', () => {
  it('is a proof question that is also a flashcard', () => {
    expect(isWrittenRecall(WRITTEN_Q)).toBe(true)
    expect(isWrittenRecall(PLAIN_PROOF_Q)).toBe(false)
    expect(isProofQuestion(WRITTEN_Q)).toBe(true)
    expect(pythonKind(WRITTEN_Q)).toBe('proof')
    // The front only: neither marker nor the back.
    expect(displayQuestion(WRITTEN_Q)).toBe('Prove that the sum of two odd numbers is even.')
  })

  it('reports a self-marked answer as a recall card', () => {
    expect(answeredKind(WRITTEN_Q, 'right')).toBe('recall')
    expect(answeredKind(WRITTEN_Q, 'wrong')).toBe('recall')
    expect(answeredKind(WRITTEN_Q, 'Let the numbers be 2a+1 ...')).toBe('proof')
    expect(answeredKind(PLAIN_PROOF_Q, 'right')).toBe('proof')
  })
})

function Box() {
  const [v, setV] = useState('')
  return (
    <>
      <PythonAnswer id="w" question={WRITTEN_Q} value={v} onChange={setV} onSubmit={() => {}} />
      <output data-testid="value">{v}</output>
    </>
  )
}

describe('written-answer box', () => {
  it('takes a written answer, with a visible escape to self-grading', () => {
    render(<Box />)
    const box = screen.getByRole('textbox', { name: 'Your answer' })
    fireEvent.change(box, { target: { value: 'Odd plus odd is even.' } })
    expect(screen.getByTestId('value').textContent).toBe('Odd plus odd is even.')
    // The back is not on screen until asked for.
    expect(screen.queryByText(/2\(a\+b\+1\)/)).toBeNull()

    fireEvent.click(screen.getByRole('button', { name: 'Just show me the answer (self-grade)' }))
    // The back is shown, the written answer is dropped, and the verdict is the learner's.
    expect(screen.queryByRole('textbox', { name: 'Your answer' })).toBeNull()
    expect(screen.getByTestId('value').textContent).toBe('')
    expect(document.body.textContent).toMatch(/2a\+1/)
    fireEvent.click(screen.getByRole('button', { name: 'I had it' }))
    expect(screen.getByTestId('value').textContent).toBe('right')
  })

  it('a plain proof has no escape', () => {
    render(<PythonAnswer question={PLAIN_PROOF_Q} value="" onChange={() => {}} />)
    expect(screen.getByRole('textbox', { name: 'Your proof' })).toBeTruthy()
    expect(screen.queryByRole('button', { name: /Just show me the answer/ })).toBeNull()
  })
})

describe('written-answer report', () => {
  it('says the grade is recorded like a self-mark', () => {
    render(<ProofReport report={REPORT} flashcard />)
    expect(screen.getByText('3/7')).toBeTruthy()
    expect(screen.getByText(/as if you had marked the card yourself/)).toBeTruthy()
    expect(screen.queryByText(/until the grader is calibrated/)).toBeNull()
  })
})

describe('written-answer flashcard in a review', () => {
  beforeEach(() => {
    vi.mocked(startReviewSession).mockResolvedValue({
      student_id: 's1',
      session_id: 'r1',
      question: { concept_id: 'calc.x', concept_name: 'X', question: WRITTEN_Q, is_review: true, attempt_id: 'a1', grading_type: 'numeric' },
    })
    vi.mocked(submitReviewAnswer).mockReset()
  })

  it('grades a written answer: score, checklist, first error, then the back', async () => {
    vi.mocked(submitReviewAnswer).mockResolvedValueOnce({
      result: {
        correct: false, feedback: 'Score 3/7.', new_status: 'PRACTICING', evidence_score: 0, evidence_band: '',
        explanation: '\\((2a+1)+(2b+1) = 2(a+b+1)\\).', streak: 0, required_streak: 3, xp: 0, expected_answer: 'recall:x#w', proof: REPORT,
      },
      next_question: null,
      done: true,
    })
    render(<ReviewHost />)
    fireEvent.click(screen.getByRole('button', { name: /Start review/ }))
    const box = await screen.findByRole('textbox', { name: 'Your answer' })
    fireEvent.change(box, { target: { value: 'Odd plus odd is even.' } })
    fireEvent.click(screen.getByRole('button', { name: 'Check Answer' }))
    expect(await screen.findByText('3/7')).toBeTruthy()
    expect(screen.getByText('The sum is asserted to be even.')).toBeTruthy()
    expect(screen.getByRole('list', { name: 'Marking scheme' }).querySelectorAll('li')).toHaveLength(2)
    expect(screen.getByText(/as if you had marked the card yourself/)).toBeTruthy()
    expect(document.body.textContent).toMatch(/2a\+1/) // the back, revealed
    const call = vi.mocked(submitReviewAnswer).mock.calls[0]
    expect(call[1]).toBe('Odd plus odd is even.')
    expect(call[4]).toEqual({ proof: true })
  })

  it('self-grades through the escape hatch', async () => {
    vi.mocked(submitReviewAnswer).mockResolvedValueOnce({
      result: {
        correct: true, feedback: '', new_status: 'PRACTICING', evidence_score: 0, evidence_band: '',
        explanation: 'The back.', streak: 0, required_streak: 3, xp: 3, expected_answer: 'recall:x#w',
      },
      next_question: null,
      done: true,
    })
    render(<ReviewHost />)
    fireEvent.click(screen.getByRole('button', { name: /Start review/ }))
    fireEvent.click(await screen.findByRole('button', { name: 'Just show me the answer (self-grade)' }))
    fireEvent.click(screen.getByRole('button', { name: 'I had it' }))
    fireEvent.click(screen.getByRole('button', { name: 'Check Answer' }))
    expect(await screen.findByText(/Correct/)).toBeTruthy()
    expect(vi.mocked(submitReviewAnswer).mock.calls[0][1]).toBe('right')
  })
})
