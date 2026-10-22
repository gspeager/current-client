import { useState, type MouseEvent } from 'react'
import { ExternalLink } from 'lucide-react'
import { HostService, PlatformService, type CIMark as CIMarkInfo } from '@current-client-bindings/app'
import type { Job } from '@current-client-bindings/core/hostapi'
import Modal from '../chrome/Modal'
import { errorMessage, ignoreDecorativeFailure } from '../../lib/errors'
import { useAsyncData } from '../../lib/useAsyncData'
import './CIMark.scss'

const LABELS: Record<string, string> = {
  success: 'CI passed',
  failed: 'CI failed',
  running: 'CI running',
  pending: 'CI pending',
  canceled: 'CI canceled',
  skipped: 'CI skipped',
}

interface CIMarkProps {
  repoPath: string
  mark: CIMarkInfo | undefined
}

// The pipeline result for a commit or branch tip: a 3px square in the status
// color. Clicking lists the jobs, fetched then, with Open on host.
function CIMark({ repoPath, mark }: CIMarkProps) {
  const [open, setOpen] = useState(false)
  if (!mark?.status) return null
  const label = LABELS[mark.status] ?? `CI ${mark.status}`
  const onClick = (e: MouseEvent) => {
    e.stopPropagation()
    setOpen(true)
  }
  return (
    <>
      <button
        type="button"
        className={`ci-mark ci-mark-${mark.status}`}
        title={label}
        aria-label={label}
        onClick={onClick}
      />
      {open && <PipelineJobs repoPath={repoPath} mark={mark} onClose={() => setOpen(false)} />}
    </>
  )
}

function PipelineJobs({ repoPath, mark, onClose }: { repoPath: string; mark: CIMarkInfo; onClose: () => void }) {
  const { data: jobs, error } = useAsyncData(
    () => HostService.PipelineJobs(repoPath, mark.pipelineId),
    [repoPath, mark.pipelineId],
  )
  return (
    <Modal title={LABELS[mark.status] ?? 'Pipeline'} onClose={onClose} className="ci-jobs-panel">
      <div className="ci-jobs">
        {error && <p className="ci-jobs-error">{errorMessage(error)}</p>}
        {jobs === null && !error && <p className="ci-jobs-hint">Loading jobs…</p>}
        {jobs?.length === 0 && <p className="ci-jobs-hint">No jobs.</p>}
        {jobs && jobs.length > 0 && (
          <ul className="ci-jobs-list" aria-label="Jobs">
            {jobs.map((job) => (
              <JobRow key={job.id} job={job} />
            ))}
          </ul>
        )}
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

function JobRow({ job }: { job: Job }) {
  return (
    <li className="ci-job">
      <span className={`ci-mark ci-mark-${job.status}`} aria-hidden="true" />
      <span className="ci-job-name">{job.name}</span>
      <span className="ci-job-stage">{job.stage}</span>
      <span className="ci-job-status">{job.status}</span>
      <span className="ci-job-duration">{formatDuration(job.duration)}</span>
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
