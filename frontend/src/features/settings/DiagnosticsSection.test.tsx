import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { PlatformService } from '@current-client-bindings/app'
import { errorMessage } from '../../lib/errors'
import DiagnosticsSection from './DiagnosticsSection'

describe('DiagnosticsSection', () => {
  it('copies a report that includes errors the UI has shown, newest first', async () => {
    const user = userEvent.setup()
    const writeText = vi.spyOn(navigator.clipboard, 'writeText')
    vi.mocked(PlatformService.GetDiagnostics).mockResolvedValue('the report')
    errorMessage(new Error('Authentication failed.'))
    errorMessage(new Error('The remote stopped responding.'))
    render(<DiagnosticsSection />)

    await user.click(screen.getByRole('button', { name: 'Copy diagnostics' }))

    const sent = vi.mocked(PlatformService.GetDiagnostics).mock.calls[0][0]
    expect(sent.slice(0, 2).map((e) => e.message)).toEqual(['The remote stopped responding.', 'Authentication failed.'])
    expect(writeText).toHaveBeenCalledWith('the report')
    expect(await screen.findByText('Copied.')).toBeInTheDocument()
  })
})
