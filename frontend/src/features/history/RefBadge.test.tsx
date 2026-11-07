import { render, screen } from '@testing-library/react'
import { BranchChange, RefInfo } from '@current-client-bindings/app'
import RefBadge from './RefBadge'

describe('RefBadge', () => {
  it('puts the change marker beside a remote ref pill, leaving the pill its lane color', () => {
    const ref = new RefInfo({ name: 'origin/feature', kind: 'remote', sha: 'abc' })
    render(
      <RefBadge
        refInfo={ref}
        color="rgb(1, 2, 3)"
        change={new BranchChange({ number: 42, prefix: '#', state: 'merged' })}
      />,
    )

    expect(screen.getByText('origin/feature').parentElement?.style.getPropertyValue('--lane')).toBe('rgb(1, 2, 3)')
    expect(screen.getByRole('button', { name: '#42 · merged' })).toHaveAttribute('data-state', 'merged')
  })

  it('is just the pill without a change', () => {
    render(<RefBadge refInfo={new RefInfo({ name: 'main', kind: 'branch', sha: 'abc' })} color="red" />)
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('starts a tag with the tag icon', () => {
    render(<RefBadge refInfo={new RefInfo({ name: 'v1.1', kind: 'tag', sha: 'abc' })} color="red" />)
    expect(screen.getByText('v1.1').parentElement?.firstElementChild).toHaveClass('ref-badge-tag-icon')
  })

  it('gives a branch no tag icon', () => {
    const { container } = render(
      <RefBadge refInfo={new RefInfo({ name: 'main', kind: 'branch', sha: 'abc' })} color="red" />,
    )
    expect(container.querySelector('.ref-badge-tag-icon')).toBeNull()
  })
})
