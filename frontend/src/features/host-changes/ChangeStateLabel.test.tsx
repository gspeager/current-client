import { render, screen } from '@testing-library/react'
import ChangeStateLabel from './ChangeStateLabel'

describe('ChangeStateLabel', () => {
  it.each([
    ['open', 'Open', 'lucide-git-pull-request'],
    ['merged', 'Merged', 'lucide-git-merge'],
    ['draft', 'Draft', 'lucide-git-pull-request-draft'],
    ['closed', 'Closed', 'lucide-git-pull-request-closed'],
  ])('gives %s its name in sentence case and its own icon', (state, name, icon) => {
    const { container } = render(<ChangeStateLabel state={state} />)
    expect(screen.getByText(name)).toHaveClass(`host-change-state-${state}`)
    expect(container.querySelector(`.${icon}`)).toBeInTheDocument()
  })

  it('uses issue icons for an issue', () => {
    const { container } = render(<ChangeStateLabel state="closed" issue />)
    expect(container.querySelector('.lucide-circle-check')).toBeInTheDocument()
  })

  it('still names a state it has no icon for', () => {
    const { container } = render(<ChangeStateLabel state="triage" issue />)
    expect(screen.getByText('Triage')).toBeInTheDocument()
    expect(container.querySelector('svg')).not.toBeInTheDocument()
  })
})
