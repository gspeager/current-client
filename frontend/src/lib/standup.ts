import type { StandupRepo } from '@current-client-bindings/app'

export type StandupPeriod = 'yesterday' | 'week'
export type StandupFormat = 'plain' | 'markdown'

// The start of yesterday, or of this week's Monday, in local time.
export function standupSince(period: StandupPeriod, now = new Date()): Date {
  const start = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  if (period === 'yesterday') start.setDate(start.getDate() - 1)
  else start.setDate(start.getDate() - ((start.getDay() + 6) % 7))
  return start
}

const REVIEW_WORDS: Record<string, string> = {
  approved: 'approved',
  changes_requested: 'changes requested',
  commented: 'commented',
}

function groups(repo: StandupRepo): [string, string[]][] {
  const change = (c: { number: number; title: string }) => `${repo.prefix}${c.number} ${c.title}`
  return [
    ['Commits', repo.commits.map((c) => c.subject)],
    ['Merged', (repo.merged ?? []).map(change)],
    ['Open', (repo.opened ?? []).map(change)],
    ['Reviewed', (repo.reviewed ?? []).map((r) => `${change(r)} (${REVIEW_WORDS[r.review] ?? r.review})`)],
  ].filter((group): group is [string, string[]] => group[1].length > 0)
}

// One block per repository with something in it; a repository with nothing
// is left out.
export function standupText(repos: StandupRepo[], format: StandupFormat): string {
  const blocks = repos.flatMap((repo) => {
    const parts = groups(repo)
    if (parts.length === 0) return []
    if (format === 'markdown') {
      return [
        `### ${repo.name}\n\n${parts.map(([label, items]) => `**${label}**\n\n${items.map((i) => `- ${i}`).join('\n')}`).join('\n\n')}`,
      ]
    }
    return [
      `${repo.name}\n${parts.map(([label, items]) => `  ${label}\n${items.map((i) => `  - ${i}`).join('\n')}`).join('\n')}`,
    ]
  })
  return blocks.join('\n\n')
}
