import type { ReadThroughFile } from '@current-client-bindings/app'

export interface FileGroup {
  folder: string
  files: ReadThroughFile[]
}

const TEST_FOLDER = /^(tests?|__tests__|specs?|__snapshots__)$/
const TEST_NAME = /^test_|[._](test|spec)(?=\.)|Tests?(?=\.)|Spec(?=\.)/

// A test goes with the code it's for: a folder of tests with the folder it's
// in, and src/test with src/main.
function folderOf({ path, kind }: ReadThroughFile): string {
  const folders = path.split('/').slice(0, -1)
  if (kind !== 'test') return folders.join('/')
  if (folders[0] === 'src' && folders[1] === 'test') return ['src', 'main', ...folders.slice(2)].join('/')
  while (folders.length > 1 && TEST_FOLDER.test(folders[folders.length - 1])) folders.pop()
  return folders.join('/')
}

const nameOf = (path: string) => path.slice(path.lastIndexOf('/') + 1)
const stemOf = (path: string) =>
  nameOf(path)
    .replace(TEST_NAME, '')
    .replace(/\.[^.]*$/, '')

export function groupFiles(files: ReadThroughFile[]): FileGroup[] {
  const groups = new Map<string, ReadThroughFile[]>()
  for (const file of files) {
    const folder = folderOf(file)
    const members = groups.get(folder)
    if (members) members.push(file)
    else groups.set(folder, [file])
  }
  const order = (a: ReadThroughFile, b: ReadThroughFile) =>
    stemOf(a.path).localeCompare(stemOf(b.path)) ||
    Number(a.kind === 'test') - Number(b.kind === 'test') ||
    a.path.localeCompare(b.path)
  return [...groups]
    .map(([folder, members]) => ({ folder, files: members.sort(order) }))
    .sort((a, b) => a.folder.localeCompare(b.folder))
}

const STATUS_ORDER = 'AMDRCT'
const STATUS_LABEL: Record<string, string> = {
  A: 'Added',
  M: 'Modified',
  D: 'Deleted',
  R: 'Renamed',
  C: 'Copied',
  T: 'Type changed',
}
const KIND_LABEL: Record<string, string> = { test: 'Tests', generated: 'Generated', lockfile: 'Lockfiles' }

export interface Total {
  label: string
  files: number
}

function totals(labels: string[]): Total[] {
  const counts = new Map<string, number>()
  for (const label of labels) counts.set(label, (counts.get(label) ?? 0) + 1)
  return [...counts].map(([label, files]) => ({ label, files }))
}

export function totalsByStatus(files: ReadThroughFile[]): Total[] {
  const letters = files.map((f) => f.status.charAt(0)).sort((a, b) => STATUS_ORDER.indexOf(a) - STATUS_ORDER.indexOf(b))
  return totals(letters.map((letter) => STATUS_LABEL[letter] ?? letter))
}

export function totalsByType(files: ReadThroughFile[]): Total[] {
  const type = (path: string) => /[^.](\.[^.]+)$/.exec(nameOf(path))?.[1] ?? nameOf(path)
  return totals(files.map((f) => type(f.path))).sort((a, b) => b.files - a.files || a.label.localeCompare(b.label))
}

export function totalsByKind(files: ReadThroughFile[]): Total[] {
  return totals(files.flatMap((f) => KIND_LABEL[f.kind] ?? []))
}
