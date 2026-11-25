import type { BranchInfo, ExploreComparison, TagInfo } from '@current-client-bindings/app'
import { useDialogs } from '../../lib/useDialogs'

interface ComparisonSelectProps {
  // Null where nothing is compared, which is only a choice with none.
  value: ExploreComparison | null
  none?: string
  branches: BranchInfo[]
  // The newest first.
  releases: TagInfo[]
  currentBranch: string | null
  onChange: (value: ExploreComparison | null) => void
}

const OTHER = 'other'

function ComparisonSelect({ value, none, branches, releases, currentBranch, onChange }: ComparisonSelectProps) {
  const { prompt } = useDialogs()
  const picked = !value ? '' : value.kind === 'working' ? 'working' : `${value.kind}:${value.ref}`
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
