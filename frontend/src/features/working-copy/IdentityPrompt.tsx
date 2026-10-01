import { useState } from 'react'
import { IdentityService } from '@current-client-bindings/app'
import { errorMessage } from '../../lib/errors'
import './IdentityPrompt.scss'

interface IdentityPromptProps {
  repoPath: string
  initialName: string
  initialEmail: string
  onSaved: () => void
}

function IdentityPrompt({ repoPath, initialName, initialEmail, onSaved }: IdentityPromptProps) {
  const [name, setName] = useState(initialName)
  const [email, setEmail] = useState(initialEmail)
  const [repoOnly, setRepoOnly] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const save = () => {
    setError(null)
    const request = repoOnly
      ? IdentityService.SetRepoUser(repoPath, name.trim(), email.trim())
      : IdentityService.SetGlobalUser(name.trim(), email.trim())
    request.then(onSaved).catch((err: unknown) => setError(errorMessage(err)))
  }

  return (
    <div className="identity-prompt">
      <p className="identity-prompt-title">Commits need a name and email.</p>
      <input aria-label="Name" placeholder="Name" value={name} onChange={(e) => setName(e.target.value)} />
      <input
        aria-label="Email"
        type="email"
        placeholder="Email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
      />
      <label className="identity-prompt-scope">
        <input type="checkbox" checked={repoOnly} onChange={(e) => setRepoOnly(e.target.checked)} />
        This repository only
      </label>
      <button type="button" onClick={save} disabled={name.trim() === '' || email.trim() === ''}>
        Save identity
      </button>
      {error && <p className="identity-prompt-error">Could not save: {error}</p>}
    </div>
  )
}

export default IdentityPrompt
