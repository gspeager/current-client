import type { LanguageStatInfo } from '@current-client-bindings/app'

// Each extension's language name and --lang-* token: its brand color, adjusted
// per theme to stay readable. YAML has no brand color.
const LANGUAGES: Record<string, [name: string, token: string]> = {
  go: ['Go', 'go'],
  ts: ['TypeScript', 'ts'],
  tsx: ['TypeScript', 'ts'],
  js: ['JavaScript', 'js'],
  jsx: ['JavaScript', 'js'],
  mjs: ['JavaScript', 'js'],
  scss: ['SCSS', 'scss'],
  sass: ['SCSS', 'scss'],
  css: ['CSS', 'css'],
  html: ['HTML', 'html'],
  md: ['Markdown', 'md'],
  py: ['Python', 'py'],
  rb: ['Ruby', 'rb'],
  java: ['Java', 'java'],
  c: ['C', 'c'],
  h: ['C', 'c'],
  cpp: ['C++', 'cpp'],
  hpp: ['C++', 'cpp'],
  cs: ['C#', 'cs'],
  rs: ['Rust', 'rs'],
  sh: ['Shell', 'sh'],
  swift: ['Swift', 'swift'],
  kt: ['Kotlin', 'kt'],
  php: ['PHP', 'php'],
  yml: ['YAML', 'scss'],
  yaml: ['YAML', 'scss'],
}

export const OTHER_LANGUAGE_COLOR = 'var(--lang-other)'

export function languageColor(extension: string): string {
  const token = LANGUAGES[extension.toLowerCase()]?.[1]
  return token ? `var(--lang-${token})` : OTHER_LANGUAGE_COLOR
}

export function isKnownLanguage(extension: string): boolean {
  return extension.toLowerCase() in LANGUAGES
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

// One row per language, so .ts and .tsx count together under one color.
export function groupByLanguage(languages: LanguageStatInfo[]): LanguageStatInfo[] {
  const byName = new Map<string, LanguageStatInfo>()
  for (const language of languages) {
    const name = LANGUAGES[language.extension.toLowerCase()]?.[0] ?? language.extension
    const seen = byName.get(name)
    if (seen) byName.set(name, { ...seen, count: seen.count + language.count })
    else byName.set(name, language)
  }
  return [...byName.values()].sort((a, b) => b.count - a.count)
}

export function languageName(extension: string): string {
  return LANGUAGES[extension.toLowerCase()]?.[0] ?? (extension === 'other' ? 'Other' : `.${extension}`)
}
