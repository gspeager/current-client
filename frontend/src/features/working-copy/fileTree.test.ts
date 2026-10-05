import type { FileStatus } from '@current-client-bindings/app'
import { describe, expect, it } from 'vitest'
import { buildFileTreeRows } from './fileTree'

function file(path: string): FileStatus {
  return {
    path,
    origPath: '',
    indexStatus: 'M',
    worktreeStatus: ' ',
    conflicted: false,
    indexAdded: 0,
    indexRemoved: 0,
    indexBinary: false,
    workAdded: 0,
    workRemoved: 0,
    workBinary: false,
    submodule: null,
    lfs: false,
  }
}

describe('buildFileTreeRows', () => {
  it('nests a multi-directory change set with folders before files, alphabetically', () => {
    const rows = buildFileTreeRows(
      [file('src/b.ts'), file('src/a.ts'), file('README.md'), file('src/components/Button.tsx')],
      () => false,
    )
    expect(rows.map((r) => (r.kind === 'folder' ? `dir:${r.path}` : `file:${r.file.path}`))).toEqual([
      'dir:src',
      'dir:src/components',
      'file:src/components/Button.tsx',
      'file:src/a.ts',
      'file:src/b.ts',
      'file:README.md',
    ])
  })

  it('assigns increasing depth per nesting level', () => {
    const rows = buildFileTreeRows([file('a/b/c.ts')], () => false)
    const folderDepths = rows.filter((r) => r.kind === 'folder').map((r) => r.depth)
    expect(folderDepths).toEqual([0, 1])
    expect(rows.find((r) => r.kind === 'file')?.depth).toBe(2)
  })

  it('omits descendants of a collapsed folder but keeps the folder row', () => {
    const rows = buildFileTreeRows([file('src/a.ts'), file('src/nested/b.ts')], (path) => path === 'src')
    expect(rows).toEqual([{ kind: 'folder', path: 'src', name: 'src', depth: 0 }])
  })
})
