import { useState, type FormEvent } from 'react'
import { RemoteService } from '@current-client-bindings/app'
import { useAsyncData } from '../../lib/useAsyncData'
import type { SignIn, SignInOptions } from '../../lib/signIn'
import Modal from './Modal'

function SignInDialog({ options, onSettle }: { options: SignInOptions; onSettle: (value: SignIn | null) => void }) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const { data: helper } = useAsyncData(() => RemoteService.CredentialHelper(), [])
  const canSubmit = username.trim() !== '' && password !== ''

  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (canSubmit) onSettle({ username: username.trim(), password })
  }

  return (
    <Modal title="Sign in to remote" onClose={() => onSettle(null)} className="dialog-panel">
      <form className="dialog-body" onSubmit={submit}>
        <p className="dialog-message" role={options.rejected ? 'alert' : undefined}>
          {options.rejected
            ? 'The remote didn’t accept that username and password. Check them and try again.'
            : 'Git needs a username and password for this remote.'}
        </p>
        <label className="dialog-field">
          <span className="dialog-message">Username</span>
          <input
            className="dialog-input"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            autoComplete="username"
            autoFocus
          />
        </label>
        <label className="dialog-field">
          <span className="dialog-message">Password or token</span>
          <input
            className="dialog-input"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
          />
        </label>
        <p className="dialog-note">
          GitHub, GitLab and Bitbucket need a personal access token here, not your account password.
        </p>
        {helper !== null && (
          <p className="dialog-note">
            {helper
              ? `Git will save this with its ${helper} credential helper, so you won’t be asked again.`
              : 'No Git credential helper is set up, so this is used once and not saved. Install Git Credential Manager to have Git remember it.'}
          </p>
        )}
        <div className="dialog-actions">
          <button type="button" className="dialog-button" onClick={() => onSettle(null)}>
            Cancel
          </button>
          <button type="submit" className="dialog-button dialog-button-primary" disabled={!canSubmit}>
            Sign in
          </button>
        </div>
      </form>
    </Modal>
  )
}

export default SignInDialog
