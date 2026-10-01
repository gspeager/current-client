import { createContext, useContext, useMemo } from 'react'

const numbered = (palette: string) => [1, 2, 3, 4, 5, 6].map((n) => `var(--lane-${palette}-${n})`)

// Tokens, so each palette follows the dark or light theme.
export const LANE_THEMES = {
  default: ['main', 'feature', 'release', 'hotfix', 'experimental', 'integration'].map((b) => `var(--branch-${b})`),
  vivid: numbered('vivid'),
  muted: numbered('muted'),
}

export type LaneThemeName = keyof typeof LANE_THEMES

export const LANE_THEME_NAMES = Object.keys(LANE_THEMES) as LaneThemeName[]

export function toLaneThemeName(value: string | undefined): LaneThemeName {
  return LANE_THEME_NAMES.find((name) => name === value) ?? 'default'
}

export function laneColor(theme: LaneThemeName, lane: number): string {
  const palette = LANE_THEMES[theme]
  return palette[lane % palette.length]
}

// HEAD always uses --primary rather than its hashed lane color.
export function branchColor(theme: LaneThemeName, name: string, isCurrent: boolean): string {
  if (isCurrent) {
    return 'var(--primary)'
  }
  let hash = 0
  for (let i = 0; i < name.length; i++) {
    hash = (hash * 31 + name.charCodeAt(i)) >>> 0
  }
  return laneColor(theme, hash)
}

export const LaneThemeContext = createContext<LaneThemeName>('default')

export function useLaneColors() {
  const theme = useContext(LaneThemeContext)
  return useMemo(
    () => ({
      laneColor: (lane: number) => laneColor(theme, lane),
      branchColor: (name: string, isCurrent: boolean) => branchColor(theme, name, isCurrent),
    }),
    [theme],
  )
}
