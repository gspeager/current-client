import { useEffect, useState } from 'react'
import { Plus, RefreshCw } from 'lucide-react'
import { HostService, type ListedIssue } from '@current-client-bindings/app'
import IssueDetail from '../host-changes/IssueDetail'
import { errorMessage } from '../../lib/errors'
import { useNewIssue } from '../../lib/issuePanel'
import { relativeTime } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import { useClockTick } from '../../lib/useClockTick'
import '../host-changes/HostChangesView.scss'
import './IssuesView.scss'

interface IssuesViewProps {
  repoPath: string
  // An issue to show, such as one an Inbox row points at; it needn't be in the list.
  selectedKey?: string | null
  // Read again when it changes, as after a refresh elsewhere.
  refreshKey: unknown
  onRefreshed?: () => void
  onBranchChanged: () => void
}

const keyOf = (listed: ListedIssue) => (/^\d+$/.test(listed.issue.key) ? `#${listed.issue.key}` : listed.issue.key)

// The issues assigned to you on the host and the tracker, from the last
// refresh, with the selected one as the pane beside the list.
function IssuesView({ repoPath, selectedKey = null, refreshKey, onRefreshed, onBranchChanged }: IssuesViewProps) {
  useClockTick()
  const { data: list, reload } = useAsyncData(() => HostService.MyIssues(repoPath), [repoPath], {
    refreshKey,
    keepData: true,
  })
  const newIssue = useNewIssue()
  const [selected, setSelected] = useState<string | null>(selectedKey)
  useEffect(() => setSelected(selectedKey), [selectedKey])
  const [refreshing, setRefreshing] = useState(false)
  const [refreshError, setRefreshError] = useState<string | null>(null)

  const refresh = () => {
    setRefreshing(true)
    setRefreshError(null)
    HostService.Refresh(repoPath)
      .then(() => {
        reload()
        onRefreshed?.()
      })
      .catch((err: unknown) => setRefreshError(errorMessage(err)))
      .finally(() => setRefreshing(false))
  }

  // Like the Pull requests tab, the first open refreshes once; later opens read the cache.
  const neverFetched = !!list?.enabled && list.sources.every((s) => !s.fetchedAt)
  useEffect(() => {
    if (neverFetched) refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [neverFetched, repoPath])

  if (!list?.enabled) return null

  const issues = list.issues ?? []
  const activeKey = selected ?? issues[0]?.issue.key ?? null
  const active = issues.find((i) => i.issue.key === activeKey)
  const fetched = list.sources
    .map((s) => s.fetchedAt)
    .filter((at): at is string => !!at)
    .sort()[0]

  return (
    <section className="host-changes" aria-label="Issues">
      <header className="host-changes-header">
        <h2 className="host-changes-title">Issues</h2>
        <span className="host-changes-fetched">
          {list.sources.map((s) => s.name).join(' · ')}
          {fetched ? ` · fetched ${relativeTime(new Date(fetched))}` : ' · not fetched yet'}
        </span>
        <span className="issues-filter">Mine {issues.length}</span>
        <button
          type="button"
          className="host-changes-refresh"
          onClick={refresh}
          disabled={refreshing}
          aria-label="Refresh"
        >
          <RefreshCw size={14} strokeWidth={1.75} />
          {refreshing ? 'Refreshing…' : 'Refresh'}
        </button>
        {newIssue && (
          <button type="button" className="host-changes-new" onClick={() => newIssue()}>
            <Plus size={14} strokeWidth={1.75} />
            New issue
          </button>
        )}
      </header>
      {refreshError && <p className="host-changes-error">{refreshError}</p>}
      <div className="host-changes-body">
        <ul className="host-changes-list" aria-label="Issues list">
          {issues.length === 0 && (
            <li className="host-changes-empty">{fetched ? 'None assigned to you.' : 'Refresh to load.'}</li>
          )}
          {issues.map((listed) => (
            <li key={`${listed.source}:${listed.issue.key}`}>
              <button
                type="button"
                className={
                  listed.issue.key === activeKey ? 'host-change-row host-change-row-selected' : 'host-change-row'
                }
                onClick={() => setSelected(listed.issue.key)}
              >
                <span className="host-change-row-head">
                  <span className="issues-row-key">{keyOf(listed)}</span>
                  <span className="host-change-row-time">{listed.source}</span>
                </span>
                <span className="host-change-row-title">{listed.issue.title}</span>
                {listed.branch && <span className="host-change-row-branches">{listed.branch}</span>}
              </button>
            </li>
          ))}
        </ul>
        {activeKey && (
          <div className="issues-pane">
            <IssueDetail
              key={activeKey}
              repoPath={repoPath}
              issueKey={activeKey}
              branch={active?.branch}
              onBranchChanged={() => {
                onBranchChanged()
                reload()
              }}
            />
          </div>
        )}
      </div>
    </section>
  )
}

export default IssuesView
