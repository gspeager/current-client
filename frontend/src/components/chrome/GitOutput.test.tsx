import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { errorMessage } from '../../lib/errors'
import DialogError from './DialogError'

const failedPush = () =>
  Object.assign(new Error('Access to this remote was denied.'), {
    cause: {
      Message: 'Access to this remote was denied.',
      Detail: 'remote: Permission to team/app.git denied.\nfatal: unable to access',
      Command: "git -c 'http.extraheader=<token>' push 'https://<token>@example.com/team/app.git' HEAD",
      ExitCode: 128,
    },
  })

it('keeps the Git output closed under the message, and copies it', async () => {
  const writeText = vi.fn().mockResolvedValue(undefined)
  Object.assign(navigator, { clipboard: { writeText } })
  render(<DialogError>{errorMessage(failedPush())}</DialogError>)

  expect(screen.getByRole('alert')).toHaveTextContent(/^Access to this remote was denied\./)
  const details = screen.getByText('Show Git output').closest('details')
  expect(details).not.toHaveAttribute('open')

  await userEvent.click(screen.getByText('Show Git output'))
  expect(details).toHaveAttribute('open')
  const text =
    "git -c 'http.extraheader=<token>' push 'https://<token>@example.com/team/app.git' HEAD\nexit 128\n\nremote: Permission to team/app.git denied.\nfatal: unable to access"
  expect(details?.querySelector('pre')?.textContent).toBe(text)
  await userEvent.click(screen.getByRole('button', { name: 'Copy' }))
  expect(writeText).toHaveBeenCalledWith(text)
})

it('shows nothing more for a failure that ran no git command', () => {
  render(<DialogError>{errorMessage(new Error('Enter a username and a password or token.'))}</DialogError>)
  expect(screen.queryByText('Show Git output')).toBeNull()
})

it('follows the latest failure with the same message', () => {
  errorMessage(failedPush())
  errorMessage(new Error('Access to this remote was denied.'))
  render(<DialogError>Access to this remote was denied.</DialogError>)
  expect(screen.queryByText('Show Git output')).toBeNull()
})
