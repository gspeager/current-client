import { gitFailureFor, type GitFailure } from '../../lib/errors'
import './GitOutput.scss'

// Closed under a failure's sentence: the git command that failed and what it said.
function GitOutput({ message }: { message: string | null | undefined }) {
  const failure = gitFailureFor(message)
  if (!failure) return null
  const text = gitOutputText(failure)
  return (
    <details className="git-output">
      <summary>Show Git output</summary>
      <pre className="git-output-text">{text}</pre>
      <button type="button" className="git-output-copy" onClick={() => void navigator.clipboard.writeText(text)}>
        Copy
      </button>
    </details>
  )
}

export function gitOutputText({ command, exitCode, output }: GitFailure): string {
  return [command + (exitCode ? `\nexit ${exitCode}` : ''), output].filter(Boolean).join('\n\n')
}

export default GitOutput
