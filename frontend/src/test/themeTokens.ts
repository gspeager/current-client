import { compile } from 'sass'
import { resolve } from 'node:path'

type Theme = 'dark' | 'light'

// Each theme's custom properties, gathered from every top-level block for its
// selector: each token file declares its own.
export function themeTokens(): Record<Theme, Map<string, string>> {
  const { css } = compile(resolve('src/styles/styles.scss'))
  const tokens = (selector: string) =>
    new Map(
      css
        .split('}')
        .filter((rule) => rule.trimStart().startsWith(selector + ' {'))
        .flatMap((rule) => [...rule.matchAll(/(--[\w-]+): ([^;]+);/g)].map((m): [string, string] => [m[1], m[2]])),
    )
  return { dark: tokens('.current-client'), light: tokens('.current-client[data-theme=light]') }
}

// WCAG contrast ratio between two #rrggbb colors.
export function contrast(a: string, b: string): number {
  const luminance = (hex: string) =>
    [1, 3, 5]
      .map((i) => parseInt(hex.slice(i, i + 2), 16) / 255)
      .map((v) => (v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4))
      .reduce((sum, v, i) => sum + v * [0.2126, 0.7152, 0.0722][i], 0)
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x)
  return (hi + 0.05) / (lo + 0.05)
}

export const tokenOf = (color: string) => /^var\((--[\w-]+)\)$/.exec(color)?.[1] ?? color
