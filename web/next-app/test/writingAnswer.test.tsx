import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { useState } from 'react'
import PythonAnswer from '../components/python/PythonAnswer'
import ProofReport from '../components/ProofReport'
import type { ProofReport as Report } from '../lib/api'
import { CARD_MARKER, PROOF_MARKER, WRITING_MARKER, displayQuestion, isProofQuestion, isWritingQuestion, pythonKind, wordCount } from '../lib/pythonQuestion'

// The shape internal/proof serves for a problem of kind "writing": the card
// marker, the task, the proof marker and then the writing marker. Literal, so
// a change to the server's format fails here.
const WRITING_Q = `${CARD_MARKER}\n**Write.** Write a formal sentence definition of a term in your field.\n\n${PROOF_MARKER}${WRITING_MARKER}`
const PROOF_Q = `${CARD_MARKER}\n**Prove.** Every convergent sequence is bounded.\n\n${PROOF_MARKER}`

const REPORT: Report = {
  score: 5,
  max: 7,
  runs: [5, 5, 4],
  fully_correct: false,
  milestones: [{ id: 'm1', desc: 'Names the class', points: 7, met: 'partial' }],
  first_error: 'The class word is too general.',
  feedback: 'Choose a narrower class word.',
  counts_for_mastery: false,
}

function Box({ question }: { question: string }) {
  const [v, setV] = useState('')
  return <PythonAnswer id="w" question={question} value={v} onChange={setV} onSubmit={() => {}} />
}

describe('writing question', () => {
  it('is a proof-kind question recognised as writing, shown without markers', () => {
    expect(pythonKind(WRITING_Q)).toBe('proof')
    expect(isProofQuestion(WRITING_Q)).toBe(true)
    expect(isWritingQuestion(WRITING_Q)).toBe(true)
    expect(isWritingQuestion(PROOF_Q)).toBe(false)
    expect(displayQuestion(WRITING_Q)).toBe('**Write.** Write a formal sentence definition of a term in your field.')
  })

  it('counts words', () => {
    expect(wordCount('')).toBe(0)
    expect(wordCount('  \n ')).toBe(0)
    expect(wordCount('A tariff is\na tax  on imports.')).toBe(7)
  })

  it('labels the box "Your writing" and shows the word count', () => {
    render(<Box question={WRITING_Q} />)
    const box = screen.getByRole('textbox', { name: 'Your writing' })
    expect(screen.queryByRole('textbox', { name: 'Your proof' })).toBeNull()
    expect(document.querySelector('[data-word-count]')?.textContent).toBe('0 words')
    fireEvent.change(box, { target: { value: 'A tariff is a tax.' } })
    expect(document.querySelector('[data-word-count]')?.textContent).toBe('5 words')
    fireEvent.change(box, { target: { value: 'Tariff' } })
    expect(document.querySelector('[data-word-count]')?.textContent).toBe('1 word')
  })

  it('leaves the proof box as it was', () => {
    render(<Box question={PROOF_Q} />)
    expect(screen.getByRole('textbox', { name: 'Your proof' })).toBeTruthy()
    expect(document.querySelector('[data-word-count]')).toBeNull()
  })

  it('reports a writing grade in writing terms', () => {
    const { container } = render(<ProofReport report={REPORT} writing />)
    expect(container.textContent).toContain('Not yet at the standard')
    expect(container.textContent).toContain('First problem')
    expect(container.textContent).not.toContain('complete proof')
  })
})
