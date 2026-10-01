import { describe, expect, it } from 'vitest'
import { contrast, themeTokens, tokenOf } from '../../test/themeTokens'
import { extensionOf, groupTopLanguages, isKnownLanguage, languageColor, OTHER_LANGUAGE_COLOR } from './languageColors'

describe('languageColor', () => {
  it('returns a token per language, shared by its extensions', () => {
    expect(languageColor('go')).toBe('var(--lang-go)')
    expect(languageColor('tsx')).toBe(languageColor('ts'))
  })

  it('is case-insensitive', () => {
    expect(languageColor('GO')).toBe('var(--lang-go)')
  })

  it('falls back to the neutral color for an unmapped extension', () => {
    expect(languageColor('xyz')).toBe(OTHER_LANGUAGE_COLOR)
    expect(languageColor('other')).toBe(OTHER_LANGUAGE_COLOR)
  })
})

describe('extensionOf', () => {
  it('extracts the extension from a nested path', () => {
    expect(extensionOf('frontend/src/App.tsx')).toBe('tsx')
  })

  it('handles Windows-style backslash paths', () => {
    expect(extensionOf('frontend\\src\\App.tsx')).toBe('tsx')
  })

  it('treats a dotfile as its own extension, matching the Go bucketing', () => {
    expect(extensionOf('.gitignore')).toBe('gitignore')
  })

  it('returns other for a file with no extension at all', () => {
    expect(extensionOf('README')).toBe('other')
  })

  it('uses the last dot for a multi-dot filename', () => {
    expect(extensionOf('main.test.go')).toBe('go')
  })
})

describe('isKnownLanguage', () => {
  it('is true for a mapped extension', () => {
    expect(isKnownLanguage('go')).toBe(true)
    expect(isKnownLanguage('GO')).toBe(true)
  })

  it('is false for an unmapped extension, including json', () => {
    expect(isKnownLanguage('json')).toBe(false)
    expect(isKnownLanguage('xyz')).toBe(false)
  })
})

describe('groupTopLanguages', () => {
  it('keeps only the top-N known languages, folding the rest into other', () => {
    const languages = [
      { extension: 'go', count: 140 },
      { extension: 'scss', count: 55 },
      { extension: 'tsx', count: 53 },
      { extension: 'ts', count: 49 },
      { extension: 'yml', count: 6 },
      { extension: 'json', count: 5 },
      { extension: 'png', count: 4 },
    ]
    const got = groupTopLanguages(languages, 4)
    expect(got).toEqual([
      { extension: 'go', count: 140 },
      { extension: 'scss', count: 55 },
      { extension: 'tsx', count: 53 },
      { extension: 'ts', count: 49 },
      { extension: 'other', count: 6 + 5 + 4 },
    ])
  })

  it('folds an unknown extension into other even if it would rank in the top N by count', () => {
    // png (unknown) outranks yml (known) by count, but only known languages
    // compete for the top-N slots — an unmapped extension never gets its
    // own row, since it would render as the same gray as "other" anyway.
    const languages = [
      { extension: 'go', count: 10 },
      { extension: 'png', count: 8 },
      { extension: 'yml', count: 1 },
    ]
    const got = groupTopLanguages(languages, 2)
    expect(got).toEqual([
      { extension: 'go', count: 10 },
      { extension: 'yml', count: 1 },
      { extension: 'other', count: 8 },
    ])
  })

  it('includes the backend\'s own "other" bucket in the folded total', () => {
    const languages = [
      { extension: 'go', count: 10 },
      { extension: 'other', count: 3 },
    ]
    expect(groupTopLanguages(languages, 4)).toEqual([
      { extension: 'go', count: 10 },
      { extension: 'other', count: 3 },
    ])
  })

  it('omits the other entry entirely when nothing is folded', () => {
    const languages = [{ extension: 'go', count: 10 }]
    expect(groupTopLanguages(languages, 4)).toEqual([{ extension: 'go', count: 10 }])
  })
})

describe('language color tokens', () => {
  const themes = themeTokens()
  const languageTokens = (theme: Map<string, string>) => [...theme.keys()].filter((t) => t.startsWith('--lang-'))

  it('are defined for the same languages in both themes', () => {
    expect(languageTokens(themes.light)).toEqual(languageTokens(themes.dark))
    expect(themes.dark.has(tokenOf(OTHER_LANGUAGE_COLOR))).toBe(true)
  })

  // They color text too (a file's type in File churn, the top percentage), so
  // they need WCAG AA for small text on the chart panels.
  it('stay readable on --surface-low in both themes', () => {
    for (const [theme, tokens] of Object.entries(themes)) {
      for (const token of languageTokens(tokens)) {
        expect(contrast(tokens.get(token)!, tokens.get('--surface-low')!), `${theme} ${token}`).toBeGreaterThanOrEqual(
          4.5,
        )
      }
    }
  })
})
