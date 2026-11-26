import type { TimelineFileInfo, TimelineInfo } from '@current-client-bindings/app'
import { dirName } from '../../lib/paths'
import { standupSince } from '../../lib/standup'
import { folderAreas, type FolderArea } from './folderAreas'
import { inFolder } from './mapLayout'

export type Span = 'today' | 'week'

export const MOST_LANES = 10

export function spanStart(span: Span, now = new Date()): Date {
  return span === 'week' ? standupSince('week', now) : new Date(now.getFullYear(), now.getMonth(), now.getDate())
}

export interface Lane {
  // null for the lane of every other folder.
  path: string | null
  slot: number
}

export interface Bubble {
  lane: number
  lines: number
  // The file with the most lines in it.
  path: string
}

export interface Entry {
  // null for what isn't committed.
  sha: string | null
  time: number
  subject: string
  lines: number
  // The folders it changed, the most lines first.
  folders: { path: string; lines: number }[]
  bubbles: Bubble[]
}

// The folder area a file is in, by the nearest folder above it that's one.
export function areaFinder(areas: FolderArea[]): (path: string) => number {
  const own = new Map<string, number>()
  const whole = new Map<string, number>()
  areas.forEach((area, i) => (area.own ? own : whole).set(area.path, i))
  return (path) => {
    let folder = dirName(path)
    const direct = own.get(folder)
    if (direct !== undefined) return direct
    for (;;) {
      const found = whole.get(folder)
      if (found !== undefined) return found
      if (folder === '') return -1
      folder = dirName(folder)
    }
  }
}

interface Sums {
  lines: number
  byArea: Map<number, { lines: number; path: string; most: number }>
  folders: Map<string, number>
}

// Lanes are the folder's areas that the range changed, in the map's order and
// colors, with those past the tenth and any folder gone from the code in one more.
export function timelineLanes(
  info: TimelineInfo,
  code: { path: string; lines: number }[],
  folder: string,
  slotOf: (path: string) => number,
  listed: (file: TimelineFileInfo) => boolean,
  now = Date.now(),
): { lanes: Lane[]; entries: Entry[] } {
  const areas = folderAreas(code.filter((f) => inFolder(f.path, folder)))
  const areaOf = areaFinder(areas)
  const sum = (files: TimelineFileInfo[]): Sums => {
    const sums: Sums = { lines: 0, byArea: new Map(), folders: new Map() }
    for (const file of files) {
      if (file.lines === 0 || !inFolder(file.path, folder) || !listed(file)) continue
      const area = areaOf(file.path)
      const name = area < 0 ? dirName(file.path) : areas[area].path
      sums.lines += file.lines
      sums.folders.set(name, (sums.folders.get(name) ?? 0) + file.lines)
      const at = sums.byArea.get(area) ?? { lines: 0, path: file.path, most: 0 }
      at.lines += file.lines
      if (file.lines > at.most) Object.assign(at, { path: file.path, most: file.lines })
      sums.byArea.set(area, at)
    }
    return sums
  }
  const changes = [
    ...info.commits.map((c) => ({
      sha: c.sha as string | null,
      time: Date.parse(c.time),
      subject: c.subject,
      ...sum(c.files),
    })),
    ...(info.now ? [{ sha: null, time: now, subject: 'Uncommitted changes', ...sum(info.working) }] : []),
  ].filter((change) => change.lines > 0)

  const touched = new Set(changes.flatMap((change) => [...change.byArea.keys()]))
  const own = areas.map((_, i) => i).filter((i) => touched.has(i))
  const named = own.slice(0, MOST_LANES)
  const rest = own.length > named.length || touched.has(-1)
  const lanes: Lane[] = named.map((i) => ({ path: areas[i].path, slot: slotOf(`${areas[i].path}/-`) }))
  if (rest) lanes.push({ path: null, slot: 0 })
  const laneOf = new Map(named.map((area, lane) => [area, lane]))

  const entries = changes
    .map(({ sha, time, subject, lines, byArea, folders }): Entry => {
      const bubbles = new Map<number, Bubble & { most: number }>()
      for (const [area, at] of byArea) {
        const lane = laneOf.get(area) ?? lanes.length - 1
        const bubble = bubbles.get(lane)
        if (!bubble) bubbles.set(lane, { lane, ...at })
        else {
          bubble.lines += at.lines
          if (at.most > bubble.most) Object.assign(bubble, { path: at.path, most: at.most })
        }
      }
      return {
        sha,
        time,
        subject,
        lines,
        folders: [...folders].map(([path, lines]) => ({ path, lines })).sort((a, b) => b.lines - a.lines),
        bubbles: [...bubbles.values()],
      }
    })
    .sort((a, b) => a.time - b.time)
  return { lanes, entries }
}

export function when(time: number, now = new Date()): string {
  const date = new Date(time)
  const clock = date.toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' })
  if (date.toDateString() === now.toDateString()) return clock
  const day = date.toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
    year: date.getFullYear() === now.getFullYear() ? undefined : 'numeric',
  })
  return `${day}, ${clock}`
}

export function linesIn({ lines, folders }: Entry): string {
  const most = folders[0]?.path || 'the top folder'
  const counted = `${lines.toLocaleString()} ${lines === 1 ? 'line' : 'lines'}`
  return folders.length === 1 ? `${counted} in ${most}` : `${counted} in ${folders.length} folders, most in ${most}`
}
