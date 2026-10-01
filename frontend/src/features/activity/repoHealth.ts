export const STALE_THRESHOLD_DAYS = 30

export function isStaleBranch(lastCommitDate: string, now: Date = new Date()): boolean {
  const days = (now.getTime() - new Date(lastCommitDate).getTime()) / (1000 * 60 * 60 * 24)
  return days >= STALE_THRESHOLD_DAYS
}

interface HealthChecks {
  staleBranchCount: number
  unpushedCount: number
  workingTreeClean: boolean
}

export function countHealthIssues(checks: HealthChecks): number {
  return (checks.staleBranchCount > 0 ? 1 : 0) + (checks.unpushedCount > 0 ? 1 : 0) + (checks.workingTreeClean ? 0 : 1)
}
