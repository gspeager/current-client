import { localDate } from './relativeTime'
import { shortSha } from './text'

export function suggestedPatchFilename(sha: string, subject: string): string {
  const slug = subject
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 50)
  return `${shortSha(sha)}-${slug || 'patch'}.patch`
}

export function suggestedWorkingTreePatchFilename(now: Date = new Date()): string {
  const time = String(now.getHours() * 100 + now.getMinutes()).padStart(4, '0')
  return `working-tree-${localDate(now)}-${time}.patch`
}
