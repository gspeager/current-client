import { useMemo, useState } from 'react'
import { useLaneColors } from '../../lib/laneColor'
import { type HostIdentity, hostUserName, useNameIdentities } from '../../lib/useNameIdentities'
import IdentityBadge from '../../components/git/IdentityBadge'
import { useBranchStatus } from '../../lib/useBranchStatus'
import {
  ArrowRight,
  BookOpen,
  CircleCheck,
  CircleHelp,
  ClipboardCheck,
  GitBranch,
  GitMerge,
  Info,
  TriangleAlert,
  type LucideIcon,
} from 'lucide-react'
import { HostService, ReadThroughService, type ChangeList, type MergeResult } from '@current-client-bindings/app'
import type { Change } from '@current-client-bindings/core/hostapi'
import { Fact, FactList } from '../../components/chrome/FactList'
import LabelChip from '../../components/git/LabelChip'
import CIMark, { PipelineJobList } from './CIMark'
import { ciLabel } from '../../components/git/CIIcon'
import { EnvironmentReach } from '../../components/git/EnvironmentMarker'
import HostMarkdown from '../../components/git/HostMarkdown'
import ReferenceText from '../../components/git/ReferenceText'
import { useHostLinks } from '../../lib/useHostLinks'
import { useReferenceCards } from '../../lib/useReferenceCards'
import { reviewGroups, reviewRequestedOf } from '../../components/git/ReviewSummary'
import Tab from '../../components/controls/Tab'
import ReviewDialog from '../review/ReviewDialog'
import { useReviewActions } from '../review/useReviewActions'
import MergeChangeDialog from './MergeChangeDialog'
import { useOpenReadThrough } from '../../lib/readThrough'
import { openWebPage } from '../../lib/hostPages'
import { relativeTime } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import { useCancellableOperation } from '../../lib/cancellableOperation'
import { hostLogo } from '../../components/git/hostLogos'
import ChangeStateLabel from './ChangeStateLabel'
import { shortSha } from '../../lib/text'
import ChangeReviewers from './ChangeReviewers'
import CommitsTab from './CommitsTab'
import FilesTab from './FilesTab'
import MergeVerdict from './MergeVerdict'
import OverviewCommits from './OverviewCommits'
import ThreadsTab from './ThreadsTab'

type DetailTab = 'overview' | 'checks' | 'threads' | 'files' | 'commits'

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
  const { branchColor } = useLaneColors()
  const currentBranch = useBranchStatus(repoPath, 0)?.current
  const authorIdentity = useNameIdentities([change.author.name], repoPath).get(change.author.name)
  const HostIcon = hostLogo(hostLinks?.style)
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
  const openReadThrough = useOpenReadThrough()
  // Only when the head is here; nothing is fetched for it.
  const { data: readable } = useAsyncData(
    () =>
      ReadThroughService.OfChange(repoPath, change.sourceBranch, change.targetBranch, change.headSha).catch(() => null),
    [repoPath, change.sourceBranch, change.targetBranch, change.headSha],
    { refreshKey: checkedOut },
  )
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
  const protection = readiness?.facts.find((f) => f.label === 'Protection')
  // What stands in the way first; protection is a standing fact, not a check.
  const tileFacts = (readiness?.facts ?? [])
    .filter((f) => f.label !== 'Protection')
    .sort((a, b) => Number(b.level === 'warn') - Number(a.level === 'warn'))
  const reviews = reviewGroups(change.reviews, change.state === 'open')
  const reviewerIdentities = useNameIdentities(
    (change.reviews ?? []).map((r) => hostUserName(r.reviewer)),
    repoPath,
  )
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
    <article className="host-change-detail" data-pane="change-detail" aria-label={`${list.prefix}${change.number}`}>
      <header className="host-change-detail-header">
        <span className="host-change-detail-number">
          {list.prefix}
          {change.number}
        </span>
        <ChangeStateLabel state={state} />
        <h3 className="host-change-detail-title">{change.title}</h3>
        <div className="host-change-actions">
          <button type="button" className="host-change-button" onClick={checkOut} disabled={checkout.running}>
            <GitBranch size={14} strokeWidth={1.75} />
            {checkout.running ? 'Checking out…' : 'Check out locally'}
          </button>
          {readable?.ref && (
            <button
              type="button"
              className="host-change-button"
              onClick={() =>
                openReadThrough({
                  ...readable,
                  name: readable.ref.startsWith('refs/current-client/') ? `${list.prefix}${change.number}` : undefined,
                })
              }
            >
              <BookOpen size={14} strokeWidth={1.75} />
              Read through
            </button>
          )}
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
            className="host-change-button host-change-button-quiet"
            onClick={() => openWebPage(change.webUrl)}
          >
            <HostIcon size={14} strokeWidth={1.75} />
            Open on host
          </button>
          {checkout.running && (
            <button type="button" className="host-change-button" onClick={checkout.cancel}>
              Cancel
            </button>
          )}
        </div>
      </header>
      <p className="host-change-subtitle">
        <span className="host-change-author">
          {authorIdentity && <IdentityBadge identity={authorIdentity} name={change.author.name} email="" size={20} />}
          {authorIdentity?.displayName ?? change.author.name}
        </span>
        <span className="host-change-flow" aria-label={`${change.sourceBranch} into ${change.targetBranch}`}>
          <span
            className="host-change-flow-dot"
            style={{ background: branchColor(change.sourceBranch, change.sourceBranch === currentBranch) }}
          />
          {change.sourceProject ? `${change.sourceProject}:` : ''}
          {change.sourceBranch}
          <ArrowRight size={12} strokeWidth={1.75} aria-hidden="true" />
          <span className="host-change-flow-dot" style={{ background: branchColor(change.targetBranch, false) }} />
          {change.targetBranch}
        </span>
        {closes.length > 0 && (
          <span className="host-change-facts-closes">
            Closes <ReferenceText text={closes.join(', ')} links={hostLinks} cards={closesCards} />
          </span>
        )}
        {(details?.environments?.length ?? 0) > 0 && <EnvironmentReach reach={details!.environments} />}
        {protection && (
          <span className="host-change-protection" title={protection.value}>
            {/^protected/i.test(protection.value) ? `${change.targetBranch} is protected` : protection.value}
          </span>
        )}
        {(change.labels?.length ?? 0) > 0 && (
          <span className="host-change-labels">
            {change.labels?.map((label) => (
              <LabelChip key={label} label={label} />
            ))}
          </span>
        )}
      </p>
      {change.state === 'open' && !merged && tileFacts.length > 0 && <MergeVerdict facts={tileFacts} />}
      <dl className="host-change-facts">
        {readiness && readiness.facts.length > 0 ? (
          tileFacts.map((f) => {
            const Icon = FACT_ICONS[f.level] ?? CircleHelp
            return (
              <div key={f.label} className={`host-change-tile host-change-tile-${f.level}`}>
                <dt>{f.label}</dt>
                <dd className={`host-change-fact host-change-fact-${f.level}`}>
                  {f.label === 'Checks' && mark ? (
                    <CIMark repoPath={repoPath} mark={mark} />
                  ) : (
                    <Icon size={14} strokeWidth={1.75} aria-hidden="true" />
                  )}
                  <span>
                    {f.label === 'Reviews' ? withDisplayNames(f.value, reviewerIdentities) : f.value}
                    {f.label === 'Conflicts' && !change.hasConflicts && conflict && forecast && (
                      <>
                        ; would conflict with {forecast.branch} in{' '}
                        <span className="host-change-facts-branches">{conflict.files.join(', ')}</span>
                      </>
                    )}
                  </span>
                </dd>
              </div>
            )
          })
        ) : (
          <>
            <div className="host-change-tile">
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
            </div>
            <div className="host-change-tile">
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
            </div>
            <div className="host-change-tile">
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
            </div>
          </>
        )}
      </dl>
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
        {tab === 'overview' && details && (
          <div className="host-change-overview">
            <div className="host-change-overview-main">
              {details.descriptionHtml ? (
                <HostMarkdown html={details.descriptionHtml} className="host-change-description" />
              ) : (
                <p className="host-changes-empty">No description.</p>
              )}
              <OverviewCommits
                repoPath={repoPath}
                target={change.targetBranch}
                source={change.sourceBranch}
                current={change.sourceBranch === currentBranch}
              />
            </div>
            <aside className="host-change-overview-aside" aria-label="People and timing">
              <ChangeReviewers repoPath={repoPath} reviews={change.reviews} />
              <FactList>
                <Fact label="Updated">{relativeTime(new Date(change.updatedAt))}</Fact>
              </FactList>
            </aside>
          </div>
        )}
        {tab === 'checks' &&
          (mark ? (
            <PipelineJobList repoPath={repoPath} pipelineId={mark.pipelineId} onLoaded={countFor.checks} />
          ) : fetchedMarks ? (
            <p className="host-changes-empty">No pipeline for {shortSha(change.headSha)}.</p>
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

// A readiness fact names reviewers as the host does; the pane uses commit names.
function withDisplayNames(text: string, identities: Map<string, HostIdentity>): string {
  return [...identities].reduce(
    (out, [name, identity]) => out.replace(new RegExp(`\\b${name}\\b`, 'g'), identity.displayName),
    text,
  )
}

export default ChangeDetail
