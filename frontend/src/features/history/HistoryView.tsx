import { useEffect, useMemo, useRef, useState, type MouseEvent } from 'react'
import { useVirtualizer } from '@tanstack/react-virtual'
import { BranchService, CommitService, PatchService, type CommitInfo } from '@current-client-bindings/app'
import ResizeHandle from '../../components/chrome/ResizeHandle'
import ContextMenu, { type ContextMenuState } from '../../components/controls/ContextMenu'
import CommitDetailPanel from './CommitDetailPanel'
import CommitRow from './CommitRow'
import HistoryToolbar from './HistoryToolbar'
import { buildAsciiGraph } from './asciiGraph'
import { ancestorShas } from './commitAncestry'
import { errorMessage } from '../../lib/errors'
import { identityKey } from './identityKey'
import { suggestedPatchFilename } from '../../lib/patchFilename'
import { useAsyncData } from '../../lib/useAsyncData'
import { useCommitGraph } from './useCommitGraph'
import { EMPTY_HISTORY_FILTER, useCommitHistory } from './useCommitHistory'
import { useCopyToClipboard } from '../../lib/useCopyToClipboard'
import { useCreateBranchFromCommit } from './useCreateBranchFromCommit'
import { useDialogs } from '../../lib/useDialogs'
import { useIdentities } from './useIdentities'
import { useRefBadges } from './useRefBadges'
import { useResizableWidth } from '../../lib/useResizableWidth'
import './HistoryView.scss'

interface HistoryViewProps {
  repoPath: string
  conventional?: boolean
  initialSelectedSha?: string | null
  // Only set when initialSelectedSha may be past the first loaded page.
  initialLoadThrough?: number | null
  // Opens History filtered to this branch instead of the current one.
  initialRef?: string | null
  onBranchChanged?: () => void
  // App keeps the last-seen top SHA across remounts so new commits can flash.
  previousTopSha?: string | null
  onTopShaChange?: (sha: string) => void
  onSelectedShaChange?: (sha: string) => void
}

// Must match --h-commit-row.
const ROW_HEIGHT = 38
const NEW_COMMIT_FLASH_MS = 400

function HistoryView({
  repoPath,
  conventional = false,
  initialSelectedSha = null,
  initialLoadThrough = null,
  initialRef = null,
  onBranchChanged,
  previousTopSha,
  onTopShaChange,
  onSelectedShaChange,
}: HistoryViewProps) {
  const { confirm } = useDialogs()
  const [filter, setFilter] = useState({ ...EMPTY_HISTORY_FILTER, ref: initialRef ?? '' })
  const { commits, loading, done, error, loadMore } = useCommitHistory(repoPath, filter, initialLoadThrough)
  const graphNodes = useCommitGraph(commits)
  const refsBySha = useRefBadges(repoPath)
  const { identities, colorError, setAuthorColor } = useIdentities(commits)
  const { createBranchFrom, branchError } = useCreateBranchFromCommit(repoPath, onBranchChanged)

  const [selectedSha, setSelectedSha] = useState<string | null>(initialSelectedSha)
  const [search, setSearch] = useState('')
  const [hideMerges, setHideMerges] = useState(false)
  const [focusedSha, setFocusedSha] = useState<string | null>(null)
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null)
  const [resetError, setResetError] = useState<string | null>(null)
  const [patchError, setPatchError] = useState<string | null>(null)
  const { copied: graphCopied, copy } = useCopyToClipboard()
  const [newShas, setNewShas] = useState<Set<string>>(new Set())
  const hasComputedNewShas = useRef(false)
  const parentRef = useRef<HTMLDivElement>(null)
  const inspectorWidth = useResizableWidth('pane-width-inspector', 376, 280, 640, 'grow-left')

  // Only the first load counts; no overlap with previousTopSha (a rebase or
  // first open) flashes nothing.
  useEffect(() => {
    if (commits.length === 0 || hasComputedNewShas.current) return
    hasComputedNewShas.current = true
    if (previousTopSha) {
      const priorIndex = commits.findIndex((c) => c.sha === previousTopSha)
      if (priorIndex > 0) {
        setNewShas(new Set(commits.slice(0, priorIndex).map((c) => c.sha)))
      }
    }
    onTopShaChange?.(commits[0].sha)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [commits])

  useEffect(() => {
    if (newShas.size === 0) return
    const timeout = setTimeout(() => setNewShas(new Set()), NEW_COMMIT_FLASH_MS)
    return () => clearTimeout(timeout)
  }, [newShas])

  const { data: branches } = useAsyncData(
    () => BranchService.ListLocal(repoPath).then((result) => result.map((b) => b.name)),
    [repoPath],
  )

  const selectSha = (sha: string) => {
    setSelectedSha(sha)
    onSelectedShaChange?.(sha)
  }

  const focusedAncestors = useMemo(() => (focusedSha ? ancestorShas(commits, focusedSha) : null), [commits, focusedSha])
  const onFocusRef = (sha: string) => setFocusedSha((prev) => (prev === sha ? null : sha))

  const selectedCommit = selectedSha ? (commits.find((c) => c.sha === selectedSha) ?? null) : null
  const selectedIdentity = selectedCommit
    ? identities.get(identityKey(selectedCommit.authorName, selectedCommit.authorEmail))
    : undefined

  // Graph nodes come from the unfiltered history so filtered rows keep their lanes.
  const query = search.trim().toLowerCase()
  const rows = commits
    .map((commit, index) => ({ commit, graphNode: graphNodes[index], index }))
    .filter(({ commit, graphNode }) => {
      if (hideMerges && (graphNode?.parentLanes.length ?? 0) > 1) return false
      if (!query) return true
      return (
        commit.subject.toLowerCase().includes(query) ||
        commit.authorName.toLowerCase().includes(query) ||
        commit.sha.toLowerCase().includes(query)
      )
    })

  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => ROW_HEIGHT,
    overscan: 10,
  })
  const virtualItems = virtualizer.getVirtualItems()

  useEffect(() => {
    const last = virtualItems[virtualItems.length - 1]
    if (last && last.index >= commits.length - 20) {
      loadMore(commits.length, done)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [virtualItems, commits.length, done])

  // initialLoadThrough guarantees the commit is loaded, not visible.
  const hasScrolledToInitial = useRef(false)
  useEffect(() => {
    if (hasScrolledToInitial.current || !initialSelectedSha) return
    const index = rows.findIndex((r) => r.commit.sha === initialSelectedSha)
    if (index === -1) return
    hasScrolledToInitial.current = true
    virtualizer.scrollToIndex(index, { align: 'center' })
  }, [rows, initialSelectedSha, virtualizer])

  if (error) {
    return <p className="history-empty-state history-empty-state-error">Could not load history: {error}</p>
  }
  if (commits.length === 0 && !loading) {
    return <p className="history-empty-state">No commits.</p>
  }

  const laneCount = graphNodes.reduce((max, n) => Math.max(max, n.lane + 1, ...n.passThrough.map((l) => l + 1)), 1)

  const resetTo = async (sha: string, mode: 'soft' | 'mixed' | 'hard') => {
    const confirmed =
      mode !== 'hard' ||
      (await confirm({
        title: 'Reset (hard)',
        message: 'Reset (hard) discards all uncommitted changes and any commits after this one.',
        confirmLabel: 'Reset',
        destructive: true,
      }))
    if (!confirmed) return
    setResetError(null)
    CommitService.Reset(repoPath, sha, mode)
      .then(() => onBranchChanged?.())
      .catch((err: unknown) => setResetError(errorMessage(err)))
  }

  const exportPatch = (sha: string, subject: string) => {
    setPatchError(null)
    PatchService.ExportCommit(repoPath, sha, suggestedPatchFilename(sha, subject)).catch((err: unknown) =>
      setPatchError(errorMessage(err)),
    )
  }

  // Copies the visible, filtered rows rather than a fresh git log --graph.
  const copyAsciiGraph = () => {
    copy(
      buildAsciiGraph(
        rows.map(({ commit, graphNode }) => ({
          sha: commit.sha,
          subject: commit.subject,
          lane: graphNode?.lane ?? 0,
          passThrough: graphNode?.passThrough ?? [],
        })),
      ),
    )
  }

  const commitContextMenu = (commit: CommitInfo) => (e: MouseEvent) => {
    e.preventDefault()
    selectSha(commit.sha)
    setContextMenu({
      x: e.clientX,
      y: e.clientY,
      items: [
        { label: 'Copy SHA', onClick: () => void navigator.clipboard.writeText(commit.sha) },
        { label: 'Branch here', onClick: () => createBranchFrom(commit.sha) },
        { label: 'Export patch…', onClick: () => exportPatch(commit.sha, commit.subject) },
        { label: 'Reset (soft) to here', onClick: () => resetTo(commit.sha, 'soft') },
        { label: 'Reset (mixed) to here', onClick: () => resetTo(commit.sha, 'mixed') },
        { label: 'Reset (hard) to here', onClick: () => resetTo(commit.sha, 'hard'), destructive: true },
      ],
    })
  }

  return (
    <div className="history-view">
      <div className="history-canvas">
        <HistoryToolbar
          search={search}
          onSearchChange={setSearch}
          hideMerges={hideMerges}
          onHideMergesChange={setHideMerges}
          branches={branches ?? []}
          filter={filter}
          onFilterChange={setFilter}
          showTypeFilter={conventional}
          onCopyAsciiGraph={copyAsciiGraph}
          copied={graphCopied}
        />

        {branchError && <p className="history-error">Could not create branch: {branchError}</p>}
        {resetError && <p className="history-error">Could not reset: {resetError}</p>}
        {patchError && <p className="history-error">Could not export patch: {patchError}</p>}

        <div ref={parentRef} className="history-scroll">
          <div style={{ height: virtualizer.getTotalSize(), position: 'relative', width: '100%' }}>
            {virtualItems.map((row) => {
              const { commit, graphNode, index } = rows[row.index]
              return (
                <CommitRow
                  key={commit.sha}
                  commit={commit}
                  graphNode={graphNode}
                  hasAbove={index > 0}
                  laneCount={laneCount}
                  rowHeight={ROW_HEIGHT}
                  identity={identities.get(identityKey(commit.authorName, commit.authorEmail))}
                  refs={refsBySha.get(commit.sha)}
                  selected={commit.sha === selectedSha}
                  dimmed={focusedAncestors !== null && !focusedAncestors.has(commit.sha)}
                  isNew={newShas.has(commit.sha)}
                  focusedSha={focusedSha}
                  conventional={conventional}
                  onFocusRef={onFocusRef}
                  searchQuery={query}
                  style={{
                    position: 'absolute',
                    top: 0,
                    left: 0,
                    width: '100%',
                    height: ROW_HEIGHT,
                    transform: `translateY(${row.start}px)`,
                  }}
                  onSelect={() => selectSha(commit.sha)}
                  onContextMenu={commitContextMenu(commit)}
                />
              )
            })}
          </div>
          {loading && <p className="history-loading">Loading…</p>}
        </div>
      </div>

      {selectedCommit && (
        <>
          <ResizeHandle onDragStart={inspectorWidth.onDragStart} ariaLabel="Resize commit inspector" />
          <div className="history-inspector" style={{ width: inspectorWidth.width, minWidth: inspectorWidth.width }}>
            <CommitDetailPanel
              repoPath={repoPath}
              commit={selectedCommit}
              identity={selectedIdentity}
              colorError={colorError}
              onColorChange={(color) => setAuthorColor(selectedCommit.authorName, selectedCommit.authorEmail, color)}
              onCreateBranchHere={() => createBranchFrom(selectedCommit.sha)}
              onBranchChanged={onBranchChanged}
            />
          </div>
        </>
      )}

      {contextMenu && <ContextMenu state={contextMenu} onClose={() => setContextMenu(null)} />}
    </div>
  )
}

export default HistoryView
