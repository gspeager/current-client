import clsx from 'clsx'
import { gitFailureFor, type GitFailure } from '../../lib/errors'
import './GitOutput.scss'

// Closed under a failure's sentence: the git command that failed and what it said.
// A popover opens over what's below instead of pushing it down.
function GitOutput({ message, popover = false }: { message: string | null | undefined; popover?: boolean }) {
  const failure = gitFailureFor(message)
  if (!failure) return null
  const text = gitOutputText(failure)
  return (
    <details className={clsx('git-output', popover && 'git-output-popover')}>
      <summary>Show Git output</summary>
      <div className="git-output-body">
        <pre className="git-output-text">{text}</pre>
        <button type="button" className="git-output-copy" onClick={() => void navigator.clipboard.writeText(text)}>
          Copy
        </button>
      </div>
    </details>
  )
}

export function gitOutputText({ command, exitCode, output }: GitFailure): string {
  return [command + (exitCode ? `\nexit ${exitCode}` : ''), output].filter(Boolean).join('\n\n')
}

export default GitOutput
