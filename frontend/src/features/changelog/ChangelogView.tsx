import { useMemo, useState } from 'react'
import { CircleHelp } from 'lucide-react'
import { ChangelogService } from '@current-client-bindings/app'
import type { ChangelogPrefs } from '@current-client-bindings/internal/config'
import ConventionalCommitsGuide from './ConventionalCommitsGuide'
import ReleaseEditor, { type ChangelogFilters } from './ReleaseEditor'
import Modal from '../../components/chrome/Modal'
import RefSelect, { useRefGroups } from '../../components/git/RefSelect'
import Checkbox from '../../components/forms/Checkbox'
import { sinceDaysAgo } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import './ChangelogView.scss'

// A colon can't appear in a ref name, so these never collide with a branch or tag.
const SINCE_PREFIX = 'since:'
const TIME_PRESETS = [7, 30, 90].map((days) => ({ value: `${SINCE_PREFIX}${days}`, label: `Past ${days} days` }))

interface ChangelogViewProps {
  repoPath: string
  // Refreshes the commits without remounting, so hand edits to the Markdown survive a fetch.
  repoVersion: number
  // Saved in settings so they survive a restart; null until settings load.
  prefs: ChangelogPrefs | null
  onPrefsChange: (prefs: ChangelogPrefs) => void
  onOpenCommit: (sha: string) => void
}

function ChangelogView({ repoPath, repoVersion, prefs, onPrefsChange, onOpenCommit }: ChangelogViewProps) {
  const refGroups = useRefGroups(repoPath)
  const { data: latestTag, error: tagError } = useAsyncData(
    () => ChangelogService.LatestTag(repoPath, 'HEAD'),
    [repoPath],
    { refreshKey: repoVersion },
  )
  const [fromChoice, setFromChoice] = useState<string | null>(null)
  const [to, setTo] = useState('HEAD')
  const [guideOpen, setGuideOpen] = useState(false)
  // Authors differ between repositories, so hiding them isn't saved.
  const [hiddenAuthors, setHiddenAuthors] = useState<ReadonlySet<string>>(new Set())
  const kinds = useMemo(() => new Set(prefs?.types), [prefs?.types])
  const filters = useMemo(() => ({ kinds, hiddenAuthors }), [kinds, hiddenAuthors])
  const options = useMemo(
    () => ({ authors: prefs?.authorNames ?? false, dates: prefs?.dates ?? false }),
    [prefs?.authorNames, prefs?.dates],
  )
  const split = prefs?.splitByRelease ?? false
  const prefsLoaded = prefs !== null
  const savePrefs = (change: Partial<ChangelogPrefs>) => {
    if (prefs) onPrefsChange({ ...prefs, ...change })
  }
  const setFilters = (next: ChangelogFilters) => {
    setHiddenAuthors(next.hiddenAuthors)
    if (next.kinds !== kinds) savePrefs({ types: [...next.kinds] })
  }
  const fromValue = fromChoice ?? latestTag
  const sinceDays = fromValue?.startsWith(SINCE_PREFIX) ? Number(fromValue.slice(SINCE_PREFIX.length)) : 0
  const from = sinceDays ? '' : fromValue
  const since = useMemo(() => sinceDaysAgo(sinceDays), [sinceDays])
  const { data: releases, error } = useAsyncData(
    () => (from === null || !prefsLoaded ? null : ChangelogService.Build(repoPath, from, to, since, split)),
    [repoPath, from, to, since, split, prefsLoaded],
    { refreshKey: repoVersion },
  )
  const range = sinceDays
    ? `in the past ${sinceDays} days${to === 'HEAD' ? '' : ` on ${to}`}`
    : `between ${from || 'the first commit'} and ${to}`

  const content = () => {
    if (error || tagError) return <p className="changelog-error">Could not build the changelog: {error ?? tagError}</p>
    if (!releases || !prefs) return <p className="changelog-hint">Loading commits…</p>
    const entries = releases.flatMap((r) => r.entries)
    if (entries.length === 0) return <p className="changelog-hint">No commits {range}.</p>
    if (entries.every((e) => !e.type)) return <Explainer range={range} />
    return (
      <ReleaseEditor
        key={`${fromValue}..${to}:${split}`}
        repoPath={repoPath}
        releases={releases}
        options={options}
        onOptionsChange={(o) => savePrefs({ authorNames: o.authors, dates: o.dates })}
        mode={prefs?.preview ? 'preview' : 'markdown'}
        onModeChange={(m) => savePrefs({ preview: m === 'preview' })}
        filters={filters}
        onFiltersChange={setFilters}
        onOpenCommit={onOpenCommit}
      />
    )
  }

  return (
    <div className="changelog-view">
      <div className="changelog-range">
        <span className="changelog-label">From</span>
        <RefSelect
          groups={refGroups}
          value={from ?? ''}
          onChange={setFromChoice}
          label="Changelog from"
          extraOptions={[{ value: '', label: 'First commit' }, ...TIME_PRESETS]}
        />
        <span className="changelog-label">to</span>
        <RefSelect
          groups={refGroups}
          value={to}
          onChange={setTo}
          label="Changelog to"
          extraOptions={[{ value: 'HEAD', label: 'HEAD' }]}
        />
        <Checkbox checked={split} onChange={(on) => savePrefs({ splitByRelease: on })} label="Split by release" />
        <span className="changelog-spacer" />
        <button
          type="button"
          className="changelog-icon-button"
          onClick={() => setGuideOpen(true)}
          aria-label="Conventional Commits guide"
          title="Conventional Commits guide"
        >
          <CircleHelp size={14} strokeWidth={1.5} />
        </button>
      </div>
      {guideOpen && (
        <Modal title="Conventional Commits" onClose={() => setGuideOpen(false)} className="changelog-guide-modal">
          <div className="changelog-guide-body">
            <ConventionalCommitsGuide />
          </div>
        </Modal>
      )}
      {content()}
    </div>
  )
}

function Explainer({ range }: { range: string }) {
  return (
    <div className="changelog-explainer">
      <h2 className="changelog-explainer-title">No Conventional Commits {range}</h2>
      <ConventionalCommitsGuide />
    </div>
  )
}

export default ChangelogView
