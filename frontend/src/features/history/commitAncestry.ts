import type { CommitInfo } from '@current-client-bindings/app'

// Lanes are reused after merges, so a branch is its tip's ancestors, not a lane.
export function ancestorShas(commits: CommitInfo[], fromSha: string): Set<string> {
  const bySha = new Map(commits.map((c) => [c.sha, c]))
  const result = new Set<string>()
  const stack = [fromSha]
  while (stack.length > 0) {
    const sha = stack.pop() as string
    if (result.has(sha)) continue
    result.add(sha)
    const commit = bySha.get(sha)
    if (commit) stack.push(...commit.parentShas)
  }
  return result
}
