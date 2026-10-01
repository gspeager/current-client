import { BranchService, TagService } from '@current-client-bindings/app'
import { useAsyncData } from '../../lib/useAsyncData'
import './RefSelect.scss'

interface RefGroup {
  label: string
  names: string[]
}

interface RefOption {
  value: string
  label: string
}

function loadRefGroups(repoPath: string): Promise<RefGroup[]> {
  return Promise.all([
    BranchService.ListLocal(repoPath),
    BranchService.ListRemote(repoPath),
    TagService.ListTags(repoPath),
  ])
    .then(([local, remote, tags]) => [
      { label: 'Branches', names: local.map((b) => b.name) },
      { label: 'Remote branches', names: remote },
      { label: 'Tags', names: tags.map((t) => t.name) },
    ])
    .catch(() => [])
}

export function useRefGroups(repoPath: string) {
  return useAsyncData(() => loadRefGroups(repoPath), [repoPath]).data ?? []
}

interface RefSelectProps {
  groups: RefGroup[]
  value: string
  onChange: (ref: string) => void
  label: string
  // Listed before the ref groups, e.g. HEAD.
  extraOptions?: RefOption[]
}

function RefSelect({ groups, value, onChange, label, extraOptions = [] }: RefSelectProps) {
  return (
    <select aria-label={label} className="ref-select" value={value} onChange={(e) => onChange(e.target.value)}>
      {extraOptions.map((o) => (
        <option key={o.value} value={o.value}>
          {o.label}
        </option>
      ))}
      {groups
        .filter((group) => group.names.length > 0)
        .map((group) => (
          <optgroup key={group.label} label={group.label}>
            {group.names.map((name) => (
              <option key={name} value={name}>
                {name}
              </option>
            ))}
          </optgroup>
        ))}
    </select>
  )
}

export default RefSelect
