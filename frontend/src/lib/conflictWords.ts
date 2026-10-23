import type { ConflictStateInfo } from '@current-client-bindings/app'

const NOUN: Record<string, string> = {
  merge: 'merge',
  'cherry-pick': 'cherry-pick',
  revert: 'revert',
  rebase: 'rebase',
  am: 'patch',
}

const BADGE: Record<string, string> = {
  merge: 'merging',
  'cherry-pick': 'cherry-picking',
  revert: 'reverting',
  rebase: 'rebasing',
  am: 'applying',
}

// "merge", "rebase", "cherry-pick", "revert" or "patch".
export function operationNoun(operation: string): string {
  return NOUN[operation] ?? operation
}

// The word for the checked-out branch's chip: "merging", "rebasing", ...
export function operationBadge(operation: string): string {
  return BADGE[operation] ?? operation
}

// One line for the banner: "Merging feature into main".
export function operationSentence(state: ConflictStateInfo): string {
  const ours = state.ours || 'HEAD'
  switch (state.operation) {
    case 'merge':
      return `Merging ${state.theirs} into ${ours}`
    case 'rebase':
      return `Rebasing ${state.theirs} onto ${ours}`
    case 'cherry-pick':
      return `Cherry-picking ${state.theirs} onto ${ours}`
    case 'revert':
      return `Reverting on ${ours}`
    case 'am':
      return `Applying a patch on ${ours}`
    default:
      return `${state.operation} in progress`
  }
}
