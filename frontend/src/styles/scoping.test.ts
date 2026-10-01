import { compile } from 'sass'
import { resolve } from 'node:path'

// Rules that would reach outside the .current-client root when Current Client is embedded.
const PAGE_SELECTOR = /^(:root|html|body|#root|\*)(?![\w-])/

function topLevelSelectors(css: string): string[] {
  return [...css.matchAll(/(?:^|[;}])\s*([^{};@]+)\{/g)].flatMap((m) => m[1].split(',').map((s) => s.trim()))
}

describe('shared styles', () => {
  it('are scoped to the .current-client root', () => {
    const { css } = compile(resolve('src/styles/styles.scss'))
    const selectors = topLevelSelectors(css)

    expect(selectors.length).toBeGreaterThan(10)
    expect(selectors.filter((s) => PAGE_SELECTOR.test(s))).toEqual([])
  })
})
