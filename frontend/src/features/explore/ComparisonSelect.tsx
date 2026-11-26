import type { BranchInfo, ExploreComparison, TagInfo } from '@current-client-bindings/app'
import { useDialogs } from '../../lib/useDialogs'

interface ComparisonSelectProps {
  // Null for nothing compared, an option only when `none` labels it.
  value: ExploreComparison | null
  none?: string
  // Today and This week as well, as kinds "today" and "week".
  spans?: boolean
  branches: BranchInfo[]
  // The newest first.
  releases: TagInfo[]
  currentBranch: string | null
  onChange: (value: ExploreComparison | null) => void
}

const OTHER = 'other'

function ComparisonSelect({ value, none, spans, branches, releases, currentBranch, onChange }: ComparisonSelectProps) {
  const { prompt } = useDialogs()
  const picked = !value ? '' : value.ref ? `${value.kind}:${value.ref}` : value.kind
  const pick = async (choice: string) => {
    if (choice === OTHER) {
      const ref = await prompt({
        title: 'Compare with another point',
        label: 'Commit, branch or tag',
        confirmLabel: 'Compare',
      })
      if (ref?.trim()) onChange({ kind: 'commit', ref: ref.trim() })
      return
    }
    const [kind, ...ref] = choice.split(':')
    onChange(kind ? { kind, ref: ref.join(':') } : null)
  }
  return (
    <select className="explore-select" aria-label="Compare" value={picked} onChange={(e) => void pick(e.target.value)}>
      {none && <option value="">{none}</option>}
      {spans && (
        <>
          <option value="today">Today</option>
          <option value="week">This week</option>
        </>
      )}
      <option value="working">Uncommitted changes</option>
      <optgroup label="A branch and its base">
        {branches.map(({ name }) => (
          <option key={name} value={`branch:${name}`}>
            {name === currentBranch ? `${name} (current)` : name}
          </option>
        ))}
      </optgroup>
      {releases.length > 0 && (
        <optgroup label="Since a release">
          {releases.map(({ name }) => (
            <option key={name} value={`release:${name}`}>
              {name}
            </option>
          ))}
        </optgroup>
      )}
      <optgroup label="Since another point">
        {value?.kind === 'commit' && <option value={picked}>{value.ref}</option>}
        <option value={OTHER}>Choose…</option>
      </optgroup>
    </select>
  )
}

export default ComparisonSelect
