import { render, screen } from '@testing-library/react'
import { ExternalLink } from 'lucide-react'
import { TrackerLinks } from '@current-client-bindings/app'
import { Links, Style } from '@current-client-bindings/core/hostlink'
import ReferenceText from './ReferenceText'
import { hostLogo, issueProvider } from './hostLogos'
import { IssuePanelContext } from '../../lib/issuePanel'

const github = new Links({ issue: 'https://github.com/o/r/issues/', style: Style.GitHub })

describe('host logos', () => {
  it('has one for each host and tracker, and the external-link icon for the rest', () => {
    for (const provider of ['github', 'gitlab', 'jira']) expect(hostLogo(provider)).not.toBe(ExternalLink)
    expect(hostLogo('gitea')).toBe(ExternalLink)
    expect(hostLogo(undefined)).toBe(ExternalLink)
  })

  it('reads numbered issues as the host’s and keys as Jira’s', () => {
    expect(issueProvider('18', github)).toBe('github')
    expect(issueProvider('#18', github)).toBe('github')
    expect(issueProvider('WEB-57', github)).toBe('jira')
    expect(issueProvider('18', null)).toBeUndefined()
  })

  it('puts the logo before a reference, kept out of its name', () => {
    const tracker = new TrackerLinks({
      browse: 'https://acme.atlassian.net/browse/',
      keyPattern: '(?:^|[^\\w/.-])([A-Z][A-Z0-9_]+-[1-9]\\d*)\\b',
    })
    render(
      <IssuePanelContext.Provider value={{ tracker }}>
        <ReferenceText text="Fix #12 for WEB-57" links={github} />
      </IssuePanelContext.Provider>,
    )
    for (const name of ['#12', 'WEB-57']) {
      const link = screen.getByRole('link', { name })
      expect(link.querySelector('svg.reference-logo')).toHaveAttribute('aria-hidden', 'true')
    }
  })
})
