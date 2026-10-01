import { createElement, type ReactNode } from 'react'
import { renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import {
  LANE_THEME_NAMES,
  LANE_THEMES,
  LaneThemeContext,
  branchColor,
  laneColor,
  toLaneThemeName,
  useLaneColors,
} from './laneColor'
import { contrast, themeTokens, tokenOf } from '../test/themeTokens'

describe('laneColor', () => {
  it('wraps around the palette length', () => {
    const palette = LANE_THEMES.default
    expect(laneColor('default', 0)).toBe(palette[0])
    expect(laneColor('default', palette.length)).toBe(palette[0])
    expect(laneColor('default', palette.length + 2)).toBe(palette[2])
  })

  it('uses the given theme', () => {
    expect(laneColor('vivid', 0)).toBe(LANE_THEMES.vivid[0])
  })
})

describe('toLaneThemeName', () => {
  it('keeps a known theme name', () => {
    expect(toLaneThemeName('muted')).toBe('muted')
  })

  it('falls back to default for an unknown or missing name', () => {
    expect(toLaneThemeName('not-a-real-theme')).toBe('default')
    expect(toLaneThemeName(undefined)).toBe('default')
  })
})

describe('branchColor', () => {
  it('uses --primary for the current branch', () => {
    expect(branchColor('default', 'main', true)).toBe('var(--primary)')
  })

  it('maps a name to the same palette color every time', () => {
    const color = branchColor('default', 'feature/x', false)
    expect(branchColor('default', 'feature/x', false)).toBe(color)
    expect(LANE_THEMES.default).toContain(color)
  })
})

describe('LANE_THEME_NAMES', () => {
  it('lists every defined theme', () => {
    expect(LANE_THEME_NAMES).toEqual(Object.keys(LANE_THEMES))
  })
})

describe('useLaneColors', () => {
  it('colors with the provided theme', () => {
    const wrapper = ({ children }: { children: ReactNode }) =>
      createElement(LaneThemeContext.Provider, { value: 'vivid' }, children)
    const { result } = renderHook(() => useLaneColors(), { wrapper })
    expect(result.current.laneColor(0)).toBe(LANE_THEMES.vivid[0])
  })

  it('falls back to the default theme without a provider', () => {
    const { result } = renderHook(() => useLaneColors())
    expect(result.current.laneColor(0)).toBe(LANE_THEMES.default[0])
  })
})

describe('LANE_THEMES', () => {
  const themes = themeTokens()
  const laneTokens = Object.values(LANE_THEMES).flat().map(tokenOf)

  // Each lane color is a token, so each theme must define every one.
  it('uses tokens that both the dark and the light theme define', () => {
    for (const token of laneTokens) {
      expect(themes.dark.has(token), token).toBe(true)
      expect(themes.light.has(token), token).toBe(true)
    }
  })

  // Ref badges write names in --on-lane on a lane color: WCAG AA for small text.
  it('keeps --on-lane readable on every lane color in both themes', () => {
    for (const [theme, tokens] of Object.entries(themes)) {
      for (const token of laneTokens) {
        expect(contrast(tokens.get('--on-lane')!, tokens.get(token)!), `${theme} ${token}`).toBeGreaterThanOrEqual(4.5)
      }
    }
  })
})
