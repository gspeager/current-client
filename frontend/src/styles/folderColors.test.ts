import { describe, expect, it } from 'vitest'
import { contrast, themeTokens } from '../test/themeTokens'

const FOLDERS = ['1', '2', '3', '4', '5', '6', '7', '8', 'other']
const FOLDER_TOKENS = FOLDERS.flatMap((folder) => [1, 2, 3, 4].map((step) => `--folder-${folder}-${step}`))
const GROUNDS = ['lowest', 'low', 'container', 'high', 'highest'].map((level) => `--surface-${level}`)

describe.each(Object.entries(themeTokens()))('folder colors, %s theme', (_theme, tokens) => {
  const value = (token: string) => {
    expect(tokens.has(token), token).toBe(true)
    return tokens.get(token)!
  }

  // WCAG's 3:1 for graphics, which the dimmest step has to reach too.
  it('reach 3:1 on every surface at every step', () => {
    for (const token of FOLDER_TOKENS) {
      for (const ground of ['--surface', ...GROUNDS]) {
        expect(contrast(value(token), value(ground)), `${token} on ${ground}`).toBeGreaterThanOrEqual(3)
      }
    }
  })

  it('keep --on-folder readable at every step', () => {
    for (const token of FOLDER_TOKENS) {
      expect(contrast(value('--on-folder'), value(token)), token).toBeGreaterThanOrEqual(4.5)
    }
  })

  it('move away from the canvas with each step', () => {
    for (const folder of FOLDERS) {
      const ratios = [1, 2, 3, 4].map((step) => contrast(value(`--folder-${folder}-${step}`), value('--surface')))
      expect(ratios, folder).toEqual([...ratios].sort((a, b) => a - b))
    }
  })

  it('are never a lane color', () => {
    const lanes = [...tokens].filter(([token]) => /^--(branch|lane)-/.test(token)).map(([, color]) => color)
    for (const token of FOLDER_TOKENS) expect(lanes, token).not.toContain(value(token))
  })
})
