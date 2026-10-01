import { useEffect, useState } from 'react'
import { IdentityService } from '@current-client-bindings/app'
import { errorMessage } from '../../lib/errors'
import SettingsRow from './SettingsRow'

function IdentitySettings() {
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [saved, setSaved] = useState({ name: '', email: '' })
  const [justSaved, setJustSaved] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Seeds editable fields once, so it stays an effect rather than useAsyncData.
  useEffect(() => {
    IdentityService.GetGlobalUser()
      .then((user) => {
        setName(user.name)
        setEmail(user.email)
        setSaved({ name: user.name, email: user.email })
      })
      .catch((err: unknown) => setError(errorMessage(err)))
  }, [])

  const dirty = name !== saved.name || email !== saved.email

  const save = () => {
    setError(null)
    setJustSaved(false)
    IdentityService.SetGlobalUser(name, email)
      .then(() => {
        setSaved({ name, email })
        setJustSaved(true)
      })
      .catch((err: unknown) => setError(errorMessage(err)))
  }

  const edit = (update: (value: string) => void) => (value: string) => {
    update(value)
    setJustSaved(false)
  }

  return (
    <SettingsRow
      label="Name and email"
      hint="Written to the global git config (user.name, user.email). A repository with its own identity overrides it."
      error={error && `Could not save: ${error}`}
      stacked
    >
      <div className="settings-identity-fields">
        <input
          className="settings-field"
          aria-label="Name"
          placeholder="Name"
          value={name}
          onChange={(e) => edit(setName)(e.target.value)}
        />
        <input
          className="settings-field"
          aria-label="Email"
          placeholder="Email"
          value={email}
          onChange={(e) => edit(setEmail)(e.target.value)}
        />
        <button
          type="button"
          className="settings-button"
          onClick={save}
          disabled={!dirty || name.trim() === '' || email.trim() === ''}
        >
          Save
        </button>
      </div>
      {justSaved && <p className="settings-hint">Saved.</p>}
    </SettingsRow>
  )
}

export default IdentitySettings
