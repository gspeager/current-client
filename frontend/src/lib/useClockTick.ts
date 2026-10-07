import { useSyncExternalStore } from 'react'

// One timer shared by every subscriber, running only while something is subscribed.
const TICK_MS = 30_000
const listeners = new Set<() => void>()
let tick = 0
let timer: number | undefined

function subscribe(listener: () => void) {
  listeners.add(listener)
  timer ??= window.setInterval(() => {
    tick++
    listeners.forEach((l) => l())
  }, TICK_MS)
  return () => {
    listeners.delete(listener)
    if (listeners.size === 0) {
      window.clearInterval(timer)
      timer = undefined
    }
  }
}

// Re-renders the caller every 30 seconds, so the relativeTime() text it shows
// ("fetched 2m ago") keeps up without re-rendering the rest of the app.
export function useClockTick(): void {
  useSyncExternalStore(subscribe, () => tick)
}
