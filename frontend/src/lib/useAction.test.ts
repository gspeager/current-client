import { act, renderHook } from '@testing-library/react'
import { useAction } from './useAction'

it('runs an action, keeps its error with a prefix, and clears it on the next run', async () => {
  const { result } = renderHook(() => useAction())
  let finish: (value: string) => void = () => {}
  let pending: Promise<string | undefined> = Promise.resolve(undefined)
  act(() => {
    pending = result.current.run(() => new Promise<string>((resolve) => (finish = resolve)))
  })
  expect(result.current.running).toBe(true)
  await act(async () => {
    finish('done')
    expect(await pending).toBe('done')
  })
  expect(result.current.running).toBe(false)

  await act(async () => {
    expect(await result.current.run(() => Promise.reject(new Error('No upstream.')), 'Could not pull')).toBeUndefined()
  })
  expect(result.current.error).toBe('Could not pull: No upstream.')

  await act(async () => {
    await result.current.run(() => Promise.resolve())
  })
  expect(result.current.error).toBeNull()
})
