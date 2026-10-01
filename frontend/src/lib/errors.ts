interface RecentError {
  at: string
  message: string
}

const RECENT_ERROR_LIMIT = 20
const recentErrors: RecentError[] = []

// String(err) would prepend the class name. Every error the UI shows passes
// through here, so it also keeps the short, local list Copy diagnostics reports.
export function errorMessage(err: unknown): string {
  const message = err instanceof Error ? err.message : String(err)
  recentErrors.push({ at: new Date().toISOString(), message })
  if (recentErrors.length > RECENT_ERROR_LIMIT) recentErrors.shift()
  return message
}

export function recentErrorsNewestFirst(): RecentError[] {
  return [...recentErrors].reverse()
}

export function ignoreDecorativeFailure(): void {}
