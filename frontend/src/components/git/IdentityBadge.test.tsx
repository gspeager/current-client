import { render, screen } from '@testing-library/react'
import IdentityBadge from './IdentityBadge'

describe('IdentityBadge', () => {
  it('renders the given initials, never a fetched image', () => {
    render(<IdentityBadge identity={{ initials: 'AB', color: '#4cd7f6' }} name="Ada Byron" email="ada@example.com" />)
    expect(screen.getByText('AB')).toBeInTheDocument()
    expect(document.querySelector('img')).not.toBeInTheDocument()
  })

  it('uses the identity color as its background, not a computed one', () => {
    render(
      <IdentityBadge
        identity={{ initials: 'GS', color: 'rgb(76, 215, 246)' }}
        name="Grace"
        email="grace@example.com"
      />,
    )
    expect(screen.getByText('GS')).toHaveStyle({ background: 'rgb(76, 215, 246)' })
  })

  it('titles the badge with name and email when an email is present', () => {
    render(<IdentityBadge identity={{ initials: 'GS', color: '#000' }} name="Grace" email="grace@example.com" />)
    expect(screen.getByTitle('Grace <grace@example.com>')).toBeInTheDocument()
  })

  it('falls back to just the name when there is no email', () => {
    render(<IdentityBadge identity={{ initials: 'GS', color: '#000' }} name="Grace" email="" />)
    expect(screen.getByTitle('Grace')).toBeInTheDocument()
  })
})
