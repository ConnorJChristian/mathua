// Code or printed output shown back to the learner ("you answered", "Your
// code", the expected output). Always the raw string as a React text child:
// no Markdown, no KaTeX, so `$d`, `\`, `**` and backticks stay as typed, and
// ligatures off (CODE_TEXT_CLASS, Q72) so `>=` reads as two characters.

import { CODE_TEXT_CLASS } from './PythonAnswer'

export default function CodeEcho({ text, inline = false, className = '' }: { text: string; inline?: boolean; className?: string }) {
  if (inline) {
    return <code data-code-echo className={`${CODE_TEXT_CLASS} font-mono whitespace-pre-wrap ${className}`}>{text}</code>
  }
  return <pre data-code-echo className={`${CODE_TEXT_CLASS} font-mono whitespace-pre ${className}`}>{text}</pre>
}
