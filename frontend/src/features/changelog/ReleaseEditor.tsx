import { useCallback, useMemo, useRef, useState } from 'react'
import { useVirtualizer } from '@tanstack/react-virtual'
import { Check, ChevronDown, Code, Copy, Eye } from 'lucide-react'
import {
  ChangelogService,
  type ChangelogEntry,
  type ChangelogMarkdownOptions,
  type ChangelogReleaseSection,
} from '@current-client-bindings/app'
import SegmentedControl from '../../components/controls/SegmentedControl'
import SplitButton from '../../components/controls/SplitButton'
import Checkbox from '../../components/forms/Checkbox'
import { changelogPdfBase64 } from './changelogPdf'
import { COMMIT_TYPES } from '../../lib/conventionalCommit'
import { errorMessage } from '../../lib/errors'
import { useAsyncData } from '../../lib/useAsyncData'
import { useCopyToClipboard } from '../../lib/useCopyToClipboard'
import { useDialogs } from '../../lib/useDialogs'
import { useDismissibleDetails } from '../../lib/useDismissibleDetails'

// A commit's kind is what the type filter toggles: breaking changes of any type
// together, then each type, then commits that don't follow the format.
const BREAKING = 'breaking'
const OTHER = 'other'

const kindOf = (e: ChangelogEntry) => (e.section === 'Breaking' ? BREAKING : e.type || OTHER)

function kindRank(kind: string): number {
  if (kind === BREAKING) return -1
  if (kind === OTHER) return Number.MAX_SAFE_INTEGER
  const i = COMMIT_TYPES.indexOf(kind)
  return i === -1 ? COMMIT_TYPES.length : i
}

function countBy(entries: ChangelogEntry[], key: (e: ChangelogEntry) => string): [string, number][] {
  const counts = new Map<string, number>()
  for (const e of entries) counts.set(key(e), (counts.get(key(e)) ?? 0) + 1)
  return [...counts]
}

function toggled(set: ReadonlySet<string>, value: string, on: boolean): ReadonlySet<string> {
  const next = new Set(set)
  if (on) next.add(value)
  else next.delete(value)
  return next
}

export interface ChangelogFilters {
  kinds: ReadonlySet<string>
  hiddenAuthors: ReadonlySet<string>
}
const UNRELEASED = 'Unreleased'

type MarkdownMode = 'markdown' | 'preview'

type ListRow =
  | { kind: 'release'; key: string; release: ChangelogReleaseSection }
  | { kind: 'section'; key: string; section: string }
  | { kind: 'entry'; key: string; entry: ChangelogEntry }

// Starting heights for the virtualizer; rows are measured once rendered.
const ROW_ESTIMATES: Record<ListRow['kind'], number> = { release: 44, section: 32, entry: 28 }

function today(): string {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

interface ReleaseEditorProps {
  repoPath: string
  releases: ChangelogReleaseSection[]
  options: ChangelogMarkdownOptions
  onOptionsChange: (options: ChangelogMarkdownOptions) => void
  mode: MarkdownMode
  onModeChange: (mode: MarkdownMode) => void
  filters: ChangelogFilters
  onFiltersChange: (filters: ChangelogFilters) => void
  onOpenCommit: (sha: string) => void
}

function ReleaseEditor({
  repoPath,
  releases,
  options,
  onOptionsChange,
  mode,
  onModeChange,
  filters,
  onFiltersChange,
  onOpenCommit,
}: ReleaseEditorProps) {
  const { confirm } = useDialogs()
  const { copied, copy } = useCopyToClipboard()
  const [excluded, setExcluded] = useState<ReadonlySet<string>>(new Set())
  // Only the commits after the newest tag take a chosen version; tagged
  // releases are named and dated by their tag.
  const untagged = releases.find((r) => !r.tag)
  const [version, setVersion] = useState(untagged?.suggestedVersion || UNRELEASED)
  const [date] = useState(today)
  const [draft, setDraft] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [menuOpen, setMenuOpen] = useState(false)

  const allEntries = useMemo(() => releases.flatMap((r) => r.entries), [releases])
  const isShown = useCallback(
    (e: ChangelogEntry) => filters.kinds.has(kindOf(e)) && !filters.hiddenAuthors.has(e.author),
    [filters],
  )
  const visibleCount = allEntries.filter(isShown).length
  const kinds = useMemo(
    () => countBy(allEntries, kindOf).sort(([a], [b]) => kindRank(a) - kindRank(b) || a.localeCompare(b)),
    [allEntries],
  )
  const authors = useMemo(
    () => countBy(allEntries, (e) => e.author).sort(([a, x], [b, y]) => y - x || a.localeCompare(b)),
    [allEntries],
  )
  const heading = (r: ChangelogReleaseSection) => (r.tag ? r.version : version.trim())
  // Releases with nothing left after filtering are dropped, unless that leaves none.
  const markdownSections = useMemo(() => {
    const sections = releases.map((r) => ({
      version: r.tag ? r.version : version.trim(),
      date: r.tag ? r.date : date,
      entries: r.entries.filter((e) => isShown(e) && !excluded.has(e.sha)),
    }))
    const nonEmpty = sections.filter((s) => s.entries.length > 0)
    return nonEmpty.length > 0 ? nonEmpty : sections.slice(0, 1)
  }, [releases, version, date, isShown, excluded])
  const {
    data: generated,
    loading: regenerating,
    error: markdownError,
  } = useAsyncData(() => ChangelogService.Markdown(markdownSections, options), [markdownSections, options], {
    keepData: true,
  })
  const markdown = draft ?? generated ?? ''
  // The old Markdown stays on screen while it regenerates, so nothing may copy
  // or write it until the new text arrives. A hand-edited draft is never stale.
  const stale = draft === null && regenerating
  const {
    data: previewHtml,
    loading: rendering,
    error: previewError,
  } = useAsyncData(() => (mode === 'preview' ? ChangelogService.RenderHTML(markdown) : null), [mode, markdown], {
    keepData: true,
  })
  const [busy, setBusy] = useState<string | null>(null)
  const blocked = stale || busy !== null || !markdown
  const renderError = markdownError ?? (mode === 'preview' ? previewError : null)
  const status = busy ?? (stale ? 'Updating…' : null)

  // Regenerating replaces the Markdown, so hand edits are only dropped on confirmation.
  const regenerate = async (change: () => void) => {
    if (
      draft !== null &&
      !(await confirm({
        title: 'Discard Markdown edits',
        message: 'Regenerating the Markdown discards the edits made to it.',
        confirmLabel: 'Discard edits',
        destructive: true,
      }))
    ) {
      return
    }
    setDraft(null)
    change()
  }

  const toggle = (sha: string, include: boolean) =>
    void regenerate(() =>
      setExcluded((prev) => {
        const next = new Set(prev)
        if (include) next.delete(sha)
        else next.add(sha)
        return next
      }),
    )

  // action resolves to the notice to show, or null when it was cancelled.
  const run = async (working: string, failure: string, action: () => Promise<string | null>) => {
    setNotice(null)
    setActionError(null)
    setMenuOpen(false)
    setBusy(working)
    try {
      setNotice(await action())
    } catch (err) {
      setActionError(`${failure}: ${errorMessage(err)}`)
    } finally {
      setBusy(null)
    }
  }

  const write = () =>
    run('Writing CHANGELOG.md…', 'Could not write CHANGELOG.md', async () => {
      const existing = (await ChangelogService.ExistingVersions(repoPath, markdown)) ?? []
      if (
        existing.length > 0 &&
        !(await confirm({
          title: 'Replace changelog sections',
          message: `CHANGELOG.md already has ${existing.length === 1 ? 'a section' : 'sections'} for ${existing.join(', ')}. Replace ${existing.length === 1 ? 'it' : 'them'}?`,
          confirmLabel: 'Replace',
          destructive: true,
        }))
      ) {
        return null
      }
      await ChangelogService.Write(repoPath, markdown, existing.length > 0)
      return 'Written to CHANGELOG.md. Commit it from Working Copy.'
    })

  const newest = markdownSections[0]?.version || UNRELEASED
  const oldest = markdownSections[markdownSections.length - 1]?.version || UNRELEASED
  const fileBase = newest === oldest ? `CHANGELOG-${newest}` : `CHANGELOG-${oldest}-to-${newest}`
  const saved = (ok: boolean) => (ok ? 'Saved.' : null)
  const saveMarkdown = () =>
    run('Saving…', 'Could not save', async () => saved(await ChangelogService.SaveMarkdown(markdown, `${fileBase}.md`)))
  const saveHTML = () =>
    run('Saving…', 'Could not save', async () => saved(await ChangelogService.SaveHTML(markdown, `${fileBase}.html`)))
  const savePDF = () =>
    run('Preparing PDF…', 'Could not save', async () => {
      const pdf = await changelogPdfBase64(await ChangelogService.RenderHTML(markdown))
      return saved(await ChangelogService.SavePDF(pdf, `${fileBase}.pdf`))
    })

  // One flat list of headings and commits, so tens of thousands of commits can
  // be virtualized like History.
  const rows = useMemo(() => {
    const list: ListRow[] = []
    for (const r of releases) {
      const shown = r.entries.filter(isShown)
      if (shown.length === 0) continue
      if (releases.length > 1) list.push({ kind: 'release', key: `release-${r.tag}`, release: r })
      for (const section of r.sections) {
        const entries = shown.filter((e) => e.section === section)
        if (entries.length === 0) continue
        list.push({ kind: 'section', key: `section-${r.tag}-${section}`, section })
        for (const entry of entries) list.push({ kind: 'entry', key: entry.sha, entry })
      }
    }
    return list
  }, [releases, isShown])
  const listRef = useRef<HTMLDivElement>(null)
  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => listRef.current,
    estimateSize: (i) => ROW_ESTIMATES[rows[i].kind],
    getItemKey: (i) => rows[i].key,
    overscan: 10,
  })

  const entryRow = (entry: ChangelogEntry) => (
    <div className="changelog-entry">
      <Checkbox
        checked={!excluded.has(entry.sha)}
        onChange={(include) => toggle(entry.sha, include)}
        ariaLabel={`Include ${entry.description}`}
      />
      <span className="changelog-entry-text">
        {entry.scope && <span className="changelog-entry-scope">{entry.scope}: </span>}
        {entry.description}
      </span>
      <button
        type="button"
        className="changelog-entry-sha"
        onClick={() => onOpenCommit(entry.sha)}
        title="Open in History"
      >
        {entry.sha.slice(0, 7)}
      </button>
    </div>
  )

  return (
    <div className="changelog-editor">
      <div className="changelog-entries">
        <div className="changelog-filters">
          <FilterMenu
            label="Types"
            items={kinds}
            mono
            isShown={(kind) => filters.kinds.has(kind)}
            onToggle={(kind, on) =>
              void regenerate(() => onFiltersChange({ ...filters, kinds: toggled(filters.kinds, kind, on) }))
            }
          />
          <FilterMenu
            label="Authors"
            items={authors}
            isShown={(author) => !filters.hiddenAuthors.has(author)}
            onToggle={(author, on) =>
              void regenerate(() =>
                onFiltersChange({ ...filters, hiddenAuthors: toggled(filters.hiddenAuthors, author, !on) }),
              )
            }
          />
        </div>
        {visibleCount === 0 && <p className="changelog-hint">Every commit in this range is filtered out.</p>}
        <div ref={listRef} className="changelog-entry-list">
          <div style={{ height: virtualizer.getTotalSize(), position: 'relative' }}>
            {virtualizer.getVirtualItems().map((item) => {
              const row = rows[item.index]
              return (
                <div
                  key={item.key}
                  ref={virtualizer.measureElement}
                  data-index={item.index}
                  className="changelog-row"
                  style={{ transform: `translateY(${item.start}px)` }}
                >
                  {row.kind === 'release' ? (
                    <h2 className="changelog-release-heading">
                      {heading(row.release)}
                      {row.release.date && <span className="changelog-date">{row.release.date}</span>}
                    </h2>
                  ) : row.kind === 'section' ? (
                    <h3 className="changelog-label changelog-section-heading">{row.section}</h3>
                  ) : (
                    entryRow(row.entry)
                  )}
                </div>
              )
            })}
          </div>
        </div>
      </div>

      <div className="changelog-output">
        <div className="changelog-output-toolbar">
          {untagged && (
            <>
              <label className="changelog-label" htmlFor="changelog-version">
                Version
              </label>
              <input
                id="changelog-version"
                className="changelog-version"
                list="changelog-versions"
                value={version}
                onChange={(e) => {
                  const value = e.target.value
                  void regenerate(() => setVersion(value))
                }}
              />
              <datalist id="changelog-versions">
                {untagged.suggestedVersion && <option value={untagged.suggestedVersion} />}
                <option value={UNRELEASED} />
              </datalist>
              {version.trim() !== UNRELEASED && <span className="changelog-date">{date}</span>}
            </>
          )}
          <Checkbox
            checked={options.authors}
            onChange={(authors) => void regenerate(() => onOptionsChange({ ...options, authors }))}
            label="Author names"
          />
          <Checkbox
            checked={options.dates}
            onChange={(dates) => void regenerate(() => onOptionsChange({ ...options, dates }))}
            label="Dates"
          />
          <span className="changelog-spacer" />
          {status && (
            <span className="changelog-status" role="status">
              {status}
            </span>
          )}
          <SegmentedControl
            value={mode}
            onChange={onModeChange}
            options={[
              { value: 'markdown', label: 'Markdown', icon: Code },
              { value: 'preview', label: 'Preview', icon: Eye },
            ]}
          />
          <button
            type="button"
            className="changelog-icon-button"
            onClick={() => copy(markdown)}
            disabled={stale || !markdown}
            aria-label="Copy Markdown"
            title="Copy Markdown"
          >
            {copied ? <Check size={14} strokeWidth={1.75} /> : <Copy size={14} strokeWidth={1.5} />}
          </button>
          <SplitButton
            onClick={() => void write()}
            disabled={blocked || (untagged !== undefined && !version.trim())}
            menuLabel="More save options"
            open={menuOpen}
            onOpenChange={setMenuOpen}
            menu={
              <>
                <button type="button" onClick={() => void saveMarkdown()} disabled={blocked}>
                  Save Markdown…
                </button>
                <button type="button" onClick={() => void saveHTML()} disabled={blocked}>
                  Save as HTML…
                </button>
                <button type="button" onClick={() => void savePDF()} disabled={blocked}>
                  Save as PDF…
                </button>
              </>
            }
          >
            Write to CHANGELOG.md
          </SplitButton>
        </div>
        {notice && <p className="changelog-notice">{notice}</p>}
        {actionError && <p className="changelog-error">{actionError}</p>}
        {renderError && <p className="changelog-error">Could not render the Markdown: {renderError}</p>}
        {mode === 'markdown' ? (
          <textarea
            className={stale ? 'changelog-markdown changelog-stale' : 'changelog-markdown'}
            aria-label="Changelog Markdown"
            aria-busy={stale}
            value={markdown}
            onChange={(e) => setDraft(e.target.value)}
            readOnly={stale}
            spellCheck={false}
          />
        ) : previewHtml === null && rendering ? (
          <p className="changelog-preview changelog-hint">Rendering preview…</p>
        ) : (
          // Rendered by goldmark, which escapes raw HTML; links stay inert so the app never navigates away.
          <div
            className={rendering || stale ? 'changelog-preview changelog-stale' : 'changelog-preview'}
            aria-label="Changelog preview"
            aria-busy={rendering || stale}
            onClick={(e) => {
              if ((e.target as HTMLElement).closest('a')) e.preventDefault()
            }}
            dangerouslySetInnerHTML={{ __html: previewHtml ?? '' }}
          />
        )}
      </div>
    </div>
  )
}

interface FilterMenuProps {
  label: string
  items: [string, number][]
  mono?: boolean
  isShown: (item: string) => boolean
  onToggle: (item: string, shown: boolean) => void
}

function FilterMenu({ label, items, mono = false, isShown, onToggle }: FilterMenuProps) {
  const ref = useDismissibleDetails()
  const shown = items.filter(([item]) => isShown(item)).length
  return (
    <details ref={ref} className="changelog-filter">
      <summary className="changelog-filter-toggle">
        <span className="changelog-label">{label}</span>
        <span className="changelog-count">{shown === items.length ? 'All' : `${shown} of ${items.length}`}</span>
        <ChevronDown size={14} strokeWidth={1.5} />
      </summary>
      <div className={mono ? 'changelog-filter-menu changelog-filter-menu-mono' : 'changelog-filter-menu'}>
        {items.map(([item, count]) => (
          <span key={item} className="changelog-filter-option">
            <Checkbox checked={isShown(item)} onChange={(on) => onToggle(item, on)} label={item} />
            <span className="changelog-count">{count}</span>
          </span>
        ))}
      </div>
    </details>
  )
}

export default ReleaseEditor
