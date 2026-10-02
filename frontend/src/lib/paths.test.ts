import { baseName, joinPath } from './paths'

describe('baseName', () => {
  it('returns the last path segment for either separator', () => {
    expect(baseName('src/components/Button.tsx')).toBe('Button.tsx')
    expect(baseName('C:\\projects\\my-app')).toBe('my-app')
  })

  it('ignores a trailing separator', () => {
    expect(baseName('/home/dev/repo/')).toBe('repo')
  })

  it('returns a bare name unchanged', () => {
    expect(baseName('README.md')).toBe('README.md')
  })
})

describe('joinPath', () => {
  it('joins with the separator the parent uses', () => {
    expect(joinPath('/Users/dev/projects', 'app')).toBe('/Users/dev/projects/app')
    expect(joinPath('C:\\projects', 'app')).toBe('C:\\projects\\app')
  })

  it('does not double a trailing separator', () => {
    expect(joinPath('/', 'app')).toBe('/app')
    expect(joinPath('C:\\', 'app')).toBe('C:\\app')
  })
})
