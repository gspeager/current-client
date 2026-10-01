import { PlatformService } from '@current-client-bindings/app'
import './ConventionalCommitsGuide.scss'

const EXAMPLES = [
  'feat(history): filter commits by type',
  'fix: keep lane color after rebase',
  'feat(config)!: drop v0 settings',
]

// Matches core/changelog's SectionFor.
const MAPPING = [
  ['! or BREAKING CHANGE: footer', 'Breaking'],
  ['feat', 'Added'],
  ['fix', 'Fixed'],
  ['perf, revert', 'Changed'],
  ['docs, style, refactor, test, build, ci, chore', 'Own sections, filtered out by default'],
  ['any other type', 'Its own section, named after the type'],
  ['no prefix', 'Other'],
]

// Everything here is local, so it works offline; only the link reaches out,
// and only when clicked.
function ConventionalCommitsGuide() {
  return (
    <div className="conventional-guide">
      <p>
        The changelog is built from commit subjects written as Conventional Commits: a type, an optional scope, and a
        description.
      </p>
      <code className="conventional-guide-format">type(scope)!: description</code>
      <ul className="conventional-guide-examples">
        {EXAMPLES.map((example) => (
          <li key={example}>{example}</li>
        ))}
      </ul>
      <table className="conventional-guide-mapping">
        <thead>
          <tr>
            <th>Commit</th>
            <th>Section</th>
          </tr>
        </thead>
        <tbody>
          {MAPPING.map(([types, section]) => (
            <tr key={types}>
              <td>{types}</td>
              <td>{section}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <p>The commit composer's type picker writes the prefix.</p>
      <button
        type="button"
        className="conventional-guide-link"
        onClick={() => void PlatformService.OpenConventionalCommitsPage()}
      >
        conventionalcommits.org
      </button>
    </div>
  )
}

export default ConventionalCommitsGuide
