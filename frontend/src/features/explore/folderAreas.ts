import { dirName } from '../../lib/paths'
// A folder with a color of its own, and its slot. Slot 0 is every other folder's.
export interface FolderArea {
  path: string
  // Only the files directly in it: its folders are areas themselves.
  own: boolean
  lines: number
  slot: number
}

interface Folder {
  path: string
  lines: number
  ownLines: number
  folders: Map<string, Folder>
}

export const COLORED = 8
const MOST_AREAS = 40

function folderTree(files: { path: string; lines: number }[]): Folder {
  const root: Folder = { path: '', lines: 0, ownLines: 0, folders: new Map() }
  for (const { path, lines } of files) {
    const names = path.split('/')
    let folder = root
    root.lines += lines
    for (let i = 0; i < names.length - 1; i++) {
      let next = folder.folders.get(names[i])
      if (!next) {
        next = { path: names.slice(0, i + 1).join('/'), lines: 0, ownLines: 0, folders: new Map() }
        folder.folders.set(names[i], next)
      }
      next.lines += lines
      folder = next
    }
    folder.ownLines += lines
  }
  return root
}

// The top folders, with any that holds more than an eighth of the code split into
// its own, so one folder that holds nearly everything isn't one color. They come
// from all of the code and nothing else, so a folder keeps its color.
export function folderAreas(files: { path: string; lines: number }[]): FolderArea[] {
  const root = folderTree(files)
  const whole: Folder[] = [root]
  const areas: Omit<FolderArea, 'slot'>[] = []
  for (;;) {
    const largest = whole
      .filter((folder) => folder.folders.size > 0 && (folder === root || folder.lines > root.lines / COLORED))
      .sort((a, b) => b.lines - a.lines || a.path.localeCompare(b.path))[0]
    if (!largest || whole.length + areas.length >= MOST_AREAS) break
    whole.splice(whole.indexOf(largest), 1, ...largest.folders.values())
    if (largest.ownLines > 0) areas.push({ path: largest.path, own: true, lines: largest.ownLines })
  }
  for (const folder of whole) areas.push({ path: folder.path, own: false, lines: folder.lines })
  return areas
    .sort((a, b) => b.lines - a.lines || a.path.localeCompare(b.path))
    .map((area, i) => ({ ...area, slot: i < COLORED ? i + 1 : 0 }))
}

export const folderOf = dirName

// Answers a file's slot, by the nearest area its folder is in.
export function slotFinder(areas: FolderArea[]): (path: string) => number {
  const known = new Map(areas.map((area) => [area.path, area.slot]))
  const slotOf = (folder: string): number => {
    const slot = known.get(folder)
    if (slot !== undefined) return slot
    const found = folder === '' ? 0 : slotOf(folderOf(folder))
    known.set(folder, found)
    return found
  }
  return (path) => slotOf(folderOf(path))
}
