import { fireEvent, render, screen } from '@testing-library/react'
import { useEffect } from 'react'
import { useWindowKeydown } from '../../lib/useWindowKeydown'
import KeepAlive from './KeepAlive'

function View({ label, onMount, onKey }: { label: string; onMount: () => void; onKey: () => void }) {
  useEffect(onMount, [onMount])
  useWindowKeydown(onKey)
  return <p>{label}</p>
}

function setup() {
  const onMount = vi.fn()
  const onKey = vi.fn()
  const view = (active: boolean, label: string) => (
    <KeepAlive active={active}>
      <View label={label} onMount={onMount} onKey={onKey} />
    </KeepAlive>
  )
  return { onMount, onKey, view }
}

describe('KeepAlive', () => {
  it('does not mount a view until it is first shown', () => {
    const { onMount, view } = setup()
    const { rerender } = render(view(false, 'v1'))
    expect(onMount).not.toHaveBeenCalled()
    expect(screen.queryByText('v1')).not.toBeInTheDocument()

    rerender(view(true, 'v1'))
    expect(onMount).toHaveBeenCalledTimes(1)
    expect(screen.getByText('v1')).toBeVisible()
  })

  it('keeps the view mounted when hidden and shown again', () => {
    const { onMount, view } = setup()
    const { rerender } = render(view(true, 'v1'))
    rerender(view(false, 'v1'))
    expect(screen.getByText('v1')).not.toBeVisible()
    rerender(view(true, 'v1'))

    expect(screen.getByText('v1')).toBeVisible()
    expect(onMount).toHaveBeenCalledTimes(1)
  })

  it('holds back new props while hidden and applies them when shown', () => {
    const { view } = setup()
    const { rerender } = render(view(true, 'v1'))
    rerender(view(false, 'v2'))
    expect(screen.getByText('v1')).toBeInTheDocument()
    expect(screen.queryByText('v2')).not.toBeInTheDocument()

    rerender(view(true, 'v2'))
    expect(screen.getByText('v2')).toBeVisible()
  })

  it('pauses window shortcuts while hidden', () => {
    const { onKey, view } = setup()
    const { rerender } = render(view(true, 'v1'))
    fireEvent.keyDown(window, { key: 'j' })
    expect(onKey).toHaveBeenCalledTimes(1)

    rerender(view(false, 'v1'))
    fireEvent.keyDown(window, { key: 'j' })
    expect(onKey).toHaveBeenCalledTimes(1)
  })
})
