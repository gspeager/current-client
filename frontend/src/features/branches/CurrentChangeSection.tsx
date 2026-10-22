import { ExternalLink, PanelRight } from 'lucide-react'
import { HostService, PlatformService } from '@current-client-bindings/app'
import ChangeMarker from '../../components/git/ChangeMarker'
import CIMark from '../../components/git/CIMark'
import { ignoreDecorativeFailure } from '../../lib/errors'
import { useAsyncData } from '../../lib/useAsyncData'
import './CurrentChangeSection.scss'

interface CurrentChangeSectionProps {
  repoPath: string
  refreshKey: number
  onOpenChange?: (number: number) => void
}

// The checked-out branch's merge request / pull request, read from the
// cache: absent while the host is off or the branch has none known.
function CurrentChangeSection({ repoPath, refreshKey, onOpenChange }: CurrentChangeSectionProps) {
  const { data } = useAsyncData(() => HostService.CurrentChange(repoPath), [repoPath], { refreshKey, keepData: true })
  const change = data?.enabled ? data.change : null
  if (!data || !change) return null

  const kind = change.prefix === '!' ? 'Merge request' : 'Pull request'
  const state = change.draft ? 'draft' : change.state
  return (
    <section className="current-change" aria-label={`${kind} for ${data.branch}`}>
      <div className="current-change-head">
        <span className="branch-sidebar-label">{kind}</span>
        <span className={`current-change-state current-change-state-${state}`}>{state}</span>
      </div>
      <div className="current-change-title">
        <ChangeMarker change={change} />
        <span className="current-change-title-text">{change.title}</span>
      </div>
      <div className="current-change-facts">
        <span className="current-change-target">→ {change.targetBranch}</span>
        {data.ci && (
          <span className="current-change-ci">
            <CIMark repoPath={repoPath} mark={data.ci} />
            {data.failedJobs?.length ? `failed: ${data.failedJobs.join(', ')}` : ciLabel(data.ci.status)}
          </span>
        )}
        {change.unresolvedThreads > 0 && (
          <span className="current-change-threads">{change.unresolvedThreads} unresolved</span>
        )}
      </div>
      <div className="current-change-actions">
        {onOpenChange && (
          <button type="button" className="current-change-button" onClick={() => onOpenChange(change.number)}>
            <PanelRight size={14} strokeWidth={1.75} />
            Details
          </button>
        )}
        <button
          type="button"
          className="current-change-button"
          onClick={() => PlatformService.OpenWebPage(change.webUrl).catch(ignoreDecorativeFailure)}
          disabled={!change.webUrl}
        >
          <ExternalLink size={14} strokeWidth={1.75} />
          Open on host
        </button>
      </div>
    </section>
  )
}

function ciLabel(status: string): string {
  const labels: Record<string, string> = {
    success: 'CI passed',
    failed: 'CI failed',
    running: 'CI running',
    pending: 'CI pending',
    canceled: 'CI canceled',
    skipped: 'CI skipped',
  }
  return labels[status] ?? `CI ${status}`
}

export default CurrentChangeSection
