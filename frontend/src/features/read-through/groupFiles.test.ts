import { ReadThroughFile } from '@current-client-bindings/app'
import { groupFiles, totalsByKind, totalsByStatus, totalsByType } from './groupFiles'

const file = (path: string, kind = '', status = 'M') => new ReadThroughFile({ path, kind, status })

describe('groupFiles', () => {
  it('groups by folder, with each test after its code', () => {
    const groups = groupFiles([
      file('src/app.ts'),
      file('src/app.test.ts', 'test'),
      file('src/apple.ts'),
      file('src/__tests__/apple.ts', 'test'),
      file('core/git/branch_test.go', 'test'),
      file('core/git/branch.go'),
      file('app/tests/test_views.py', 'test'),
      file('app/views.py'),
      file('src/test/java/x/UserTest.java', 'test'),
      file('src/main/java/x/User.java'),
      file('README.md'),
    ])
    expect(groups.map((g) => [g.folder, g.files.map((f) => f.path)])).toEqual([
      ['', ['README.md']],
      ['app', ['app/views.py', 'app/tests/test_views.py']],
      ['core/git', ['core/git/branch.go', 'core/git/branch_test.go']],
      ['src', ['src/app.ts', 'src/app.test.ts', 'src/apple.ts', 'src/__tests__/apple.ts']],
      ['src/main/java/x', ['src/main/java/x/User.java', 'src/test/java/x/UserTest.java']],
    ])
  })

  it('keeps a folder of tests at the top by itself', () => {
    expect(groupFiles([file('tests/a.py', 'test')])[0].folder).toBe('tests')
  })
})

it('totals by status, type and kind', () => {
  const files = [
    file('a.go', '', 'M'),
    file('b.go', '', 'A'),
    file('b_test.go', 'test', 'A'),
    file('moved.ts', '', 'R087'),
    file('go.sum', 'lockfile', 'D'),
    file('Makefile'),
    file('.gitignore'),
  ]
  expect(totalsByStatus(files).map((t) => `${t.label} ${t.files}`)).toEqual([
    'Added 2',
    'Modified 3',
    'Deleted 1',
    'Renamed 1',
  ])
  expect(totalsByType(files).map((t) => `${t.label} ${t.files}`)).toEqual([
    '.go 3',
    '.gitignore 1',
    '.sum 1',
    '.ts 1',
    'Makefile 1',
  ])
  expect(totalsByKind(files).map((t) => `${t.label} ${t.files}`)).toEqual(['Tests 1', 'Lockfiles 1'])
})
