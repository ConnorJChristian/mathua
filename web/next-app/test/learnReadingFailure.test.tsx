import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, within, waitFor } from '@testing-library/react'
import LearnStepper from '../components/LearnStepper'
import { getLessonKPs, getLessonPractice, getLessonReadiness, submitStudyAnswer } from '../lib/api'
import { answerCard, answerField, typeAnswer } from './helpers/answerInput'

vi.mock('../lib/api', async (importOriginal) => {
  const mod = await importOriginal<typeof import('../lib/api')>()
  return {
    ...mod,
    getLessonKPs: vi.fn(),
    getLessonPractice: vi.fn(),
    getLessonReadiness: vi.fn(),
    submitStudyAnswer: vi.fn(),
  }
})

Element.prototype.scrollIntoView = vi.fn() as unknown as typeof Element.prototype.scrollIntoView

const CID = 'arith.add.single'
const QS = [
  { question: '2 + 3 = ?', answer: '5', explanation: 'Add them.' },
  { question: '4 + 1 = ?', answer: '5', explanation: 'Add them.' },
]

beforeEach(() => {
  vi.mocked(getLessonPractice).mockResolvedValue({ questions: QS, concept_id: CID })
  vi.mocked(getLessonReadiness).mockResolvedValue({ concept_id: CID, ready: true, weak: [], missing: [] })
})

// The reading and the readiness check never gate the questions: when either
// request fails (network error, server down) the lesson still serves its
// questions, and says where the reading would have been.
describe('LearnStepper when the reading fails', () => {
  it('a failed reading fetch still serves the questions, with a note', async () => {
    vi.mocked(getLessonKPs).mockRejectedValue(new TypeError('Failed to fetch'))
    render(<LearnStepper conceptId={CID} />)
    expect(await screen.findByText(/The reading could not be loaded right now/)).toBeTruthy()
    expect(screen.queryByText('Loading lesson…')).toBeNull()
    fireEvent.click(await screen.findByRole('button', { name: 'Next →' }))
    expect(await screen.findByText('2 + 3 = ?')).toBeTruthy()
  })

  it('a reading that has not arrived does not hold back the first question', async () => {
    vi.mocked(getLessonKPs).mockReturnValue(new Promise(() => {}))
    render(<LearnStepper conceptId={CID} />)
    expect(await screen.findByText('Loading the reading…')).toBeTruthy()
    fireEvent.click(await screen.findByRole('button', { name: 'Next →' }))
    expect(await screen.findByText('2 + 3 = ?')).toBeTruthy()
  })

  it('a failed readiness check still serves the questions', async () => {
    vi.mocked(getLessonKPs).mockResolvedValue({ concept_id: CID, kps: [{ label: 'Add', subgoals: [], worked_example: '2 + 3 = 5' }] })
    vi.mocked(getLessonReadiness).mockRejectedValue(new TypeError('Failed to fetch'))
    render(<LearnStepper conceptId={CID} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Next →' }))
    expect(await screen.findByText('2 + 3 = ?')).toBeTruthy()
    expect(screen.queryByText(/could not be loaded/)).toBeNull()
  })
})

// The API going away mid-session, or before the questions load, is said in
// words where the learner is looking, with a way to try again.
describe('LearnStepper when the server does not answer', () => {
  beforeEach(() => {
    vi.mocked(getLessonKPs).mockResolvedValue({ concept_id: CID, kps: [{ label: 'Add', subgoals: [], worked_example: '2 + 3 = 5' }] })
  })

  it('a failed question load reads as such, and Retry loads the questions', async () => {
    vi.mocked(getLessonPractice).mockReset()
      .mockRejectedValueOnce(new TypeError('Failed to fetch'))
      .mockResolvedValue({ questions: QS, concept_id: CID })
    render(<LearnStepper conceptId={CID} />)
    const alert = await screen.findByRole('alert')
    expect(alert.textContent).toMatch(/server did not answer/)
    expect(alert.textContent).not.toMatch(/Failed to fetch/)
    fireEvent.click(within(alert).getByRole('button', { name: 'Retry' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Next →' }))
    expect(await screen.findByText('2 + 3 = ?')).toBeTruthy()
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('an answer that could not be graded says so under the question, not at the top of the feed', async () => {
    vi.mocked(submitStudyAnswer).mockRejectedValue(new TypeError('Failed to fetch'))
    render(<LearnStepper conceptId={CID} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Next →' }))
    await waitFor(() => expect(answerField()).toBeTruthy())
    const input = answerField()
    typeAnswer(input, '5')
    fireEvent.click(within(answerCard(input)).getByRole('button', { name: 'Check' }))
    const alert = await screen.findByRole('alert')
    expect(alert.textContent).toMatch(/Could not reach the server to grade/)
    // The message follows the question card in the page, where the learner is.
    expect(answerCard(input).compareDocumentPosition(alert) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })
})
