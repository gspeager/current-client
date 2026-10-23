// A copy of set with value added or removed; without `on`, whichever it isn't.
export function toggled<T>(set: ReadonlySet<T>, value: T, on = !set.has(value)): Set<T> {
  const next = new Set(set)
  if (on) next.add(value)
  else next.delete(value)
  return next
}
