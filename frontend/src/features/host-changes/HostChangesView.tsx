import { Fragment, useEffect, useMemo, useState } from 'react'
import {
  CircleCheck,
  CircleHelp,
  ClipboardCheck,
  ExternalLink,
  GitBranch,
  GitMerge,
  Info,
  Plus,
  RefreshCw,
  TriangleAlert,
  type LucideIcon,
} from 'lucide-react'
import {
  BranchChange,
  HostService,
  PlatformService,
  type ChangeList,
  type MergeResult,
} from '@current-client-bindings/app'
import type { Change } from '@current-client-bindings/core/hostapi'
import ChangeMarker from '../../components/git/ChangeMarker'
import CIMark, { ciLabel, PipelineJobList } from '../../components/git/CIMark'
import { EnvironmentReach } from '../../components/git/EnvironmentMarker'
import DiffStat from '../../components/git/DiffStat'
import HostMarkdown from '../../components/git/HostMarkdown'
import ReferenceText from '../../components/git/ReferenceText'
import { useHostLinks } from '../../lib/useHostLinks'
import { useReferenceCards } from '../../lib/useReferenceCards'
import PathText from '../../components/git/PathText'
import ReviewSummary, { reviewGroups, reviewRequestedOf } from '../../components/git/ReviewSummary'
import StatusBadge from '../../components/git/StatusBadge'
import SegmentedControl from '../../components/controls/SegmentedControl'
import Tab from '../../components/controls/Tab'
import ReviewThreadBlock from '../diff/ReviewThreadBlock'
import ReviewDialog from '../review/ReviewDialog'
import { useReviewActions } from '../review/useReviewActions'
import { useReviewDraft } from '../../lib/reviewDraft'
import MergeChangeDialog from './MergeChangeDialog'
import { createChangeFor } from '../../lib/createChange'
import { errorMessage, ignoreDecorativeFailure } from '../../lib/errors'
import { plural } from '../../lib/plural'
import { relativeTime } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import { useCancellableOperation } from '../../lib/cancellableOperation'
import { useClockTick } from '../../lib/useClockTick'
import './HostChangesView.scss'

type Filter = 'open' | 'mine' | 'review' | 'closed'
type DetailTab = 'overview' | 'checks' | 'threads' | 'files' | 'commits'

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

// The merge request / pull request panel: the open changes from the last
// refresh, details for one, and checking it out locally. The host is
// contacted when the panel first opens, on Refresh, when the Closed filter
// is first chosen, and when a detail tab needs what the refresh didn't bring.
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

  const refresh = () => {
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
  }

  // Opening the panel is a refresh trigger, once: later opens read the cache.
  const neverFetched = list?.enabled === true && list.fetchedAt === null
  useEffect(() => {
    if (neverFetched) refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [neverFetched, repoPath])

  if (!list?.enabled) return null

  const kind = list.prefix === '!' ? 'merge request' : 'pull request'
  const open = list.changes ?? []
  const mine = open.filter((c) => c.author.username === list.me)
  const toReview = open.filter((c) => reviewRequestedOf(c.reviews, list.me))
  const shown = { open, mine, review: toReview, closed: closedList?.changes ?? [] }[filter]
  const active = shown.find((c) => c.number === selected) ?? shown[0] ?? null
  const currentHasChange = !!current?.change
  const newLabel = `New ${kind}`
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
          className="host-changes-new"
          onClick={() => current && void createChangeFor(repoPath, current.branch, onCreateChange)}
          disabled={!current?.branch || currentHasChange}
          title={
            currentHasChange && current?.change
              ? `${current.branch} already has ${current.change.prefix}${current.change.number}`
              : current?.branch
                ? `From ${current.branch}`
                : undefined
          }
        >
          <Plus size={14} strokeWidth={1.75} />
          {newLabel}
        </button>
      </header>
      {(error || refreshError || closedError) && (
        <p className="host-changes-error">{refreshError ?? error ?? closedError}</p>
      )}
      <div className="host-changes-body">
        <div className="host-changes-column">
          <ul className="host-changes-list" aria-label={`${label} list`}>
            {shown.length === 0 && (
              <li className="host-changes-empty">
                {filter === 'closed'
                  ? closedList
                    ? 'None closed.'
                    : 'Loading…'
                  : list.fetchedAt
                    ? filter === 'review'
                      ? 'None waiting on a review from you.'
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

interface ChangeRowProps {
  change: Change
  prefix: string
  selected: boolean
  onSelect: () => void
  // The signed-in username, so a request for review reads as yours or someone else's.
  me: string
}

function ChangeRow({ change, prefix, selected, onSelect, me }: ChangeRowProps) {
  return (
    <button
      type="button"
      className={selected ? 'host-change-row host-change-row-selected' : 'host-change-row'}
      onClick={onSelect}
    >
      <span className="host-change-row-head">
        <ChangeMarker
          change={
            new BranchChange({ number: change.number, prefix, state: change.state, draft: change.draft, ci: change.ci })
          }
        />
        <span className="host-change-row-meta">
          <ReviewSummary reviews={change.reviews} open={change.state === 'open'} me={me} />
          <span className="host-change-row-time">{relativeTime(new Date(change.updatedAt))}</span>
        </span>
      </span>
      <span className="host-change-row-title">{change.title}</span>
      <span className="host-change-row-branches">
        {change.sourceProject ? `${change.sourceProject}:` : ''}
        {change.sourceBranch} → {change.targetBranch}
      </span>
    </button>
  )
}

interface ChangeDetailProps {
  repoPath: string
  change: Change
  list: ChangeList
  onBranchChanged: () => void
  onMerged?: (result: MergeResult) => void
  onReviewed?: () => void
}

const NO_CHANGES = {}

const FACT_ICONS: Record<string, LucideIcon> = {
  ok: CircleCheck,
  warn: TriangleAlert,
  info: Info,
  unknown: CircleHelp,
}

function ChangeDetail({ repoPath, change, list, onBranchChanged, onMerged, onReviewed }: ChangeDetailProps) {
  const { data: details } = useAsyncData(
    () => HostService.ChangeDetails(repoPath, change.number),
    [repoPath, change.number],
  )
  const hostLinks = useHostLinks(repoPath)
  const closes = useMemo(() => details?.closes ?? [], [details?.closes])
  const closesCards = useReferenceCards(repoPath, closes, NO_CHANGES)
  const { data: forecast } = useAsyncData(() => HostService.Forecast(repoPath), [repoPath])
  const { data: marks } = useAsyncData(
    () => (change.headSha ? HostService.CIStatus(repoPath, [change.headSha]) : null),
    [repoPath, change.headSha],
  )
  const [tab, setTab] = useState<DetailTab>('overview')
  // The refresh caches no pipeline for a change whose list entry carried its
  // CI, so the Checks tab asks for the head's once when the cache has none.
  const { data: fetchedMarks } = useAsyncData(
    () =>
      tab === 'checks' && marks && !marks[change.headSha]
        ? HostService.FetchCIStatus(repoPath, [change.headSha])
        : null,
    [tab, marks, repoPath, change.headSha],
  )
  const mark = marks?.[change.headSha] ?? fetchedMarks?.[change.headSha]
  const conflict = forecast?.conflicts?.find((c) => c.number === change.number)
  const checkout = useCancellableOperation(`${repoPath}:${change.number}`)
  const [checkedOut, setCheckedOut] = useState<string | null>(null)
  // What the last refresh knows about merging it; cache and Git only.
  const { data: readiness } = useAsyncData(
    () => (change.state === 'open' ? HostService.MergeReadiness(repoPath, change.number) : null),
    [repoPath, change],
  )
  const [merging, setMerging] = useState(false)
  const [merged, setMerged] = useState<string | null>(null)
  const reviewActions = useReviewActions(repoPath, change.number)
  const [reviewing, setReviewing] = useState(false)
  const [reviewed, setReviewed] = useState(false)
  const [counts, setCounts] = useState<Partial<Record<DetailTab, number>>>({})
  const countFor = useMemo(() => {
    const setter = (which: DetailTab) => (count: number) =>
      setCounts((c) => (c[which] === count ? c : { ...c, [which]: count }))
    return { checks: setter('checks'), threads: setter('threads'), files: setter('files'), commits: setter('commits') }
  }, [])
  const state = change.draft ? 'draft' : change.state
  const reviews = reviewGroups(change.reviews, change.state === 'open')
  const threadCount = counts.threads ?? (change.unresolvedThreads > 0 ? change.unresolvedThreads : undefined)
  // The pane's one primary follows the state: your review when it is asked
  // for, Merge once nothing stands in the way, otherwise none.
  const primary =
    change.state !== 'open' || merged
      ? null
      : reviewActions && reviewRequestedOf(change.reviews, list.me)
        ? 'review'
        : readiness && readiness.facts.every((f) => f.level === 'ok' || f.level === 'info')
          ? 'merge'
          : null

  const checkOut = () =>
    checkout.run(
      (auth) => HostService.CheckoutChange(repoPath, change.number, auth),
      (result) => {
        setCheckedOut(
          result.existed ? `${result.branch} already existed and is checked out.` : `Checked out ${result.branch}.`,
        )
        onBranchChanged()
      },
    )

  const tabs: { id: DetailTab; label: string; count?: number }[] = [
    { id: 'overview', label: 'Overview' },
    { id: 'checks', label: 'Checks', count: counts.checks },
    { id: 'threads', label: 'Threads', count: threadCount },
    { id: 'files', label: 'Files', count: counts.files },
    { id: 'commits', label: 'Commits', count: counts.commits },
  ]

  return (
    <article className="host-change-detail" aria-label={`${list.prefix}${change.number}`}>
      <header className="host-change-detail-header">
        <span className="host-change-detail-number">
          {list.prefix}
          {change.number}
        </span>
        <span className={`host-change-state host-change-state-${state}`}>{state}</span>
        <h3 className="host-change-detail-title">{change.title}</h3>
      </header>
      <dl className="host-change-facts">
        <dt>Author</dt>
        <dd>{change.author.name}</dd>
        <dt>Branches</dt>
        <dd className="host-change-facts-branches">
          {change.sourceProject ? `${change.sourceProject}:` : ''}
          {change.sourceBranch} → {change.targetBranch}
        </dd>
        {closes.length > 0 && (
          <>
            <dt>Closes</dt>
            <dd className="host-change-facts-closes">
              <ReferenceText text={closes.join(', ')} links={hostLinks} cards={closesCards} />
            </dd>
          </>
        )}
        {(details?.environments?.length ?? 0) > 0 && (
          <>
            <dt>Environments</dt>
            <dd>
              <EnvironmentReach reach={details!.environments} />
            </dd>
          </>
        )}
        {readiness && readiness.facts.length > 0 ? (
          readiness.facts.map((f) => {
            const Icon = FACT_ICONS[f.level] ?? CircleHelp
            return (
              <Fragment key={f.label}>
                <dt>{f.label}</dt>
                <dd className={`host-change-fact host-change-fact-${f.level}`}>
                  {f.label === 'Checks' && mark ? (
                    <CIMark repoPath={repoPath} mark={mark} />
                  ) : (
                    <Icon size={14} strokeWidth={1.75} aria-hidden="true" />
                  )}
                  <span>
                    {f.value}
                    {f.label === 'Conflicts' && !change.hasConflicts && conflict && forecast && (
                      <>
                        ; would conflict with {forecast.branch} in{' '}
                        <span className="host-change-facts-branches">{conflict.files.join(', ')}</span>
                      </>
                    )}
                  </span>
                </dd>
              </Fragment>
            )
          })
        ) : (
          <>
            <dt>Reviews</dt>
            <dd className="host-change-facts-reviews">
              {reviews.length > 0
                ? reviews.map((group) => (
                    <span key={group.label}>
                      {group.label}: {group.names.join(', ')}
                    </span>
                  ))
                : 'None requested'}
            </dd>
            <dt>Checks</dt>
            <dd className="host-change-facts-checks">
              {mark ? (
                <>
                  <CIMark repoPath={repoPath} mark={mark} />
                  {ciLabel(mark.status)}
                </>
              ) : change.ci ? (
                ciLabel(change.ci)
              ) : (
                'None reported'
              )}
            </dd>
            <dt>Conflicts</dt>
            <dd>
              {change.hasConflicts ? (
                `Conflicts with ${change.targetBranch} on ${list.host}.`
              ) : conflict && forecast ? (
                <>
                  Would conflict with {forecast.branch} in{' '}
                  <span className="host-change-facts-branches">{conflict.files.join(', ')}</span>
                </>
              ) : (
                'None reported'
              )}
            </dd>
          </>
        )}
        {change.labels?.length > 0 && (
          <>
            <dt>Labels</dt>
            <dd>{change.labels.join(', ')}</dd>
          </>
        )}
      </dl>
      <div className="host-change-actions">
        <button type="button" className="host-change-button" onClick={checkOut} disabled={checkout.running}>
          <GitBranch size={14} strokeWidth={1.75} />
          {checkout.running ? 'Checking out…' : 'Check out locally'}
        </button>
        {change.state === 'open' && !merged && reviewActions && (
          <button
            type="button"
            className={primary === 'review' ? 'host-changes-new' : 'host-change-button'}
            onClick={() => setReviewing(true)}
          >
            <ClipboardCheck size={14} strokeWidth={1.75} />
            Review…
          </button>
        )}
        {change.state === 'open' && !merged && (
          <button
            type="button"
            className={primary === 'merge' ? 'host-changes-new' : 'host-change-button'}
            onClick={() => setMerging(true)}
          >
            <GitMerge size={14} strokeWidth={1.75} />
            Merge…
          </button>
        )}
        <button
          type="button"
          className="host-change-button"
          onClick={() => PlatformService.OpenWebPage(change.webUrl).catch(ignoreDecorativeFailure)}
        >
          <ExternalLink size={14} strokeWidth={1.75} />
          Open on {list.host}
        </button>
        {checkout.running && (
          <button type="button" className="host-change-button" onClick={checkout.cancel}>
            Cancel
          </button>
        )}
      </div>
      {checkedOut && (
        <p className="host-changes-notice" role="status">
          {checkedOut}
        </p>
      )}
      {checkout.error && <p className="host-changes-error">Could not check out: {checkout.error}</p>}
      {merged && (
        <p className="host-changes-notice" role="status">
          {merged}
        </p>
      )}
      {reviewed && !merged && (
        <p className="host-changes-notice" role="status">
          Review submitted.
        </p>
      )}
      {reviewing && (
        <ReviewDialog
          repoPath={repoPath}
          number={change.number}
          prefix={list.prefix}
          onSubmitted={() => {
            setReviewing(false)
            setReviewed(true)
            onReviewed?.()
          }}
          onClose={() => setReviewing(false)}
        />
      )}
      {merging && (
        <MergeChangeDialog
          repoPath={repoPath}
          change={change}
          prefix={list.prefix}
          host={list.host}
          facts={readiness?.facts ?? []}
          onMerged={(result) => {
            setMerging(false)
            setMerged(
              result.localBranch
                ? `Merged ${list.prefix}${change.number}. ${result.localBranch} can be cleaned up.`
                : `Merged ${list.prefix}${change.number}.`,
            )
            onMerged?.(result)
          }}
          onClose={() => setMerging(false)}
        />
      )}
      <nav className="host-change-tabs" aria-label="Detail">
        {tabs.map((t) => (
          <Tab
            key={t.id}
            active={tab === t.id}
            onClick={() => setTab(t.id)}
            label={t.count !== undefined ? `${t.label} ${t.count}` : t.label}
          >
            {t.label}
            {t.count !== undefined && <span className="host-change-tab-count">{t.count}</span>}
          </Tab>
        ))}
      </nav>
      <div className="host-change-tab-body">
        {tab === 'overview' &&
          details &&
          (details.descriptionHtml ? (
            <HostMarkdown html={details.descriptionHtml} className="host-change-description" />
          ) : (
            <p className="host-changes-empty">No description.</p>
          ))}
        {tab === 'checks' &&
          (mark ? (
            <PipelineJobList repoPath={repoPath} pipelineId={mark.pipelineId} onLoaded={countFor.checks} />
          ) : fetchedMarks ? (
            <p className="host-changes-empty">No pipeline for {change.headSha.slice(0, 7)}.</p>
          ) : (
            <p className="host-changes-empty">Looking for a pipeline…</p>
          ))}
        {tab === 'threads' && <ThreadsTab repoPath={repoPath} number={change.number} onLoaded={countFor.threads} />}
        {tab === 'files' && <FilesTab repoPath={repoPath} number={change.number} onLoaded={countFor.files} />}
        {tab === 'commits' && <CommitsTab repoPath={repoPath} number={change.number} onLoaded={countFor.commits} />}
      </div>
    </article>
  )
}

interface TabProps {
  repoPath: string
  number: number
  onLoaded: (count: number) => void
}

function ThreadsTab({ repoPath, number, onLoaded }: TabProps) {
  const actions = useReviewActions(repoPath, number)
  const { version } = useReviewDraft()
  const { data, error } = useAsyncData(() => HostService.ChangeThreads(repoPath, number), [repoPath, number], {
    refreshKey: version,
    keepData: true,
  })
  useEffect(() => {
    if (data) onLoaded(data.threads?.length ?? 0)
  }, [data, onLoaded])
  if (error) return <p className="host-changes-error">{error}</p>
  if (!data) return <p className="host-changes-empty">Loading…</p>
  if (!data.threads?.length) return <p className="host-changes-empty">No threads.</p>
  return (
    <div className="host-change-threads">
      {data.threads.map((thread) => (
        <ReviewThreadBlock
          key={thread.id}
          thread={thread}
          where={thread.path ? `${thread.path}${thread.line ? `:${thread.line}` : ''}` : undefined}
          actions={actions?.threadActions}
        />
      ))}
    </div>
  )
}

const FILE_LETTER: Record<string, string> = { added: 'A', deleted: 'D', renamed: 'R', modified: 'M' }

function FilesTab({ repoPath, number, onLoaded }: TabProps) {
  const { data, error } = useAsyncData(() => HostService.ChangeFiles(repoPath, number), [repoPath, number])
  useEffect(() => {
    if (data) onLoaded(data.length)
  }, [data, onLoaded])
  if (error) return <p className="host-changes-error">{error}</p>
  if (!data) return <p className="host-changes-empty">Loading…</p>
  if (data.length === 0) return <p className="host-changes-empty">No files.</p>
  return (
    <ul className="host-change-files" aria-label="Files">
      {data.map((file) => (
        <li key={file.path} className="host-change-file">
          <StatusBadge status={FILE_LETTER[file.status] ?? 'M'} />
          <span className="host-change-file-path">
            <PathText path={file.path} />
            {file.oldPath && <span className="host-change-file-from"> from {file.oldPath}</span>}
          </span>
          <DiffStat added={file.additions} removed={file.deletions} binary={false} />
        </li>
      ))}
    </ul>
  )
}

function CommitsTab({ repoPath, number, onLoaded }: TabProps) {
  const { data, error } = useAsyncData(() => HostService.ChangeCommits(repoPath, number), [repoPath, number])
  useEffect(() => {
    if (data) onLoaded(data.length)
  }, [data, onLoaded])
  if (error) return <p className="host-changes-error">{error}</p>
  if (!data) return <p className="host-changes-empty">Loading…</p>
  if (data.length === 0) return <p className="host-changes-empty">No commits.</p>
  return (
    <ul className="host-change-commits" aria-label={plural(data.length, 'commit')}>
      {data.map((commit) => (
        <li key={commit.sha} className="host-change-commit">
          <span className="host-change-commit-sha">{commit.sha.slice(0, 7)}</span>
          <span className="host-change-commit-subject">{commit.subject}</span>
          <span className="host-change-commit-meta">
            {commit.author} · {relativeTime(new Date(commit.date))}
          </span>
        </li>
      ))}
    </ul>
  )
}

export function changesLabelFor(provider: string): string {
  return provider === 'gitlab' ? 'Merge requests' : 'Pull requests'
}

export type { ChangeList }
export default HostChangesView
