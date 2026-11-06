import { useState } from 'react'
import { GitPullRequest, PanelRight, TriangleAlert, Upload } from 'lucide-react'
import { HostService, PlatformService } from '@current-client-bindings/app'
import ChangeMarker from '../../components/git/ChangeMarker'
import CIMark, { ciLabel } from '../../components/git/CIMark'
import { createChangeFor } from '../../lib/createChange'
import { ignoreDecorativeFailure } from '../../lib/errors'
import { useAsyncData } from '../../lib/useAsyncData'
import { useReviewThreads } from '../../lib/useReviewThreads'
import './CurrentChangeSection.scss'
import { useHostLogo } from '../../components/git/hostLogos'

interface CurrentChangeSectionProps {
  repoPath: string
  // Changes when the repository moves or a host list is refreshed.
  refreshKey: unknown
  onOpenChange?: (number: number) => void
  // Opens a file in Changes, for the first unresolved thread.
  onOpenFile?: (path: string) => void
  // Opens the in-app create form; without it the host's form is used.
  onCreateChange?: (branch: string) => void
  // Pushes the branch with an upstream, then opens the create form.
  onPushToCreate?: () => void
}

// The checked-out branch's merge request / pull request, read from the
// cache: the summary when it has one, else one row that leads to creating it.
function CurrentChangeSection({
  repoPath,
  refreshKey,
  onOpenChange,
  onOpenFile,
  onCreateChange,
  onPushToCreate,
}: CurrentChangeSectionProps) {
  const [checks, setChecks] = useState(0)
  const [checking, setChecking] = useState(false)
  const { data } = useAsyncData(() => HostService.CurrentChange(repoPath), [repoPath], {
    refreshKey: `${refreshKey}:${checks}`,
    keepData: true,
  })
  const HostIcon = useHostLogo(repoPath)
  const { data: forecast } = useAsyncData(() => HostService.Forecast(repoPath), [repoPath], {
    refreshKey,
    keepData: true,
  })
  const reviewThreads = useReviewThreads(repoPath, refreshKey)
  const firstThreadPath = Object.keys(reviewThreads.counts).find((path) => (reviewThreads.counts[path] ?? 0) > 0)
  if (!data?.enabled) return null

  const change = data.change
  const conflicts = forecast?.enabled ? (forecast.conflicts ?? []) : []
  const kindWord = data.kind || 'pull request'
  const kind = kindWord[0].toUpperCase() + kindWord.slice(1)
  const createChange = () => createChangeFor(repoPath, data.branch, onCreateChange)

  const forecastBlock = forecast?.enabled && conflicts.length > 0 && (
    <div className="current-change-forecast" aria-label="Conflict forecast">
      <ul className="current-change-conflicts">
        {conflicts.map((c) => (
          <li key={c.number}>
            <TriangleAlert size={12} strokeWidth={1.75} className="current-change-conflict-icon" />
            <button
              type="button"
              className="current-change-conflict"
              onClick={() =>
                onOpenChange
                  ? onOpenChange(c.number)
                  : PlatformService.OpenWebPage(c.webUrl).catch(ignoreDecorativeFailure)
              }
              title={c.title}
            >
              Conflicts with {c.prefix}
              {c.number}
              {c.author ? ` by ${c.author}` : ''} in{' '}
              <span className="current-change-conflict-files">{c.files.join(', ')}</span>
            </button>
          </li>
        ))}
      </ul>
      {forecast.checked < forecast.total && (
        <span className="current-change-forecast-cap">
          {forecast.checked} of {forecast.total} checked
        </span>
      )}
    </div>
  )

  if (!change) {
    if (!data.proposable && conflicts.length === 0) return null
    // Without an upstream there is nothing on the host to open a change from.
    const row = !data.proposable ? null : !data.checked && data.upstream ? (
      <button
        type="button"
        className="current-change-row"
        disabled={checking}
        onClick={() => {
          setChecking(true)
          void HostService.Refresh(repoPath)
            .catch(ignoreDecorativeFailure)
            .finally(() => {
              setChecking(false)
              setChecks((n) => n + 1)
            })
        }}
      >
        <GitPullRequest size={14} strokeWidth={1.75} className="current-change-row-icon" />
        <span className="current-change-row-text">
          {checking ? (
            'Checking…'
          ) : (
            <>
              <span className="current-change-row-action">Check for a {kindWord}</span>
            </>
          )}
        </span>
      </button>
    ) : data.upstream ? (
      <button type="button" className="current-change-row" onClick={() => void createChange()}>
        <GitPullRequest size={14} strokeWidth={1.75} className="current-change-row-icon" />
        <span className="current-change-row-text">
          No {kindWord} <span className="current-change-row-sep">·</span>{' '}
          <span className="current-change-row-action">Create…</span>
        </span>
      </button>
    ) : (
      <button type="button" className="current-change-row" onClick={onPushToCreate} disabled={!onPushToCreate}>
        <Upload size={14} strokeWidth={1.75} className="current-change-row-icon" />
        <span className="current-change-row-text">Push to create a {kindWord}</span>
      </button>
    )
    const hint = !data.proposable
      ? null
      : !data.checked && data.upstream
        ? 'Not checked yet'
        : data.upstream
          ? `None open from ${data.branch}`
          : 'This branch has no upstream'
    return (
      <section className="current-change-slot" aria-label={`${kind} for ${data.branch}`}>
        {row}
        {hint && <p className="current-change-slot-hint">{hint}</p>}
        {forecastBlock}
      </section>
    )
  }

  const state = change.draft ? 'draft' : change.state
  return (
    <section className="current-change" aria-label={`${kind} for ${data.branch}`}>
      <div className="current-change-title">
        <ChangeMarker change={change} />
        <span className="current-change-title-text" title={change.title}>
          {change.title}
        </span>
        {onOpenChange && (
          <button
            type="button"
            className="current-change-button"
            aria-label="Details"
            title="Details"
            onClick={() => onOpenChange(change.number)}
          >
            <PanelRight size={14} strokeWidth={1.75} />
          </button>
        )}
        <button
          type="button"
          className="current-change-button"
          aria-label="Open on host"
          title="Open on host"
          onClick={() => PlatformService.OpenWebPage(change.webUrl).catch(ignoreDecorativeFailure)}
          disabled={!change.webUrl}
        >
          <HostIcon size={14} strokeWidth={1.75} />
        </button>
      </div>
      <div className="current-change-facts">
        <span className="current-change-target">→ {change.targetBranch}</span>
        {state && state !== 'open' && (
          <span className={`current-change-state current-change-state-${state}`}>{state}</span>
        )}
        {data.ci && (
          <span className="current-change-ci">
            <CIMark repoPath={repoPath} mark={data.ci} />
            {data.failedJobs?.length ? `failed: ${data.failedJobs.join(', ')}` : ciLabel(data.ci.status)}
          </span>
        )}
        {change.unresolvedThreads > 0 &&
          (onOpenFile && firstThreadPath ? (
            <button
              type="button"
              className="current-change-threads current-change-threads-link"
              onClick={() => onOpenFile(firstThreadPath)}
            >
              {change.unresolvedThreads} unresolved
            </button>
          ) : (
            <span className="current-change-threads">{change.unresolvedThreads} unresolved</span>
          ))}
      </div>
      {forecastBlock}
    </section>
  )
}

export default CurrentChangeSection
