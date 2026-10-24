import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HostNotice } from '@current-client-bindings/app'
import HostNoticeBanner from './HostNoticeBanner'

describe('HostNoticeBanner', () => {
  it('shows one line with every notice and sends each to its place', async () => {
    const onOpenChange = vi.fn()
    const onOpenFile = vi.fn()
    const onDismiss = vi.fn()
    render(
      <HostNoticeBanner
        notices={[
          new HostNotice({ kind: 'threads', text: '2 new comments on !7', number: 7, path: 'app/x.go' }),
          new HostNotice({ kind: 'ci', text: 'Pipeline failed: test on !7', number: 7 }),
        ]}
        onOpenChange={onOpenChange}
        onOpenFile={onOpenFile}
        onDismiss={onDismiss}
      />,
    )
    expect(screen.getByRole('status')).toHaveTextContent('2 new comments on !7·Pipeline failed: test on !7')
    await userEvent.click(screen.getByRole('button', { name: '2 new comments on !7' }))
    expect(onOpenFile).toHaveBeenCalledWith('app/x.go')
    await userEvent.click(screen.getByRole('button', { name: 'Pipeline failed: test on !7' }))
    expect(onOpenChange).toHaveBeenCalledWith(7)
    await userEvent.click(screen.getByRole('button', { name: 'Dismiss' }))
    expect(onDismiss).toHaveBeenCalledTimes(3)
  })

  it('renders nothing without notices', () => {
    render(<HostNoticeBanner notices={[]} onOpenChange={vi.fn()} onOpenFile={vi.fn()} onDismiss={vi.fn()} />)
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })
})
