export const COMMIT_TYPES = [
  'feat',
  'fix',
  'perf',
  'revert',
  'docs',
  'style',
  'refactor',
  'test',
  'build',
  'ci',
  'chore',
]

// Matches core/git's BreakingType.
export const BREAKING_FILTER = '!'

// The prefix part of core/git's conventionalSubject. The composer needs it to
// match before a description is typed, so unlike core/git it doesn't require one.
const PREFIX = /^([A-Za-z]+)(?:\(([^()]+)\))?(!)?: /

interface ConventionalPrefix {
  prefix: string
  type: string
  scope: string
  breaking: boolean
}

export function parsePrefix(subject: string): ConventionalPrefix | null {
  const m = PREFIX.exec(subject)
  if (!m) return null
  return { prefix: m[0], type: m[1].toLowerCase(), scope: m[2] ?? '', breaking: m[3] === '!' }
}

// An empty type removes the prefix; a breaking "!" already typed is kept.
export function withPrefix(subject: string, type: string, scope: string): string {
  const current = parsePrefix(subject)
  const rest = current ? subject.slice(current.prefix.length) : subject
  if (!type) return rest
  return `${type}${scope ? `(${scope})` : ''}${current?.breaking ? '!' : ''}: ${rest}`
}
