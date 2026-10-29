import { useEffect, useState } from 'react'
import { Plus, RefreshCw } from 'lucide-react'
import { HostService, type ListedIssue } from '@current-client-bindings/app'
import IssueDetail from '../host-changes/IssueDetail'
import { hostLogo, issueProvider } from '../../components/git/hostLogos'
import SegmentedControl from '../../components/controls/SegmentedControl'
import { errorMessage } from '../../lib/errors'
import { useNewIssue } from '../../lib/issuePanel'
import { relativeTime } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import { useClockTick } from '../../lib/useClockTick'
import { useHostLinks } from '../../lib/useHostLinks'
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

type Filter = 'mine' | 'all' | 'closed'

const keyOf = (listed: ListedIssue) => (/^\d+$/.test(listed.issue.key) ? `#${listed.issue.key}` : listed.issue.key)

// The issues on the host and the tracker: assigned to you from the last
// refresh, or the project's open or closed ones once asked for, with the
// selected one as the pane beside the list.
function IssuesView({ repoPath, selectedKey = null, refreshKey, onRefreshed, onBranchChanged }: IssuesViewProps) {
  useClockTick()
  const { data: list, reload } = useAsyncData(() => HostService.MyIssues(repoPath), [repoPath], {
    refreshKey,
    keepData: true,
  })
  const newIssue = useNewIssue()
  const hostLinks = useHostLinks(repoPath)
  const [selected, setSelected] = useState<string | null>(selectedKey)
  useEffect(() => setSelected(selectedKey), [selectedKey])
  const [refreshing, setRefreshing] = useState(false)
  const [refreshError, setRefreshError] = useState<string | null>(null)
  const [filter, setFilter] = useState<Filter>('mine')
  // All and Closed ask the services the first time each is chosen and are
  // kept until a refresh, as the Pull requests tab's Closed is.
  const {
    data: projectList,
    error: projectError,
    reload: reloadProject,
  } = useAsyncData(
    () => (filter === 'mine' ? null : HostService.ProjectIssues(repoPath, filter === 'all')),
    [repoPath, filter],
    { refreshKey },
  )

  const refresh = () => {
    setRefreshing(true)
    setRefreshError(null)
    HostService.Refresh(repoPath)
      .then(() => {
        reload()
        reloadProject()
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

  const shownList = filter === 'mine' ? list : projectList
  const issues = shownList?.issues ?? []
  const activeKey = selected ?? issues[0]?.issue.key ?? null
  const active = issues.find((i) => i.issue.key === activeKey)
  const fetched = (shownList ?? list).sources
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
        <SegmentedControl
          value={filter}
          onChange={(next) => {
            setFilter(next)
            setSelected(null)
          }}
          options={[
            { value: 'mine' as const, label: `Mine ${list.issues?.length ?? 0}` },
            { value: 'all' as const, label: 'All' },
            { value: 'closed' as const, label: 'Closed' },
          ]}
        />
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
      {(refreshError || projectError) && <p className="host-changes-error">{refreshError ?? projectError}</p>}
      <div className="host-changes-body">
        <ul className="host-changes-list" aria-label="Issues list">
          {issues.length === 0 && (
            <li className="host-changes-empty">
              {filter !== 'mine' && !projectList
                ? 'Loading…'
                : !fetched
                  ? 'Refresh to load.'
                  : { mine: 'None assigned to you.', all: 'None open.', closed: 'None closed.' }[filter]}
            </li>
          )}
          {issues.map((listed) => {
            const SourceLogo = hostLogo(issueProvider(listed.issue.key, hostLinks))
            return (
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
                    {listed.issue.status && <span className="issues-row-status">{listed.issue.status}</span>}
                    <span className="host-change-row-time issues-row-source">
                      <SourceLogo size={12} strokeWidth={1.75} aria-hidden="true" />
                      {listed.source}
                    </span>
                  </span>
                  <span className="host-change-row-title">{listed.issue.title}</span>
                  {listed.branch && <span className="host-change-row-branches">{listed.branch}</span>}
                </button>
              </li>
            )
          })}
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
