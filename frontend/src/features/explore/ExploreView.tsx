import clsx from 'clsx'
import { useMemo, useRef, useState } from 'react'
import { ImageDown } from 'lucide-react'
import { useDebounceValue } from 'usehooks-ts'
import {
  BranchService,
  ExploreService,
  TagService,
  type ExploreComparison,
  type MapChangeInfo,
} from '@current-client-bindings/app'
import SaveImageDialog from '../../components/chrome/SaveImageDialog'
import SegmentedControl from '../../components/controls/SegmentedControl'
import Checkbox from '../../components/forms/Checkbox'
import { baseName } from '../../lib/paths'
import { withHeading, type HeadingColors, type Picture } from '../../lib/picture'
import { plural } from '../../lib/plural'
import { localDate, relativeTime } from '../../lib/relativeTime'
import { useAsyncData } from '../../lib/useAsyncData'
import CompareModal from '../history/CompareModal'
import { useFileTools } from '../history/useFileTools'
import { useOpenReadThrough } from '../../lib/readThrough'
import CodeMap from './CodeMap'
import ComparisonSelect from './ComparisonSelect'
import StructureView from './StructureView'
import { folderAreas, slotFinder, type FolderArea } from './folderAreas'
import { readMapColors, type MapColors } from './drawMap'
import { inFolder, type MapItem, type MapLayout } from './mapLayout'
import { mapPicture, type LegendEntry } from './mapPicture'
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

const CHANGE_KEYS = [
  ...['A', 'M', 'R', 'D'].map((letter) => ({ letter, label: STATUS[letter].word })),
  { letter: '!', label: 'Predicted to conflict' },
]

interface ExploreViewProps {
  repoPath: string
  currentBranch: string | null
  repoVersion: number
  // Changes with uncommitted edits as well.
  workingVersion: unknown
  onOpenWorkingFile: (path: string) => void
}

// Explore's modes share what's being compared and the folder that fills the view.
function ExploreView({ repoPath, currentBranch, repoVersion, workingVersion, onOpenWorkingFile }: ExploreViewProps) {
  const [view, setView] = useState<'code' | 'changes' | 'structure'>('code')
  const [structureCompared, setStructureCompared] = useState(false)
  const [folder, setFolder] = useState('')
  const [comparison, setComparison] = useState<ExploreComparison>({ kind: 'working', ref: '' })
  const [coloring, setColoring] = useState<'commits' | 'age'>('commits')
  const [period, setPeriod] = useState(1)
  const [shown, setShown] = useState({ test: true, generated: true, lockfile: false, binary: false })
  // An index into releases, oldest first; null or past the last is HEAD.
  const [position, setPosition] = useState<number | null>(null)
  const [active, setActive] = useState<MapItem | null>(null)
  const [comparing, setComparing] = useState<string | null>(null)
  const [saving, setSaving] = useState<{ picture: Picture; about: string; colors: HeadingColors } | null>(null)
  const layout = useRef<MapLayout | null>(null)
  const box = useRef<HTMLDivElement>(null)
  const readThrough = useOpenReadThrough()
  const tools = useFileTools(repoPath)

  const { data: tags } = useAsyncData(() => TagService.ListTags(repoPath), [repoPath], { refreshKey: repoVersion })
  const { data: branches } = useAsyncData(() => BranchService.ListLocal(repoPath), [repoPath], {
    refreshKey: repoVersion,
  })
  // Git lists the newest first.
  const releases = useMemo(() => [...(tags ?? [])].reverse(), [tags])
  const release = position === null ? null : (releases[position] ?? null)
  // Debounced, so dragging the slider doesn't read every release it passes.
  const [settled] = useDebounceValue(release?.name ?? 'HEAD', 120)
  const { data: now, error } = useAsyncData(() => ExploreService.Files(repoPath, 'HEAD'), [repoPath], {
    refreshKey: repoVersion,
  })
  const { data: then, error: thenError } = useAsyncData(
    () =>
      settled === 'HEAD'
        ? null
        : ExploreService.Files(repoPath, `refs/tags/${settled}`).then((files) => ({ files, release: settled })),
    [repoPath, settled],
    { keepData: true },
  )
  const { data: activity, error: activityError } = useAsyncData(() => ExploreService.Activity(repoPath), [repoPath], {
    refreshKey: repoVersion,
  })
  const { data: changes, error: changesError } = useAsyncData(
    () =>
      view === 'changes'
        ? ExploreService.Changes(repoPath, comparison).then((found) => ({ ...found, comparison }))
        : null,
    [repoPath, view, comparison],
    { refreshKey: workingVersion, keepData: true },
  )

  const areas = useMemo(() => folderAreas((now ?? []).filter((f) => f.kind !== 'lockfile')), [now])
  const slotOf = useMemo(() => slotFinder(areas), [areas])
  const listed = (file: { kind: string; binary: boolean }) =>
    (!file.binary || shown.binary) && (file.kind === '' || shown[file.kind as keyof typeof shown])

  const atRelease = settled !== 'HEAD'
  const files = useMemo(
    () => ((atRelease ? then?.files : now) ?? []).filter(listed),
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
    if (!activity) return lines
    const file = activity.files[item.path]
    const commits = `${plural(commitsOf(item.path), 'commit')}${PERIODS[period].words && ` ${PERIODS[period].words}`}`
    const last = file ? `changed ${relativeTime(new Date(file.last))}` : 'changed earlier'
    return `${lines}, ${commits}, ${last}`
  }

  const openCompared = (path: string) => {
    if (comparison.kind === 'working') onOpenWorkingFile(path)
    else if (comparison.kind === 'branch') {
      readThrough({ ref: `refs/heads/${comparison.ref}`, uncommitted: false, at: { path, line: 0 } })
    } else setComparing(path)
  }

  const open = (item: MapItem) => {
    if (view === 'code') tools.openHistory(item.path)
    else openCompared(item.path)
  }

  const save = () => {
    if (!layout.current || !box.current) return
    const colors = readMapColors(box.current)
    const about =
      view === 'code'
        ? [
            'Code map',
            root,
            atRelease
              ? then?.release
              : coloring === 'age'
                ? 'by age'
                : `commits ${PERIODS[period].words || 'of all time'}`,
          ]
        : ['Changes', root, changes && comparisonName(changes.comparison, changes.target)]
    setSaving({
      picture: mapPicture(layout.current, legendEntries(areas, legend, view === 'changes', colors), colors),
      about: about.filter(Boolean).join(', '),
      colors: { ground: colors.ground, name: colors.name, muted: colors.muted },
    })
  }

  const legend: StepLegend[] =
    view === 'changes' ? [] : atRelease ? [] : coloring === 'commits' ? steps.legend : ageLegend
  const failure = view === 'code' ? (error ?? (atRelease ? thenError : null)) : changesError
  const loading = view === 'code' ? (atRelease ? !then : !now) : !changes
  const crumbs = root === '' ? [] : root.split('/').map((_, i, names) => names.slice(0, i + 1).join('/'))
  const total = within.reduce((sum, item) => sum + item.size, 0)

  return (
    <div className="explore-view" ref={box}>
      <h2 className="visually-hidden">Explore</h2>
      <div className="explore-toolbar">
        <SegmentedControl
          value={view}
          onChange={setView}
          options={[
            { value: 'code', label: 'Code' },
            { value: 'changes', label: 'Changes' },
            { value: 'structure', label: 'Structure' },
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
          <ComparisonSelect
            value={view === 'structure' && !structureCompared ? null : comparison}
            none={view === 'structure' ? 'Not compared' : undefined}
            branches={branches ?? []}
            releases={[...releases].reverse()}
            currentBranch={currentBranch}
            onChange={(next) => {
              setStructureCompared(!!next)
              if (next) setComparison(next)
            }}
          />
        )}
        {view !== 'structure' && (
          <>
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
            <button
              type="button"
              className="explore-save"
              disabled={!!failure || loading || items.length === 0}
              onClick={save}
              aria-label="Save as image"
              title="Save as image"
            >
              <ImageDown size={14} strokeWidth={1.75} />
            </button>
          </>
        )}
      </div>
      {view === 'structure' ? (
        <StructureView
          repoPath={repoPath}
          folder={folder}
          onFolder={setFolder}
          comparison={structureCompared ? comparison : null}
          version={workingVersion}
          commitsOf={(path) => activity?.files[path]?.commits?.[PERIODS.length - 1]}
          onHistory={tools.openHistory}
          onBlame={tools.openBlame}
          onDiff={openCompared}
        />
      ) : (
        <>
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
                  {baseName(path)}
                </button>
              ))}
            </nav>
            <span className="explore-total">
              {within.length.toLocaleString()} {within.length === 1 ? 'file' : 'files'} · {total.toLocaleString()}{' '}
              {view === 'code' ? 'lines' : 'lines changed'}
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
                  onChange={(e) =>
                    setPosition(Number(e.target.value) >= releases.length ? null : Number(e.target.value))
                  }
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
              onMenu={(e, item) =>
                tools.openMenu(e, tools.pathItems(item.path, item.status !== 'A' && item.status !== 'D'))
              }
              onRoot={setFolder}
              onActive={setActive}
              onLayout={(next) => (layout.current = next)}
            />
          )}
          <div className="explore-facts">
            {active ? (
              <span className="explore-fact">{describe(active)}</span>
            ) : (
              <span className="explore-hint">
                {activityError ??
                  (view === 'code' && atRelease
                    ? 'Brightness is off at a release.'
                    : view === 'code' && activity?.limit
                      ? `Commits are counted over the last ${activity.limit.toLocaleString()}.`
                      : 'A tile is a file. A folder’s name opens it.')}
              </span>
            )}
          </div>
          <Legend areas={areas} steps={legend} changes={view === 'changes'} />
        </>
      )}
      {tools.overlays}
      {saving && (
        <SaveImageDialog
          name={`${baseName(repoPath)}-${view === 'code' ? 'code-map' : 'changes'}`}
          picture={() =>
            withHeading(saving.picture, {
              title: baseName(repoPath),
              about: saving.about,
              date: localDate(),
              colors: saving.colors,
            })
          }
          onClose={() => setSaving(null)}
        />
      )}
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

function comparisonName({ kind, ref }: ExploreComparison, target?: string): string {
  if (kind === 'working') return 'uncommitted'
  return kind === 'branch' ? `${ref} against ${target}` : `since ${ref}`
}

function legendEntries(areas: FolderArea[], steps: StepLegend[], changes: boolean, colors: MapColors): LegendEntry[] {
  const colored = areas.filter((area) => area.slot > 0)
  return [
    ...colored.map((area) => ({ label: areaName(area), color: colors.steps[area.slot][1] })),
    ...(areas.length > colored.length ? [{ label: 'Other folders', color: colors.steps[0][1] }] : []),
    ...steps.map(({ step, label }) => ({ label, color: colors.steps[0][step - 1] })),
    ...(changes ? CHANGE_KEYS : []),
  ]
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
          CHANGE_KEYS.map(({ letter, label }) => (
            <span key={letter} className="explore-key">
              <span className={clsx('explore-letter', letter === '!' && 'explore-letter-conflict')}>{letter}</span>
              {label}
            </span>
          ))}
      </span>
    </div>
  )
}

export default ExploreView
