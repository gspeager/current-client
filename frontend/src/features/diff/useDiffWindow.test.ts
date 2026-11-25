import { renderHook } from '@testing-library/react'
import { giveElementsLayout } from '../../test/bindings'
import { useDiffWindow } from './useDiffWindow'

it('works out no position again when redrawn', () => {
  giveElementsLayout()
  const pane = document.createElement('div')
  const heightOf = vi.fn(() => 20)
  const keyOf = (index: number) => String(index)
  const { rerender } = renderHook(() =>
    useDiffWindow({
      windowed: true,
      count: 5000,
      keyOf,
      layout: 'unified',
      heightOf,
      measured: () => false,
      keep: [],
      scroller: () => pane,
      paneRef: { current: pane },
    }),
  )
  expect(heightOf).toHaveBeenCalled()
  heightOf.mockClear()
  rerender()
  expect(heightOf).not.toHaveBeenCalled()
})
