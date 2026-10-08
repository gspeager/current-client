import type { DragEvent } from 'react'
import { X } from 'lucide-react'
import type { RepoSummaryInfo } from '@current-client-bindings/app'
import { baseName } from '../../lib/paths'
import './RepoTabChip.scss'

interface RepoTabChipProps {
  path: string
  summary?: RepoSummaryInfo
  onFocus: () => void
  onClose: () => void
  onDragStart: () => void
  onDragOver: (e: DragEvent) => void
  onDrop: () => void
}

// What a screen reader hears for the tab: name, branch, and how far it is from its upstream.
function tabLabel(path: string, summary?: RepoSummaryInfo): string {
  const parts = [baseName(path)]
  if (summary && !summary.available) parts.push('not found')
  if (summary?.available && summary.currentBranch) parts.push(summary.currentBranch)
  if (summary?.available && summary.behind > 0) parts.push(`${summary.behind} behind`)
  if (summary?.available && summary.ahead > 0) parts.push(`${summary.ahead} ahead`)
  if (summary?.available && summary.dirty > 0) parts.push('uncommitted changes')
  return parts.join(', ')
}

// Background tabs only; the focused tab renders as HeaderBar's breadcrumb.
function RepoTabChip({ path, summary, onFocus, onClose, onDragStart, onDragOver, onDrop }: RepoTabChipProps) {
  const dotClass = !summary
    ? 'header-bar-tab-dot'
    : !summary.available
      ? 'header-bar-tab-dot header-bar-tab-dot-unavailable'
      : summary.dirty > 0
        ? 'header-bar-tab-dot header-bar-tab-dot-dirty'
        : 'header-bar-tab-dot header-bar-tab-dot-clean'

  return (
    <div
      className="header-bar-tab"
      draggable
      onDragStart={onDragStart}
      onDragOver={onDragOver}
      onDrop={onDrop}
      title={path}
    >
      <button type="button" className="header-bar-tab-open" onClick={onFocus} aria-label={tabLabel(path, summary)}>
        <span className={dotClass} />
        <span className="header-bar-tab-name">{baseName(path)}</span>
        {summary?.available && summary.currentBranch && (
          <>
            <span className="header-bar-tab-sep">|</span>
            <span className="header-bar-tab-branch">{summary.currentBranch}</span>
          </>
        )}
        {summary?.available && (summary.ahead > 0 || summary.behind > 0) && (
          <span className="header-bar-tab-meta">
            {summary.behind > 0 && <span className="header-bar-tab-behind">↓{summary.behind}</span>}
            {summary.ahead > 0 && <span className="header-bar-tab-ahead">↑{summary.ahead}</span>}
          </span>
        )}
      </button>
      <button
        type="button"
        className="header-bar-tab-close"
        aria-label={`Close ${baseName(path)}`}
        onClick={(e) => {
          e.stopPropagation()
          onClose()
        }}
      >
        <X size={11} strokeWidth={1.75} />
      </button>
    </div>
  )
}

export default RepoTabChip
