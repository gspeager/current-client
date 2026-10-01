import { baseName } from './paths'

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
