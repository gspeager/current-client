import { useState, type KeyboardEvent, type ReactNode } from 'react'
import Switch from '../../components/forms/Switch'
import { errorMessage } from '../../lib/errors'
import { useActionError } from '../../lib/useActionError'

export function useIntegrationAction(reload: () => void, onChanged?: () => void) {
  const { error: actionError, setError: setActionError, fail } = useActionError()
  const run = (action: Promise<unknown>) =>
    action
      .then(() => {
        setActionError(null)
        reload()
        onChanged?.()
      })
      .catch(fail)
  return { run, actionError }
}

export type IntegrationAction = ReturnType<typeof useIntegrationAction>['run']

export function blurOnEnter(e: KeyboardEvent<HTMLInputElement>) {
  if (e.key === 'Enter') e.currentTarget.blur()
}

interface IntegrationHeadProps {
  name: string
  enabled: boolean
  locked: boolean
  onToggle: (on: boolean) => void
  children: ReactNode
}

export function IntegrationHead({ name, enabled, locked, onToggle, children }: IntegrationHeadProps) {
  return (
    <div className="settings-host-head">
      <div className="settings-row-text">
        <span className="settings-host-name">{name}</span>
        {children}
      </div>
      <div className="settings-host-switch">
        {locked && <span className="settings-hint">Managed by organization policy</span>}
        <Switch
          checked={enabled}
          onChange={onToggle}
          ariaLabel={`Integration for ${name}`}
          label={enabled ? 'On' : 'Off'}
          disabled={locked}
        />
      </div>
    </div>
  )
}

interface ConnectionActionsProps {
  enabled: boolean
  hasToken: boolean
  noTokenHint: string
  onSetToken: () => void
  onRemoveToken: () => void
  testConnection: () => Promise<string>
  onRemove?: () => void
}

export function ConnectionActions({
  enabled,
  hasToken,
  noTokenHint,
  onSetToken,
  onRemoveToken,
  testConnection,
  onRemove,
}: ConnectionActionsProps) {
  const [testing, setTesting] = useState(false)
  const [testResult, setTestResult] = useState<{ ok: boolean; text: string } | null>(null)

  const test = () => {
    setTesting(true)
    setTestResult(null)
    testConnection()
      .then((text) => setTestResult({ ok: true, text }))
      .catch((err: unknown) => setTestResult({ ok: false, text: errorMessage(err) }))
      .finally(() => setTesting(false))
  }

  return (
    <div className="settings-actions">
      <span className="settings-hint">{hasToken ? 'Token saved with Git’s credential helper.' : noTokenHint}</span>
      {hasToken ? (
        <button type="button" className="settings-button settings-button-small" onClick={onRemoveToken}>
          Remove token
        </button>
      ) : (
        <button type="button" className="settings-button settings-button-small" onClick={onSetToken}>
          Set token…
        </button>
      )}
      <button
        type="button"
        className="settings-button settings-button-small"
        onClick={test}
        disabled={!enabled || testing}
        title={enabled ? undefined : 'Turn the integration on first.'}
      >
        {testing ? 'Testing…' : 'Test connection'}
      </button>
      {onRemove && (
        <button type="button" className="settings-button settings-button-small" onClick={onRemove}>
          Remove
        </button>
      )}
      {testResult && (
        <span className={testResult.ok ? 'settings-hint' : 'settings-error'} role="status">
          {testResult.text}
        </span>
      )}
    </div>
  )
}
