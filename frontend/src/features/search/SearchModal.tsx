import { useRef, useState } from 'react'
import { useDebounceValue } from 'usehooks-ts'
import type { CancellablePromise } from '@wailsio/runtime'
import { PlatformService, SearchService, type SearchMatch, type SearchResult } from '@current-client-bindings/app'
import Modal from '../../components/chrome/Modal'
import Checkbox from '../../components/forms/Checkbox'
import PathText from '../../components/git/PathText'
import RefSelect, { useRefGroups } from '../../components/git/RefSelect'
import { errorMessage } from '../../lib/errors'
import { plural } from '../../lib/plural'
import { useAsyncData } from '../../lib/useAsyncData'
import { matchRanges, type SearchFlags } from './matchRanges'
import SegmentedControl from '../../components/controls/SegmentedControl'
import { useSearchElsewhere } from '../../lib/useSearchElsewhere'
import { baseName } from '../../lib/paths'
import './SearchModal.scss'

const SEARCH_DEBOUNCE_MS = 250

interface SearchModalProps {
  repoPath: string
  // The other open repositories, which a search can take in too.
  others?: string[]
  onClose: () => void
  // A past commit's file can't open in the editor at that version, so it opens in File history.
  onOpenFileHistory: (path: string) => void
}

function Highlighted({ text, pattern, flags }: { text: string; pattern: string; flags: SearchFlags }) {
  const parts = []
  let at = 0
  for (const [start, end] of matchRanges(text, pattern, flags)) {
    parts.push(
      text.slice(at, start),
      <mark key={start} className="search-match">
        {text.slice(start, end)}
      </mark>,
    )
    at = end
  }
  parts.push(text.slice(at))
  return <>{parts}</>
}

function groupByPath(matches: SearchMatch[]): [string, SearchMatch[]][] {
  const groups = new Map<string, SearchMatch[]>()
  for (const m of matches) groups.set(m.path, [...(groups.get(m.path) ?? []), m])
  return [...groups]
}

function SearchModal({ repoPath, others = [], onClose, onOpenFileHistory }: SearchModalProps) {
  const [pattern, setPattern] = useState('')
  const [flags, setFlags] = useState<SearchFlags>({ ignoreCase: true, wholeWord: false, regexp: false })
  const [scope, setScope] = useState<'repository' | 'open'>('repository')
  const [chosenRev, setRev] = useState('')
  // Every open repository is searched in its working tree.
  const rev = scope === 'open' ? '' : chosenRev
  const [openError, setOpenError] = useState<string | null>(null)
  const [query] = useDebounceValue(pattern, SEARCH_DEBOUNCE_MS)
  const refGroups = useRefGroups(repoPath)
  const running = useRef<CancellablePromise<SearchResult> | null>(null)

  const { data, error, loading } = useAsyncData(() => {
    void running.current?.cancel()
    running.current = query ? SearchService.SearchContents(repoPath, { pattern: query, rev, ...flags }) : null
    return running.current
  }, [repoPath, query, rev, flags])

  const elsewhere = useSearchElsewhere(
    scope === 'open' ? others : [],
    query && `${query}\0${JSON.stringify(flags)}`,
    (path) => SearchService.SearchContents(path, { pattern: query, rev: '', ...flags }),
  )

  const setFlag = (name: keyof SearchFlags) => (value: boolean) => setFlags((prev) => ({ ...prev, [name]: value }))

  const open = (m: SearchMatch, path = repoPath) => {
    if (rev) {
      onOpenFileHistory(m.path)
      return
    }
    setOpenError(null)
    PlatformService.OpenFileInEditor(path, m.path, m.line).catch((err: unknown) => setOpenError(errorMessage(err)))
  }

  const fileGroups = (matches: SearchMatch[], path = repoPath) =>
    groupByPath(matches).map(([file, lines]) => (
      <section key={`${path}\0${file}`} className="search-file">
        <PathText path={file} className="search-file-path" />
        <ul className="search-lines">
          {lines.map((m) => (
            <li key={m.line}>
              <button type="button" className="search-line" onClick={() => open(m, path)}>
                <span className="search-line-number">{m.line}</span>
                <span className="search-line-text">
                  <Highlighted text={m.text} pattern={query} flags={flags} />
                </span>
              </button>
            </li>
          ))}
        </ul>
      </section>
    ))

  const groups = groupByPath(data?.matches ?? [])

  return (
    <Modal title="Search in files" onClose={onClose} className="search-panel">
      <div className="search-controls">
        <input
          className="search-input"
          aria-label="Search for"
          placeholder="Search for"
          value={pattern}
          onChange={(e) => setPattern(e.target.value)}
          autoFocus
        />
        <div className="search-options">
          <Checkbox checked={!flags.ignoreCase} onChange={(v) => setFlag('ignoreCase')(!v)} label="Match case" />
          <Checkbox checked={flags.wholeWord} onChange={setFlag('wholeWord')} label="Whole word" />
          <Checkbox checked={flags.regexp} onChange={setFlag('regexp')} label="Regular expression" />
          {scope === 'repository' && (
            <RefSelect
              groups={refGroups}
              value={rev}
              onChange={setRev}
              label="Search in"
              extraOptions={[{ value: '', label: 'Working tree' }]}
            />
          )}
        </div>
        {others.length > 0 && (
          <SegmentedControl
            value={scope}
            onChange={setScope}
            options={[
              { value: 'repository', label: 'This repository' },
              { value: 'open', label: `Open repositories (${others.length + 1})` },
            ]}
          />
        )}
      </div>

      <div className="search-results">
        {error ? (
          <p className="search-error">{error}</p>
        ) : !query ? (
          <p className="search-hint">Type to search the contents of every file.</p>
        ) : !data ? (
          <p className="search-hint">Searching…</p>
        ) : groups.length === 0 ? (
          <p className="search-hint">{scope === 'open' ? `No matches in ${baseName(repoPath)}.` : 'No matches.'}</p>
        ) : (
          <>
            <p className="search-summary" role="status">
              {data.truncated
                ? `Showing the first ${data.matches.length} matches. Narrow the search to see more.`
                : `${plural(data.matches.length, 'match', 'matches')} in ${plural(groups.length, 'file')}`}
              {loading && ' · Searching…'}
            </p>
            {openError && <p className="search-error">Could not open: {openError}</p>}
            {fileGroups(data.matches)}
          </>
        )}
        {scope === 'open' &&
          elsewhere.found
            .filter(({ matches }) => matches.matches.length > 0)
            .map(({ path, matches }) => (
              <section key={path} className="search-repository" aria-label={`In ${baseName(path)}`}>
                <h3 className="search-repository-name">
                  In {baseName(path)} · {plural(matches.matches.length, 'match', 'matches')}
                  {matches.truncated && ', the first only'}
                </h3>
                {fileGroups(matches.matches, path)}
              </section>
            ))}
        {scope === 'open' && query && elsewhere.waiting > 0 && (
          <p className="search-hint">Searching {plural(elsewhere.waiting, 'more repository', 'more repositories')}…</p>
        )}
      </div>
    </Modal>
  )
}

export default SearchModal
