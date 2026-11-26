import { useEffect, useState } from 'react'
import { useDebounceValue } from 'usehooks-ts'
import type { HistoryFilterInfo } from '@current-client-bindings/app'
import { ALL_BRANCHES, isBranchName } from './branchFilter'
import { FILTER_KEYS, typedFilters, typedValues, withTyped, type FilterKey } from './typedFilters'
import { EMPTY_HISTORY_FILTER } from './useCommitHistory'

interface HistoryFilterOptions {
  initialRef: string | null
  defaultRef: string
  firstParent: boolean
  // The local branch names, once known, to drop a saved branch that's gone.
  branches: string[] | undefined
  onRefChange?: (ref: string) => void
  onFirstParentChange?: (on: boolean) => void
}

// The filter the history is read with, and the search box, whose typed
// filters (author:, since: …) set the same filter as the menu.
export function useHistoryFilter({
  initialRef,
  defaultRef,
  firstParent,
  branches,
  onRefChange,
  onFirstParentChange,
}: HistoryFilterOptions) {
  const [filter, setFilter] = useState<HistoryFilterInfo>({
    ...EMPTY_HISTORY_FILTER,
    ref: initialRef ?? defaultRef,
    firstParent,
  })
  const [search, setSearch] = useState('')
  // The box's typed filters apply once typing pauses.
  const [typedSearch] = useDebounceValue(search, 300)
  useEffect(() => {
    const typed = typedFilters(typedSearch)
    if (FILTER_KEYS.some((key) => typed[key] !== filter[key])) setFilter((prev) => ({ ...prev, ...typed }))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [typedSearch])

  // A saved branch that's since been deleted, or belongs to another repository.
  useEffect(() => {
    if (branches && !initialRef && isBranchName(filter.ref) && !branches.includes(filter.ref)) {
      setFilter((prev) => ({ ...prev, ref: ALL_BRANCHES }))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [branches])

  const changeFilter = (next: HistoryFilterInfo) => {
    if (next.ref !== filter.ref) onRefChange?.(next.ref)
    if (next.firstParent !== filter.firstParent) onFirstParentChange?.(next.firstParent)
    setFilter(next)
  }

  // The menu writes its filter into the box too.
  const setTyped = (key: FilterKey, value: string) => {
    const box = withTyped(search, key, value)
    setSearch(box)
    setFilter((prev) => ({ ...prev, ...typedFilters(box) }))
  }

  return { filter, changeFilter, search, setSearch, typed: typedValues(search), setTyped }
}
