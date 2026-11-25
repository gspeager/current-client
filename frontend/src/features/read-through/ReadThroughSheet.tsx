import { useEffect, useMemo, useRef, useState } from 'react'
import { RefreshCw } from 'lucide-react'
import {
  ReadThroughInfo,
  ReadThroughRequest,
  ReadThroughService,
  type ReadThroughFile,
} from '@current-client-bindings/app'
import DiffSheet from '../../components/chrome/DiffSheet'
import { Fact, FactList } from '../../components/chrome/FactList'
import Checkbox from '../../components/forms/Checkbox'
import ChangedFileDiff from '../../components/git/ChangedFileDiff'
import PathText from '../../components/git/PathText'
import RefSelect, { useRefGroups } from '../../components/git/RefSelect'
import ChangedFileRow from '../history/ChangedFileRow'
import { plural } from '../../lib/plural'
import { useActionError } from '../../lib/useActionError'
import { useAsyncData } from '../../lib/useAsyncData'
import { useRovingRows } from '../../lib/useRovingRows'
import { groupFiles, totalsByKind, totalsByStatus, totalsByType, type Total } from './groupFiles'
import type { ReadThroughTarget } from './readThrough'
import './ReadThroughSheet.scss'

const versionOf = (file: ReadThroughFile) => `${file.newBlob}:${file.path}`

interface ReadThroughSheetProps {
  repoPath: string
  request: Partial<ReadThroughTarget>
  refreshKey: unknown
  onClose: () => void
}

function ReadThroughSheet({ repoPath, request, refreshKey, onClose }: ReadThroughSheetProps) {
  const [target, setTarget] = useState(request.target ?? '')
  const [uncommitted, setUncommitted] = useState(request.uncommitted ?? true)
  const {
    data: info,
    error,
    reload,
  } = useAsyncData(
    () => ReadThroughService.Open(repoPath, new ReadThroughRequest({ ...request, target, uncommitted })),
    [repoPath, request, target, uncommitted],
    { refreshKey, keepData: true },
  )
  const refGroups = useRefGroups(repoPath)
  const { error: actionError, setError: setActionError, fail } = useActionError()
  // Null is the overview.
  const [path, setPath] = useState<string | null>(null)
  // Ticks shown before they're saved.
  const [ticking, setTicking] = useState<ReadonlyMap<string, boolean>>(new Map())
  const rows = useRovingRows('.read-through-row')

  // Another worktree isn't watched, so what changed there shows on coming back.
  const reloadRef = useRef(reload)
  reloadRef.current = reload
  useEffect(() => {
    const onFocus = () => reloadRef.current()
    window.addEventListener('focus', onFocus)
    return () => window.removeEventListener('focus', onFocus)
  }, [])

  const files = useMemo(
    () => (info?.files ?? []).map((f) => ({ ...f, read: ticking.get(versionOf(f)) ?? f.read })),
    [info, ticking],
  )
  const groups = useMemo(() => groupFiles(files), [files])
  const ordered = useMemo(() => groups.flatMap((group) => group.files), [groups])
  const selected = files.find((f) => f.path === path) ?? null
  const readCount = files.filter((f) => f.read).length

  // Without its files, which every call would send back.
  const range = info && new ReadThroughInfo({ ...info, files: [] })
  const setRead = (file: ReadThroughFile, read: boolean) => {
    if (!range) return
    setActionError(null)
    const tick = (value: boolean) => setTicking((prev) => new Map(prev).set(versionOf(file), value))
    tick(read)
    ReadThroughService.SetRead(range, [file], read)
      .catch((err: unknown) => {
        tick(!read)
        fail(err)
      })
      .finally(reload)
  }
  const nextUnread = () => {
    const at = ordered.findIndex((f) => f.path === path)
    const after = [...ordered.slice(at + 1), ...ordered.slice(0, Math.max(0, at))]
    setPath(after.find((f) => !f.read)?.path ?? null)
  }
  const unreadElsewhere = files.some((f) => !f.read && f.path !== path)

  return (
    <DiffSheet
      title="Read through"
      onClose={onClose}
      headerContent={
        <div className="read-through-header">
          <span className="read-through-name" title={info?.name}>
            {info?.name}
          </span>
          <span className="read-through-against">against</span>
          <RefSelect
            groups={refGroups}
            value={info?.target ?? target}
            onChange={setTarget}
            label="Compare with"
            extraOptions={info?.target ? [] : [{ value: '', label: 'Choose a branch' }]}
          />
          {info?.checkedOut && <Checkbox checked={uncommitted} onChange={setUncommitted} label="Uncommitted changes" />}
          <span className="read-through-progress" role="status">
            {info && files.length > 0 && `${readCount} of ${files.length} read`}
          </span>
          <button type="button" className="read-through-refresh" onClick={reload} aria-label="Refresh" title="Refresh">
            <RefreshCw size={14} strokeWidth={1.75} />
          </button>
        </div>
      }
    >
      <div className="changed-files-browser">
        <div className="changed-files-browser-files read-through-files">
          {error ? (
            <p className="changed-files-browser-error">{error}</p>
          ) : info === null ? (
            <p className="changed-files-browser-hint">Loading changed files…</p>
          ) : info.target === '' ? (
            <p className="changed-files-browser-hint">This repository has no default branch to compare with.</p>
          ) : files.length === 0 ? (
            <p className="changed-files-browser-hint">
              No changes against {info.target}
              {info.uncommitted ? ', committed or not' : ''}.
            </p>
          ) : (
            <div ref={rows}>
              <div className="read-through-row">
                <button
                  type="button"
                  className={path === null ? 'changed-file changed-file-selected' : 'changed-file'}
                  aria-pressed={path === null}
                  onClick={() => setPath(null)}
                >
                  Overview
                </button>
              </div>
              {groups.map((group) => (
                <section key={group.folder} aria-label={group.folder || 'Top folder'}>
                  <h3 className="read-through-folder">
                    <PathText path={group.folder || 'Top folder'} className="read-through-folder-path" />
                    <span className="changed-files-browser-count">{group.files.length}</span>
                  </h3>
                  <ul className="changed-file-list">
                    {group.files.map((file) => (
                      <ChangedFileRow
                        key={file.path}
                        file={file}
                        selected={file.path === path}
                        onSelect={() => setPath(file.path)}
                        relativeTo={group.folder}
                        read={file.read}
                        onRead={(read) => setRead(file, read)}
                      />
                    ))}
                  </ul>
                </section>
              ))}
            </div>
          )}
        </div>
        <div className="changed-files-browser-diff">
          {actionError && <p className="changed-files-browser-error">{actionError}</p>}
          {range && selected ? (
            <>
              <div className="read-through-file">
                <PathText
                  path={selected.origPath ? `${selected.origPath} → ${selected.path}` : selected.path}
                  className="read-through-file-path"
                />
                <Checkbox checked={selected.read} onChange={(read) => setRead(selected, read)} label="Read" />
                <button type="button" className="read-through-next" disabled={!unreadElsewhere} onClick={nextUnread}>
                  Next unread
                </button>
              </div>
              <ChangedFileDiff
                file={selected}
                version={`${range.base}:${range.head}:${selected.oldBlob}:${selected.newBlob}`}
                loadDiff={(file, force, ignoreWhitespace) =>
                  ReadThroughService.FileDiff(range, file.origPath, file.path, force, ignoreWhitespace)
                }
                images={{
                  repoPath: range.path,
                  before: { kind: 'commit', rev: range.base },
                  after: range.uncommitted ? { kind: 'worktree', rev: '' } : { kind: 'commit', rev: range.head },
                }}
              />
            </>
          ) : (
            info && files.length > 0 && <Overview info={info} files={files} />
          )}
        </div>
      </div>
    </DiffSheet>
  )
}

function Overview({ info, files }: { info: ReadThroughInfo; files: ReadThroughFile[] }) {
  const sum = (count: (file: ReadThroughFile) => number) => files.reduce((total, file) => total + count(file), 0)
  const kinds = totalsByKind(files)
  return (
    <div className="read-through-overview">
      <FactList>
        <Fact label="Files">{plural(files.length, 'file')}</Fact>
        <Fact label="Lines" className="fact-list-data">
          +{sum((f) => f.added)} -{sum((f) => f.removed)}
        </Fact>
        <Fact label="Status">
          <Totals totals={totalsByStatus(files)} />
        </Fact>
        <Fact label="Types" className="fact-list-data read-through-wrapping">
          <Totals totals={totalsByType(files)} />
        </Fact>
        {kinds.length > 0 && (
          <Fact label="Of them">
            <Totals totals={kinds} />
          </Fact>
        )}
        {info.skipped > 0 && (
          <Fact label="Left out">{plural(info.skipped, 'large untracked file')}, too large to compare</Fact>
        )}
      </FactList>
    </div>
  )
}

function Totals({ totals }: { totals: Total[] }) {
  return <>{totals.map(({ label, files }) => `${label} ${files}`).join(' · ')}</>
}

export default ReadThroughSheet
