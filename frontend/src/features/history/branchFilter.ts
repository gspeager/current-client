// The settings file keeps History's branch filter with All branches as the
// unset default, so it's saved differently from the git ref the filter uses.
export const ALL_BRANCHES = '--all'
const CURRENT_BRANCH = ''
const SAVED_CURRENT_BRANCH = 'HEAD'

export function refFromSaved(saved: string | undefined): string {
  if (!saved) return ALL_BRANCHES
  return saved === SAVED_CURRENT_BRANCH ? CURRENT_BRANCH : saved
}

export function savedFromRef(ref: string): string {
  if (ref === ALL_BRANCHES) return ''
  return ref === CURRENT_BRANCH ? SAVED_CURRENT_BRANCH : ref
}

export function isBranchName(ref: string): boolean {
  return ref !== ALL_BRANCHES && ref !== CURRENT_BRANCH
}
