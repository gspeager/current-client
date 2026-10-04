import { useState } from 'react'
import { Check, ExternalLink } from 'lucide-react'
import { DiffService } from '@current-client-bindings/app'
import ResizeHandle from '../../components/chrome/ResizeHandle'
import Checkbox from '../../components/forms/Checkbox'
import CommitComposer from './CommitComposer'
import DiffViewer from '../diff/DiffViewer'
import DiffViewModeToggle, { type DiffViewMode } from '../diff/DiffViewModeToggle'
import WorkingTreeFileList from './WorkingTreeFileList'
import { diffStats } from '../diff/diffRows'
import { errorMessage } from '../../lib/errors'
import { useAsyncData } from '../../lib/useAsyncData'
import { useDialogs } from '../../lib/useDialogs'
import { useHeadCommit } from './useHeadCommit'
import { useResizableWidth } from '../../lib/useResizableWidth'
import type { WorkingTreeSection, useWorkingTree } from './useWorkingTree'
import { useWorkingTreeActions } from './useWorkingTreeActions'
import './ChangesView.scss'
import { useWindowKeydown } from '../../lib/useWindowKeydown'

interface ChangesViewProps {
  repoPath: string
  workingTree: ReturnType<typeof useWorkingTree>
  onPushRequested?: () => void
  onBranchChanged?: () => void
  defaultIgnoreWhitespace?: boolean
  initialSelectedPath?: string | null
  onSelectedPathChange?: (path: string | null) => void
  initialCommitDraft?: string
  onCommitDraftChange?: (message: string) => void
  showCommitTypePicker?: boolean
}

function splitDirFile(path: string): { dir: string; file: string } {
  const slash = path.lastIndexOf('/')
  return slash === -1 ? { dir: '', file: path } : { dir: path.slice(0, slash + 1), file: path.slice(slash + 1) }
}

function ChangesView({
  repoPath,
  workingTree,
  onPushRequested,
  onBranchChanged,
  defaultIgnoreWhitespace = false,
  initialSelectedPath = null,
  onSelectedPathChange,
  initialCommitDraft = '',
  onCommitDraftChange,
  showCommitTypePicker = false,
}: ChangesViewProps) {
  const { files, staged, unstaged, loadStatus } = workingTree
  const actions = useWorkingTreeActions(repoPath, loadStatus)
  const { confirm } = useDialogs()
  const [selectedPath, setSelectedPath] = useState<string | null>(initialSelectedPath)
  // A staged file's diff is HEAD→index; an unstaged file's is index→working tree.
  const [openSection, setOpenSection] = useState<WorkingTreeSection>('unstaged')
  const [forcedPath, setForcedPath] = useState<string | null>(null)
  const [ignoreWhitespace, setIgnoreWhitespace] = useState(defaultIgnoreWhitespace)
  const [viewMode, setViewMode] = useState<DiffViewMode>('split')
  const [externalToolError, setExternalToolError] = useState<string | null>(null)
  const [externalToolBusy, setExternalToolBusy] = useState(false)
  const headCommit = useHeadCommit(repoPath)
  const filesWidth = useResizableWidth('pane-width-changes-files', 318, 240, 640)
  const showsStaged = openSection === 'staged'

  const selectPath = (path: string | null) => {
    setSelectedPath(path)
    onSelectedPathChange?.(path)
  }

  const openFile = (path: string, section: WorkingTreeSection) => {
    selectPath(path)
    setOpenSection(section)
  }

  // Large files load only after an explicit request, and only for that file.
  // Every status reload (file watcher, staging, hunk actions) also refreshes the open diff.
  const force = forcedPath !== null && forcedPath === selectedPath
  const { data: diff, error: diffError } = useAsyncData(
    () => {
      if (!selectedPath) return null
      return showsStaged
        ? DiffService.GetIndexDiff(repoPath, selectedPath, ignoreWhitespace)
        : DiffService.GetWorkingTreeDiff(repoPath, selectedPath, force, ignoreWhitespace)
    },
    [repoPath, selectedPath, showsStaged, ignoreWhitespace, force],
    { refreshKey: files },
  )

  const hunkAction = (apply: (path: string, hunk: string) => Promise<void>) => (hunk: string) => {
    if (selectedPath) actions.run(apply(selectedPath, hunk))
  }

  const stageHunk = hunkAction((path, hunk) => DiffService.StageHunk(repoPath, path, hunk))
  const unstageHunk = hunkAction((path, hunk) => DiffService.UnstageHunk(repoPath, path, hunk))
  const applyDiscardHunk = hunkAction((path, hunk) =>
    (showsStaged ? DiffService.DiscardStagedHunk : DiffService.DiscardHunk)(repoPath, path, hunk),
  )
  const discardHunk = async (hunk: string) => {
    const confirmed = await confirm({
      title: 'Discard hunk',
      message: showsStaged
        ? 'Discard this staged hunk from the index and the working tree? This cannot be undone.'
        : 'Discard this hunk? This cannot be undone.',
      confirmLabel: 'Discard',
      destructive: true,
    })
    if (confirmed) applyDiscardHunk(hunk)
  }

  const openInExternalTool = () => {
    if (!selectedPath) return
    setExternalToolError(null)
    setExternalToolBusy(true)
    DiffService.OpenInExternalTool(repoPath, selectedPath, showsStaged)
      .catch((err: unknown) => setExternalToolError(errorMessage(err)))
      .finally(() => setExternalToolBusy(false))
  }

  useWindowKeydown((e) => {
    if (e.code !== 'Space' || !selectedPath) return
    const target = e.target as HTMLElement | null
    if (target instanceof HTMLInputElement || target instanceof HTMLTextAreaElement || target?.isContentEditable) return
    e.preventDefault()
    const f = (showsStaged ? staged : unstaged).find((x) => x.path === selectedPath)
    if (f) (showsStaged ? actions.unstage : actions.stage)(f)
  })

  // A staged diff disappears once committed. repoVersion must bump so the
  // header's ahead count sees the new commit.
  const onCommitted = () => {
    loadStatus()
    onBranchChanged?.()
    if (showsStaged) {
      selectPath(null)
    }
  }

  const fullyStaged =
    selectedPath !== null &&
    staged.some((f) => f.path === selectedPath) &&
    !unstaged.some((f) => f.path === selectedPath)
  const stats = diff ? diffStats(diff) : null
  const { dir, file } = selectedPath ? splitDirFile(selectedPath) : { dir: '', file: '' }

  return (
    <div className="changes-view">
      <div className="changes-files" style={{ width: filesWidth.width, minWidth: filesWidth.width }}>
        <WorkingTreeFileList
          repoPath={repoPath}
          workingTree={workingTree}
          actions={actions}
          openPath={selectedPath}
          openSection={openSection}
          onOpen={openFile}
        />
        <CommitComposer
          repoPath={repoPath}
          stagedCount={staged.length}
          onCommitted={onCommitted}
          onPushRequested={onPushRequested}
          initialMessage={initialCommitDraft}
          onMessageChange={onCommitDraftChange}
          showTypePicker={showCommitTypePicker}
        />
      </div>

      <ResizeHandle onDragStart={filesWidth.onDragStart} ariaLabel="Resize file list" />

      <div className="changes-diff">
        {selectedPath ? (
          <>
            <div className="changes-diff-header">
              <span className="changes-diff-path">
                {dir && <span className="changes-diff-path-dir">{dir}</span>}
                <span className="changes-diff-path-file">{file}</span>
              </span>
              {stats && (
                <span className="changes-diff-stats">
                  <span className="changes-diff-stat-added">+{stats.added}</span>
                  <span className="changes-diff-stat-removed">-{stats.removed}</span>
                  <span className="changes-diff-stat-hunks">
                    {stats.hunks} hunk{stats.hunks === 1 ? '' : 's'}
                  </span>
                </span>
              )}

              <div className="changes-diff-header-spacer" />

              <DiffViewModeToggle value={viewMode} onChange={setViewMode} />
              <Checkbox checked={ignoreWhitespace} onChange={setIgnoreWhitespace} label="Ignore whitespace" />
              <button
                type="button"
                className="changes-diff-external-tool"
                onClick={openInExternalTool}
                disabled={externalToolBusy}
                aria-label="Open in external tool"
                title="Open in external tool"
              >
                <ExternalLink size={14} strokeWidth={1.75} />
              </button>
              {fullyStaged && (
                <span className="changes-diff-staged-badge">
                  <Check size={12} strokeWidth={2} /> Fully staged
                </span>
              )}
            </div>
            {externalToolError && <p className="changes-error">Could not open external tool: {externalToolError}</p>}
            {diffError ? (
              <p className="changes-empty">Could not load diff: {diffError}</p>
            ) : diff ? (
              <DiffViewer
                diff={diff}
                path={selectedPath}
                viewMode={viewMode}
                baseSide={
                  showsStaged
                    ? { label: 'HEAD · working tree base', sha: headCommit?.sha.slice(0, 7) }
                    : { label: 'Staged index' }
                }
                targetSide={showsStaged ? { label: 'Staged index' } : { label: 'Working tree' }}
                onStageHunk={showsStaged ? undefined : stageHunk}
                onUnstageHunk={showsStaged ? unstageHunk : undefined}
                onDiscardHunk={discardHunk}
                onForceLoad={() => setForcedPath(selectedPath)}
                images={{
                  repoPath,
                  before: showsStaged ? { kind: 'commit', rev: 'HEAD' } : { kind: 'index', rev: '' },
                  after: showsStaged ? { kind: 'index', rev: '' } : { kind: 'worktree', rev: '' },
                }}
              />
            ) : (
              <p className="changes-empty">Loading diff…</p>
            )}
          </>
        ) : (
          <p className="changes-empty">Select a file to see its diff.</p>
        )}
      </div>
    </div>
  )
}

export default ChangesView
