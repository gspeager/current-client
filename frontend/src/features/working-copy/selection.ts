import type { WorkingTreeSection } from './useWorkingTree'

export type SectionPaths = Record<WorkingTreeSection, string[]>

export interface Selection {
  path: string
  section: WorkingTreeSection
}

const ORDER: WorkingTreeSection[] = ['conflicted', 'unstaged', 'staged']

// After a status refresh: keep the open file while its list still has it; when
// it leaves (staged, discarded, committed), open the one that took its place;
// otherwise the first file. previous is null on the first load, when a
// remembered file opens in whichever list now has it.
export function nextSelection(
  lists: SectionPaths,
  current: Selection | null,
  previous: SectionPaths | null,
): Selection | null {
  if (current) {
    if (lists[current.section].includes(current.path)) return current
    if (previous) {
      const index = previous[current.section].indexOf(current.path)
      const list = lists[current.section]
      if (index !== -1 && list.length > 0)
        return { path: list[Math.min(index, list.length - 1)], section: current.section }
    } else {
      const section = ORDER.find((s) => lists[s].includes(current.path))
      if (section) return { path: current.path, section }
    }
  }
  const section = ORDER.find((s) => lists[s].length > 0)
  return section ? { path: lists[section][0], section } : null
}
