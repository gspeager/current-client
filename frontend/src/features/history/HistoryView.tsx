import { useEffect, useMemo, useRef, useState, type MouseEvent, type KeyboardEvent } from 'react'
import { useVirtualizer } from '@tanstack/react-virtual'
import { BranchService, type CommitInfo } from '@current-client-bindings/app'
import ContextMenu, { menuAnchor, type ContextMenuState } from '../../components/controls/ContextMenu'
import CommitDetailPanel from './CommitDetailPanel'
import CommitFileDiff from './CommitFileDiff'
import CommitRow from './CommitRow'
import WorkingChangesRow from './WorkingChangesRow'
import HistoryToolbar from './HistoryToolbar'
import { buildAsciiGraph } from './asciiGraph'
import { ancestorShas } from './commitAncestry'
import { identityKey } from './identityKey'
import { plural } from '../../lib/plural'
import { useAsyncData } from '../../lib/useAsyncData'
import { useCommitActions } from './useCommitActions'
import { useCommitGraph } from './useCommitGraph'
import { EMPTY_HISTORY_FILTER, useCommitHistory } from './useCommitHistory'
import { useCopyToClipboard } from '../../lib/useCopyToClipboard'
import { useWindowKeydown } from '../../lib/useWindowKeydown'
import { useHeadCommit } from '../working-copy/useHeadCommit'
import { useCurrentUser } from '../../lib/useCurrentUser'
import { useHostLinks } from '../../lib/useHostLinks'
import { useBranchChanges } from '../../lib/useBranchChanges'
import { useCIStatus } from '../../lib/useCIStatus'
import { useReferenceCards } from '../../lib/useReferenceCards'
import { useReviewThreads } from '../../lib/useReviewThreads'
import { useIdentities } from './useIdentities'
import { useRefBadges } from './useRefBadges'
import { ALL_BRANCHES, isBranchName } from './branchFilter'
import './HistoryView.scss'

interface HistoryViewProps {
  repoPath: string
  conventional?: boolean
  initialSelectedSha?: string | null
  // Only set when initialSelectedSha may be past the first loaded page.
  initialLoadThrough?: number | null
  // Opens History filtered to this branch instead of defaultRef.
  initialRef?: string | null
  // The branch filter to open with, and where a change to it is saved.
  defaultRef?: string
  onRefChange?: (ref: string) => void
  firstParent?: boolean
  onFirstParentChange?: (on: boolean) => void
  // The working tree's changed files, from the status App already holds;
  // with any, a row above HEAD says so and opens Changes.
  changedFiles?: number
  onOpenChanges?: () => void
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
  defaultRef = ALL_BRANCHES,
  onRefChange,
  firstParent = false,
  onFirstParentChange,
  changedFiles = 0,
  onOpenChanges,
  onBranchChanged,
  previousTopSha,
  onTopShaChange,
  onSelectedShaChange,
}: HistoryViewProps) {
  const [filter, setFilter] = useState({ ...EMPTY_HISTORY_FILTER, ref: initialRef ?? defaultRef, firstParent })
  const { commits, loading, done, error, loadMore } = useCommitHistory(repoPath, filter, initialLoadThrough)
  const head = useHeadCommit(repoPath)
  // Read only where a working-changes row can appear.
  const currentUser = useCurrentUser(changedFiles > 0 && onOpenChanges ? repoPath : null)
  // Only where HEAD is in the list, and no filter narrows it.
  const workingTreeParent =
    changedFiles > 0 &&
    onOpenChanges &&
    head &&
    (filter.ref === ALL_BRANCHES || filter.ref === '') &&
    !filter.since &&
    !filter.author &&
    !filter.type &&
    commits.some((c) => c.sha === head.sha)
      ? head.sha
      : null
  const { graphNodes, workingNode } = useCommitGraph(commits, filter.firstParent, workingTreeParent)
  const refsBySha = useRefBadges(repoPath)
  const hostLinks = useHostLinks(repoPath)
  // History remounts on every ref move, so the repository path is the only key.
  const changes = useBranchChanges(repoPath)
  const commitTexts = useMemo(() => commits.map((c) => c.subject + '\n' + c.body), [commits])
  const referenceCards = useReferenceCards(repoPath, commitTexts, changes)
  const { identities, colorError, setAuthorColor } = useIdentities(commits)
  const commitActions = useCommitActions(repoPath, onBranchChanged, hostLinks)

  const [selectedSha, setSelectedSha] = useState<string | null>(initialSelectedSha)
  const [search, setSearch] = useState('')
  const [hideMerges, setHideMerges] = useState(false)
  const [focusedSha, setFocusedSha] = useState<string | null>(null)
  // One highlight at a time: an author (by mailmapped email) or a lane.
  const [highlightedAuthor, setHighlightedAuthor] = useState<string | null>(null)
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null)
  // A file opened from the inspector; it only counts for the commit it was opened from.
  const [openedFile, setOpenedFile] = useState<{ sha: string; path: string } | null>(null)
  const { copied: graphCopied, copy } = useCopyToClipboard()
  const [newShas, setNewShas] = useState<Set<string>>(new Set())
  const hasComputedNewShas = useRef(false)
  const parentRef = useRef<HTMLDivElement>(null)

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

  const { data: localBranches } = useAsyncData(() => BranchService.ListLocal(repoPath), [repoPath])
  const branches = useMemo(() => localBranches?.map((b) => b.name), [localBranches])
  const upstreams = useMemo(
    () => Object.fromEntries((localBranches ?? []).filter((b) => b.upstream).map((b) => [b.name, b.upstream])),
    [localBranches],
  )

  // A saved branch that's since been deleted, or belongs to another repository.
  useEffect(() => {
    if (branches && !initialRef && isBranchName(filter.ref) && !branches.includes(filter.ref)) {
      setFilter((prev) => ({ ...prev, ref: ALL_BRANCHES }))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [branches])

  const changeFilter = (next: typeof filter) => {
    if (next.ref !== filter.ref) onRefChange?.(next.ref)
    if (next.firstParent !== filter.firstParent) onFirstParentChange?.(next.firstParent)
    setFilter(next)
  }

  const selectSha = (sha: string) => {
    setSelectedSha(sha)
    onSelectedShaChange?.(sha)
  }

  const focusedAncestors = useMemo(() => (focusedSha ? ancestorShas(commits, focusedSha) : null), [commits, focusedSha])
  const onFocusRef = (sha: string) => {
    setHighlightedAuthor(null)
    setFocusedSha((prev) => (prev === sha ? null : sha))
  }
  const highlightAuthor = (email: string) => {
    setFocusedSha(null)
    setHighlightedAuthor((prev) => (prev === email ? null : email))
  }
  useWindowKeydown((e) => {
    if (e.key === 'Escape' && highlightedAuthor) setHighlightedAuthor(null)
  })
  const isDimmed = (commit: CommitInfo) =>
    (focusedAncestors !== null && !focusedAncestors.has(commit.sha)) ||
    (highlightedAuthor !== null && commit.authorEmail !== highlightedAuthor)

  const selectedCommit = selectedSha ? (commits.find((c) => c.sha === selectedSha) ?? null) : null
  const reviewThreads = useReviewThreads(repoPath, undefined, selectedSha ?? undefined)
  const openedPath = selectedCommit && openedFile?.sha === selectedCommit.sha ? openedFile.path : null
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

  const workingRow = workingNode && workingTreeParent && !query ? workingNode : null
  const offset = workingRow ? 1 : 0
  const virtualizer = useVirtualizer({
    count: rows.length + offset,
    getScrollElement: () => parentRef.current,
    estimateSize: () => ROW_HEIGHT,
    overscan: 10,
  })
  const virtualItems = virtualizer.getVirtualItems()
  const ciMarks = useCIStatus(
    repoPath,
    virtualItems.flatMap((row) => (row.index < offset ? [] : [rows[row.index - offset].commit.sha])),
  )

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
    virtualizer.scrollToIndex(index + offset, { align: 'center' })
  }, [rows, offset, initialSelectedSha, virtualizer])

  if (error) {
    return <p className="history-empty-state history-empty-state-error">Could not load history: {error}</p>
  }
  if (commits.length === 0 && !loading) {
    return <p className="history-empty-state">No commits.</p>
  }

  const laneCount = [...graphNodes, ...(workingNode ? [workingNode] : [])].reduce(
    (max, n) => Math.max(max, n.lane + 1, ...n.passThrough.map((l) => l + 1)),
    1,
  )

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

  const commitContextMenu = (commit: CommitInfo) => (e: MouseEvent | KeyboardEvent) => {
    e.preventDefault()
    e.stopPropagation()
    selectSha(commit.sha)
    setContextMenu({
      ...menuAnchor(e),
      items: [
        ...commitActions
          .actionsFor(commit)
          .map((a) => ({ label: a.label, onClick: a.run, destructive: a.destructive })),
        {
          label: highlightedAuthor === commit.authorEmail ? 'Clear author highlight' : 'Highlight author’s commits',
          onClick: () => highlightAuthor(commit.authorEmail),
        },
      ],
    })
  }

  return (
    <div className="history-view">
      <div className="history-canvas">
        {selectedCommit && openedPath ? (
          <CommitFileDiff
            repoPath={repoPath}
            commit={selectedCommit}
            path={openedPath}
            reviewThreads={reviewThreads}
            onBack={() => setOpenedFile(null)}
          />
        ) : (
          <>
            <HistoryToolbar
              search={search}
              onSearchChange={setSearch}
              hideMerges={hideMerges}
              onHideMergesChange={setHideMerges}
              branches={branches ?? []}
              filter={filter}
              onFilterChange={changeFilter}
              showTypeFilter={conventional}
              onCopyAsciiGraph={copyAsciiGraph}
              copied={graphCopied}
            />

            <div ref={parentRef} className="history-scroll">
              <div style={{ height: virtualizer.getTotalSize(), position: 'relative', width: '100%' }}>
                {virtualItems.map((row) => {
                  const rowStyle = {
                    position: 'absolute' as const,
                    top: 0,
                    left: 0,
                    width: '100%',
                    height: ROW_HEIGHT,
                    transform: `translateY(${row.start}px)`,
                  }
                  if (workingRow && row.index === 0) {
                    return (
                      <WorkingChangesRow
                        key="working-tree"
                        node={workingRow}
                        laneCount={laneCount}
                        rowHeight={ROW_HEIGHT}
                        changedFiles={changedFiles}
                        user={currentUser}
                        style={rowStyle}
                        onOpen={onOpenChanges!}
                      />
                    )
                  }
                  const { commit, graphNode } = rows[row.index - offset]
                  return (
                    <CommitRow
                      key={commit.sha}
                      commit={commit}
                      graphNode={graphNode}
                      laneCount={laneCount}
                      hostLinks={hostLinks}
                      changes={changes}
                      repoPath={repoPath}
                      ci={ciMarks[commit.sha]}
                      referenceCards={referenceCards}
                      rowHeight={ROW_HEIGHT}
                      identity={identities.get(identityKey(commit.authorName, commit.authorEmail))}
                      refs={refsBySha.get(commit.sha)}
                      upstreams={upstreams}
                      selected={commit.sha === selectedSha}
                      dimmed={isDimmed(commit)}
                      isNew={newShas.has(commit.sha)}
                      focusedSha={focusedSha}
                      conventional={conventional}
                      onFocusRef={onFocusRef}
                      searchQuery={query}
                      style={rowStyle}
                      onSelect={() => selectSha(commit.sha)}
                      onContextMenu={commitContextMenu(commit)}
                      onHighlightAuthor={() => highlightAuthor(commit.authorEmail)}
                      authorHighlighted={highlightedAuthor === commit.authorEmail}
                    />
                  )
                })}
              </div>
              {loading && <p className="history-loading">Loading…</p>}
              {done && !loading && (
                <p className="history-footer">
                  {plural(commits.length, 'commit')}
                  {filter.ref === ALL_BRANCHES && ' · stashes hidden'}
                </p>
              )}
            </div>
          </>
        )}
      </div>

      {selectedCommit && (
        <div className="history-inspector">
          <CommitDetailPanel
            repoPath={repoPath}
            commit={selectedCommit}
            identity={selectedIdentity}
            hostLinks={hostLinks}
            referenceCards={referenceCards}
            colorError={colorError}
            onColorChange={(color) => setAuthorColor(selectedCommit.authorName, selectedCommit.authorEmail, color)}
            actions={commitActions.actionsFor(selectedCommit)}
            actionError={commitActions.error}
            openedPath={openedPath}
            onOpenFile={(path) => setOpenedFile({ sha: selectedCommit.sha, path })}
            threadCounts={reviewThreads.counts}
          />
        </div>
      )}

      {commitActions.overlays}
      {contextMenu && <ContextMenu state={contextMenu} onClose={() => setContextMenu(null)} />}
    </div>
  )
}

export default HistoryView
