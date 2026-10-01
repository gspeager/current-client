import { DashboardService } from '@current-client-bindings/app'
import { extensionOf, languageColor } from './languageColors'
import { useAsyncData } from '../../lib/useAsyncData'
import PathText from '../../components/git/PathText'
import './FileChurnList.scss'

interface FileChurnListProps {
  repoPath: string
  repoVersion: number
}

function FileChurnList({ repoPath, repoVersion }: FileChurnListProps) {
  const { data: churn, error } = useAsyncData(() => DashboardService.GetFileChurn(repoPath), [repoPath], {
    refreshKey: repoVersion,
  })

  const maxChanges = Math.max(1, ...(churn ?? []).map((c) => c.changes))

  return (
    <div className="file-churn">
      <div className="file-churn-header">
        <div className="file-churn-heading">
          <span className="file-churn-title">File churn analysis</span>
          <span className="file-churn-subtitle">Files with the highest revision rate in the last 90 days</span>
        </div>
        <span className="file-churn-badge">Filtered: Last 90 days</span>
      </div>

      <div className="file-churn-body">
        {error ? (
          <p className="file-churn-hint">Could not load file churn: {error}</p>
        ) : churn === null ? (
          <p className="file-churn-hint">Loading…</p>
        ) : churn.length === 0 ? (
          <p className="file-churn-hint">No changes in this window.</p>
        ) : (
          <ul className="file-churn-list">
            {churn.map((c) => {
              const extension = extensionOf(c.path)
              const color = languageColor(extension)
              return (
                <li key={c.path} className="file-churn-row">
                  <span className="file-churn-extension" style={{ color }}>
                    {extension === 'other' ? '—' : extension.toUpperCase()}
                  </span>
                  <PathText path={c.path} className="file-churn-path" />
                  <span className="file-churn-bar-wrap">
                    <span
                      className="file-churn-bar"
                      style={{ width: `${(c.changes / maxChanges) * 100}%`, background: color }}
                    />
                  </span>
                  <span className="file-churn-count">{c.changes}</span>
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </div>
  )
}

export default FileChurnList
