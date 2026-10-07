import type { FileStatus } from '@current-client-bindings/app'

interface FileTreeFolderRow {
  kind: 'folder'
  path: string
  name: string
  depth: number
}

interface FileTreeFileRow {
  kind: 'file'
  file: FileStatus
  depth: number
}

type FileTreeRow = FileTreeFolderRow | FileTreeFileRow

interface TreeNode {
  folders: Map<string, TreeNode>
  files: FileStatus[]
}

function insert(root: TreeNode, file: FileStatus): void {
  // A nested repository is listed as a folder, with a trailing slash.
  const segments = file.path.replace(/\/$/, '').split('/')
  let node = root
  for (let i = 0; i < segments.length - 1; i++) {
    const segment = segments[i]
    let child = node.folders.get(segment)
    if (!child) {
      child = { folders: new Map(), files: [] }
      node.folders.set(segment, child)
    }
    node = child
  }
  node.files.push(file)
}

// VS Code explorer convention: folders before files, alphabetical within each group.
function flatten(
  node: TreeNode,
  pathPrefix: string,
  depth: number,
  isCollapsed: (path: string) => boolean,
  out: FileTreeRow[],
): void {
  const folderNames = [...node.folders.keys()].sort((a, b) => a.localeCompare(b))
  for (const name of folderNames) {
    const path = pathPrefix ? `${pathPrefix}/${name}` : name
    out.push({ kind: 'folder', path, name, depth })
    if (!isCollapsed(path)) {
      flatten(node.folders.get(name)!, path, depth + 1, isCollapsed, out)
    }
  }
  const files = [...node.files].sort((a, b) => a.path.localeCompare(b.path))
  for (const file of files) {
    out.push({ kind: 'file', file, depth })
  }
}

export function buildFileTreeRows(files: FileStatus[], isCollapsed: (path: string) => boolean): FileTreeRow[] {
  const root: TreeNode = { folders: new Map(), files: [] }
  for (const file of files) insert(root, file)
  const rows: FileTreeRow[] = []
  flatten(root, '', 0, isCollapsed, rows)
  return rows
}
