import { useEffect, useState } from 'react'
import { useEventCallback } from 'usehooks-ts'
import { useRovingRows } from '../../lib/useRovingRows'
import { hostUserName, useNameIdentities } from '../../lib/useNameIdentities'
import { Plus, RefreshCw } from 'lucide-react'
import { HostService, type ChangeList, type MergeResult } from '@current-client-bindings/app'
import { reviewRequestedOf } from '../../components/git/ReviewSummary'
import SegmentedControl from '../../components/controls/SegmentedControl'
import { createChangeFor } from '../../lib/createChange'
import { errorMessage } from '../../lib/errors'
import { relativeTime } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import { useClockTick } from '../../lib/useClockTick'
import './HostChangesView.scss'
import { changeNoun } from '../../lib/changeNoun'
import ChangeDetail from './ChangeDetail'
import ChangeRow from './ChangeRow'

type Filter = 'open' | 'mine' | 'review' | 'closed'

interface HostChangesViewProps {
  repoPath: string
  label: string
  // A change to select, from the sidebar's Details; the list's own selection otherwise.
  selectedNumber?: number | null
  onBranchChanged: () => void
  // After a refresh from this view's button.
  onRefreshed?: () => void
  // Opens the in-app create form for a branch; without it the host's form is used.
  onCreateChange?: (branch: string) => void
  onOpenSettings?: () => void
  // After a merge from the app, with whether a local branch tracks the merged one.
  onMerged?: (result: MergeResult) => void
}

// The host is contacted when the panel first opens, on Refresh, when the Closed
// filter is first chosen, and when a detail tab needs what the refresh didn't bring.
function HostChangesView({
  repoPath,
  label,
  selectedNumber = null,
  onBranchChanged,
  onRefreshed,
  onCreateChange,
  onOpenSettings,
  onMerged,
}: HostChangesViewProps) {
  const {
    data: list,
    error,
    reload,
  } = useAsyncData(() => HostService.Changes(repoPath), [repoPath], { keepData: true })
  const rovingRows = useRovingRows('.host-change-row', { selectOnMove: true })
  const [refreshing, setRefreshing] = useState(false)
  const [refreshError, setRefreshError] = useState<string | null>(null)
  const [refreshCount, setRefreshCount] = useState(0)
  const [filter, setFilter] = useState<Filter>('open')
  const { data: closedList, error: closedError } = useAsyncData(
    () => (filter === 'closed' ? HostService.ClosedChanges(repoPath) : null),
    [filter, repoPath],
    { refreshKey: refreshCount, keepData: true },
  )
  const { data: current } = useAsyncData(() => HostService.CurrentChange(repoPath), [repoPath], {
    refreshKey: refreshCount,
    keepData: true,
  })
  const [selected, setSelected] = useState<number | null>(null)
  useEffect(() => {
    if (selectedNumber !== null) {
      setSelected(selectedNumber)
      setFilter('open')
    }
  }, [selectedNumber])
  useClockTick()

  const refresh = useEventCallback(() => {
    setRefreshing(true)
    setRefreshError(null)
    HostService.Refresh(repoPath)
      .then(() => {
        reload()
        setRefreshCount((n) => n + 1)
        onRefreshed?.()
      })
      .catch((err: unknown) => setRefreshError(errorMessage(err)))
      .finally(() => setRefreshing(false))
  })

  const identities = useNameIdentities(
    (list?.changes ?? []).flatMap((c) => [
      hostUserName(c.author),
      ...(c.reviews ?? []).map((r) => hostUserName(r.reviewer)),
    ]),
    repoPath,
  )
  // Opening the panel is a refresh trigger, once: later opens read the cache.
  const neverFetched = list?.enabled === true && list.fetchedAt === null
  useEffect(() => {
    if (neverFetched) refresh()
  }, [neverFetched, repoPath, refresh])

  if (!list?.enabled) return null

  const kind = changeNoun(list.provider)
  const open = list.changes ?? []
  const mine = open.filter((c) => c.author.username === list.me)
  const toReview = open.filter((c) => reviewRequestedOf(c.reviews, list.me))
  const shown = { open, mine, review: toReview, closed: closedList?.changes ?? [] }[filter]
  const active = shown.find((c) => c.number === selected) ?? shown[0] ?? null
  const existingChange = current?.change
    ? `${current.branch} already has ${current.change.prefix}${current.change.number}`
    : null
  const filters = [
    { value: 'open' as const, label: `Open ${open.length}` },
    ...(list.me
      ? [
          { value: 'mine' as const, label: `Mine ${mine.length}` },
          { value: 'review' as const, label: `To review ${toReview.length}` },
        ]
      : []),
    { value: 'closed' as const, label: 'Closed' },
  ]

  return (
    <section className="host-changes" aria-label={label}>
      <header className="host-changes-header">
        <h2 className="host-changes-title">{label}</h2>
        <span className="host-changes-fetched">
          {list.host}
          {list.fetchedAt ? ` · fetched ${relativeTime(new Date(list.fetchedAt))}` : ' · not fetched yet'}
          {existingChange ? ` · ${existingChange}` : ''}
        </span>
        <SegmentedControl value={filter} onChange={setFilter} options={filters} />
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
        <button
          type="button"
          className="host-change-button"
          onClick={() => current && void createChangeFor(repoPath, current.branch, onCreateChange)}
          disabled={!current?.branch || !!existingChange}
          title={existingChange ?? (current?.branch ? `From ${current.branch}` : undefined)}
        >
          <Plus size={14} strokeWidth={1.75} />
          New {kind}
        </button>
      </header>
      {(error || refreshError || closedError) && (
        <p className="host-changes-error">{refreshError ?? error ?? closedError}</p>
      )}
      <div className="host-changes-body">
        <div className="host-changes-column" data-pane="change-list">
          <ul ref={rovingRows} className="host-changes-list" aria-label={`${label} list`}>
            {shown.length === 0 && (
              <li className="host-changes-empty">
                {filter === 'closed'
                  ? closedList
                    ? 'None closed.'
                    : 'Loading…'
                  : list.fetchedAt
                    ? filter === 'review'
                      ? 'No reviews waiting.'
                      : 'None open.'
                    : 'Refresh to load.'}
              </li>
            )}
            {shown.map((change) => (
              <li key={change.number}>
                <ChangeRow
                  change={change}
                  prefix={list.prefix}
                  selected={active?.number === change.number}
                  onSelect={() => setSelected(change.number)}
                  me={list.me}
                  identity={identities.get(hostUserName(change.author))}
                  displayName={(name) => identities.get(name)?.displayName ?? name}
                />
              </li>
            ))}
          </ul>
          <footer className="host-changes-privacy">
            {(filter === 'closed' ? closedList?.truncated : list.truncated) && <p>List cut at the page cap.</p>}
            {onOpenSettings && (
              <button
                type="button"
                className="host-changes-privacy-link"
                onClick={onOpenSettings}
                title={`${list.host} is contacted on Refresh, when this tab first opens, and when a detail tab needs more.`}
              >
                Integration settings
              </button>
            )}
          </footer>
        </div>
        {active && (
          <ChangeDetail
            key={active.number}
            repoPath={repoPath}
            change={active}
            list={list}
            onBranchChanged={onBranchChanged}
            onMerged={onMerged}
            onReviewed={reload}
          />
        )}
      </div>
    </section>
  )
}

export type { ChangeList }

export default HostChangesView
