import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { useState } from 'react'
import PythonAnswer from '../components/python/PythonAnswer'
import CodeEcho from '../components/python/CodeEcho'
import KatexContent from '../components/KatexContent'
import LatexField from '../components/math/LatexField'
import { MathAnswerInput } from '../components/math/MathInput'
import ReviewHost from '../app/review/ReviewHost'
import { startReviewSession, submitReviewAnswer } from '../lib/api'
import { PROOF_MARKER, RECALL_MARKER, TERM_MARKER, WRITING_MARKER, pythonKind } from '../lib/pythonQuestion'

vi.mock('../lib/api', async (importOriginal) => {
  const mod = await importOriginal<typeof import('../lib/api')>()
  return { ...mod, startReviewSession: vi.fn(), submitReviewAnswer: vi.fn() }
})

// Q71 / Q72 (local-data/qa/FSRS_PORT_PLAN.md rows 0a, 0b). Question shapes as
// internal/pytrack serves them, kept literal so a format change fails here.
const OUTPUT_Q =
  'Like the tutorial\'s loop that measures some words, what does this print?\n\n```python\nfor series in ["GDP", "CPI", "unemployment"]:\n    print(series, len(series))\n```'
const CODE_Q =
  'Set `status`.\n\n```python starter task=py.basics.conditionals#c1\nif rate >= 6:\n    status = ...\n```'
const PROOF_Q = `**Prove.** Every convergent sequence is bounded.\n\n${PROOF_MARKER}`
const WRITING_Q = `Write a paragraph.\n\n${PROOF_MARKER}${WRITING_MARKER}`
const WRITTEN_RECALL_Q = `Derive the OLS estimator.\n\n${PROOF_MARKER}${RECALL_MARKER}\n\\(\\hat\\beta = (X'X)^{-1}X'y\\)`
const TERM_Q = `The name of this estimator?\n\n${TERM_MARKER}`

// Every operator a ligature font merges, plus what Markdown or KaTeX would eat.
const TRICKY = 'x >= 1 and y <= 2 != 3 -> a ** b ^ c $d \\'

function Box({ question, onSubmit }: { question: string; onSubmit: () => void }) {
  const [v, setV] = useState('')
  return (
    <>
      <PythonAnswer id="a" question={question} value={v} onChange={setV} onSubmit={onSubmit} />
      <output data-testid="value">{v}</output>
    </>
  )
}
const value = () => screen.getByTestId('value').textContent

// Typing, one change per character, as a browser reports it.
function type(box: HTMLTextAreaElement | HTMLInputElement, text: string) {
  const start = box.value
  for (let i = 1; i <= text.length; i++) fireEvent.change(box, { target: { value: start + text.slice(0, i) } })
}

const MULTILINE: { name: string; question: string; label: string }[] = [
  { name: 'output', question: OUTPUT_Q, label: 'What the code prints' },
  { name: 'code', question: CODE_Q, label: 'Your Python code' },
  { name: 'proof', question: PROOF_Q, label: 'Your proof' },
  { name: 'writing', question: WRITING_Q, label: 'Your writing' },
  { name: 'written flashcard', question: WRITTEN_RECALL_Q, label: 'Your answer' },
]

describe('multi-line answer boxes (Q71)', () => {
  it('an output item is a multi-line textarea', () => {
    expect(pythonKind(OUTPUT_Q)).toBe('output')
    render(<Box question={OUTPUT_Q} onSubmit={() => {}} />)
    expect(screen.getByRole('textbox', { name: 'What the code prints' }).tagName).toBe('TEXTAREA')
  })

  for (const { name, question, label } of MULTILINE) {
    it(`${name}: Enter adds a new line and does not check`, () => {
      const submit = vi.fn()
      render(<Box question={question} onSubmit={submit} />)
      const box = screen.getByRole('textbox', { name: label }) as HTMLTextAreaElement
      fireEvent.change(box, { target: { value: 'GDP 3' } })
      const ev = fireEvent.keyDown(box, { key: 'Enter' })
      expect(ev).toBe(false) // handled here, not left to a form
      expect(submit).not.toHaveBeenCalled()
      expect(value()).toBe('GDP 3\n')
    })

    it(`${name}: Ctrl+Enter and Cmd+Enter check`, () => {
      const submit = vi.fn()
      render(<Box question={question} onSubmit={submit} />)
      const box = screen.getByRole('textbox', { name: label })
      fireEvent.change(box, { target: { value: 'GDP 3' } })
      fireEvent.keyDown(box, { key: 'Enter', ctrlKey: true })
      fireEvent.keyDown(box, { key: 'Enter', metaKey: true })
      expect(submit).toHaveBeenCalledTimes(2)
      expect(value()).toBe('GDP 3')
    })

    it(`${name}: says how to check under the box`, () => {
      render(<Box question={question} onSubmit={() => {}} />)
      expect(document.querySelector('[data-submit-hint]')?.textContent).toMatch(/Enter adds a line.*Ctrl\+Enter to check/)
    })
  }

  it('output: the three printed lines typed with Enter between them are the answer', () => {
    const submit = vi.fn()
    render(<Box question={OUTPUT_Q} onSubmit={submit} />)
    const box = screen.getByRole('textbox', { name: 'What the code prints' }) as HTMLTextAreaElement
    for (const [i, line] of ['GDP 3', 'CPI 3', 'unemployment 12'].entries()) {
      if (i > 0) fireEvent.keyDown(box, { key: 'Enter' })
      type(box, line)
    }
    expect(submit).not.toHaveBeenCalled()
    expect(value()).toBe('GDP 3\nCPI 3\nunemployment 12')
  })

  it('code: Enter keeps the indentation and indents after a colon', () => {
    render(<Box question={CODE_Q} onSubmit={() => {}} />)
    const box = screen.getByRole('textbox', { name: 'Your Python code' }) as HTMLTextAreaElement
    fireEvent.change(box, { target: { value: '' } })
    type(box, 'for x in xs:')
    fireEvent.keyDown(box, { key: 'Enter' })
    expect(value()).toBe('for x in xs:\n    ')
  })
})

describe('code box dedent', () => {
  it('Shift+Tab puts the caret back before the next keystroke can land', () => {
    render(<Box question={CODE_Q} onSubmit={() => {}} />)
    const box = screen.getByRole('textbox', { name: 'Your Python code' }) as HTMLTextAreaElement
    fireEvent.change(box, { target: { value: 'if x:\n    y\nz' } })
    box.selectionStart = box.selectionEnd = 11 // after "    y"
    fireEvent.keyDown(box, { key: 'Tab', shiftKey: true })
    expect(value()).toBe('if x:\ny\nz')
    // Synchronously, not on the next animation frame.
    expect(box.selectionStart).toBe(7)
  })
})

describe('single-line answer boxes keep Enter to check (Q71)', () => {
  it('term', () => {
    const submit = vi.fn()
    render(<Box question={TERM_Q} onSubmit={submit} />)
    const box = screen.getByRole('textbox', { name: 'The term' })
    expect(box.tagName).toBe('INPUT')
    fireEvent.keyDown(box, { key: 'Enter' })
    expect(submit).toHaveBeenCalledTimes(1)
  })

  it('a text answer (numbers, an MC letter)', () => {
    const submit = vi.fn()
    render(<MathAnswerInput value="B" onChange={() => {}} onSubmit={submit} gradingType="multiple_choice" ariaLabel="Your answer" />)
    fireEvent.keyDown(screen.getByRole('textbox', { name: 'Your answer' }), { key: 'Enter' })
    expect(submit).toHaveBeenCalledTimes(1)
  })

  it('the LaTeX field', () => {
    const submit = vi.fn()
    render(<LatexField value="" onChange={() => {}} onSubmit={submit} />)
    fireEvent.keyDown(screen.getByRole('textbox'), { key: 'Enter' })
    expect(submit).toHaveBeenCalledTimes(1)
  })
})

describe('code, output boxes are plain text (Q72)', () => {
  for (const { name, question, label } of MULTILINE.slice(0, 2)) {
    it(`${name}: what is typed is the value, byte for byte, and is what Ctrl+Enter sends`, () => {
      let sent = ''
      function Host() {
        const [v, setV] = useState('')
        return <PythonAnswer question={question} value={v} onChange={setV} onSubmit={() => { sent = v }} />
      }
      render(<Host />)
      const box = screen.getByRole('textbox', { name: label }) as HTMLTextAreaElement
      fireEvent.change(box, { target: { value: '' } })
      type(box, TRICKY)
      expect(box.value).toBe(TRICKY)
      fireEvent.keyDown(box, { key: 'Enter', ctrlKey: true })
      expect(sent).toBe(TRICKY)
    })

    it(`${name}: ligatures off, no spell-check, autocorrect, autocapitalise or autocomplete`, () => {
      render(<Box question={question} onSubmit={() => {}} />)
      const box = screen.getByRole('textbox', { name: label })
      expect(box.className).toContain('mathua-code-text')
      expect(box.getAttribute('spellcheck')).toBe('false')
      expect(box.getAttribute('autocorrect')).toBe('off')
      expect(box.getAttribute('autocapitalize')).toBe('off')
      expect(box.getAttribute('autocomplete')).toBe('off')
    })
  }

  it('the code echo is the raw string, untypeset, with ligatures off', () => {
    const src = `if rate >= 6:\n    s = "$d" ** 2  # \\(x\\) \`y\``
    const { container } = render(<><CodeEcho text={src} /><CodeEcho text={TRICKY} inline /></>)
    const [block, inline] = Array.from(container.querySelectorAll('[data-code-echo]'))
    expect(block.textContent).toBe(src)
    expect(inline.textContent).toBe(TRICKY)
    expect(container.querySelector('.katex')).toBeNull()
    for (const el of [block, inline]) expect(el.className).toContain('mathua-code-text')
  })

  it('fenced code in a prompt or explanation is shown as written, ligatures off', () => {
    const { container } = render(<KatexContent>{'Check `x >= 1` first.\n\n```python\nif a >= b and c != d:\n    print("$d \\\\")\n```'}</KatexContent>)
    const pre = container.querySelector('pre')!
    expect(pre.textContent!.trimEnd()).toBe('if a >= b and c != d:\n    print("$d \\\\")')
    expect(pre.className).toContain('mathua-code-text')
    expect(container.querySelector('p code')!.textContent).toBe('x >= 1')
    expect(container.querySelector('p code')!.className).toContain('mathua-code-text')
  })
})

describe('a code answer in a review is submitted as typed (Q72)', () => {
  beforeEach(() => {
    vi.mocked(startReviewSession).mockResolvedValue({
      student_id: 's1',
      session_id: 'r1',
      question: { concept_id: 'py.basics.conditionals', concept_name: 'Conditionals', question: CODE_Q, is_review: true, attempt_id: 'a1', grading_type: 'bank' },
    })
    vi.mocked(submitReviewAnswer).mockReset()
    vi.mocked(submitReviewAnswer).mockResolvedValue({
      result: {
        correct: true, feedback: 'Correct', new_status: 'PRACTICING', evidence_score: 0, evidence_band: '',
        explanation: '', streak: 1, required_streak: 3, xp: 0, expected_answer: 'code:py.basics.conditionals#c1',
      },
      next_question: null,
      done: true,
    })
  })

  it('sends the typed operators unchanged', async () => {
    render(<ReviewHost />)
    fireEvent.click(screen.getByRole('button', { name: /Start review/ }))
    const box = (await screen.findByRole('textbox', { name: 'Your Python code' })) as HTMLTextAreaElement
    fireEvent.change(box, { target: { value: '' } })
    type(box, TRICKY)
    fireEvent.click(screen.getByRole('button', { name: 'Check Answer' }))
    await vi.waitFor(() => expect(submitReviewAnswer).toHaveBeenCalled())
    expect(vi.mocked(submitReviewAnswer).mock.calls[0][1]).toBe(TRICKY)
  })
})
