import { createContext, useContext, useMemo } from 'react'

const numbered = (palette: string) => [1, 2, 3, 4, 5, 6, 7, 8].map((n) => `var(--lane-${palette}-${n})`)

// Tokens, so each palette follows the dark or light theme.
export const LANE_THEMES = {
  default: ['main', 'feature', 'release', 'hotfix', 'experimental', 'integration', 'spare', 'sand'].map(
    (b) => `var(--branch-${b})`,
  ),
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

const TRUNK_NAMES = new Set(['main', 'master', 'trunk'])
const SLOTS = LANE_THEMES.default.length - 1

function nameHash(name: string): number {
  let hash = 0
  for (let i = 0; i < name.length; i++) {
    hash = (hash * 31 + name.charCodeAt(i)) >>> 0
  }
  return hash
}

// Spreads a repository's branches across the colors: each, in name order,
// takes its hashed color unless another color is less used. Stable for a given
// set of names, and a new branch moves few others.
export function assignLaneSlots(names: string[]): Map<string, number> {
  const uses = new Array<number>(SLOTS).fill(0)
  const slots = new Map<string, number>()
  for (const name of [...names].filter((n) => !TRUNK_NAMES.has(n)).sort()) {
    const start = nameHash(name) % SLOTS
    let best = start
    for (let i = 0; i < SLOTS; i++) {
      const slot = (start + i) % SLOTS
      if (uses[slot] < uses[best]) best = slot
    }
    uses[best]++
    slots.set(name, best)
  }
  return slots
}

// HEAD always uses --primary. The trunk has the palette's first color, which no
// other branch shares.
export function branchColor(
  theme: LaneThemeName,
  name: string,
  isCurrent: boolean,
  slots?: Map<string, number>,
): string {
  if (isCurrent) {
    return 'var(--primary)'
  }
  if (TRUNK_NAMES.has(name)) return laneColor(theme, 0)
  const palette = LANE_THEMES[theme]
  const slot = slots?.get(name) ?? nameHash(name) % (palette.length - 1)
  return palette[1 + (slot % (palette.length - 1))]
}

export const LaneThemeContext = createContext<LaneThemeName>('default')
export const LaneSlotsContext = createContext<Map<string, number> | undefined>(undefined)

export function useLaneColors() {
  const theme = useContext(LaneThemeContext)
  const slots = useContext(LaneSlotsContext)
  return useMemo(
    () => ({
      laneColor: (lane: number) => laneColor(theme, lane),
      branchColor: (name: string, isCurrent: boolean) => branchColor(theme, name, isCurrent, slots),
    }),
    [theme, slots],
  )
}
