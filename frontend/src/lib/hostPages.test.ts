import { Links } from '@current-client-bindings/core/hostlink'
import { branchPage, commitPage, filePage, remoteBranchName, repositoryPage, tagPage } from './hostPages'

const github = new Links({
  project: 'https://github.com/o/r',
  commit: 'https://github.com/o/r/commit/',
  tree: 'https://github.com/o/r/tree/',
  tag: 'https://github.com/o/r/releases/tag/',
  blob: 'https://github.com/o/r/blob/',
})
const gitlab = new Links({
  project: 'https://gitlab.com/g/s/r',
  commit: 'https://gitlab.com/g/s/r/-/commit/',
  tree: 'https://gitlab.com/g/s/r/-/tree/',
  tag: 'https://gitlab.com/g/s/r/-/tags/',
  blob: 'https://gitlab.com/g/s/r/-/blob/',
})

describe('host pages', () => {
  it('builds every page in both styles', () => {
    expect(commitPage(github, 'abc123')).toBe('https://github.com/o/r/commit/abc123')
    expect(commitPage(gitlab, 'abc123')).toBe('https://gitlab.com/g/s/r/-/commit/abc123')
    expect(filePage(github, 'abc123', 'app/x.go', 10)).toBe('https://github.com/o/r/blob/abc123/app/x.go#L10')
    expect(filePage(gitlab, 'abc123', 'docs/a b.md')).toBe('https://gitlab.com/g/s/r/-/blob/abc123/docs/a%20b.md')
    expect(branchPage(github, 'feature/login')).toBe('https://github.com/o/r/tree/feature/login')
    expect(branchPage(gitlab, 'feature/login')).toBe('https://gitlab.com/g/s/r/-/tree/feature/login')
    expect(tagPage(github, 'v1.2.0')).toBe('https://github.com/o/r/releases/tag/v1.2.0')
    expect(tagPage(gitlab, 'v1.2.0')).toBe('https://gitlab.com/g/s/r/-/tags/v1.2.0')
    expect(repositoryPage(gitlab)).toBe('https://gitlab.com/g/s/r')
  })

  it('strips the remote from an upstream, longest remote name first', () => {
    expect(remoteBranchName('origin/feature/login', ['origin'])).toBe('feature/login')
    expect(remoteBranchName('team/origin/x', ['team', 'team/origin'])).toBe('x')
    expect(remoteBranchName('feature', ['origin'])).toBe('feature')
  })
})
