import { useEffect, useState } from 'react'
import { Search, SkipForward, Target, ThumbsDown, ThumbsUp } from 'lucide-react'
import { BisectService, HistoryService, type BisectStatusInfo, type CommitInfo } from '@current-client-bindings/app'
import { errorMessage } from '../../lib/errors'
import { relativeTime } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import { EMPTY_HISTORY_FILTER } from './useCommitHistory'
import Modal from '../../components/chrome/Modal'
import './BisectWizard.scss'

interface BisectWizardProps {
  repoPath: string
  onClose: () => void
  onViewCommit: (sha: string) => void
}

type Stage =
  | { kind: 'loading' }
  | { kind: 'setup' }
  | { kind: 'testing'; sha: string; revisionsLeft: number; stepsRemaining: number }
  | { kind: 'done'; sha: string }

function stageFromStatus(status: BisectStatusInfo): Stage {
  return status.done
    ? { kind: 'done', sha: status.foundSha }
    : {
        kind: 'testing',
        sha: status.currentSha,
        revisionsLeft: status.revisionsLeft,
        stepsRemaining: status.stepsRemaining,
      }
}

function plural(count: number, word: string): string {
  return `${count} ${word}${count === 1 ? '' : 's'}`
}

function Candidate({ sha, commit }: { sha: string; commit: CommitInfo | null }) {
  return (
    <div className="bisect-candidate">
      <span className="bisect-candidate-sha">{sha.slice(0, 7)}</span>
      {commit && (
        <>
          <span className="bisect-candidate-subject">{commit.subject}</span>
          <span className="bisect-candidate-meta">
            {commit.authorName} · {relativeTime(new Date(commit.date))}
          </span>
        </>
      )}
    </div>
  )
}

function BisectWizard({ repoPath, onClose, onViewCommit }: BisectWizardProps) {
  const [stage, setStage] = useState<Stage>({ kind: 'loading' })
  const [badRev, setBadRev] = useState('HEAD')
  const [goodRev, setGoodRev] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const candidateSha = stage.kind === 'testing' || stage.kind === 'done' ? stage.sha : null

  useEffect(() => {
    BisectService.IsActive(repoPath)
      .then((active) => {
        if (!active) {
          setStage({ kind: 'setup' })
          return
        }
        // When resuming, HEAD is the commit bisect last checked out; the
        // remaining counts are unknown until the next mark.
        return HistoryService.GetHistory(repoPath, 1, 0, EMPTY_HISTORY_FILTER).then((result) =>
          setStage({ kind: 'testing', sha: result[0]?.sha ?? '', revisionsLeft: -1, stepsRemaining: -1 }),
        )
      })
      .catch((err: unknown) => setError(errorMessage(err)))
  }, [repoPath])

  const commit =
    useAsyncData(
      () =>
        candidateSha
          ? HistoryService.GetHistory(repoPath, 1, 0, { ...EMPTY_HISTORY_FILTER, ref: candidateSha }).then(
              (result) => result[0] ?? null,
            )
          : null,
      [repoPath, candidateSha],
    ).data ?? null

  const runStep = (step: Promise<unknown>) => {
    setError(null)
    setBusy(true)
    step.catch((err: unknown) => setError(errorMessage(err))).finally(() => setBusy(false))
  }

  const start = () => {
    if (!goodRev.trim()) return
    runStep(BisectService.Start(repoPath, badRev.trim(), goodRev.trim()).then((s) => setStage(stageFromStatus(s))))
  }

  const mark = (verdict: 'good' | 'bad' | 'skip') => {
    runStep(BisectService.Mark(repoPath, verdict).then((s) => setStage(stageFromStatus(s))))
  }

  const reset = () => runStep(BisectService.Reset(repoPath).then(onClose))

  return (
    <Modal title="Bisect" onClose={onClose} className="bisect-wizard">
      <div className="bisect-body">
        {error && <p className="bisect-error">{error}</p>}

        {stage.kind === 'loading' && <p className="bisect-hint">Checking bisect status…</p>}

        {stage.kind === 'setup' && (
          <>
            <p className="bisect-hint">
              Pick a known-bad and a known-good commit. git checks out the commits in between for testing.
            </p>
            <label className="bisect-field">
              Bad commit
              <input value={badRev} onChange={(e) => setBadRev(e.target.value)} placeholder="HEAD" />
            </label>
            <label className="bisect-field">
              Good commit
              <input
                value={goodRev}
                onChange={(e) => setGoodRev(e.target.value)}
                placeholder="a known-good ref or SHA"
                autoFocus
              />
            </label>
            <div className="bisect-actions">
              <button type="button" onClick={start} disabled={busy || goodRev.trim() === ''}>
                <Search size={14} strokeWidth={1.75} />
                Start bisect
              </button>
            </div>
          </>
        )}

        {stage.kind === 'testing' && (
          <>
            <p className="bisect-progress">
              {stage.revisionsLeft < 0
                ? 'Resuming bisect…'
                : `${plural(stage.revisionsLeft, 'revision')} left — roughly ${plural(stage.stepsRemaining, 'step')}`}
            </p>
            <Candidate sha={stage.sha} commit={commit} />
            <p className="bisect-hint">Test this commit, then mark it:</p>
            <div className="bisect-actions">
              <button type="button" className="bisect-good" onClick={() => mark('good')} disabled={busy}>
                <ThumbsUp size={14} strokeWidth={1.75} />
                Good
              </button>
              <button type="button" className="bisect-bad" onClick={() => mark('bad')} disabled={busy}>
                <ThumbsDown size={14} strokeWidth={1.75} />
                Bad
              </button>
              <button type="button" onClick={() => mark('skip')} disabled={busy}>
                <SkipForward size={14} strokeWidth={1.75} />
                Skip
              </button>
            </div>
            <button type="button" className="bisect-reset-link" onClick={reset} disabled={busy}>
              Abort bisect
            </button>
          </>
        )}

        {stage.kind === 'done' && (
          <>
            <div className="bisect-found">
              <Target size={20} strokeWidth={1.75} />
              <span>First bad commit</span>
            </div>
            <Candidate sha={stage.sha} commit={commit} />
            <div className="bisect-actions">
              <button type="button" onClick={() => onViewCommit(stage.sha)}>
                View in History
              </button>
              <button type="button" onClick={reset} disabled={busy}>
                End bisect
              </button>
            </div>
          </>
        )}
      </div>
    </Modal>
  )
}

export default BisectWizard
