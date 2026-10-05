import { useMemo, useRef, useState, type CSSProperties, type MouseEvent } from 'react'
import { useVirtualizer } from '@tanstack/react-virtual'
import { ChevronDown, ChevronRight, FileDown, Folder, FolderTree, List, RefreshCw, Search } from 'lucide-react'
import { PatchService, type FileStatus } from '@current-client-bindings/app'
import type { ContextMenuItem } from '../../components/controls/ContextMenu'
import SegmentedControl from '../../components/controls/SegmentedControl'
import { errorMessage } from '../../lib/errors'
import { buildFileTreeRows } from './fileTree'
import { baseName } from '../../lib/paths'
import { suggestedWorkingTreePatchFilename } from '../../lib/patchFilename'
import { useFileTools } from '../history/useFileTools'
import type { WorkingTreeSection, useWorkingTree } from './useWorkingTree'
import type { WorkingTreeActions } from './useWorkingTreeActions'
import FileRow from './FileRow'
import './WorkingTreeFileList.scss'

interface WorkingTreeFileListProps {
  repoPath: string
  workingTree: ReturnType<typeof useWorkingTree>
  actions: WorkingTreeActions
  openPath: string | null
  openSection: WorkingTreeSection
  onOpen: (path: string, section: WorkingTreeSection) => void
}

type ListRow =
  | { kind: 'header'; section: WorkingTreeSection }
  | { kind: 'folder'; section: WorkingTreeSection; path: string; name: string; depth: number }
  | { kind: 'file'; section: WorkingTreeSection; file: FileStatus; depth: number }

// Must match --h-micro, --h-row-tree, --h-row-table, and --space-md.
const HEADER_ROW_HEIGHT = 20
const FOLDER_ROW_HEIGHT = 26
const FILE_ROW_HEIGHT = 28
const TREE_INDENT_PX = 12

function label(f: FileStatus): string {
  return f.origPath ? `${f.origPath} → ${f.path}` : f.path
}

function treeLabel(f: FileStatus): string {
  return f.origPath ? `${baseName(f.origPath)} → ${baseName(f.path)}` : baseName(f.path)
}

const SECTION_LABEL: Record<WorkingTreeSection, string> = {
  conflicted: 'Conflicted',
  staged: 'Staged',
  unstaged: 'Unstaged',
}

// A leading dot (".env") isn't an extension.
function fileExtension(path: string): string | null {
  const name = baseName(path)
  const dot = name.lastIndexOf('.')
  return dot > 0 ? name.slice(dot + 1) : null
}

function WorkingTreeFileList({
  repoPath,
  workingTree,
  actions,
  openPath,
  openSection,
  onOpen,
}: WorkingTreeFileListProps) {
  const { files, conflicted, staged, unstaged, error, loadStatus } = workingTree
  const sectionFiles = useMemo<Record<WorkingTreeSection, FileStatus[]>>(
    () => ({ conflicted, staged, unstaged }),
    [conflicted, staged, unstaged],
  )
  const [viewMode, setViewMode] = useState<'flat' | 'tree'>('flat')
  const [filter, setFilter] = useState('')
  const [collapsedFolders, setCollapsedFolders] = useState<Set<string>>(new Set())
  const fileTools = useFileTools(repoPath)
  const [patchError, setPatchError] = useState<string | null>(null)
  const [patchBusy, setPatchBusy] = useState(false)
  // Multi-select for bulk actions; a selection never spans two sections.
  const [selectedPaths, setSelectedPaths] = useState<Set<string>>(new Set())
  const [selectionSection, setSelectionSection] = useState<WorkingTreeSection | null>(null)
  const [anchorPath, setAnchorPath] = useState<string | null>(null)
  const scrollRef = useRef<HTMLDivElement>(null)

  const toggleFolder = (section: WorkingTreeSection, path: string) => {
    const key = `${section}:${path}`
    setCollapsedFolders((prev) => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }

  const query = filter.trim().toLowerCase()
  const filtered = useMemo(() => {
    const match = (list: FileStatus[]) => (query ? list.filter((f) => f.path.toLowerCase().includes(query)) : list)
    return { conflicted: match(conflicted), staged: match(staged), unstaged: match(unstaged) }
  }, [conflicted, staged, unstaged, query])
  const hasConflicts = conflicted.length > 0

  // One flattened list, so every section shares a single virtualized scroll region.
  const listRows = useMemo<ListRow[]>(() => {
    const sectionRows = (section: WorkingTreeSection, sectionFiles: FileStatus[]): ListRow[] => {
      if (viewMode === 'flat') {
        return sectionFiles.map((file): ListRow => ({ kind: 'file', section, file, depth: 0 }))
      }
      return buildFileTreeRows(sectionFiles, (path) => collapsedFolders.has(`${section}:${path}`)).map(
        (row): ListRow =>
          row.kind === 'folder'
            ? { kind: 'folder', section, path: row.path, name: row.name, depth: row.depth }
            : { kind: 'file', section, file: row.file, depth: row.depth },
      )
    }
    // The conflicted section only exists while a merge, rebase or similar is stopped on conflicts.
    const conflictedRows: ListRow[] = hasConflicts
      ? [{ kind: 'header', section: 'conflicted' }, ...sectionRows('conflicted', filtered.conflicted)]
      : []
    return [
      ...conflictedRows,
      { kind: 'header', section: 'staged' },
      ...sectionRows('staged', filtered.staged),
      { kind: 'header', section: 'unstaged' },
      ...sectionRows('unstaged', filtered.unstaged),
    ]
  }, [hasConflicts, filtered, viewMode, collapsedFolders])

  const orderedSectionPaths = (section: WorkingTreeSection): string[] =>
    listRows
      .filter((r): r is Extract<ListRow, { kind: 'file' }> => r.kind === 'file' && r.section === section)
      .map((r) => r.file.path)

  // Drops selected paths that have since been staged, committed, or discarded.
  const effectiveSelectedPaths = useMemo(() => {
    if (!selectionSection) return new Set<string>()
    const valid = new Set(sectionFiles[selectionSection].map((f) => f.path))
    return new Set([...selectedPaths].filter((p) => valid.has(p)))
  }, [selectedPaths, selectionSection, sectionFiles])

  const rowVirtualizer = useVirtualizer({
    count: listRows.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: (i) => {
      const row = listRows[i]
      if (row.kind === 'header') return HEADER_ROW_HEIGHT
      if (row.kind === 'folder') return FOLDER_ROW_HEIGHT
      return FILE_ROW_HEIGHT
    },
    overscan: 10,
  })

  const runPatchExport = (exportPatch: Promise<void>) => {
    setPatchError(null)
    setPatchBusy(true)
    exportPatch.catch((err: unknown) => setPatchError(errorMessage(err))).finally(() => setPatchBusy(false))
  }

  const onFileRowClick = (section: WorkingTreeSection, f: FileStatus) => (e: MouseEvent) => {
    const path = f.path
    if (e.shiftKey && selectionSection === section && anchorPath) {
      const paths = orderedSectionPaths(section)
      const anchorIndex = paths.indexOf(anchorPath)
      const clickIndex = paths.indexOf(path)
      if (anchorIndex !== -1 && clickIndex !== -1) {
        const [start, end] = anchorIndex < clickIndex ? [anchorIndex, clickIndex] : [clickIndex, anchorIndex]
        setSelectedPaths(new Set(paths.slice(start, end + 1)))
      }
    } else if (e.ctrlKey || e.metaKey) {
      setSelectedPaths((prev) => {
        const next = new Set(selectionSection === section ? prev : [])
        if (next.has(path)) next.delete(path)
        else next.add(path)
        return next
      })
      setSelectionSection(section)
      setAnchorPath(path)
    } else {
      setSelectedPaths(new Set([path]))
      setSelectionSection(section)
      setAnchorPath(path)
    }

    if (section === 'staged' || f.worktreeStatus !== '?') {
      onOpen(path, section)
    }
  }

  const groupContextMenuItems = (section: WorkingTreeSection): ContextMenuItem[] => {
    const paths = [...effectiveSelectedPaths]
    const items: ContextMenuItem[] = [
      section === 'staged'
        ? { label: `Unstage ${paths.length} files`, onClick: () => actions.unstagePaths(paths) }
        : {
            label: section === 'conflicted' ? `Mark ${paths.length} files resolved` : `Stage ${paths.length} files`,
            onClick: () => actions.stagePaths(paths),
          },
      {
        label: 'Export patch…',
        onClick: () =>
          runPatchExport(PatchService.ExportWorkingTreePaths(repoPath, paths, suggestedWorkingTreePatchFilename())),
      },
      { label: `Copy ${paths.length} paths`, onClick: () => void navigator.clipboard.writeText(paths.join('\n')) },
    ]
    if (section !== 'conflicted') {
      const files = sectionFiles[section].filter((f) => effectiveSelectedPaths.has(f.path))
      items.push({ label: `Stash ${paths.length} files`, onClick: () => actions.stashFiles(files) })
    }
    if (section === 'unstaged') {
      items.push({ label: 'Discard', onClick: () => actions.discardPaths(paths), destructive: true })
    }
    return items
  }

  const rowContextMenu = (f: FileStatus, section: WorkingTreeSection) => (e: MouseEvent) => {
    if (selectionSection === section && effectiveSelectedPaths.size > 1 && effectiveSelectedPaths.has(f.path)) {
      fileTools.openMenu(e, groupContextMenuItems(section))
      return
    }
    const untracked = section === 'unstaged' && f.worktreeStatus === '?'
    const items: ContextMenuItem[] = [
      section === 'staged'
        ? { label: 'Unstage', onClick: () => actions.unstage(f) }
        : { label: section === 'conflicted' ? 'Mark resolved' : 'Stage', onClick: () => actions.stage(f) },
      ...fileTools.pathItems(f.path, !untracked),
    ]
    if (untracked) {
      items.push({ label: 'Add to .gitignore', onClick: () => actions.addToGitignore(f.path) })
      const ext = fileExtension(f.path)
      if (ext) {
        items.push({ label: `Add *.${ext} to .gitignore`, onClick: () => actions.addToGitignore(`*.${ext}`) })
      }
    }
    if (section !== 'conflicted') {
      items.push({ label: 'Stash', onClick: () => actions.stashFiles([f]) })
    }
    if (section === 'unstaged') {
      items.push({ label: 'Discard', onClick: () => actions.discardFile(f), destructive: true })
    }
    fileTools.openMenu(e, items)
  }

  const renderRow = (row: ListRow, style: CSSProperties) => {
    const isStaged = row.section === 'staged'

    if (row.kind === 'header') {
      const all = sectionFiles[row.section]
      return (
        <div key={`header-${row.section}`} className="changes-section-header" style={style}>
          <span className={`changes-section-label changes-section-label-${row.section}`}>
            {SECTION_LABEL[row.section]} <span className="changes-section-count">{filtered[row.section].length}</span>
          </span>
          {/* Marking every conflict resolved at once is too easy to do by accident. */}
          {row.section !== 'conflicted' && all.length > 0 && (
            <button
              type="button"
              className="changes-section-bulk"
              onClick={() => (isStaged ? actions.unstagePaths : actions.stagePaths)(all.map((f) => f.path))}
            >
              {isStaged ? 'Unstage all' : 'Stage all'}
            </button>
          )}
        </div>
      )
    }

    if (row.kind === 'folder') {
      const collapsed = collapsedFolders.has(`${row.section}:${row.path}`)
      return (
        <button
          key={`folder-${row.section}-${row.path}`}
          type="button"
          className="changes-folder-row"
          style={{ ...style, paddingLeft: `calc(var(--space-sm) + ${row.depth * TREE_INDENT_PX}px)` }}
          onClick={() => toggleFolder(row.section, row.path)}
        >
          {collapsed ? <ChevronRight size={14} strokeWidth={1.5} /> : <ChevronDown size={14} strokeWidth={1.5} />}
          <Folder size={14} strokeWidth={1.5} className="changes-folder-row-icon" />
          <span className="changes-folder-row-name">{row.name}</span>
        </button>
      )
    }

    const f = row.file
    return (
      <FileRow
        // A partially staged file appears in both sections.
        key={`${row.section}-${f.path}`}
        style={style}
        indent={row.depth * TREE_INDENT_PX}
        status={isStaged ? f.indexStatus : f.worktreeStatus}
        label={viewMode === 'tree' ? treeLabel(f) : label(f)}
        selected={
          selectionSection === row.section
            ? effectiveSelectedPaths.has(f.path)
            : openPath === f.path && openSection === row.section
        }
        checked={isStaged}
        checkLabel={row.section === 'conflicted' ? `Mark ${label(f)} resolved` : undefined}
        added={isStaged ? f.indexAdded : f.workAdded}
        removed={isStaged ? f.indexRemoved : f.workRemoved}
        binary={isStaged ? f.indexBinary : f.workBinary}
        tag={f.submodule ? 'submodule' : undefined}
        onToggleChecked={() => (isStaged ? actions.unstage(f) : actions.stage(f))}
        onSelect={onFileRowClick(row.section, f)}
        onDiscard={row.section === 'unstaged' ? () => actions.discardFile(f) : undefined}
        onContextMenu={rowContextMenu(f, row.section)}
      />
    )
  }

  return (
    <>
      <div className="changes-files-header">
        <span className="changes-files-title">Working tree</span>
        <div className="changes-files-header-spacer" />
        <SegmentedControl
          value={viewMode}
          onChange={setViewMode}
          options={[
            { value: 'flat', label: 'List', icon: List },
            { value: 'tree', label: 'Tree', icon: FolderTree },
          ]}
        />
        <button
          type="button"
          className="changes-export-patch"
          onClick={() => runPatchExport(PatchService.ExportWorkingTree(repoPath, suggestedWorkingTreePatchFilename()))}
          disabled={patchBusy || (files?.length ?? 0) === 0}
          aria-label="Export patch"
          title="Export working tree as a patch"
        >
          <FileDown size={18} strokeWidth={1.75} />
        </button>
        <button type="button" className="changes-refresh" onClick={loadStatus} aria-label="Refresh" title="Refresh">
          <RefreshCw size={18} strokeWidth={1.75} />
        </button>
      </div>

      <div className="changes-filter-wrap">
        <Search size={14} strokeWidth={1.5} className="changes-filter-icon" />
        <input
          className="changes-filter-input"
          type="search"
          placeholder="Filter files…"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          aria-label="Filter working tree files"
        />
      </div>

      {actions.actionError && <p className="changes-error">Could not update changes: {actions.actionError}</p>}
      {patchError && <p className="changes-error">Could not export patch: {patchError}</p>}

      {/* Stays mounted when empty so the commit composer remains pinned to the bottom. */}
      <div ref={scrollRef} className="changes-files-scroll">
        {error ? (
          <p className="changes-empty">Could not load changes: {error}</p>
        ) : files === null ? (
          <p className="changes-empty">Loading changes…</p>
        ) : files.length === 0 ? (
          <p className="changes-empty">No changes.</p>
        ) : (
          <div style={{ height: rowVirtualizer.getTotalSize(), position: 'relative', width: '100%' }}>
            {rowVirtualizer.getVirtualItems().map((vi) =>
              renderRow(listRows[vi.index], {
                position: 'absolute',
                top: 0,
                left: 0,
                width: '100%',
                height: vi.size,
                transform: `translateY(${vi.start}px)`,
              }),
            )}
          </div>
        )}
      </div>

      {fileTools.overlays}
    </>
  )
}

export default WorkingTreeFileList
