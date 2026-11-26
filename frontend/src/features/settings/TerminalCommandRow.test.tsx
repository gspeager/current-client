import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { PlatformService } from '@current-client-bindings/app'
import { Launcher } from '@current-client-bindings/internal/platform'
import TerminalCommandRow from './TerminalCommandRow'

const path = '/Users/me/.local/bin/current-client'

it('says where it will write, and writes only on Install', async () => {
  vi.mocked(PlatformService.TerminalCommand).mockResolvedValue(new Launcher({ path }))
  vi.mocked(PlatformService.InstallTerminalCommand).mockResolvedValue(
    new Launcher({ path, installed: true, pathLine: 'export PATH="$HOME/.local/bin:$PATH"' }),
  )
  render(<TerminalCommandRow />)
  expect(await screen.findByText(/Install writes it here/)).toBeInTheDocument()
  expect(screen.getByTitle(path)).toBeInTheDocument()
  expect(PlatformService.InstallTerminalCommand).not.toHaveBeenCalled()

  await userEvent.click(screen.getByRole('button', { name: 'Install' }))
  expect(PlatformService.InstallTerminalCommand).toHaveBeenCalledTimes(1)
  expect(await screen.findByText('export PATH="$HOME/.local/bin:$PATH"')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Remove' })).toBeInTheDocument()
})

it('removes it, and shows why it could not', async () => {
  vi.mocked(PlatformService.TerminalCommand).mockResolvedValue(new Launcher({ path, installed: true, onPath: true }))
  vi.mocked(PlatformService.RemoveTerminalCommand).mockRejectedValueOnce(
    new Error('The terminal command couldn’t be removed.'),
  )
  render(<TerminalCommandRow />)
  await userEvent.click(await screen.findByRole('button', { name: 'Remove' }))
  expect(await screen.findByText('The terminal command couldn’t be removed.')).toBeInTheDocument()

  vi.mocked(PlatformService.RemoveTerminalCommand).mockResolvedValueOnce(new Launcher({ path, onPath: true }))
  await userEvent.click(screen.getByRole('button', { name: 'Remove' }))
  expect(await screen.findByRole('button', { name: 'Install' })).toBeInTheDocument()
  expect(screen.queryByText('The terminal command couldn’t be removed.')).toBeNull()
})

it('only reports a command a package installed', async () => {
  vi.mocked(PlatformService.TerminalCommand).mockResolvedValue(
    new Launcher({ path: '/usr/bin/current-client', installed: true, byPackage: true, onPath: true }),
  )
  render(<TerminalCommandRow />)
  expect(await screen.findByText(/Installed by the app’s package/)).toBeInTheDocument()
  expect(screen.queryByRole('button')).toBeNull()
})

it('is left out of an app that embeds Current Client', async () => {
  vi.mocked(PlatformService.TerminalCommand).mockResolvedValue(new Launcher())
  const { container } = render(<TerminalCommandRow />)
  await vi.waitFor(() => expect(PlatformService.TerminalCommand).toHaveBeenCalled())
  expect(container).toBeEmptyDOMElement()
})
