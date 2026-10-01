import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import FileRow from './FileRow'

describe('FileRow staging interactions', () => {
  it('calls onToggleChecked when the checkbox is clicked', async () => {
    const user = userEvent.setup()
    const onToggleChecked = vi.fn()
    render(
      <FileRow
        status="M"
        label="src/app.ts"
        selected={false}
        checked={false}
        onToggleChecked={onToggleChecked}
        onSelect={vi.fn()}
      />,
    )

    await user.click(screen.getByRole('checkbox', { name: 'Stage src/app.ts' }))
    expect(onToggleChecked).toHaveBeenCalledTimes(1)
  })

  it('labels the checkbox for unstaging once the file is checked', () => {
    render(
      <FileRow status="M" label="src/app.ts" selected={false} checked onToggleChecked={vi.fn()} onSelect={vi.fn()} />,
    )
    expect(screen.getByRole('checkbox', { name: 'Unstage src/app.ts' })).toBeChecked()
  })

  it('calls onSelect when the file path is clicked', async () => {
    const user = userEvent.setup()
    const onSelect = vi.fn()
    render(
      <FileRow
        status="M"
        label="src/app.ts"
        selected={false}
        checked={false}
        onToggleChecked={vi.fn()}
        onSelect={onSelect}
      />,
    )

    await user.click(screen.getByText('src/app.ts'))
    expect(onSelect).toHaveBeenCalledTimes(1)
  })

  it('calls onDiscard when the discard button is clicked, and omits it when absent', async () => {
    const user = userEvent.setup()
    const onDiscard = vi.fn()
    render(
      <FileRow
        status="M"
        label="src/app.ts"
        selected={false}
        checked={false}
        onToggleChecked={vi.fn()}
        onSelect={vi.fn()}
        onDiscard={onDiscard}
      />,
    )
    await user.click(screen.getByRole('button', { name: 'Discard src/app.ts' }))
    expect(onDiscard).toHaveBeenCalledTimes(1)
  })

  it('renders no discard control when onDiscard is not provided', () => {
    render(
      <FileRow status="M" label="src/app.ts" selected={false} checked onToggleChecked={vi.fn()} onSelect={vi.fn()} />,
    )
    expect(screen.queryByRole('button', { name: /Discard/ })).not.toBeInTheDocument()
  })
})
