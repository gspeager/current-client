import { baseName, joinPath, tildePath } from './paths'

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

describe('tildePath', () => {
  it('shows the home folder as ~', () => {
    expect(tildePath('/Users/sam/projects/acme', '/Users/sam')).toBe('~/projects/acme')
    expect(tildePath('/Users/sam', '/Users/sam')).toBe('~')
  })

  it('leaves other paths, and paths that only start with the same letters, alone', () => {
    expect(tildePath('/Users/samantha/acme', '/Users/sam')).toBe('/Users/samantha/acme')
    expect(tildePath('/srv/acme', '/Users/sam')).toBe('/srv/acme')
    expect(tildePath('C:\\Users\\sam\\acme', '')).toBe('C:\\Users\\sam\\acme')
  })
})
