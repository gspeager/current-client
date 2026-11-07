export function relativeTime(from: Date, now: Date = new Date()): string {
  const elapsedSeconds = Math.max(0, Math.floor((now.getTime() - from.getTime()) / 1000))
  if (elapsedSeconds < 10) return 'just now'
  if (elapsedSeconds < 60) return `${elapsedSeconds}s ago`
  if (elapsedSeconds < 3600) return `${Math.floor(elapsedSeconds / 60)}m ago`
  if (elapsedSeconds < 86400) return `${Math.floor(elapsedSeconds / 3600)}h ago`
  const elapsedDays = Math.floor(elapsedSeconds / 86400)
  if (elapsedDays < 7) return `${elapsedDays}d ago`
  // Past a year, the day stops helping and "Mar 14" alone would read as this year's.
  return from.toLocaleDateString(
    undefined,
    elapsedDays < 365 ? { month: 'short', day: 'numeric' } : { month: 'short', year: 'numeric' },
  )
}

// 0 means no lower bound, which git's --since takes as an empty string.
export function sinceDaysAgo(days: number, now: Date = new Date()): string {
  if (days === 0) return ''
  const since = new Date(now)
  since.setDate(since.getDate() - days)
  return since.toISOString()
}
