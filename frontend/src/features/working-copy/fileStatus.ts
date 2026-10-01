export function isStagedStatus(indexStatus: string): boolean {
  return indexStatus !== '.' && indexStatus !== '?'
}

export function isUnstagedStatus(worktreeStatus: string): boolean {
  return worktreeStatus !== '.' && worktreeStatus !== '!'
}
