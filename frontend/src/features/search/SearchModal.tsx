import { useRef, useState } from 'react'
import { useDebounceValue } from 'usehooks-ts'
import type { CancellablePromise } from '@wailsio/runtime'
import { PlatformService, SearchService, type SearchMatch, type SearchResult } from '@current-client-bindings/app'
import Modal from '../../components/chrome/Modal'
import Checkbox from '../../components/forms/Checkbox'
import PathText from '../../components/git/PathText'
import RefSelect, { useRefGroups } from '../../components/git/RefSelect'
import { errorMessage } from '../../lib/errors'
import { useAsyncData } from '../../lib/useAsyncData'
import { matchRanges, type SearchFlags } from './matchRanges'
import './SearchModal.scss'

const SEARCH_DEBOUNCE_MS = 250

interface SearchModalProps {
  repoPath: string
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

function SearchModal({ repoPath, onClose, onOpenFileHistory }: SearchModalProps) {
  const [pattern, setPattern] = useState('')
  const [flags, setFlags] = useState<SearchFlags>({ ignoreCase: true, wholeWord: false, regexp: false })
  const [rev, setRev] = useState('')
  const [openError, setOpenError] = useState<string | null>(null)
  const [query] = useDebounceValue(pattern, SEARCH_DEBOUNCE_MS)
  const refGroups = useRefGroups(repoPath)
  const running = useRef<CancellablePromise<SearchResult> | null>(null)

  const { data, error, loading } = useAsyncData(() => {
    void running.current?.cancel()
    running.current = query ? SearchService.SearchContents(repoPath, { pattern: query, rev, ...flags }) : null
    return running.current
  }, [repoPath, query, rev, flags])

  const setFlag = (name: keyof SearchFlags) => (value: boolean) => setFlags((prev) => ({ ...prev, [name]: value }))

  const open = (m: SearchMatch) => {
    if (rev) {
      onOpenFileHistory(m.path)
      return
    }
    setOpenError(null)
    PlatformService.OpenFileInEditor(repoPath, m.path, m.line).catch((err: unknown) => setOpenError(errorMessage(err)))
  }

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
          <RefSelect
            groups={refGroups}
            value={rev}
            onChange={setRev}
            label="Search in"
            extraOptions={[{ value: '', label: 'Working tree' }]}
          />
        </div>
      </div>

      <div className="search-results">
        {error ? (
          <p className="search-error">{error}</p>
        ) : !query ? (
          <p className="search-hint">Type to search the contents of every file.</p>
        ) : !data ? (
          <p className="search-hint">Searching…</p>
        ) : groups.length === 0 ? (
          <p className="search-hint">No matches.</p>
        ) : (
          <>
            <p className="search-summary" role="status">
              {data.truncated
                ? `Showing the first ${data.matches.length} matches. Narrow the search to see more.`
                : `${data.matches.length} ${data.matches.length === 1 ? 'match' : 'matches'} in ${groups.length} ${groups.length === 1 ? 'file' : 'files'}`}
              {loading && ' · Searching…'}
            </p>
            {openError && <p className="search-error">Could not open: {openError}</p>}
            {groups.map(([path, matches]) => (
              <section key={path} className="search-file">
                <PathText path={path} className="search-file-path" />
                <ul className="search-lines">
                  {matches.map((m) => (
                    <li key={m.line}>
                      <button type="button" className="search-line" onClick={() => open(m)}>
                        <span className="search-line-number">{m.line}</span>
                        <span className="search-line-text">
                          <Highlighted text={m.text} pattern={query} flags={flags} />
                        </span>
                      </button>
                    </li>
                  ))}
                </ul>
              </section>
            ))}
          </>
        )}
      </div>
    </Modal>
  )
}

export default SearchModal
