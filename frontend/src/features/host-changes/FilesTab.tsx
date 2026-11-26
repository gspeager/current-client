import { Fragment } from 'react'
import { HostService } from '@current-client-bindings/app'
import DiffStat from '../../components/git/DiffStat'
import PathText from '../../components/git/PathText'
import DependencySummary from '../diff/DependencySummary'
import StatusBadge from '../../components/git/StatusBadge'
import { useAsyncData } from '../../lib/useAsyncData'
import useTabPlaceholder, { type TabProps } from './useTabPlaceholder'
import './changeRows.scss'
import './FilesTab.scss'

const FILE_LETTER: Record<string, string> = { added: 'A', deleted: 'D', renamed: 'R', modified: 'M' }

function FilesTab({ repoPath, number, onLoaded }: TabProps) {
  const { data, error } = useAsyncData(() => HostService.ChangeFiles(repoPath, number), [repoPath, number])
  const placeholder = useTabPlaceholder(data?.length, error, onLoaded, 'No files.')
  const { data: dependencies } = useAsyncData(
    () => (data ? HostService.ChangeDependencySummaries(repoPath, number, data) : null),
    [repoPath, number, data],
  )
  if (placeholder || !data) return placeholder
  return (
    <>
      {dependencies?.hasManifests && !dependencies.headAvailable && (
        <p className="host-change-files-hint">
          Package changes show once this change's head and target are on this computer.
        </p>
      )}
      <ul className="host-change-files" aria-label="Files">
        {data.map((file) => {
          const summary = dependencies?.summaries[file.path]
          return (
            <Fragment key={file.path}>
              <li className="host-change-file">
                <StatusBadge status={FILE_LETTER[file.status] ?? 'M'} />
                <span className="host-change-file-path">
                  <PathText path={file.path} />
                  {file.oldPath && <span className="host-change-file-from"> from {file.oldPath}</span>}
                </span>
                <DiffStat added={file.additions} removed={file.deletions} binary={false} />
              </li>
              {summary && summary.changes.length > 0 && (
                <li>
                  <DependencySummary changes={summary.changes} />
                </li>
              )}
            </Fragment>
          )
        })}
      </ul>
    </>
  )
}

export default FilesTab
