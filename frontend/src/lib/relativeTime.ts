export function relativeTime(from: Date, now: Date = new Date()): string {
  const elapsedSeconds = Math.max(0, Math.floor((now.getTime() - from.getTime()) / 1000))
  if (elapsedSeconds < 10) return 'just now'
  if (elapsedSeconds < 60) return `${elapsedSeconds}s ago`
  if (elapsedSeconds < 3600) return `${Math.floor(elapsedSeconds / 60)}m ago`
  if (elapsedSeconds < 86400) return `${Math.floor(elapsedSeconds / 3600)}h ago`
  return `${Math.floor(elapsedSeconds / 86400)}d ago`
}

// 0 means no lower bound, which git's --since takes as an empty string.
export function sinceDaysAgo(days: number, now: Date = new Date()): string {
  if (days === 0) return ''
  const since = new Date(now)
  since.setDate(since.getDate() - days)
  return since.toISOString()
}
