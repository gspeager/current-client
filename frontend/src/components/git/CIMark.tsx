import { useEffect, useState, type MouseEvent } from 'react'
import {
  CircleCheck,
  CircleDashed,
  CircleMinus,
  CircleSlash,
  CircleX,
  Clock,
  ExternalLink,
  type LucideIcon,
} from 'lucide-react'
import { HostService, PlatformService, type CIMark as CIMarkInfo } from '@current-client-bindings/app'
import type { Job } from '@current-client-bindings/core/hostapi'
import HoverCard from '../chrome/HoverCard'
import Modal from '../chrome/Modal'
import { errorMessage, ignoreDecorativeFailure } from '../../lib/errors'
import { useAsyncData } from '../../lib/useAsyncData'
import './CIMark.scss'

// One map for every place that shows a CI result, so the mark, the PR
// marker and the job rows always agree. The running icon doesn't spin.
const STATUSES: Record<string, { icon: LucideIcon; label: string }> = {
  success: { icon: CircleCheck, label: 'CI passed' },
  failed: { icon: CircleX, label: 'CI failed' },
  running: { icon: CircleDashed, label: 'CI running' },
  pending: { icon: Clock, label: 'CI pending' },
  canceled: { icon: CircleSlash, label: 'CI canceled' },
  skipped: { icon: CircleMinus, label: 'CI skipped' },
}

export function ciLabel(status: string): string {
  return STATUSES[status]?.label ?? `CI ${status}`
}

export function CIIcon({ status, size = 14 }: { status: string; size?: number }) {
  const Icon = STATUSES[status]?.icon ?? CircleDashed
  return <Icon className={`ci-icon ci-icon-${status}`} size={size} strokeWidth={1.75} aria-hidden="true" />
}

interface CIMarkProps {
  repoPath: string
  mark: CIMarkInfo | undefined
}

// The pipeline result for a commit or branch tip. Clicking lists the jobs,
// fetched then, with Open on host.
function CIMark({ repoPath, mark }: CIMarkProps) {
  const [open, setOpen] = useState(false)
  if (!mark?.status) return null
  const label = ciLabel(mark.status)
  const onClick = (e: MouseEvent) => {
    e.stopPropagation()
    setOpen(true)
  }
  return (
    <>
      <HoverCard card={<CICard repoPath={repoPath} mark={mark} label={label} />}>
        <button type="button" className="ci-mark" aria-label={label} onClick={onClick}>
          <CIIcon status={mark.status} />
        </button>
      </HoverCard>
      {open && <PipelineJobs repoPath={repoPath} mark={mark} onClose={() => setOpen(false)} />}
    </>
  )
}

// Reads the jobs a click already fetched; hovering never fetches.
function CICard({ repoPath, mark, label }: { repoPath: string; mark: CIMarkInfo; label: string }) {
  const { data: jobs } = useAsyncData(
    () => HostService.CachedPipelineJobs(repoPath, mark.pipelineId),
    [repoPath, mark.pipelineId],
  )
  const failed = (jobs ?? []).filter((job) => job.status === 'failed')
  const total = (jobs ?? []).reduce((sum, job) => sum + job.duration, 0)
  return (
    <>
      <p className="hover-card-title">{label}</p>
      {failed.length > 0 && (
        <p className="hover-card-line hover-card-data">{failed.map((job) => job.name).join(', ')}</p>
      )}
      <p className="hover-card-line hover-card-muted">
        {jobs?.length ? `${jobs.length} jobs${total > 0 ? ` · ${formatDuration(total)}` : ''}` : 'Click for the jobs.'}
      </p>
    </>
  )
}

// The jobs of one pipeline, fetched when first shown. onLoaded reports the
// count for a caller that shows it elsewhere.
export function PipelineJobList({
  repoPath,
  pipelineId,
  onLoaded,
}: {
  repoPath: string
  pipelineId: number
  onLoaded?: (count: number) => void
}) {
  const { data: jobs, error } = useAsyncData(
    () => HostService.PipelineJobs(repoPath, pipelineId),
    [repoPath, pipelineId],
  )
  useEffect(() => {
    if (jobs) onLoaded?.(jobs.length)
  }, [jobs, onLoaded])
  return (
    <>
      {error && <p className="ci-jobs-error">Could not load the checks: {errorMessage(error)}</p>}
      {jobs === null && !error && <p className="ci-jobs-hint">Loading jobs…</p>}
      {jobs?.length === 0 && <p className="ci-jobs-hint">No jobs.</p>}
      {jobs && jobs.length > 0 && (
        <ul className="ci-jobs-list" aria-label="Jobs">
          {jobs.map((job) => (
            <JobRow key={job.id} job={job} repoPath={repoPath} />
          ))}
        </ul>
      )}
    </>
  )
}

function PipelineJobs({ repoPath, mark, onClose }: { repoPath: string; mark: CIMarkInfo; onClose: () => void }) {
  return (
    <Modal title={ciLabel(mark.status)} onClose={onClose} className="ci-jobs-panel">
      <div className="ci-jobs">
        <PipelineJobList repoPath={repoPath} pipelineId={mark.pipelineId} />
        <div className="ci-jobs-actions">
          <button
            type="button"
            className="ci-jobs-button"
            onClick={() => PlatformService.OpenWebPage(mark.webUrl).catch(ignoreDecorativeFailure)}
            disabled={!mark.webUrl}
          >
            <ExternalLink size={14} strokeWidth={1.75} />
            Open on host
          </button>
        </div>
      </div>
    </Modal>
  )
}

// A failed job opens its log on click; the log is fetched then, never
// before. Show full log asks for the rest, up to the cap.
function JobRow({ job, repoPath }: { job: Job; repoPath: string }) {
  const [open, setOpen] = useState(false)
  const [full, setFull] = useState(false)
  const failed = job.status === 'failed'
  const { data: log, error } = useAsyncData(
    () => (open ? HostService.JobLog(repoPath, job.id, full) : null),
    [open, full, repoPath, job.id],
    { keepData: true },
  )
  const row = (
    <>
      <CIIcon status={job.status} />
      <span className="ci-job-name">{job.name}</span>
      <span className="ci-job-stage">{job.stage}</span>
      <span className="ci-job-status">{job.status}</span>
      <span className="ci-job-duration">{formatDuration(job.duration)}</span>
    </>
  )
  if (!failed) return <li className="ci-job">{row}</li>
  return (
    <li className="ci-job ci-job-failed">
      <button type="button" className="ci-job-row" onClick={() => setOpen((o) => !o)} aria-expanded={open}>
        {row}
      </button>
      {open && (
        <div className="ci-job-log" aria-label={`Log of ${job.name}`}>
          {error && <p className="ci-jobs-error">{errorMessage(error)}</p>}
          {!log && !error && <p className="ci-jobs-hint">Loading log…</p>}
          {log && (
            <>
              <pre className="ci-job-log-text">{log.text}</pre>
              <div className="ci-job-log-actions">
                {log.truncated && !log.full && (
                  <button type="button" className="ci-jobs-button" onClick={() => setFull(true)}>
                    Show full log
                  </button>
                )}
                {log.full && log.truncated && (
                  <span className="ci-jobs-hint">Cut at the cap; the rest is on the host.</span>
                )}
                <button
                  type="button"
                  className="ci-jobs-button"
                  onClick={() => PlatformService.OpenWebPage(job.webUrl).catch(ignoreDecorativeFailure)}
                  disabled={!job.webUrl}
                >
                  <ExternalLink size={14} strokeWidth={1.75} />
                  Open on host
                </button>
              </div>
            </>
          )}
        </div>
      )}
    </li>
  )
}

// Durations arrive in nanoseconds, as Go's time.Duration serializes.
export function formatDuration(nanoseconds: number): string {
  const seconds = Math.round(nanoseconds / 1e9)
  if (seconds <= 0) return ''
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ${seconds % 60}s`
  return `${Math.floor(minutes / 60)}h ${minutes % 60}m`
}

export default CIMark
