import { SubmoduleService } from '@current-client-bindings/app'
import { useAsyncData } from '../../lib/useAsyncData'
import type { SubmoduleChange } from './submoduleChange'

interface SubmoduleDiffProps {
  change: SubmoduleChange
  path: string
  // Without it only the two commits are shown, not the ones between them.
  repoPath?: string
}

const short = (sha: string) => sha.slice(0, 7)

function SubmoduleDiff({ change, path, repoPath }: SubmoduleDiffProps) {
  const { from, to, dirty } = change
  const moved = from !== null && to !== null && from !== to
  const { data: commits, error } = useAsyncData(
    () => (repoPath && moved ? SubmoduleService.Commits(repoPath, path, from, to) : null),
    [repoPath, path, from, to, moved],
  )

  return (
    <div className="submodule-diff">
      <p className="submodule-diff-summary">
        Submodule{' '}
        {from && to ? (
          <span className="submodule-diff-shas">
            {short(from)} → {short(to)}
          </span>
        ) : to ? (
          <>
            added at <span className="submodule-diff-shas">{short(to)}</span>
          </>
        ) : (
          from && (
            <>
              removed, was at <span className="submodule-diff-shas">{short(from)}</span>
            </>
          )
        )}
      </p>
      {dirty && <p className="submodule-diff-note">Has uncommitted changes inside it.</p>}
      {error && <p className="submodule-diff-note">Update the submodule to see the commits between these versions.</p>}
      {commits && commits.length > 0 && (
        <ul className="submodule-diff-commits">
          {commits.map((c) => (
            <li key={c.sha} className={c.added ? 'submodule-diff-added' : 'submodule-diff-removed'}>
              <span className="submodule-diff-marker">{c.added ? '+' : '−'}</span>
              <span className="submodule-diff-sha">{c.sha}</span>
              <span className="submodule-diff-subject">{c.subject}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

export default SubmoduleDiff
