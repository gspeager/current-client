interface RecentError {
  at: string
  message: string
}

const RECENT_ERROR_LIMIT = 20
const recentErrors: RecentError[] = []

// String(err) would prepend the class name.
export function messageOf(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

// The git command behind a failure, from gitexec.AppError, secrets already
// redacted in Go.
export interface GitFailure {
  command: string
  exitCode: number
  output: string
}

// By message, since the UI keeps a failure as its text. The latest with a text wins.
const failures = new Map<string, GitFailure>()

function failureOf(err: unknown): GitFailure | null {
  const cause =
    err instanceof Error ? ((err as { cause?: unknown }).cause as Record<string, unknown> | undefined) : undefined
  if (!cause || typeof cause !== 'object' || typeof cause.Command !== 'string' || !cause.Command) return null
  return {
    command: cause.Command,
    exitCode: typeof cause.ExitCode === 'number' ? cause.ExitCode : 0,
    output: typeof cause.Detail === 'string' ? cause.Detail : '',
  }
}

export function gitFailureFor(message: string | null | undefined): GitFailure | null {
  return (message && failures.get(message)) || null
}

// Every error the UI shows passes through here, so it also keeps the short,
// local list Copy diagnostics reports.
export function errorMessage(err: unknown): string {
  const message = messageOf(err)
  recentErrors.push({ at: new Date().toISOString(), message })
  if (recentErrors.length > RECENT_ERROR_LIMIT) recentErrors.shift()
  const failure = failureOf(err)
  failures.delete(message)
  if (failure) failures.set(message, failure)
  if (failures.size > RECENT_ERROR_LIMIT) failures.delete(failures.keys().next().value as string)
  return message
}

export function recentErrorsNewestFirst(): RecentError[] {
  return [...recentErrors].reverse()
}

export function ignoreDecorativeFailure(): void {}
