import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import AssistantDock, {
	ASSISTANT_CONSENT_KEY,
	ASSISTANT_CONSENT_VERSION,
} from '../components/AssistantDock'

const getConfigMock = vi.fn()
const askAssistantMock = vi.fn()

vi.mock('../lib/api', () => ({
	getConfig: (...a: unknown[]) => getConfigMock(...a),
	askAssistant: (...a: unknown[]) => askAssistantMock(...a),
}))

vi.mock('next/link', () => ({
	default: ({ href, children, ...rest }: { href: string; children: React.ReactNode }) => (
		<a href={href} {...rest}>
			{children}
		</a>
	),
}))

// next/dynamic would load the real KaTeX chunk asynchronously, which this test
// is not about. The stub records that the dock routed mathematics to the math
// component instead of printing the delimiters as text.
vi.mock('next/dynamic', () => ({
	default: () => ({ tex, display }: { tex: string; display: boolean }) => (
		<span data-testid="math" data-display={String(display)}>
			{tex}
		</span>
	),
}))

// The icon wrapper renders animated SVG through @animateicons/react, which is
// not what this test is about. A plain span keeps the assertions about copy and
// links, so a change to the icon set does not fail a behaviour test.
vi.mock('../components/icons', () => ({
	ArrowUp: () => <span aria-hidden="true">up</span>,
	Close: () => <span aria-hidden="true">close</span>,
	MessagesSquare: () => <span aria-hidden="true">chat</span>,
}))

function openPanel() {
	fireEvent.click(screen.getByRole('button', { name: 'Open assistant' }))
}

beforeEach(() => {
	getConfigMock.mockReset()
	askAssistantMock.mockReset()
	// These tests are about what the panel does, not about the notice that comes
	// before it. Seeding the acceptance keeps that step out of their way, and the
	// consent block below removes the seed to test the step itself.
	window.localStorage.clear()
	window.localStorage.setItem(ASSISTANT_CONSENT_KEY, ASSISTANT_CONSENT_VERSION)
})

describe('AssistantDock', () => {
	it('offers the button when the server says AI is off', async () => {
		getConfigMock.mockResolvedValue({ ai_enabled: false })
		render(<AssistantDock />)

		const trigger = await screen.findByRole('button', { name: 'Open assistant' })
		openPanel()

		expect(trigger).toBeInTheDocument()
		expect(await screen.findByText(/assistant is unavailable right now/i)).toBeInTheDocument()
		// The panel still routes a visitor somewhere useful, without a provider.
		expect(screen.getByRole('link', { name: 'Study' })).toHaveAttribute('href', '/today')
		// And it does not offer an input that cannot answer.
		expect(screen.queryByLabelText('Your question')).toBeNull()
	})

	it('still offers the button when the config read fails', async () => {
		getConfigMock.mockRejectedValue(new Error('offline'))
		render(<AssistantDock />)

		expect(await screen.findByRole('button', { name: 'Open assistant' })).toBeInTheDocument()
		openPanel()

		expect(await screen.findByText(/assistant is unavailable right now/i)).toBeInTheDocument()
	})

	it('answers, replays history and renders the reply as text', async () => {
		getConfigMock.mockResolvedValue({ ai_enabled: true })
		askAssistantMock.mockResolvedValue({
			reply: '<b>not markup</b> Learn',
			segments: [
				{ text: '<b>not markup</b> ' },
				{ link: { id: 'learn', label: 'Learn', href: '/learn' } },
			],
		})
		render(<AssistantDock />)

		await screen.findByRole('button', { name: 'Open assistant' })
		openPanel()

		const input = await screen.findByLabelText('Your question')
		fireEvent.change(input, { target: { value: 'Where do I start?' } })
		fireEvent.click(screen.getByRole('button', { name: 'Send' }))

		expect(await screen.findByText('<b>not markup</b>')).toBeInTheDocument()
		expect(screen.queryByText('not markup', { selector: 'b' })).toBeNull()
		// The label is printed once, as the link itself: no chip repeats it under
		// the sentence, which is what made a reply read "start here: Calculus"
		// and then show a Calculus box.
		expect(screen.getAllByText('Learn')).toHaveLength(1)
		expect(screen.getByRole('link', { name: 'Learn' })).toHaveAttribute('href', '/learn')
		expect(askAssistantMock).toHaveBeenCalledWith({ message: 'Where do I start?', history: [] })
	})

	it('renders mathematics instead of printing its delimiters', async () => {
		getConfigMock.mockResolvedValue({ ai_enabled: true })
		// The exact shape a live reply used for "explain how to find a derivative".
		const reply = String.raw`Apply the limit definition \(f'(x)=\lim_{h\to0}\frac{f(x+h)-f(x)}{h}\).`
		askAssistantMock.mockResolvedValue({ reply, segments: [{ text: reply }] })
		render(<AssistantDock />)

		await screen.findByRole('button', { name: 'Open assistant' })
		openPanel()

		fireEvent.change(await screen.findByLabelText('Your question'), {
			target: { value: 'how do i find a derivative' },
		})
		fireEvent.click(screen.getByRole('button', { name: 'Send' }))

		const math = await screen.findByTestId('math')
		expect(math).toHaveTextContent(String.raw`f'(x)=\lim_{h\to0}\frac{f(x+h)-f(x)}{h}`)
		expect(math).toHaveAttribute('data-display', 'false')
		// The delimiter itself must not survive as visible text.
		expect(screen.queryByText(String.raw`\(`)).toBeNull()
		expect(screen.getByText(/Apply the limit definition/)).toBeInTheDocument()
	})

	it('sends earlier turns as history', async () => {
		getConfigMock.mockResolvedValue({ ai_enabled: true })
		askAssistantMock.mockResolvedValue({ reply: 'Yes.', segments: [] })
		render(<AssistantDock />)

		await screen.findByRole('button', { name: 'Open assistant' })
		openPanel()

		const input = await screen.findByLabelText('Your question')
		fireEvent.change(input, { target: { value: 'What is Mathua?' } })
		fireEvent.click(screen.getByRole('button', { name: 'Send' }))
		await screen.findByText('Yes.')

		fireEvent.change(input, { target: { value: 'And where do I start?' } })
		fireEvent.click(screen.getByRole('button', { name: 'Send' }))

		await waitFor(() => expect(askAssistantMock).toHaveBeenCalledTimes(2))
		expect(askAssistantMock.mock.calls[1][0]).toEqual({
			message: 'And where do I start?',
			history: [
				{ role: 'user', content: 'What is Mathua?' },
				{ role: 'assistant', content: 'Yes.' },
			],
		})
	})

	it('names a spent quota differently from an outage', async () => {
		getConfigMock.mockResolvedValue({ ai_enabled: true })
		askAssistantMock.mockRejectedValue(new Error('429 Too Many Requests'))
		render(<AssistantDock />)

		await screen.findByRole('button', { name: 'Open assistant' })
		openPanel()

		fireEvent.change(await screen.findByLabelText('Your question'), { target: { value: 'hi' } })
		fireEvent.click(screen.getByRole('button', { name: 'Send' }))

		expect(await screen.findByText(/today’s limit/i)).toBeInTheDocument()
	})
})

// What a visitor types here is the only thing Mathua sends to another company,
// so the panel has to say so before it offers an input. The state that matters
// is the one before the first message: no acceptance, no form.
describe('AssistantDock consent', () => {
	beforeEach(() => {
		window.localStorage.clear()
	})

	it('asks before the first message and has no input until accepted', async () => {
		getConfigMock.mockResolvedValue({ ai_enabled: true })
		render(<AssistantDock />)

		await screen.findByRole('button', { name: 'Open assistant' })
		openPanel()

		expect(await screen.findByText(/before you send a message/i)).toBeInTheDocument()
		expect(screen.queryByLabelText('Your question')).toBeNull()

		const start = screen.getByRole('button', { name: 'Start' })
		expect(start).toBeDisabled()
		fireEvent.click(start)
		expect(screen.queryByLabelText('Your question')).toBeNull()

		fireEvent.click(screen.getByRole('checkbox', { name: /13 or older/i }))
		expect(start).toBeEnabled()
		fireEvent.click(start)

		expect(await screen.findByLabelText('Your question')).toBeInTheDocument()
		expect(window.localStorage.getItem(ASSISTANT_CONSENT_KEY)).toBe(ASSISTANT_CONSENT_VERSION)
	})

	it('does not ask again once accepted', async () => {
		window.localStorage.setItem(ASSISTANT_CONSENT_KEY, ASSISTANT_CONSENT_VERSION)
		getConfigMock.mockResolvedValue({ ai_enabled: true })
		render(<AssistantDock />)

		await screen.findByRole('button', { name: 'Open assistant' })
		openPanel()

		expect(await screen.findByLabelText('Your question')).toBeInTheDocument()
		expect(screen.queryByText(/before you send a message/i)).toBeNull()
	})

	it('asks again when the notice changes', async () => {
		// An old acceptance covers old wording. A changed version must re-ask, which
		// is the whole reason the stored value is a version and not a boolean.
		window.localStorage.setItem(ASSISTANT_CONSENT_KEY, 'v0')
		getConfigMock.mockResolvedValue({ ai_enabled: true })
		render(<AssistantDock />)

		await screen.findByRole('button', { name: 'Open assistant' })
		openPanel()

		expect(await screen.findByText(/before you send a message/i)).toBeInTheDocument()
		expect(screen.queryByLabelText('Your question')).toBeNull()
	})
})
