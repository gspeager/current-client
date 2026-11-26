import { HostService } from '@current-client-bindings/app'
import { plural } from '../../lib/plural'
import { relativeTime } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import { shortSha } from '../../lib/text'
import useTabPlaceholder, { type TabProps } from './useTabPlaceholder'

function CommitsTab({ repoPath, number, onLoaded }: TabProps) {
  const { data, error } = useAsyncData(() => HostService.ChangeCommits(repoPath, number), [repoPath, number])
  const placeholder = useTabPlaceholder(data?.length, error, onLoaded, 'No commits.')
  if (placeholder || !data) return placeholder
  return (
    <ul className="host-change-commits" aria-label={plural(data.length, 'commit')}>
      {data.map((commit) => (
        <li key={commit.sha} className="host-change-commit">
          <span className="host-change-commit-sha">{shortSha(commit.sha)}</span>
          <span className="host-change-commit-subject">{commit.subject}</span>
          <span className="host-change-commit-meta">
            {commit.author} · {relativeTime(new Date(commit.date))}
          </span>
        </li>
      ))}
    </ul>
  )
}

export default CommitsTab
