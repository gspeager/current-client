import type { LanguageStatInfo } from '@current-client-bindings/app'
// Each extension's --lang-* token: its brand color, adjusted per theme to stay
// readable. Extensions that share a language share its token.
const LANGUAGE_TOKENS: Record<string, string> = {
  go: 'go',
  ts: 'ts',
  tsx: 'ts',
  js: 'js',
  jsx: 'js',
  mjs: 'js',
  scss: 'scss',
  sass: 'scss',
  css: 'css',
  html: 'html',
  md: 'md',
  py: 'py',
  rb: 'rb',
  java: 'java',
  c: 'c',
  h: 'c',
  cpp: 'cpp',
  hpp: 'cpp',
  cs: 'cs',
  rs: 'rs',
  sh: 'sh',
  swift: 'swift',
  kt: 'kt',
  php: 'php',
}

// YAML has no brand color.
const LANGUAGE_COLORS: Record<string, string> = {
  ...Object.fromEntries(Object.entries(LANGUAGE_TOKENS).map(([ext, token]) => [ext, `var(--lang-${token})`])),
  yml: 'var(--error)',
  yaml: 'var(--error)',
}

export const OTHER_LANGUAGE_COLOR = 'var(--lang-other)'

export function languageColor(extension: string): string {
  return LANGUAGE_COLORS[extension.toLowerCase()] ?? OTHER_LANGUAGE_COLOR
}

export function isKnownLanguage(extension: string): boolean {
  return extension.toLowerCase() in LANGUAGE_COLORS
}

// Folds unknown extensions and anything past `limit` into "other".
export function groupTopLanguages(languages: LanguageStatInfo[], limit: number): LanguageStatInfo[] {
  const known = languages.filter((l) => l.extension !== 'other' && isKnownLanguage(l.extension))
  const top = known.slice(0, limit)
  const kept = new Set(top.map((l) => l.extension))

  const otherCount = languages.filter((l) => !kept.has(l.extension)).reduce((sum, l) => sum + l.count, 0)

  const result = [...top]
  if (otherCount > 0) result.push({ extension: 'other', count: otherCount })
  return result
}

// Must match core/git/language.go's bucketing.
export function extensionOf(path: string): string {
  const base = path.split(/[\\/]/).pop() ?? path
  const dotIndex = base.lastIndexOf('.')
  if (dotIndex === -1) return 'other'
  return base.slice(dotIndex + 1).toLowerCase()
}
