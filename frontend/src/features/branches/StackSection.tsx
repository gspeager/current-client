import { useState, type MouseEvent, type KeyboardEvent } from 'react'
import { StackService, type StackInfo, type StackListInfo } from '@current-client-bindings/app'
import BranchPill from '../../components/git/BranchPill'
import { useCancellableOperation } from '../../lib/cancellableOperation'
import { errorMessage } from '../../lib/errors'
import { plural } from '../../lib/plural'
import { protectionFor } from '../../lib/protection'
import { primaryRemoteName } from '../../lib/remotes'
import { useDialogs } from '../../lib/useDialogs'
import { useLaneColors } from '../../lib/laneColor'
import RestackDialog from './RestackDialog'

interface StackSectionProps {
  repoPath: string
  list: StackListInfo
  current: string | null
  inOperation: boolean
  // Set from elsewhere, such as Go to anything.
  restack: string | null
  onRestackChange: (root: string | null) => void
  onCheckout: (branch: string) => void
  onRowMenu: (branch: string) => ((e: MouseEvent | KeyboardEvent) => void) | undefined
  onChanged: () => void
}

function layerMark(layer: StackInfo['layers'][number]) {
  if (layer.missing) return 'Gone'
  if (layer.merged) return 'Merged'
  if (layer.behind) return 'Behind'
  return null
}

function StackSection({
  repoPath,
  list,
  current,
  inOperation,
  restack,
  onRestackChange,
  onCheckout,
  onRowMenu,
  onChanged,
}: StackSectionProps) {
  const { branchColor } = useLaneColors()
  const { confirm } = useDialogs()
  const push = useCancellableOperation(repoPath)
  const [notice, setNotice] = useState<string | null>(null)
  const [stopped, setStopped] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const runRestack = (root: string) => {
    onRestackChange(null)
    setError(null)
    setNotice(null)
    StackService.Restack(repoPath, root)
      .then((result) => {
        const left = result.left ?? []
        setStopped(result.stopped || null)
        setNotice(left.length > 0 ? `${left.join(', ')} left the stack.` : null)
      })
      .catch((err: unknown) => setError(errorMessage(err)))
      .finally(onChanged)
  }

  const undoRestack = () => {
    setError(null)
    setNotice(null)
    StackService.UndoRestack(repoPath)
      .catch((err: unknown) => setError(errorMessage(err)))
      .finally(onChanged)
  }

  const pushAll = async (stack: StackInfo) => {
    setError(null)
    setNotice(null)
    try {
      const remote = await primaryRemoteName(repoPath)
      const branches = stack.layers.filter((layer) => !layer.missing && !layer.merged).map((layer) => layer.branch)
      const protections = await Promise.all(
        branches.map((branch) => protectionFor(repoPath, `${remote}/${branch}`, 'force-push')),
      )
      const confirmed = await confirm({
        title: 'Push all',
        message: [
          `Push ${branches.join(', ')} to ${remote} with --force-with-lease?`,
          'A branch someone else has pushed to since the last fetch stops the push.',
          ...protections,
        ]
          .filter(Boolean)
          .join(' '),
        confirmLabel: 'Push all',
      })
      if (!confirmed) return
      void push.run(
        (auth) => StackService.PushAll(repoPath, stack.root, remote, auth),
        () => {
          setNotice(`Pushed ${plural(branches.length, 'branch', 'branches')}.`)
          onChanged()
        },
      )
    } catch (err: unknown) {
      setError(errorMessage(err))
    }
  }

  return (
    <div className="stack-section">
      {error && (
        <p className="branch-sidebar-error" role="alert">
          {error}
        </p>
      )}
      {push.error && (
        <p className="branch-sidebar-error" role="alert">
          {push.error}
        </p>
      )}
      {stopped && inOperation && (
        <p className="branch-sidebar-notice" role="status">
          Stopped on {stopped}. Resolve it, then restack again for the layers above.
        </p>
      )}
      {notice && (
        <p className="branch-sidebar-notice" role="status">
          {notice}
        </p>
      )}
      {list.canUndo && (
        <button type="button" className="stack-action stack-undo" onClick={undoRestack}>
          Undo restack
        </button>
      )}
      {list.stacks.map((stack) => (
        <div key={stack.root} className="stack" role="group" aria-label={`Stack on ${stack.base}`}>
          <div className="stack-header">
            <span className="stack-base">on {stack.base}</span>
            <button
              type="button"
              className="stack-action"
              disabled={!stack.needsRestack}
              onClick={() => onRestackChange(stack.root)}
            >
              Restack
            </button>
            <button type="button" className="stack-action" disabled={push.running} onClick={() => void pushAll(stack)}>
              Push all
            </button>
          </div>
          <ul className="branch-sidebar-list">
            {stack.layers.map((layer) => {
              const mark = layerMark(layer)
              const isCurrent = layer.branch === current
              return (
                <li
                  key={layer.branch}
                  className="branch-row stack-row"
                  style={{ paddingLeft: `calc(var(--space-md) * ${layer.depth})` }}
                  role="button"
                  tabIndex={0}
                  onClick={() => !isCurrent && !layer.missing && onCheckout(layer.branch)}
                  onKeyDown={(e) => {
                    if ((e.key === 'Enter' || e.key === ' ') && !isCurrent && !layer.missing) {
                      e.preventDefault()
                      onCheckout(layer.branch)
                    }
                  }}
                  onContextMenu={onRowMenu(layer.branch)}
                >
                  <BranchPill name={layer.branch} color={branchColor(layer.branch, isCurrent)} current={isCurrent} />
                  {isCurrent && <span className="branch-row-head">HEAD</span>}
                  {mark && <span className={`stack-mark stack-mark-${mark.toLowerCase()}`}>{mark}</span>}
                </li>
              )
            })}
          </ul>
        </div>
      ))}
      {restack && (
        <RestackDialog
          repoPath={repoPath}
          root={restack}
          onRestack={() => runRestack(restack)}
          onClose={() => onRestackChange(null)}
        />
      )}
    </div>
  )
}

export default StackSection
