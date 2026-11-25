import { useMemo, useState } from 'react'
import { useDebounceValue } from 'usehooks-ts'
import {
  BranchService,
  ExploreService,
  TagService,
  type ExploreComparison,
  type MapChangeInfo,
} from '@current-client-bindings/app'
import SegmentedControl from '../../components/controls/SegmentedControl'
import Checkbox from '../../components/forms/Checkbox'
import { plural } from '../../lib/plural'
import { relativeTime } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import CompareModal from '../history/CompareModal'
import { useFileTools } from '../history/useFileTools'
import { useOpenReadThrough } from '../read-through/readThrough'
import CodeMap from './CodeMap'
import { folderAreas, slotFinder, type FolderArea } from './folderAreas'
import { inFolder, type MapItem } from './mapLayout'
import { ageLegend, ageStep, commitSteps, type Step, type StepLegend } from './mapSteps'
import './ExploreView.scss'

const PERIODS = [
  { label: '30 days', words: 'in 30 days' },
  { label: '90 days', words: 'in 90 days' },
  { label: 'A year', words: 'in a year' },
  { label: 'All', words: '' },
]

const STATUS: Record<string, { word: string; step: Step }> = {
  A: { word: 'Added', step: 4 },
  M: { word: 'Modified', step: 2 },
  T: { word: 'Type changed', step: 2 },
  R: { word: 'Renamed', step: 3 },
  D: { word: 'Deleted', step: 2 },
}

interface ExploreViewProps {
  repoPath: string
  currentBranch: string | null
  repoVersion: number
  // Changes with uncommitted edits as well.
  workingVersion: unknown
  onOpenWorkingFile: (path: string) => void
}

// Explore's modes share what's being compared and the folder that fills the view.
// The Code map is its one mode so far.
function ExploreView({ repoPath, currentBranch, repoVersion, workingVersion, onOpenWorkingFile }: ExploreViewProps) {
  const [view, setView] = useState<'code' | 'changes'>('code')
  const [folder, setFolder] = useState('')
  const [comparison, setComparison] = useState<ExploreComparison>({ kind: 'working', ref: '' })
  const [coloring, setColoring] = useState<'commits' | 'age'>('commits')
  const [period, setPeriod] = useState(1)
  const [shown, setShown] = useState({ test: true, generated: true, lockfile: false, binary: false })
  // A position among the releases; past the last is now.
  const [position, setPosition] = useState<number | null>(null)
  const [active, setActive] = useState<MapItem | null>(null)
  const [comparing, setComparing] = useState<string | null>(null)
  const readThrough = useOpenReadThrough()
  const tools = useFileTools(repoPath)

  const { data: tags } = useAsyncData(() => TagService.ListTags(repoPath), [repoPath], { refreshKey: repoVersion })
  const { data: branches } = useAsyncData(() => BranchService.ListLocal(repoPath), [repoPath], {
    refreshKey: repoVersion,
  })
  const releases = useMemo(() => [...(tags ?? [])].sort((a, b) => a.date.localeCompare(b.date)), [tags])
  const release = position === null ? null : (releases[position] ?? null)
  // Dragging the slider passes releases nobody stops at.
  const [settled] = useDebounceValue(release?.name ?? 'HEAD', 120)
  const { data: now, error } = useAsyncData(() => ExploreService.Files(repoPath, 'HEAD'), [repoPath], {
    refreshKey: repoVersion,
  })
  const { data: then, error: thenError } = useAsyncData(
    () => (settled === 'HEAD' ? null : ExploreService.Files(repoPath, `refs/tags/${settled}`)),
    [repoPath, settled],
    { keepData: true },
  )
  const { data: activity, error: activityError } = useAsyncData(() => ExploreService.Activity(repoPath), [repoPath], {
    refreshKey: repoVersion,
  })
  const { data: changes, error: changesError } = useAsyncData(
    () => (view === 'changes' ? ExploreService.Changes(repoPath, comparison) : null),
    [repoPath, view, comparison],
    { refreshKey: workingVersion, keepData: true },
  )

  const areas = useMemo(() => folderAreas((now ?? []).filter((f) => f.kind !== 'lockfile')), [now])
  const slotOf = useMemo(() => slotFinder(areas), [areas])
  const listed = (file: { kind: string; binary: boolean }) =>
    (!file.binary || shown.binary) && (file.kind === '' || shown[file.kind as keyof typeof shown])

  const atRelease = settled !== 'HEAD'
  const files = useMemo(
    () => ((atRelease ? then : now) ?? []).filter(listed),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [atRelease, then, now, shown],
  )
  const commitsOf = (path: string) => activity?.files[path]?.commits?.[period] ?? 0
  const steps = useMemo(
    () => commitSteps(files.map((f) => commitsOf(f.path))),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [files, activity, period],
  )
  const codeItems = useMemo(() => {
    const time = Date.now()
    const stepOf = (path: string): Step => {
      if (atRelease) return 2
      if (coloring === 'commits') return steps.stepOf(commitsOf(path))
      const last = activity?.files[path]?.last
      return ageStep(last ? Date.parse(last) : undefined, time)
    }
    return files.map((f): MapItem => ({ path: f.path, size: f.lines, slot: slotOf(f.path), step: stepOf(f.path) }))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [files, steps, coloring, atRelease, slotOf, activity])
  const changed = useMemo(
    () => (changes?.files ?? []).filter(listed),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [changes, shown],
  )
  const changeItems = useMemo(
    () =>
      changed.map((f): MapItem => ({
        path: f.path,
        size: f.added + f.removed,
        slot: slotOf(f.path),
        step: STATUS[f.status]?.step ?? 2,
        status: f.status,
        conflict: f.conflict,
      })),
    [changed, slotOf],
  )
  const items = view === 'code' ? codeItems : changeItems
  const changeOf = useMemo(() => new Map(changed.map((f) => [f.path, f])), [changed])
  // The folder stays chosen while nothing in it is shown, and shows itself again.
  const root = items.some((item) => inFolder(item.path, folder)) ? folder : ''
  const within = items.filter((item) => inFolder(item.path, root))

  const describe = (item: MapItem): string => {
    const change = changeOf.get(item.path)
    if (view === 'changes' && change) return describeChange(change)
    const lines = `${item.path}, ${plural(item.size, 'line')}`
    if (atRelease) return `${lines} at ${settled}`
    const file = activity?.files[item.path]
    const commits = `${plural(commitsOf(item.path), 'commit')}${PERIODS[period].words && ` ${PERIODS[period].words}`}`
    const last = file ? `changed ${relativeTime(new Date(file.last))}` : 'changed earlier'
    return `${lines}, ${commits}, ${last}`
  }

  const open = (item: MapItem) => {
    if (view === 'code') tools.openHistory(item.path)
    else if (comparison.kind === 'working') onOpenWorkingFile(item.path)
    else if (comparison.kind === 'branch') {
      readThrough({ ref: `refs/heads/${comparison.ref}`, uncommitted: false, at: { path: item.path, line: 0 } })
    } else setComparing(item.path)
  }

  const legend: StepLegend[] =
    view === 'changes' ? [] : atRelease ? [] : coloring === 'commits' ? steps.legend : ageLegend
  const failure = view === 'code' ? (error ?? (atRelease ? thenError : null)) : changesError
  const loading = view === 'code' ? (atRelease ? !then : !now) : !changes
  const crumbs = root === '' ? [] : root.split('/').map((_, i, names) => names.slice(0, i + 1).join('/'))
  const total = within.reduce((sum, item) => sum + item.size, 0)
  const picked = comparison.kind === 'working' ? 'working' : `${comparison.kind}:${comparison.ref}`

  return (
    <div className="explore-view">
      <h2 className="visually-hidden">Explore</h2>
      <div className="explore-toolbar">
        <SegmentedControl
          value={view}
          onChange={setView}
          options={[
            { value: 'code', label: 'Code' },
            { value: 'changes', label: 'Changes' },
          ]}
        />
        {view === 'code' ? (
          <>
            <SegmentedControl
              value={coloring}
              onChange={setColoring}
              options={[
                { value: 'commits', label: 'Commits' },
                { value: 'age', label: 'Age' },
              ]}
            />
            {coloring === 'commits' && (
              <select
                className="explore-select"
                aria-label="Period"
                value={period}
                onChange={(e) => setPeriod(Number(e.target.value))}
              >
                {PERIODS.map(({ label }, i) => (
                  <option key={label} value={i}>
                    {label}
                  </option>
                ))}
              </select>
            )}
          </>
        ) : (
          <select
            className="explore-select"
            aria-label="Compare"
            value={picked}
            onChange={(e) => {
              const [kind, ...ref] = e.target.value.split(':')
              setComparison({ kind, ref: ref.join(':') })
            }}
          >
            <option value="working">Uncommitted changes</option>
            <optgroup label="A branch and its base">
              {(branches ?? []).map(({ name }) => (
                <option key={name} value={`branch:${name}`}>
                  {name === currentBranch ? `${name} (current)` : name}
                </option>
              ))}
            </optgroup>
            {releases.length > 0 && (
              <optgroup label="Since a release">
                {[...releases].reverse().map(({ name }) => (
                  <option key={name} value={`release:${name}`}>
                    {name}
                  </option>
                ))}
              </optgroup>
            )}
          </select>
        )}
        <div className="explore-shown" role="group" aria-label="Show">
          <Checkbox checked={shown.test} onChange={(test) => setShown({ ...shown, test })} label="Tests" />
          <Checkbox
            checked={shown.generated}
            onChange={(generated) => setShown({ ...shown, generated })}
            label="Generated"
          />
          <Checkbox
            checked={shown.lockfile}
            onChange={(lockfile) => setShown({ ...shown, lockfile })}
            label="Lockfiles"
          />
          <Checkbox checked={shown.binary} onChange={(binary) => setShown({ ...shown, binary })} label="Binary" />
        </div>
      </div>
      <div className="explore-toolbar">
        <nav className="explore-path" aria-label="Folder">
          <button type="button" className="explore-crumb" disabled={root === ''} onClick={() => setFolder('')}>
            All
          </button>
          {crumbs.map((path) => (
            <button
              key={path}
              type="button"
              className="explore-crumb"
              disabled={path === root}
              onClick={() => setFolder(path)}
            >
              {path.slice(path.lastIndexOf('/') + 1)}
            </button>
          ))}
        </nav>
        <span className="explore-total">
          {plural(within.length, 'file')} · {total.toLocaleString()} {view === 'code' ? 'lines' : 'lines changed'}
        </span>
        {view === 'code' && releases.length > 0 && (
          <label className="explore-release">
            <span className="explore-release-name">{release?.name ?? 'Now'}</span>
            <input
              type="range"
              min={0}
              max={releases.length}
              value={position ?? releases.length}
              aria-label="Release"
              aria-valuetext={release?.name ?? 'Now'}
              onChange={(e) => setPosition(Number(e.target.value) >= releases.length ? null : Number(e.target.value))}
            />
          </label>
        )}
        {view === 'changes' && changes?.target && <span className="explore-note">against {changes.target}</span>}
        {view === 'changes' && comparison.kind === 'branch' && changes && !changes.predicted && (
          <span className="explore-note">Conflicts need Git 2.38 or newer</span>
        )}
        {view === 'changes' && (changes?.skipped ?? 0) > 0 && (
          <span className="explore-note">{plural(changes!.skipped, 'large untracked file')} left out</span>
        )}
      </div>
      {failure ? (
        <p className="explore-message explore-error">{failure}</p>
      ) : loading ? (
        <p className="explore-message">Reading files…</p>
      ) : items.length === 0 ? (
        <p className="explore-message">
          {view === 'code'
            ? 'No files to show.'
            : comparison.kind === 'branch' && !changes?.target
              ? 'This repository has no default branch to compare with.'
              : 'Nothing has changed.'}
        </p>
      ) : (
        <CodeMap
          items={items}
          root={root}
          label={view === 'code' ? 'Code map' : 'Map of the changes'}
          describe={describe}
          onOpen={open}
          onMenu={(e, item) => tools.openMenu(e, tools.pathItems(item.path, item.status !== 'A'))}
          onRoot={setFolder}
          onActive={setActive}
        />
      )}
      <div className="explore-facts" role="status">
        {active ? (
          <span className="explore-fact">{describe(active)}</span>
        ) : (
          <span className="explore-hint">
            {activityError ??
              (view === 'code' && activity?.limit
                ? `Commits are counted over the last ${activity.limit.toLocaleString()}.`
                : 'A tile is a file. A folder’s name opens it.')}
          </span>
        )}
      </div>
      <Legend areas={areas} steps={legend} changes={view === 'changes'} />
      {tools.overlays}
      {comparing !== null && (
        <CompareModal
          repoPath={repoPath}
          initialFromRef={comparison.ref}
          initialToRef="HEAD"
          initialPath={comparing}
          onClose={() => setComparing(null)}
        />
      )}
    </div>
  )
}

function describeChange({ path, origPath, status, added, removed, binary, conflict }: MapChangeInfo): string {
  const what = STATUS[status]?.word ?? status
  const from = origPath ? ` from ${origPath}` : ''
  const lines = binary ? 'binary' : `+${added} -${removed}`
  return `${path}, ${what.toLowerCase()}${from}, ${lines}${conflict ? ', predicted to conflict' : ''}`
}

const areaName = (area: FolderArea) => area.path || 'Top folder'

function Legend({ areas, steps, changes }: { areas: FolderArea[]; steps: StepLegend[]; changes: boolean }) {
  const colored = areas.filter((area) => area.slot > 0)
  return (
    <div className="explore-legend" aria-label="Legend">
      {colored.map((area) => (
        <span key={`${area.path}-${area.own}`} className="explore-key">
          <span className={`explore-swatch explore-swatch-${area.slot}`} />
          {areaName(area)}
        </span>
      ))}
      {areas.length > colored.length && (
        <span className="explore-key">
          <span className="explore-swatch explore-swatch-other" />
          Other folders
        </span>
      )}
      <span className="explore-legend-steps">
        {steps.map(({ step, label }) => (
          <span key={step} className="explore-key">
            <span className={`explore-step explore-step-${step}`} />
            {label}
          </span>
        ))}
        {changes &&
          ['A', 'M', 'R', 'D'].map((status) => (
            <span key={status} className="explore-key">
              <span className="explore-letter">{status}</span>
              {STATUS[status].word}
            </span>
          ))}
        {changes && (
          <span className="explore-key">
            <span className="explore-letter explore-letter-conflict">!</span>
            Predicted to conflict
          </span>
        )}
      </span>
    </div>
  )
}

export default ExploreView
