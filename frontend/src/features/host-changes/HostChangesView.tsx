import { useEffect, useState } from 'react'
import { ExternalLink, GitBranch, RefreshCw } from 'lucide-react'
import {
  HostService,
  IdentityService,
  PlatformService,
  type ChangeList,
  type IdentityInfo,
} from '@current-client-bindings/app'
import type { Change } from '@current-client-bindings/core/hostapi'
import ChangeMarker from '../../components/git/ChangeMarker'
import HostMarkdown from '../../components/git/HostMarkdown'
import IdentityBadge from '../../components/git/IdentityBadge'
import Switch from '../../components/forms/Switch'
import { errorMessage, ignoreDecorativeFailure } from '../../lib/errors'
import { relativeTime } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import { useCancellableOperation } from '../../lib/cancellableOperation'
import { useClockTick } from '../../lib/useClockTick'
import './HostChangesView.scss'

interface HostChangesViewProps {
  repoPath: string
  label: string
  // A change to select, from the sidebar's Details; the list's own selection otherwise.
  selectedNumber?: number | null
  onBranchChanged: () => void
}

// The merge request / pull request panel: open changes from the last
// refresh, details for one, and checking it out locally. Opening the panel
// for the first time, and its refresh button, are the only things here
// that contact the host.
function HostChangesView({ repoPath, label, selectedNumber = null, onBranchChanged }: HostChangesViewProps) {
  const {
    data: list,
    error,
    reload,
  } = useAsyncData(() => HostService.Changes(repoPath), [repoPath], { keepData: true })
  const [refreshing, setRefreshing] = useState(false)
  const [refreshError, setRefreshError] = useState<string | null>(null)
  const [onlyMine, setOnlyMine] = useState(false)
  const [selected, setSelected] = useState<number | null>(null)
  useEffect(() => {
    if (selectedNumber !== null) setSelected(selectedNumber)
  }, [selectedNumber])
  useClockTick()

  const refresh = () => {
    setRefreshing(true)
    setRefreshError(null)
    HostService.Refresh(repoPath)
      .then(reload)
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

  const changes = (list.changes ?? []).filter((c) => !onlyMine || c.author.username === list.me)
  const current = changes.find((c) => c.number === selected) ?? changes[0] ?? null

  return (
    <section className="host-changes" aria-label={label}>
      <header className="host-changes-header">
        <h2 className="host-changes-title">{label}</h2>
        <span className="host-changes-fetched">
          {list.fetchedAt ? `fetched ${relativeTime(new Date(list.fetchedAt))}` : 'not fetched yet'}
          {list.truncated && ' · list cut at the page cap'}
        </span>
        {list.me && <Switch checked={onlyMine} onChange={setOnlyMine} label="Authored by me" />}
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
      </header>
      {(error || refreshError) && <p className="host-changes-error">{refreshError ?? error}</p>}
      <div className="host-changes-body">
        <ul className="host-changes-list" aria-label={`${label} list`}>
          {changes.length === 0 && (
            <li className="host-changes-empty">{list.fetchedAt ? 'None open.' : 'Refresh to load.'}</li>
          )}
          {changes.map((change) => (
            <li key={change.number}>
              <ChangeRow
                change={change}
                prefix={list.prefix}
                selected={current?.number === change.number}
                onSelect={() => setSelected(change.number)}
              />
            </li>
          ))}
        </ul>
        {current && (
          <ChangeDetail
            key={current.number}
            repoPath={repoPath}
            change={current}
            prefix={list.prefix}
            onBranchChanged={onBranchChanged}
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
}

function ChangeRow({ change, prefix, selected, onSelect }: ChangeRowProps) {
  const identity = useIdentity(change.author.name)
  return (
    <button
      type="button"
      className={selected ? 'host-change-row host-change-row-selected' : 'host-change-row'}
      onClick={onSelect}
    >
      <ChangeMarker
        change={{
          number: change.number,
          prefix,
          state: change.state,
          draft: change.draft,
          ci: change.ci,
          title: '',
          author: '',
          webUrl: '',
          sourceBranch: '',
          targetBranch: '',
          unresolvedThreads: 0,
          updatedAt: '',
        }}
      />
      <span className="host-change-row-main">
        <span className="host-change-row-title">{change.title}</span>
        <span className="host-change-row-branches">
          {change.sourceProject ? `${change.sourceProject}:` : ''}
          {change.sourceBranch} → {change.targetBranch}
        </span>
      </span>
      {change.labels?.length > 0 && (
        <span className="host-change-row-labels">
          {change.labels.map((l) => (
            <span key={l} className="host-change-label">
              {l}
            </span>
          ))}
        </span>
      )}
      {identity && <IdentityBadge identity={identity} name={change.author.name} email="" size={18} />}
      <span className="host-change-row-time">{relativeTime(new Date(change.updatedAt))}</span>
    </button>
  )
}

interface ChangeDetailProps {
  repoPath: string
  change: Change
  prefix: string
  onBranchChanged: () => void
}

function ChangeDetail({ repoPath, change, prefix, onBranchChanged }: ChangeDetailProps) {
  const { data: details } = useAsyncData(
    () => HostService.ChangeDetails(repoPath, change.number),
    [repoPath, change.number],
  )
  const checkout = useCancellableOperation(`${repoPath}:${change.number}`)
  const [checkedOut, setCheckedOut] = useState<string | null>(null)
  const state = change.draft ? 'draft' : change.state

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

  return (
    <article className="host-change-detail" aria-label={`${prefix}${change.number}`}>
      <header className="host-change-detail-header">
        <span className="host-change-detail-number">
          {prefix}
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
        {change.labels?.length > 0 && (
          <>
            <dt>Labels</dt>
            <dd>{change.labels.join(', ')}</dd>
          </>
        )}
        {change.ci && (
          <>
            <dt>CI</dt>
            <dd>{change.ci}</dd>
          </>
        )}
        {change.hasConflicts && (
          <>
            <dt>Conflicts</dt>
            <dd>Has conflicts with {change.targetBranch} on the host.</dd>
          </>
        )}
      </dl>
      <div className="host-change-actions">
        <button type="button" className="host-change-button" onClick={checkOut} disabled={checkout.running}>
          <GitBranch size={14} strokeWidth={1.75} />
          {checkout.running ? 'Checking out…' : 'Check out locally'}
        </button>
        <button
          type="button"
          className="host-change-button"
          onClick={() => PlatformService.OpenWebPage(change.webUrl).catch(ignoreDecorativeFailure)}
        >
          <ExternalLink size={14} strokeWidth={1.75} />
          Open on host
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
      {details &&
        (details.descriptionHtml ? (
          <HostMarkdown html={details.descriptionHtml} className="host-change-description" />
        ) : (
          <p className="host-changes-empty">No description.</p>
        ))}
    </article>
  )
}

function useIdentity(name: string): IdentityInfo | null {
  const { data } = useAsyncData(() => IdentityService.GetIdentities([{ name, email: '' }]), [name])
  return data?.[0] ?? null
}

export function changesLabelFor(provider: string): string {
  return provider === 'gitlab' ? 'Merge requests' : 'Pull requests'
}

export type { ChangeList }
export default HostChangesView
