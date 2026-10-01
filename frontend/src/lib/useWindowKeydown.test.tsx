import { fireEvent, render } from '@testing-library/react'
import { CurrentClientRootContext } from './currentClientRoot'
import { useWindowKeydown } from './useWindowKeydown'

function Shortcut({ onKey }: { onKey: (e: KeyboardEvent) => void }) {
  useWindowKeydown(onKey)
  return null
}

function renderShortcut(active: boolean) {
  const onKey = vi.fn()
  render(
    <CurrentClientRootContext.Provider value={{ element: null, active }}>
      <Shortcut onKey={onKey} />
    </CurrentClientRootContext.Provider>,
  )
  fireEvent.keyDown(window, { key: 'k', ctrlKey: true })
  return onKey
}

describe('useWindowKeydown', () => {
  it('handles keys while Current Client is shown', () => {
    expect(renderShortcut(true)).toHaveBeenCalledTimes(1)
  })

  it('ignores keys while a host app shows another view', () => {
    expect(renderShortcut(false)).not.toHaveBeenCalled()
  })
})
